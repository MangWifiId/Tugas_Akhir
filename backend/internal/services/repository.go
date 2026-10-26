package services

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// Create membuat data servis baru.
func (r *Repository) Create(
	ctx context.Context,
	req CreateServiceRequest,
) (Service, error) {
	var service Service

	service.ServiceNumber = fmt.Sprintf(
		"SVC-%s",
		time.Now().Format("20060102150405.000000"),
	)

	err := r.db.QueryRow(ctx, `
		INSERT INTO services (
			service_number,
			customer_name,
			customer_phone,
			vehicle_plate,
			vehicle_brand,
			complaint,
			mechanic_name,
			notes
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING
			id,
			service_number,
			customer_name,
			customer_phone,
			vehicle_plate,
			vehicle_brand,
			complaint,
			diagnosis,
			mechanic_name,
			labor_cost,
			parts_cost,
			total_cost,
			status,
			payment_status,
			notes,
			created_at,
			updated_at
	`,
		service.ServiceNumber,
		req.CustomerName,
		req.CustomerPhone,
		req.VehiclePlate,
		req.VehicleBrand,
		req.Complaint,
		req.MechanicName,
		req.Notes,
	).Scan(
		&service.ID,
		&service.ServiceNumber,
		&service.CustomerName,
		&service.CustomerPhone,
		&service.VehiclePlate,
		&service.VehicleBrand,
		&service.Complaint,
		&service.Diagnosis,
		&service.MechanicName,
		&service.LaborCost,
		&service.PartsCost,
		&service.TotalCost,
		&service.Status,
		&service.PaymentStatus,
		&service.Notes,
		&service.CreatedAt,
		&service.UpdatedAt,
	)
	if err != nil {
		return Service{}, err
	}

	return service, nil
}

