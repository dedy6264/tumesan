package middlewares

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"ppobcore/configs"
	"ppobcore/utils"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
)

// SignatureSigningString = timestamp \n METHOD \n request-uri(+query) \n body
func SignatureSigningString(timestamp, method, uri string, body []byte) string {
	return timestamp + "\n" + method + "\n" + uri + "\n" + string(body)
}

func Sign(secret, timestamp, method, uri string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(SignatureSigningString(timestamp, method, uri, body)))
	return hex.EncodeToString(mac.Sum(nil))
}

// HMACAuth memverifikasi request internal Laravel -> Core:
//
//	X-Timestamp : unix detik (toleransi SIGN_MAX_AGE_SECOND, anti replay)
//	X-Signature : hex(HMAC-SHA256(APP_KEY, timestamp\nMETHOD\nURI\nBODY))
func HMACAuth(next echo.HandlerFunc) echo.HandlerFunc {
	return func(ctx echo.Context) error {
		req := ctx.Request()
		deny := func() error {
			return utils.Reply(ctx, http.StatusUnauthorized, configs.RC_APP_INVALID_SIGNATURE, "", nil)
		}
		ts := req.Header.Get("X-Timestamp")
		sig := req.Header.Get("X-Signature")
		sec, err := strconv.ParseInt(ts, 10, 64)
		if err != nil || sig == "" {
			return deny()
		}
		age := time.Since(time.Unix(sec, 0))
		if age > configs.SIGN_MAX_AGE || age < -configs.SIGN_MAX_AGE {
			return deny()
		}
		var body []byte
		if req.Body != nil {
			body, err = io.ReadAll(req.Body)
			if err != nil {
				return deny()
			}
			req.Body = io.NopCloser(bytes.NewReader(body))
		}
		expected := Sign(configs.APP_KEY, ts, req.Method, req.URL.RequestURI(), body)
		if !hmac.Equal([]byte(expected), []byte(sig)) {
			return deny()
		}
		return next(ctx)
	}
}
