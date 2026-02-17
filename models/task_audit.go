package models

import "time"

type TaskAudit struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	TaskID    uint      `json:"task_id"`
	Action    string    `json:"action"`
	ActorID   uint      `json:"actor_id"`
	Comments  string    `json:"comments"`
	CreatedAt time.Time `json:"created_at"`
}
