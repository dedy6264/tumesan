package routes

import (
	"github.com/labstack/echo/v4"
)

func AppRoutes(e *echo.Echo) {

	MobileApi(e)
	WebhookApi(e)

}
