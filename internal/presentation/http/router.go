package http

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// NewRouter assembles the Gin engine: middleware, health check, and one
// route group per bounded context under /api/v1.
func NewRouter(
	logger *slog.Logger,
	beers *BeerHandlers,
	stock *StockHandlers,
	orders *OrderHandlers,
	batches *BatchHandlers,
) *gin.Engine {
	engine := gin.New()
	engine.Use(gin.Recovery(), requestLogger(logger))

	engine.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	v1 := engine.Group("/api/v1")

	beerRoutes := v1.Group("/beers")
	{
		beerRoutes.POST("", beers.Create)
		beerRoutes.GET("", beers.List)
		beerRoutes.GET("/:id", beers.Get)
		beerRoutes.PUT("/:id/price", beers.ChangePrice)
		beerRoutes.DELETE("/:id", beers.Retire)
	}

	stockRoutes := v1.Group("/stock")
	{
		stockRoutes.POST("", stock.Track)
		stockRoutes.GET("", stock.List)
		stockRoutes.GET("/:beerId", stock.Get)
		stockRoutes.POST("/:beerId/replenish", stock.Replenish)
	}

	orderRoutes := v1.Group("/orders")
	{
		orderRoutes.POST("", orders.Place)
		orderRoutes.GET("", orders.List)
		orderRoutes.GET("/:id", orders.Get)
		orderRoutes.POST("/:id/cancel", orders.Cancel)
	}

	batchRoutes := v1.Group("/batches")
	{
		batchRoutes.POST("", batches.Start)
		batchRoutes.GET("", batches.List)
		batchRoutes.GET("/:id", batches.Get)
		batchRoutes.POST("/:id/complete", batches.Complete)
	}

	return engine
}

func requestLogger(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		logger.Info("http.request",
			slog.String("method", c.Request.Method),
			slog.String("path", c.Request.URL.Path),
			slog.Int("status", c.Writer.Status()),
			slog.Duration("duration", time.Since(start)),
		)
	}
}
