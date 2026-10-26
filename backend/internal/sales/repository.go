package sales

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(
	ctx context.Context,
	req CreateSaleRequest,
) (Sale, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Sale{}, err
	}
	defer tx.Rollback(ctx)

	sale := Sale{
		InvoiceNumber: fmt.Sprintf(
			"INV-%s",
			time.Now().Format("20060102150405.000000"),
		),
		PaymentMethod: "cash",
		Status:        "completed",
		Discount:      0,
		Notes:         req.Notes,
		Items:         make([]SaleItem, 0, len(req.Items)),
	}

	// Buat invoice sementara. Total diperbarui setelah seluruh
	// harga dan subtotal barang berhasil dihitung.
	err = tx.QueryRow(ctx, `
		INSERT INTO sales (
			invoice_number, subtotal, discount, total,
			payment_method, status, notes
		)
		VALUES ($1, 0, 0, 0, 'cash', 'completed', $2)
		RETURNING id, created_at, updated_at
	`, sale.InvoiceNumber, sale.Notes).Scan(
		&sale.ID,
		&sale.CreatedAt,
		&sale.UpdatedAt,
	)
	if err != nil {
		return Sale{}, err
	}

	// Kunci baris produk supaya stok tidak digunakan
	// oleh transaksi bersamaan secara berlebihan.
	for _, item := range req.Items {
		var name string
		var price int64
		var stock int
		var active bool

		err = tx.QueryRow(ctx, `
			SELECT name, selling_price, stock, is_active
			FROM products
			WHERE id = $1
			FOR UPDATE
		`, item.ProductID).Scan(
			&name,
			&price,
			&stock,
			&active,
		)
		if err != nil {
			if err == pgx.ErrNoRows {
				return Sale{}, fmt.Errorf(
					"produk dengan ID %d tidak ditemukan",
					item.ProductID,
				)
			}
			return Sale{}, err
		}

		if !active {
			return Sale{}, fmt.Errorf(
				"produk %s tidak aktif",
				name,
			)
		}

		if stock < item.Quantity {
			return Sale{}, fmt.Errorf(
				"stok produk %s tidak mencukupi",
				name,
			)
		}

		itemSubtotal := price * int64(item.Quantity)
		sale.Subtotal += itemSubtotal

		// Simpan detail barang pada invoice.
		var saleItem SaleItem
		err = tx.QueryRow(ctx, `
			INSERT INTO sale_items (
				sale_id, product_id, quantity,
				unit_price, subtotal
			)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING id, sale_id, product_id,
			          quantity, unit_price, subtotal
		`,
			sale.ID,
			item.ProductID,
			item.Quantity,
			price,
			itemSubtotal,
		).Scan(
			&saleItem.ID,
			&saleItem.SaleID,
			&saleItem.ProductID,
			&saleItem.Quantity,
			&saleItem.UnitPrice,
			&saleItem.Subtotal,
		)
		if err != nil {
			return Sale{}, err
		}

		sale.Items = append(sale.Items, saleItem)

		// Kurangi stok dan catat pergerakannya.
		stockAfter := stock - item.Quantity

		_, err = tx.Exec(ctx, `
			UPDATE products
			SET stock = $1, updated_at = NOW()
			WHERE id = $2
		`, stockAfter, item.ProductID)
		if err != nil {
			return Sale{}, err
		}

		_, err = tx.Exec(ctx, `
			INSERT INTO stock_movements (
				product_id, movement_type, quantity,
				stock_before, stock_after, notes
			)
			VALUES ($1, 'out', $2, $3, $4, $5)
		`,
			item.ProductID,
			item.Quantity,
			stock,
			stockAfter,
			"Penjualan "+sale.InvoiceNumber,
		)
		if err != nil {
			return Sale{}, err
		}
	}

	sale.Total = sale.Subtotal

	// Simpan total akhir invoice.
	err = tx.QueryRow(ctx, `
		UPDATE sales
		SET subtotal = $1, total = $1, updated_at = NOW()
		WHERE id = $2
		RETURNING updated_at
	`, sale.Total, sale.ID).Scan(&sale.UpdatedAt)
	if err != nil {
		return Sale{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Sale{}, err
	}

	return sale, nil
}

func (r *Repository) GetAll(ctx context.Context) ([]Sale, error) {
	rows, err := r.db.Query(ctx, `
		SELECT
			id,
			invoice_number,
			subtotal,
			discount,
			total,
			payment_method,
			status,
			notes,
			created_at,
			updated_at
		FROM sales
		ORDER BY created_at DESC, id DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	salesList := make([]Sale, 0)

	for rows.Next() {
		var sale Sale

		err := rows.Scan(
			&sale.ID,
			&sale.InvoiceNumber,
			&sale.Subtotal,
			&sale.Discount,
			&sale.Total,
			&sale.PaymentMethod,
			&sale.Status,
			&sale.Notes,
			&sale.CreatedAt,
			&sale.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		salesList = append(salesList, sale)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return salesList, nil
}

func (r *Repository) GetByID(
	ctx context.Context,
	id int64,
) (Sale, error) {
	var sale Sale

	err := r.db.QueryRow(ctx, `
		SELECT
			id,
			invoice_number,
			subtotal,
			discount,
			total,
			payment_method,
			status,
			notes,
			created_at,
			updated_at
		FROM sales
		WHERE id = $1
	`, id).Scan(
		&sale.ID,
		&sale.InvoiceNumber,
		&sale.Subtotal,
		&sale.Discount,
		&sale.Total,
		&sale.PaymentMethod,
		&sale.Status,
		&sale.Notes,
		&sale.CreatedAt,
		&sale.UpdatedAt,
	)
	if err != nil {
		return Sale{}, err
	}

	rows, err := r.db.Query(ctx, `
		SELECT
			id,
			sale_id,
			product_id,
			quantity,
			unit_price,
			subtotal
		FROM sale_items
		WHERE sale_id = $1
		ORDER BY id
	`, sale.ID)
	if err != nil {
		return Sale{}, err
	}
	defer rows.Close()

	sale.Items = make([]SaleItem, 0)

	for rows.Next() {
		var item SaleItem

		if err := rows.Scan(
			&item.ID,
			&item.SaleID,
			&item.ProductID,
			&item.Quantity,
			&item.UnitPrice,
			&item.Subtotal,
		); err != nil {
			return Sale{}, err
		}

		sale.Items = append(sale.Items, item)
	}

	if err := rows.Err(); err != nil {
		return Sale{}, err
	}

	return sale, nil
}
