package main

import (
	"fmt"
	"math/rand"
	"net"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

//Test send and receive message
func TestGetHelloMessage(t *testing.T) {
	port := "8080"
	listener, globals := makeServer(port)
	client := CreateClient("eugene", port)
	client.SendMessage("hello\n")
	mess := client.GetMessage()
	if mess != client.name+":\thello" {
		t.Errorf("Expect '"+client.name+":\thello', got %v\n", mess)
	}
	listener.Close()
	globals.wgMainLoop.Wait()
}

//Test server sends disconnecting message when servers stops
func TestStopServerMessage(t *testing.T) {
	port := "8081"
	listener, globals := makeServer(port)

	client1 := CreateClient("eugene1", port)
	client2 := CreateClient("eugene2", port)

	listener.Close()

	//skip message about client2 connection
	//client1.GetMessage()

	mess1 := client1.GetMessage()
	mess2 := client2.GetMessage()

	if !(mess1 == "Server:\tServer shut down." && mess2 == "Server:\tServer shut down.") {
		t.Errorf(`
Expect:
'Server:	Server shut down.'
'Server:	Server shut down.'
Got:
%v
%v`, mess1, mess2)
	}

	globals.wgMainLoop.Wait()
}

//Message receives first on 3rd client, 1st or 2nd does not block it
func TestNotBlockingMessage(t *testing.T) {
	port := "8082"
	listener, globals := makeServer(port)

	client1 := CreateClient("eugene1", port)
	client2 := CreateClient("eugene2", port)
	client3 := CreateClient("eugene3", port)

	client1.SendMessage("hello\n")

	mess := client3.GetMessage()

	if mess != client1.name+":\thello" {
		t.Errorf("Expect '"+client1.name+":\thello', got %v\n", mess)
	}

	client1.GetMessage()
	client2.GetMessage()

	listener.Close()
	globals.wgMainLoop.Wait()
}

// Three clients send messages, third receives them in send order
func TestMessageOrder(t *testing.T) {
	port := "8083"
	listener, globals := makeServer(port)

	client1 := CreateClient("eugene1", port)
	client2 := CreateClient("eugene2", port)
	client3 := CreateClient("eugene3", port)

	client1.SendMessage("hello from eugene1\n")
	time.Sleep(time.Millisecond)
	client2.SendMessage("hello from eugene2\n")
	time.Sleep(time.Millisecond)
	client3.SendMessage("hello from eugene3\n")
	time.Sleep(time.Millisecond)

	mess1 := client3.GetMessage()
	fmt.Printf("GOT MESS1, %v\n", mess1)
	mess2 := client3.GetMessage()
	fmt.Printf("GOT MESS2, %v\n", mess2)
	mess3 := client3.GetMessage()
	fmt.Printf("GOT MESS3, %v\n", mess3)

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
	globals.wgMainLoop.Wait()
}

// Two clients send messages, third receives them in send order
func TestMessageOrder2Mess(t *testing.T) {
	port := "8084"
	listener, globals := makeServer(port)

	client1 := CreateClient("eugene1", port)

	client1.SendMessage("hello1 from eugene1\n")
	time.Sleep(time.Microsecond)
	client1.SendMessage("hello2 from eugene1\n")

	mess1 := client1.GetMessage()
	fmt.Printf("GOT MESS1, %v\n", mess1)
	mess2 := client1.GetMessage()
	fmt.Printf("GOT MESS2, %v\n", mess2)

	if !(mess1 == (client1.name+":\thello1 from eugene1") &&
		mess2 == (client1.name+":\thello2 from eugene1")) {
		t.Errorf(`
	Expect:
		message 1: '`+client1.name+`:	hello1 from eugene1'
		message 2: '`+client1.name+`:	hello2 from eugene1'
	Got:
		message 1: '%v'
		message 2: '%v'
	`, mess1, mess2)
	}

	listener.Close()
	globals.wgMainLoop.Wait()
}

//Connect 1000 clients, send from random client, read from random client
func TestCreate1000Clients(t *testing.T) {
	port := "8085"
	listener, globals := makeServer(port)
	clients := make([]*Client, 1000)

	for i := 0; i < 1000; i++ {
		clients[i] = CreateClient("eugene"+strconv.Itoa(i), port)
	}

	rand.Seed(time.Now().Unix())
	outCln := rand.Intn(1000)
	inCln1 := rand.Intn(1000)
	inCln2 := rand.Intn(1000)

	clients[outCln].SendMessage("hello\n")
	mess1 := clients[inCln1].GetMessage()
	mess2 := clients[inCln2].GetMessage()
	if !(mess1 == clients[outCln].name+":\thello" && mess2 == clients[outCln].name+":\thello") {
		t.Errorf(`
Expect:
`+clients[outCln].name+`:	hello
`+clients[outCln].name+`:	hello
Got:
%v
%v`, mess1, mess2)
	}
	listener.Close()
	globals.wgMainLoop.Wait()
}

func makeServer(port string) (net.Listener, *Globals) {
	globals := &Globals{
		muConnections: sync.Mutex{},
		connections:   make(map[string]*Client),
		brChan:        make(chan Message, 10000),
		wgMainLoop:    sync.WaitGroup{},
		wgSendMess:    sync.WaitGroup{},
		db:            nil,
	}

	listener, err := CreateServer(port)
	if err != nil {
		panic(err)
	}
	//db = connectDB()
	//initDB()
	go sendAll(globals)
	globals.wgMainLoop.Add(1)
	go MainLoop(globals, listener)
	return listener, globals
}

func CreateClient(login string, port string) *Client {

	conn, err := net.Dial("tcp", ":"+port)
	if err != nil {
		panic(err)
	}
	client := new(Client)
	client.address = conn.RemoteAddr().String()
	client.name = strings.TrimSpace(login)
	client.conn = conn

	_, err = conn.Write([]byte("login:" + login + "\n"))
	if err != nil {
		panic(err)
	}

	//skip *connected* message
	//client.GetMessage()

	return client
}

func (client *Client) SendMessage(mess string) {
	_, err := client.conn.Write([]byte(mess))
	if err != nil {
		panic(err)
	}
}

func (client *Client) GetMessage() string {
	//mess := ""

	////not responding
	//scanner := bufio.NewScanner(client.conn)
	//if scanner.Scan() {
	//	mess = scanner.Text()
	//}

	////not responding
	//reader := bufio.NewReader(client.conn)
	//mess, err := reader.ReadString('\n')
	//if err != nil {
	//	panic(err)
	//}

	//this works
	buf1 := make([]byte, 1)
	buf := make([]byte, 0, 1024)
	lineBreak := byte('\n')
	for buf1[0] != lineBreak {
		_, err := client.conn.Read(buf1)
		if err != nil {
			panic(err)
		}
		buf = append(buf, buf1...)
	}
	mess := string(buf[:len(buf)-1])
	return mess
}
