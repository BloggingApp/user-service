package utils

import "github.com/golang-jwt/jwt/v5"

type JWTPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type GenerateJWTPairDto struct {
	Method        jwt.SigningMethod
	AccessSecret  []byte
	AccessClaims  jwt.MapClaims
	RefreshSecret []byte
	RefreshClaims jwt.MapClaims
}

func GenerateJWTPair(dto GenerateJWTPairDto) (*JWTPair, error) {
	accessToken := jwt.NewWithClaims(dto.Method, dto.AccessClaims)
	accessTokenString, err := accessToken.SignedString(dto.AccessSecret)
	if err != nil {
		return nil, err
	}

	refreshToken := jwt.NewWithClaims(dto.Method, dto.RefreshClaims)
	refreshTokenString, err := refreshToken.SignedString(dto.RefreshSecret)
	if err != nil {
		return nil, err
	}

	return &JWTPair{
		AccessToken:  accessTokenString,
		RefreshToken: refreshTokenString,
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
