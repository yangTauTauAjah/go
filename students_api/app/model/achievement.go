package model

import "time"

type Achievement struct {
	ID        int       `json:"id"`
	StudentID int       `json:"student_id"`
	Name      string    `json:"name"`
	Score     float64   `json:"score"`
	CreatedAt time.Time `json:"created_at"`
}
