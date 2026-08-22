# 列表授权范围异常

## Bug 是什么
组长可以借助外来筛选条件读取其他处理组的工单，超时列表也存在相同的范围扩大风险。

## 如何触发
使用组长身份请求列表并附加其他处理组的筛选条件，再观察普通列表和超时列表的返回范围；管理员的同类筛选仍应保留。

## 根因
涉及 internal/middleware/auth.go、internal/handler/auth_handler.go、internal/handler/ticket_handler.go、internal/service/ticket_service.go。相关符号是 AuthMiddleware、CurrentUser、ListTickets、ListOverdueTickets。跨层数据流没有收敛可信身份所属组和外来筛选组，服务授权边界因此被请求条件扩大。

## 运行指令
```text
go test -v ./internal/service -run '^TestLeaderCannotExpandListScopeWithRequestedGroup$' -count=1
```

## 错误信息
```text
outside own group
```

## 错误堆栈
```text
--- FAIL: TestLeaderCannotExpandListScopeWithRequestedGroup (0.00s)
internal/service/group_scope_verification_test.go:44
```
