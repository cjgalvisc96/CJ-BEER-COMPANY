package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/cjgalvisc96/cj-beer-company/internal/brewing/application/commands"
	"github.com/cjgalvisc96/cj-beer-company/internal/brewing/application/queries"
)

type BatchHandlers struct {
	startBatch    *commands.StartBatchHandler
	completeBatch *commands.CompleteBatchHandler
	getBatch      *queries.GetBatchHandler
	listBatches   *queries.ListBatchesHandler
}

func NewBatchHandlers(
	startBatch *commands.StartBatchHandler,
	completeBatch *commands.CompleteBatchHandler,
	getBatch *queries.GetBatchHandler,
	listBatches *queries.ListBatchesHandler,
) *BatchHandlers {
	return &BatchHandlers{
		startBatch:    startBatch,
		completeBatch: completeBatch,
		getBatch:      getBatch,
		listBatches:   listBatches,
	}
}

type startBatchRequest struct {
	BeerID string `json:"beer_id" binding:"required"`
	Units  int    `json:"units" binding:"required,gt=0"`
}

func (h *BatchHandlers) Start(c *gin.Context) {
	var req startBatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBindError(c, err)
		return
	}
	batch, err := h.startBatch.Handle(c.Request.Context(), commands.StartBatchInput{
		BeerID: req.BeerID,
		Units:  req.Units,
	})
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, batch)
}

type completeBatchRequest struct {
	ProducedUnits int `json:"produced_units" binding:"required,gt=0"`
}

func (h *BatchHandlers) Complete(c *gin.Context) {
	var req completeBatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBindError(c, err)
		return
	}
	batch, err := h.completeBatch.Handle(c.Request.Context(), commands.CompleteBatchInput{
		BatchID:       c.Param("id"),
		ProducedUnits: req.ProducedUnits,
	})
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, batch)
}

func (h *BatchHandlers) Get(c *gin.Context) {
	batch, err := h.getBatch.Handle(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, batch)
}

func (h *BatchHandlers) List(c *gin.Context) {
	batches, err := h.listBatches.Handle(c.Request.Context())
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, batches)
}
