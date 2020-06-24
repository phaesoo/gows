package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	_ "github.com/mattn/go-sqlite3"
)

const (
	writeWait = 10 * time.Second

	pongWait = 60 * time.Second

	pingPeriod = (pongWait * 9) / 10

	maxMessageSize = 512
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type ohlcvDaily struct {
	ID        int       `json:"id"`
	Code      string    `json:"code"`
	Date      int       `json:"date"`
	Open      float64   `json:"open"`
	High      float64   `json:"high"`
	Low       float64   `json:"low"`
	Close     float64   `json:"close"`
	Volume    float64   `json:"volume"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Client struct {
	latestID int

	conn *websocket.Conn

	send chan []byte
	done chan int
}

func (c *Client) watchDB() {
	defer func() {
		c.conn.Close()
	}()

	ticker := time.NewTicker(time.Microsecond)

	db, err := sql.Open("sqlite3", "./foo.db")
	defer db.Close()
	checkErr(err)
	d := ohlcvDaily{}
	latestID := 0

	for {
		select {
		case <-ticker.C:
			q := fmt.Sprintf("SELECT * FROM ohlcv_daily WHERE id > %v", c.latestID)
			rows, err := db.Query(q)
			checkErr(err)

			for rows.Next() {
				err = rows.Scan(&d.ID, &d.Code, &d.Date, &d.Open, &d.High, &d.Low, &d.Close, &d.Volume, &d.UpdatedAt)
				checkErr(err)

				b, err := json.Marshal(d)
				checkErr(err)

				var temp ohlcvDaily

				err = json.Unmarshal(b, &temp)
				checkErr(err)

				log.Println("start to send", len(c.send), cap(c.send), len(b))
				c.send <- b
				log.Println("end to send")

				latestID = d.ID
			}
			rows.Close()

			c.latestID = latestID
		case <-c.done:
			log.Println("Done!!!!!!!!!!!!!!!!!!!!!!!")
			return
		}
	}
}

func (c *Client) sendMessage() {
	defer func() {
		c.conn.Close()
	}()

	d := ohlcvDaily{}

	for {
		select {
		case message, _ := <-c.send:
			log.Println("get message")
			err := json.Unmarshal(message, &d)
			checkErr(err)
			if err := c.conn.WriteJSON(d); err != nil {
				log.Println("send finished")
				c.done <- 1
				close(c.send)
				return
			}
			time.Sleep(time.Second * 1)
			log.Println("write message")
		}
	}
}

func serveWs(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	client := &Client{latestID: 0, conn: conn, send: make(chan []byte, 1024), done: make(chan int, 1)}

	go client.watchDB()
	time.Sleep(time.Second * 3)

	go client.sendMessage()
}
