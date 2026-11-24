package authservices

import (
	"database/sql"
	"fmt"
	"net/http"
	"ppobcore/configs"
	"ppobcore/connections"
	"ppobcore/helpers"
	"ppobcore/models"
	userrepositories "ppobcore/repositories/userRepositories"
	otpservices "ppobcore/services/otpServices"
	"ppobcore/utils"
	"regexp"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

var phoneRegex = regexp.MustCompile(`^08\d{8,12}$`)

func maskPhone(p string) string {
	if len(p) < 7 {
		return p
	}
	return p[:4] + "****" + p[len(p)-3:]
}

func checkPassword(db *sql.DB, phone, password string) (models.User, error) {
	user, err := userrepositories.GetUserByPhone(db, phone)
	if err != nil {
		fmt.Println("Err::", err)
		if err == sql.ErrNoRows {
			fmt.Println("NOROWS::")
			// samakan perilaku agar tidak membocorkan keberadaan akun
			_ = bcrypt.CompareHashAndPassword([]byte("$2a$10$7EqJtq98hPqEX7fNZaFWoOhi5BcrqPmDkYQ1e/u7hS7wD3v5U5b6W"), []byte(password))
			return user, helpers.NewAppError(configs.RC_APP_INVALID_CREDENTIAL, "Nomor HP atau kata sandi salah")
		}
		return user, err
	}
	fmt.Println("HashPassword::", []byte(user.PasswordHash))
	fmt.Println("HashPasswordInput::", []byte(password))
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		return user, helpers.NewAppError(configs.RC_APP_INVALID_CREDENTIAL, "Nomor HP atau kata sandi salah")
	}
	return user, nil
}

// RequestOTP : POST /auth/otp/request
// - REGISTER        : nomor belum terdaftar
// - DEVICE_BINDING  : wajib kredensial valid (cegah spam OTP ke nomor orang lain)
func RequestOTP(ctx echo.Context) error {
	var (
		svcname = "RequestOTP"
		req     models.ReqRequestOTP
		db      = connections.DBconn()
	)
	if err := utils.BindReq(ctx, &req, svcname); err != nil {
		return utils.ReplyError(ctx, svcname, err)
	}

	var phone string
	switch req.ActionType {
	case configs.OTP_ACTION_REGISTER: //register, cek nomer hp pada user, apakah sudah terdaftar
		phone = utils.NormalizePhone(req.PhoneNumber)
		if !phoneRegex.MatchString(phone) {
			return utils.ReplyError(ctx, svcname, helpers.NewAppError(configs.RC_APP_INVALID_PARAM, "Format nomor WhatsApp tidak valid"))
		}
		exists, err := userrepositories.PhoneExists(db, phone)
		if err != nil {
			return utils.ReplyError(ctx, svcname, err)
		}
		if exists {
			return utils.ReplyError(ctx, svcname, helpers.NewAppError(configs.RC_APP_PHONE_REGISTERED, "Nomor sudah terdaftar"))
		}
	case configs.OTP_ACTION_LOGIN: //login, cek nomer hp pada user, apakah sudah terdaftar
		phone = utils.NormalizePhone(req.PhoneNumber)
		if !phoneRegex.MatchString(phone) {
			return utils.ReplyError(ctx, svcname, helpers.NewAppError(configs.RC_APP_INVALID_PARAM, "Format nomor WhatsApp tidak valid"))
		}
		exists, err := userrepositories.PhoneExists(db, phone)
		if err != nil {
			return utils.ReplyError(ctx, svcname, err)
		}
		if !exists {
			return utils.ReplyError(ctx, svcname, helpers.NewAppError(configs.RC_APP_INVALID_CREDENTIAL, "Nomor HP tidak terdaftar"))
		}
	case configs.OTP_ACTION_DEVICE_BINDING: //device binding, cek nomor hp / kredensial
		phone = utils.NormalizePhone(req.PhoneNumber)
		if !phoneRegex.MatchString(phone) {
			return utils.ReplyError(ctx, svcname, helpers.NewAppError(configs.RC_APP_INVALID_PARAM, "Format nomor WhatsApp tidak valid"))
		}
		if req.Password != "" {
			if _, err := checkPassword(db, phone, req.Password); err != nil {
				return utils.ReplyError(ctx, svcname, err)
			}
		} else {
			exists, err := userrepositories.PhoneExists(db, phone)
			if err != nil {
				return utils.ReplyError(ctx, svcname, err)
			}
			if !exists {
				return utils.ReplyError(ctx, svcname, helpers.NewAppError(configs.RC_APP_INVALID_CREDENTIAL, "Nomor HP tidak terdaftar"))
			}
		}
	default:
		return utils.ReplyError(ctx, svcname, helpers.NewAppError(configs.RC_APP_INVALID_PARAM, "action_type tidak dikenal"))
	}

	rawOTP, err := otpservices.GenerateOTP(db, phone, req.ActionType)
	if err != nil {
		return utils.ReplyError(ctx, svcname, err)
	}
	result := map[string]any{
		"phone_masked":     maskPhone(phone),
		"expires_in":       configs.OTP_TTL_SECOND,
		"cooldown_seconds": configs.OTP_COOLDOWN_SECOND,
	}
	if configs.APP_ENV == "local" && configs.WA_PROVIDER == "log" {
		result["debug_otp"] = rawOTP // hanya untuk development
	}
	return utils.ReplyOK(ctx, "Kode OTP berhasil dikirim ke nomor WhatsApp Anda", result)
}

