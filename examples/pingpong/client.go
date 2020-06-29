package main

import (
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

type client struct {
	conn *websocket.Conn
}

const (
	writeWait = 5 * time.Second

	pongWait = 10 * time.Second

	// pingPeriod should have to less than pongWait
	pingPeriod = (pongWait * 9) / 10

	maxMessageSize = 512
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func (c *client) reader() {
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))

	// register pong handler
	c.conn.SetPongHandler(func(s string) error {
		log.Println("PongHandler: ", s)

		// update read deadline again
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}

		log.Println("Message: ", message)
	}
}

func (c *client) writer() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))

			// send ping message
			log.Println("send ping")
			err := c.conn.WriteMessage(websocket.PingMessage, []byte("ping~"))
			checkErr(err)
		}
	}
}

func serveWs(w http.ResponseWriter, r *http.Request) {
	log.Println("Start client")

	// upgrade connection
	conn, err := upgrader.Upgrade(w, r, nil)
	checkErr(err)

	c := client{conn: conn}
	go c.writer()
	go c.reader()
}
