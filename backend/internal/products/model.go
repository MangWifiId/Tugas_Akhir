package products

import "time"

type Product struct {
	ID            int64     `json:"id"`
	Name          string    `json:"name"`
	SKU           string    `json:"sku"`
	Category      *string   `json:"category"`
	PurchasePrice int64     `json:"purchase_price"`
	SellingPrice  int64     `json:"selling_price"`
	Stock         int       `json:"stock"`
	IsActive      bool      `json:"is_active"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type CreateProductRequest struct {
	Name          string  `json:"name" binding:"required"`
	SKU           string  `json:"sku" binding:"required"`
	Category      *string `json:"category"`
	PurchasePrice int64   `json:"purchase_price" binding:"gte=0"`
	SellingPrice  int64   `json:"selling_price" binding:"gte=0"`
}

type UpdateProductRequest struct {
	Name         string  `json:"name" binding:"required"`
	SKU          string  `json:"sku" binding:"required"`
	Category     *string `json:"category"`
	SellingPrice int64   `json:"selling_price" binding:"gte=0"`
	IsActive     bool    `json:"is_active"`
}
