package routes

import (
	"github.com/Faanilo/Golang-auth/controllers"
	"github.com/gin-gonic/gin"
)

func AuthRoutes(router *gin.Engine) {
	router.POST("/register", controllers.Register())
	router.POST("/login", controllers.Login())
}
