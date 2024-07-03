package main

import (
	"bufio"
	"crypto/tls"
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
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
	defer wg.Done()

	for {
		p, err := readEPPFrame(lconn)
		if err != nil {
			log.Println(err)
			return
		}

		//log.Printf("Passing message: %s", p)

		writeEPPFrame(rconn, p)
	}
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

func readEPPFrame(conn net.Conn) ([]byte, error) {
	r := bufio.NewReader(conn)

	// Read header
	p := make([]byte, 4)
	n, err := r.Read(p)

	if err == io.EOF {
		return []byte{}, errors.New(fmt.Sprintf("Remote host closed the connection %s", conn.RemoteAddr()))
	}

	if err != nil || n != 4 {
		log.Printf("Header length: %d", n)
		log.Printf("Header error: %s", err)
		return []byte{}, errors.New(fmt.Sprintf("Error reading frame header from %s", conn.RemoteAddr()))
	}

	log.Printf("Header raw content: %s", p)

	// Calculate content length
	rawl := binary.BigEndian.Uint32(p)
	log.Printf("Header stated content length: %d", rawl)

	l := rawl - 4
	log.Printf("Calculated content length: %d", l)

	// Read content
	p = make([]byte, l)
	r.Read(p)

	return p, nil
}

func writeEPPFrame(w net.Conn, c []byte) {
	l := len(c) + 4
	header := make([]byte, 4)
	binary.BigEndian.PutUint32(header, uint32(l))

	w.Write(header)
	w.Write(c)
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
