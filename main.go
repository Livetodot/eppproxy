package main

import (
	"crypto/tls"
	"flag"
	"io"
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

func proxy(lconn, rconn net.Conn, wg *sync.WaitGroup) {

	defer lconn.Close()
	defer rconn.Close()
	
	// Copy data from source to destination
	_, err := io.Copy(rconn, lconn)
	
	if err != nil {
	  log.Printf("Error copying data: %s", err)
	}

	return
}

func handleConn(lconn net.Conn) {
	defer lconn.Close()

	log.Printf("Accepted connection from %s", lconn.RemoteAddr())

	// Dial out to the server we are proxying to
	log.Printf("Dialing out to remote %s", serverAddr)
	rconn, err := tls.Dial("tcp", serverAddr, &tls.Config{})
	if err != nil {
		log.Println(err)
		return
	}
	defer rconn.Close()

	var wg sync.WaitGroup

	log.Printf("Starting proxy session between %s and %s", lconn.RemoteAddr(), rconn.RemoteAddr())
	go proxy(lconn, rconn, &wg)
	go proxy(rconn, lconn, &wg)

	// Block until ONE of the goroutines ends
	wg.Add(1)
	wg.Wait()

	// Add 1 to the wg so that when the other goroutine closes and decrements
	// the wg we don't end with a negative wg count which causes a panic
	wg.Add(1)

	log.Printf("Dead connection, closing proxy session between %s and %s", lconn.RemoteAddr(), rconn.RemoteAddr())
}

func main() {
	flag.Parse()
	log.Printf("Listening for connections on %s", listenAddr)

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
			log.Println(err)
			continue
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
