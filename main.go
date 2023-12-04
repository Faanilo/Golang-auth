package main

import (
	"os"

	routes "Golang-auth/routes"

	"github.com/gin-gonic/gin"
)

func main() {
	var port string
	port = os.Getenv("PORT")

	if port == "" {
		port = "5200"
	}
	router := gin.New()
	router.Use(gin.Logger())

	routes.AuthRoutes(router)
	routes.UserRoutes(router)
}
