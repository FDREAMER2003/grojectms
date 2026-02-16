package controllers

import (
	"net/http"
	"taskmanager/constants"
	"taskmanager/models"
	"taskmanager/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type TaskController struct {
	DB *gorm.DB
}

func (tc *TaskController) CreateTask(c *gin.Context) {
	userID := uint(c.GetFloat64("user_id"))
	role := c.GetString("role")

	var task models.Task

	if err := c.BindJSON(&task); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	canAssign, err := utils.CanAssignTask(userID, role, task.AssignedToID, tc.DB)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify assignment permissions"})
		return
	}

	if !canAssign {
		c.JSON(http.StatusForbidden, gin.H{"error": "You do not have permission to assign a task to this user"})
		return
	}

	task.CreatedByID = userID

	tc.DB.Create(&task)

	c.JSON(http.StatusOK, task)
}

func (tc *TaskController) GetTasks(c *gin.Context) {
	userID := uint(c.GetFloat64("user_id"))
	role := c.GetString("role")

	var tasks []models.Task

	switch role {

	case constants.RoleAdmin:
		tc.DB.Find(&tasks)

	case constants.RoleManager:

		// Get members under this manager
		var members []models.User
		tc.DB.Where("manager_id = ?", userID).
			Find(&members)

		var memberIDs []uint
		for _, m := range members {
			memberIDs = append(memberIDs, m.ID)
		}

		// Include manager's own tasks as well if needed,
		// but usually manager sees tasks assigned to their members
		tc.DB.Where(
			"assigned_to_id IN ? OR created_by_id = ?",
			memberIDs, userID,
		).Find(&tasks)

	case constants.RoleMember:

		tc.DB.Where(
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

	if err := tc.DB.First(&task, id).Error; err != nil {
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

	c.BindJSON(&task)

	canAssign, err := utils.CanAssignTask(userID, role, task.AssignedToID, tc.DB)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify assignment permissions"})
		return
	}

	if !canAssign {
		c.JSON(http.StatusForbidden, gin.H{"error": "You do not have permission to assign a task to this user"})
		return
	}

	tc.DB.Save(&task)

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
