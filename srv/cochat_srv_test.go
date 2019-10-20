package main

import (
	"bufio"
	"net"
	"strings"
	"sync"
	"testing"
	"time"
)

//Test send and receive message
func TestGetHelloMessage(t *testing.T) {
	port := "8080"
	listener := makeServer(port)

	client, err := CreateClient("eugene", port)
	if err != nil {
		panic(err)
	}
	err = client.SendMessage("hello\n")
	if err != nil {
		panic(err)
	}
	mess, err := client.GetMessage()
	if err != nil {
		panic(err)
	}
	if mess != client.name+":\thello" {
		t.Errorf("Expect '"+client.name+":\thello', got %v\n", mess)
	}

	listener.Close()
	prepareForNextTest()
}

//Test server sends disconnecting message when servers stops
func TestStopServerMessage(t *testing.T) {
	port := "8081"
	listener := makeServer(port)

	client1, err := CreateClient("eugene1", port)
	if err != nil {
		panic(err)
	}
	client2, err := CreateClient("eugene2", port)
	if err != nil {
		panic(err)
	}

	listener.Close()
	time.Sleep(3 * time.Second)

	_, err = client1.GetMessage()
	if err != nil {
		panic(err)
	}
	mess1, err := client1.GetMessage()
	if err != nil {
		panic(err)
	}
	mess2, err := client2.GetMessage()
	if err != nil {
		panic(err)
	}
	if !(mess1 == "Server:\tServer shut down." && mess2 == "Server:\tServer shut down.") {
		t.Errorf(`
Expect:
'Server:	Server shut down.'
'Server:	Server shut down.'
Got:
%v
%v`, mess1, mess2)
	}

	prepareForNextTest()
}

//Message receives first on 3rd client, 1st or 2nd does not block it
func TestNotBlockingMessage(t *testing.T) {
	port := "8082"
	listener := makeServer(port)

	client1, err := CreateClient("eugene1", port)
	if err != nil {
		panic(err)
	}
	client2, err := CreateClient("eugene2", port)
	if err != nil {
		panic(err)
	}
	client3, err := CreateClient("eugene3", port)
	if err != nil {
		panic(err)
	}

	err = client1.SendMessage("hello\n")
	if err != nil {
		panic(err)
	}
	mess, err := client3.GetMessage()
	if err != nil {
		panic(err)
	}
	if mess != client1.name+":\thello" {
		t.Errorf("Expect '"+client1.name+":\thello', got %v\n", mess)
	}

	_, err = client1.GetMessage()
	if err != nil {
		panic(err)
	}
	_, err = client2.GetMessage()
	if err != nil {
		panic(err)
	}

	listener.Close()
	prepareForNextTest()
}

// Two clients send messages, third receives them in send order
func TestMessageOrder(t *testing.T) {
	port := "8083"
	listener := makeServer(port)

	client1, err := CreateClient("eugene1", port)
	if err != nil {
		panic(err)
	}
	client2, err := CreateClient("eugene2", port)
	if err != nil {
		panic(err)
	}
	client3, err := CreateClient("eugene3", port)
	if err != nil {
		panic(err)
	}

	err = client1.SendMessage("hello from eugene1\n")
	if err != nil {
		panic(err)
	}
	time.Sleep(time.Microsecond)
	err = client2.SendMessage("hello from eugene2\n")
	if err != nil {
		panic(err)
	}
	time.Sleep(time.Microsecond)
	err = client3.SendMessage("hello from eugene3\n")
	if err != nil {
		panic(err)
	}
	mess1, err := client3.GetMessage()
	if err != nil {
		panic(err)
	}

	mess2, err := client3.GetMessage()
	if err != nil {
		panic(err)
	}

	mess3, err := client3.GetMessage()
	if err != nil {
		panic(err)
	}

	if !(mess1 == (client1.name+":\thello from eugene1") &&
		mess2 == (client2.name+":\thello from eugene2") &&
		mess3 == (client3.name+":\thello from eugene3")) {
		t.Errorf(`
Expect:
	message 1: '`+client1.name+`:	hello from eugene1'
	message 2: '`+client2.name+`:	hello from eugene2'
	message 3: '`+client3.name+`:	hello from eugene3'
Got:
	message 1: '%v'
	message 2: '%v'
	message 3: '%v'
`, mess1, mess2, mess3)
	}

	listener.Close()
	prepareForNextTest()
}

func prepareForNextTest() {
	for c := range connections {
		muConnections.Lock()
		delete(connections, c)
		muConnections.Unlock()
	}
	close(brChan)
	brChan = make(chan Message, 10000)
	wg = sync.WaitGroup{}
}

func makeServer(port string) net.Listener {
	listener, err := CreateServer(port)
	if err != nil {
		panic(err)
	}
	go sendAll()
	wg.Add(1)
	go MainLoop(listener)
	return listener
}

func CreateClient(login string, port string) (*Client, error) {

	conn, err := net.Dial("tcp", ":"+port)
	if err != nil {
		return nil, err
	}
	client := new(Client)
	client.address = conn.RemoteAddr().String()
	client.name = strings.TrimSpace(login)
	client.conn = conn

	_, err = conn.Write([]byte("login:" + login + "\n"))
	if err != nil {
		return nil, err
	}

	_, err = client.GetMessage()
	if err != nil {
		return nil, err
	}
	return client, nil
}

func (client *Client) SendMessage(mess string) error {
	_, err := client.conn.Write([]byte(mess))
	return err
}

func (client *Client) GetMessage() (string, error) {
	mess := ""
	scanner := bufio.NewScanner(client.conn)
	if scanner.Scan() {
		mess = scanner.Text()
	}
	return mess, nil
}
