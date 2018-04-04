@ECHO OFF
echo . build
RENAME bind.go bind.go.bak
set APPDATA=""
%GOPATH%\bin\astilectron-bundler.exe -v
RENAME bind.go.bak bind.go
echo . build ok