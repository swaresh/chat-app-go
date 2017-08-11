package main

import (
	"bufio"
	"net"
	"fmt"
	"bytes"
	"log"
	"os/exec"
	"strings"
	"net/http"
	"github.com/gin-gonic/gin"
)

type Client struct {
	incoming chan string
	outgoing chan string
	reader   *bufio.Reader
	writer   *bufio.Writer
}

func (client *Client) Read() {
	for {

		line, _ := client.reader.ReadString('\n')
		client.incoming <- line
		line = strings.TrimSpace(line)
		cmd := exec.Command("go", "run", "agent.go")
		switch line {
			case "build agent" :
				cmd = exec.Command("go", "run", "agent.go")
			case "build asset" :
				cmd = exec.Command("go", "run", "asset.go")
			case "build performance" :
				cmd = exec.Command("go", "run", "performance.go")
			}
				cmd.Stdin = strings.NewReader("")
				var out bytes.Buffer
				cmd.Stdout = &out
				err := cmd.Run()
				if err != nil {
					log.Fatal(err)
				}
				parts := strings.Split(out.String(), "\n")
				for _, part := range parts{
					client.outgoing <- part
					client.outgoing <- "\n"
				}
				client.outgoing <- "\n"
	}
}

func (client *Client) Write() {
	for data := range client.outgoing {
		client.writer.WriteString(data)
		client.writer.Flush()
	}
}

func (client *Client) Listen() {
	go client.Read()
	go client.Write()
}

func NewClient(connection net.Conn) *Client {
	writer := bufio.NewWriter(connection)
	reader := bufio.NewReader(connection)

	client := &Client{
		incoming: make(chan string),
		outgoing: make(chan string),
		reader: reader,
		writer: writer,
	}

	client.Listen()

	return client
}

type ChatRoom struct {
	clients []*Client
	joins chan net.Conn
	incoming chan string
	outgoing chan string
}

func (chatRoom *ChatRoom) Broadcast(data string) {
	for _, client := range chatRoom.clients {
		client.outgoing <- data
	}
}

func (chatRoom *ChatRoom) Join(connection net.Conn) {
	client := NewClient(connection)
	chatRoom.clients = append(chatRoom.clients, client)
	client.outgoing <- "Options: 1.build agent 2.build asset 3.build performance\n"
	go func() { for { chatRoom.incoming <- <- client.incoming } }()
}

func (chatRoom *ChatRoom) Listen() {
	go func() {
		for {
			select {
			case data := <-chatRoom.incoming:
				//chatRoom.Broadcast(data)
				fmt.Println(data)
			case conn := <-chatRoom.joins:
				chatRoom.Join(conn)
				//default : chatRoom.Broadcast("Options: 1.build agent 2.build asset 3.build performance")
			}
		}
	}()
}

func NewChatRoom() *ChatRoom {
	chatRoom := &ChatRoom{
		clients: make([]*Client, 0),
		joins: make(chan net.Conn),
		incoming: make(chan string),
		outgoing: make(chan string),
	}

	chatRoom.Listen()

	return chatRoom
}

func main() {
	r := gin.Default()

	r.GET("/", func(c *gin.Context) {
		http.ServeFile(c.Writer, c.Request, "index.html")
	})
	r.Run(":5000")

	Chatter()
}

func Chatter() {
	chatRoom := NewChatRoom()

	listener, _ := net.Listen("tcp", ":6666")

	for {
		conn, _ := listener.Accept()
		chatRoom.joins <- conn
	}
}
}
