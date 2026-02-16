package routes

import (
	"taskmanager/controllers"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func SetupRouter(db *gorm.DB) *gin.Engine {
	r := gin.Default()

	taskController := controllers.TaskController{DB: db}

	r.POST("/tasks", taskController.CreateTask)
	r.GET("/tasks", taskController.GetTasks)
	r.GET("/tasks/:id", taskController.GetTask)
	r.PUT("/tasks/:id", taskController.UpdateTask)
	r.DELETE("/tasks/:id", taskController.DeleteTask)

	return r
}
