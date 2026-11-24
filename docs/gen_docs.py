#!/usr/bin/env python3
"""Generator dokumentasi API PPOB Core (Go/Echo).

Satu sumber data (ENDPOINTS) -> dua keluaran:
  - PPOB_Core.postman_collection.json  (+ PPOB_Core.postman_environment.json)
  - PPOB_Core_API_Integrasi.docx

Jalankan:  python3 gen_docs.py
"""
import json
import os

HERE = os.path.dirname(os.path.abspath(__file__))
BASE_URL = "http://localhost:10001"
TS = "2026-10-12T08:30:00+07:00"
REF_NO = "TRX-20261012-7B0C1E2A5D3F4C9A9E1B2F6D8A4C0B11"


def env(status_code, status_description, data=None):
    return {
        "status_code": status_code,
        "status_description": status_description,
        "data": {} if data is None else data,
    }


USER = {
    "id": 12,
    "price_group_id": 1,
    "name": "Budi Santoso",
    "phone_number": "081234567890",
    "role": "MERCHANT",
    "store_name": "Toko Budi",
    "current_device_id": "device-abc-123",
    "is_phone_verified": True,
    "created_at": "2026-10-01T09:00:00+07:00",
}

USER_LOGIN = dict(
    USER,
    token="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoxMiwicm9sZSI6Ik1FUkNIQU5UIiwiZGV2aWNlX2lkIjoiZGV2aWNlLWFiYy0xMjMiLCJpYXQiOjE3NjAyMzI2MDAsImV4cCI6MTc2MDMxOTAwMH0.signature",
)

TRX_INQUIRY = {
    "reference_no": REF_NO,
    "customer_phone": "081234567890",
    "customer_id": "081234567890",
    "customer_name": None,
    "product_name": "Telkomsel 10.000",
    "product_price": 10000,
    "admin_fee": 500,
    "total_amount": 10500,
    "merchant_markup": 0,
    "grand_total": 10500,
    "serial_number": None,
    "allow_markup": True,
    "status_code": "06",
    "status_description": "Inquiry berhasil, siap untuk pembayaran",
    "created_at": TS,
    "updated_at": TS,
}

TRX_PENDING = dict(
    TRX_INQUIRY,
    merchant_markup=500,
    grand_total=11000,
    allow_markup=False,
    status_code="01",
    status_description="Sedang diproses oleh provider",
)

TRX_SUCCESS = dict(
    TRX_INQUIRY,
    merchant_markup=500,
    grand_total=11000,
    serial_number="1234567890123456",
    allow_markup=False,
    status_code="00",
    status_description="Transaksi Sukses",
)

PRODUCT = {
    "product_id": 5,
    "sku_internal": "TSEL10",
    "name": "Telkomsel 10.000",
    "nominal_value": 10000,
    "category_code": "PULSA",
    "brand_id": 1,
    "brand_code": "TELKOMSEL",
    "brand_name": "Telkomsel",
    "price_group_id": 1,
    "product_price": 10000,
    "admin_fee": 500,
    "total_amount": 10500,
}

BRAND = {
    "id": 1,
    "code": "TELKOMSEL",
    "name": "Telkomsel",
    "logo_url": "https://cdn.example.com/telkomsel.png",
}

# (nama, tipe, wajib, keterangan)
F = lambda *a: a


def err(http, sc, msg, kondisi):
    return (http, sc, msg, kondisi)


COMMON_ERR = [
    err(401, "05", "INVALID DIGITAL SIGNATURE", "Header X-Timestamp/X-Signature hilang, salah, atau kedaluwarsa"),
    err(400, "03", "Parameter tidak valid / data tidak lengkap", "Body bukan JSON valid / field wajib kosong"),
    err(500, "99", "Terjadi gangguan pada sistem", "Gangguan sistem tak terduga"),
]
SIGNED = True

