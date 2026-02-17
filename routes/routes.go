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
		taskRoutes.PUT("/:id", taskController.UpdateTask)
		taskRoutes.POST("/:id/approve", middleware.RoleMiddleware(constants.RoleAdmin, constants.RoleManager), taskController.ApproveTask)
		taskRoutes.POST("/:id/reject", middleware.RoleMiddleware(constants.RoleAdmin, constants.RoleManager), taskController.RejectTask)
		taskRoutes.DELETE("/:id", middleware.RoleMiddleware(constants.RoleAdmin), taskController.DeleteTask)
	}

	authController := controllers.AuthController{DB: db}
	r.POST("/register", authController.Register)
	r.POST("/login", authController.Login)

	userController := controllers.UserController{DB: db}
	userRoutes := r.Group("/users")
	userRoutes.Use(middleware.AuthMiddleware(), middleware.RoleMiddleware(constants.RoleAdmin))
	{
		userRoutes.GET("", userController.GetUsers)
		userRoutes.PUT("/:id", userController.UpdateUser)
	}

	return r
}
