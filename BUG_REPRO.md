# 令牌有效期传播异常

## Bug 是什么
新签发的一小时令牌可能在有效期内被立即判定为过期，配置、签发和解析之间的期限语义不一致。

## 如何触发
加载一小时有效期配置并立即签发令牌，随后走认证解析链路；合法令牌不应在边界前失效，错误签名仍须拒绝。

## 根因
涉及 config/config.go、internal/service/auth_service.go、internal/middleware/auth.go、internal/handler/auth_handler.go。相关符号是 JWTConfig.ExpireHours、NewAuthService、issueToken、ParseToken。Context 生命周期在配置期限、声明生成和认证解析之间传播不一致，时间单位或边界语义偏移后造成新会话提前过期。

## 运行指令
```text
go test -v ./internal/service -run '^TestIssuedTokenKeepsConfiguredOneHourLifetime$' -count=1
```

## 错误信息
```text
fresh token was rejected
```

## 错误堆栈
```text
--- FAIL: TestIssuedTokenKeepsConfiguredOneHourLifetime (0.00s)
internal/service/token_expiry_verification_test.go:35
```
