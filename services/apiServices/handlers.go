package apiservices

import (
	"database/sql"
	"errors"
	"net/http"
	"ppobcore/configs"
	"ppobcore/connections"
	"ppobcore/helpers"
	"ppobcore/middlewares"
	"ppobcore/models"
	productrepositories "ppobcore/repositories/productRepositories"
	transactionrepositories "ppobcore/repositories/transactionRepositories"
	userrepositories "ppobcore/repositories/userRepositories"
	catalogservices "ppobcore/services/catalogServices"
	otpservices "ppobcore/services/otpServices"
	transactionservices "ppobcore/services/transactionServices"
	"ppobcore/utils"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

type envelope struct {
	StatusCode  string `json:"status_code"`
	Description string `json:"status_description"`
	Data        any    `json:"data"`
}

func reply(c echo.Context, code, description string, data any) error {
	httpCode := http.StatusOK
	switch code {
	case "03":
		httpCode = http.StatusBadRequest
	case "04":
		httpCode = http.StatusNotFound
	case "05":
		httpCode = http.StatusUnauthorized
	case "99":
		httpCode = http.StatusInternalServerError
	}
	return c.JSON(httpCode, envelope{StatusCode: code, Description: description, Data: data})
}
func userID(c echo.Context) int64 {
	return middlewares.AuthUserID(c)
}
func priceGroup(c echo.Context) (int, error) {
	id, _, err := catalogservices.ResolvePriceGroup(connections.DBconn(), userID(c))
	return id, err
}

type productsRequest struct {
	CategoryCode  string `json:"category_code"`
	BrandID       int    `json:"brand_id"`
	IsFastProduct *bool  `json:"is_fast_product"`
	SKU           string `json:"sku"`
	Search        string `json:"search"`
}

func Products(c echo.Context) error {
	var req productsRequest
	if err := c.Bind(&req); err != nil {
		return reply(c, "03", "Parameter tidak valid / data tidak lengkap", map[string]any{})
	}
	group, err := priceGroup(c)
	if err != nil {
		return utils.ReplyError(c, "Products", err)
	}
	items, err := productrepositories.GetCatalog(connections.DBconn(), group, strings.TrimSpace(req.CategoryCode), req.BrandID, req.IsFastProduct, strings.TrimSpace(req.SKU), strings.TrimSpace(req.Search))
	if err != nil {
		return utils.ReplyError(c, "Products", err)
	}
	return reply(c, "00", "Berhasil / Sukses", map[string]any{"products": items})
}

type prefixesRequest struct {
	CustomerID string `json:"customer_id"`
}

func Prefixes(c echo.Context) error {
	var req prefixesRequest
	if err := c.Bind(&req); err != nil || strings.TrimSpace(req.CustomerID) == "" {
		return reply(c, "03", "customer_id wajib diisi", map[string]any{})
	}
	brand, err := catalogservices.DetectBrand(connections.DBconn(), req.CustomerID)
	if err != nil {
		return reply(c, "04", "Prefix/operator tidak ditemukan", map[string]any{})
	}
	phone := utils.NormalizePhone(req.CustomerID)
	rows, err := connections.DBconn().Query(`SELECT prefix_number FROM brand_prefixes WHERE brand_id=$1 AND is_active=TRUE AND $2 LIKE prefix_number || '%' ORDER BY length(prefix_number) DESC`, brand.ID, phone)
	if err != nil {
		return utils.ReplyError(c, "Prefixes", err)
	}
	defer rows.Close()
	prefixes := []string{}
	for rows.Next() {
		var p string
		if err = rows.Scan(&p); err != nil {
			return utils.ReplyError(c, "Prefixes", err)
		}
		prefixes = append(prefixes, p)
	}
	return reply(c, "00", "Berhasil / Sukses", map[string]any{"customer_id": phone, "brand": brand, "prefixes": prefixes})
}

func Inquiry(c echo.Context) error {
	var req models.InquiryRequest
	if err := c.Bind(&req); err != nil || strings.TrimSpace(req.CustomerID) == "" { //validasi form data
		return reply(c, "03", "Parameter tidak valid / data tidak lengkap", map[string]any{})
	}
	if req.DigitalProductID < 1 { //validasi value
		return reply(c, "03", "digital_product_id wajib diisi", map[string]any{})
	}
	phone := utils.NormalizePhone(req.CustomerPhone) //formating no phone, default 0, bukan 62
	uid := userID(c)                                 //get userId dari token
	if phone == "" && uid == 0 {                     //validasi value
		return reply(c, "03", "customer_phone wajib diisi untuk transaksi guest", map[string]any{})
	}
	group, user, err := catalogservices.ResolvePriceGroup(connections.DBconn(), uid)
	if err != nil {
		return utils.ReplyError(c, "Inquiry", err)
	}
	p, err := productrepositories.GetPricedProduct(connections.DBconn(), group, req.DigitalProductID)
	if err != nil {
		if err == sql.ErrNoRows {
			utils.Log("inquiry denied: product unavailable", "Inquiry", sql.ErrNoRows)
			return reply(c, "07", "Inquiry gagal: produk tidak tersedia", map[string]any{"customer_id": req.CustomerID})
		}
		return utils.ReplyError(c, "Inquiry", err)
	}
	customer := strings.TrimSpace(req.CustomerID)
	if utils.InArray(p.CategoryCode, configs.PREFIX_CATEGORIES) {
		customer = utils.NormalizePhone(customer)
		b, e := catalogservices.DetectBrand(connections.DBconn(), customer)
		if e != nil || b.ID != p.BrandID {
			utils.Log("inquiry denied: invalid target/operator", "Inquiry", errors.New("target/operator validation failed"))
			return reply(c, "07", "Inquiry gagal: nomor tujuan/operator tidak valid", map[string]any{"customer_id": customer})
		}
	}
	if phone == "" && user != nil {
		phone = user.PhoneNumber
	}
	if phone == "" {
		phone = customer
	}
	ref, err := transactionrepositories.GenerateReferenceNo(connections.DBconn())
	if err != nil {
		return utils.ReplyError(c, "Inquiry", err)
	}
	var userPtr *int64
	if user != nil {
		userPtr = &user.ID
	}
	//lokasi hit ke provider, lalu simpan di transaksi
	pid, gid := p.ProductID, p.PriceGroupID
	t := models.Transaction{ReferenceNo: ref, UserID: userPtr, PriceGroupID: &gid, CustomerPhone: phone, DigitalProductID: &pid, CustomerID: customer, BaseHPP: p.BaseHPP, ProductPrice: p.ProductPrice, AdminFee: p.AdminFee, TotalAmount: p.TotalAmount, GrandTotal: p.TotalAmount, StatusCode: "06", StatusDescription: "Inquiry berhasil, siap untuk pembayaran"}
	if _, err = transactionrepositories.CreateInquiry(connections.DBconn(), t, p.ProviderAdminFee, p.ProviderMerchantFee); err != nil {
		return utils.ReplyError(c, "Inquiry", err)
	}
	allowMarkup := user != nil && user.Role == "MERCHANT" && p.ProductType != "POSTPAID"
	now := time.Now()
	return reply(c, "06", "Inquiry berhasil, siap untuk pembayaran", transactionData(ref, phone, customer, nil, p.Name, p.ProductPrice, p.AdminFee, p.TotalAmount, 0, p.TotalAmount, nil, allowMarkup, "06", "Inquiry berhasil, siap untuk pembayaran", now, now))
}

func ConfirmPayment(c echo.Context) error {
	var req models.ConfirmRequest
	if err := c.Bind(&req); err != nil || req.ReferenceNo == "" {
		return reply(c, "03", "reference_no wajib diisi", map[string]any{})
	}
	db := connections.DBconn()
	var trx models.Transaction
	err := helpers.DBTransaction(db, func(tx *sql.Tx) error {
		v, e := transactionrepositories.GetByReferenceForUpdate(tx, req.ReferenceNo)
		if e != nil {
			return e
		}
		trx = v
		uid := userID(c)
		if trx.UserID != nil {
			if uid == 0 || *trx.UserID != uid {
				return helpers.NewAppError(configs.RC_APP_SESSION_INVALID, "Otorisasi tidak valid: transaksi pengguna wajib menyertakan token yang sesuai")
			}
		} else if uid > 0 {
			return sql.ErrNoRows
		}
		if trx.StatusCode != "06" {
			return helpers.NewAppError(configs.RC_APP_INVALID_PARAM, "Transaksi tidak berada pada status inquiry")
		}
		if req.MerchantMarkup < 0 || req.MerchantMarkup > configs.MAX_MERCHANT_MARKUP {
			return helpers.NewAppError(configs.RC_APP_INVALID_PARAM, "Nilai markup tidak valid")
		}
		if trx.DigitalProductID == nil {
			return helpers.NewAppError(configs.RC_APP_INVALID_TRANSACTION, "Produk transaksi tidak tersedia")
		}
		var productType string
		if e = tx.QueryRow(`SELECT product_type FROM digital_products WHERE id=$1`, *trx.DigitalProductID).Scan(&productType); e != nil {
			return e
		}
		if trx.UserID == nil || productType == "POSTPAID" || trx.CategoryCode == "PLN_TAGIHAN" || trx.CategoryCode == "PLN_POSTPAID" {
			req.MerchantMarkup = 0
		} else {
			user, userErr := userrepositories.GetUserByID(tx, *trx.UserID)
			if userErr != nil {
				return userErr
			}
			if user.Role != "MERCHANT" {
				req.MerchantMarkup = 0
			}
		}
		//lokasi hit ke provider lalu simpan ke db
		markup := utils.Round2(req.MerchantMarkup)
		grand := utils.Round2(trx.TotalAmount + markup)
		_, e = tx.Exec(`UPDATE transactions SET merchant_markup=$1,grand_total=$2,status_code=$3,status_description='Sedang diproses oleh provider',updated_at=NOW() WHERE id=$4::uuid`, markup, grand, configs.STATUS_PROCESSING, trx.ID)
		trx.MerchantMarkup = markup
		trx.GrandTotal = grand
		trx.StatusCode = configs.STATUS_PROCESSING
		trx.StatusDescription = "Sedang diproses oleh provider"
		return e
	})
	if err != nil {
		if err == sql.ErrNoRows {
			return reply(c, "04", "Transaksi tidak ditemukan", map[string]any{})
		}
		return utils.ReplyError(c, "ConfirmPayment", err)
	}
	if err = transactionservices.DispatchToProvider(req.ReferenceNo); err != nil {
		utils.Log("provider dispatch; transaction retained for polling", "ConfirmPayment", err)
	}
	trx, err = transactionrepositories.GetByReference(db, req.ReferenceNo)
	if err != nil {
		return utils.ReplyError(c, "ConfirmPayment", err)
	}
	code := publicStatus(trx.StatusCode)
	return reply(c, code, trx.StatusDescription, transactionData(trx.ReferenceNo, trx.CustomerPhone, trx.CustomerID, trx.CustomerName, trx.ProductName, trx.ProductPrice, trx.AdminFee, trx.TotalAmount, trx.MerchantMarkup, trx.GrandTotal, trx.SerialNumber, false, code, trx.StatusDescription, trx.CreatedAt, trx.UpdatedAt))
}

func Status(c echo.Context) error {
	ref := c.Param("reference_no")
	trx, err := transactionrepositories.GetByReference(connections.DBconn(), ref)
	if err != nil {
		if err == sql.ErrNoRows {
			return reply(c, "04", "Transaksi tidak ditemukan", map[string]any{})
		}
		return utils.ReplyError(c, "TransactionStatus", err)
	}
	if uid := userID(c); uid > 0 && (trx.UserID == nil || *trx.UserID != uid) {
		return reply(c, "04", "Transaksi tidak ditemukan", map[string]any{})
	}
	code := publicStatus(trx.StatusCode)
	allowMarkup := code == "06" && trx.UserID != nil && trx.ProductType != "POSTPAID"
	return reply(c, code, trx.StatusDescription, transactionData(trx.ReferenceNo, trx.CustomerPhone, trx.CustomerID, trx.CustomerName, trx.ProductName, trx.ProductPrice, trx.AdminFee, trx.TotalAmount, trx.MerchantMarkup, trx.GrandTotal, trx.SerialNumber, allowMarkup, code, trx.StatusDescription, trx.CreatedAt, trx.UpdatedAt))
}
func publicStatus(code string) string {
	switch code {
	case "00":
		return "00"
	case "01", "03":
		return "01"
	case "02":
		return "02"
	case "06":
		return "06"
	case "07":
		return "07"
	default:
		return code
	}
}
func transactionData(ref, phone, customer string, customerName *string, product string, price, admin, total, markup, grand float64, serialNumber *string, allowMarkup bool, status, description string, created, updated time.Time) map[string]any {
	return map[string]any{"reference_no": ref, "customer_phone": phone, "customer_id": customer, "customer_name": customerName, "product_name": product, "product_price": price, "admin_fee": admin, "total_amount": total, "merchant_markup": markup, "grand_total": grand, "serial_number": serialNumber, "allow_markup": allowMarkup, "status_code": status, "status_description": description, "created_at": created, "updated_at": updated}
}

type historyRequest struct {
	CustomerPhone string `json:"customer_phone"`
	Limit         int    `json:"limit"`
}

func Transactions(c echo.Context) error {
	var req historyRequest
	if err := c.Bind(&req); err != nil {
		return reply(c, "03", "Parameter tidak valid", map[string]any{})
	}
	limit := req.Limit
	if limit < 1 || limit > 100 {
		limit = 30
	}
	var items []models.Transaction
	var err error
	if uid := userID(c); uid > 0 {
		items, err = transactionrepositories.ListByUser(connections.DBconn(), uid, limit)
	} else if phone := utils.NormalizePhone(req.CustomerPhone); phone != "" {
		items, err = transactionrepositories.ListByCustomerPhone(connections.DBconn(), phone, limit)
	} else {
		return reply(c, "03", "customer_phone wajib untuk melihat riwayat guest", map[string]any{})
	}
	if err != nil {
		return utils.ReplyError(c, "Transactions", err)
	}
	data := make([]map[string]any, 0, len(items))
	for _, t := range items {
		code := publicStatus(t.StatusCode)
		if code != "00" && code != "01" && code != "02" {
			continue
		}
		data = append(data, transactionData(t.ReferenceNo, t.CustomerPhone, t.CustomerID, t.CustomerName, t.ProductName, t.ProductPrice, t.AdminFee, t.TotalAmount, t.MerchantMarkup, t.GrandTotal, t.SerialNumber, false, code, t.StatusDescription, t.CreatedAt, t.UpdatedAt))
	}
	return reply(c, "00", "Berhasil / Sukses", map[string]any{"transactions": data})
}

// VerifyOTP is the public OTP verification endpoint used before register/login flows.
type verifyOTPRequest struct {
	PhoneNumber string `json:"phone_number"`
	ActionType  string `json:"action_type"`
	OTPCode     string `json:"otp_code"`
}

func VerifyOTP(c echo.Context) error {
	var req verifyOTPRequest
	if err := c.Bind(&req); err != nil || req.PhoneNumber == "" || req.OTPCode == "" {
		return reply(c, "03", "Parameter tidak valid", map[string]any{})
	}
	if err := otpservices.CheckOTP(connections.DBconn(), utils.NormalizePhone(req.PhoneNumber), strings.ToUpper(req.ActionType), req.OTPCode); err != nil {
		return utils.ReplyError(c, "VerifyOTP", err)
	}
	return reply(c, "00", "OTP berhasil diverifikasi", map[string]any{"verified": true})
}
