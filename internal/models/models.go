package models

import "time"

type UserState struct {
	UserID      int64
	CurrentStep string
	TempData    map[string]interface{}
}

type Availability struct {
	Date      time.Time `json:"date"`
	ItemID    int       `json:"item_id"`
	Booked    int       `json:"booked"`
	Available int       `json:"available"`
}
