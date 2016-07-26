package main

import (
	"bufio"
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net"
	"sync"
)

var (
	listenAddr string
	certFile   string
	keyFile    string
	serverAddr string
)

func proxy(lconn, rconn net.Conn, wg sync.WaitGroup) {
	defer wg.Done()

	r := bufio.NewReader(lconn)

	for {
		// Read a line from lconn...
		line, err := r.ReadString('\n')
		if err != nil {
			log.Println(err)
			return
		}

		// ...and write it to rconn...
		n, err := rconn.Write([]byte(line))
		if err != nil {
			log.Println(n, err)
			return
		}
	}
}

func handleConn(lconn net.Conn) {
	defer lconn.Close()

	// Dial out to the server we are proxying to
	log.Println("Dialing out")
	rconn, err := tls.Dial("tcp", serverAddr, &tls.Config{})
	if err != nil {
		log.Println(err)
		return
	}
	defer rconn.Close()

	var wg sync.WaitGroup

	// Run the proxy in both directions
	wg.Add(2)
	go proxy(lconn, rconn, wg)
	go proxy(rconn, lconn, wg)

	wg.Wait()
}

func main() {
	flag.Parse()
	fmt.Println("Listening now..")

	// Load our certificate details
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		log.Fatal(err)
	}

	c := &tls.Config{
		Certificates: []tls.Certificate{cert},
	}

	// Listen for incoming connections
	listener, err := tls.Listen("tcp", listenAddr, c)
	if err != nil {
		log.Fatal(err)
	}

	// Accept all incoming connections and dispatch a handler for each
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Fatal(err)
		}

		go handleConn(conn)
	}
}

func init() {
	flag.StringVar(&listenAddr, "listen", "127.0.0.1:9000", "the address and port to listen on locally")
	flag.StringVar(&certFile, "cert", "./cert.pem", "the filename of the certificate to use")
	flag.StringVar(&keyFile, "key", "./key.pem", "the filename of the key to use")
	flag.StringVar(&serverAddr, "server", "", "the address and port of the server to proxy to")
}
