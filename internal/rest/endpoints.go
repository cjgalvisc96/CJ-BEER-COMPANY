// Package rest is the BrewUp.Rest equivalent: the HTTP entry points of
// the monolith. By design (and enforced by the architecture fitness test
// in tests/), it depends ONLY on each module's facade — never on a
// module's domain, shared kernel, or read-model internals.
package rest

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/cjgalvisc96/cj-beer-company/internal/muflone"
	"github.com/cjgalvisc96/cj-beer-company/internal/sales"
	"github.com/cjgalvisc96/cj-beer-company/internal/warehouses"
)

// NewRouter maps the endpoints: /v1/sales and /v1/warehouses, mirroring
// the book's MapSalesEndpoints / MapWarehousesEndpoints.
func NewRouter(logger *slog.Logger, salesFacade *sales.Facade, warehousesFacade *warehouses.Facade) *gin.Engine {
	engine := gin.New()
	engine.Use(gin.Recovery(), requestLogger(logger))

	engine.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	v1 := engine.Group("/v1")

	salesRoutes := v1.Group("/sales")
	{
		salesRoutes.POST("", func(c *gin.Context) {
			var body sales.SalesOrderJson
			if err := c.ShouldBindJSON(&body); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			orderId, err := salesFacade.CreateSalesOrder(c.Request.Context(), body)
			if err != nil {
				respondError(c, err)
				return
			}
			// The book returns Created with the pre-generated id; the
			// projection appears in the read model moments later
			// (eventual consistency).
			c.Header("Location", "/v1/sales/"+orderId)
			c.JSON(http.StatusCreated, gin.H{"id": orderId})
		})
		salesRoutes.GET("", func(c *gin.Context) {
			c.JSON(http.StatusOK, salesFacade.GetSalesOrders(c.Request.Context()))
		})
		salesRoutes.GET("/:id", func(c *gin.Context) {
			order, found := salesFacade.GetSalesOrder(c.Request.Context(), c.Param("id"))
			if !found {
				c.JSON(http.StatusNotFound, gin.H{"error": "sales order not found"})
				return
			}
			c.JSON(http.StatusOK, order)
		})
	}

	warehousesRoutes := v1.Group("/warehouses")
	{
		warehousesRoutes.POST("/availability", func(c *gin.Context) {
			var body warehouses.ProductionOrderJson
			if err := c.ShouldBindJSON(&body); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			beerId, err := warehousesFacade.UpdateAvailabilityDueToProductionOrder(c.Request.Context(), body)
			if err != nil {
				respondError(c, err)
				return
			}
			c.JSON(http.StatusAccepted, gin.H{"beer_id": beerId})
		})
		warehousesRoutes.GET("/availability", func(c *gin.Context) {
			c.JSON(http.StatusOK, warehousesFacade.GetAvailabilities(c.Request.Context()))
		})
		warehousesRoutes.GET("/availability/:beerId", func(c *gin.Context) {
			availability, found := warehousesFacade.GetAvailability(c.Request.Context(), c.Param("beerId"))
			if !found {
				c.JSON(http.StatusNotFound, gin.H{"error": "availability not found"})
				return
			}
			c.JSON(http.StatusOK, availability)
		})
	}

	return engine
}

func respondError(c *gin.Context, err error) {
	var invalid muflone.ErrInvalid
	if errors.As(err, &invalid) {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
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
