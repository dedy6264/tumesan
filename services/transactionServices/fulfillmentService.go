package transactionservices

import (
	"context"
	"database/sql"
	"ppobcore/configs"
	"ppobcore/connections"
	"ppobcore/helpers"
	"ppobcore/models"
	transactionrepositories "ppobcore/repositories/transactionRepositories"
	workerservices "ppobcore/services/workerServices"
	"ppobcore/utils"
	"time"
)

// Provider = Single Provider Dispatcher (mock / digiflazz, diatur PROVIDER_MODE).
var Provider = workerservices.NewProvider()

// DispatchToProvider mengirim pesanan (status PROCESSING) ke provider tunggal.
// ref_id ke provider = nomor invoice, sehingga pengiriman ulang bersifat idempotent di sisi provider.
func DispatchToProvider(invoice string) error {
	db := connections.DBconn()
	trx, err := transactionrepositories.GetByReference(db, invoice)
	if err != nil {
		return err
	}
	if trx.StatusCode != configs.STATUS_PROCESSING {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), configs.PROVIDER_TIMEOUT)
	defer cancel()
	res := Provider.Purchase(ctx, trx.ReferenceNo, trx.ProviderSku, trx.CustomerID)
	return ApplyProviderResult(db, invoice, res)
}

// ApplyProviderResult menerapkan hasil provider (dispatch / polling / webhook) secara idempotent.
// Hanya transaksi berstatus PROCESSING atau HOLD_PENDING yang diproses.
func ApplyProviderResult(db *sql.DB, invoice string, res models.ProviderResult) error {
	return helpers.DBTransaction(db, func(tx *sql.Tx) error {
		trx, err := transactionrepositories.GetByReferenceForUpdate(tx, invoice)
		if err != nil {
			return err
		}
		if trx.StatusCode != configs.STATUS_PROCESSING && trx.StatusCode != configs.STATUS_HOLD_PENDING {
			return nil // sudah final
		}
		provCode := res.State
		provDesc := res.Description
		if res.Code != "" {
			provDesc = "[" + res.Code + "] " + res.Description
		}

		switch res.State {
		case configs.PROV_SUCCESS:
			return transactionrepositories.UpdateProviderStatus(tx, trx.ID, configs.STATUS_SUCCESS, "Transaksi berhasil",
				provCode, provDesc, res.RefID, res.SerialNo)

		case configs.PROV_FAILED:
			return transactionrepositories.UpdateProviderStatus(tx, trx.ID, configs.STATUS_FAILED, "Transaksi gagal",
				provCode, provDesc, res.RefID, res.SerialNo)

		default: // PENDING / tidak pasti
			return transactionrepositories.UpdateProviderStatus(tx, trx.ID, configs.STATUS_HOLD_PENDING,
				"Transaksi sedang diproses oleh provider. Mohon tunggu konfirmasi",
				configs.PROV_PENDING, provDesc, res.RefID, "")
		}
	})
}

// StartPoller menjalankan worker latar belakang:
//   - HOLD_PENDING          : cek status ke provider sampai final
//   - PROCESSING macet >2m  : kirim ulang (idempotent via ref_id) - pemulihan bila proses sempat mati
func StartPoller(ctx context.Context) {
	ticker := time.NewTicker(configs.POLL_INTERVAL)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			pollOnce()
		}
	}
}

func pollOnce() {
	db := connections.DBconn()
	held, err := transactionrepositories.ListStale(db, configs.STATUS_HOLD_PENDING, int(configs.POLL_INTERVAL.Seconds()), 50)
	if err != nil {
		utils.Log("ListStale HOLD_PENDING", "Poller", err)
	}
	for _, t := range held {
		cctx, cancel := context.WithTimeout(context.Background(), configs.PROVIDER_TIMEOUT)
		res := Provider.CheckStatus(cctx, t.ReferenceNo, t.ProviderSku, t.CustomerID)
		cancel()
		if res.State == configs.PROV_PENDING {
			// tetap pending: sentuh updated_at agar antrean bergilir
			_ = transactionrepositories.UpdateProviderStatus(db, t.ID, configs.STATUS_HOLD_PENDING, t.StatusDescription,
				configs.PROV_PENDING, res.Description, "", "")
			continue
		}
		if err := ApplyProviderResult(db, t.ReferenceNo, res); err != nil {
			utils.Log("ApplyProviderResult "+t.ReferenceNo, "Poller", err)
		}
	}
	stuck, err := transactionrepositories.ListStale(db, configs.STATUS_PROCESSING, 120, 50)
	if err != nil {
		utils.Log("ListStale PROCESSING", "Poller", err)
	}
	for _, t := range stuck {
		if err := DispatchToProvider(t.ReferenceNo); err != nil {
			utils.Log("DispatchToProvider "+t.ReferenceNo, "Poller", err)
		}
	}
}
