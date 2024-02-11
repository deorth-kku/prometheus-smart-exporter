package main

import (
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

type httpserver struct {
	*http.Server
	*http.ServeMux
}

func NewHttpServer() (server httpserver) {
	server.ServeMux = http.NewServeMux()
	server.Server = &http.Server{Handler: server.ServeMux}
	return
}

func (se *httpserver) ListenAndServe(listen string) (err error) {
	var lis net.Listener
	if filepath.IsAbs(listen) {
		f, m, found := strings.Cut(listen, ",")
		if found {
			var t uint64
			t, err = strconv.ParseUint(m, 8, 32)
			if err != nil {
				err = fmt.Errorf("failed to parse mode %s :%s", m, err)
				return
			}
			umask := int(^t)
			syscall.Umask(umask)
		}
		addr := &net.UnixAddr{Name: f, Net: "unix"}
		lis, err = net.ListenUnix("unix", addr)
	} else {
		var addr *net.TCPAddr
		addr, err = net.ResolveTCPAddr("tcp", listen)
		if err != nil {
			return
		}
		lis, err = net.ListenTCP("tcp", addr)
	}
	if err != nil {
		return
	}
	err = se.Server.Serve(lis)
	if err == http.ErrServerClosed {
		err = nil
	}
	return
}