ENDPOINTS = [
    # ---- Auth
    {
        "folder": "Auth",
        "name": "Request OTP",
        "method": "POST",
        "path": "/api/v1/auth/otp/request",
        "desc": "Meminta kode OTP via WhatsApp. Perilaku bergantung pada `action_type`:\n"
                "- `REGISTER`: nomor harus belum terdaftar (isi `phone_number`).\n"
                "- `LOGIN`: nomor harus sudah terdaftar (isi `phone_number`).\n"
                "- `DEVICE_BINDING`: wajib `phone_number` + `password` valid (mencegah spam OTP ke nomor orang lain).\n"
                "Ada cooldown antar permintaan OTP. Pada `APP_ENV=local` dan `WA_PROVIDER=log`, respons memuat `debug_otp` (hanya development).",
        "signed": SIGNED,
        "body": {"action_type": "REGISTER", "phone_number": "081234567890"},
        "body_fields": [
            F("action_type", "string", "Ya", "REGISTER | LOGIN | DEVICE_BINDING"),
            F("phone_number", "string", "Ya", "Nomor WhatsApp (08xx / 62xx / +62xx)"),
            F("password", "string", "Kondisional", "Wajib untuk DEVICE_BINDING"),
        ],
        "response": env(
            "00",
            "Kode OTP berhasil dikirim ke nomor WhatsApp Anda",
            {"phone_masked": "0812****890", "expires_in": 300, "cooldown_seconds": 60, "debug_otp": "123456"},
        ),
        "status": 200,
        "errors": [
            err(409, "03", "Nomor sudah terdaftar", "REGISTER: nomor sudah terdaftar"),
            err(401, "05", "Nomor HP tidak terdaftar / kata sandi salah", "LOGIN: nomor belum terdaftar, atau DEVICE_BINDING: kredensial salah"),
            err(429, "03", "Silakan tunggu N detik sebelum meminta OTP kembali", "Permintaan OTP terlalu cepat (cooldown)"),
            err(422, "03", "action_type tidak dikenal / Format nomor WhatsApp tidak valid", "Parameter tidak valid"),
        ],
    },
    {
        "folder": "Auth",
        "name": "Verifikasi OTP",
        "method": "POST",
        "path": "/api/v1/auth/otp/verify",
        "desc": "Memvalidasi kode OTP sebelum melanjutkan ke tahap registrasi atau login. "
                "Endpoint ini hanya memeriksa kecocokan OTP tanpa menandainya sebagai consumed agar kode OTP tetap dapat dipakai saat memanggil `/api/v1/auth/register-with-otp` atau `/api/v1/auth/login`.",
        "signed": SIGNED,
        "body": {"phone_number": "081234567890", "action_type": "REGISTER", "otp_code": "123456"},
        "body_fields": [
            F("phone_number", "string", "Ya", "Nomor WhatsApp"),
            F("action_type", "string", "Ya", "REGISTER | LOGIN | DEVICE_BINDING"),
            F("otp_code", "string", "Ya", "Kode OTP 6 digit"),
        ],
        "response": env("00", "OTP berhasil diverifikasi", {"verified": True}),
        "status": 200,
        "errors": [
            err(400, "03", "Parameter tidak valid", "phone_number atau otp_code kosong"),
            err(422, "03", "Kode OTP salah / telah kedaluwarsa / batas percobaan habis", "OTP salah, expired, atau melebihi batas percobaan"),
            err(404, "04", "Kode OTP tidak ditemukan atau sudah tidak berlaku", "Tidak ada OTP aktif untuk nomor & aksi ini"),
        ],
    },
    {
        "folder": "Auth",
        "name": "Register dengan OTP",
        "method": "POST",
        "path": "/api/v1/auth/register-with-otp",
        "desc": "Membuat akun MERCHANT baru setelah OTP `REGISTER` valid. Akun hanya dibuat di tabel `users` setelah OTP benar. Nomor HP valid: diawali `08`, 10–14 digit.",
        "signed": SIGNED,
        "body": {
            "name": "Budi Santoso",
            "phone_number": "081234567890",
            "password": "rahasia123",
            "store_name": "Toko Budi",
            "otp_code": "123456",
        },
        "body_fields": [
            F("name", "string", "Ya", "Nama lengkap"),
            F("phone_number", "string", "Ya", "Nomor WhatsApp"),
            F("password", "string", "Ya", "Minimal 6 karakter"),
            F("store_name", "string", "Ya", "Nama toko"),
            F("otp_code", "string", "Ya", "Kode OTP dari Request OTP (REGISTER)"),
        ],
        "response": env("00", "Pendaftaran berhasil", USER),
        "status": 201,
        "errors": [
            err(409, "03", "Nomor sudah terdaftar", "Nomor sudah terdaftar"),
            err(422, "03", "Kode OTP salah / kedaluwarsa / batas percobaan habis", "Validasi OTP gagal"),
            err(404, "04", "Kode OTP tidak ditemukan atau sudah tidak berlaku", "Tidak ada OTP aktif untuk nomor ini"),
        ],
    },
    {
        "folder": "Auth",
        "name": "Login",
        "method": "POST",
        "path": "/api/v1/auth/login",
        "desc": "Login menggunakan nomor HP dengan `password` ATAU `otp_code` (dari aksi `LOGIN`). "
                "Perangkat pertama otomatis terikat (`current_device_id`). Saat login sukses, sistem membentuk JWT token (`data.token`) yang memuat `user_id`, `role`, dan `device_id` untuk otorisasi endpoint transaksi & katalog user login. "
                "Bila `device_id` berbeda dari yang terikat, API mengembalikan HTTP 422 (`requires_otp: true`, `action_type: DEVICE_BINDING`) dan klien harus menjalankan alur Device Binding terlebih dahulu sebelum login ulang.",
        "signed": SIGNED,
        "save_jwt": True,
        "body": {
            "phone_number": "081234567890",
            "password": "rahasia123",
            "otp_code": "",
            "device_id": "device-abc-123",
        },
        "body_fields": [
            F("phone_number", "string", "Ya", "Nomor WhatsApp"),
            F("password", "string", "Kondisional", "Kata sandi (wajib jika otp_code kosong)"),
            F("otp_code", "string", "Kondisional", "Kode OTP LOGIN (opsional, alternatif login tanpa password)"),
            F("device_id", "string", "Ya", "ID unik perangkat klien"),
        ],
        "response": env("00", "Login berhasil", USER_LOGIN),
        "status": 200,
        "errors": [
            err(401, "05", "Nomor HP atau kata sandi salah", "Kredensial salah"),
            err(422, "99", "Login dari perangkat baru terdeteksi. Verifikasi OTP diperlukan", "Perangkat berbeda dari current_device_id"),
        ],
        "extra_responses": [
            (
                "Perangkat baru (Wajib OTP Device Binding)",
                422,
                env(
                    "99",
                    "Login dari perangkat baru terdeteksi. Verifikasi OTP diperlukan",
                    {"requires_otp": True, "action_type": "DEVICE_BINDING"},
                ),
            )
        ],
    },
    {
        "folder": "Auth",
        "name": "Device Binding",
        "method": "POST",
        "path": "/api/v1/auth/device-binding",
        "desc": "Memindahkan ikatan akun ke perangkat baru setelah OTP `DEVICE_BINDING` valid. Sesi di perangkat lama otomatis tidak valid. "
                "Urutan: Login (perangkat baru) -> Request OTP (DEVICE_BINDING) -> Device Binding.",
        "signed": SIGNED,
        "body": {
            "phone_number": "081234567890",
            "password": "rahasia123",
            "device_id": "device-baru-456",
            "otp_code": "123456",
        },
        "body_fields": [
            F("phone_number", "string", "Ya", "Nomor WhatsApp"),
            F("password", "string", "Ya", "Kata sandi"),
            F("device_id", "string", "Ya", "ID perangkat baru"),
            F("otp_code", "string", "Ya", "OTP DEVICE_BINDING"),
        ],
        "response": env("00", "Perangkat berhasil diganti", dict(USER, current_device_id="device-baru-456")),
        "status": 200,
        "errors": [
            err(401, "05", "Nomor HP atau kata sandi salah", "Kredensial salah"),
            err(422, "03", "Kode OTP salah / telah kedaluwarsa", "OTP salah atau kedaluwarsa"),
        ],
    },
    {
        "folder": "Auth",
        "name": "Cek Sesi",
        "method": "GET",
        "path": "/api/v1/auth/session",
        "desc": "Validasi sesi single-device. Dipanggil pada setiap request terautentikasi untuk memastikan perangkat masih sesuai dengan `current_device_id`.",
        "signed": SIGNED,
        "query": [
            F("user_id", "integer", "Ya", "ID pengguna", "12"),
            F("device_id", "string", "Ya", "ID perangkat aktif", "device-abc-123"),
        ],
        "response": env("00", "SUCCESS", USER),
        "status": 200,
        "errors": [
            err(401, "05", "Akun Anda masuk di perangkat lain. Silakan login kembali", "user tidak ada / device_id tidak cocok"),
        ],
    },
    # ---- Katalog
    {
        "folder": "Katalog",
        "name": "Katalog Produk (Filter & Pencarian)",
        "method": "POST",
        "path": "/api/v1/products",
        "desc": "Mengambil daftar produk aktif beserta harga sesuai price group pengguna.\n\n"
                "- **User Login:** Sertakan header `Authorization: Bearer <token>` (opsional `X-User-ID` harus sesuai `user_id` di token). Bila token tidak valid/expired atau perangkat sudah berpindah, ditolak dengan `401 Unauthorized` (`status_code: \"05\"`).\n"
                "- **Guest:** Kosongkan header `Authorization` dan `X-User-ID` (menggunakan price group default).",
        "signed": SIGNED,
        "headers": [
            ("Authorization", "Bearer {{jwt_token}}", "Wajib untuk User Login (kosongkan/nonaktifkan untuk Guest)"),
            ("X-User-ID", "12", "Opsional: jika diisi, wajib sama dengan user_id pada JWT"),
        ],
        "body": {
            "category_code": "PULSA",
            "brand_id": 1,
            "is_fast_product": True,
            "sku": "",
            "search": "",
        },
        "body_fields": [
            F("category_code", "string", "Tidak", "Filter kode kategori (mis. PULSA, DATA)"),
            F("brand_id", "integer", "Tidak", "Filter ID brand"),
            F("is_fast_product", "boolean", "Tidak", "Filter produk cepat/unggulan (true/false)"),
            F("sku", "string", "Tidak", "Filter berdasarkan sku_internal"),
            F("search", "string", "Tidak", "Kata kunci pencarian nama produk / SKU / brand / kategori"),
        ],
        "response": env("00", "Berhasil / Sukses", {"products": [PRODUCT]}),
        "status": 200,
        "errors": [
            err(401, "05", "Otorisasi tidak valid", "Token JWT tidak ada saat X-User-ID dikirim, token expired/tidak sah, atau device_id sudah berpindah"),
        ],
    },
    {
        "folder": "Katalog",
        "name": "Deteksi Prefix & Operator",
        "method": "POST",
        "path": "/api/v1/prefixes",
        "desc": "Mendeteksi brand/operator sekaligus mengembalikan daftar prefix yang cocok berdasarkan `customer_id` (nomor HP tujuan).",
        "signed": SIGNED,
        "body": {"customer_id": "081234567890"},
        "body_fields": [
            F("customer_id", "string", "Ya", "Nomor pelanggan / HP tujuan (minimal 4 digit)"),
        ],
        "response": env(
            "00",
            "Berhasil / Sukses",
            {"customer_id": "081234567890", "brand": BRAND, "prefixes": ["0812"]},
        ),
        "status": 200,
        "errors": [
            err(400, "03", "customer_id wajib diisi", "customer_id kosong"),
            err(404, "04", "Prefix/operator tidak ditemukan", "Nomor < 4 digit atau prefix operator tidak dikenali"),
        ],
    },
    {
        "folder": "Katalog",
        "name": "Daftar Kategori",
        "method": "GET",
        "path": "/api/v1/catalog/categories",
        "desc": "Daftar kategori produk aktif. `use_prefix=true` (PULSA, DATA) berarti operator dideteksi otomatis dari nomor tujuan.",
        "signed": SIGNED,
        "response": env(
            "00",
            "SUCCESS",
            [{"id": 1, "code": "PULSA", "name": "Pulsa", "icon_url": "https://cdn.example.com/pulsa.png", "use_prefix": True}],
        ),
        "status": 200,
        "errors": [],
    },
    {
        "folder": "Katalog",
        "name": "Daftar Brand",
        "method": "GET",
        "path": "/api/v1/catalog/brands",
        "desc": "Daftar brand/operator aktif yang memiliki produk aktif pada kategori tertentu.",
        "signed": SIGNED,
        "query": [F("category_code", "string", "Ya", "Kode kategori, mis. PULSA", "PULSA")],
        "response": env("00", "SUCCESS", [BRAND]),
        "status": 200,
        "errors": [],
    },
    {
        "folder": "Katalog",
        "name": "Deteksi Operator (GET)",
        "method": "GET",
        "path": "/api/v1/catalog/detect",
        "desc": "Deteksi operator dari 4 digit pertama nomor tujuan (setelah dinormalisasi ke format 08xx).",
        "signed": SIGNED,
        "query": [F("customer_id", "string", "Ya", "Nomor tujuan, minimal 4 digit", "081234567890")],
        "response": env("00", "SUCCESS", BRAND),
        "status": 200,
        "errors": [
            err(422, "04", "Operator tidak dikenali dari nomor tujuan", "Nomor < 4 digit atau prefix tidak dikenal"),
        ],
    },
    {
        "folder": "Katalog",
        "name": "Daftar Produk per Kategori/Brand (GET)",
        "method": "GET",
        "path": "/api/v1/catalog/products",
        "desc": "Daftar produk aktif beserta harga sesuai price group pengguna (guest memakai group default). "
                "Untuk kategori PULSA/DATA, `customer_id` wajib karena brand dideteksi otomatis; untuk kategori lain gunakan `brand_code`.",
        "signed": SIGNED,
        "query": [
            F("category_code", "string", "Ya", "Kode kategori", "PULSA"),
            F("customer_id", "string", "Kondisional", "Wajib untuk PULSA/DATA", "081234567890"),
            F("brand_code", "string", "Kondisional", "Untuk kategori non-prefix", ""),
            F("user_id", "integer", "Tidak", "Menentukan price group; kosong = guest", "12"),
        ],
        "response": env("00", "SUCCESS", {"brand": BRAND, "products": [PRODUCT]}),
        "status": 200,
        "errors": [
            err(422, "04", "Operator tidak dikenali dari nomor tujuan", "Kategori prefix & operator tak dikenali"),
            err(404, "04", "Pengguna tidak ditemukan", "user_id tidak ditemukan"),
        ],
    },
    # ---- Transaksi
    {
        "folder": "Transaksi",
        "name": "Inquiry Transaksi",
        "method": "POST",
        "path": "/api/v1/transaction/inquiry",
        "desc": "Melakukan validasi produk & nomor tujuan serta membuat record transaksi berstatus inquiry (`status_code: \"06\"`).\n\n"
                "- **User Login:** Wajib menyertakan header `Authorization: Bearer <token>` hasil login. Jika membawa `Authorization` atau `X-User-ID` tetapi token tidak ada/expired/invalid atau sesi perangkat sudah berpindah, langsung ditolak `401 Unauthorized` (`status_code: \"05\"`).\n"
                "- **Guest:** Jangan menyertakan header `Authorization` maupun `X-User-ID`, dan wajib mengisi `customer_phone`.\n"
                "- `allow_markup` bernilai `true` apabila pengguna ber-role `MERCHANT` dan tipe produk bukan `POSTPAID`.",
        "signed": SIGNED,
        "headers": [
            ("Authorization", "Bearer {{jwt_token}}", "Wajib untuk User Login (kosongkan/nonaktifkan untuk transaksi Guest)"),
            ("X-User-ID", "12", "Opsional: jika dikirim, wajib sama dengan user_id pada JWT"),
        ],
        "body": {
            "digital_product_id": 5,
            "customer_id": "081234567890",
            "customer_phone": "081234567890",
        },
        "body_fields": [
            F("digital_product_id", "integer", "Ya", "ID produk digital (mendukung juga alias product_id)"),
            F("customer_id", "string", "Ya", "Nomor tujuan / ID pelanggan"),
            F("customer_phone", "string", "Kondisional", "Nomor HP pembeli; wajib untuk transaksi guest"),
        ],
        "response": env("06", "Inquiry berhasil, siap untuk pembayaran", TRX_INQUIRY),
        "status": 200,
        "errors": [
            err(401, "05", "Otorisasi tidak valid", "Token JWT tidak sah/expired, X-User-ID tanpa token, atau sesi perangkat tidak valid"),
            err(400, "03", "Parameter tidak valid / customer_phone wajib diisi untuk transaksi guest", "Field wajib kosong"),
            err(200, "07", "Inquiry gagal: produk tidak tersedia / nomor tujuan tidak valid", "Produk nonaktif atau prefix nomor tidak sesuai brand produk"),
        ],
        "extra_responses": [
            (
                "Inquiry Gagal (status_code 07)",
                200,
                env("07", "Inquiry gagal: nomor tujuan/operator tidak valid", {"customer_id": "081234567890"}),
            )
        ],
    },
    {
        "folder": "Transaksi",
        "name": "Confirm Payment (Konfirmasi Pembayaran)",
        "method": "POST",
        "path": "/api/v1/transaction/confirm-payment",
        "desc": "Mengonfirmasi pembayaran atas transaksi berstatus inquiry (`06`), menyimpan `merchant_markup` & `grand_total`, mengubah status ke pemrosesan provider, lalu langsung men-dispatch transaksi ke provider.\n\n"
                "- **Wajib JWT untuk Transaksi User Login:** Jika saat Inquiry transaksi dibuat oleh User Login (`trx.UserID != nil`), maka pemanggilan Confirm Payment **wajib** menyertakan `Authorization: Bearer <token>` yang valid dengan `user_id` yang sama (jika tanpa token atau berbeda user -> `401 Unauthorized` / `status_code: \"05\"`).\n"
                "- `merchant_markup` hanya berlaku untuk transaksi milik user dengan role `MERCHANT` pada produk non-postpaid (0 s.d. `MAX_MERCHANT_MARKUP`).\n"
                "- Jika provider langsung memberi hasil, respons langsung mengembalikan `status_code` `00` (sukses) atau `02` (gagal); jika masih menunggu provider, mengembalikan `01` (pending).",
        "signed": SIGNED,
        "headers": [
            ("Authorization", "Bearer {{jwt_token}}", "Wajib jika transaksi dibuat oleh User Login (kosongkan/nonaktifkan untuk transaksi Guest)"),
            ("X-User-ID", "12", "Opsional: jika dikirim, wajib sama dengan user_id pada JWT"),
        ],
        "body": {
            "reference_no": REF_NO,
            "merchant_markup": 500,
        },
        "body_fields": [
            F("reference_no", "string", "Ya", "Nomor referensi dari respons Inquiry"),
            F("merchant_markup", "number", "Tidak", "Markup harga jual merchant (Rp, 0 s.d. MAX_MERCHANT_MARKUP)"),
        ],
        "response": env("00", "Transaksi Sukses", TRX_SUCCESS),
        "status": 200,
        "errors": [
            err(401, "05", "Otorisasi tidak valid", "Transaksi user login ditembak tanpa token JWT, token expired/tidak sah, atau user_id berbeda"),
            err(400, "03", "reference_no wajib diisi", "reference_no kosong"),
            err(404, "04", "Transaksi tidak ditemukan", "reference_no tidak ditemukan"),
            err(422, "03", "Transaksi tidak berada pada status inquiry / Nilai markup tidak valid", "Transaksi sudah pernah diproses atau markup di luar batas"),
        ],
        "extra_responses": [
            (
                "Sedang Diproses Provider (status_code 01)",
                200,
                env("01", "Sedang diproses oleh provider", TRX_PENDING),
            )
        ],
    },
    {
        "folder": "Transaksi",
        "name": "Cek Status Transaksi",
        "method": "GET",
        "path": "/api/v1/transaction/status/:reference_no",
        "desc": "Mengecek status terkini satu transaksi berdasarkan `reference_no`. Bila menyertakan `Authorization: Bearer <token>` (atau `X-User-ID`), token JWT wajib valid dan `user_id` harus cocok dengan pemilik transaksi.",
        "signed": SIGNED,
        "headers": [
            ("Authorization", "Bearer {{jwt_token}}", "Opsional untuk User Login (kosongkan untuk transaksi Guest)"),
            ("X-User-ID", "12", "Opsional: jika dikirim, wajib sama dengan user_id pada JWT"),
        ],
        "path_vars": [("reference_no", REF_NO)],
        "response": env("00", "Transaksi Sukses", TRX_SUCCESS),
        "status": 200,
        "errors": [
            err(401, "05", "Otorisasi tidak valid", "Token JWT tidak sah/expired atau sesi perangkat tidak valid"),
            err(404, "04", "Transaksi tidak ditemukan", "reference_no tidak ada atau bukan milik user"),
        ],
    },
    {
        "folder": "Transaksi",
        "name": "Riwayat Transaksi",
        "method": "POST",
        "path": "/api/v1/transactions",
        "desc": "Mengambil daftar riwayat transaksi (hanya transaksi yang sudah melewati tahap inquiry, yaitu berstatus `00`, `01`, atau `02`).\n\n"
                "- **User Login:** Sertakan header `Authorization: Bearer <token>` untuk menampilkan riwayat transaksi milik user tersebut.\n"
                "- **Guest:** Tanpa header `Authorization` dan `X-User-ID`, wajib mengisi `customer_phone` pada body.",
        "signed": SIGNED,
        "headers": [
            ("Authorization", "Bearer {{jwt_token}}", "Wajib untuk riwayat User Login (kosongkan untuk riwayat Guest via customer_phone)"),
            ("X-User-ID", "12", "Opsional: jika dikirim, wajib sama dengan user_id pada JWT"),
        ],
        "body": {
            "customer_phone": "081234567890",
            "limit": 30,
        },
        "body_fields": [
            F("customer_phone", "string", "Kondisional", "Wajib diisi untuk melihat riwayat transaksi guest (tanpa token JWT)"),
            F("limit", "integer", "Tidak", "Batas jumlah transaksi (1-100, default 30)"),
        ],
        "response": env("00", "Berhasil / Sukses", {"transactions": [TRX_SUCCESS]}),
        "status": 200,
        "errors": [
            err(401, "05", "Otorisasi tidak valid", "Token JWT tidak sah/expired atau sesi perangkat tidak valid"),
            err(400, "03", "customer_phone wajib untuk melihat riwayat guest", "Tanpa token JWT dan customer_phone kosong"),
        ],
    },
    # ---- Webhook
    {
        "folder": "Webhook",
        "name": "Webhook Provider",
        "method": "POST",
        "path": "/webhooks/provider",
        "desc": "Dipanggil provider (format Digiflazz). **Tanpa HMAC**; autentikasi via header `X-Callback-Token` (= `PROVIDER_CALLBACK_TOKEN`) atau query `?token=`. "
                "Payload boleh flat atau dibungkus objek `data`. Memperbarui status transaksi menjadi `00` (sukses), `01` (pending), atau `02` (gagal).",
        "signed": False,
        "callback": "provider_callback_token",
        "body": {
            "data": {
                "ref_id": REF_NO,
                "status": "Sukses",
                "rc": "00",
                "message": "Transaksi Sukses",
                "sn": "1234567890123456",
            }
        },
        "body_fields": [
            F("ref_id", "string", "Ya", "Nomor referensi transaksi (reference_no)"),
            F("status", "string", "Ya", "Status provider (Sukses / Pending / Gagal)"),
            F("rc", "string", "Tidak", "Response code provider"),
            F("message", "string", "Tidak", "Pesan status dari provider"),
            F("sn", "string", "Tidak", "Serial number / token PLN"),
            F("data", "object", "Tidak", "Bungkus payload standar webhook Digiflazz (berisi field yang sama)"),
        ],
        "response": env("00", "SUCCESS", {}),
        "status": 200,
        "errors": [
            err(401, "05", "INVALID DIGITAL SIGNATURE", "X-Callback-Token salah"),
            err(404, "04", "TRANSACTION IS NOT FOUND", "ref_id tidak ditemukan"),
            err(422, "03", "ref_id wajib", "ref_id kosong"),
        ],
    },
]

