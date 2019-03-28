package main

import (
	"flag"
	"log"
	"net"
	"github.com/dustin/go-nntp/server"
	"github.com/vharitonsky/iniflags"
	"time"
	"math/rand"
)
var sqlbk = flag.String("sqlengine", "sqlite3", "SQL database engine")
var sqlpath = flag.String("sqldatabase", "./nntp.db", "SQL database path or url")
var listen = flag.String("listen", ":1942", "Listen address")
var hostname = flag.String("hostname", "misconfigured", "Hostname")
var listenhttp = flag.String("httplisten", ":1943", "HTTP listen address")
var disallowsignup = flag.Bool("disallowsignup", false, "Disallow signups (AUTHINFO USER signup:...)")
var adminname = flag.String("admin", "", "Name of administrator user")
var uplinks = flag.String("uplinks", "", "Uplink servers")
var defaultuplinks = flag.String("defaultuplinks", ",73.137.75.34:1942,64.52.84.87:1942", "Default uplink servers (use -defaultuplinks \"\" to disable)")
func main() {
	rand.Seed(time.Now().UnixNano())
	iniflags.Parse()
	backend, err := NewBackend(*sqlbk, *sqlpath)
	if err != nil {
		log.Fatal(err)
	}
	err = backend.Init()
	if err != nil {
		log.Fatal("Initialization failure: " + err.Error())
	}
	backend.NodeName = *hostname
	backend.AllowSignup = !*disallowsignup
	lst, err := net.Listen("tcp", *listen)
	if err != nil {
		log.Fatal(err)
	}
	go StartSync(backend)
	go StartHttp(backend, *listenhttp, *adminname)
	srv := nntpserver.NewServer(backend)
	log.Println("Ready")
	for {
		conn, err := lst.Accept()
		if err != nil {
			log.Println("Warning: failed to accept():", err)
		}
		go srv.Process(conn)
	}
	_ = backend
}
