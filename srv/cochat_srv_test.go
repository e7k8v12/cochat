package main

import (
	"bufio"
	"net"
	"strings"
	"testing"
)

type testClient struct {
	Client
	login string
}

//Test send and receive message
func TestGetHelloMessage(t *testing.T) {
	port := "8081"
	makeServer(port)

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
	if mess != client.login+":\thello" {
		t.Errorf("Expect '"+client.login+":\thello', got %v\n", mess)
	}

	prepareForNextTest()
}

//Message receives first on 3rd client, 1st or 2nd does not block it
func TestNotBlockingMessage(t *testing.T) {
	port := "8082"
	makeServer(port)

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
	if mess != client1.login+":\thello" {
		t.Errorf("Expect '"+client1.login+":\thello', got %v\n", mess)
	}

	_, err = client2.GetMessage()
	if err != nil {
		panic(err)
	}

	prepareForNextTest()
}

// Two clients send messages, third receives them in send order
func TestMessageOrder(t *testing.T) {
	port := "8083"
	makeServer(port)

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
	err = client2.SendMessage("hello from eugene2\n")
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

	if !(mess1 == client1.login+":\thello from eugene1" && mess2 == client2.login+":\thello from eugene2") {
		t.Errorf(`Expect:
	message 1: '`+client1.login+`:	hello from eugene1'
	message 2: '`+client2.login+`:	hello from eugene2'
Got:
	message 1: '%v'
	message 2: '%v'`, mess1, mess2)
	}

	prepareForNextTest()
}

func prepareForNextTest() {
	for c := range connections {
		muConnections.Lock()
		delete(connections, c)
		muConnections.Unlock()
	}
	close(brChan)
	brChan = make(chan Message, 1000)
}

func makeServer(port string) {
	listener, err := CreateServer(port)
	if err != nil {
		panic(err)
	}
	go MainLoop(listener)
}

func CreateClient(login string, port string) (*testClient, error) {

	conn, err := net.Dial("tcp", ":"+port)
	if err != nil {
		return nil, err
	}
	client := new(testClient)
	client.address = conn.RemoteAddr().String()
	client.login = strings.TrimSpace(login)
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

func (client *testClient) SendMessage(mess string) error {
	_, err := client.conn.Write([]byte(mess))
	return err
}

func (client *testClient) GetMessage() (string, error) {
	mess := ""
	scanner := bufio.NewScanner(client.conn)
	if scanner.Scan() {
		mess = scanner.Text()
	}
	return mess, nil
}
