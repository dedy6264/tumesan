package utils

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"math/big"
	"net/http"
	"net/url"
	"ppobcore/configs"
	"ppobcore/helpers"
	"ppobcore/models"
	"regexp"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

func Response(rc, rm, rdt string, result any) models.GlobalResponse {
	return models.GlobalResponse{
		ResponseCode:     rc,
		ResponseMessage:  rm,
		ResponseDateTime: rdt,
		Result:           result,
	}
}

// Reply menulis envelope respon standar. rc berupa pasangan {kode, pesan} dari configs.
func Reply(ctx echo.Context, httpStatus int, rc []string, message string, result any) error {
	if message == "" {
		message = rc[1]
	}
	if result == nil {
		result = map[string]any{}
	}
	return ctx.JSON(httpStatus, Response(publicStatusCode(rc[0]), message, time.Now().Format(time.RFC3339), result))
}

func publicStatusCode(code string) string {
	switch code {
	case "00":
		return "00"
	case "10", "22", "59", "60", "61", "54":
		return "04"
	case "13", "58", "66":
		return "05"
	case "15", "17", "38", "62", "63", "65":
		return "02"
	case "39", "50", "51", "52", "53", "55", "57":
		return "03"
	case "01":
		return "99" // internal DB failure; "01" is reserved for pending transactions
	default:
		return "99"
	}
}

func ReplyOK(ctx echo.Context, message string, result any) error {
	return Reply(ctx, http.StatusOK, configs.RC_APP_SUCCESS, message, result)
}

// ReplyError memetakan error aplikasi (AppError) ke HTTP status + envelope.
func ReplyError(ctx echo.Context, svcname string, err error) error {
	if ae, ok := helpers.AsAppError(err); ok {
		return Reply(ctx, httpStatusFor(ae.RC), ae.RC, ae.Message, ae.Data)
	}
	Log("", svcname, err)
	return Reply(ctx, http.StatusInternalServerError, configs.RC_APP_SYSTEM_ERROR, "Terjadi gangguan pada sistem. Silakan coba beberapa saat lagi.", nil)
}

func httpStatusFor(rc []string) int {
	switch rc[0] {
	case configs.RC_APP_OTP_COOLDOWN[0]:
		return http.StatusTooManyRequests
	case configs.RC_APP_INVALID_SIGNATURE[0], configs.RC_APP_INVALID_CREDENTIAL[0], configs.RC_APP_SESSION_INVALID[0]:
		return http.StatusUnauthorized
	case configs.RC_APP_TRX_NOT_FOUND[0], configs.RC_APP_USER_NOT_FOUND[0], configs.RC_APP_PRODUCT_NOT_FOUND[0], configs.RC_APP_OTP_NOT_FOUND[0]:
		return http.StatusNotFound
	case configs.RC_APP_PHONE_REGISTERED[0]:
		return http.StatusConflict
	case configs.RC_APP_SYSTEM_ERROR[0], configs.RC_APP_DB_ERROR[0]:
		return http.StatusInternalServerError
	}
	return http.StatusUnprocessableEntity
}

func BindReq(e echo.Context, req any, svcname string) (err error) {
	if err = e.Bind(req); err != nil {
		Log("Bind", svcname, err)
		return helpers.NewAppError(configs.RC_APP_INVALID_PARAM, "Data permintaan tidak valid")
	}
	if err = e.Validate(req); err != nil {
		Log("Validate", svcname, err)
		return helpers.NewAppError(configs.RC_APP_INVALID_PARAM, "Data permintaan tidak lengkap atau tidak valid")
	}
	return nil
}

func Log(desc, svcname string, err error) {
	if err != nil {
		log.Println("[ERROR]", svcname, desc, err)
		return
	}
	log.Println("[INFO]", svcname, desc)
}

// RandomDigits menghasilkan n digit numerik memakai crypto/rand (tanpa modulo bias).
func RandomDigits(n int) (string, error) {
	out := make([]byte, n)
	max := big.NewInt(10)
	for i := range out {
		d, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		out[i] = byte('0' + d.Int64())
	}
	return string(out), nil
}

var nonDigit = regexp.MustCompile(`\D`)

// NormalizePhone: "+62812..", "62812.." -> "0812.."; hanya digit.
func NormalizePhone(p string) string {
	p = nonDigit.ReplaceAllString(p, "")
	switch {
	case strings.HasPrefix(p, "62"):
		p = "0" + p[2:]
	case strings.HasPrefix(p, "8"):
		p = "0" + p
	}
	return p
}

func Round2(v float64) float64 {
	return math.Round(v*100) / 100
}

func InArray(v string, in []string) bool {
	for _, s := range in {
		if s == v {
			return true
		}
	}
	return false
}

// SendWhatsAppMessage mengirim pesan OTP. WA_PROVIDER=log hanya mencetak ke log (development).
func SendWhatsAppMessage(to, message string) error {
	switch configs.WA_PROVIDER {
	case "fonnte":
		form := url.Values{"target": {to}, "message": {message}, "countryCode": {"62"}}
		req, _ := http.NewRequest(http.MethodPost, "https://api.fonnte.com/send", strings.NewReader(form.Encode()))
		req.Header.Set("Authorization", configs.WA_TOKEN)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		var r struct {
			Status bool   `json:"status"`
			Reason string `json:"reason"`
		}
		_ = json.Unmarshal(body, &r)
		if resp.StatusCode != http.StatusOK || !r.Status {
			return fmt.Errorf("fonnte: status=%d reason=%s", resp.StatusCode, r.Reason)
		}
		return nil
	default:
		log.Printf("[WA-LOG] to=%s msg=%q", to, message)
		return nil
	}
}
