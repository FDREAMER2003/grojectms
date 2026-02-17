package main

import (
	"os"
	"taskmanager/config"
	"taskmanager/models"
	"taskmanager/routes"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	db := config.ConnectDB()
	db.AutoMigrate(
		&models.Task{},
		&models.TaskAudit{},
		&models.User{},
	)
	r := routes.SetupRouter(db)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}
	r.Run(":" + port)
}
