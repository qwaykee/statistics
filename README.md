# statistics
Simple server-side analytics for Gin Gonic

# Usage

On each route: gin.Context.Set("PageType", statistics.Dynamic || statistics.Static)
If not set, the middleware will determine the content type (Dynamic page or static file) via the Content-Type header (if it's equal to "text/html")