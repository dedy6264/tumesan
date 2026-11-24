package main

import (
	"context"
	"net/http"
	"ppobcore/configs"
	"ppobcore/connections"
	"ppobcore/middlewares"
	"ppobcore/routes"
	transactionservices "ppobcore/services/transactionServices"

	"github.com/go-playground/validator"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type (
	CustomValidator struct {
		validator *validator.Validate
	}
)

func main() {

	connections.Connect()
	defer connections.DBconn().Close()
	e := echo.New()
	e.HideBanner = true
	e.HTTPErrorHandler = func(err error, c echo.Context) {
		if c.Response().Committed {
			return
		}
		code := http.StatusInternalServerError
		if he, ok := err.(*echo.HTTPError); ok {
			code = he.Code
		}
		status := "99"
		desc := "Terjadi gangguan pada sistem"
		if code == http.StatusNotFound {
			status, desc = "04", "Endpoint atau data tidak ditemukan"
		}
		if code == http.StatusBadRequest || code == http.StatusUnprocessableEntity {
			status, desc = "03", "Parameter tidak valid / data tidak lengkap"
		}
		if code == http.StatusUnauthorized || code == http.StatusForbidden {
			status, desc = "05", "Otorisasi tidak valid"
		}
		_ = c.JSON(code, map[string]any{"status_code": status, "status_description": desc, "data": map[string]any{}})
	}
	e.Validator = &CustomValidator{validator: validator.New()}
	e.Use(middleware.Recover())
	e.Use(middlewares.RequestLogger)
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, "X-Timestamp", "X-Signature", "X-Callback-Token", "X-User-ID", "Authorization"},
		AllowMethods: []string{http.MethodGet, http.MethodHead, http.MethodPut, http.MethodPatch, http.MethodPost, http.MethodDelete},
	}))
	routes.AppRoutes(e)

	// Worker polling status provider (HOLD_PENDING) & pemulihan transaksi PROCESSING macet
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go transactionservices.StartPoller(ctx)

	e.Logger.Fatal(e.Start(":" + configs.APP_PORT))
}

func (cv *CustomValidator) Validate(i interface{}) error {
	if err := cv.validator.Struct(i); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	return nil
}