RC_TABLE = [
    ("00", "Berhasil / Sukses", "200 / 201", "Operasi atau transaksi berhasil"),
    ("01", "Sedang diproses oleh provider", "200", "Transaksi masih diproses / pending di sisi provider"),
    ("02", "Transaksi Gagal", "200 / 422", "Transaksi gagal di sisi provider atau status transaksi tidak valid"),
    ("03", "Parameter tidak valid / kesalahan OTP", "400 / 409 / 422 / 429", "Request tidak lengkap, format salah, OTP salah/expired/cooldown, atau nomor sudah terdaftar"),
    ("04", "Data tidak ditemukan", "404 / 422", "Produk, pengguna, transaksi, OTP, atau prefix operator tidak ditemukan"),
    ("05", "Otorisasi tidak valid", "401", "Signature HMAC salah, kredensial login salah, atau sesi perangkat tidak valid"),
    ("06", "Inquiry berhasil, siap untuk pembayaran", "200", "Inquiry berhasil dibuat dan siap dikonfirmasi pembayarannya"),
    ("07", "Inquiry gagal", "200", "Inquiry ditolak karena produk tidak tersedia atau nomor tujuan/operator tidak valid"),
    ("99", "Gangguan sistem / butuh aksi tambahan", "422 / 500", "Kesalahan internal sistem atau login perangkat baru membutuhkan OTP DEVICE_BINDING"),
]

