package routes

import (
	"taskmanager/constants"
	"taskmanager/controllers"

	"taskmanager/middleware"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func SetupRouter(db *gorm.DB) *gin.Engine {
	r := gin.Default()

	taskController := controllers.TaskController{DB: db}
	taskRoutes := r.Group("/tasks")
	taskRoutes.Use(middleware.AuthMiddleware())
	{
		taskRoutes.POST("", middleware.RoleMiddleware(constants.RoleAdmin, constants.RoleManager), taskController.CreateTask)
		taskRoutes.GET("", taskController.GetTasks)
		taskRoutes.GET("/:id", taskController.GetTask)
		taskRoutes.PUT("/:id", middleware.RoleMiddleware(constants.RoleAdmin, constants.RoleManager), taskController.UpdateTask)
		taskRoutes.DELETE("/:id", middleware.RoleMiddleware(constants.RoleAdmin), taskController.DeleteTask)
	}

	authController := controllers.AuthController{DB: db}
	r.POST("/register", authController.Register)
	r.POST("/login", authController.Login)

	return r
}
