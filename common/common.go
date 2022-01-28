package common

import (
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"strings"
)

var NewLineBytes = []byte("\n")
var DefaultCXX = "/Applications/Xcode.app/Contents/Developer/Toolchains/XcodeDefault.xctoolchain/usr/bin/c++"
var DefaultListenPort = 3896
var DefaultListenAddress = fmt.Sprintf("0.0.0.0:%d", DefaultListenPort)

type Pair[T any, S any] struct {
	first  T
	second S
}

func MakePair[T any, S any](t T, s S) (res Pair[T, S]) {
	res.first = t
	res.second = s
	return res
}

func RandBytes(length int) ([]byte, error) {
	var n int
	var err error
	buf := make([]byte, length)
	if n, err = rand.Read(buf); err != nil {
		return nil, err
	}
	// rand.Read uses io.ReadFull internally, so this check should never fail.
	if n != length {
		return nil, fmt.Errorf("RandBytes got too few bytes, %d < %d", n, length)
	}
	return buf, nil
}

func RandString(prefix string, numbytes int) (string, error) {
	buf, err := RandBytes(numbytes)
	if err != nil {
		return "", err
	}
	str := base32.StdEncoding.EncodeToString(buf)
	if prefix != "" {
		str = strings.Join([]string{prefix, str}, "")
	}
	return str, nil
}
