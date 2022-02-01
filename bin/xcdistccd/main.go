package main

import (
	"errors"
	"flag"
	"log"
	"os"
	"strconv"

	"mmaxim.org/xcdistcc/common"
	"mmaxim.org/xcdistcc/server"
)

type Options struct {
	Address      string
	MaxWorkers   int
	MaxQueueSize int
	CxxPath      string
	KeyPair      *common.KeyPair
}

func (o Options) check() {}

func usage() {
	flag.Usage()
	os.Exit(3)
}

func envIntValue(name string, def int) (ret int) {
	ret = def
	envStr := os.Getenv(name)
	parsed, err := strconv.ParseInt(envStr, 0, 0)
	if err == nil {
		ret = int(parsed)
	}
	return ret
}

func configKeypair() (*common.KeyPair, error) {
	publicStr := os.Getenv("XCDISTCCD_PUBLICKEY")
	if len(publicStr) == 0 {
		return nil, nil
	}
	privateStr := os.Getenv("XCDISTCCD_PRIVATEKEY")
	if len(privateStr) == 0 {
		return nil, errors.New("must supply private key with public key")
	}
	return common.NewKeyPairFromString(privateStr, publicStr)
}

func config() (opts Options) {

	flag.StringVar(&opts.Address, "address", os.Getenv("XCDISTCCD_ADDRESS"),
		"(optional) listen address (XCDISTCCD_ADDRESS env)")
	flag.IntVar(&opts.MaxWorkers, "max-workers", envIntValue("XCDISTCCD_MAXWORKERS", 5),
		"(optional) max compile workers (XCDISTCCD_MAXWORKERS env)")
	flag.IntVar(&opts.MaxQueueSize, "max-queue-size", envIntValue("XCDISTCCD_MAXQUEUESIZE", 500),
		"(optional) max compile queue size (XCDISTCCD_MAXQUEUESIZE env)")
	flag.StringVar(&opts.CxxPath, "cxx-path", os.Getenv("XCDISTCCD_CXXPATH"),
		"(optional) xcode c++ compiler path (XCDISTCCD_CXXPATH env)")
	flag.Parse()
	opts.check()

	var err error
	opts.KeyPair, err = configKeypair()
	if err != nil {
		log.Printf("unable to configure key pair: %s", err)
		os.Exit(3)
	}
	return opts
}

func getOptional(val, def string) string {
	if len(val) == 0 {
		return def
	}
	return val
}

func main() {
	opts := config()
	logger := common.NewStdLogger()
	runner := server.NewRunner(opts.MaxWorkers, opts.MaxQueueSize, logger)
	listener := server.NewListener(runner, getOptional(opts.Address, common.DefaultListenAddress),
		opts.KeyPair, logger)
	if err := listener.Run(); err != nil {
		log.Fatalf("error running listener: %s", err)
	}
}
