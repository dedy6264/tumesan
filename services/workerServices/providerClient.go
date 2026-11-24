package workerservices

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"ppobcore/configs"
	"ppobcore/models"
	"strings"
	"time"
)

// Provider adalah abstraksi Single Provider B2B. Implementasi WAJIB mengembalikan
// State=01 (pending) untuk timeout atau respons ambigu agar transaksi dipantau ulang.
// padahal provider mungkin sudah memproses (timeout safeguard).
type Provider interface {
	Purchase(ctx context.Context, refID, sku, customerNo string) models.ProviderResult
	CheckStatus(ctx context.Context, refID, sku, customerNo string) models.ProviderResult
}

func NewProvider() Provider {
	if configs.PROVIDER_MODE == "digiflazz" {
		return &DigiflazzProvider{client: &http.Client{Timeout: configs.PROVIDER_TIMEOUT}}
	}
	return &MockProvider{}
}

// ---------------------------------------------------------------- Mock
// Perilaku berdasarkan digit terakhir customer_id (untuk pengujian):
//
//	...9 -> gagal    ...8 -> pending lalu berhasil saat polling    ...7 -> pending lalu gagal saat polling
//	lainnya -> berhasil
type MockProvider struct{}

func (m *MockProvider) Purchase(_ context.Context, refID, _ string, customerNo string) models.ProviderResult {
	switch customerNo[len(customerNo)-1] {
	case '9':
		return models.ProviderResult{State: configs.PROV_FAILED, Description: "[MOCK-FAIL] Nomor tujuan tidak dapat diisi (mock)", RefID: refID}
	case '8', '7':
		return models.ProviderResult{State: configs.PROV_PENDING, Description: "[MOCK-PENDING] Transaksi sedang diproses provider (mock)", RefID: refID}
	}
	return models.ProviderResult{State: configs.PROV_SUCCESS, Description: "Transaksi berhasil (mock)", RefID: refID,
		SerialNo: fmt.Sprintf("%s%d", time.Now().Format("060102150405"), time.Now().Nanosecond()%100000)}
}

func (m *MockProvider) CheckStatus(ctx context.Context, refID, sku, customerNo string) models.ProviderResult {
	switch customerNo[len(customerNo)-1] {
	case '8':
		return models.ProviderResult{State: configs.PROV_SUCCESS, Description: "Transaksi berhasil (mock)", RefID: refID,
			SerialNo: fmt.Sprintf("%s%d", time.Now().Format("060102150405"), time.Now().Nanosecond()%100000)}
	case '7':
		return models.ProviderResult{State: configs.PROV_FAILED, Description: "[MOCK-FAIL] Transaksi gagal di provider (mock)", RefID: refID}
	}
	return m.Purchase(ctx, refID, sku, customerNo)
}

// ---------------------------------------------------------------- Digiflazz
type DigiflazzProvider struct {
	client *http.Client
}

type digiflazzResponse struct {
	Data struct {
		RefID   string `json:"ref_id"`
		Status  string `json:"status"`
		RC      string `json:"rc"`
		Message string `json:"message"`
		SN      string `json:"sn"`
	} `json:"data"`
}

func (d *DigiflazzProvider) Purchase(ctx context.Context, refID, sku, customerNo string) models.ProviderResult {
	return d.call(ctx, refID, sku, customerNo)
}

// Digiflazz: request ulang dengan ref_id yang sama = cek status (idempotent).
func (d *DigiflazzProvider) CheckStatus(ctx context.Context, refID, sku, customerNo string) models.ProviderResult {
	return d.call(ctx, refID, sku, customerNo)
}

func (d *DigiflazzProvider) call(ctx context.Context, refID, sku, customerNo string) models.ProviderResult {
	pending := func(desc string) models.ProviderResult {
		return models.ProviderResult{State: configs.PROV_PENDING, Description: "[TIMEOUT] " + desc, RefID: refID}
	}
	sum := md5.Sum([]byte(configs.PROVIDER_USERNAME + configs.PROVIDER_API_KEY + refID))
	payload, _ := json.Marshal(map[string]string{
		"username":       configs.PROVIDER_USERNAME,
		"buyer_sku_code": sku,
		"customer_no":    customerNo,
		"ref_id":         refID,
		"sign":           hex.EncodeToString(sum[:]),
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(configs.PROVIDER_BASE_URL, "/")+"/transaction", bytes.NewReader(payload))
	if err != nil {
		return pending(err.Error())
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := d.client.Do(req)
	if err != nil {
		return pending("Koneksi ke provider terputus / timeout: " + err.Error())
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var r digiflazzResponse
	if err := json.Unmarshal(body, &r); err != nil || r.Data.Status == "" {
		return pending("Respon provider tidak dapat dibaca")
	}
	return MapDigiflazz(r.Data.Status, r.Data.RC, r.Data.Message, r.Data.SN, refID)
}

// MapDigiflazz menerjemahkan status provider menjadi kode numerik: 00, 01, atau 02.
func MapDigiflazz(status, rc, message, sn, refID string) models.ProviderResult {
	description := message
	if rc != "" {
		description = "[" + rc + "] " + message
	}
	res := models.ProviderResult{Description: description, RefID: refID, SerialNo: sn}
	switch strings.ToLower(status) {
	case "sukses", "success":
		res.State = configs.PROV_SUCCESS
	case "gagal", "failed":
		res.State = configs.PROV_FAILED
	default:
		res.State = configs.PROV_PENDING
	}
	return res
}
