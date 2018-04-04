package jd

import (
	"errors"
	"net/http"
)

const (

	// QRURL 获取二维码
	QRURL = "https://qr.m.jd.com/show?appid=133&size=147"

	// CheckURL 验证二维码
	CheckURL = "https://qr.m.jd.com/check?callback=callback&appid=133"

	// AuthURL 认证ticket
	AuthURL = "https://passport.jd.com/uc/qrCodeTicketValidation"

	// TryURL 试用
	TryURL = "http://try.jd.com/migrate/apply?source=0&activityId="

	// TryProductURL 试用商品
	TryProductURL = "https://try.jd.com/activity/getActivityList"

	// MyTrial 我的试用
	MyTrial = "https://try.jd.com/user/myTrial"

	// Referer .
	Referer = "https://passport.jd.com/new/login.aspx"

	// UserURL 我的信息
	UserURL = "https://i.jd.com/user/info"
)

const (

	// CMDFirst 启动
	CMDFirst = "cmd_first"
	// CMDLoginQR 登录二维码
	CMDLoginQR = "cmd_login_qr"
	// CMDProductTotalPage 商品页数
	CMDProductTotalPage = "cmd_total_page_product"
	// CMDProductLoadAll 所有试用商品
	CMDProductLoadAll = "cmd_product_load_all"
	// CMDProductLoad 分页加载试用商品
	CMDProductLoad = "cmd_product_load"
	// CMDMyTrialLoad 分页加载已申请的商品
	CMDMyTrialLoad = "cmd_my_trial_load"
	// CMDMyTrialLoadAll 所有已申请商品
	CMDMyTrialLoadAll = "cmd_my_trial_load_all"
	// CMDTry 申请试用
	CMDTry = "cmd_try"

	// EventError 错误
	EventError = "event_error"
	// EventProductLoad .
	EventProductLoad = "event_product_load"
	// EventMyTrialLoad .
	EventMyTrialLoad = "event_my_trial_load"
	// EventLoginQR .
	EventLoginQR = "event_login_qr"
	// EventLoginQRTimeOut 登录二维码过期
	EventLoginQRTimeOut = "event_login_qr_timeout"
	// EventTotalPage 总页数
	EventTotalPage = "event_total_page"
	// EventLoadPage 已加载页数
	EventLoadPage = "event_load_page"
	// EventTryAlready 商品已申请
	EventTryAlready = "event_try_already"
	// EventTrySuccess 商品申请成功
	EventTrySuccess = "event_try_success"
	// EventTryFailed 商品申请失败
	EventTryFailed = "event_try_failed"
	// EventMyInfo 我的信息
	EventMyInfo = "event_my_info"
)

// Cookie 结构
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

// Channel 通道结构
type Channel struct {
	Cmd  string      `json:"cmd"`
	Data interface{} `json:"data"`
}

// Product 商品结构
type Product struct {
	AID   string `json:"id"`
	Name  string `json:"name"`
	Price string `json:"price"`
	Img   string `json:"img"`
	Page  int    `json:"page"`
	Idx   int    `json:"idx"`
}

// SortByProductIdx 排序
type SortByProductIdx []*Product

func (a SortByProductIdx) Len() int {
	return len(a)
}

func (a SortByProductIdx) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a SortByProductIdx) Less(i, j int) bool {
	return a[i].Idx < a[j].Idx
}

// User 用户信息结构
type User struct {
	Name       string `json:"name"`
	Avatar     string `json:"avatar"`
	FirstLogin string `json:"firstLogin"`
}

// Option 实例化参数结构
type Option struct {
	Callback func(*Channel)
}

// TryResult 试用返回结果结构
type TryResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Code    string `json:"code"`
}

// JD 机构
type JD struct {
	qrCookie       *jdCookie
	thorCookie     *http.Cookie
	option         *Option
	channal        chan Channel
	currentTryPage int //当前试用页数
	totalPage      int
	tryProducts    []*Product // 当前试用商品
	p              *Persistence
	user           *User
	proCount       int
	tryCount       int
}

// New 创建实例
func New(option *Option) (*JD, error) {
	if option == nil || option.Callback == nil {
		return nil, errors.New("参数不能为空")
	}
	jd := JD{option: option, channal: make(chan Channel), p: NewPersistence()}
	err := jd.p.Open()
	if err != nil {
		return nil, err
	}
	go jd.onChannel()
	return &jd, nil
}
