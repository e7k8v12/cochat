package main

import (
	"bufio"
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
)

func main() {

	srvIp := flag.String("ip", "127.0.0.1", "IP address of a server")
	srvPort := flag.String("port", "8080", "Port of a server")
	flag.Parse()

	reader := bufio.NewReader(os.Stdin)

	fmt.Print("login: ")
	login, err := reader.ReadString('\n')
	if err != nil {
		panic(err)
	}

	conn, err := net.Dial("tcp", *srvIp+":"+*srvPort)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Connected to %v:%v\n", *srvIp, *srvPort)
	defer conn.Close()

	conn.Write([]byte("login:" + login))

	go getMessages(conn)
	login = strings.TrimSpace(login)
	for {
		mess, _ := reader.ReadString('\n')
		conn.Write([]byte(mess))
		if strings.TrimSpace(strings.ToLower(mess)) == "exit" {
			conn.Close()
			fmt.Print("Client closed\n")
			break
		}
	}
}

func getMessages(conn net.Conn) {
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		text := scanner.Text()
		fmt.Println(text)
	}
	fmt.Println("Server lost.")
	os.Exit(0)
}
