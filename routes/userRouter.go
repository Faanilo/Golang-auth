package routes

import (
	"github.com/Faanilo/Golang-auth/controllers"
	middleware "github.com/Faanilo/Golang-auth/middleware"
	"github.com/gin-gonic/gin"
)

func UserRoutes(router *gin.Engine) {
	router.Use(middleware.Authentificate())
	router.GET("/get-all-users", controllers.GetUsers())
	router.GET("/get-UserById/:userId", controllers.GetUserById())
}
