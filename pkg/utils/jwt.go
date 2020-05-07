package utils

import (
	"fmt"

	jwt "github.com/dgrijalva/jwt-go"
)

// Claims is alias of jwt.MapClaims
type Claims = jwt.MapClaims

// GenerateToken will generate JWT token
func GenerateToken(secret string, payload Claims) (*string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, payload)
	ts, err := token.SignedString([]byte(secret))
	return &ts, err
}

// ValidateToken will validate wether a token valid
func ValidateToken(secret string, tokenString string) (Claims, error) {
	// validate token
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		_, ok := t.Method.(*jwt.SigningMethodHMAC)
		if !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	payload, ok := token.Claims.(Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("failed to parse token payload")
	}
	return payload, nil
}
