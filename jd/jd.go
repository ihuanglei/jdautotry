package jd

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// 启动
func (jd *JD) first() {
	totalPageStr, err := jd.p.Get("totalPage")
	if err != nil {
		// 第一次打开，初始化及获取当前试用商品数据
		jd.loadProducts()
	} else {
		jd.totalPage = toInt(totalPageStr, 0)
		// 总共商品页数
		jd.callback(EventTotalPage, jd.totalPage)
		// 加载完成消息
		jd.callback(EventLoadPage, jd.totalPage)
		// 第一页数据
		// jd.getProductsAndSend(1)
	}
	proCount, err := jd.p.Get("/pro/count")
	if err == nil {
		jd.proCount = toInt(proCount, 0)
	}
	return
}

func (jd *JD) loginSuccess() {
	user, err := jd.getUser()
	if err != nil {
		jd.e(err)
		return
	}
	jd.user = user
	// 是否第一次登录
	firstLogin, _ := jd.p.Has("firstLogin")
	if firstLogin {
		// 已经登录则直接从本地获取
		user.FirstLogin = "1"
		tryCount, _ := jd.p.Get("/try/count")
		jd.tryCount = toInt(tryCount, 0)
		defer func() {
			jd.sendTryTip()
		}()
	} else {
		user.FirstLogin = ""
		defer func() {
			go jd.loadMyTrials()
		}()
	}
	jd.callback(EventMyInfo, jd.user)
}

// 拉取商品
func (jd *JD) loadProducts() {
	count := 0
	// 延时返回第一页数据
	// defer jd.getProductsAndSend(1)
	body, err := jd.get(TryProductURL)
	if err != nil {
		jd.e(err)
		return
	}
	doc, err := jd.document(body)
	if err != nil {
		jd.e(err)
		return
	}
	// 获取总共多少条数据
	totalPage, err := strconv.Atoi(doc.Find(".page .p-skip b").Text())
	if err != nil {
		jd.e(errors.New("获取商品失败"))
		return
	}

	// 清空数据
	jd.p.DeleteByPrefix("/pro/")

	// totalPage = 3
	jd.totalPage = totalPage
	jd.p.Put("totalPage", strconv.Itoa(totalPage))
	// 回调前台loading共多少页
	jd.callback(EventTotalPage, totalPage)
	jd.p.Batch()
	// 解析产品数据
	parseProduct := func(page int, doc *goquery.Document) {
		doc.Find("#goods-list .items .item").Each(func(i int, s *goquery.Selection) {
			aID, _ := s.Attr("activity_id")
			name := s.Find(".p-name").Text()
			price := s.Find(".p-price").Text()
			img, _ := s.Find(".p-img img").Attr("src")
			product := Product{AID: aID, Name: name, Price: price, Img: "http:" + img, Page: page, Idx: i}
			bs, err := json.Marshal(product)
			if err != nil {
				jd.e(err)
				return
			}
			jd.p.BatchPutByte("/pro/"+strconv.Itoa(page)+"/"+aID, bs)
			count++
		})
		// 更新了多少页
		jd.callback(EventLoadPage, page)
	}
	// 第一页已经取出 直接使用
	parseProduct(1, doc)
	// 从第二页开始循环
	for i := 2; i <= totalPage; i++ {
		body, err := jd.get(TryProductURL + "?page=" + strconv.Itoa(i))
		if err != nil {
			jd.e(err)
			continue
		}
		doc, err := jd.document(body)
		if err != nil {
			jd.e(err)
			continue
		}
		parseProduct(i, doc)
	}
	jd.p.BatchPutString("/pro/count", strconv.Itoa(count))
	if err := jd.p.BatchCommit(); err != nil {
		jd.e(err)
	}
}

// 按页数获取商品数据
func (jd *JD) getProducts(page int) ([]*Product, error) {
	var products = []*Product{}
	jd.p.ForEach("/pro/"+strconv.Itoa(page)+"/", func(key string, value []byte) {
		var product Product
		err := json.Unmarshal(value, &product)
		if err != nil {
			jd.e(err)
			return
		}
		products = append(products, &product)
	})
	sort.Sort(SortByProductIdx(products))
	return products, nil
}

