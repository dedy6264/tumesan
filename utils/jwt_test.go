package utils

import (
	"ppobcore/configs"
	"ppobcore/models"
	"testing"
)

func TestGenerateAndVerifyJWT(t *testing.T) {
	devID := "device-xyz-999"
	u := models.User{
		ID:              42,
		Role:            "MERCHANT",
		CurrentDeviceID: &devID,
	}

	tok, err := GenerateJWT(u)
	if err != nil {
		t.Fatalf("GenerateJWT failed: %v", err)
	}
	if tok == "" {
		t.Fatal("expected non-empty token")
	}

	claims, err := VerifyJWT(tok)
	if err != nil {
		t.Fatalf("VerifyJWT failed: %v", err)
	}
	if claims.UserID != 42 || claims.Role != "MERCHANT" || claims.DeviceID != devID {
		t.Fatalf("unexpected claims: %+v", claims)
	}

	// Tampered signature must fail
	if _, err := VerifyJWT(tok + "a"); err == nil {
		t.Fatal("expected tampered token to fail")
	}

	// Expired token must fail
	origTTL := configs.JWT_TTL_SECOND
	configs.JWT_TTL_SECOND = -10
	defer func() { configs.JWT_TTL_SECOND = origTTL }()

	expiredTok, err := GenerateJWT(u)
	if err != nil {
		t.Fatalf("GenerateJWT expired failed: %v", err)
	}
	if _, err := VerifyJWT(expiredTok); err == nil {
		t.Fatal("expected expired token to fail")
	}
}
