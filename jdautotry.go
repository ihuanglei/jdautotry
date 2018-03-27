package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/asticode/go-astilectron"
	bootstrap "github.com/asticode/go-astilectron-bootstrap"
	"github.com/ihuanglei/jdautotry/jd"
	"github.com/pkg/errors"
)

// Vars
var (
	AppName string
	BuiltAt string
	w       *astilectron.Window
	j       *jd.JD
)

func main() {
	initJD()
	initUI()
}

func initJD() {
	jd, err := jd.New(&jd.Option{Callback: jdCallback})
	if err != nil {
		fmt.Println(err.Error())
	}
	j = jd
}

func initUI() {
	if err := bootstrap.Run(bootstrap.Options{
		AstilectronOptions: astilectron.Options{
			AppIconDarwinPath:  "resources/icon.icns",
			AppIconDefaultPath: "resources/icon.png",
		},
		Debug:    true,
		Homepage: "index.html",
		MenuOptions: []*astilectron.MenuItemOptions{{
			Label: astilectron.PtrStr("File"),
			SubMenu: []*astilectron.MenuItemOptions{
				{Label: astilectron.PtrStr("About")},
				{Role: astilectron.MenuItemRoleClose},
			},
		}},
		OnWait: func(_ *astilectron.Astilectron, iw *astilectron.Window, _ *astilectron.Menu, _ *astilectron.Tray, _ *astilectron.Menu) error {
			w = iw
			w.OnMessage(jsCallback)
			return nil
		},
		WindowOptions: &astilectron.WindowOptions{
			BackgroundColor: astilectron.PtrStr("#fff"),
			Center:          astilectron.PtrBool(true),
			Height:          astilectron.PtrInt(700),
			Width:           astilectron.PtrInt(700),
		},
	}); err != nil {
		fmt.Println(errors.Wrap(err, "running bootstrap failed"))
	}
}

// 页面回调
func jsCallback(m *astilectron.EventMessage) interface{} {
	var s string
	m.Unmarshal(&s)
	fmt.Println(s)
	if s == "getQRImg" {
		j.Send(&jd.Channel{Cmd: 1})
	} else if s == "getProduct" {
		j.Send(&jd.Channel{Cmd: 2})
	}
	return nil
}

// 京东回调
func jdCallback(c *jd.Channel) {
	bs, err := json.Marshal(c)
	if err != nil {
		fmt.Println(err)
		return
	}
	jsonStr := string(bs)
	log.Println(jsonStr)
	w.SendMessage(jsonStr, func(m *astilectron.EventMessage) {
	})
}