// 按页数获取我的商品数据回调前端
func (jd *JD) getProductsAndSend(page int) {
	products, err := jd.getProducts(page)
	if err != nil {
		jd.e(err)
		return
	}
	jd.callback(EventProductLoad, products)
}

// 我的试用记录
func (jd *JD) loadMyTrials() {

	defer jd.sendTryTip()
	// 解析试用数据
	count := 0
	parseTrial := func(page int, doc *goquery.Document) {
		doc.Find("#try-list .list-detail-item").Each(func(i int, s *goquery.Selection) {
			aID, _ := s.Attr("activity_id")
			name := s.Find(".p-name a").Text()
			price := s.Find(".p-price").Text()
			img, _ := s.Find(".p-img img").Attr("src")
			product := Product{AID: aID, Name: name, Price: price, Img: "http:" + img}
			bs, err := json.Marshal(product)
			if err != nil {
				jd.e(err)
				return
			}
			jd.p.BatchPutByte("/try/pro/"+strconv.Itoa(page)+"/"+aID, bs)
			jd.p.BatchPutString("/try/"+aID, "1")
			count++
		})
		jd.callback(EventLoadPage, page)
	}
	// 获取我的试用第一页
	body, err := jd.get(MyTrial)
	if err != nil {
		jd.e(err)
		return
	}
	doc, err := jd.document(body)
	if err != nil {
		jd.e(err)
		return
	}
	// 获取总共多少条数据
	totalPage := toInt(doc.Find(".page .p-skip b").Text(), 0)
	jd.p.Batch()
	if totalPage > 0 {
		jd.callback(EventTotalPage, totalPage)
		// 第一页已经取出 直接使用
		parseTrial(1, doc)
		// 从第二页开始循环
		for i := 2; i <= totalPage; i++ {
			body, err := jd.get(MyTrial + "?page=" + strconv.Itoa(i))
			if err != nil {
				jd.e(err)
				continue
			}
			doc, err := jd.document(body)
			if err != nil {
				jd.e(err)
				continue
			}
			parseTrial(i, doc)
		}
	}
	jd.p.BatchPutString("firstLogin", "1")
	jd.p.BatchPutString("/try/count", toStr(count))
	jd.tryCount = count
	if err := jd.p.BatchCommit(); err != nil {
		jd.e(err)
		return
	}
}

// 按页数获取我的试用数据
func (jd *JD) getMyTrial(page int) ([]*Product, error) {
	var products = []*Product{}
	jd.p.ForEach("/try/pro/"+strconv.Itoa(page)+"/", func(key string, value []byte) {
		var product Product
		err := json.Unmarshal(value, &product)
		if err != nil {
			jd.e(err)
			return
		}
		products = append(products, &product)
	})
	sort.Sort(SortByProductIdx(products))
	return products, nil
}

// 按页数获取我的试用数据回调前端
func (jd *JD) getMyTrialAndSend(page int) {
	products, err := jd.getMyTrial(page)
	if err != nil {
		jd.e(err)
		return
	}
	jd.callback(EventMyTrialLoad, products)
}

