package controllers

import (
	"errors"
	"io"
	"net/http"
	"taskmanager/constants"
	"taskmanager/models"
	"taskmanager/utils"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type TaskController struct {
	DB *gorm.DB
}

type updateTaskInput struct {
	Title              *string `json:"title"`
	Description        *string `json:"description"`
	AssignedToID       *uint   `json:"assigned_to_id"`
	Status             *string `json:"status"`
	ProgressPercentage *int    `json:"progress_percentage"`
}

type taskDecisionInput struct {
	Comments string `json:"comments"`
	Reason   string `json:"reason"`
}

func (tc *TaskController) CreateTask(c *gin.Context) {
	userID := uint(c.GetFloat64("user_id"))
	role := c.GetString("role")

	var task models.Task
	if err := c.BindJSON(&task); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if task.ProgressPercentage < 0 || task.ProgressPercentage > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "progress_percentage must be between 0 and 100"})
		return
	}

	if task.AssignedToID != 0 {
		canAssign, err := utils.CanAssignTask(userID, role, task.AssignedToID, tc.DB)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify assignment permissions"})
			return
		}
		if !canAssign {
			c.JSON(http.StatusForbidden, gin.H{"error": "You do not have permission to assign a task to this user"})
			return
		}
	}

	task.CreatedByID = userID

	if task.AssignedToID == 0 {
		task.Status = constants.TaskStatusCreated
	} else {
		task.Status = constants.TaskStatusAssigned
	}

	if err := tc.DB.Create(&task).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create task"})
		return
	}

	c.JSON(http.StatusOK, task)
}

func (tc *TaskController) GetTasks(c *gin.Context) {
	userID := uint(c.GetFloat64("user_id"))
	role := c.GetString("role")

	var tasks []models.Task
	query := tc.DB.Preload("AuditTrail")

	switch role {
	case constants.RoleAdmin:
		query.Find(&tasks)

	case constants.RoleManager:
		memberIDs := utils.GetRecursiveReportIDs(userID, tc.DB)
		query.Where(
			"assigned_to_id IN ? OR created_by_id = ? OR assigned_to_id = ?",
			memberIDs, userID, userID,
		).Find(&tasks)

	case constants.RoleMember:
		query.Where(
			"created_by_id = ? OR assigned_to_id = ?",
			userID, userID,
		).Find(&tasks)

	default:
		c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized role"})
		return
	}

	c.JSON(http.StatusOK, tasks)
}

func (tc *TaskController) GetTask(c *gin.Context) {
	id := c.Param("id")
	userID := uint(c.GetFloat64("user_id"))
	role := c.GetString("role")

	var task models.Task
	if err := tc.DB.Preload("AuditTrail").First(&task, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		return
	}

	if !utils.CanAccessTask(task, userID, role, tc.DB) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized access"})
		return
	}

	c.JSON(http.StatusOK, task)
}

func (tc *TaskController) UpdateTask(c *gin.Context) {
	id := c.Param("id")
	userID := uint(c.GetFloat64("user_id"))
	role := c.GetString("role")

	var task models.Task
	if err := tc.DB.First(&task, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		return
	}

	if !utils.CanAccessTask(task, userID, role, tc.DB) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized access"})
		return
	}

	var input updateTaskInput
	if err := c.BindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if role == constants.RoleMember {
		if task.AssignedToID != userID {
			c.JSON(http.StatusForbidden, gin.H{"error": "Members can only update tasks assigned to themselves"})
			return
		}
		if input.Title != nil || input.Description != nil || input.AssignedToID != nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "Members can only update progress and status on their own tasks"})
			return
		}
	}

	if input.AssignedToID != nil {
		canAssign, err := utils.CanAssignTask(userID, role, *input.AssignedToID, tc.DB)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify assignment permissions"})
			return
		}
		if !canAssign {
			c.JSON(http.StatusForbidden, gin.H{"error": "You do not have permission to assign a task to this user"})
			return
		}
	}

	if input.ProgressPercentage != nil {
		if *input.ProgressPercentage < 0 || *input.ProgressPercentage > 100 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "progress_percentage must be between 0 and 100"})
			return
		}
		task.ProgressPercentage = *input.ProgressPercentage
	}

	if input.Title != nil {
		task.Title = *input.Title
	}
	if input.Description != nil {
		task.Description = *input.Description
	}
	if input.AssignedToID != nil {
		task.AssignedToID = *input.AssignedToID
		if task.Status == constants.TaskStatusCreated && task.AssignedToID != 0 {
			task.Status = constants.TaskStatusAssigned
		}
	}

	if input.Status != nil {
		mappedStatus := normalizeStatus(*input.Status)
		if mappedStatus == constants.TaskStatusApproved || mappedStatus == constants.TaskStatusRejected {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Use /tasks/:id/approve or /tasks/:id/reject for approval decisions"})
			return
		}
		if !isValidStatus(mappedStatus) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task status"})
			return
		}
		if !isAllowedStatusTransition(task.Status, mappedStatus) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid status transition"})
			return
		}
		if mappedStatus == constants.TaskStatusPendingApproval && task.ProgressPercentage < 100 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "progress_percentage must be 100 before moving to pending_approval"})
			return
		}
		if task.Status == constants.TaskStatusApproved && mappedStatus != constants.TaskStatusApproved {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Approved tasks are locked"})
			return
		}

		now := time.Now()
		if mappedStatus == constants.TaskStatusPendingApproval {
			if task.CompletedAt == nil {
				task.CompletedAt = &now
			}
			task.PendingApprovalNotifiedAt = &now
		}
		if mappedStatus == constants.TaskStatusInProgress && task.Status == constants.TaskStatusRejected {
			task.RejectionReason = ""
			task.RejectedByID = nil
			task.RejectedAt = nil
		}
		task.Status = mappedStatus
	}

	if task.CompletionLocked {
		if input.Status != nil && *input.Status != constants.TaskStatusApproved {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Completion date is locked for approved tasks"})
			return
		}
	}

	if err := tc.DB.Save(&task).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update task"})
		return
	}

	c.JSON(http.StatusOK, task)
}

