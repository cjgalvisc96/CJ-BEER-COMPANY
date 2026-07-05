package services_test

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cjgalvisc96/cj-beer-company/internal/muflone"
	"github.com/cjgalvisc96/cj-beer-company/internal/sales/readmodel/dtos"
	"github.com/cjgalvisc96/cj-beer-company/internal/sales/readmodel/services"
	"github.com/cjgalvisc96/cj-beer-company/internal/shared/customtypes"
)

func newPostgresService(t *testing.T) (*services.PostgresSalesOrderService, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return services.NewPostgresSalesOrderService(db), mock
}

func sampleOrder() dtos.SalesOrder {
	return dtos.SalesOrder{
		Id: "order-1", SalesOrderNumber: "20240315-1500", OrderDate: time.Now().UTC(),
		CustomerId: "customer-1", CustomerName: "Muflone",
		Rows: []dtos.SalesOrderRow{{
			BeerId: "beer-1", BeerName: "BrewUp IPA",
			Quantity: customtypes.NewQuantity(10, "Lt"),
			Price:    customtypes.NewPrice(5, "EUR"),
		}},
	}
}

func orderColumns() *sqlmock.Rows {
	return sqlmock.NewRows([]string{"id", "sales_order_number", "order_date", "customer_id", "customer_name",
		"allocation_status", "rejection_reason"})
}

func addOrderRow(rows *sqlmock.Rows, order dtos.SalesOrder) *sqlmock.Rows {
	return rows.AddRow(order.Id, order.SalesOrderNumber, order.OrderDate,
		order.CustomerId, order.CustomerName, "pending", "")
}

func rowColumns() *sqlmock.Rows {
	return sqlmock.NewRows([]string{"beer_id", "beer_name", "quantity", "unit_of_measure", "price", "currency"})
}

func TestPostgresCreateSalesOrder(t *testing.T) {
	service, mock := newPostgresService(t)
	order := sampleOrder()
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO sales_orders").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("DELETE FROM sales_order_rows").WithArgs(order.Id).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("INSERT INTO sales_order_rows").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	require.NoError(t, service.CreateSalesOrder(context.Background(), order))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresCreateSalesOrderErrors(t *testing.T) {
	order := sampleOrder()
	ctx := context.Background()

	t.Run("begin fails", func(t *testing.T) {
		service, mock := newPostgresService(t)
		mock.ExpectBegin().WillReturnError(assert.AnError)
		assert.ErrorIs(t, service.CreateSalesOrder(ctx, order), assert.AnError)
	})
	t.Run("order upsert fails", func(t *testing.T) {
		service, mock := newPostgresService(t)
		mock.ExpectBegin()
		mock.ExpectExec("INSERT INTO sales_orders").WillReturnError(assert.AnError)
		mock.ExpectRollback()
		assert.ErrorIs(t, service.CreateSalesOrder(ctx, order), assert.AnError)
	})
	t.Run("row delete fails", func(t *testing.T) {
		service, mock := newPostgresService(t)
		mock.ExpectBegin()
		mock.ExpectExec("INSERT INTO sales_orders").WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec("DELETE FROM sales_order_rows").WillReturnError(assert.AnError)
		mock.ExpectRollback()
		assert.ErrorIs(t, service.CreateSalesOrder(ctx, order), assert.AnError)
	})
	t.Run("row insert fails", func(t *testing.T) {
		service, mock := newPostgresService(t)
		mock.ExpectBegin()
		mock.ExpectExec("INSERT INTO sales_orders").WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec("DELETE FROM sales_order_rows").WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectExec("INSERT INTO sales_order_rows").WillReturnError(assert.AnError)
		mock.ExpectRollback()
		assert.ErrorIs(t, service.CreateSalesOrder(ctx, order), assert.AnError)
	})
	t.Run("commit fails", func(t *testing.T) {
		service, mock := newPostgresService(t)
		mock.ExpectBegin()
		mock.ExpectExec("INSERT INTO sales_orders").WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec("DELETE FROM sales_order_rows").WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectExec("INSERT INTO sales_order_rows").WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit().WillReturnError(assert.AnError)
		assert.ErrorIs(t, service.CreateSalesOrder(ctx, order), assert.AnError)
	})
}

func TestPostgresUpdateAllocationStatus(t *testing.T) {
	service, mock := newPostgresService(t)
	mock.ExpectExec("INSERT INTO sales_orders").
		WithArgs("order-1", "rejected", "shortage").
		WillReturnResult(sqlmock.NewResult(0, 1))

	require.NoError(t, service.UpdateAllocationStatus(context.Background(), "order-1", "rejected", "shortage"))
	require.NoError(t, mock.ExpectationsWereMet())

	failing, failingMock := newPostgresService(t)
	failingMock.ExpectExec("INSERT INTO sales_orders").WillReturnError(assert.AnError)
	assert.ErrorIs(t,
		failing.UpdateAllocationStatus(context.Background(), "order-1", "allocated", ""),
		assert.AnError)
}

func TestPostgresGetSalesOrder(t *testing.T) {
	service, mock := newPostgresService(t)
	order := sampleOrder()
	mock.ExpectQuery("SELECT id, sales_order_number").WithArgs(order.Id).
		WillReturnRows(addOrderRow(orderColumns(), order))
	mock.ExpectQuery("SELECT beer_id, beer_name").WithArgs(order.Id).
		WillReturnRows(rowColumns().AddRow("beer-1", "BrewUp IPA", 10, "Lt", 5.0, "EUR"))

	loaded, err := service.GetSalesOrder(context.Background(), order.Id)

	require.NoError(t, err)
	assert.Equal(t, order.SalesOrderNumber, loaded.SalesOrderNumber)
	require.Len(t, loaded.Rows, 1)
	assert.Equal(t, 10, loaded.Rows[0].Quantity.Value)
}

