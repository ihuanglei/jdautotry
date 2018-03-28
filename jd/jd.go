package jd

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
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

	// UserURL .
	UserURL = "https://i.jd.com/user/info"
)

type jdCookie struct {
	cookies []*http.Cookie
}

func (cookie *jdCookie) getCookie(key string) (*http.Cookie, error) {
	for _, cookie := range cookie.cookies {
		if cookie.Name == key {
			return cookie, nil
		}
	}
	return nil, errors.New("cookie not found")
}

// Channel 通道
type Channel struct {
	Cmd  int16       `json:"cmd"`
	Data interface{} `json:"data"`
}

// Product 商品
type Product struct {
	AID   string `json:"id"`
	Name  string `json:"name"`
	Price string `json:"price"`
	Img   string `json:"img"`
}

// User 用户信息你
type User struct {
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
}

// Option 参数
type Option struct {
	Callback func(*Channel)
}

// JD ...
type JD struct {
	qrCookie   *jdCookie
	thorCookie *http.Cookie
	option     *Option
	channal    chan Channel
}

// 获取商品
func (jd *JD) getProducts() {
	url := TryProductURL
	resp, err := http.Get(url)
	if err != nil {
		jd.e(err)
		return
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		jd.e(err)
		return
	}
	resp.Body.Close()
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		jd.e(err)
		return
	}

	// 获取总共多少条数据
	totalPage, err := strconv.Atoi(doc.Find(".page .p-skip b").Text())
	if err != nil {
		jd.e(err)
		return
	}
	jd.option.Callback(&Channel{Cmd: 21, Data: totalPage})

	// 解析产品数据
	parseProduct := func(doc *goquery.Document) {
		var products = []Product{}
		doc.Find("#goods-list .items .item").Each(func(i int, s *goquery.Selection) {
			aID, _ := s.Attr("activity_id")
			name := s.Find(".p-name").Text()
			price := s.Find(".p-price").Text()
			img, _ := s.Find(".p-img img").Attr("src")
			product := Product{AID: aID, Name: name, Price: price, Img: "http://" + img}
			products = append(products, product)
		})
		jd.option.Callback(&Channel{Cmd: 20, Data: products})
	}

	// 第一页已经取出 直接使用
	go parseProduct(doc)
	// 从第二页开始循环
	for i := 2; i <= totalPage; i++ {
		productDoc, _ := goquery.NewDocument(url + "?page=" + strconv.Itoa(i))
		go parseProduct(productDoc)
	}
}

// 获取cookies
func (jd *JD) getCookie(ticket string) {
	url := AuthURL + "?t=" + ticket
	resp, err := http.Get(url)
	if err != nil {
		jd.e(err)
		return
	}
	jd.thorCookie, _ = (&jdCookie{resp.Cookies()}).getCookie("thor")
	jd.getUser()
}

// 获取登录二维码
func (jd *JD) getQRImage() {
	log.Println("获取登录二维码")
	url := QRURL + "&t=" + jd.getTimestamp()
	resp, err := http.Get(url)
	if err != nil {
		jd.e(err)
		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		jd.e(err)
		return
	}
	jd.qrCookie = &jdCookie{resp.Cookies()}
	image := "data:image/png;base64," + base64.StdEncoding.EncodeToString(body)
	jd.option.Callback(&Channel{Cmd: 10, Data: image})
}

// 获取用户信息
func (jd *JD) getUser() {
	req, err := http.NewRequest("GET", UserURL, nil)
	if err != nil {
		jd.e(err)
		return
	}
	req.AddCookie(jd.thorCookie)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		jd.e(err)
		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		jd.e(err)
		return
	}
	doc, _ := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	name := doc.Find("#user-info .info-m B").Text()
	avatar, _ := doc.Find("#user-info .u-pic img").Attr("src")
	jd.option.Callback(&Channel{Cmd: 30, Data: User{Name: name, Avatar: "http://" + avatar}})
}

