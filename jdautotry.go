package main

import (
	"encoding/json"
	"log"

	"github.com/asticode/go-astilectron"
	bootstrap "github.com/asticode/go-astilectron-bootstrap"
	"github.com/ihuanglei/jdautotry/jd"
	"github.com/pkg/errors"
)

var (
	// AppName .
	AppName string
	// BuiltAt .
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
		log.Println(err.Error())
		return
	}
	j = jd
}

func initUI() {
	if err := bootstrap.Run(bootstrap.Options{
		Asset:         Asset,
		RestoreAssets: RestoreAssets,
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
			return nil
		},
		MessageHandler: handleMessages,
		WindowOptions: &astilectron.WindowOptions{
			BackgroundColor: astilectron.PtrStr("#fff"),
			Center:          astilectron.PtrBool(true),
			Height:          astilectron.PtrInt(700),
			Width:           astilectron.PtrInt(700),
		},
	}); err != nil {
		log.Println(errors.Wrap(err, "running bootstrap failed"))
	}
}

// 消息回调
func handleMessages(_ *astilectron.Window, m bootstrap.MessageIn) (payload interface{}, err error) {
	if m.Name == "getQRImg" {
		j.Send(&jd.Channel{Cmd: 1})
	} else if m.Name == "getProduct" {
		j.Send(&jd.Channel{Cmd: 2})
	} else if m.Name == "tryProduct" {
		var id string
		if err = json.Unmarshal(m.Payload, &id); err != nil {
			jdCallback(&jd.Channel{Cmd: -100, Data: err.Error()})
			return
		}
		j.Send(&jd.Channel{Cmd: 5, Data: id})
	}
	return
}

// 京东回调
func jdCallback(c *jd.Channel) {
	bs, err := json.Marshal(c)
	if err != nil {
		log.Println(err)
		return
	}
	jsonStr := string(bs)
	// log.Println(jsonStr)
	w.SendMessage(jsonStr, func(m *astilectron.EventMessage) {
	})
}
