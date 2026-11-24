package userrepositories

import (
	"database/sql"
	"ppobcore/helpers"
	"ppobcore/models"
)

const userColumns = `id, identity_id, price_group_id, name, phone_number, COALESCE(password_hash, ''), role,
	store_name, store_address, current_device_id, is_phone_verified, created_at`

func scanUser(row *sql.Row) (models.User, error) {
	var u models.User
	err := row.Scan(&u.ID, &u.IdentityID, &u.PriceGroupID, &u.Name, &u.PhoneNumber, &u.PasswordHash, &u.Role,
		&u.StoreName, &u.StoreAddress, &u.CurrentDeviceID, &u.IsPhoneVerified, &u.CreatedAt)
	return u, err
}

func GetUserByPhone(q helpers.Querier, phone string) (models.User, error) {
	return scanUser(q.QueryRow(`SELECT `+userColumns+` FROM users WHERE phone_number = $1`, phone))
}

func GetUserByID(q helpers.Querier, id int64) (models.User, error) {
	return scanUser(q.QueryRow(`SELECT `+userColumns+` FROM users WHERE id = $1`, id))
}

func PhoneExists(q helpers.Querier, phone string) (bool, error) {
	var exists bool
	err := q.QueryRow(`SELECT EXISTS(SELECT 1 FROM users WHERE phone_number = $1)`, phone).Scan(&exists)
	return exists, err
}

// GetDefaultPriceGroupID mengambil price group bertanda is_default = true.
func GetDefaultPriceGroupID(q helpers.Querier) (int, error) {
	var id int
	err := q.QueryRow(`SELECT id FROM price_groups WHERE is_default = TRUE AND is_active = TRUE LIMIT 1`).Scan(&id)
	return id, err
}

func AddUser(q helpers.Querier, u models.User) (int64, error) {
	var id int64
	err := q.QueryRow(`INSERT INTO users (price_group_id, name, phone_number, password_hash, role, store_name, current_device_id, is_phone_verified)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id`,
		u.PriceGroupID, u.Name, u.PhoneNumber, u.PasswordHash, u.Role, u.StoreName, u.CurrentDeviceID, u.IsPhoneVerified).Scan(&id)
	return id, err
}

func UpdateDevice(q helpers.Querier, id int64, deviceID string) error {
	_, err := q.Exec(`UPDATE users SET current_device_id = $1, updated_at = NOW() WHERE id = $2`, deviceID, id)
	return err
}
