@ECHO OFF
ECHO . build
RENAME bind.go bind.go.bak
set APPDATA=""
%GOPATH%\bin\astilectron-bundler.exe -v

ECHO . clean

RENAME bind.go.bak bind.go
DEL bind_windows_amd64.go
