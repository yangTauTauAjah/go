package model

import "time"

type Achievement struct {
	ID        int       `json:"id"`
	StudentID string    `json:"user_id"`
	Name      string    `json:"name"`
	Score     string    `json:"score"`
	CreatedAt time.Time `json:"created_at"`
}
