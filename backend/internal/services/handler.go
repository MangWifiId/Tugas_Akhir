package services

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type Handler struct {
	repo *Repository
}

func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

// POST /api/services/
func (h *Handler) Create(c *gin.Context) {
	var req CreateServiceRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Data servis tidak valid",
		})
		return
	}

	req.CustomerName = strings.TrimSpace(req.CustomerName)
	req.VehiclePlate = strings.TrimSpace(req.VehiclePlate)
	req.Complaint = strings.TrimSpace(req.Complaint)

	if req.CustomerName == "" ||
		req.VehiclePlate == "" ||
		req.Complaint == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Nama pelanggan, nomor kendaraan, dan keluhan wajib diisi",
		})
		return
	}

	service, err := h.repo.Create(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal membuat data servis",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Data servis berhasil dibuat",
		"data":    service,
	})
}

// GET /api/services/
func (h *Handler) GetAll(c *gin.Context) {
	services, err := h.repo.GetAll(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal mengambil data servis",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": services,
	})
}

// GET /api/services/:id
func (h *Handler) GetByID(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "ID servis tidak valid",
		})
		return
	}

	service, err := h.repo.GetByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Data servis tidak ditemukan",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal mengambil detail servis",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": service,
	})
}

func (h *Handler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "ID servis tidak valid",
		})
		return
	}

	var req UpdateServiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Data pembaruan servis tidak valid",
		})
		return
	}

	// Servis harus selesai sebelum pembayaran dikonfirmasi.
	if req.PaymentStatus == "paid" && req.Status != "completed" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Servis harus selesai sebelum pembayaran dikonfirmasi",
		})
		return
	}

	service, err := h.repo.Update(c.Request.Context(), id, req)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Data servis tidak ditemukan",
			})
			return
		}

		message := err.Error()

		if strings.Contains(message, "tidak dapat diubah") ||
			strings.Contains(message, "harus selesai") {
			c.JSON(http.StatusConflict, gin.H{
				"error": message,
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal memperbarui data servis",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Data servis berhasil diperbarui",
		"data":    service,
	})
}

func (h *Handler) AddPart(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "ID servis tidak valid",
		})
		return
	}

	var req AddServicePartRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Data spare part tidak valid",
		})
		return
	}

	part, err := h.repo.AddPart(
		c.Request.Context(),
		id,
		req,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Servis atau produk tidak ditemukan",
			})
			return
		}

		message := err.Error()

		if strings.Contains(message, "stok tidak cukup") ||
			strings.Contains(message, "tidak dapat ditambahkan") {
			c.JSON(http.StatusConflict, gin.H{
				"error": message,
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal menambahkan spare part ke servis",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Spare part berhasil ditambahkan ke servis",
		"data":    part,
	})
}

func (h *Handler) CreatePayment(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "ID servis tidak valid",
		})
		return
	}

	var req CreatePaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Data pembayaran tidak valid",
		})
		return
	}

	payment, err := h.repo.CreatePayment(
		c.Request.Context(),
		id,
		req,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Servis tidak ditemukan",
			})
			return
		}

		message := err.Error()

		if strings.Contains(message, "harus selesai") ||
			strings.Contains(message, "sudah dibayar") ||
			strings.Contains(message, "total biaya servis") {
			c.JSON(http.StatusConflict, gin.H{
				"error": message,
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal memproses pembayaran servis",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Pembayaran servis berhasil dicatat",
		"data":    payment,
	})
}
