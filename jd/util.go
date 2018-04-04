package jd

import (
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
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

func openBrower(uri string) error {

	var commands = map[string]string{
		"windows": "cmd /c start",
		"darwin":  "open",
		"linux":   "xdg-open",
	}

	run, ok := commands[runtime.GOOS]
	if !ok {
		return fmt.Errorf("don't know how to open things on %s platform", runtime.GOOS)
	}

	cmd := exec.Command(run, uri)
	return cmd.Start()
}
