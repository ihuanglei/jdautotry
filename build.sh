mv bind.go bind.go.bak
$GOPATH/bin/astilectron-bundler -v
mv bind.go.bak bind.go
rm bind_darwin_amd64.go
