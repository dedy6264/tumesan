package routes

import (
	"log"
	"ppobcore/middlewares"
	apiservices "ppobcore/services/apiServices"
	authservices "ppobcore/services/authServices"
	catalogservices "ppobcore/services/catalogServices"
	transactionservices "ppobcore/services/transactionServices"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// MobileApi: endpoint internal untuk Laravel (ditandatangani HMAC).
func MobileApi(e *echo.Echo) {
	pt := e.Group("/api/v1")
	pt.Use(middlewares.HMACAuth)
	pt.Use(
		middleware.BodyDump(
			func(c echo.Context, reqBody, resBody []byte) {
				log.Println("+++")
				log.Println("[EndPoint]-[Method]-[Ip] :", c.Path(), "-", c.Request().Method, "-", c.Request().RemoteAddr)
				log.Println("[Header] :", c.Request().Header)
				log.Println("[Body] :", string(reqBody))
				log.Println("[Response] :", string(resBody))
				log.Println("---")
			},
		),
	)
	// OTP & autentikasi
	pt.POST("/auth/otp/request", authservices.RequestOTP)
	pt.POST("/auth/otp/verify", apiservices.VerifyOTP)
	// pt.POST("/auth/register", authservices.RegisterWithOTP) - ga kepake
	pt.POST("/auth/register-with-otp", authservices.RegisterWithOTP)
	pt.POST("/auth/login", authservices.Login)
	pt.POST("/auth/device-binding", authservices.DeviceBinding)
	pt.GET("/auth/session", authservices.Session)

	// Katalog & deteksi operator
	pt.POST("/products", apiservices.Products, middlewares.JWTAuthOptional)
	pt.POST("/prefixes", apiservices.Prefixes)
	pt.POST("/transaction/inquiry", apiservices.Inquiry, middlewares.JWTAuthOptional)
	pt.POST("/transaction/confirm-payment", apiservices.ConfirmPayment, middlewares.JWTAuthOptional)
	pt.GET("/transaction/status/:reference_no", apiservices.Status, middlewares.JWTAuthOptional)
	pt.POST("/transactions", apiservices.Transactions, middlewares.JWTAuthOptional)
	pt.GET("/catalog/categories", catalogservices.Categories)
	pt.GET("/catalog/brands", catalogservices.Brands)
	pt.GET("/catalog/detect", catalogservices.Detect)
	pt.GET("/catalog/products", catalogservices.Products)

}

// WebhookApi: dipanggil Payment Gateway / Provider (autentikasi via X-Callback-Token).
func WebhookApi(e *echo.Echo) {
	wh := e.Group("/webhooks")
	wh.POST("/provider", transactionservices.ProviderWebhook)
}
