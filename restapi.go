package main

import (
	"net/http"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"
	"github.com/mopsalarm/go-pr0gramm-tags/tagsapi"
	"strconv"
	"github.com/gin-gonic/contrib/ginrus"
)

func restApi(httpListen string, actions *storeActions, checkpointFile string) {
	r := gin.New()
	gin.SetMode(gin.ReleaseMode)
	r.Use(gin.Recovery())
	r.Use(ginrus.Ginrus(logrus.StandardLogger(), time.RFC3339, true))

	r.GET("/query/:query", func(c *gin.Context) {
		query := c.ParamValue("query")
		shuffle := c.FormValue("shuffle") == "true"

		olderThan := int32(0)
		if olderThanValue := c.FormValue("older"); olderThanValue != "" {
			value, err := strconv.ParseInt(olderThanValue, 10, 32)
			if err == nil {
				olderThan = int32(value)
			}
		}

		start := time.Now()
		items, err := actions.Search(query, olderThan, shuffle)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, tagsapi.SearchResult{
			Duration: time.Since(start).String(),
			Items: items,
		})
	})

	r.POST("/checkpoint", func(c *gin.Context) {
		start := time.Now()
		actions.WriteCheckpoint(checkpointFile)
		c.JSON(http.StatusOK, gin.H{"duration": time.Since(start).String()})
	})

	logrus.Fatal(r.Run(httpListen))
}