// 检查是否登录
func (jd *JD) onCheck() {
	type CheckResult struct {
		Code   int    `json:"code"`
		Msg    string `json:"msg"`
		Ticket string `json:"ticket"`
	}
	for {
		time.Sleep(3000 * time.Millisecond)
		if jd.qrCookie == nil {
			continue
		}
		qrCookie, err := jd.qrCookie.getCookie("QRCodeKey")
		if err != nil {
			continue
		}
		tokenCookie, err := jd.qrCookie.getCookie("wlfstk_smdl")
		if err != nil {
			continue
		}
		url := CheckURL + "&token=" + tokenCookie.Value + "&_=" + jd.getTimestamp()
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			jd.e(err)
			continue
		}
		// 设置cookies
		req.AddCookie(&http.Cookie{Name: "QRCodeKey", Value: qrCookie.Value})
		req.AddCookie(&http.Cookie{Name: "wlfstk_smdl", Value: tokenCookie.Value})
		req.Header.Add("Referer", Referer)
		// 发送请求
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			jd.e(err)
			continue
		}
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			jd.e(err)
			continue
		}
		resp.Body.Close()
		str := strings.Replace(strings.Replace(string(body), "callback({", "{", 1), "})", "}", 1)
		var result CheckResult
		if err := json.Unmarshal([]byte(str), &result); err != nil {
			jd.e(err)
			continue
		}
		if result.Code == 200 {
			jd.getCookie(result.Ticket)
			break
		}
		if result.Code == 205 || result.Code == 203 {
			jd.qrCookie = nil
			jd.option.Callback(&Channel{Cmd: 11, Data: ""})
			continue
		}
	}
}

func (jd *JD) try(id interface{}) {
	switch id.(type) {
	case string:
		idss := id.(string)
		p := NewPersistence()
		dbErr := p.Open()
		defer p.Close()
		if dbErr == nil {
			data, _ := p.Get(idss)
			if data != "" {
				jd.option.Callback(&Channel{Cmd: 53, Data: map[string]string{"code": "-1", "id": idss, "message": "您的申请已成功提交，请勿重复申请…"}})
				return
			}
		}
		url := TryURL + idss
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			jd.e(err)
			return
		}
		req.AddCookie(jd.thorCookie)
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			jd.e(err)
			return
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			jd.e(err)
			return
		}

		type TryResult struct {
			Success bool   `json:"success"`
			Message string `json:"message"`
			Code    string `json:"code"`
		}

		var result TryResult
		if err := json.Unmarshal(body, &result); err != nil {
			jd.e(err)
			return
		}

		fmt.Println(string(body))

		if result.Success {
			jd.option.Callback(&Channel{Cmd: 51, Data: map[string]string{"code": "1", "id": idss}})
			if dbErr == nil {
				p.Put(idss, "1")
			}
		} else {
			if result.Code == "-110" {
				if dbErr == nil {
					p.Put(idss, "1")
				}
			}
			jd.option.Callback(&Channel{Cmd: 52, Data: map[string]string{"code": "-1", "id": idss, "message": result.Message}})
		}
	default:
		jd.e(errors.New("试用参数错误"))
	}
}

// Send 发送消息
func (jd *JD) Send(c *Channel) {
	jd.channal <- *c
}

// 通道获取数据
func (jd *JD) onChannel() {
	for {
		send := <-jd.channal
		switch send.Cmd {
		case 1:
			go jd.getQRImage()
			go jd.onCheck()
		case 2:
			go jd.getProducts()
		case 5:
			go jd.try(send.Data)
		}
	}
}

// 当前时间戳
func (jd *JD) getTimestamp() string {
	return strconv.FormatInt(time.Now().Unix(), 10)
}

func (jd *JD) e(err error) {
	log.Println(err)
	jd.option.Callback(&Channel{Cmd: -100, Data: err.Error()})
}

// New 创建实例
func New(option *Option) (*JD, error) {
	if option == nil || option.Callback == nil {
		return nil, errors.New("参数不能为空")
	}
	jd := JD{option: option, channal: make(chan Channel)}
	go jd.onChannel()
	return &jd, nil
}
