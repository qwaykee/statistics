package main

import (
	"github.com/gin-gonic/gin"
	"github.com/qwaykee/statistics"

	"net/http"
	"time"
	"log"
)

func main() {
	r := gin.Default()

	r.LoadHTMLGlob("*.html")

	st := statistics.New()

	r.Use(st.Middleware())

	r.GET("/", func(c *gin.Context) {
		// c.Set("PageType", statistics.Static)
		c.JSON(http.StatusOK, gin.H{
			"ok": "found",
		})
	})

	r.GET("/html", func(c *gin.Context) {
		// c.Set("PageType", statistics.Dynamic)
		c.HTML(http.StatusOK, "index.html", gin.H{})
	})

	r.GET("/stats", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"VisitsCount": st.VisitsCount(),
			"VisitorsCount": st.VisitorsCount(),
			"EstimatedCurrentVisitors": st.EstimatedCurrentVisitors(),
			"LanguagesCount": st.LanguagesCount(),
			"Time": time.Now().Format(time.RFC3339),
		})
	})

	r.GET("/get-visit/:date", func(c *gin.Context) {
		date, err := time.Parse(time.RFC3339, c.Param("date"))
		if err != nil {
			log.Println(err)
			return
		}

		visit, err := st.GetVisitor(c.ClientIP()).GetVisit(date)
		if err != nil {
			log.Println(err)
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"Visit": visit,
		})
	})

	r.Run()
}