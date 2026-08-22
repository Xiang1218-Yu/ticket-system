# 状态流转一致性异常

## Bug 是什么
处理中工单可以绕过完成阶段直接关闭，并发推进时状态、时间字段和时间线可能互相矛盾。

## 如何触发
同时提交处理中到关闭以及合法状态推进请求，观察状态规则判断与条件更新之间的竞争；合法的完成后关闭路径仍应可用。

## 根因
涉及 internal/handler/ticket_handler.go、internal/model/ticket.go、internal/repository/ticket_repo.go、internal/service/ticket_service.go。相关符号是 UpdateStatus、LegalTransition、TransitionStatus、TicketService.UpdateStatus。状态污染来自流转规则过宽和更新未绑定读取到的旧状态，并发请求缺少比较并更新的一致性保护。

## 运行指令
```text
go test -v ./internal/model -run '^TestProcessingCannotCloseDirectlyUnderConcurrentRequests$' -race -count=20
```

## 错误信息
```text
processing status bypassed
```

## 错误堆栈
```text
--- FAIL: TestProcessingCannotCloseDirectlyUnderConcurrentRequests (0.03s)
internal/model/status_transition_verification_test.go:47
```
