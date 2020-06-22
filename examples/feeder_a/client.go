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
}

func (c *Client) watchDB() {
	defer func() {
		c.conn.Close()
	}()

	ticker := time.NewTicker(2 * time.Second)

	db, err := sql.Open("sqlite3", "./foo.db")
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

				//log.Printf("%t", rows)
				//[]byte(fmt.Sprintf("%v", d))

				log.Println(d)

				b, err := json.Marshal(d)
				checkErr(err)

				log.Println("watch")

				var temp ohlcvDaily

				err = json.Unmarshal(b, &temp)
				checkErr(err)
				fmt.Printf("%+v\n", temp)

				c.send <- b

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

	d := ohlcvDaily{}

	for {
		select {
		case message, _ := <-c.send:
			err := json.Unmarshal(message, &d)
			checkErr(err)
			c.conn.WriteJSON(d)
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
	client.sendMessage()
}
