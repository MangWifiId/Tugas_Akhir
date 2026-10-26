package stock

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	repo *Repository
}

func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) Create(c *gin.Context) {
	var req CreateStockMovementRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Format atau data request tidak valid",
		})
		return
	}

	if req.MovementType == "adjustment" &&
		req.AdjustmentDirection == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "adjustment_direction wajib diisi untuk koreksi stok",
		})
		return
	}

	if req.MovementType != "adjustment" &&
		req.AdjustmentDirection != "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "adjustment_direction hanya digunakan untuk koreksi stok",
		})
		return
	}

	if req.Notes != nil {
		notes := strings.TrimSpace(*req.Notes)
		req.Notes = &notes
	}

	movement, err := h.repo.Create(c.Request.Context(), req)
	if err != nil {
		message := err.Error()

		switch {
		case strings.Contains(message, "produk tidak ditemukan"):
			c.JSON(http.StatusNotFound, gin.H{"error": message})
		case strings.Contains(message, "produk tidak aktif"):
			c.JSON(http.StatusConflict, gin.H{"error": message})
		case strings.Contains(message, "stok tidak mencukupi"):
			c.JSON(http.StatusConflict, gin.H{"error": message})
		case strings.Contains(message, "arah koreksi"),
			strings.Contains(message, "jenis pergerakan"):
			c.JSON(http.StatusBadRequest, gin.H{"error": message})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Gagal memproses pergerakan stok",
			})
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Pergerakan stok berhasil dicatat",
		"data":    movement,
	})
}

func (h *Handler) GetMovements(c *gin.Context) {
	var productID int64

	idParam := c.Query("product_id")
	if idParam != "" {
		id, err := strconv.ParseInt(idParam, 10, 64)
		if err != nil || id <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "product_id harus berupa angka positif",
			})
			return
		}
		productID = id
	}

	movements, err := h.repo.GetMovements(
		c.Request.Context(),
		productID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal mengambil riwayat stok",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": movements,
	})
}
