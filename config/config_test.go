package config

import (
	"os"
	"testing"
)

// 验证 TS_JWT_EXPIRE_HOURS 这类点分隔键的环境变量覆盖确实生效。
// 修复前 viper 用带点的非法变量名去 LookupEnv，覆盖静默失效。
func TestEnvOverrideExpireHours(t *testing.T) {
	const key = "TS_JWT_EXPIRE_HOURS"
	os.Setenv(key, "3")
	t.Cleanup(func() { os.Unsetenv(key) })

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.JWT.ExpireHours != 3 {
		t.Fatalf("环境变量覆盖失效: got=%d, want=3", cfg.JWT.ExpireHours)
	}
}

// 验证非法/为 0 的过期小时数被钳制到 1，而非让上层“减一”后变成 0。
func TestNormalizeExpireHours(t *testing.T) {
	for _, c := range []struct{ in, want int }{
		{-5, 1}, {0, 1}, {1, 1}, {24, 24},
	} {
		if got := normalizeExpireHours(c.in); got != c.want {
			t.Errorf("normalizeExpireHours(%d)=%d, want=%d", c.in, got, c.want)
		}
	}
}