STATUS_TABLES = {
    "status_code (status transaksi publik)": [
        ("06", "Inquiry berhasil, siap untuk pembayaran"),
        ("07", "Inquiry gagal (produk tidak tersedia / operator nomor tujuan tidak cocok)"),
        ("01", "Sedang diproses oleh provider (PROCESSING / HOLD_PENDING)"),
        ("00", "Transaksi berhasil (serial_number terisi bila ada)"),
        ("02", "Transaksi gagal"),
    ],
    "action_type OTP": [
        ("REGISTER", "Pendaftaran akun baru"),
        ("LOGIN", "Login menggunakan OTP"),
        ("DEVICE_BINDING", "Pengikatan perangkat baru saat pindah perangkat"),
    ],
}


# --------------------------------------------------------------------------- Postman
def pm_script_sign():
    return [
        "// Signature HMAC-SHA256: timestamp \\n METHOD \\n request-uri(+query) \\n body",
        "const CryptoJS = require('crypto-js');",
        "const ts = Math.floor(Date.now() / 1000).toString();",
        "const method = pm.request.method;",
        "const uri = pm.variables.replaceIn(pm.request.url.getPathWithQuery());",
        "const raw = (pm.request.body && pm.request.body.raw) ? pm.variables.replaceIn(pm.request.body.raw) : '';",
        "const sig = CryptoJS.HmacSHA256(ts + '\\n' + method + '\\n' + uri + '\\n' + raw, pm.variables.get('app_key')).toString(CryptoJS.enc.Hex);",
        "pm.request.headers.upsert({ key: 'X-Timestamp', value: ts });",
        "pm.request.headers.upsert({ key: 'X-Signature', value: sig });",
    ]