// RegisterWithOTP : POST /auth/register-with-otp
// Akun baru HANYA dibuat di tabel users setelah OTP valid.
func RegisterWithOTP(ctx echo.Context) error {
	var (
		svcname = "RegisterWithOTP"
		req     models.ReqRegisterWithOTP
		db      = connections.DBconn()
	)
	if err := utils.BindReq(ctx, &req, svcname); err != nil {
		return utils.ReplyError(ctx, svcname, err)
	}
	phone := utils.NormalizePhone(req.PhoneNumber)
	if !phoneRegex.MatchString(phone) {
		return utils.ReplyError(ctx, svcname, helpers.NewAppError(configs.RC_APP_INVALID_PARAM, "Format nomor WhatsApp tidak valid"))
	}
	if exists, err := userrepositories.PhoneExists(db, phone); err != nil {
		return utils.ReplyError(ctx, svcname, err)
	} else if exists {
		return utils.ReplyError(ctx, svcname, helpers.NewAppError(configs.RC_APP_PHONE_REGISTERED, "Nomor sudah terdaftar"))
	}

	if err := otpservices.VerifyOTP(db, phone, configs.OTP_ACTION_REGISTER, req.OTPCode); err != nil {
		return utils.ReplyError(ctx, svcname, err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return utils.ReplyError(ctx, svcname, err)
	}
	groupID, err := userrepositories.GetDefaultPriceGroupID(db)
	if err != nil {
		return utils.ReplyError(ctx, svcname, err)
	}
	store := req.StoreName
	id, err := userrepositories.AddUser(db, models.User{
		PriceGroupID:    &groupID,
		Name:            req.Name,
		PhoneNumber:     phone,
		PasswordHash:    string(hash),
		Role:            "MERCHANT",
		StoreName:       &store,
		IsPhoneVerified: true,
	})
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return utils.ReplyError(ctx, svcname, helpers.NewAppError(configs.RC_APP_PHONE_REGISTERED, "Nomor sudah terdaftar"))
		}
		return utils.ReplyError(ctx, svcname, err)
	}
	user, err := userrepositories.GetUserByID(db, id)
	if err != nil {
		return utils.ReplyError(ctx, svcname, err)
	}
	return utils.Reply(ctx, http.StatusCreated, configs.RC_APP_SUCCESS, "Pendaftaran berhasil", user)
}

