package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	_ "github.com/mattn/go-sqlite3"
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type Client struct {
	hub *Hub

	conn *websocket.Conn

	send chan []byte
}

func (c *Client) sendMessage() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	d := OhlcvDaily{}

	for {
		select {
		case message, _ := <-c.send:
			log.Println("get message")
			err := json.Unmarshal(message, &d)
			checkErr(err)
			if err := c.conn.WriteJSON(d); err != nil {
				log.Println("send finished")
				close(c.send)
				return
			}
			time.Sleep(time.Second * 1)
			log.Println("write message")
		}
	}
}

func serveWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	client := &Client{hub: hub, conn: conn, send: make(chan []byte, 1024)}
	client.hub.register <- client

	go client.sendMessage()
}
