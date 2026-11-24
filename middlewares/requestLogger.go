package middlewares

import (
	"bytes"
	"io"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

var sensitiveFieldPattern = regexp.MustCompile(`(?i)"(pin|password|otp_code|otp|secret|token)"\s*:\s*"[^"]*"`)

// RequestLogger mencetak log untuk setiap HTTP request yang masuk ke Core.
func RequestLogger(next echo.HandlerFunc) echo.HandlerFunc {
	return func(ctx echo.Context) error {
		start := time.Now()
		req := ctx.Request()

		var bodyBytes []byte
		if req.Body != nil {
			bodyBytes, _ = io.ReadAll(req.Body)
			req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		}

		err := next(ctx)
		if err != nil {
			ctx.Error(err)
		}

		res := ctx.Response()
		latency := time.Since(start).Round(time.Microsecond)
		bodyLog := sanitizeBody(bodyBytes)

		if bodyLog != "" {
			log.Printf("[REQUEST] %s %s | status=%d latency=%s ip=%s body=%s",
				req.Method, req.URL.RequestURI(), res.Status, latency, ctx.RealIP(), bodyLog)

		} else {
			log.Printf("[REQUEST]= %s %s | status=%d latency=%s ip=%s",
				req.Method, req.URL.RequestURI(), res.Status, latency, ctx.RealIP())
		}

		return nil
	}
}

func sanitizeBody(raw []byte) string {
	s := strings.TrimSpace(string(raw))
	if s == "" {
		return ""
	}
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", "")
	s = sensitiveFieldPattern.ReplaceAllString(s, `"$1":"***"`)
	if len(s) > 1000 {
		s = s[:1000] + "...(truncated)"
	}
	return s
}
