package jd

import (
	"errors"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

func toInt(s string, defInt int) (i int) {
	i, err := strconv.Atoi(s)
	if err != nil {
		i = defInt
	}
	return
}

func toStr(i int) (s string) {
	s = strconv.Itoa(i)
	return
}

// OpenBrower .
func OpenBrower(uri string) error {
	var ss string
	switch runtime.GOOS {
	case "windows":
		ss = "cmd /c start " + uri
	case "darwin":
		ss = "open " + uri
	case "linux":
		ss = "xdg-open " + uri
	default:
		return errors.New("Command Not Found")
	}
	args := strings.Split(ss, " ")
	cmd := exec.Command(args[0], args[1:]...)
	return cmd.Start()
}
