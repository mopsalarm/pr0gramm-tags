package main

import (
	"net/http"
	"time"

	"bytes"
	"github.com/Sirupsen/logrus"
	"github.com/gin-gonic/contrib/ginrus"
	"github.com/gin-gonic/gin"
	"github.com/mopsalarm/go-pr0gramm-tags/parser"
	"github.com/mopsalarm/go-pr0gramm-tags/tagsapi"
	"strconv"
)

func restApi(httpListen string, actions *storeActions, checkpointFile string) {
	searchHandler := func(c *gin.Context) {
		query := c.ParamValue("query")
		random := c.FormValue("random") == "true"

		olderThan := int32(0)
		if olderThanValue := c.FormValue("older"); olderThanValue != "" {
			value, err := strconv.ParseInt(olderThanValue, 10, 32)
			if err == nil {
				olderThan = int32(value)
			}
		}

		start := time.Now()
		items, err := actions.Search(query, olderThan, random)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, tagsapi.SearchResult{
			Duration: time.Since(start).String(),
			Items:    items,
		})
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(ginrus.Ginrus(logrus.StandardLogger(), time.RFC3339, true))

	r.GET("/query/", searchHandler)
	r.GET("/query/:query", searchHandler)

	r.POST("/admin/write-checkpoint", func(c *gin.Context) {
		start := time.Now()
		actions.WriteCheckpoint(checkpointFile)
		c.JSON(http.StatusOK, gin.H{"duration": time.Since(start).String()})
	})

	r.POST("/admin/rebuild-items", func(c *gin.Context) {
		actions.WithWriteLock(func() {
			actions.storeState.LastItemId = 0
		})
	})

	r.POST("/admin/rebuild-tags", func(c *gin.Context) {
		actions.WithWriteLock(func() {
			actions.storeState.LastTagId = 0
		})
	})

	r.GET("/admin/parse/:query", func(c *gin.Context) {
		p := parser.NewParser(bytes.NewBufferString(c.ParamValue("query")))

		tree, err := p.Parse()
		if err != nil {
			c.AbortWithError(http.StatusBadRequest, err)
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"parsed":    tree,
			"optimized": parser.Optimize(tree),
		})
	})

	r.POST("/admin/config", func(c *gin.Context) {
		if value := c.FormValue("optimize"); value != "" {
			actions.UseOptimizer = value == "true"
		}
	})

	r.DELETE("/admin/tag/:word", func(c *gin.Context) {
		words := ExtractWords(c.ParamValue("word"))
		actions.WithWriteLock(func() {
			for _, word := range words {
				hash := HashWord(word)
				actions.store.Replace(hash, []int32{})
			}
		})
	})

	logrus.Fatal(r.Run(httpListen))
}
