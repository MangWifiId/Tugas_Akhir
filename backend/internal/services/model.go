package services

import "time"

type Service struct {
	ID            int64     `json:"id"`
	ServiceNumber string    `json:"service_number"`
	CustomerName  string    `json:"customer_name"`
	CustomerPhone *string   `json:"customer_phone"`
	VehiclePlate  string    `json:"vehicle_plate"`
	VehicleBrand  *string   `json:"vehicle_brand"`
	Complaint     string    `json:"complaint"`
	Diagnosis     *string   `json:"diagnosis"`
	MechanicName  *string   `json:"mechanic_name"`
	LaborCost     int64     `json:"labor_cost"`
	PartsCost     int64     `json:"parts_cost"`
	TotalCost     int64     `json:"total_cost"`
	Status        string    `json:"status"`
	PaymentStatus string    `json:"payment_status"`
	Notes         *string   `json:"notes"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type CreateServiceRequest struct {
	CustomerName  string  `json:"customer_name" binding:"required"`
	CustomerPhone *string `json:"customer_phone"`
	VehiclePlate  string  `json:"vehicle_plate" binding:"required"`
	VehicleBrand  *string `json:"vehicle_brand"`
	Complaint     string  `json:"complaint" binding:"required"`
	MechanicName  *string `json:"mechanic_name"`
	Notes         *string `json:"notes"`
}

type UpdateServiceRequest struct {
	Diagnosis     *string `json:"diagnosis"`
	MechanicName  *string `json:"mechanic_name"`
	LaborCost     int64   `json:"labor_cost" binding:"gte=0"`
	Status        string  `json:"status" binding:"required,oneof=pending in_progress completed cancelled"`
	PaymentStatus string  `json:"payment_status" binding:"required,oneof=unpaid paid"`
	Notes         *string `json:"notes"`
}

type ServicePart struct {
	ID        int64     `json:"id"`
	ServiceID int64     `json:"service_id"`
	ProductID int64     `json:"product_id"`
	Quantity  int       `json:"quantity"`
	UnitCost  int64     `json:"unit_cost"`
	UnitPrice int64     `json:"unit_price"`
	Subtotal  int64     `json:"subtotal"`
	CreatedAt time.Time `json:"created_at"`
}

type AddServicePartRequest struct {
	ProductID int64 `json:"product_id" binding:"required,gt=0"`
	Quantity  int   `json:"quantity" binding:"required,gt=0"`
}
