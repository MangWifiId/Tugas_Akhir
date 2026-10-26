package purchases

import (
	"net/http"
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
	var req CreatePurchaseRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Format data pembelian tidak valid",
		})
		return
	}

	req.InvoiceNumber = strings.TrimSpace(req.InvoiceNumber)
	if req.InvoiceNumber == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Nomor faktur wajib diisi",
		})
		return
	}

	if len(req.Items) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Minimal satu item pembelian harus diisi",
		})
		return
	}

	purchase, err := h.repo.Create(c.Request.Context(), req)
	if err != nil {
		message := err.Error()

		switch {
		case strings.Contains(message, "sudah digunakan"):
			c.JSON(http.StatusConflict, gin.H{"error": message})
		case strings.Contains(message, "tidak ditemukan"),
			strings.Contains(message, "tidak aktif"),
			strings.Contains(message, "tidak mencukupi"),
			strings.Contains(message, "tidak valid"),
			strings.Contains(message, "tidak boleh"):
			c.JSON(http.StatusBadRequest, gin.H{"error": message})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Gagal menyimpan pembelian",
			})
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":  "Pembelian berhasil disimpan",
		"purchase": purchase,
	})
}