def pm_desc(ep):
    lines = [ep["desc"], ""]
    if ep.get("headers"):
        lines += ["**Headers**", "", "| Header | Contoh | Keterangan |", "|---|---|---|"]
        lines += [f"| {h[0]} | {h[1]} | {h[2]} |" for h in ep["headers"]]
        lines.append("")
    if ep.get("body_fields"):
        lines += ["**Body**", "", "| Field | Tipe | Wajib | Keterangan |", "|---|---|---|---|"]
        lines += [f"| {n} | {t} | {r} | {d} |" for n, t, r, d in ep["body_fields"]]
        lines.append("")
    if ep.get("query"):
        lines += ["**Query**", "", "| Param | Tipe | Wajib | Keterangan |", "|---|---|---|---|"]
        lines += [f"| {q[0]} | {q[1]} | {q[2]} | {q[3]} |" for q in ep["query"]]
        lines.append("")
    if ep["errors"]:
        lines += ["**Kemungkinan error**", "", "| HTTP | status_code | Kondisi |", "|---|---|---|"]
        lines += [f"| {h} | {rc} | {k} |" for h, rc, _, k in ep["errors"]]
    return "\n".join(lines)


def pm_request(ep):
    path = ep["path"]
    segs = [s for s in path.strip("/").split("/")]
    variables = []
    out_segs = []
    for s in segs:
        out_segs.append(s)
    for k, v in ep.get("path_vars", []):
        variables.append({"key": k, "value": v})
    url = {
        "raw": "{{base_url}}" + path + ("?" + "&".join(f"{q[0]}={q[4]}" for q in ep["query"]) if ep.get("query") else ""),
        "host": ["{{base_url}}"],
        "path": out_segs,
    }
    if ep.get("query"):
        url["query"] = [{"key": q[0], "value": q[4], "description": q[3], "disabled": q[4] == ""} for q in ep["query"]]
    if variables:
        url["variable"] = variables
    headers = []
    if ep.get("callback"):
        headers.append({"key": "X-Callback-Token", "value": "{{" + ep["callback"] + "}}"})
    for hk, hv, hdesc in ep.get("headers", []):
        headers.append({"key": hk, "value": hv, "description": hdesc})
    req = {"method": ep["method"], "header": headers, "url": url, "description": pm_desc(ep)}
    if ep.get("body") is not None:
        headers.append({"key": "Content-Type", "value": "application/json"})
        req["body"] = {
            "mode": "raw",
            "raw": json.dumps(ep["body"], indent=2, ensure_ascii=False),
            "options": {"raw": {"language": "json"}},
        }
    return req


def build_postman():
    folders, order = {}, []
    for ep in ENDPOINTS:
        f = ep["folder"]
        if f not in folders:
            folders[f] = []
            order.append(f)
        req = pm_request(ep)
        responses = [
            {
                "name": "Sukses",
                "originalRequest": req,
                "status": "OK",
                "code": ep["status"],
                "header": [{"key": "Content-Type", "value": "application/json"}],
                "body": json.dumps(ep["response"], indent=2, ensure_ascii=False),
            }
        ]
        for nm, code, body in ep.get("extra_responses", []):
            responses.append(
                {
                    "name": nm,
                    "originalRequest": req,
                    "status": "OK" if code < 400 else "Error",
                    "code": code,
                    "header": [{"key": "Content-Type", "value": "application/json"}],
                    "body": json.dumps(body, indent=2, ensure_ascii=False),
                }
            )
        item_obj = {"name": ep["name"], "request": req, "response": responses, "_signed": ep["signed"]}
        if ep.get("save_jwt"):
            item_obj["event"] = [
                {
                    "listen": "test",
                    "script": {
                        "type": "text/javascript",
                        "exec": [
                            "try {",
                            "  const res = pm.response.json();",
                            "  if (res && res.data && res.data.token) {",
                            "    pm.environment.set('jwt_token', res.data.token);",
                            "    pm.collectionVariables.set('jwt_token', res.data.token);",
                            "  }",
                            "} catch (e) {}",
                        ],
                    },
                }
            ]
        folders[f].append(item_obj)

    items = []
    signed_items, webhook_items = [], []
    for f in order:
        sub = []
        for i in folders[f]:
            entry = {"name": i["name"], "request": i["request"], "response": i["response"]}
            if "event" in i:
                entry["event"] = i["event"]
            sub.append(entry)
        node = {"name": f, "item": sub}
        (signed_items if folders[f][0]["_signed"] else webhook_items).append(node)
    items.append(
        {
            "name": "Internal API (HMAC)",
            "description": "Endpoint /api/v1/*. Header X-Timestamp & X-Signature dibuat otomatis oleh pre-request script.",
            "event": [{"listen": "prerequest", "script": {"type": "text/javascript", "exec": pm_script_sign()}}],
            "item": signed_items,
        }
    )
    items.append({"name": "Webhook (X-Callback-Token)", "description": "Endpoint /webhooks/*. Tanpa HMAC.", "item": webhook_items})

    return {
        "info": {
            "name": "PPOB Core API",
            "schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json",
            "description": "Dokumentasi integrasi PPOB Core (Go/Echo). Set variabel `base_url` dan `app_key` (APP_KEY di .env Core), "
                           "`jwt_token` (otomatis terisi saat hit endpoint Login), serta `provider_callback_token` untuk webhook.",
        },
        "item": items,
        "variable": [
            {"key": "base_url", "value": BASE_URL},
            {"key": "app_key", "value": "dev-shared-secret-change-me"},
            {"key": "jwt_token", "value": ""},
            {"key": "provider_callback_token", "value": "dev-provider-token"},
        ],
    }


