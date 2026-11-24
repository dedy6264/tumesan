package catalogservices

import (
	"database/sql"
	"ppobcore/configs"
	"ppobcore/connections"
	"ppobcore/helpers"
	"ppobcore/models"
	productrepositories "ppobcore/repositories/productRepositories"
	userrepositories "ppobcore/repositories/userRepositories"
	"ppobcore/utils"
	"strconv"

	"github.com/labstack/echo/v4"
)

// ResolvePriceGroup: price group milik user; fallback ke group is_default untuk guest / user tanpa group.
func ResolvePriceGroup(q helpers.Querier, userID int64) (groupID int, user *models.User, err error) {
	if userID != 0 {
		u, err := userrepositories.GetUserByID(q, userID)
		if err != nil {
			if err == sql.ErrNoRows {
				return 0, nil, helpers.NewAppError(configs.RC_APP_USER_NOT_FOUND, "Pengguna tidak ditemukan")
			}
			return 0, nil, err
		}
		user = &u
		if u.PriceGroupID != nil {
			return *u.PriceGroupID, user, nil
		}
	}
	groupID, err = userrepositories.GetDefaultPriceGroupID(q)
	return groupID, user, err
}

// DetectBrand: deteksi operator dari 4 digit pertama customer_id via tabel product_prefixes.
func DetectBrand(q helpers.Querier, customerID string) (models.ProductBrand, error) {
	phone := utils.NormalizePhone(customerID)
	if len(phone) < 4 {
		return models.ProductBrand{}, helpers.NewAppError(configs.RC_APP_PREFIX_NOT_FOUND, "Masukkan minimal 4 digit nomor tujuan")
	}
	brand, err := productrepositories.ResolveBrandByPrefix(q, phone)
	if err != nil {
		if err == sql.ErrNoRows {
			return brand, helpers.NewAppError(configs.RC_APP_PREFIX_NOT_FOUND, "Operator tidak dikenali dari nomor tujuan")
		}
		return brand, err
	}
	return brand, nil
}

// Categories : GET /catalog/categories
func Categories(ctx echo.Context) error {
	svcname := "Categories"
	data, err := productrepositories.GetCategories(connections.DBconn())
	if err != nil {
		return utils.ReplyError(ctx, svcname, err)
	}
	for i := range data {
		data[i].UsePrefix = utils.InArray(data[i].Code, configs.PREFIX_CATEGORIES)
	}
	return utils.ReplyOK(ctx, "", data)
}

// Brands : GET /catalog/brands?category_code=
func Brands(ctx echo.Context) error {
	svcname := "Brands"
	data, err := productrepositories.GetBrandsByCategory(connections.DBconn(), ctx.QueryParam("category_code"))
	if err != nil {
		return utils.ReplyError(ctx, svcname, err)
	}
	return utils.ReplyOK(ctx, "", data)
}

// Detect : GET /catalog/detect?customer_id=
func Detect(ctx echo.Context) error {
	svcname := "Detect"
	brand, err := DetectBrand(connections.DBconn(), ctx.QueryParam("customer_id"))
	if err != nil {
		return utils.ReplyError(ctx, svcname, err)
	}
	return utils.ReplyOK(ctx, "", brand)
}

// Products : GET /catalog/products?category_code=&customer_id=|brand_code=&user_id=
// Untuk kategori berbasis prefix (PULSA/DATA), brand dideteksi dari customer_id.
func Products(ctx echo.Context) error {
	var (
		svcname      = "Products"
		db           = connections.DBconn()
		categoryCode = ctx.QueryParam("category_code")
		brandCode    = ctx.QueryParam("brand_code")
		customerID   = ctx.QueryParam("customer_id")
		userID, _    = strconv.ParseInt(ctx.QueryParam("user_id"), 10, 64)
		brand        *models.ProductBrand
	)
	if utils.InArray(categoryCode, configs.PREFIX_CATEGORIES) {
		b, err := DetectBrand(db, customerID)
		if err != nil {
			return utils.ReplyError(ctx, svcname, err)
		}
		brand, brandCode = &b, b.Code
	}
	groupID, _, err := ResolvePriceGroup(db, userID)
	if err != nil {
		return utils.ReplyError(ctx, svcname, err)
	}
	products, err := productrepositories.GetPricedProducts(db, groupID, categoryCode, brandCode)
	if err != nil {
		return utils.ReplyError(ctx, svcname, err)
	}
	return utils.ReplyOK(ctx, "", map[string]any{"brand": brand, "products": products})
}