func TestPostgresGetSalesOrderErrors(t *testing.T) {
	ctx := context.Background()

	t.Run("not found", func(t *testing.T) {
		service, mock := newPostgresService(t)
		mock.ExpectQuery("SELECT id, sales_order_number").WillReturnRows(orderColumns())
		_, err := service.GetSalesOrder(ctx, "missing")
		assert.ErrorIs(t, err, muflone.ErrNotFound)
	})
	t.Run("query fails", func(t *testing.T) {
		service, mock := newPostgresService(t)
		mock.ExpectQuery("SELECT id, sales_order_number").WillReturnError(assert.AnError)
		_, err := service.GetSalesOrder(ctx, "x")
		assert.ErrorIs(t, err, assert.AnError)
	})
	t.Run("rows query fails", func(t *testing.T) {
		service, mock := newPostgresService(t)
		order := sampleOrder()
		mock.ExpectQuery("SELECT id, sales_order_number").WillReturnRows(addOrderRow(orderColumns(), order))
		mock.ExpectQuery("SELECT beer_id, beer_name").WillReturnError(assert.AnError)
		_, err := service.GetSalesOrder(ctx, order.Id)
		assert.ErrorIs(t, err, assert.AnError)
	})
	t.Run("row scan fails", func(t *testing.T) {
		service, mock := newPostgresService(t)
		order := sampleOrder()
		mock.ExpectQuery("SELECT id, sales_order_number").WillReturnRows(addOrderRow(orderColumns(), order))
		mock.ExpectQuery("SELECT beer_id, beer_name").WillReturnRows(
			rowColumns().AddRow("b", "n", "not-an-int", "Lt", 5.0, "EUR"))
		_, err := service.GetSalesOrder(ctx, order.Id)
		assert.ErrorContains(t, err, "rows")
	})
	t.Run("row iteration fails", func(t *testing.T) {
		service, mock := newPostgresService(t)
		order := sampleOrder()
		mock.ExpectQuery("SELECT id, sales_order_number").WillReturnRows(addOrderRow(orderColumns(), order))
		mock.ExpectQuery("SELECT beer_id, beer_name").WillReturnRows(
			rowColumns().AddRow("b", "n", 1, "Lt", 5.0, "EUR").RowError(0, assert.AnError))
		_, err := service.GetSalesOrder(ctx, order.Id)
		assert.ErrorIs(t, err, assert.AnError)
	})
}

func TestPostgresGetSalesOrders(t *testing.T) {
	service, mock := newPostgresService(t)
	order := sampleOrder()
	mock.ExpectQuery("SELECT id, sales_order_number").WillReturnRows(addOrderRow(orderColumns(), order))
	mock.ExpectQuery("SELECT beer_id, beer_name").WithArgs(order.Id).
		WillReturnRows(rowColumns().AddRow("beer-1", "BrewUp IPA", 10, "Lt", 5.0, "EUR"))

	orders, err := service.GetSalesOrders(context.Background(), customtypes.NewPage(0, 0))

	require.NoError(t, err)
	require.Len(t, orders, 1)
	require.Len(t, orders[0].Rows, 1)
}

func TestPostgresGetSalesOrdersErrors(t *testing.T) {
	ctx := context.Background()
	order := sampleOrder()

	t.Run("query fails", func(t *testing.T) {
		service, mock := newPostgresService(t)
		mock.ExpectQuery("SELECT id, sales_order_number").WillReturnError(assert.AnError)
		_, err := service.GetSalesOrders(ctx, customtypes.NewPage(0, 0))
		assert.ErrorIs(t, err, assert.AnError)
	})
	t.Run("scan fails", func(t *testing.T) {
		service, mock := newPostgresService(t)
		mock.ExpectQuery("SELECT id, sales_order_number").WillReturnRows(
			orderColumns().AddRow(nil, nil, "not-a-time", nil, nil, nil, nil))
		_, err := service.GetSalesOrders(ctx, customtypes.NewPage(0, 0))
		assert.Error(t, err)
	})
	t.Run("iteration fails", func(t *testing.T) {
		service, mock := newPostgresService(t)
		mock.ExpectQuery("SELECT id, sales_order_number").WillReturnRows(addOrderRow(orderColumns(), order).RowError(0, assert.AnError))
		_, err := service.GetSalesOrders(ctx, customtypes.NewPage(0, 0))
		assert.ErrorIs(t, err, assert.AnError)
	})
	t.Run("rows lookup fails", func(t *testing.T) {
		service, mock := newPostgresService(t)
		mock.ExpectQuery("SELECT id, sales_order_number").WillReturnRows(addOrderRow(orderColumns(), order))
		mock.ExpectQuery("SELECT beer_id, beer_name").WillReturnError(assert.AnError)
		_, err := service.GetSalesOrders(ctx, customtypes.NewPage(0, 0))
		assert.ErrorIs(t, err, assert.AnError)
	})
}

func TestPostgresReset(t *testing.T) {
	service, mock := newPostgresService(t)
	mock.ExpectExec("DELETE FROM sales_orders").WillReturnResult(sqlmock.NewResult(0, 3))
	require.NoError(t, service.Reset(context.Background()))

	failing, failingMock := newPostgresService(t)
	failingMock.ExpectExec("DELETE FROM sales_orders").WillReturnError(assert.AnError)
	assert.ErrorIs(t, failing.Reset(context.Background()), assert.AnError)
}
