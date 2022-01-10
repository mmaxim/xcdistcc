package common

import "fmt"

var NewLineBytes = []byte("\n")
var DefaultCXX = "/Applications/Xcode.app/Contents/Developer/Toolchains/XcodeDefault.xctoolchain/usr/bin/c++"
var DefaultListenPort = 3896
var DefaultListenAddress = fmt.Sprintf("0.0.0.0:%d", DefaultListenPort)
