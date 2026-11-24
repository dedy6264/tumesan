package otpservices

import (
	"database/sql"
	"fmt"
	"ppobcore/configs"
	"ppobcore/helpers"
	otprepositories "ppobcore/repositories/otpRepositories"
	"ppobcore/utils"

	"golang.org/x/crypto/bcrypt"
)

const svcname = "OtpService"

// GenerateOTP membuat kode 6 digit (crypto/rand), menyimpan HASH bcrypt-nya, lalu mengirim via WhatsApp.
// Cooldown 60 detik per (phone, action) dijaga advisory lock + pengecekan created_at.
// Mengembalikan kode mentah (hanya untuk dipakai handler pada mode local/debug).
func GenerateOTP(db *sql.DB, phone, action string) (string, error) {
	rawOTP, err := utils.RandomDigits(configs.OTP_LENGTH)
	if err != nil {
		return "", err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(rawOTP), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	var otpID int64
	err = helpers.DBTransaction(db, func(tx *sql.Tx) error {
		if err := otprepositories.LockKey(tx, phone, action); err != nil {
			return err
		}
		remaining, err := otprepositories.CooldownRemaining(tx, phone, action, configs.OTP_COOLDOWN_SECOND)
		if err != nil {
			return err
		}
		if remaining > 0 {
			ae := helpers.NewAppError(configs.RC_APP_OTP_COOLDOWN,
				fmt.Sprintf("Silakan tunggu %d detik sebelum meminta OTP kembali", remaining))
			ae.Data = map[string]any{"retry_after": remaining}
			return ae
		}
		// Invalidate OTP sebelumnya yang belum terpakai untuk aksi yang sama
		if err := otprepositories.InvalidateActive(tx, phone, action); err != nil {
			return err
		}
		otpID, err = otprepositories.AddOtp(tx, phone, string(hash), action, "WHATSAPP", configs.OTP_TTL_SECOND)
		return err
	})
	if err != nil {
		return "", err
	}

	msg := fmt.Sprintf("Kode OTP Anda adalah: %s. Berlaku %d menit. Jangan bagikan kode ini kepada siapapun.", rawOTP, configs.OTP_TTL_SECOND/60)
	if err := utils.SendWhatsAppMessage(phone, msg); err != nil {
		utils.Log("SendWhatsAppMessage", svcname, err)
		_ = otprepositories.DeleteOtp(db, otpID)
		return "", helpers.NewAppError(configs.RC_APP_SYSTEM_ERROR, "Gagal mengirim OTP ke WhatsApp. Silakan coba lagi.")
	}
	return rawOTP, nil
}

// VerifyOTP memvalidasi kode secara atomik (row lock). OTP bersifat sekali pakai:
// berhasil -> consumed, kedaluwarsa / salah 3x -> consumed (wajib request ulang).
func VerifyOTP(db *sql.DB, phone, action, inputCode string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	otp, err := otprepositories.GetLatestActiveForUpdate(tx, phone, action)
	if err != nil {
		if err == sql.ErrNoRows {
			return helpers.NewAppError(configs.RC_APP_OTP_NOT_FOUND, "Kode OTP tidak ditemukan atau sudah tidak berlaku. Silakan minta kode baru")
		}
		return err
	}

	if otp.IsExpired {
		if err := otprepositories.Consume(tx, otp.ID); err != nil {
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
		return helpers.NewAppError(configs.RC_APP_OTP_EXPIRED, "Kode OTP telah kedaluwarsa. Silakan minta kode baru")
	}

	if otp.AttemptsCount >= configs.OTP_MAX_ATTEMPT {
		if err := otprepositories.Consume(tx, otp.ID); err != nil {
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
		return helpers.NewAppError(configs.RC_APP_OTP_ATTEMPT_LIMIT, "Batas percobaan OTP telah habis. Silakan minta kode baru")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(otp.OTPCodeHash), []byte(inputCode)); err != nil {
		attempts, err := otprepositories.IncrementAttempts(tx, otp.ID)
		if err != nil {
			return err
		}
		var aerr *helpers.AppError
		if attempts >= configs.OTP_MAX_ATTEMPT {
			if err := otprepositories.Consume(tx, otp.ID); err != nil {
				return err
			}
			aerr = helpers.NewAppError(configs.RC_APP_OTP_ATTEMPT_LIMIT, "Kode OTP salah. Batas percobaan habis, silakan minta kode baru")
		} else {
			aerr = helpers.NewAppError(configs.RC_APP_OTP_INVALID,
				fmt.Sprintf("Kode OTP salah. Sisa percobaan: %d", configs.OTP_MAX_ATTEMPT-attempts))
			aerr.Data = map[string]any{"attempts_left": configs.OTP_MAX_ATTEMPT - attempts}
		}
		// commit agar penambahan counter tidak ikut di-rollback
		if err := tx.Commit(); err != nil {
			return err
		}
		return aerr
	}

	if err := otprepositories.Consume(tx, otp.ID); err != nil {
		return err
	}
	return tx.Commit()
}

// CheckOTP validates a code for the standalone verify endpoint. The code remains
// available for the following register/login call, which consumes it atomically.
func CheckOTP(db *sql.DB, phone, action, inputCode string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	otp, err := otprepositories.GetLatestActiveForUpdate(tx, phone, action)
	if err != nil {
		if err == sql.ErrNoRows {
			return helpers.NewAppError(configs.RC_APP_OTP_NOT_FOUND, "Kode OTP tidak ditemukan atau sudah tidak berlaku")
		}
		return err
	}
	if otp.IsExpired || otp.AttemptsCount >= configs.OTP_MAX_ATTEMPT {
		if err = otprepositories.Consume(tx, otp.ID); err != nil {
			return err
		}
		if err = tx.Commit(); err != nil {
			return err
		}
		if otp.IsExpired {
			return helpers.NewAppError(configs.RC_APP_OTP_EXPIRED, "Kode OTP telah kedaluwarsa")
		}
		return helpers.NewAppError(configs.RC_APP_OTP_ATTEMPT_LIMIT, "Batas percobaan OTP telah habis")
	}
	if bcrypt.CompareHashAndPassword([]byte(otp.OTPCodeHash), []byte(inputCode)) != nil {
		attempts, updateErr := otprepositories.IncrementAttempts(tx, otp.ID)
		if updateErr != nil {
			return updateErr
		}
		if attempts >= configs.OTP_MAX_ATTEMPT {
			if updateErr = otprepositories.Consume(tx, otp.ID); updateErr != nil {
				return updateErr
			}
		}
		if updateErr = tx.Commit(); updateErr != nil {
			return updateErr
		}
		return helpers.NewAppError(configs.RC_APP_OTP_INVALID, "Kode OTP salah")
	}
	return tx.Commit()
}
