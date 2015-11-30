// Copyright 2015 Luke Shumaker
// Copyright 2015 Davis Webb

package main

import (
	"locale"
	"os"
	"periwinkle"
	"periwinkle/backend"
	"periwinkle/cfg"

	"lukeshu.com/git/go/libsystemd.git/sd_daemon/lsb"
)

const usage = `
Usage: %[1]s [-c CONFIG_FILE]
       %[1]s -h | --help
Set up the RDBMS schema and seed data.

Options:
  -h, --help      Display this message.
  -c CONFIG_FILE  Specify the configuration file [default: ./config.yaml].`

func main() {
	options := periwinkle.Docopt(usage)

	configFile, uerr := os.Open(options["-c"].(string))
	if uerr != nil {
		periwinkle.LogErr(locale.UntranslatedError(uerr))
		os.Exit(int(lsb.EXIT_NOTCONFIGURED))
	}

	config, err := cfg.Parse(configFile)
	if err != nil {
		periwinkle.LogErr(err)
		os.Exit(int(lsb.EXIT_NOTCONFIGURED))
	}

	err = backend.DbSchema(config.DB)
	if err != nil {
		periwinkle.Logf("Encountered an error while setting up the database schema, not attempting to seed data:")
		periwinkle.LogErr(err)
		os.Exit(int(lsb.EXIT_FAILURE))
	}

	err = backend.DbSeed(config.DB)
	if err != nil {
		periwinkle.Logf("Encountered an error while seeding the database:")
		periwinkle.LogErr(err)
		os.Exit(int(lsb.EXIT_FAILURE))
	}
}
