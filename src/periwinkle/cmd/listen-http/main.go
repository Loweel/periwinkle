// Copyright 2015 Luke Shumaker

package main

import (
	"locale"
	"net"
	"os"
	"os/signal"
	"periwinkle"
	"periwinkle/cmdutil"
	"periwinkle/httpapi"
	"strconv"
	"strings"
	"syscall"

	sd "lukeshu.com/git/go/libsystemd.git/sd_daemon"
	"lukeshu.com/git/go/libsystemd.git/sd_daemon/lsb"
)

const usage = `
Usage: %[1]s [-c CONFIG_FILE] [ADDR_TYPE] [ADDR]
       %[1]s -h | --help
Do the HTTP that you do, baby.

Address types are "tcp", "tcp4", "tcp6", "unix", and "fd".

If only one argument is given, if it matches one of type it is taken
to be the type; otherwise it is taken as an address.

  | type            | default address |
  |-----------------+-----------------|
  | tcp, tcp4, tcp6 | :8080           |
  | unix            | ./http.sock     |
  | fd              | stdin           |

If only the address is given, the type is assumed to be "unix" if it
contains a slash, "fd" if it only contains numeric digits or matches
one of the special "fd" values (below), or "tcp" otherwise.  If no
arguments are given, "tcp" is used.

The address for "fd" is numeric; however, there are several special
cases. "stdin", "stdout", and "stderr" are aliases for "0", "1", and
2", respectively. "systemd" causes it to look up the file descriptor
from systemd socket-activation.

Options:
  -h, --help      Display this message.
  -c CONFIG_FILE  Specify the configuration file [default: ./config.yaml].`

func parseArgs(args []string) net.Listener {
	var stype, saddr string

	switch len(args) {
	case 0:
		stype = "tcp"
		saddr = ":8080"
	case 1:
		switch args[0] {
		case "tcp", "tcp4", "tcp6":
			stype = args[0]
			saddr = ":8080"
		case "unix":
			stype = args[0]
			saddr = "./http.sock"
		case "fd":
			stype = args[0]
			saddr = "stdin"
		case "systemd", "stdin", "stdout", "stderr":
			stype = "fd"
			saddr = args[0]
		default:
			if strings.ContainsRune(args[0], '/') {
				stype = "unix"
			} else if _, err := strconv.Atoi(args[0]); err == nil {
				stype = "fd"
			} else {
				stype = "tcp"
			}
			saddr = args[0]
		}
	case 2:
		stype = args[0]
		saddr = args[1]
	default:
		periwinkle.Logf(usage, os.Args[0])
		os.Exit(int(lsb.EXIT_INVALIDARGUMENT))
	}

	var socket net.Listener
	var err locale.Error

	if stype == "fd" {
		switch saddr {
		case "systemd":
			socket, err = sdGetSocket()
		case "stdin":
			socket, err = listenfd(0, "/dev/stdin")
		case "stdout":
			socket, err = listenfd(1, "/dev/stdout")
		case "stderr":
			socket, err = listenfd(2, "/dev/stderr")
		default:
			n, uerr := strconv.Atoi(saddr)
			if uerr == nil {
				socket, err = listenfd(n, "/dev/fd/"+saddr)
			}
		}
	} else {
		var uerr error
		socket, uerr = net.Listen(stype, saddr)
		err = locale.UntranslatedError(uerr)
		if tcpsock, ok := socket.(*net.TCPListener); ok {
			socket = tcpKeepAliveListener{tcpsock}
		}
	}
	if err != nil {
		periwinkle.LogErr(err)
		os.Exit(int(lsb.EXIT_FAILURE))
	}
	return socket
}

func listenfd(fd int, name string) (net.Listener, locale.Error) {
	l, e := net.FileListener(os.NewFile(uintptr(fd), name))
	return l, locale.UntranslatedError(e)
}

func sdGetSocket() (socket net.Listener, err locale.Error) {
	fds := sd.ListenFds(true)
	if fds == nil {
		err = locale.Errorf("Failed to aquire sockets from systemd")
		return
	}
	if len(fds) != 1 {
		err = locale.Errorf("Wrong number of sockets from systemd: expected %d but got %d", 1, len(fds))
		return
	}
	socket, uerr := net.FileListener(fds[0])
	err = locale.UntranslatedError(uerr)
	fds[0].Close()
	return
}

func main() {
	options := cmdutil.Docopt(usage)

	args := []string{}
	if options["ADDR_TYPE"] != nil {
		args = append(args, options["ADDR_TYPE"].(string))
	}
	if options["ADDR"] != nil {
		args = append(args, options["ADDR"].(string))
	}
	socket := parseArgs(args)

	config := cmdutil.GetConfig(options["-c"].(string))

	signals := make(chan os.Signal)
	signal.Notify(signals, syscall.SIGTERM, syscall.SIGHUP)

	periwinkle.Logf("Ready; listening")
	sd.Notify(false, "READY=1")

	done := make(chan uint8)
	server := httpapi.MakeServer(socket, config)
	server.Start()
	go func() {
		err := server.Wait()
		if err != nil {
			periwinkle.LogErr(err)
			done <- 1
		} else {
			done <- 0
		}
	}()

	for {
		select {
		case sig := <-signals:
			switch sig {
			case syscall.SIGTERM:
				sd.Notify(false, "STOPPING=1")
				server.Stop()
			case syscall.SIGHUP:
				sd.Notify(false, "RELOADING=1")
				// TODO: reload configuration file
				sd.Notify(false, "READY=1")
			}
		case status := <-done:
			os.Exit(int(status))
		}
	}
}
