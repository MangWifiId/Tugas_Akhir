package stock

import (
	"context"
	"errors"
	"fmt"

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

// Create mencatat pergerakan stok dan memperbarui stok produk
// dalam satu transaksi database.
func (r *Repository) Create(
	ctx context.Context,
	req CreateStockMovementRequest,
) (StockMovement, error) {
	var movement StockMovement

	err := pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		// Kunci baris produk agar perubahan stok bersamaan
		// tidak membaca dan mengubah stok secara tidak aman.
		var stockBefore int
		var isActive bool

		err := tx.QueryRow(ctx, `
			SELECT stock, is_active
			FROM products
			WHERE id = $1
			FOR UPDATE
		`, req.ProductID).Scan(&stockBefore, &isActive)

		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("produk tidak ditemukan")
		}
		if err != nil {
			return fmt.Errorf("gagal membaca stok produk: %w", err)
		}
		if !isActive {
			return fmt.Errorf("produk tidak aktif")
		}

		stockAfter := stockBefore

		switch req.MovementType {
		case "in":
			stockAfter += req.Quantity

		case "out":
			if req.Quantity > stockBefore {
				return fmt.Errorf("stok tidak mencukupi")
			}
			stockAfter -= req.Quantity

		case "adjustment":
			switch req.AdjustmentDirection {
			case "increase":
				stockAfter += req.Quantity
			case "decrease":
				if req.Quantity > stockBefore {
					return fmt.Errorf("stok tidak mencukupi untuk koreksi")
				}
				stockAfter -= req.Quantity
			default:
				return fmt.Errorf(
					"arah koreksi harus increase atau decrease",
				)
			}

		default:
			return fmt.Errorf("jenis pergerakan stok tidak valid")
		}

		// Perbarui stok produk.
		_, err = tx.Exec(ctx, `
			UPDATE products
			SET stock = $1, updated_at = NOW()
			WHERE id = $2
		`, stockAfter, req.ProductID)
		if err != nil {
			return fmt.Errorf("gagal memperbarui stok: %w", err)
		}

		// Simpan riwayat pergerakan stok.
		err = tx.QueryRow(ctx, `
			INSERT INTO stock_movements (
				product_id,
				movement_type,
				quantity,
				stock_before,
				stock_after,
				notes
			)
			VALUES ($1, $2, $3, $4, $5, $6)
			RETURNING id, product_id, movement_type, quantity,
			          stock_before, stock_after, notes, created_at
		`,
			req.ProductID,
			req.MovementType,
			req.Quantity,
			stockBefore,
			stockAfter,
			req.Notes,
		).Scan(
			&movement.ID,
			&movement.ProductID,
			&movement.MovementType,
			&movement.Quantity,
			&movement.StockBefore,
			&movement.StockAfter,
			&movement.Notes,
			&movement.CreatedAt,
		)
		if err != nil {
			return fmt.Errorf("gagal menyimpan riwayat stok: %w", err)
		}

		return nil
	})

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			return StockMovement{}, fmt.Errorf(
				"database gagal menyimpan pergerakan stok: %w", err,
			)
		}
		return StockMovement{}, err
	}

	return movement, nil
}

func (r *Repository) GetMovements(
	ctx context.Context,
	productID int64,
) ([]StockMovement, error) {
	query := `
		SELECT id, product_id, movement_type, quantity,
		       stock_before, stock_after, notes, created_at
		FROM stock_movements
	`

	args := []any{}

	if productID > 0 {
		query += ` WHERE product_id = $1`
		args = append(args, productID)
	}

	query += ` ORDER BY created_at DESC, id DESC`

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil riwayat stok: %w", err)
	}
	defer rows.Close()

	movements := make([]StockMovement, 0)

	for rows.Next() {
		var movement StockMovement

		err := rows.Scan(
			&movement.ID,
			&movement.ProductID,
			&movement.MovementType,
			&movement.Quantity,
			&movement.StockBefore,
			&movement.StockAfter,
			&movement.Notes,
			&movement.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("gagal membaca riwayat stok: %w", err)
		}

		movements = append(movements, movement)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("gagal membaca hasil riwayat stok: %w", err)
	}

	return movements, nil
}
