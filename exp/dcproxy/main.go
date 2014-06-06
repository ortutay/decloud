package main

import (
	"encoding/binary"
	"time"
	"fmt"
	"net"
	"sync"

	"github.com/droundy/goopt"
	"github.com/ortutay/decloud/cred"
	"github.com/ortutay/decloud/msg"
	"github.com/ortutay/decloud/node"
	"github.com/ortutay/decloud/util"
)

// General flags
var fAppDir = goopt.String([]string{"--app-dir"}, "~/.decloud", "")
var fCoinsLower = goopt.String([]string{"--coins-lower"}, "0btc", "")
var fCoinsUpper = goopt.String([]string{"--coins-upper"}, "10btc", "")
var fVerbosity = goopt.Int([]string{"-v", "--verbosity"}, 0, "")

var fListen = goopt.String([]string{"-l", "--listen"}, "", "Listen address")

const (
	CONNECT_TIMEOUT  = 30 * time.Second
	ECHO_BUF_BYTES = 1024
)

const (
	COMMAND_STREAM = 0x01
	COMMAND_BIND   = 0x02
	STATUS_SUCCESS = 0x5a
	STATUS_FAILURE = 0x5b
)

type Socks4ClientRequest struct {
	Version uint8
	Command uint8
	Port    uint16
	Address [4]byte
}

type Socks4ServerResponse struct {
	Null    uint8
	Status  uint8
	Ignored [6]uint8
}

func readUntilNul(conn net.Conn) error {
	char_buffer := [1]byte{1}
	for char_buffer[0] != 0 {
		n, err := conn.Read(char_buffer[0:])
		if err != nil {
			return err
		}
		if n == 0 {
			return fmt.Errorf("Read zero bytes from connection")
		}
	}
	return nil
}

func echoLoop(src net.Conn, dst net.Conn, wg *sync.WaitGroup) {
	var buf [ECHO_BUF_BYTES]byte
	for {
		n, err := src.Read(buf[0:])
		if err != nil {
			break
		}
		fmt.Printf("fwd %v [%v...]\n", n, string(buf[0:util.MinInt(n, 20)]))
		_, err = dst.Write(buf[0:n])
		if err != nil {
			break
		}
	}
	dst.Close()
	wg.Done()
}

func sendSocksResponse(conn net.Conn, status uint8) {
	response := Socks4ServerResponse{Status: status}
	binary.Write(conn, binary.BigEndian, &response)
}

func handleConn(client net.Conn) error {
	defer client.Close()

	request := Socks4ClientRequest{}
	err := binary.Read(client, binary.BigEndian, &request)
	util.Ferr(err)

	// Read and ignore username
	err = readUntilNul(client)
	util.Ferr(err)

	if request.Command != COMMAND_STREAM {
		sendSocksResponse(client, STATUS_FAILURE)
		return fmt.Errorf("Unsupported command: %v", request.Command)
	}

	remoteAddr := fmt.Sprintf("%v:%v", net.IP(request.Address[0:]), request.Port)
	remote, err := net.DialTimeout("tcp", remoteAddr, CONNECT_TIMEOUT)
	if err != nil {
		sendSocksResponse(client, STATUS_FAILURE)
		return fmt.Errorf("Could not connect to %v: %v", remoteAddr, err)
	}
	defer remote.Close()

	sendSocksResponse(client, STATUS_SUCCESS)
	fmt.Printf("connected on %v\n", remoteAddr)

	var wg sync.WaitGroup
	wg.Add(2)
	go echoLoop(remote, client, &wg)
	go echoLoop(client, remote, &wg)
	wg.Wait()

	return nil
}

func main() {
	goopt.Parse(nil)
	if *fListen == "" {
		fmt.Printf("Usage: dcproxy -l ip:port\n")
		return
	}

	ocCred, err := cred.NewOcCredLoadOrCreate("")
	util.Ferr(err)

	bConf, err := util.LoadBitcoindConf("")
	util.Ferr(err)

	pvLower, err := msg.NewPaymentValueParseString(*fCoinsLower)
	util.Ferr(err)
	pvUpper, err := msg.NewPaymentValueParseString(*fCoinsUpper)
	util.Ferr(err)
	coins, err := cred.GetBtcCredInRange(pvLower.Amount, pvUpper.Amount, bConf)
	util.Ferr(err)

	c := node.Client{
		BtcConf: bConf,
		Cred: cred.Cred{
			OcCred:  *ocCred,
			BtcConf: bConf,
			Coins:   *coins,
		},
	}

	fmt.Printf("client: %v\n", c)

	listenAddr := *fListen

	listener, err := net.Listen("tcp", listenAddr)
	util.Ferr(err)

	for {
		fmt.Printf("listening on %v\n", listenAddr)
		conn, err := listener.Accept()
		fmt.Printf("accepted connection\n")
		util.Ferr(err)
		go func() {
			err := handleConn(conn)
			util.Ferr(err)
		}()
	}
}
