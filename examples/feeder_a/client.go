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
	code      string    `json:code`
	date      int       `json:date`
	open      float64   `json:open`
	high      float64   `json:high`
	low       float64   `json:low`
	close     float64   `json:close`
	volume    float64   `json:volume`
	updatedAt time.Time `json:updated_at`
}

type Client struct {
	latestID int

	conn *websocket.Conn

	send chan []byte
}

func (c *Client) watchDB() {
	defer func() {
		c.conn.Close()
	}()

	ticker := time.NewTicker(2 * time.Second)

	db, err := sql.Open("sqlite3", "./foo.db")
	checkErr(err)
	d := ohlcvDaily{}

	for {
		select {
		case <-ticker.C:
			q := fmt.Sprintf("SELECT * FROM ohlcv_daily WHERE id > %v", c.latestID)
			rows, err := db.Query(q)
			checkErr(err)

			latestID := 0
			for rows.Next() {
				err = rows.Scan(&d.ID, &d.code, &d.date, &d.open, &d.high, &d.low, &d.close, &d.volume, &d.updatedAt)
				checkErr(err)

				//log.Printf("%t", rows)
				//[]byte(fmt.Sprintf("%v", d))

				b, err := json.Marshal(d)

				log.Println("watch")
				log.Println(b)

				checkErr(err)

				c.send <- b

				log.Println(c.send)

				latestID = d.ID
			}

			c.latestID = latestID
		}
	}
}

func (c *Client) sendMessage() {
	defer func() {
		c.conn.Close()
	}()

	for {
		select {
		case message, _ := <-c.send:
			var d ohlcvDaily
			err := json.Unmarshal(message, &d)
			checkErr(err)
			log.Println("send")
			fmt.Printf("%+v\n", d)
		}
	}
}

func serveWs(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	client := &Client{latestID: 0, conn: conn, send: make(chan []byte, 1024)}

	go client.watchDB()
	go client.sendMessage()
}
