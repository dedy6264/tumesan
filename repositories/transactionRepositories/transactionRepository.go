package transactionrepositories

import (
	"fmt"
	"ppobcore/helpers"
	"ppobcore/models"
	"strconv"
)

const trxSelect = `SELECT t.id::text, t.reference_no, t.user_id, t.price_group_id, t.customer_phone, t.digital_product_id,
	COALESCE(dp.name, ''), COALESCE(pc.code, ''), COALESCE(dp.product_type, ''), t.customer_name, COALESCE(dp.provider_sku, ''), t.customer_id,
	t.base_hpp::float8, t.product_price::float8, t.admin_fee::float8, t.total_amount::float8,
	COALESCE(t.merchant_markup, 0)::float8, t.grand_total::float8,
	COALESCE(t.status_code, ''), COALESCE(t.status_description, ''),
	t.serial_number,
	t.created_at, t.updated_at
	FROM transactions t
	LEFT JOIN digital_products dp ON dp.id = t.digital_product_id
	LEFT JOIN product_categories pc ON pc.id = dp.category_id`

type rowScanner interface{ Scan(dest ...any) error }

func scanTrx(r rowScanner) (models.Transaction, error) {
	var t models.Transaction
	err := r.Scan(&t.ID, &t.ReferenceNo, &t.UserID, &t.PriceGroupID, &t.CustomerPhone, &t.DigitalProductID,
		&t.ProductName, &t.CategoryCode, &t.ProductType, &t.CustomerName, &t.ProviderSku, &t.CustomerID,
		&t.BaseHPP, &t.ProductPrice, &t.AdminFee, &t.TotalAmount,
		&t.MerchantMarkup, &t.GrandTotal,
		&t.StatusCode, &t.StatusDescription,
		&t.SerialNumber,
		&t.CreatedAt, &t.UpdatedAt)
	return t, err
}

func GenerateReferenceNo(q helpers.Querier) (string, error) {
	var ref string
	err := q.QueryRow(`SELECT 'TRX-' || to_char(NOW() AT TIME ZONE 'Asia/Jakarta', 'YYYYMMDD') || '-' || upper(replace(gen_random_uuid()::text, '-', ''))`).Scan(&ref)
	return ref, err
}

func CreateInquiry(q helpers.Querier, t models.Transaction, providerAdminFee, providerMerchantFee float64) (string, error) {
	var id string
	err := q.QueryRow(`INSERT INTO transactions (reference_no, user_id, price_group_id, customer_phone,
		digital_product_id, customer_id, base_hpp, provider_admin_fee, provider_merchant_fee,
		product_price, admin_fee, total_amount, merchant_markup, grand_total, status_code, status_description)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,'06','Inquiry berhasil, siap untuk pembayaran') RETURNING id::text`,
		t.ReferenceNo, t.UserID, t.PriceGroupID, t.CustomerPhone, t.DigitalProductID, t.CustomerID,
		t.BaseHPP, providerAdminFee, providerMerchantFee, t.ProductPrice, t.AdminFee, t.TotalAmount,
		t.MerchantMarkup, t.GrandTotal).Scan(&id)
	return id, err
}

func GetByReference(q helpers.Querier, invoice string) (models.Transaction, error) {
	return scanTrx(q.QueryRow(trxSelect+` WHERE t.reference_no = $1`, invoice))
}

// GetByReferenceForUpdate mengunci baris transaksi (serialisasi pay / webhook / refund).
func GetByReferenceForUpdate(q helpers.Querier, invoice string) (models.Transaction, error) {
	return scanTrx(q.QueryRow(trxSelect+` WHERE t.reference_no = $1 FOR UPDATE OF t`, invoice))
}

func ListByUser(q helpers.Querier, userID int64, limit int) ([]models.Transaction, error) {
	fmt.Println("Listing transactions for user:", userID)
	return list(q, trxSelect+` WHERE t.user_id = $1 and t.status_code not in ('06', '07') ORDER BY t.created_at DESC LIMIT $2`, userID, limit)
}

func ListByCustomerPhone(q helpers.Querier, phone string, limit int) ([]models.Transaction, error) {
	return list(q, trxSelect+` WHERE t.customer_phone = $1 and t.status_code not in ('06', '07') ORDER BY t.created_at DESC LIMIT $2`, phone, limit)
}

// ListStale: transaksi berstatus tertentu yang sudah tidak berubah >= olderThanSecond detik
// (HOLD_PENDING -> polling status, PROCESSING macet -> recovery dispatch).
func ListStale(q helpers.Querier, statusCode string, olderThanSecond, limit int) ([]models.Transaction, error) {
	return list(q, trxSelect+` WHERE t.status_code = $1 AND t.updated_at <= NOW() - ($2 || ' seconds')::interval ORDER BY t.updated_at LIMIT $3`,
		statusCode, strconv.Itoa(olderThanSecond), limit)
}

func list(q helpers.Querier, query string, args ...any) ([]models.Transaction, error) {
	rows, err := q.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []models.Transaction{}
	for rows.Next() {
		t, err := scanTrx(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, t)
	}
	return result, rows.Err()
}

func UpdateProviderStatus(q helpers.Querier, id, statusCode, statusDesc, provCode, provDesc, provRef, sn string) error {
	_, err := q.Exec(`UPDATE transactions SET status_code=$1, status_description=$2, provider_status_code=$3, provider_status_description=$4,
		provider_ref_id=COALESCE(NULLIF($5,''), provider_ref_id), serial_number=COALESCE(NULLIF($6,''), serial_number), updated_at=NOW() WHERE id=$7::uuid`,
		statusCode, statusDesc, provCode, provDesc, provRef, sn, id)
	return err
}
