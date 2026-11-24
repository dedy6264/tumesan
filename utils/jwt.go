package utils

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"ppobcore/configs"
	"ppobcore/models"
	"strings"
	"time"
)

type JWTClaims struct {
	UserID   int64  `json:"user_id"`
	Role     string `json:"role"`
	DeviceID string `json:"device_id"`
	IssuedAt int64  `json:"iat"`
	Expires  int64  `json:"exp"`
}

type jwtHeader struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
}

func signJWT(signingInput, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signingInput))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func GenerateJWT(user models.User) (string, error) {
	hdrBytes, err := json.Marshal(jwtHeader{Alg: "HS256", Typ: "JWT"})
	if err != nil {
		return "", err
	}
	deviceID := ""
	if user.CurrentDeviceID != nil {
		deviceID = *user.CurrentDeviceID
	}
	now := time.Now().Unix()
	claimsBytes, err := json.Marshal(JWTClaims{
		UserID:   user.ID,
		Role:     user.Role,
		DeviceID: deviceID,
		IssuedAt: now,
		Expires:  now + int64(configs.JWT_TTL_SECOND),
	})
	if err != nil {
		return "", err
	}
	encodedHdr := base64.RawURLEncoding.EncodeToString(hdrBytes)
	encodedPayload := base64.RawURLEncoding.EncodeToString(claimsBytes)
	signingInput := encodedHdr + "." + encodedPayload
	sig := signJWT(signingInput, configs.JWT_SECRET)
	return signingInput + "." + sig, nil
}

func VerifyJWT(tokenStr string) (*JWTClaims, error) {
	parts := strings.Split(tokenStr, ".")
	if len(parts) != 3 {
		return nil, errors.New("invalid jwt format")
	}
	hdrBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, errors.New("invalid jwt header encoding")
	}
	var hdr jwtHeader
	if err := json.Unmarshal(hdrBytes, &hdr); err != nil || hdr.Alg != "HS256" {
		return nil, errors.New("unsupported jwt algorithm")
	}
	signingInput := parts[0] + "." + parts[1]
	expectedSig := signJWT(signingInput, configs.JWT_SECRET)
	if !hmac.Equal([]byte(expectedSig), []byte(parts[2])) {
		return nil, errors.New("invalid jwt signature")
	}
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, errors.New("invalid jwt payload encoding")
	}
	var claims JWTClaims
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		return nil, errors.New("invalid jwt claims")
	}
	if claims.UserID <= 0 || claims.DeviceID == "" {
		return nil, errors.New("incomplete jwt claims")
	}
	if time.Now().Unix() > claims.Expires {
		return nil, errors.New("jwt token expired")
	}
	return &claims, nil
}