// Login : POST /auth/login
// Perangkat pertama otomatis terikat. Perangkat berbeda -> RC 56 (wajib OTP DEVICE_BINDING).
func Login(ctx echo.Context) error {
	var (
		svcname = "Login"
		req     models.ReqLogin
		db      = connections.DBconn()
	)
	if err := utils.BindReq(ctx, &req, svcname); err != nil {
		return utils.ReplyError(ctx, svcname, err)
	}
	phone := utils.NormalizePhone(req.PhoneNumber)
	var user models.User
	var err error
	if req.OTPCode != "" { //jika membawa otp
		if err := otpservices.VerifyOTP(db, phone, configs.OTP_ACTION_LOGIN, req.OTPCode); err != nil {

			return utils.ReplyError(ctx, svcname, err)
		}
		user, err = userrepositories.GetUserByPhone(db, phone)
	} else if req.Password != "" {
		user, err = checkPassword(db, phone, req.Password)
	} else {
		err = helpers.NewAppError(configs.RC_APP_INVALID_PARAM, "Password atau OTP wajib diisi")
	}
	if err != nil {
		return utils.ReplyError(ctx, svcname, err)
	}
	if user.CurrentDeviceID == nil || *user.CurrentDeviceID == "" {
		if err := userrepositories.UpdateDevice(db, user.ID, req.DeviceID); err != nil {
			return utils.ReplyError(ctx, svcname, err)
		}
		user.CurrentDeviceID = &req.DeviceID
	} else if *user.CurrentDeviceID != req.DeviceID {
		ae := helpers.NewAppError(configs.RC_APP_DEVICE_OTP, "Login dari perangkat baru terdeteksi. Verifikasi OTP diperlukan")
		ae.Data = map[string]any{"requires_otp": true, "action_type": configs.OTP_ACTION_DEVICE_BINDING}
		return utils.ReplyError(ctx, svcname, ae)
	}
	token, err := utils.GenerateJWT(user)
	if err != nil {
		return utils.ReplyError(ctx, svcname, err)
	}
	user.Token = token
	return utils.ReplyOK(ctx, "Login berhasil", user)
}

// DeviceBinding : POST /auth/device-binding
// Setelah OTP valid, current_device_id diganti sehingga sesi perangkat lama otomatis tidak valid.
func DeviceBinding(ctx echo.Context) error {
	var (
		svcname = "DeviceBinding"
		req     models.ReqDeviceBinding
		db      = connections.DBconn()
	)
	if err := utils.BindReq(ctx, &req, svcname); err != nil {
		return utils.ReplyError(ctx, svcname, err)
	}
	phone := utils.NormalizePhone(req.PhoneNumber)
	user, err := checkPassword(db, phone, req.Password)
	if err != nil {
		return utils.ReplyError(ctx, svcname, err)
	}
	if err := otpservices.VerifyOTP(db, phone, configs.OTP_ACTION_DEVICE_BINDING, req.OTPCode); err != nil {
		return utils.ReplyError(ctx, svcname, err)
	}
	if err := userrepositories.UpdateDevice(db, user.ID, req.DeviceID); err != nil {
		return utils.ReplyError(ctx, svcname, err)
	}
	user.CurrentDeviceID = &req.DeviceID
	return utils.ReplyOK(ctx, "Perangkat berhasil diganti", user)
}

// Session : GET /auth/session?user_id=&device_id=
// Dipanggil frontend tiap request terautentikasi (single-device lock).
func Session(ctx echo.Context) error {
	svcname := "Session"
	uid, _ := strconv.ParseInt(ctx.QueryParam("user_id"), 10, 64)
	deviceID := ctx.QueryParam("device_id")
	user, err := userrepositories.GetUserByID(connections.DBconn(), uid)
	if err != nil {
		if err == sql.ErrNoRows {
			return utils.ReplyError(ctx, svcname, helpers.NewAppError(configs.RC_APP_SESSION_INVALID, "Sesi tidak valid"))
		}
		return utils.ReplyError(ctx, svcname, err)
	}
	if user.CurrentDeviceID == nil || *user.CurrentDeviceID != deviceID || deviceID == "" {
		return utils.ReplyError(ctx, svcname, helpers.NewAppError(configs.RC_APP_SESSION_INVALID, "Akun Anda masuk di perangkat lain. Silakan login kembali"))
	}
	return utils.ReplyOK(ctx, "", user)
}
