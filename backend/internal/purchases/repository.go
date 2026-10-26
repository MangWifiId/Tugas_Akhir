package purchases

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/big"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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
	req CreatePurchaseRequest,
) (Purchase, error) {
	var purchase Purchase

	// Validasi duplikasi produk dan overflow subtotal sebelum transaksi.
	seen := make(map[int64]bool)
	var subtotal int64

	for _, item := range req.Items {
		if item.ProductID <= 0 || item.Quantity <= 0 || item.UnitCost < 0 {
			return Purchase{}, fmt.Errorf("detail pembelian tidak valid")
		}
		if seen[item.ProductID] {
			return Purchase{}, fmt.Errorf("produk yang sama tidak boleh muncul dua kali")
		}
		seen[item.ProductID] = true

		line := new(big.Int).Mul(
			big.NewInt(int64(item.Quantity)),
			big.NewInt(item.UnitCost),
		)
		if !line.IsInt64() {
			return Purchase{}, fmt.Errorf("nilai detail pembelian terlalu besar")
		}

		if item.UnitCost > 0 &&
			int64(item.Quantity) > math.MaxInt64/item.UnitCost {
			return Purchase{}, fmt.Errorf("nilai detail pembelian terlalu besar")
		}

		if subtotal > math.MaxInt64-line.Int64() {
			return Purchase{}, fmt.Errorf("subtotal pembelian terlalu besar")
		}
		subtotal += line.Int64()
	}

	err := pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, `
			INSERT INTO purchases (
				invoice_number, supplier_name, subtotal, notes
			)
			VALUES ($1, $2, $3, $4)
			RETURNING id, invoice_number, supplier_name, subtotal,
			          notes, created_at, updated_at
		`,
			req.InvoiceNumber,
			req.SupplierName,
			subtotal,
			req.Notes,
		).Scan(
			&purchase.ID,
			&purchase.InvoiceNumber,
			&purchase.SupplierName,
			&purchase.Subtotal,
			&purchase.Notes,
			&purchase.CreatedAt,
			&purchase.UpdatedAt,
		)
		if err != nil {
			return fmt.Errorf("gagal membuat faktur pembelian: %w", err)
		}

		for _, item := range req.Items {
			var currentStock int
			var oldCost int64
			var isActive bool

			err := tx.QueryRow(ctx, `
				SELECT stock, purchase_price, is_active
				FROM products
				WHERE id = $1
				FOR UPDATE
			`, item.ProductID).Scan(
				&currentStock,
				&oldCost,
				&isActive,
			)
			if errors.Is(err, pgx.ErrNoRows) {
				return fmt.Errorf("produk ID %d tidak ditemukan", item.ProductID)
			}
			if err != nil {
				return fmt.Errorf("gagal membaca produk: %w", err)
			}
			if !isActive {
				return fmt.Errorf("produk ID %d tidak aktif", item.ProductID)
			}

			if item.Quantity > math.MaxInt-currentStock {
				return fmt.Errorf("stok produk ID %d terlalu besar", item.ProductID)
			}

			newStock := currentStock + item.Quantity

			// Jika belum ada stok, HPP awal mengikuti harga pembelian.
			// Jika sudah ada stok, gunakan moving weighted average.
			var newCost int64
			if currentStock == 0 {
				newCost = item.UnitCost
			} else {
				oldValue := new(big.Int).Mul(
					big.NewInt(int64(currentStock)),
					big.NewInt(oldCost),
				)
				newValue := new(big.Int).Mul(
					big.NewInt(int64(item.Quantity)),
					big.NewInt(item.UnitCost),
				)
				totalValue := new(big.Int).Add(oldValue, newValue)
				totalQty := big.NewInt(int64(newStock))
				totalValue.Quo(totalValue, totalQty)

				if !totalValue.IsInt64() {
					return fmt.Errorf("HPP produk ID %d terlalu besar", item.ProductID)
				}
				newCost = totalValue.Int64()
			}

			if _, err := tx.Exec(ctx, `
				UPDATE products
				SET stock = $1,
				    purchase_price = $2,
				    updated_at = NOW()
				WHERE id = $3
			`, newStock, newCost, item.ProductID); err != nil {
				return fmt.Errorf("gagal memperbarui stok dan HPP: %w", err)
			}

			if _, err := tx.Exec(ctx, `
    INSERT INTO purchase_items (
        purchase_id, product_id, quantity, unit_cost
    )
    VALUES ($1, $2, $3, $4)
`,
				purchase.ID,
				item.ProductID,
				item.Quantity,
				item.UnitCost,
			); err != nil {
				return fmt.Errorf("gagal menyimpan detail pembelian: %w", err)
			}

			if _, err := tx.Exec(ctx, `
				INSERT INTO stock_movements (
					product_id, movement_type, quantity,
					stock_before, stock_after, notes
				)
				VALUES ($1, 'in', $2, $3, $4, $5)
			`,
				item.ProductID,
				item.Quantity,
				currentStock,
				newStock,
				"Pembelian "+req.InvoiceNumber,
			); err != nil {
				return fmt.Errorf("gagal mencatat riwayat stok: %w", err)
			}
		}

		return nil
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return Purchase{}, fmt.Errorf("nomor faktur sudah digunakan")
		}
		return Purchase{}, err
	}

	return purchase, nil
}
