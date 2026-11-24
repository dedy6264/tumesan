package configs

import (
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

func GetEnv(key, defaultValue string) string {
	// .env bersifat opsional (di container cukup environment variable)
	_ = godotenv.Load(".env")

	if os.Getenv(key) != "" {
		return os.Getenv(key)
	}
	return defaultValue
}

func GetEnvInt(key string, def int) int {
	v, err := strconv.Atoi(GetEnv(key, strconv.Itoa(def)))
	if err != nil {
		return def
	}
	return v
}

func GetEnvFloat(key string, def float64) float64 {
	v, err := strconv.ParseFloat(GetEnv(key, strconv.FormatFloat(def, 'f', -1, 64)), 64)
	if err != nil {
		return def
	}
	return v
}

var (
	APP_PORT = GetEnv("APP_PORT", "10001")
	APP_ENV  = GetEnv("APP_ENV", "local")
	// APP_KEY dipakai sebagai secret HMAC-SHA256 antara Laravel <-> Core
	APP_KEY        = GetEnv("APP_KEY", "change-me")
	JWT_SECRET     = GetEnv("JWT_SECRET", APP_KEY)
	JWT_TTL_SECOND = GetEnvInt("JWT_TTL_SECOND", 86400) // 24 jam
	DB_DRIVER      = GetEnv("DB_DRIVER", "postgres")
	DB_HOST      = GetEnv("DB_HOST", "localhost")
	DB_PORT      = GetEnv("DB_PORT", "5432")
	DB_NAME      = GetEnv("DB_NAME", "ppob")
	DB_USER      = GetEnv("DB_USER", "postgres")
	DB_PASS      = GetEnv("DB_PASS", "1234")
	SSL_MODE     = GetEnv("SSL_MODE", "disable")
	SIGN_MAX_AGE = time.Duration(GetEnvInt("SIGN_MAX_AGE_SECOND", 300)) * time.Second

	// OTP
	OTP_LENGTH          = 6
	OTP_TTL_SECOND      = GetEnvInt("OTP_TTL_SECOND", 300)     // 5 menit
	OTP_COOLDOWN_SECOND = GetEnvInt("OTP_COOLDOWN_SECOND", 60) // 60 detik
	OTP_MAX_ATTEMPT     = GetEnvInt("OTP_MAX_ATTEMPT", 3)      // maksimal salah 3x
	WA_PROVIDER         = GetEnv("WA_PROVIDER", "log")         // log | fonnte
	WA_TOKEN            = GetEnv("WA_TOKEN", "")
	// Transaksi
	MAX_MERCHANT_MARKUP = GetEnvFloat("MAX_MERCHANT_MARKUP", 500000)

	// Provider tunggal
	PROVIDER_MODE           = GetEnv("PROVIDER_MODE", "mock") // mock | digiflazz
	PROVIDER_BASE_URL       = GetEnv("PROVIDER_BASE_URL", "https://api.digiflazz.com/v1")
	PROVIDER_USERNAME       = GetEnv("PROVIDER_USERNAME", "")
	PROVIDER_API_KEY        = GetEnv("PROVIDER_API_KEY", "")
	PROVIDER_CALLBACK_TOKEN = GetEnv("PROVIDER_CALLBACK_TOKEN", "change-me")
	PROVIDER_TIMEOUT        = time.Duration(GetEnvInt("PROVIDER_TIMEOUT_SECOND", 20)) * time.Second
	POLL_INTERVAL           = time.Duration(GetEnvInt("POLL_INTERVAL_SECOND", 30)) * time.Second
)
