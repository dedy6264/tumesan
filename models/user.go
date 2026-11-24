package models

import "time"

type (
	OTPVerification struct {
		ID            int64     `json:"id"`
		PhoneNumber   string    `json:"phone_number"`
		OTPCodeHash   string    `json:"-"`
		ActionType    string    `json:"action_type"`
		Channel       string    `json:"channel"`
		AttemptsCount int       `json:"attempts_count"`
		IsConsumed    bool      `json:"is_consumed"`
		IsExpired     bool      `json:"is_expired"`
		ExpiresAt     time.Time `json:"expires_at"`
		CreatedAt     time.Time `json:"created_at"`
	}

	User struct {
		ID              int64     `json:"id"`
		IdentityID      *int64    `json:"identity_id,omitempty"`
		PriceGroupID    *int      `json:"price_group_id,omitempty"`
		Name            string    `json:"name"`
		PhoneNumber     string    `json:"phone_number"`
		PasswordHash    string    `json:"-"`
		Role            string    `json:"role"`
		StoreName       *string   `json:"store_name,omitempty"`
		StoreAddress    *string   `json:"store_address,omitempty"`
		CurrentDeviceID *string   `json:"current_device_id"`
		IsPhoneVerified bool      `json:"is_phone_verified"`
		CreatedAt       time.Time `json:"created_at"`
		Token           string    `json:"token,omitempty"`
	}

	// ---- Request ----
	ReqRequestOTP struct {
		ActionType  string `json:"action_type" validate:"required"`
		PhoneNumber string `json:"phone_number"`
		Password    string `json:"password"`
	}
	ReqRegisterWithOTP struct {
		Name        string `json:"name" validate:"required"`
		PhoneNumber string `json:"phone_number" validate:"required"`
		Password    string `json:"password" validate:"required,min=6"`
		StoreName   string `json:"store_name" validate:"required"`
		OTPCode     string `json:"otp_code" validate:"required"`
	}
	ReqLogin struct {
		PhoneNumber string `json:"phone_number" validate:"required"`
		Password    string `json:"password"`
		OTPCode     string `json:"otp_code"`
		DeviceID    string `json:"device_id" validate:"required"`
	}
	ReqDeviceBinding struct {
		PhoneNumber string `json:"phone_number" validate:"required"`
		Password    string `json:"password" validate:"required"`
		DeviceID    string `json:"device_id" validate:"required"`
		OTPCode     string `json:"otp_code" validate:"required"`
	}
)