// GetAll mengambil seluruh data servis.
func (r *Repository) GetAll(
	ctx context.Context,
) ([]Service, error) {
	rows, err := r.db.Query(ctx, `
		SELECT
			id,
			service_number,
			customer_name,
			customer_phone,
			vehicle_plate,
			vehicle_brand,
			complaint,
			diagnosis,
			mechanic_name,
			labor_cost,
			parts_cost,
			total_cost,
			status,
			payment_status,
			notes,
			created_at,
			updated_at
		FROM services
		ORDER BY created_at DESC, id DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]Service, 0)

	for rows.Next() {
		var service Service

		err := rows.Scan(
			&service.ID,
			&service.ServiceNumber,
			&service.CustomerName,
			&service.CustomerPhone,
			&service.VehiclePlate,
			&service.VehicleBrand,
			&service.Complaint,
			&service.Diagnosis,
			&service.MechanicName,
			&service.LaborCost,
			&service.PartsCost,
			&service.TotalCost,
			&service.Status,
			&service.PaymentStatus,
			&service.Notes,
			&service.CreatedAt,
			&service.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		result = append(result, service)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

// GetByID mengambil satu servis berdasarkan ID.
func (r *Repository) GetByID(
	ctx context.Context,
	id int64,
) (Service, error) {
	var service Service

	err := r.db.QueryRow(ctx, `
		SELECT
			id,
			service_number,
			customer_name,
			customer_phone,
			vehicle_plate,
			vehicle_brand,
			complaint,
			diagnosis,
			mechanic_name,
			labor_cost,
			parts_cost,
			total_cost,
			status,
			payment_status,
			notes,
			created_at,
			updated_at
		FROM services
		WHERE id = $1
	`, id).Scan(
		&service.ID,
		&service.ServiceNumber,
		&service.CustomerName,
		&service.CustomerPhone,
		&service.VehiclePlate,
		&service.VehicleBrand,
		&service.Complaint,
		&service.Diagnosis,
		&service.MechanicName,
		&service.LaborCost,
		&service.PartsCost,
		&service.TotalCost,
		&service.Status,
		&service.PaymentStatus,
		&service.Notes,
		&service.CreatedAt,
		&service.UpdatedAt,
	)
	if err != nil {
		return Service{}, err
	}

	return service, nil
}

// Update memperbarui informasi servis tanpa mencatat pembayaran langsung.
func (r *Repository) Update(
	ctx context.Context,
	id int64,
	req UpdateServiceRequest,
) (Service, error) {
	var service Service

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Service{}, err
	}
	defer tx.Rollback(ctx)

	var currentStatus string
	var currentPaymentStatus string

	err = tx.QueryRow(ctx, `
		SELECT status, payment_status
		FROM services
		WHERE id = $1
		FOR UPDATE
	`, id).Scan(&currentStatus, &currentPaymentStatus)
	if err != nil {
		return Service{}, err
	}

	if currentStatus == "cancelled" {
		return Service{}, fmt.Errorf(
			"servis yang dibatalkan tidak dapat diubah",
		)
	}

	if currentStatus == "completed" && req.Status != "completed" {
		return Service{}, fmt.Errorf(
			"servis yang sudah selesai tidak dapat dibuka kembali",
		)
	}

	if currentPaymentStatus == "paid" &&
		(req.Status != "completed" || req.PaymentStatus != "paid") {
		return Service{}, fmt.Errorf(
			"servis yang sudah dibayar tidak dapat diubah statusnya",
		)
	}

	// Pembayaran hanya boleh dicatat melalui CreatePayment.
	if currentPaymentStatus != "paid" && req.PaymentStatus == "paid" {
		return Service{}, fmt.Errorf(
			"gunakan endpoint pembayaran untuk mengonfirmasi pembayaran",
		)
	}

	err = tx.QueryRow(ctx, `
		UPDATE services
		SET
			diagnosis = $1,
			mechanic_name = $2,
			labor_cost = $3,
			total_cost = $3 + parts_cost,
			status = $4,
			payment_status = $5,
			notes = $6,
			updated_at = NOW()
		WHERE id = $7
		RETURNING
			id,
			service_number,
			customer_name,
			customer_phone,
			vehicle_plate,
			vehicle_brand,
			complaint,
			diagnosis,
			mechanic_name,
			labor_cost,
			parts_cost,
			total_cost,
			status,
			payment_status,
			notes,
			created_at,
			updated_at
	`,
		req.Diagnosis,
		req.MechanicName,
		req.LaborCost,
		req.Status,
		req.PaymentStatus,
		req.Notes,
		id,
	).Scan(
		&service.ID,
		&service.ServiceNumber,
		&service.CustomerName,
		&service.CustomerPhone,
		&service.VehiclePlate,
		&service.VehicleBrand,
		&service.Complaint,
		&service.Diagnosis,
		&service.MechanicName,
		&service.LaborCost,
		&service.PartsCost,
		&service.TotalCost,
		&service.Status,
		&service.PaymentStatus,
		&service.Notes,
		&service.CreatedAt,
		&service.UpdatedAt,
	)
	if err != nil {
		return Service{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Service{}, err
	}

	return service, nil
}

// AddPart menambahkan spare part ke servis dan mengurangi stok produk.
func (r *Repository) AddPart(
	ctx context.Context,
	serviceID int64,
	req AddServicePartRequest,
) (ServicePart, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return ServicePart{}, err
	}
	defer tx.Rollback(ctx)

	var serviceStatus string
	var paymentStatus string

	err = tx.QueryRow(ctx, `
		SELECT status, payment_status
		FROM services
		WHERE id = $1
		FOR UPDATE
	`, serviceID).Scan(&serviceStatus, &paymentStatus)
	if err != nil {
		return ServicePart{}, err
	}

	if serviceStatus == "completed" || serviceStatus == "cancelled" {
		return ServicePart{}, fmt.Errorf(
			"spare part tidak dapat ditambahkan pada servis %s",
			serviceStatus,
		)
	}

	if paymentStatus == "paid" {
		return ServicePart{}, fmt.Errorf(
			"spare part tidak dapat ditambahkan karena servis sudah dibayar",
		)
	}

	if req.ProductID <= 0 || req.Quantity <= 0 {
		return ServicePart{}, fmt.Errorf(
			"produk dan jumlah spare part harus valid",
		)
	}

	var purchasePrice int64
	var sellingPrice int64
	var stock int

	err = tx.QueryRow(ctx, `
		SELECT purchase_price, selling_price, stock
		FROM products
		WHERE id = $1 AND is_active = TRUE
		FOR UPDATE
	`, req.ProductID).Scan(
		&purchasePrice,
		&sellingPrice,
		&stock,
	)
	if err != nil {
		return ServicePart{}, err
	}

	if stock < req.Quantity {
		return ServicePart{}, fmt.Errorf(
			"stok tidak cukup: tersedia %d, diminta %d",
			stock,
			req.Quantity,
		)
	}

	stockAfter := stock - req.Quantity

	var part ServicePart

	err = tx.QueryRow(ctx, `
		INSERT INTO service_parts (
			service_id,
			product_id,
			quantity,
			unit_cost,
			unit_price
		)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING
			id,
			service_id,
			product_id,
			quantity,
			unit_cost,
			unit_price,
			subtotal,
			created_at
	`,
		serviceID,
		req.ProductID,
		req.Quantity,
		purchasePrice,
		sellingPrice,
	).Scan(
		&part.ID,
		&part.ServiceID,
		&part.ProductID,
		&part.Quantity,
		&part.UnitCost,
		&part.UnitPrice,
		&part.Subtotal,
		&part.CreatedAt,
	)
	if err != nil {
		return ServicePart{}, err
	}

	_, err = tx.Exec(ctx, `
		UPDATE products
		SET stock = $1, updated_at = NOW()
		WHERE id = $2
	`, stockAfter, req.ProductID)
	if err != nil {
		return ServicePart{}, err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO stock_movements (
			product_id,
			movement_type,
			quantity,
			stock_before,
			stock_after,
			notes
		)
		VALUES ($1, 'out', $2, $3, $4, $5)
	`,
		req.ProductID,
		req.Quantity,
		stock,
		stockAfter,
		fmt.Sprintf("Penggunaan spare part servis %d", serviceID),
	)
	if err != nil {
		return ServicePart{}, err
	}

	_, err = tx.Exec(ctx, `
		UPDATE services
		SET
			parts_cost = (
				SELECT COALESCE(SUM(subtotal), 0)
				FROM service_parts
				WHERE service_id = $1
			),
			total_cost = labor_cost + (
				SELECT COALESCE(SUM(subtotal), 0)
				FROM service_parts
				WHERE service_id = $1
			),
			updated_at = NOW()
		WHERE id = $1
	`, serviceID)
	if err != nil {
		return ServicePart{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return ServicePart{}, err
	}

	return part, nil
}

// CreatePayment mencatat pembayaran tunai untuk servis yang telah selesai.
func (r *Repository) CreatePayment(
	ctx context.Context,
	serviceID int64,
	req CreatePaymentRequest,
) (ServicePayment, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return ServicePayment{}, err
	}
	defer tx.Rollback(ctx)

	var status string
	var paymentStatus string
	var totalCost int64

	err = tx.QueryRow(ctx, `
		SELECT status, payment_status, total_cost
		FROM services
		WHERE id = $1
		FOR UPDATE
	`, serviceID).Scan(
		&status,
		&paymentStatus,
		&totalCost,
	)
	if err != nil {
		return ServicePayment{}, err
	}

	if status != "completed" {
		return ServicePayment{}, fmt.Errorf(
			"servis harus selesai sebelum pembayaran",
		)
	}

	if paymentStatus == "paid" {
		return ServicePayment{}, fmt.Errorf(
			"servis sudah dibayar",
		)
	}

	if totalCost <= 0 {
		return ServicePayment{}, fmt.Errorf(
			"total biaya servis harus lebih dari 0",
		)
	}

	var payment ServicePayment

	err = tx.QueryRow(ctx, `
		INSERT INTO service_payments (
			service_id,
			amount,
			payment_method,
			notes
		)
		VALUES ($1, $2, 'cash', $3)
		RETURNING
			id,
			service_id,
			amount,
			payment_method,
			paid_at,
			notes,
			created_at
	`,
		serviceID,
		totalCost,
		req.Notes,
	).Scan(
		&payment.ID,
		&payment.ServiceID,
		&payment.Amount,
		&payment.PaymentMethod,
		&payment.PaidAt,
		&payment.Notes,
		&payment.CreatedAt,
	)
	if err != nil {
		return ServicePayment{}, err
	}

	_, err = tx.Exec(ctx, `
		UPDATE services
		SET payment_status = 'paid',
		    updated_at = NOW()
		WHERE id = $1
	`, serviceID)
	if err != nil {
		return ServicePayment{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return ServicePayment{}, err
	}

	return payment, nil
}
