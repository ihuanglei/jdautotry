package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	sciter "github.com/sciter-sdk/go-sciter"
	"github.com/sciter-sdk/go-sciter/window"
)

const (
	// QRURL .
	QRURL = "https://qr.m.jd.com/show?appid=133&size=147"
	// CheckURL .
	CheckURL = "https://qr.m.jd.com/check?callback=callback&appid=133"
	// AuthURL .
	AuthURL = "https://passport.jd.com/uc/qrCodeTicketValidation"
	// TryURL .
	TryURL = "http://try.jd.com/migrate/apply?source=0&activityId="
	// TryProductURL .
	TryProductURL = "https://try.jd.com/activity/getActivityList"
	// Referer .
	Referer = "https://passport.jd.com/new/login.aspx"
)

// JDCookie .
type JDCookie struct {
	cookies []*http.Cookie
}

func (jdCookie *JDCookie) getCookie(key string) (*http.Cookie, error) {
	for _, cookie := range jdCookie.cookies {
		if cookie.Name == key {
			return cookie, nil
		}
	}
	return nil, errors.New("cookie not found")
}

// CheckResult .
type CheckResult struct {
	Code   int    `json:"code"`
	Msg    string `json:"msg"`
	Ticket string `json:"ticket"`
}

// TryResult .
type TryResult struct {
	Code    string `json:"code"`
	Msg     string `json:"message"`
	Success bool   `json:"success"`
}

var jdCookie *JDCookie
var callbackChan = make(chan string)
var authCookieChan = make(chan JDCookie)
var uiLog *sciter.Element

func main() {
	go check()
	w, err := window.New(sciter.SW_TITLEBAR|sciter.SW_CONTROLS|sciter.SW_MAIN, &sciter.Rect{Left: 0, Top: 0, Right: 250, Bottom: 390})
	if err != nil {
		panic(err)
	}
	w.LoadFile("index.html")
	w.SetTitle("京东试用")
	initUI(w)
	initFunc(w)
	initCallback(w)
	w.Show()
	w.Run()
}

// 初始化页面
func initUI(w *window.Window) {
	root, err := w.GetRootElement()
	if err != nil {
		panic(err)
	}
	el, err := root.SelectById("ui-log")
	if err != nil {
		panic(err)
	}
	uiLog = el
}

//定义函数
func initFunc(w *window.Window) {
	//定义reloadQR重新获取
	w.DefineFunction("getQRImage", func(args ...*sciter.Value) *sciter.Value {
		data, err := getQRImage()
		if err != nil {
			allPrintLn(err.Error())
			return sciter.NullValue()
		}
		return sciter.NewValue("data:image/png;base64," + data)
	})
}

// 定义回调
func initCallback(w *window.Window) {
	go func() {
		for {
			val := <-callbackChan
			w.Call("callback", sciter.NewValue(val))
		}
	}()
}

// 页面显示日志
func uiPrintLn(message string) {
	uiLog.SetHtml("<p>"+message+"</p>", sciter.SIH_INSERT_AT_START)
}

// 控制台及页面显示日志
func allPrintLn(message string) {
	log.Println(message)
	uiPrintLn(message)
}

// 登录跳转及获取cookies
func login(ticket string) {
	url := AuthURL + "?t=" + ticket
	resp, err := http.Get(url)
	if err != nil {
		panic(err)
	}
	callbackChan <- "qr-login-ok"
	uiPrintLn("获取试用商品申请试用")
	cookies := &JDCookie{resp.Cookies()}
	fmt.Println(cookies)
}

// 试用
func tryIt(cookies *JDCookie) {
	url := TryURL
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		panic(err)
	}

	for _, cookie := range cookies.cookies {
		req.AddCookie(cookie)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	var result TryResult
	if err := json.Unmarshal(body, &result); err != nil {
		panic(err)
	}
	fmt.Println(result)
}

// 获取登录二维码
func getQRImage() (string, error) {
	log.Println("获取登录二维码")
	url := QRURL + "&t=" + getTimestamp()
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	jdCookie = &JDCookie{resp.Cookies()}
	return base64.StdEncoding.EncodeToString(body), nil
}

// 检查是否登录
func check() {
	var lastCode int
	for {
		log.Println("检查登录")
		time.Sleep(3000 * time.Millisecond)
		if jdCookie == nil {
			continue
		}
		qrCookie, err := jdCookie.getCookie("QRCodeKey")
		if err != nil {
			continue
		}
		tokenCookie, err := jdCookie.getCookie("wlfstk_smdl")
		if err != nil {
			continue
		}
		url := CheckURL + "&token=" + tokenCookie.Value + "&_=" + getTimestamp()
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			allPrintLn(err.Error())
			continue
		}
		// 设置cookies
		req.AddCookie(&http.Cookie{Name: "QRCodeKey", Value: qrCookie.Value})
		req.AddCookie(&http.Cookie{Name: "wlfstk_smdl", Value: tokenCookie.Value})
		req.Header.Add("Referer", Referer)

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			allPrintLn(err.Error())
			continue
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			allPrintLn(err.Error())
			continue
		}
		log.Println(string(body))
		str := strings.Replace(strings.Replace(string(body), "callback({", "{", 1), "})", "}", 1)
		var result CheckResult
		if err := json.Unmarshal([]byte(str), &result); err != nil {
			allPrintLn(err.Error())
			continue
		}

		if lastCode != result.Code {
			uiPrintLn(result.Msg)
		}
		if result.Code == 200 {
			allPrintLn("二维码授权成功")
			login(result.Ticket)
			break
		}
		if result.Code == 205 || result.Code == 203 {
			allPrintLn(result.Msg)
			lastCode = 0
			jdCookie = nil
			callbackChan <- "qr-timeout"
			continue
		}
		lastCode = result.Code
	}
}

// 当前时间戳
func getTimestamp() string {
	return strconv.FormatInt(time.Now().Unix(), 10)
}

func getProducts() {

}

func saveQRImage(b []byte) {
	out, err := os.Create("jdqr.png")
	if err != nil {
		panic(err)
	}
	defer out.Close()
	_, err = io.Copy(out, bytes.NewReader(b))
}