def build_postman_env():
    return {
        "name": "PPOB Core - Local",
        "_postman_variable_scope": "environment",
        "values": [
            {"key": "base_url", "value": BASE_URL, "enabled": True},
            {"key": "app_key", "value": "dev-shared-secret-change-me", "type": "secret", "enabled": True},
            {"key": "jwt_token", "value": "", "type": "secret", "enabled": True},
            {"key": "provider_callback_token", "value": "dev-provider-token", "type": "secret", "enabled": True},
        ],
    }


# --------------------------------------------------------------------------- DOCX
SIGN_SAMPLES = {
    "Go": '''import ("crypto/hmac"; "crypto/sha256"; "encoding/hex"; "strconv"; "time")

ts := strconv.FormatInt(time.Now().Unix(), 10)
msg := ts + "\\n" + method + "\\n" + requestURI + "\\n" + string(body)
mac := hmac.New(sha256.New, []byte(appKey))
mac.Write([]byte(msg))
sig := hex.EncodeToString(mac.Sum(nil))
// header: X-Timestamp: ts , X-Signature: sig''',
    "PHP (Laravel)": '''$ts  = (string) time();
$msg = $ts . "\\n" . $method . "\\n" . $requestUri . "\\n" . $body;
$sig = hash_hmac('sha256', $msg, $appKey);   // hex
// header: X-Timestamp: $ts , X-Signature: $sig''',
    "Node.js": '''const crypto = require('crypto');
const ts = Math.floor(Date.now() / 1000).toString();
const msg = `${ts}\\n${method}\\n${requestUri}\\n${body}`;
const sig = crypto.createHmac('sha256', appKey).update(msg).digest('hex');''',
    "Python": '''import hmac, hashlib, time
ts = str(int(time.time()))
msg = f"{ts}\\n{method}\\n{request_uri}\\n{body}".encode()
sig = hmac.new(app_key.encode(), msg, hashlib.sha256).hexdigest()''',
}


