package models

import "time"

type (
	Transaction struct {
		ID                string    `json:"id"`
		ReferenceNo       string    `json:"reference_no"`
		UserID            *int64    `json:"user_id"`
		PriceGroupID      *int      `json:"price_group_id"`
		CustomerPhone     string    `json:"customer_phone"`
		DigitalProductID  *int      `json:"digital_product_id"`
		ProductName       string    `json:"product_name"`
		CategoryCode      string    `json:"category_code"`
		ProductType       string    `json:"-"`
		CustomerName      *string   `json:"customer_name"`
		ProviderSku       string    `json:"-"`
		CustomerID        string    `json:"customer_id"`
		BaseHPP           float64   `json:"-"`
		ProductPrice      float64   `json:"product_price"`
		AdminFee          float64   `json:"admin_fee"`
		TotalAmount       float64   `json:"total_amount"`
		MerchantMarkup    float64   `json:"merchant_markup"`
		GrandTotal        float64   `json:"grand_total"`
		StatusCode        string    `json:"status_code"`
		StatusDescription string    `json:"status_description"`
		SerialNumber      *string   `json:"serial_number"`
		CreatedAt         time.Time `json:"created_at"`
		UpdatedAt         time.Time `json:"updated_at"`
	}

	// ProviderResult = hasil terstandar dari Single Provider (Purchase / CheckStatus / Webhook).
	ProviderResult struct {
		State       string // 00=success, 01=pending, 02=failed
		Code        string // kode mentah provider (rc)
		Description string
		RefID       string
		SerialNo    string
	}

	ReqProviderWebhook struct {
		RefID   string `json:"ref_id"`
		Status  string `json:"status"`
		RC      string `json:"rc"`
		Message string `json:"message"`
		SN      string `json:"sn"`
		Data    *struct {
			RefID   string `json:"ref_id"`
			Status  string `json:"status"`
			RC      string `json:"rc"`
			Message string `json:"message"`
			SN      string `json:"sn"`
		} `json:"data"`
	}
	InquiryRequest struct {
		ProductID        int    `json:"product_id"`
		DigitalProductID int    `json:"digital_product_id"`
		CustomerID       string `json:"customer_id"`
		CustomerPhone    string `json:"customer_phone"`
	}
	ConfirmRequest struct {
		ReferenceNo    string  `json:"reference_no"`
		MerchantMarkup float64 `json:"merchant_markup"`
	}
)
