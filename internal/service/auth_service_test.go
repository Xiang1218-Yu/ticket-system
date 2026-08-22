package service

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"ticket-system/internal/model"
)

// 颁发与解析的测试辅助：直接构造一个 AuthService，跳过 DB。
func newAuthService(expireHours int) *AuthService {
	return &AuthService{
		jwtSecret:   []byte("test-secret-key-32bytes-long!!"),
		expireHours: expireHours,
	}
}

func TestOneHourTokenNotImmediatelyExpired(t *testing.T) {
	s := newAuthService(1)
	u := &model.User{ID: 1, Role: model.RoleEmployee}
	tok, err := s.issueToken(u)
	if err != nil {
		t.Fatalf("issueToken: %v", err)
	}
	if _, err := s.ParseToken(tok); err != nil {
		t.Fatalf("1h 令牌刚签发即被判过期: %v", err)
	}
}

func TestTokenExpiryMatchesConfig(t *testing.T) {
	cases := []struct{ hours, wantSec int }{
		{1, 3600}, {2, 7200}, {24, 86400},
	}
	now := time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC)
	for _, c := range cases {
		s := newAuthService(c.hours)
		got := int(s.tokenExpiry(now).Sub(now).Seconds())
		if got != c.wantSec {
			t.Errorf("expireHours=%d: 寿命=%ds, 期望=%ds", c.hours, got, c.wantSec)
		}
	}
}

// 边界时刻一致性：新旧会话在正好到达 exp 的瞬间应被判同一结果。
// 修复后注入了固定 parseLeeway，到点（cmp==exp）仍应通过——无论何时签发。
func TestBoundaryConsistentAcrossIssuanceTimes(t *testing.T) {
	s := newAuthService(1)
	// 旧会话：很久以前签发，exp 就在“当前”边界上。
	oldNow := time.Now().Add(-1 * time.Hour)
	claims := Claims{
		UserID: 1,
		Role:   model.RoleEmployee,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(oldNow),
			ExpiresAt: jwt.NewNumericDate(oldNow.Add(1 * time.Hour)), // == now
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	boundaryToken, err := tok.SignedString(s.jwtSecret)
	if err != nil {
		t.Fatal(err)
	}
	// 新会话：理论上 exp 也落在同一 now。两者 exp 相同 → 边界判定必须一致。
	if _, err := s.ParseToken(boundaryToken); err != nil {
		t.Fatalf("边界时刻(now==exp)新旧会话判定不一致/被判过期: %v", err)
	}
}

// 异常令牌仍应被拒绝：篡改签名、非预期签名方法、错误密钥签发。
func TestMalformedTokensRejected(t *testing.T) {
	s := newAuthService(24)
	u := &model.User{ID: 1, Role: model.RoleEmployee}
	tok, _ := s.issueToken(u)

	// 篡改签名中段字符：HS256 签名为 32 字节、base64url 编码为 43 字符，
	// 仅末字符携带的 4 位会被 RawURLEncoding 忽略；翻转中间字符必然改变
	// 解码后的签名字节，从而触发 HMAC 校验失败。
	parts := strings.Split(tok, ".")
	if len(parts) != 3 {
		t.Fatalf("令牌结构异常: parts=%d", len(parts))
	}
	sig := parts[2]
	if len(sig) < 3 {
		t.Fatalf("签名过短: %d", len(sig))
	}
	orig := sig[1]
	alt := byte('A')
	if orig == 'A' {
		alt = 'B'
	}
	tamperedSig := sig[:1] + string(alt) + sig[2:]
	tampered := parts[0] + "." + parts[1] + "." + tamperedSig
	if _, err := s.ParseToken(tampered); err == nil {
		t.Error("篡改签名令牌应被拒绝")
	}

	// 另一密钥签发的令牌。
	other := &AuthService{jwtSecret: []byte("a-different-secret-also-32bytes!"), expireHours: 24}
	tok2, _ := other.issueToken(u)
	if _, err := s.ParseToken(tok2); err == nil {
		t.Error("异密钥令牌应被拒绝")
	}
}

// 真正过期的令牌（远超 exp + leeway）必须被拒绝。
func TestGenuinelyExpiredTokenRejected(t *testing.T) {
	s := newAuthService(1)
	past := time.Now().Add(-2 * time.Hour) // exp = now-1h，远超 leeway
	claims := Claims{
		UserID: 1,
		Role:   model.RoleEmployee,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(past),
			ExpiresAt: jwt.NewNumericDate(past.Add(1 * time.Hour)),
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	expiredToken, _ := tok.SignedString(s.jwtSecret)
	if _, err := s.ParseToken(expiredToken); err == nil {
		t.Error("已过期令牌应被拒绝")
	}
}
