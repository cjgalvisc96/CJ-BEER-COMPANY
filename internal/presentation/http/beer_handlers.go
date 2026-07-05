package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/cjgalvisc96/cj-beer-company/internal/catalog/application/commands"
	"github.com/cjgalvisc96/cj-beer-company/internal/catalog/application/queries"
)

type BeerHandlers struct {
	createBeer  *commands.CreateBeerHandler
	changePrice *commands.ChangeBeerPriceHandler
	retireBeer  *commands.RetireBeerHandler
	getBeer     *queries.GetBeerHandler
	listBeers   *queries.ListBeersHandler
}

func NewBeerHandlers(
	createBeer *commands.CreateBeerHandler,
	changePrice *commands.ChangeBeerPriceHandler,
	retireBeer *commands.RetireBeerHandler,
	getBeer *queries.GetBeerHandler,
	listBeers *queries.ListBeersHandler,
) *BeerHandlers {
	return &BeerHandlers{
		createBeer:  createBeer,
		changePrice: changePrice,
		retireBeer:  retireBeer,
		getBeer:     getBeer,
		listBeers:   listBeers,
	}
}

type createBeerRequest struct {
	Name        string  `json:"name" binding:"required"`
	Style       string  `json:"style" binding:"required"`
	ABV         float64 `json:"abv"`
	PriceCents  int64   `json:"price_cents" binding:"required"`
	Currency    string  `json:"currency" binding:"required"`
	Description string  `json:"description"`
}

func (h *BeerHandlers) Create(c *gin.Context) {
	var req createBeerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBindError(c, err)
		return
	}
	beer, err := h.createBeer.Handle(c.Request.Context(), commands.CreateBeerInput{
		Name:        req.Name,
		Style:       req.Style,
		ABV:         req.ABV,
		PriceCents:  req.PriceCents,
		Currency:    req.Currency,
		Description: req.Description,
	})
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, beer)
}

type changeBeerPriceRequest struct {
	PriceCents int64  `json:"price_cents" binding:"required"`
	Currency   string `json:"currency" binding:"required"`
}

func (h *BeerHandlers) ChangePrice(c *gin.Context) {
	var req changeBeerPriceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBindError(c, err)
		return
	}
	beer, err := h.changePrice.Handle(c.Request.Context(), commands.ChangeBeerPriceInput{
		BeerID:     c.Param("id"),
		PriceCents: req.PriceCents,
		Currency:   req.Currency,
	})
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, beer)
}

func (h *BeerHandlers) Retire(c *gin.Context) {
	if err := h.retireBeer.Handle(c.Request.Context(), c.Param("id")); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *BeerHandlers) Get(c *gin.Context) {
	beer, err := h.getBeer.Handle(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, beer)
}

func (h *BeerHandlers) List(c *gin.Context) {
	beers, err := h.listBeers.Handle(c.Request.Context())
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, beers)
}
