package models

import "time"

type Task struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Title        string    `json:"title"`
	Description  string    `json:"description"`
	Status       string    `json:"status"`
	CreatedByID  uint      `json:"created_by_id"`
	AssignedToID uint      `json:"assigned_to_id"`
	CreatedAt    time.Time `json:"created_at"`
}
