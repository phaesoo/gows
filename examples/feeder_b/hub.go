package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

type Hub struct {
	clients map[*Client]bool

	broadcast chan []byte

	register chan *Client

	unregister chan *Client
}

func newHub() *Hub {
	return &Hub{
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
	}
}

func (h *Hub) watchDB() {
	ticker := time.NewTicker(time.Microsecond)

	db, err := sql.Open("sqlite3", "./foo.db")
	defer db.Close()
	checkErr(err)
	d := OhlcvDaily{}

	latestID := 0

	for {
		select {
		case <-ticker.C:
			q := fmt.Sprintf("SELECT * FROM ohlcv_daily WHERE id > %v", latestID)
			rows, err := db.Query(q)
			checkErr(err)

			for rows.Next() {
				err = rows.Scan(&d.ID, &d.Code, &d.Date, &d.Open, &d.High, &d.Low, &d.Close, &d.Volume, &d.UpdatedAt)
				checkErr(err)

				b, err := json.Marshal(d)
				checkErr(err)

				var temp OhlcvDaily

				err = json.Unmarshal(b, &temp)
				checkErr(err)

				h.broadcast <- b

				latestID = d.ID
			}
			rows.Close()
		}
	}
}

func (h *Hub) run() {
	log.Println("run")
	time.Sleep(time.Second * 2)
	go h.watchDB()

	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
		case message := <-h.broadcast:
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
		}
	}
}
