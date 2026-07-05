package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/cjgalvisc96/cj-beer-company/internal/inventory/application/commands"
	"github.com/cjgalvisc96/cj-beer-company/internal/inventory/application/queries"
)

type StockHandlers struct {
	trackItem *commands.TrackStockItemHandler
	replenish *commands.ReplenishStockHandler
	getStock  *queries.GetStockHandler
	listStock *queries.ListStockHandler
}

func NewStockHandlers(
	trackItem *commands.TrackStockItemHandler,
	replenish *commands.ReplenishStockHandler,
	getStock *queries.GetStockHandler,
	listStock *queries.ListStockHandler,
) *StockHandlers {
	return &StockHandlers{
		trackItem: trackItem,
		replenish: replenish,
		getStock:  getStock,
		listStock: listStock,
	}
}

type trackStockRequest struct {
	BeerID       string `json:"beer_id" binding:"required"`
	ReorderLevel int    `json:"reorder_level"`
}

func (h *StockHandlers) Track(c *gin.Context) {
	var req trackStockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBindError(c, err)
		return
	}
	stock, err := h.trackItem.Handle(c.Request.Context(), commands.TrackStockItemInput{
		BeerID:       req.BeerID,
		ReorderLevel: req.ReorderLevel,
	})
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, stock)
}

type replenishStockRequest struct {
	Units int `json:"units" binding:"required"`
}

func (h *StockHandlers) Replenish(c *gin.Context) {
	var req replenishStockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBindError(c, err)
		return
	}
	stock, err := h.replenish.Handle(c.Request.Context(), commands.ReplenishStockInput{
		BeerID: c.Param("beerId"),
		Units:  req.Units,
	})
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, stock)
}

func (h *StockHandlers) Get(c *gin.Context) {
	stock, err := h.getStock.Handle(c.Request.Context(), c.Param("beerId"))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, stock)
}

func (h *StockHandlers) List(c *gin.Context) {
	stock, err := h.listStock.Handle(c.Request.Context())
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, stock)
}
