package models

import "time"

type Task struct {
	ID                         uint        `gorm:"primaryKey" json:"id"`
	Title                      string      `json:"title"`
	Description                string      `json:"description"`
	Status                     string      `json:"status"`
	Deadline                   *time.Time  `json:"deadline"`
	DeadlineStatus             string      `gorm:"default:'on_time'" json:"deadline_status"`
	ProgressPercentage         int         `gorm:"default:0" json:"progress_percentage"`
	CreatedByID                uint        `json:"created_by_id"`
	AssignedToID               uint        `json:"assigned_to_id"`
	ExtensionRequested         bool        `gorm:"default:false" json:"extension_requested"`
	ExtensionRequestedByID     *uint       `json:"extension_requested_by_id"`
	ExtensionRequestedAt       *time.Time  `json:"extension_requested_at"`
	ExtensionRequestedDeadline *time.Time  `json:"extension_requested_deadline"`
	ExtensionReason            string      `json:"extension_reason"`
	ExtensionApprovedByID      *uint       `json:"extension_approved_by_id"`
	ExtensionApprovedAt        *time.Time  `json:"extension_approved_at"`
	CompletedAt                *time.Time  `json:"completed_at"`
	CompletionLocked           bool        `gorm:"default:false" json:"completion_locked"`
	PendingApprovalNotifiedAt  *time.Time  `json:"pending_approval_notified_at"`
	ApprovedByID               *uint       `json:"approved_by_id"`
	ApprovedAt                 *time.Time  `json:"approved_at"`
	RejectedByID               *uint       `json:"rejected_by_id"`
	RejectedAt                 *time.Time  `json:"rejected_at"`
	RejectionReason            string      `json:"rejection_reason"`
	CreatedAt                  time.Time   `json:"created_at"`
	AuditTrail                 []TaskAudit `json:"audit_trail,omitempty"`
}