// 试用
func (jd *JD) try(id string) {
	var product *Product
	// 没有传id，使用本地存的商品
	if id == "" {
		for jd.tryProducts == nil || len(jd.tryProducts) == 0 {
			if jd.currentTryPage > jd.totalPage {
				return
			}

			// 当前试用的页数
			jd.currentTryPage++
			// 获取第一页的数据
			tmpProducts, err := jd.getProducts(jd.currentTryPage)
			if err != nil {
				jd.e(err)
				return
			}
			for _, tmpProduct := range tmpProducts {
				has, _ := jd.p.Has("/try/" + tmpProduct.AID)
				if has {
					continue
				}
				jd.tryProducts = append(jd.tryProducts, tmpProduct)
			}
		}
		product = jd.tryProducts[0]
		jd.tryProducts = append(jd.tryProducts[1:])
		id = product.AID
	}
	if id == "" {
		return
	}
	// 提交申请请求
	body, err := jd.get(TryURL + id)
	if err != nil {
		jd.e(err)
		return
	}
	var result TryResult
	if err := json.Unmarshal(body, &result); err != nil {
		jd.e(err)
		return
	}
	bs, err := json.Marshal(product)
	if err != nil {
		jd.e(err)
		return
	}
	// 试用成功保存数据
	if result.Success || result.Code == "-110" {
		jd.tryCount++
		jd.p.Put("/try/"+id, "1")
		jd.p.PutByte("/try/pro/"+toStr(jd.currentTryPage)+"/"+id, bs)
		jd.p.Put("/try/count", toStr(jd.tryCount))
		jd.sendTryTip()
	} else {
		jd.callback(EventTryFailed, map[string]string{"code": "-1", "id": id, "count": "", "message": result.Message})
	}
}

func (jd *JD) sendTryTip() {
	var buffer bytes.Buffer
	buffer.WriteString("当前商品")
	buffer.WriteString(toStr(jd.proCount))
	buffer.WriteString(" ")
	buffer.WriteString("已申请")
	buffer.WriteString(toStr(jd.tryCount))
	jd.callback(EventTrySuccess, map[string]string{"code": "1", "count": buffer.String()})
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
	jd.callback(EventLoginQR, image)
}

// 获取用户信息
func (jd *JD) getUser() (*User, error) {
	body, err := jd.get(UserURL)
	if err != nil {
		return nil, err
	}
	doc, _ := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	name := doc.Find("#user-info .info-m B").Text()
	avatar, _ := doc.Find("#user-info .u-pic img").Attr("src")
	user := &User{Name: name, Avatar: "http://" + avatar}
	return user, nil
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
			// 登录成功
			go jd.loginSuccess()
			break
		}
		if result.Code == 205 || result.Code == 203 {
			jd.qrCookie = nil
			jd.callback(EventLoginQRTimeOut, nil)
			continue
		}
	}
}

func (jd *JD) get(url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	if jd.thorCookie != nil {
		req.AddCookie(jd.thorCookie)
	}
	req.Header.Add("Referer", "https://try.jd.com")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

// 前端回调
func (jd *JD) callback(cmd string, data interface{}) {
	if data == nil {
		data = ""
	}
	jd.option.Callback(&Channel{Cmd: cmd, Data: data})
}

func (jd *JD) document(bs []byte) (*goquery.Document, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(bs)))
	return doc, err
}

// Send 发送消息
func (jd *JD) Send(cmd string, data interface{}) {
	jd.channal <- Channel{Cmd: cmd, Data: data}
}

// 通道获取数据
func (jd *JD) onChannel() {
	for {
		c := <-jd.channal
		switch c.Cmd {
		case CMDFirst:
			go jd.first()
		case CMDLoginQR:
			go jd.getQRImage()
			go jd.onCheck()
		case CMDProductLoadAll:
			go jd.loadProducts()
		case CMDProductLoad:
			var page int
			if err := json.Unmarshal(c.Data.(json.RawMessage), &page); err == nil {
				go jd.getProductsAndSend(page)
			}
		case CMDMyTrialLoadAll:
			go jd.loadMyTrials()
		case CMDMyTrialLoad:
			switch c.Data.(type) {
			case int:
				go jd.getMyTrialAndSend(c.Data.(int))
			}
		case CMDTry:
			switch c.Data.(type) {
			case string:
				go jd.try(c.Data.(string))
			default:
				go jd.try("")
			}
		}
	}
}

// 当前时间戳
func (jd *JD) getTimestamp() string {
	return strconv.FormatInt(time.Now().Unix(), 10)
}

func (jd *JD) e(err error) {
	log.Println(err)
	jd.callback(EventError, err.Error())
}
