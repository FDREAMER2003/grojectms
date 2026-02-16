package main

import (
	"taskmanager/config"
	"taskmanager/models"
	"taskmanager/routes"
)

func main() {
	db := config.ConnectDB()
	db.AutoMigrate(&models.Task{})
	r := routes.SetupRouter(db)
	r.Run(":8000")
}
