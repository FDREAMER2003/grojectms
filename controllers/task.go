package controllers

import (
	"net/http"
	"taskmanager/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type TaskController struct {
	DB *gorm.DB
}

func (tc *TaskController) CreateTask(c *gin.Context) {
	var task models.Task

	if err := c.BindJSON(&task); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tc.DB.Create(&task)

	c.JSON(http.StatusOK, task)
}

func (tc *TaskController) GetTasks(c *gin.Context) {
	var tasks []models.Task

	tc.DB.Find(&tasks)

	c.JSON(http.StatusOK, tasks)
}

func (tc *TaskController) GetTask(c *gin.Context) {
	id := c.Param("id")

	var task models.Task

	if err := tc.DB.First(&task, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		return
	}

	c.JSON(http.StatusOK, task)
}

func (tc *TaskController) UpdateTask(c *gin.Context) {
	id := c.Param("id")

	var task models.Task

	if err := tc.DB.First(&task, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		return
	}

	c.BindJSON(&task)

	tc.DB.Save(&task)

	c.JSON(http.StatusOK, task)
}

func (tc *TaskController) DeleteTask(c *gin.Context) {
	id := c.Param("id")

	tc.DB.Delete(&models.Task{}, id)

	c.JSON(http.StatusOK, gin.H{"message": "Deleted"})
}
