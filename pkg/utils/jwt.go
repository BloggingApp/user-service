package utils

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type JWTPair struct {
	AccessToken     string        `json:"access_token"`
	AccessTokenExp  time.Duration `json:"access_token_exp"`
	RefreshToken    string        `json:"refresh_token"`
	RefreshTokenExp time.Duration `json:"refresh_token_exp"`
}

type GenerateJWTPairDto struct {
	Method        jwt.SigningMethod
	AccessSecret  []byte
	AccessClaims  jwt.MapClaims
	AccessExpiry  time.Duration
	RefreshSecret []byte
	RefreshClaims jwt.MapClaims
	RefreshExpiry time.Duration
}

func GenerateJWTPair(dto GenerateJWTPairDto) (*JWTPair, error) {
	dto.AccessClaims["exp"] = time.Now().Add(dto.AccessExpiry).Unix()
	accessToken := jwt.NewWithClaims(dto.Method, dto.AccessClaims)
	accessTokenString, err := accessToken.SignedString(dto.AccessSecret)
	if err != nil {
		return nil, err
	}

	dto.RefreshClaims["exp"] = time.Now().Add(dto.RefreshExpiry).Unix()
	refreshToken := jwt.NewWithClaims(dto.Method, dto.RefreshClaims)
	refreshTokenString, err := refreshToken.SignedString(dto.RefreshSecret)
	if err != nil {
		return nil, err
	}

	return &JWTPair{
		AccessToken: accessTokenString,
		AccessTokenExp: dto.AccessExpiry,
		RefreshToken: refreshTokenString,
		RefreshTokenExp: dto.RefreshExpiry,
	}, nil
}

func DecodeJWT(token string, secret []byte) (jwt.MapClaims, error) {
	parsedToken, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
		return secret, nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	if !ok || !parsedToken.Valid {
		return nil, err
	}

	return claims, nil
}
