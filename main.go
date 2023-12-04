package main

import (
	"os"

	routes "github.com/Faanilo/Golang-auth/routes"
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
	router.GET("/api-v1", func(c *gin.Context) {
		c.JSON(200, gin.H{"success": "Hello world 1"})
	})
	router.GET("api-v2", func(c *gin.Context) {
		c.JSON(200, gin.H{"success": "Hello World 2"})
	})

	router.Run(":" + port)
}
