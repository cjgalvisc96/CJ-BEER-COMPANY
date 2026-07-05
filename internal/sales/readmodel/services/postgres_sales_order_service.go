package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/cjgalvisc96/cj-beer-company/internal/muflone"
	"github.com/cjgalvisc96/cj-beer-company/internal/sales/readmodel/dtos"
	"github.com/cjgalvisc96/cj-beer-company/internal/shared/customtypes"
)

// PostgresSalesOrderService is the durable adapter over the projection
// tables (sales_orders, sales_order_rows) versioned in migrations/.
type PostgresSalesOrderService struct {
	db *sql.DB
}

func NewPostgresSalesOrderService(db *sql.DB) *PostgresSalesOrderService {
	return &PostgresSalesOrderService{db: db}
}

func (s *PostgresSalesOrderService) CreateSalesOrder(ctx context.Context, order dtos.SalesOrder) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("project sales order %s: %w", order.Id, err)
	}
	defer func() { _ = tx.Rollback() }()

	// The upsert deliberately leaves allocation_status/rejection_reason
	// alone: a saga outcome that landed first must be preserved.
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO sales_orders (id, sales_order_number, order_date, customer_id, customer_name)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (id) DO UPDATE SET
		   sales_order_number = excluded.sales_order_number,
		   order_date = excluded.order_date,
		   customer_id = excluded.customer_id,
		   customer_name = excluded.customer_name`,
		order.Id, order.SalesOrderNumber, order.OrderDate, order.CustomerId, order.CustomerName,
	); err != nil {
		return fmt.Errorf("project sales order %s: %w", order.Id, err)
	}
	if _, err := tx.ExecContext(ctx,
		`DELETE FROM sales_order_rows WHERE sales_order_id = $1`, order.Id,
	); err != nil {
		return fmt.Errorf("project sales order %s rows: %w", order.Id, err)
	}
	for _, row := range order.Rows {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO sales_order_rows
			   (sales_order_id, beer_id, beer_name, quantity, unit_of_measure, price, currency)
			 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			order.Id, row.BeerId, row.BeerName,
			row.Quantity.Value, row.Quantity.UnitOfMeasure,
			row.Price.Value, row.Price.Currency,
		); err != nil {
			return fmt.Errorf("project sales order %s row %s: %w", order.Id, row.BeerId, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("project sales order %s: %w", order.Id, err)
	}
	return nil
}

// Reset wipes the projections so a rebuild can replay the event store
// from scratch (rows cascade from orders).
func (s *PostgresSalesOrderService) Reset(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM sales_orders`); err != nil {
		return fmt.Errorf("reset sales projections: %w", err)
	}
	return nil
}

// UpdateAllocationStatus records the saga outcome; a stub row absorbs the
// status when it lands before the created projection.
func (s *PostgresSalesOrderService) UpdateAllocationStatus(ctx context.Context, salesOrderId, status, reason string) error {
	if _, err := s.db.ExecContext(ctx,
		`INSERT INTO sales_orders (id, sales_order_number, order_date, customer_id, customer_name,
		                           allocation_status, rejection_reason)
		 VALUES ($1, '', now(), gen_random_uuid(), '', $2, $3)
		 ON CONFLICT (id) DO UPDATE SET
		   allocation_status = excluded.allocation_status,
		   rejection_reason = excluded.rejection_reason`,
		salesOrderId, status, reason,
	); err != nil {
		return fmt.Errorf("project allocation status of %s: %w", salesOrderId, err)
	}
	return nil
}

func (s *PostgresSalesOrderService) GetSalesOrder(ctx context.Context, id string) (dtos.SalesOrder, error) {
	var order dtos.SalesOrder
	err := s.db.QueryRowContext(ctx,
		`SELECT id, sales_order_number, order_date, customer_id, customer_name,
		        allocation_status, rejection_reason
		   FROM sales_orders WHERE id = $1`, id,
	).Scan(&order.Id, &order.SalesOrderNumber, &order.OrderDate, &order.CustomerId, &order.CustomerName,
		&order.AllocationStatus, &order.RejectionReason)
	if errors.Is(err, sql.ErrNoRows) {
		return dtos.SalesOrder{}, fmt.Errorf("%w: sales order %s", muflone.ErrNotFound, id)
	}
	if err != nil {
		return dtos.SalesOrder{}, fmt.Errorf("get sales order %s: %w", id, err)
	}
	rows, err := s.rowsOf(ctx, id)
	if err != nil {
		return dtos.SalesOrder{}, err
	}
	order.Rows = rows
	return order, nil
}

func (s *PostgresSalesOrderService) GetSalesOrders(ctx context.Context, page customtypes.Page) ([]dtos.SalesOrder, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, sales_order_number, order_date, customer_id, customer_name,
		        allocation_status, rejection_reason
		   FROM sales_orders ORDER BY projected_at, id LIMIT $1 OFFSET $2`,
		page.Limit, page.Offset)
	if err != nil {
		return nil, fmt.Errorf("list sales orders: %w", err)
	}
	defer rows.Close()

	orders := make([]dtos.SalesOrder, 0)
	for rows.Next() {
		var order dtos.SalesOrder
		if err := rows.Scan(&order.Id, &order.SalesOrderNumber, &order.OrderDate,
			&order.CustomerId, &order.CustomerName,
			&order.AllocationStatus, &order.RejectionReason); err != nil {
			return nil, fmt.Errorf("list sales orders: %w", err)
		}
		orders = append(orders, order)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list sales orders: %w", err)
	}
	for i := range orders {
		orderRows, err := s.rowsOf(ctx, orders[i].Id)
		if err != nil {
			return nil, err
		}
		orders[i].Rows = orderRows
	}
	return orders, nil
}

func (s *PostgresSalesOrderService) rowsOf(ctx context.Context, orderId string) ([]dtos.SalesOrderRow, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT beer_id, beer_name, quantity, unit_of_measure, price, currency
		   FROM sales_order_rows WHERE sales_order_id = $1 ORDER BY beer_id`, orderId)
	if err != nil {
		return nil, fmt.Errorf("get sales order %s rows: %w", orderId, err)
	}
	defer rows.Close()

	orderRows := make([]dtos.SalesOrderRow, 0)
	for rows.Next() {
		var row dtos.SalesOrderRow
		if err := rows.Scan(&row.BeerId, &row.BeerName,
			&row.Quantity.Value, &row.Quantity.UnitOfMeasure,
			&row.Price.Value, &row.Price.Currency); err != nil {
			return nil, fmt.Errorf("get sales order %s rows: %w", orderId, err)
		}
		orderRows = append(orderRows, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("get sales order %s rows: %w", orderId, err)
	}
	return orderRows, nil
}
