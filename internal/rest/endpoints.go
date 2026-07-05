// Package rest is the BrewUp.Rest equivalent: the HTTP entry points of
// the monolith. By design (and enforced by the architecture fitness test
// in tests/), it depends ONLY on each module's facade — never on a
// module's domain, shared kernel, or read-model internals.
package rest

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/cjgalvisc96/cj-beer-company/internal/muflone"
	"github.com/cjgalvisc96/cj-beer-company/internal/sales"
	"github.com/cjgalvisc96/cj-beer-company/internal/warehouses"
)

// ReadinessCheck reports whether downstream dependencies (the database)
// are reachable. Nil means the app has none (in-memory mode).
type ReadinessCheck func(ctx context.Context) error

// NewRouter maps the endpoints: /v1/sales and /v1/warehouses, mirroring
// the book's MapSalesEndpoints / MapWarehousesEndpoints, plus the
// liveness and readiness probes. A non-nil verifier turns on
// authentication (OIDC bearer tokens) and RBAC on every /v1 route; the
// probes stay open.
func NewRouter(
	logger *slog.Logger,
	salesFacade *sales.Facade,
	warehousesFacade *warehouses.Facade,
	ready ReadinessCheck,
	verifier TokenVerifier,
) *gin.Engine {
	engine := gin.New()
	engine.Use(gin.Recovery(), requestLogger(logger))

	engine.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	engine.GET("/readyz", func(c *gin.Context) {
		if ready != nil {
			if err := ready(c.Request.Context()); err != nil {
				c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not ready", "error": err.Error()})
				return
			}
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})

	v1 := engine.Group("/v1")
	v1.Use(authenticate(verifier))
	viewer := requireRole(verifier, RoleViewer)

	salesRoutes := v1.Group("/sales")
	{
		salesRoutes.POST("", requireRole(verifier, RoleSalesManager), func(c *gin.Context) {
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
		salesRoutes.GET("", viewer, func(c *gin.Context) {
			orders, err := salesFacade.GetSalesOrders(c.Request.Context())
			if err != nil {
				respondError(c, err)
				return
			}
			c.JSON(http.StatusOK, orders)
		})
		salesRoutes.GET("/:id", viewer, func(c *gin.Context) {
			order, err := salesFacade.GetSalesOrder(c.Request.Context(), c.Param("id"))
			if err != nil {
				respondError(c, err)
				return
			}
			c.JSON(http.StatusOK, order)
		})
	}

	warehousesRoutes := v1.Group("/warehouses")
	{
		warehousesRoutes.POST("/availability", requireRole(verifier, RoleWarehouseOperator), func(c *gin.Context) {
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
		warehousesRoutes.GET("/availability", viewer, func(c *gin.Context) {
			availabilities, err := warehousesFacade.GetAvailabilities(c.Request.Context())
			if err != nil {
				respondError(c, err)
				return
			}
			c.JSON(http.StatusOK, availabilities)
		})
		warehousesRoutes.GET("/availability/:beerId", viewer, func(c *gin.Context) {
			availability, err := warehousesFacade.GetAvailability(c.Request.Context(), c.Param("beerId"))
			if err != nil {
				respondError(c, err)
				return
			}
			c.JSON(http.StatusOK, availability)
		})
	}

	return engine
}

func respondError(c *gin.Context, err error) {
	var invalid muflone.ErrInvalid
	switch {
	case errors.As(err, &invalid):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, muflone.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}
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
