package productrepositories

import (
	"ppobcore/helpers"
	"ppobcore/models"
	"ppobcore/utils"
	"strings"
)

const pricingSelect = `SELECT p.id, p.sku_internal, p.provider_sku, p.name, p.product_type, p.nominal_value::float8,
		c.code, b.id, b.code, b.name, r.price_group_id,
		p.base_cost::float8, p.provider_admin_fee::float8, p.provider_merchant_fee::float8,
		r.spread::float8, r.admin_fee::float8, r.merchant_fee::float8
	FROM digital_products p
	JOIN product_categories c ON c.id = p.category_id AND c.is_active = TRUE
	JOIN product_brands b ON b.id = p.brand_id AND b.is_active = TRUE
	JOIN pricing_rules r ON r.digital_product_id = p.id AND r.price_group_id = $1
	WHERE p.is_active = TRUE`

type rowScanner interface{ Scan(dest ...any) error }

func scanPricing(r rowScanner) (models.ProductPricing, error) {
	var p models.ProductPricing
	err := r.Scan(&p.ProductID, &p.SkuInternal, &p.ProviderSku, &p.Name, &p.ProductType, &p.NominalValue,
		&p.CategoryCode, &p.BrandID, &p.BrandCode, &p.BrandName, &p.PriceGroupID,
		&p.BaseHPP, &p.ProviderAdminFee, &p.ProviderMerchantFee, &p.Spread, &p.AdminFee, &p.MerchantFee)
	if err != nil {
		return p, err
	}
	applyFormula(&p)
	return p, nil
}

// applyFormula:
//
//	product_price = base_hpp + spread
//	total_amount  = product_price + admin_fee
func applyFormula(p *models.ProductPricing) {
	p.ProductPrice = utils.Round2(p.BaseHPP + p.Spread)
	p.TotalAmount = utils.Round2(p.ProductPrice + p.ProviderAdminFee + p.AdminFee)
}

// GetPricedProduct: produk aktif + kategori aktif + aturan harga untuk price group.
// sql.ErrNoRows bila produk nonaktif / tidak ada aturan harga untuk group tersebut.
func GetPricedProduct(q helpers.Querier, priceGroupID, productID int) (models.ProductPricing, error) {
	return scanPricing(q.QueryRow(pricingSelect+` AND p.id = $2`, priceGroupID, productID))
}

// GetPricedProducts: katalog varian denominasi, difilter kategori dan/atau brand.
func GetPricedProducts(q helpers.Querier, priceGroupID int, categoryCode string, brandCode string) ([]models.ProductPricing, error) {
	query := pricingSelect
	args := []any{priceGroupID}
	if categoryCode != "" {
		args = append(args, categoryCode)
		query += ` AND c.code = $` + itoa(len(args))
	}
	if brandCode != "" {
		args = append(args, brandCode)
		query += ` AND b.code = $` + itoa(len(args))
	}
	query += ` ORDER BY p.nominal_value, p.id`
	rows, err := q.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []models.ProductPricing{}
	for rows.Next() {
		p, err := scanPricing(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, p)
	}
	return result, rows.Err()
}

// GetCatalog returns only active products priced for the selected user segment.
func GetCatalog(q helpers.Querier, groupID int, categoryCode string, brandID int, fast *bool, sku, search string) ([]models.ProductPricing, error) {
	query := pricingSelect
	args := []any{groupID}
	add := func(condition string, value any) {
		args = append(args, value)
		query += " AND " + strings.ReplaceAll(condition, "?", "$"+itoa(len(args)))
	}
	if categoryCode != "" {
		add("c.code = ?", categoryCode)
	}
	if brandID > 0 {
		add("b.id = ?", brandID)
	}
	if fast != nil {
		add("p.is_fast_product = ?", *fast)
	}
	if sku != "" {
		add("p.sku_internal = ?", sku)
	}
	if search != "" {
		args = append(args, "%"+search+"%")
		query += " AND (p.name ILIKE $" + itoa(len(args)) + " OR p.sku_internal ILIKE $" + itoa(len(args)) + " OR b.name ILIKE $" + itoa(len(args)) + " OR c.name ILIKE $" + itoa(len(args)) + ")"
	}
	query += ` ORDER BY c.sort_order, b.sort_order, p.nominal_value, p.id`
	rows, err := q.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []models.ProductPricing{}
	for rows.Next() {
		p, err := scanPricing(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, p)
	}
	return result, rows.Err()
}

func GetCategories(q helpers.Querier) ([]models.ProductCategory, error) {
	rows, err := q.Query(`SELECT id, code, name, COALESCE(icon_url, '') FROM product_categories WHERE is_active = TRUE ORDER BY sort_order, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []models.ProductCategory{}
	for rows.Next() {
		var c models.ProductCategory
		if err = rows.Scan(&c.ID, &c.Code, &c.Name, &c.IconURL); err != nil {
			return nil, err
		}
		result = append(result, c)
	}
	return result, rows.Err()
}

// GetBrandsByCategory: brand aktif yang memiliki produk aktif pada kategori tsb.
func GetBrandsByCategory(q helpers.Querier, categoryCode string) ([]models.ProductBrand, error) {
	rows, err := q.Query(`SELECT b.id, b.code, b.name, COALESCE(b.logo_url, '')
		FROM product_brands b
		JOIN digital_products p ON p.brand_id = b.id AND p.is_active = TRUE
		JOIN product_categories c ON c.id = p.category_id AND c.is_active = TRUE AND c.code = $1
		WHERE b.is_active = TRUE
		GROUP BY b.id, b.code, b.name, b.logo_url
		ORDER BY MIN(b.sort_order), b.id`, categoryCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []models.ProductBrand{}
	for rows.Next() {
		var b models.ProductBrand
		if err = rows.Scan(&b.ID, &b.Code, &b.Name, &b.LogoURL); err != nil {
			return nil, err
		}
		result = append(result, b)
	}
	return result, rows.Err()
}

// ResolveBrandByPrefix mencocokkan 4 digit awal nomor ke product_prefixes.
// sql.ErrNoRows bila prefix tidak dikenal.
func ResolveBrandByPrefix(q helpers.Querier, prefix string) (models.ProductBrand, error) {
	var b models.ProductBrand
	err := q.QueryRow(`SELECT b.id, b.code, b.name, COALESCE(b.logo_url, '')
		FROM brand_prefixes px JOIN product_brands b ON b.id = px.brand_id
		WHERE $1 LIKE px.prefix_number || '%' AND px.is_active = TRUE AND b.is_active = TRUE
		ORDER BY length(px.prefix_number) DESC LIMIT 1`, prefix).
		Scan(&b.ID, &b.Code, &b.Name, &b.LogoURL)
	return b, err
}

func itoa(i int) string {
	const digits = "0123456789"
	if i < 10 {
		return string(digits[i])
	}
	return itoa(i/10) + string(digits[i%10])
}
