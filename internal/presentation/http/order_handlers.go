package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/cjgalvisc96/cj-beer-company/internal/orders/application/commands"
	"github.com/cjgalvisc96/cj-beer-company/internal/orders/application/queries"
)

type OrderHandlers struct {
	placeOrder  *commands.PlaceOrderHandler
	cancelOrder *commands.CancelOrderHandler
	getOrder    *queries.GetOrderHandler
	listOrders  *queries.ListOrdersHandler
}

func NewOrderHandlers(
	placeOrder *commands.PlaceOrderHandler,
	cancelOrder *commands.CancelOrderHandler,
	getOrder *queries.GetOrderHandler,
	listOrders *queries.ListOrdersHandler,
) *OrderHandlers {
	return &OrderHandlers{
		placeOrder:  placeOrder,
		cancelOrder: cancelOrder,
		getOrder:    getOrder,
		listOrders:  listOrders,
	}
}

type placeOrderRequest struct {
	CustomerName string                  `json:"customer_name" binding:"required"`
	Lines        []placeOrderLineRequest `json:"lines" binding:"required,min=1,dive"`
}

type placeOrderLineRequest struct {
	BeerID string `json:"beer_id" binding:"required"`
	Units  int    `json:"units" binding:"required,gt=0"`
}

func (h *OrderHandlers) Place(c *gin.Context) {
	var req placeOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBindError(c, err)
		return
	}
	input := commands.PlaceOrderInput{CustomerName: req.CustomerName}
	for _, line := range req.Lines {
		input.Lines = append(input.Lines, commands.PlaceOrderLine{
			BeerID: line.BeerID,
			Units:  line.Units,
		})
	}
	order, err := h.placeOrder.Handle(c.Request.Context(), input)
	if err != nil {
		respondError(c, err)
		return
	}
	// 202: the order is accepted as pending; confirmation happens
	// asynchronously once inventory reserves the stock.
	c.JSON(http.StatusAccepted, order)
}

func (h *OrderHandlers) Cancel(c *gin.Context) {
	order, err := h.cancelOrder.Handle(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, order)
}

func (h *OrderHandlers) Get(c *gin.Context) {
	order, err := h.getOrder.Handle(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, order)
}

func (h *OrderHandlers) List(c *gin.Context) {
	orders, err := h.listOrders.Handle(c.Request.Context())
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, orders)
}
