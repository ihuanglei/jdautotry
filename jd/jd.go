package jd

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
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
		totalPage, _ := strconv.Atoi(totalPageStr)
		jd.totalPage = totalPage
		// 总共商品页数
		jd.callback(CMDProductTotalPage, totalPage)
		// 总页数消息
		jd.callback(CMDTotalPage, totalPage)
		// 加载完成消息
		jd.callback(CMDLoadPage, totalPage)
		// 第一页数据
		jd.getProductsAndSend(1)
	}
	return
}

func (jd *JD) loginSuccess() {
	user, err := jd.getUser()
	if err != nil {
		jd.e(err)
		return
	}
	// 是否第一次登录
	firstLogin, err := jd.p.Get("firstLogin")
	if err != nil || firstLogin == "" {
		user.FirstLogin = ""
	} else {
		user.FirstLogin = "1"
	}
	jd.user = user
	jd.callback(CMDMyInfo, jd.user)

	proCount, err := jd.p.Get("/pro/count")
	if err == nil {
		jd.proCount, _ = strconv.Atoi(proCount)
	}
	tryCount, err := jd.p.Get("/try/count")
	if err == nil {
		jd.tryCount, _ = strconv.Atoi(tryCount)
	}
	jd.callback(CMDTrySuccess, map[string]string{"code": "1", "count": tryCount + "/" + proCount})
}

// 拉取商品
func (jd *JD) loadProducts() {
	count := 0
	// 延时返回第一页数据
	defer jd.getProductsAndSend(1)
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

	totalPage = 3
	jd.totalPage = totalPage
	jd.p.Put("totalPage", strconv.Itoa(totalPage))
	// 回调前台loading共多少页
	jd.callback(CMDTotalPage, totalPage)
	// 商品有多少页
	jd.callback(CMDProductTotalPage, totalPage)
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
		jd.callback(CMDLoadPage, page)
	}
	// 第一页已经取出 直接使用
	parseProduct(1, doc)
	// 从第二页开始循环
	for i := 2; i <= totalPage; i++ {
		body, err := jd.get(TryProductURL + "?page=" + strconv.Itoa(i))
		if err != nil {
			jd.e(err)
			return
		}
		doc, err := jd.document(body)
		if err != nil {
			jd.e(err)
			return
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
	jd.callback(CMDProductLoad, products)
}

// 我的试用记录
func (jd *JD) loadMyTrials() {
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
		jd.callback(CMDLoadPage, page)
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
	totalPage, err := strconv.Atoi(doc.Find(".page .p-skip b").Text())
	if err != nil {
		jd.e(errors.New("获取我的试用失败"))
		return
	}
	jd.callback(CMDTotalPage, totalPage)
	jd.p.Batch()
	// 第一页已经取出 直接使用
	parseTrial(1, doc)
	// 从第二页开始循环
	for i := 2; i <= totalPage; i++ {
		body, err := jd.get(MyTrial + "?page=" + strconv.Itoa(i))
		if err != nil {
			jd.e(err)
			return
		}
		doc, err := jd.document(body)
		if err != nil {
			jd.e(err)
			return
		}
		parseTrial(i, doc)
	}
	jd.p.BatchPutString("firstLogin", "1")
	jd.p.BatchPutString("/try/count", strconv.Itoa(count))
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
	jd.callback(CMDMyTrialLoad, products)
}

// 试用
func (jd *JD) try(id string) {
	var product *Product
	// 没有传id，使用本地存的商品
	if id == "" {
		for jd.tryProducts == nil || len(jd.tryProducts) == 0 {
			if jd.currentTryPage > jd.totalPage {
				fmt.Println("end...")
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
		jd.p.Put("/try/"+id, "1")
		jd.p.PutByte("/try/pro/"+strconv.Itoa(jd.currentTryPage)+"/"+id, bs)
		jd.tryCount++
		proCount := strconv.Itoa(jd.proCount)
		tryCount := strconv.Itoa(jd.tryCount)
		jd.p.Put("/try/count", tryCount)
		jd.callback(CMDTrySuccess, map[string]string{"code": "1", "id": id, "count": tryCount + "/" + proCount})
	} else {
		jd.callback(CMDTryFailed, map[string]string{"code": "-1", "id": id, "count": "", "message": result.Message})
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
	jd.callback(CMDLoginQR, image)
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
			jd.callback(CMDLoginQRTimeOut, nil)
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
	jd.callback(CMDError, err.Error())
}
