package controllers

import (
	"net/http"
	"taskmanager/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type UserController struct {
	DB *gorm.DB
}

func (uc *UserController) GetUsers(c *gin.Context) {
	var users []models.User
	uc.DB.Find(&users)
	c.JSON(http.StatusOK, users)
}

func (uc *UserController) UpdateUser(c *gin.Context) {
	id := c.Param("id")
	var user models.User

	if err := uc.DB.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	var input struct {
		Role      string `json:"role"`
		ManagerID *uint  `json:"manager_id"`
	}

	if err := c.BindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Simple validation to prevent a user from being their own manager
	if input.ManagerID != nil && *input.ManagerID == user.ID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User cannot be their own manager"})
		return
	}

	if input.Role != "" {
		user.Role = input.Role
	}
	user.ManagerID = input.ManagerID

	uc.DB.Save(&user)

	c.JSON(http.StatusOK, user)
}