def build_docx(path):
    """Membangun file .docx (OpenXML) murni menggunakan stdlib Python (zipfile + xml) tanpa dependensi eksternal."""
    import zipfile
    from xml.sax.saxutils import escape

    body_parts = []

    def w_p(runs_xml, pPr_xml=""):
        body_parts.append(f"<w:p>{pPr_xml}{runs_xml}</w:p>")

    def w_r(text, bold=False, size_pt=10.5, color=None, font="Calibri"):
        rPr = [f'<w:rFonts w:ascii="{font}" w:hAnsi="{font}" w:cs="{font}"/>']
        if bold:
            rPr.append("<w:b/>")
        if size_pt:
            sz = int(size_pt * 2)
            rPr.append(f'<w:sz w:val="{sz}"/><w:szCs w:val="{sz}"/>')
        if color:
            rPr.append(f'<w:color w:val="{color}"/>')
        rPr_xml = f"<w:rPr>{''.join(rPr)}</w:rPr>"
        lines = str(text).split("\n")
        t_parts = []
        for idx, ln in enumerate(lines):
            if idx > 0:
                t_parts.append("<w:br/>")
            t_parts.append(f'<w:t xml:space="preserve">{escape(ln)}</w:t>')
        return f"<w:r>{rPr_xml}{''.join(t_parts)}</w:r>"

    def add_heading(text, level=1):
        sizes = {1: (16, "1F3A6E", 240, 120), 2: (13, "2F5496", 200, 80), 3: (11.5, "1F3A6E", 160, 60)}
        sz, col, before, after = sizes.get(level, (11, "1F3A6E", 120, 60))
        pPr = f'<w:pPr><w:keepNext/><w:spacing w:before="{before}" w:after="{after}"/><w:outlineLvl w:val="{level-1}"/></w:pPr>'
        w_p(w_r(text, bold=True, size_pt=sz, color=col), pPr)

    def para(text="", bold=False):
        pPr = '<w:pPr><w:spacing w:after="100" w:line="260" w:lineRule="auto"/></w:pPr>'
        w_p(w_r(text, bold=bold) if text else "", pPr)

    def bullets(items):
        for item in items:
            pPr = '<w:pPr><w:ind w:left="360" w:hanging="180"/><w:spacing w:after="60"/></w:pPr>'
            w_p(w_r("•  " + item), pPr)

    def add_page_break():
        body_parts.append('<w:p><w:r><w:br w:type="page"/></w:r></w:p>')

    borders_xml = (
        '<w:tblBorders>'
        '<w:top w:val="single" w:sz="4" w:space="0" w:color="BFBFBF"/>'
        '<w:left w:val="single" w:sz="4" w:space="0" w:color="BFBFBF"/>'
        '<w:bottom w:val="single" w:sz="4" w:space="0" w:color="BFBFBF"/>'
        '<w:right w:val="single" w:sz="4" w:space="0" w:color="BFBFBF"/>'
        '<w:insideH w:val="single" w:sz="4" w:space="0" w:color="BFBFBF"/>'
        '<w:insideV w:val="single" w:sz="4" w:space="0" w:color="BFBFBF"/>'
        '</w:tblBorders>'
    )
    cell_mar = (
        '<w:tblCellMar>'
        '<w:top w:w="80" w:type="dxa"/><w:left w:w="120" w:type="dxa"/>'
        '<w:bottom w:w="80" w:type="dxa"/><w:right w:w="120" w:type="dxa"/>'
        '</w:tblCellMar>'
    )

    def table(headers, rows, widths=None):
        tr_list = []
        h_cells = []
        for h in headers:
            tcPr = '<w:tcPr><w:shd w:val="clear" w:color="auto" w:fill="D9E2F3"/></w:tcPr>'
            p = f'<w:p><w:pPr><w:spacing w:after="40"/></w:pPr>{w_r(h, bold=True, size_pt=9.5)}</w:p>'
            h_cells.append(f"<w:tc>{tcPr}{p}</w:tc>")
        tr_list.append(f"<w:tr><w:trPr><w:tblHeader/></w:trPr>{''.join(h_cells)}</w:tr>")
        for row in rows:
            d_cells = []
            for v in row:
                p = f'<w:p><w:pPr><w:spacing w:after="40"/></w:pPr>{w_r(str(v), size_pt=9.5)}</w:p>'
                d_cells.append(f"<w:tc><w:tcPr/>{p}</w:tc>")
            tr_list.append(f"<w:tr>{''.join(d_cells)}</w:tr>")
        tblPr = f'<w:tblPr><w:tblW w:w="5000" w:type="pct"/>{borders_xml}{cell_mar}</w:tblPr>'
        body_parts.append(f"<w:tbl>{tblPr}{''.join(tr_list)}</w:tbl>")
        para()

    def code(text):
        tcPr = '<w:tcPr><w:shd w:val="clear" w:color="auto" w:fill="F2F2F2"/></w:tcPr>'
        p = f'<w:p><w:pPr><w:spacing w:after="40"/></w:pPr>{w_r(text, size_pt=8.5, font="Courier New")}</w:p>'
        tblPr = f'<w:tblPr><w:tblW w:w="5000" w:type="pct"/>{borders_xml}{cell_mar}</w:tblPr>'
        body_parts.append(f"<w:tbl>{tblPr}<w:tr><w:tc>{tcPr}{p}</w:tc></w:tr></w:tbl>")
        para()

    # ---- Cover
    para()
    w_p(w_r("PPOB Core API", bold=True, size_pt=28, color="1F3A6E"), '<w:pPr><w:jc w:val="center"/><w:spacing w:before="600" w:after="160"/></w:pPr>')
    w_p(w_r("Panduan Integrasi untuk Integrator", size_pt=15, color="2F5496"), '<w:pPr><w:jc w:val="center"/><w:spacing w:after="120"/></w:pPr>')
    w_p(w_r("Backend Go (Echo) - PPOB & Produk Digital", size_pt=11), '<w:pPr><w:jc w:val="center"/><w:spacing w:after="400"/></w:pPr>')
    add_page_break()

    # ---- 1
    add_heading("1. Pendahuluan", 1)
    para("PPOB Core adalah layanan backend (Go + Echo + PostgreSQL) yang menangani autentikasi OTP, katalog produk, "
         "perhitungan harga, inquiry transaksi, konfirmasi pembayaran, serta pemrosesan transaksi ke provider. "
         "Dokumen ini menjelaskan cara memanggil API tersebut. File Postman terlampir (PPOB_Core.postman_collection.json) "
         "berisi seluruh endpoint lengkap dengan contoh request/response dan perhitungan signature otomatis.")
    add_heading("Dua kelompok endpoint", 2)
    table(["Kelompok", "Prefix", "Autentikasi", "Pemanggil"], [
        ["Internal API", "/api/v1/*", "HMAC-SHA256 (X-Timestamp + X-Signature)", "Server integrator (mis. Laravel). Bukan untuk dipanggil langsung dari browser/aplikasi klien"],
        ["Webhook", "/webhooks/*", "Header X-Callback-Token", "Provider (Digiflazz)"],
    ])
    add_heading("Base URL & konfigurasi", 2)
    table(["Item", "Nilai"], [
        ["Base URL (development)", BASE_URL],
        ["Port", "APP_PORT (default 10001)"],
        ["Content-Type", "application/json"],
        ["Secret HMAC", "APP_KEY di file .env Core (dibagikan ke integrator, jangan di-commit)"],
        ["Token webhook provider", "PROVIDER_CALLBACK_TOKEN"],
    ])

    # ---- 2
    add_heading("2. Autentikasi (HMAC-SHA256)", 1)
    para("Setiap request ke /api/v1/* wajib membawa dua header:")
    table(["Header", "Isi"], [
        ["X-Timestamp", "Unix time (detik). Selisih dengan waktu server maksimal SIGN_MAX_AGE_SECOND di kedua arah (anti replay)."],
        ["X-Signature", "hex( HMAC-SHA256( APP_KEY, signing_string ) ) - huruf kecil"],
    ])
    para("Signing string (dipisah karakter newline \\n, tanpa newline di akhir):", bold=True)
    code("timestamp \\n METHOD \\n REQUEST_URI \\n BODY")
    bullets([
        "METHOD: huruf besar (GET, POST, PUT).",
        "REQUEST_URI: path + query string persis seperti yang dikirim, mis. /api/v1/catalog/products?category_code=PULSA&customer_id=081234567890.",
        "BODY: body mentah persis seperti yang dikirim (byte per byte). Untuk GET tanpa body gunakan string kosong.",
        "Jangan mengubah/format ulang JSON setelah signature dihitung.",
        "Gagal verifikasi -> HTTP 401, status_code 05 (INVALID DIGITAL SIGNATURE).",
    ])
    add_heading("Contoh implementasi", 2)
    for lang, src in SIGN_SAMPLES.items():
        para(lang, bold=True)
        code(src)
    para("Contoh signing string untuk POST /api/v1/auth/login:", bold=True)
    code('1760232600\\nPOST\\n/api/v1/auth/login\\n{"phone_number":"081234567890","password":"rahasia123","device_id":"device-abc-123"}')

    # ---- 3
    add_heading("3. Format Respons", 1)
    para("Semua respons memakai envelope JSON yang sama:")
    code(json.dumps(env("00", "SUCCESS", {"...": "..."}), indent=2, ensure_ascii=False))
    table(["Field", "Keterangan"], [
        ["status_code", "Kode hasil aplikasi (string). \"00\" = sukses, \"06\" = inquiry siap bayar. Gunakan field ini untuk logika bisnis."],
        ["status_description", "Pesan untuk manusia (Bahasa Indonesia); dapat berubah, jangan dijadikan acuan logika."],
        ["data", "Payload data (objek atau array)."],
    ])
    para("Catatan: nilai uang berupa angka (Rupiah, boleh desimal). Field waktu berformat RFC 3339. "
         "Field opsional bernilai null bila kosong.")

    # ---- 4
    add_heading("4. Kode Status (status_code) & HTTP Status", 1)
    table(["status_code", "Arti Umum", "HTTP", "Keterangan"], RC_TABLE)

    # ---- 5
    add_heading("5. Alur Integrasi", 1)
    add_heading("5.1 Registrasi & Login", 2)
    bullets([
        "1. POST /api/v1/auth/otp/request (action_type=REGISTER).",
        "2. (Opsional) POST /api/v1/auth/otp/verify untuk memvalidasi OTP sebelum submit form.",
        "3. POST /api/v1/auth/register-with-otp -> akun MERCHANT dibuat.",
        "4. POST /api/v1/auth/login (dengan password atau otp_code dari action_type=LOGIN). Jika perangkat baru (requires_otp=true, action_type=DEVICE_BINDING): lanjut langkah 5-6.",
        "5. POST /api/v1/auth/otp/request (action_type=DEVICE_BINDING, kirim phone_number + password).",
        "6. POST /api/v1/auth/device-binding -> perangkat lama otomatis tidak valid.",
        "7. Setiap request klien terautentikasi: GET /api/v1/auth/session untuk memastikan sesi masih valid.",
    ])
    add_heading("5.2 Pembelian produk", 2)
    bullets([
        "1. Gunakan POST /api/v1/products, POST /api/v1/prefixes, atau GET /api/v1/catalog/* untuk memilih produk dan melihat harga.",
        "2. POST /api/v1/transaction/inquiry -> validasi nomor tujuan & membuat transaksi berstatus '06' (Inquiry berhasil, siap untuk pembayaran); simpan reference_no.",
        "3. POST /api/v1/transaction/confirm-payment -> mengonfirmasi pembayaran (beserta merchant_markup bila ada) dan memproses ke provider.",
        "4. Pantau hasil: GET /api/v1/transaction/status/{reference_no} sampai status_code '00' (berhasil) atau '02' (gagal). Status '01' berarti provider masih memproses.",
        "5. Riwayat transaksi dapat diambil melalui POST /api/v1/transactions.",
    ])
    add_heading("5.3 Perhitungan harga", 2)
    code("product_price = base_hpp + spread                    (spread dirahasiakan)\n"
         "total_amount  = product_price + provider_admin_fee + admin_fee\n"
         "grand_total   = total_amount + merchant_markup")
    para("Seluruh harga dihitung ulang di server; nilai dari klien tidak dipercaya. merchant_markup hanya berlaku bagi role MERCHANT "
         "untuk produk non-postpaid dan dibatasi 0 s.d. MAX_MERCHANT_MARKUP (default 500.000); selain itu dipaksa 0.")
    add_heading("5.4 Tabel Status", 2)
    for title, rows in STATUS_TABLES.items():
        para(title, bold=True)
        table(["Nilai", "Arti"], rows)

    # ---- 6
    add_heading("6. Referensi Endpoint", 1)
    cur = None
    folder_list = ["Auth", "Katalog", "Transaksi", "Webhook"]
    for ep in ENDPOINTS:
        if ep["folder"] != cur:
            cur = ep["folder"]
            add_heading(f"6.{folder_list.index(cur)+1} {cur}", 2)
        add_heading(f"{ep['name']}", 3)
        code(f"{ep['method']}  {ep['path']}")
        para("Autentikasi: " + ("HMAC (X-Timestamp, X-Signature)" if ep["signed"] else "Header X-Callback-Token"))
        for line in ep["desc"].split("\n"):
            if line.strip():
                para(line.replace("`", "").replace("**", ""))
        if ep.get("headers"):
            para("Header tambahan", bold=True)
            table(["Header", "Contoh", "Keterangan"], [list(h) for h in ep["headers"]])
        if ep.get("path_vars"):
            para("Path parameter", bold=True)
            table(["Nama", "Contoh"], [[k, v] for k, v in ep["path_vars"]])
        if ep.get("query"):
            para("Query parameter", bold=True)
            table(["Nama", "Tipe", "Wajib", "Keterangan"], [[q[0], q[1], q[2], q[3]] for q in ep["query"]])
        if ep.get("body_fields"):
            para("Body (JSON)", bold=True)
            table(["Field", "Tipe", "Wajib", "Keterangan"], ep["body_fields"])
        if ep.get("body") is not None:
            para("Contoh request", bold=True)
            code(json.dumps(ep["body"], indent=2, ensure_ascii=False))
        para(f"Contoh respons sukses (HTTP {ep['status']})", bold=True)
        code(json.dumps(ep["response"], indent=2, ensure_ascii=False))
        for nm, c, body in ep.get("extra_responses", []):
            para(f"{nm} (HTTP {c})", bold=True)
            code(json.dumps(body, indent=2, ensure_ascii=False))
        errs = ep["errors"] + (COMMON_ERR if ep["signed"] else [])
        if errs:
            para("Kemungkinan error", bold=True)
            table(["HTTP", "status_code", "Pesan", "Kondisi"], [list(e) for e in errs])

    # ---- 7
    add_heading("7. Webhook - Keamanan & Perilaku", 1)
    bullets([
        "Kirim token pada header X-Callback-Token (alternatif: query ?token=). Token salah -> HTTP 401 status_code 05.",
        "Webhook provider menerima payload flat atau dibungkus 'data' (format Digiflazz).",
        "Core juga menjalankan worker polling untuk transaksi pending ('01') dan memulihkan transaksi processing ('03') yang macet.",
    ])

    # ---- 8
    add_heading("8. Variabel Lingkungan Relevan", 1)
    table(["Variabel", "Fungsi"], [
        ["APP_KEY", "Secret HMAC bersama antara Laravel dan Core"],
        ["JWT_SECRET / JWT_TTL_SECOND", "Secret dan masa berlaku (default 86400 detik) JWT untuk otorisasi user login"],
        ["SIGN_MAX_AGE_SECOND", "Toleransi umur timestamp HMAC (default 300 detik)"],
        ["OTP_TTL_SECOND / OTP_COOLDOWN_SECOND / OTP_MAX_ATTEMPT", "Masa berlaku, jeda, dan batas percobaan OTP"],
        ["MAX_MERCHANT_MARKUP", "Batas maksimum markup merchant (default 500000)"],
        ["PROVIDER_MODE", "mock | digiflazz"],
        ["PROVIDER_CALLBACK_TOKEN", "Token autentikasi webhook provider"],
    ])

    sectPr = (
        '<w:sectPr>'
        '<w:pgSz w:w="11906" w:h="16838"/>'
        '<w:pgMar w:top="1440" w:right="1440" w:bottom="1440" w:left="1440" w:header="708" w:footer="708" w:gutter="0"/>'
        '</w:sectPr>'
    )
    document_xml = (
        '<?xml version="1.0" encoding="UTF-8" standalone="yes"?>'
        '<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">'
        f'<w:body>{"".join(body_parts)}{sectPr}</w:body></w:document>'
    )
    content_types_xml = (
        '<?xml version="1.0" encoding="UTF-8" standalone="yes"?>'
        '<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">'
        '<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>'
        '<Default Extension="xml" ContentType="application/xml"/>'
        '<Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>'
        '</Types>'
    )
    rels_xml = (
        '<?xml version="1.0" encoding="UTF-8" standalone="yes"?>'
        '<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">'
        '<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>'
        '</Relationships>'
    )

    with zipfile.ZipFile(path, "w", zipfile.ZIP_DEFLATED) as zf:
        zf.writestr("[Content_Types].xml", content_types_xml)
        zf.writestr("_rels/.rels", rels_xml)
        zf.writestr("word/document.xml", document_xml.encode("utf-8"))


def main():
    with open(os.path.join(HERE, "PPOB_Core.postman_collection.json"), "w", encoding="utf-8") as f:
        json.dump(build_postman(), f, indent=2, ensure_ascii=False)
    with open(os.path.join(HERE, "PPOB_Core.postman_environment.json"), "w", encoding="utf-8") as f:
        json.dump(build_postman_env(), f, indent=2, ensure_ascii=False)
    print("Postman OK")
    build_docx(os.path.join(HERE, "PPOB_Core_API_Integrasi.docx"))
    print("DOCX OK")


if __name__ == "__main__":
    main()
