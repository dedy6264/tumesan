package middlewares

import (
	"database/sql"
	"net/http"
	"ppobcore/configs"
	"ppobcore/connections"
	userrepositories "ppobcore/repositories/userRepositories"
	"ppobcore/utils"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
)

const CtxAuthUserIDKey = "auth_user_id"

// AuthUserID mengembalikan user_id yang telah terverifikasi JWT dari context (0 jika Guest).
func AuthUserID(ctx echo.Context) int64 {
	if v, ok := ctx.Get(CtxAuthUserIDKey).(int64); ok {
		return v
	}
	return 0
}

// JWTAuthOptional memvalidasi JWT secara kondisional (Hybrid Guest vs User Login):
//   - Jika request TIDAK membawa header Authorization dan TIDAK membawa X-User-ID (> 0),
//     request dianggap sebagai transaksi Guest (AuthUserID = 0).
//   - Jika request membawa header Authorization ATAU membawa X-User-ID (> 0),
//     maka JWT Bearer token WAJIB ada, valid, belum expired, cocok dengan X-User-ID (bila disertakan),
//     serta device_id di dalam token masih aktif di database (single-device lock).
func JWTAuthOptional(next echo.HandlerFunc) echo.HandlerFunc {
	return func(ctx echo.Context) error {
		req := ctx.Request()
		authHeader := strings.TrimSpace(req.Header.Get("Authorization"))
		xUserIDStr := strings.TrimSpace(req.Header.Get("X-User-ID"))

		isGuest := authHeader == "" && (xUserIDStr == "" || xUserIDStr == "0")
		if isGuest {
			return next(ctx)
		}

		deny := func(msg string) error {
			return utils.Reply(ctx, http.StatusUnauthorized, configs.RC_APP_SESSION_INVALID, msg, nil)
		}

		if len(authHeader) < 8 || !strings.EqualFold(authHeader[:7], "Bearer ") {
			return deny("Otorisasi tidak valid: token Bearer wajib disertakan")
		}
		rawToken := strings.TrimSpace(authHeader[7:])
		if rawToken == "" {
			return deny("Otorisasi tidak valid: token Bearer kosong")
		}

		claims, err := utils.VerifyJWT(rawToken)
		if err != nil {
			return deny("Otorisasi tidak valid: token tidak sah atau telah kedaluwarsa")
		}

		if xUserIDStr != "" && xUserIDStr != "0" {
			xUID, err := strconv.ParseInt(xUserIDStr, 10, 64)
			if err != nil || xUID != claims.UserID {
				return deny("Otorisasi tidak valid: identitas pengguna tidak sesuai token")
			}
		}

		user, err := userrepositories.GetUserByID(connections.DBconn(), claims.UserID)
		if err != nil {
			if err == sql.ErrNoRows {
				return deny("Otorisasi tidak valid: pengguna tidak ditemukan")
			}
			return utils.ReplyError(ctx, "JWTAuthOptional", err)
		}

		if user.CurrentDeviceID == nil || *user.CurrentDeviceID == "" || *user.CurrentDeviceID != claims.DeviceID {
			return deny("Akun Anda masuk di perangkat lain. Silakan login kembali")
		}

		ctx.Set(CtxAuthUserIDKey, user.ID)
		return next(ctx)
	}
}
