package models

import "time"

type Booking struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	UserName  string    `json:"user_name"`
	Phone     string    `json:"phone"`
	ItemID    int       `json:"item_id"`
	ItemName  string    `json:"item_name"`
	Date      time.Time `json:"date"`
	Status    string    `json:"status"` // pending, confirmed, cancelled
	CreatedAt time.Time `json:"created_at"`
}
