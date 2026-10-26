package products

import (
	"context"
	"errors"
	"fmt"

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
	req CreateProductRequest,
) (Product, error) {
	var product Product

	query := `
		INSERT INTO products (
			name, sku, category, purchase_price, selling_price
		)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, name, sku, category, purchase_price,
		          selling_price, stock, is_active, created_at, updated_at
	`

	err := r.db.QueryRow(
		ctx,
		query,
		req.Name,
		req.SKU,
		req.Category,
		req.PurchasePrice,
		req.SellingPrice,
	).Scan(
		&product.ID,
		&product.Name,
		&product.SKU,
		&product.Category,
		&product.PurchasePrice,
		&product.SellingPrice,
		&product.Stock,
		&product.IsActive,
		&product.CreatedAt,
		&product.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return Product{}, fmt.Errorf("SKU sudah digunakan: %w", err)
		}

		return Product{}, fmt.Errorf("gagal membuat produk: %w", err)
	}

	return product, nil
}

func (r *Repository) GetAll(ctx context.Context) ([]Product, error) {
	query := `
		SELECT id, name, sku, category, purchase_price,
		       selling_price, stock, is_active, created_at, updated_at
	FROM products
	WHERE is_active = TRUE
	ORDER BY id DESC
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil produk: %w", err)
	}
	defer rows.Close()

	products := make([]Product, 0)

	for rows.Next() {
		var product Product

		err := rows.Scan(
			&product.ID,
			&product.Name,
			&product.SKU,
			&product.Category,
			&product.PurchasePrice,
			&product.SellingPrice,
			&product.Stock,
			&product.IsActive,
			&product.CreatedAt,
			&product.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("gagal membaca data produk: %w", err)
		}

		products = append(products, product)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("gagal membaca hasil query: %w", err)
	}

	return products, nil
}

func (r *Repository) GetByID(ctx context.Context, id int64) (Product, error) {
	var product Product

	query := `
		SELECT id, name, sku, category, purchase_price,
		       selling_price, stock, is_active, created_at, updated_at
		FROM products
		WHERE id = $1
		  AND is_active = TRUE
	`

	err := r.db.QueryRow(ctx, query, id).Scan(
		&product.ID,
		&product.Name,
		&product.SKU,
		&product.Category,
		&product.PurchasePrice,
		&product.SellingPrice,
		&product.Stock,
		&product.IsActive,
		&product.CreatedAt,
		&product.UpdatedAt,
	)
	if err != nil {
		return Product{}, fmt.Errorf("gagal mengambil detail produk: %w", err)
	}

	return product, nil
}

func (r *Repository) Update(
	ctx context.Context,
	id int64,
	req UpdateProductRequest,
) (Product, error) {
	var product Product

	query := `
		UPDATE products
		SET name = $1,
		    sku = $2,
		    category = $3,
		    selling_price = $4,
		    is_active = $5,
		    updated_at = NOW()
		WHERE id = $6
		  AND is_active = TRUE
		RETURNING id, name, sku, category, purchase_price,
		          selling_price, stock, is_active, created_at, updated_at
	`

	err := r.db.QueryRow(
		ctx,
		query,
		req.Name,
		req.SKU,
		req.Category,
		req.SellingPrice,
		req.IsActive,
		id,
	).Scan(
		&product.ID,
		&product.Name,
		&product.SKU,
		&product.Category,
		&product.PurchasePrice,
		&product.SellingPrice,
		&product.Stock,
		&product.IsActive,
		&product.CreatedAt,
		&product.UpdatedAt,
	)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return Product{}, fmt.Errorf("SKU sudah digunakan: %w", err)
		}

		return Product{}, fmt.Errorf("gagal mengubah produk: %w", err)
	}

	return product, nil
}

func (r *Repository) SoftDelete(ctx context.Context, id int64) error {
	query := `
		UPDATE products
		SET is_active = FALSE,
		    updated_at = NOW()
		WHERE id = $1
		  AND is_active = TRUE
	`

	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("gagal menonaktifkan produk: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("produk tidak ditemukan atau sudah tidak aktif")
	}

	return nil
}
