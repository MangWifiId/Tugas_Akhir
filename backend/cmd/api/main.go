package main

import (
	"context"
	"log"
	"net/http"

	"Tugas_Akhir/backend/internal/auth"
	"Tugas_Akhir/backend/internal/config"
	"Tugas_Akhir/backend/internal/database"
	"Tugas_Akhir/backend/internal/products"
	"Tugas_Akhir/backend/internal/purchases"
	"Tugas_Akhir/backend/internal/sales"
	"Tugas_Akhir/backend/internal/services"
	"Tugas_Akhir/backend/internal/stock"

	"github.com/gin-gonic/gin"
)

func main() {
	// 1. Membaca konfigurasi aplikasi.
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	// 2. Menghubungkan aplikasi ke PostgreSQL.
	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// 3. Memastikan nama database yang digunakan backend.
	var dbName string
	err = db.QueryRow(
		context.Background(),
		"SELECT current_database()",
	).Scan(&dbName)

	if err != nil {
		log.Fatal("Gagal membaca nama database: ", err)
	}

	log.Println("Database aktif:", dbName)
	log.Println("Berhasil terhubung ke PostgreSQL!")

	// 4. Memeriksa migrasi database.
	if err := database.Migrate(db); err != nil {
		log.Fatal("Migration gagal: ", err)
	}

	log.Println("Migration berhasil diperiksa!")

	// 5. Membuat router Gin.
	router := gin.Default()

	router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"name":   "Tugas_Akhir API",
			"status": "running",
		})
	})

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":   "healthy",
			"database": "connected",
		})
	})

	// 6. Membuat grup API.
	api := router.Group("/api")

	// 7. Inisialisasi autentikasi.
	authRepo := auth.NewRepository(db)
	authHandler := auth.NewHandler(authRepo, cfg.JWTSecret)

	// Register dan login tetap publik.
	auth.RegisterRoutes(api.Group("/auth"), authHandler)

	// 8. Grup privat: seluruh route di bawahnya wajib memiliki JWT.
	private := api.Group("")
	private.Use(auth.AuthMiddleware(cfg.JWTSecret))

	// 9. Modul produk.
	productsRepo := products.NewRepository(db)
	productsHandler := products.NewHandler(productsRepo)
	products.RegisterRoutes(private.Group("/products"), productsHandler)

	// 10. Modul stok.
	stockRepo := stock.NewRepository(db)
	stockHandler := stock.NewHandler(stockRepo)
	stock.RegisterRoutes(private.Group("/stock"), stockHandler)

	// 11. Modul penjualan.
	salesRepo := sales.NewRepository(db)
	salesHandler := sales.NewHandler(salesRepo)
	sales.RegisterRoutes(private.Group("/sales"), salesHandler)

	// 12. Modul servis.
	servicesRepo := services.NewRepository(db)
	servicesHandler := services.NewHandler(servicesRepo)
	services.RegisterRoutes(private.Group("/services"), servicesHandler)
	// 13. Modul pembelian.
	purchasesRepo := purchases.NewRepository(db)
	purchasesHandler := purchases.NewHandler(purchasesRepo)

	purchaseRoutes := private.Group("/purchases")
	purchaseRoutes.Use(auth.RequireRoles("owner", "admin"))

	purchases.RegisterRoutes(purchaseRoutes, purchasesHandler)
	// 14. Menjalankan server.
	log.Println("Server berjalan pada port:", cfg.AppPort)

	if err := router.Run(":" + cfg.AppPort); err != nil {
		log.Fatal("Gagal menjalankan server: ", err)
	}
}
