package otprepositories

import "ppobcore/helpers"

// DeleteOtp menghapus OTP yang gagal terkirim sehingga cooldown tidak menghukum pengguna.
func DeleteOtp(q helpers.Querier, id int64) error {
	_, err := q.Exec(`DELETE FROM otp_verifications WHERE id = $1`, id)
	return err
}
