package main

import (
	"net/http"	
	"github.com/gin-gonic/gin"
)
func main() {
  router := gin.Default()
  router.GET("/", func(c *gin.Context) {
    c.JSON(http.StatusOK, gin.H{
      "message": "Hello from Docker ! damn the request actually went through the tunnel.",
    })
  })
  router.Run(":8080") 
}