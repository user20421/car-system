package middleware

import (
	"testing"
	"time"
)

func setup() {
	InitJWT("test-secret", time.Hour)
}

func TestGenerateAndParseToken(t *testing.T) {
	setup()
	token, err := GenerateToken(1, "admin", "admin")
	if err != nil {
		t.Fatalf("生成 Token 失败: %v", err)
	}
	claims, err := ParseToken(token)
	if err != nil {
		t.Fatalf("解析 Token 失败: %v", err)
	}
	if claims.UserID != 1 || claims.Username != "admin" || claims.Role != "admin" {
		t.Errorf("Claims 内容不匹配: %+v", claims)
	}
}

func TestParseInvalidToken(t *testing.T) {
	setup()
	if _, err := ParseToken("invalid.token.string"); err == nil {
		t.Error("解析非法 Token 应返回错误")
	}
}

func TestTokenExpiry(t *testing.T) {
	InitJWT("test-secret", -time.Hour) // 过期时长为负，立即过期
	token, _ := GenerateToken(1, "admin", "admin")
	if _, err := ParseToken(token); err == nil {
		t.Error("过期 Token 应校验失败")
	}
}
