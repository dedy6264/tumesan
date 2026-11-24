package otprepositories

import (
	"database/sql"
	"ppobcore/helpers"
	"ppobcore/models"
	"strconv"
)

// LockKey mengambil advisory lock transaksi per (phone, action) agar cooldown 60 detik
// tidak bisa ditembus oleh request paralel.
func LockKey(q helpers.Querier, phone, action string) error {
	_, err := q.Exec(`SELECT pg_advisory_xact_lock(hashtext($1))`, phone+"|"+action)
	return err
}

// CooldownRemaining mengembalikan sisa detik cooldown (0 = boleh request lagi).
func CooldownRemaining(q helpers.Querier, phone, action string, cooldownSecond int) (int, error) {
	var remaining int
	err := q.QueryRow(`SELECT COALESCE(MAX(CEIL(EXTRACT(EPOCH FROM (created_at + ($3 || ' seconds')::interval - NOW())))), 0)::int
		FROM otp_verifications
		WHERE phone_number = $1 AND action_type = $2 AND created_at > NOW() - ($3 || ' seconds')::interval`,
		phone, action, strconv.Itoa(cooldownSecond)).Scan(&remaining)
	if remaining < 0 {
		remaining = 0
	}
	return remaining, err
}

func InvalidateActive(q helpers.Querier, phone, action string) error {
	_, err := q.Exec(`UPDATE otp_verifications SET is_consumed = TRUE
		WHERE phone_number = $1 AND action_type = $2 AND is_consumed = FALSE`, phone, action)
	return err
}

func AddOtp(q helpers.Querier, phone, hash, action, channel string, ttlSecond int) (int64, error) {
	var id int64
	err := q.QueryRow(`INSERT INTO otp_verifications (phone_number, otp_code_hash, action_type, channel, expires_at)
		VALUES ($1, $2, $3, $4, NOW() + ($5 || ' seconds')::interval) RETURNING id`,
		phone, hash, action, channel, strconv.Itoa(ttlSecond)).Scan(&id)
	return id, err
}

// GetLatestActiveForUpdate mengunci OTP aktif terakhir. sql.ErrNoRows bila tidak ada.
func GetLatestActiveForUpdate(q helpers.Querier, phone, action string) (models.OTPVerification, error) {
	var o models.OTPVerification
	err := q.QueryRow(`SELECT id, phone_number, otp_code_hash, action_type, channel, attempts_count, is_consumed,
			expires_at < NOW(), expires_at, created_at
		FROM otp_verifications
		WHERE phone_number = $1 AND action_type = $2 AND is_consumed = FALSE
		ORDER BY created_at DESC, id DESC LIMIT 1 FOR UPDATE`, phone, action).Scan(
		&o.ID, &o.PhoneNumber, &o.OTPCodeHash, &o.ActionType, &o.Channel, &o.AttemptsCount, &o.IsConsumed,
		&o.IsExpired, &o.ExpiresAt, &o.CreatedAt)
	if err == sql.ErrNoRows {
		return o, sql.ErrNoRows
	}
	return o, err
}

func IncrementAttempts(q helpers.Querier, id int64) (int, error) {
	var n int
	err := q.QueryRow(`UPDATE otp_verifications SET attempts_count = attempts_count + 1 WHERE id = $1 RETURNING attempts_count`, id).Scan(&n)
	return n, err
}

func Consume(q helpers.Querier, id int64) error {
	_, err := q.Exec(`UPDATE otp_verifications SET is_consumed = TRUE WHERE id = $1`, id)
	return err
}
