package configs

var (
	RC_APP_SUCCESS             = []string{"00", "SUCCESS"}
	RC_APP_DB_ERROR            = []string{"01", "DB ERROR"}
	RC_APP_SYSTEM_ERROR        = []string{"07", "SYSTEM ERROR"}
	RC_APP_PRODUCT_NOT_FOUND   = []string{"10", "PRODUCT NOT FOUND OR INACTIVE"}
	RC_APP_INVALID_SIGNATURE   = []string{"13", "INVALID DIGITAL SIGNATURE"}
	RC_APP_INVALID_TRANSACTION = []string{"15", "INVALID TRANSACTION STATE"}
	RC_APP_TRX_NOT_FOUND       = []string{"22", "TRANSACTION IS NOT FOUND"}
	RC_APP_INVALID_PARAM       = []string{"39", "INVALID REQUEST PARAMETER"}
	RC_APP_OTP_COOLDOWN        = []string{"50", "OTP REQUEST COOLDOWN"}
	RC_APP_OTP_INVALID         = []string{"51", "OTP INVALID"}
	RC_APP_OTP_EXPIRED         = []string{"52", "OTP EXPIRED"}
	RC_APP_OTP_ATTEMPT_LIMIT   = []string{"53", "OTP ATTEMPT LIMIT REACHED"}
	RC_APP_OTP_NOT_FOUND       = []string{"54", "OTP NOT FOUND"}
	RC_APP_DEVICE_OTP          = []string{"56", "NEW DEVICE REQUIRES OTP"}
	RC_APP_PHONE_REGISTERED    = []string{"57", "PHONE ALREADY REGISTERED"}
	RC_APP_INVALID_CREDENTIAL  = []string{"58", "INVALID PHONE OR PASSWORD"}
	RC_APP_USER_NOT_FOUND      = []string{"59", "USER NOT FOUND"}
	RC_APP_PREFIX_NOT_FOUND    = []string{"60", "OPERATOR NOT RECOGNIZED"}
	RC_APP_SESSION_INVALID     = []string{"66", "DEVICE SESSION IS NO LONGER VALID"}
)

// Kode status transaksi (kolom transactions.status_code)
const (
	STATUS_PROCESSING   = "03"
	STATUS_HOLD_PENDING = "01"
	STATUS_SUCCESS      = "00"
	STATUS_FAILED       = "02"
)

// Kode status provider (kolom transactions.provider_status_code)
const (
	PROV_SUCCESS = "00"
	PROV_PENDING = "01"
	PROV_FAILED  = "02"
)

// Aksi OTP (otp_verifications.action_type)
const (
	OTP_ACTION_REGISTER       = "REGISTER"
	OTP_ACTION_LOGIN          = "LOGIN"
	OTP_ACTION_DEVICE_BINDING = "DEVICE_BINDING"
)

// Kategori yang memakai deteksi operator via product_prefixes (customer_id = nomor HP)
var PREFIX_CATEGORIES = []string{"PULSA", "DATA"}
