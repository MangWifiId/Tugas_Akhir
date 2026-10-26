package products

import (
	"net/http"
	"strings"

	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Handler struct {
	repo *Repository
}

func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) Create(c *gin.Context) {
	var req CreateProductRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Data produk tidak valid",
		})
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	req.SKU = strings.ToUpper(strings.TrimSpace(req.SKU))

	if req.Name == "" || req.SKU == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Nama dan SKU wajib diisi",
		})
		return
	}

	if req.PurchasePrice < 0 || req.SellingPrice < 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Harga tidak boleh negatif",
		})
		return
	}

	product, err := h.repo.Create(c.Request.Context(), req)
	if err != nil {
		var pgErr *pgconn.PgError

		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			c.JSON(http.StatusConflict, gin.H{
				"error": "SKU sudah digunakan",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal membuat produk",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Produk berhasil dibuat",
		"data":    product,
	})
}

func (h *Handler) GetAll(c *gin.Context) {
	products, err := h.repo.GetAll(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal mengambil daftar produk",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": products,
	})
}

func (h *Handler) GetByID(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "ID produk tidak valid",
		})
		return
	}

	product, err := h.repo.GetByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Produk tidak ditemukan",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal mengambil detail produk",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": product,
	})
}

func (h *Handler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "ID produk tidak valid",
		})
		return
	}

	var req UpdateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Data produk tidak valid",
		})
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	req.SKU = strings.ToUpper(strings.TrimSpace(req.SKU))

	if req.Name == "" || req.SKU == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Nama dan SKU wajib diisi",
		})
		return
	}
	if req.SellingPrice < 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Harga jual tidak boleh negatif",
		})
		return
	}

	product, err := h.repo.Update(c.Request.Context(), id, req)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			c.JSON(http.StatusConflict, gin.H{
				"error": "SKU sudah digunakan oleh produk lain",
			})
			return
		}

		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Produk tidak ditemukan",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal mengubah produk",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Produk berhasil diubah",
		"data":    product,
	})
}

func (h *Handler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "ID produk tidak valid",
		})
		return
	}

	err = h.repo.SoftDelete(c.Request.Context(), id)
	if err != nil {
		if strings.Contains(err.Error(), "tidak ditemukan") {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Produk tidak ditemukan atau sudah tidak aktif",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal menonaktifkan produk",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Produk berhasil dinonaktifkan",
	})
}
