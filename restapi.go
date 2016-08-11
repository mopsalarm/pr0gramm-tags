package main

import (
	"net/http"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"
)

func restApi(httpListen string, actions *storeActions, checkpointFile string) {
	r := gin.New()
	r.Use(gin.Recovery(), gin.LoggerWithWriter(logrus.StandardLogger().Writer()))

	r.GET("/query/:query", func(c *gin.Context) {
		start := time.Now()
		items, err := actions.Search(c.ParamValue("query"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"items":    items,
			"duration": time.Since(start).String(),
		})
	})

	r.POST("/checkpoint", func(c *gin.Context) {
		start := time.Now()
		actions.WriteCheckpoint(checkpointFile)
		c.JSON(http.StatusOK, gin.H{"duration": time.Since(start).String()})
	})

	r.Run(httpListen)
}
