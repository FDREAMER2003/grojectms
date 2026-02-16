package controllers

import (
	"net/http"
	"taskmanager/models"
	"taskmanager/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AuthController struct {
	DB *gorm.DB
}

func (ac *AuthController) Register(c *gin.Context) {
	var user models.User

	if err := c.BindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	hashed, _ := utils.HashPassword(user.Password)
	user.Password = hashed

	ac.DB.Create(&user)

	c.JSON(http.StatusOK, gin.H{
		"message": "User registered",
	})
}

func (ac *AuthController) Login(c *gin.Context) {
	var input models.User
	var user models.User

	if err := c.BindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := ac.DB.
		Where("email = ?", input.Email).
		First(&user).Error; err != nil {

		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	if !utils.CheckPassword(input.Password, user.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	token, _ := utils.GenerateJWT(user)

	c.JSON(http.StatusOK, gin.H{
		"token": token,
	})
}
