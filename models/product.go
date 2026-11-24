package models

type (
	ProductCategory struct {
		ID        int    `json:"id"`
		Code      string `json:"code"`
		Name      string `json:"name"`
		IconURL   string `json:"icon_url"`
		UsePrefix bool   `json:"use_prefix"`
	}

	ProductBrand struct {
		ID      int    `json:"id"`
		Code    string `json:"code"`
		Name    string `json:"name"`
		LogoURL string `json:"logo_url"`
	}

	// ProductPricing = produk aktif + aturan harga untuk price group pengguna.
	// Spread bersifat rahasia (tidak pernah diserialisasi / dicatat ke transaksi).
	ProductPricing struct {
		ProductID           int     `json:"product_id"`
		SkuInternal         string  `json:"sku_internal"`
		ProviderSku         string  `json:"-"`
		Name                string  `json:"name"`
		ProductType         string  `json:"-"`
		NominalValue        float64 `json:"nominal_value"`
		CategoryCode        string  `json:"category_code"`
		BrandID             int     `json:"brand_id"`
		BrandCode           string  `json:"brand_code"`
		BrandName           string  `json:"brand_name"`
		PriceGroupID        int     `json:"price_group_id"`
		BaseHPP             float64 `json:"-"`
		ProviderAdminFee    float64 `json:"-"`
		ProviderMerchantFee float64 `json:"-"`
		Spread              float64 `json:"-"`
		ProductPrice        float64 `json:"product_price"`
		AdminFee            float64 `json:"admin_fee"`
		TotalAmount         float64 `json:"total_amount"`
		MerchantFee         float64 `json:"-"`
	}
)