func (tc *TaskController) ApproveTask(c *gin.Context) {
	id := c.Param("id")
	userID := uint(c.GetFloat64("user_id"))
	role := c.GetString("role")

	if role != constants.RoleManager && role != constants.RoleAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only manager/admin can approve tasks"})
		return
	}

	var task models.Task
	if err := tc.DB.First(&task, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		return
	}

	if !utils.CanAccessTask(task, userID, role, tc.DB) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized access"})
		return
	}

	if task.Status != constants.TaskStatusPendingApproval {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Only pending_approval tasks can be approved"})
		return
	}

	var input taskDecisionInput
	if err := c.BindJSON(&input); err != nil && !errors.Is(err, io.EOF) {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	now := time.Now()
	task.Status = constants.TaskStatusApproved
	task.ApprovedByID = &userID
	task.ApprovedAt = &now
	task.CompletionLocked = true
	if task.CompletedAt == nil {
		task.CompletedAt = &now
	}

	if err := tc.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&task).Error; err != nil {
			return err
		}

		audit := models.TaskAudit{
			TaskID:   task.ID,
			Action:   constants.TaskStatusApproved,
			ActorID:  userID,
			Comments: input.Comments,
		}
		return tx.Create(&audit).Error
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to approve task"})
		return
	}

	c.JSON(http.StatusOK, task)
}

func (tc *TaskController) RejectTask(c *gin.Context) {
	id := c.Param("id")
	userID := uint(c.GetFloat64("user_id"))
	role := c.GetString("role")

	if role != constants.RoleManager && role != constants.RoleAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only manager/admin can reject tasks"})
		return
	}

	var task models.Task
	if err := tc.DB.First(&task, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		return
	}

	if !utils.CanAccessTask(task, userID, role, tc.DB) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized access"})
		return
	}

	if task.Status != constants.TaskStatusPendingApproval {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Only pending_approval tasks can be rejected"})
		return
	}

	var input taskDecisionInput
	if err := c.BindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if input.Reason == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "rejection reason is required"})
		return
	}

	now := time.Now()
	task.Status = constants.TaskStatusRejected
	task.RejectedByID = &userID
	task.RejectedAt = &now
	task.RejectionReason = input.Reason
	task.CompletionLocked = false
	task.ApprovedByID = nil
	task.ApprovedAt = nil

	if err := tc.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&task).Error; err != nil {
			return err
		}

		audit := models.TaskAudit{
			TaskID:   task.ID,
			Action:   constants.TaskStatusRejected,
			ActorID:  userID,
			Comments: input.Comments,
		}
		if input.Comments == "" {
			audit.Comments = input.Reason
		}
		return tx.Create(&audit).Error
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reject task"})
		return
	}

	c.JSON(http.StatusOK, task)
}

func (tc *TaskController) DeleteTask(c *gin.Context) {
	id := c.Param("id")
	userID := uint(c.GetFloat64("user_id"))
	role := c.GetString("role")

	var task models.Task
	if err := tc.DB.First(&task, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		return
	}

	if !utils.CanAccessTask(task, userID, role, tc.DB) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized access"})
		return
	}

	tc.DB.Delete(&models.Task{}, id)
	c.JSON(http.StatusOK, gin.H{"message": "Deleted"})
}

func isValidStatus(status string) bool {
	switch status {
	case constants.TaskStatusCreated,
		constants.TaskStatusAssigned,
		constants.TaskStatusInProgress,
		constants.TaskStatusPendingApproval,
		constants.TaskStatusApproved,
		constants.TaskStatusRejected:
		return true
	default:
		return false
	}
}

func normalizeStatus(status string) string {
	if status == "completed" {
		return constants.TaskStatusPendingApproval
	}
	return status
}

func isAllowedStatusTransition(from, to string) bool {
	if from == to {
		return true
	}

	switch from {
	case constants.TaskStatusCreated:
		return to == constants.TaskStatusAssigned
	case constants.TaskStatusAssigned:
		return to == constants.TaskStatusInProgress
	case constants.TaskStatusInProgress:
		return to == constants.TaskStatusPendingApproval
	case constants.TaskStatusPendingApproval:
		return to == constants.TaskStatusApproved || to == constants.TaskStatusRejected
	case constants.TaskStatusRejected:
		return to == constants.TaskStatusInProgress
	case constants.TaskStatusApproved:
		return false
	default:
		return false
	}
}
