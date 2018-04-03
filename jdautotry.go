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

var (
	// AppName .
	AppName string
	// BuiltAt .
	BuiltAt  string
	win      *astilectron.Window
	jdClient *jd.JD
	tray     *astilectron.Tray
)

func main() {
	initJDClient()
	initUI()
}

func initJDClient() {
	var err error
	jdClient, err = jd.New(&jd.Option{Callback: jdCallback})
	if err != nil {
		log.Println(err.Error())
		return
	}

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
		TrayMenuOptions: []*astilectron.MenuItemOptions{{
			Label: astilectron.PtrStr("关于"),
			OnClick: func(e astilectron.Event) (deleteListener bool) {
				fmt.Println("我来了")
				return
			},
		}},
		TrayOptions: &astilectron.TrayOptions{Image: astilectron.PtrStr("resources/icon.png"), Tooltip: astilectron.PtrStr("京东试用")},
		OnWait: func(_ *astilectron.Astilectron, iw *astilectron.Window, _ *astilectron.Menu, tray *astilectron.Tray, _ *astilectron.Menu) error {
			win = iw
			iw.On(astilectron.EventNameWindowEventMinimize, func(event astilectron.Event) (deleteListener bool) {
				win.Hide()
				return
			})
			tray.On(astilectron.EventNameTrayEventClicked, func(event astilectron.Event) (deleteListener bool) {
				win.Show()
				return
			})
			return nil
		},
		MessageHandler: handleMessages,
		WindowOptions: &astilectron.WindowOptions{
			BackgroundColor: astilectron.PtrStr("#fff"),
			Center:          astilectron.PtrBool(true),
			Height:          astilectron.PtrInt(700),
			Width:           astilectron.PtrInt(800),
			AutoHideMenuBar: astilectron.PtrBool(true),
			Maximizable:     astilectron.PtrBool(false),
		},
	}); err != nil {
		log.Println(errors.Wrap(err, "running bootstrap failed"))
	}
}

// 消息回调
func handleMessages(iw *astilectron.Window, m bootstrap.MessageIn) (payload interface{}, err error) {
	if win == nil {
		win = iw
	}
	jdClient.Send(m.Name, m.Payload)
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
	win.SendMessage(jsonStr, func(m *astilectron.EventMessage) {
	})
}
