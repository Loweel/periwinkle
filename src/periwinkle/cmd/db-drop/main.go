// Copyright 2015 Luke Shumaker
// Copyright 2015 Davis Webb

package main

import (
	"fmt"
	"locale"
	"os"
	"periwinkle"
	"periwinkle/backend"
	"periwinkle/cfg"

	docopt "github.com/LukeShu/go-docopt"
	"lukeshu.com/git/go/libsystemd.git/sd_daemon/lsb"
)

var usage = fmt.Sprintf(`Periwinkle db-drop

Usage: %[1]s [-c CONFIG_FILE]
       %[1]s -h | --help
Drop the tables in the RDBMS, in the correct order.

Options:
  -h --help       Display this message.
  -c CONFIG_FILE  Specify the configuration file [default: ./config.yaml].`,
	os.Args[0])

func main() {
	options, _ := docopt.Parse(usage, os.Args[1:], true, "", false, true)

	configFile, err1 := os.Open(options["-c"].(string))
	if err1 != nil {
		periwinkle.LogErr(locale.UntranslatedError(err1))
		os.Exit(int(lsb.EXIT_NOTCONFIGURED))
	}

	config, err2 := cfg.Parse(configFile)
	if err2 != nil {
		periwinkle.LogErr(err2)
		os.Exit(int(lsb.EXIT_NOTCONFIGURED))
	}

	backend.DbDrop(config.DB)
}
