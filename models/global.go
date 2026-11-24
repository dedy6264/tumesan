package models

type (
	GlobalResponse struct {
		ResponseCode     string `json:"status_code"`
		ResponseMessage  string `json:"status_description"`
		ResponseDateTime string `json:"-"`
		Result           any    `json:"data"`
	}
)
