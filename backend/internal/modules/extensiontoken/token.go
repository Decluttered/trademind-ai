package extensiontoken

import (
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/trademind-ai/trademind/backend/internal/config"
	"github.com/trademind-ai/trademind/backend/internal/modules/auth"
	"strings"
	"time"
)

const Audience = "mindbay-extension"

type Claims struct {
	TokenType   string `json:"typ"`
	WorkspaceID int64  `json:"workspace_id"`
	Scope       string `json:"scope"`
	jwt.RegisteredClaims
}

func Mint(cfg *config.Config, grant Grant) (string, error) {
	keys, err := auth.BuildKeySet(cfg)
	if err != nil {
		return "", err
	}
	claims := Claims{TokenType: "extension_capture", WorkspaceID: grant.WorkspaceID, Scope: grant.Scope, RegisteredClaims: jwt.RegisteredClaims{Subject: grant.AdminUserID.String(), ID: grant.JTI, Audience: jwt.ClaimStrings{Audience}, Issuer: "trademind-backend", IssuedAt: jwt.NewNumericDate(time.Now().UTC()), ExpiresAt: jwt.NewNumericDate(grant.ExpiresAt)}}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token.Header["kid"] = keys.ActiveID
	return token.SignedString(keys.ActiveSecret)
}
func Parse(cfg *config.Config, raw string) (*Claims, error) {
	keys, err := auth.BuildKeySet(cfg)
	if err != nil {
		return nil, err
	}
	parser := jwt.NewParser(jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithAudience(Audience), jwt.WithIssuer("trademind-backend"))
	token, err := parser.ParseWithClaims(strings.TrimSpace(raw), &Claims{}, func(t *jwt.Token) (any, error) {
		kid, _ := t.Header["kid"].(string)
		switch strings.TrimSpace(kid) {
		case "", keys.ActiveID:
			return keys.ActiveSecret, nil
		case keys.PreviousID:
			if len(keys.PreviousSecret) > 0 && !keys.GraceUntil.IsZero() && time.Now().UTC().Before(keys.GraceUntil) {
				return keys.PreviousSecret, nil
			}
		}
		return nil, fmt.Errorf("unknown extension token key")
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid || claims.TokenType != "extension_capture" || claims.Scope != "capture" || claims.WorkspaceID < 0 || claims.ID == "" {
		return nil, fmt.Errorf("invalid extension token claims")
	}
	if _, err := uuid.Parse(claims.Subject); err != nil {
		return nil, fmt.Errorf("invalid extension token subject")
	}
	return claims, nil
}
