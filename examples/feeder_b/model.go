package main

import "time"

type OhlcvDaily struct {
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
