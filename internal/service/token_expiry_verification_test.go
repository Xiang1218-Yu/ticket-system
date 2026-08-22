package service

import (
	"testing"
	"time"

	"ticket-system/internal/model"
)

func TestIssuedTokenKeepsConfiguredOneHourLifetime(t *testing.T) {
	svc := NewAuthService(nil, "secret", 1)
	issued := time.Now()
	token, err := svc.issueToken(&model.User{ID:1,Role:model.RoleEmployee}); if err != nil { t.Fatal(err) }
	claims, err := svc.ParseToken(token); if err != nil { t.Fatalf("fresh token was rejected: %v", err) }
	if claims.ExpiresAt == nil || claims.ExpiresAt.Time.Sub(issued) < 59*time.Minute { t.Fatalf("token lifetime = %v, want about one hour", claims.ExpiresAt.Time.Sub(issued)) }
}

func TestParseTokenRejectsWrongSignature(t *testing.T) {
	issuer := NewAuthService(nil, "issuer-secret", 1)
	token, err := issuer.issueToken(&model.User{ID:1, Role:model.RoleEmployee})
	if err != nil { t.Fatal(err) }
	if _, err := NewAuthService(nil, "other-secret", 1).ParseToken(token); err == nil {
		t.Fatal("token signed with another secret was accepted")
	}
}
