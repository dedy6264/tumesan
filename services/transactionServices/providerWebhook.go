package transactionservices

import (
	"crypto/subtle"
	"database/sql"
	"net/http"
	"ppobcore/configs"
	"ppobcore/connections"
	"ppobcore/helpers"
	"ppobcore/models"
	workerservices "ppobcore/services/workerServices"
	"ppobcore/utils"

	"github.com/labstack/echo/v4"
)

func tokenOK(ctx echo.Context, expected string) bool {
	got := ctx.Request().Header.Get("X-Callback-Token")
	if got == "" {
		got = ctx.QueryParam("token")
	}
	return subtle.ConstantTimeCompare([]byte(got), []byte(expected)) == 1
}

// ProviderWebhook : POST /webhooks/provider
func ProviderWebhook(ctx echo.Context) error {
	const svcname = "ProviderWebhook"
	var req models.ReqProviderWebhook
	if !tokenOK(ctx, configs.PROVIDER_CALLBACK_TOKEN) {
		return utils.Reply(ctx, http.StatusUnauthorized, configs.RC_APP_INVALID_SIGNATURE, "", nil)
	}
	if err := ctx.Bind(&req); err != nil {
		return utils.ReplyError(ctx, svcname, err)
	}
	refID, status, rc, msg, sn := req.RefID, req.Status, req.RC, req.Message, req.SN
	if req.Data != nil {
		refID, status, rc, msg, sn = req.Data.RefID, req.Data.Status, req.Data.RC, req.Data.Message, req.Data.SN
	}
	if refID == "" {
		return utils.ReplyError(ctx, svcname, helpers.NewAppError(configs.RC_APP_INVALID_PARAM, "ref_id wajib"))
	}
	res := workerservices.MapDigiflazz(status, rc, msg, sn, refID)
	if err := ApplyProviderResult(connections.DBconn(), refID, res); err != nil {
		if err == sql.ErrNoRows {
			return utils.Reply(ctx, http.StatusNotFound, configs.RC_APP_TRX_NOT_FOUND, "", nil)
		}
		return utils.ReplyError(ctx, svcname, err)
	}
	return utils.ReplyOK(ctx, "", nil)
}
