# statistics
Simple server-side analytics for Gin Gonic

# Usage

On each route: gin.Context.Set("PageType", statistics.Dynamic || statistics.Static)
If not set, the middleware will determine the content type (Dynamic page or static file) via the Content-Type header (if it's equal to "text/html")

## Example

```golang
	// no need to set the PageType manually
	r.Static("/static", "./files")

	r.GET("/static-route", func(c *gin.Context) {
		c.Set("PageType", statistics.Static) // optional because the route doesn't return html
		c.JSON(http.StatusOK, gin.H{
			"ok": "found",
		})
	})

	r.GET("/html", func(c *gin.Context) {
		c.Set("PageType", statistics.Dynamic) // optional because the route returns html
		c.HTML(http.StatusOK, "index.html", gin.H{})
	})

	r.GET("/special-route", func(c *gin.Context) {
		c.Set("PageType", statistics.Dynamic)
		c.JSON(http.StatusOK, gin.H{
			"ok": "found",
		})
	})
```