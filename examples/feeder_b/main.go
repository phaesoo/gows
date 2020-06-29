package main

import (
	"database/sql"
	"flag"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var addr = flag.String("addr", ":5000", "http service address")

func serveHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	http.ServeFile(w, r, "home.html")
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}

func feedData() {
	os.Remove("./foo.db")

	db, err := sql.Open("sqlite3", "./foo.db")
	checkErr(err)

	stmp, err := db.Prepare(`CREATE TABLE ohlcv_daily(
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		code TEXT,
		date INTEGER,
		open REAL,
		high REAL,
		low REAL,
		close REAL,
		volume REAL,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);`)
	checkErr(err)

	_, err = stmp.Exec()
	checkErr(err)

	ticker := time.NewTicker(time.Second)
	defer func() {
		ticker.Stop()
		db.Close()
	}()

	for {
		select {
		case <-ticker.C:
			stmp, err := db.Prepare(`INSERT INTO ohlcv_daily(code, date, open, high, low, close, volume)
			VALUES(?, ?, ?, ?, ?, ?, ?);`)
			checkErr(err)

			res, err := stmp.Exec("SPY", 20200619, 100.*rand.Float64(), 100.*rand.Float64(), 100.*rand.Float64(), 100.*rand.Float64(), 100.*rand.Float64())
			checkErr(err)

			affect, err := res.RowsAffected()
			checkErr(err)

			stmp.Close()
			log.Println(affect)
		}
	}
}

func main() {
	flag.Parse()

	// start to feed data
	go feedData()

	hub := newHub()
	go hub.run()

	http.HandleFunc("/", serveHome)
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(hub, w, r)
	})
	err := http.ListenAndServe(*addr, nil)
	if err != nil {
		log.Fatal("ListenAndServer: ", err)
	}
}
