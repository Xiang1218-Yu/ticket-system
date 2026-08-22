# 业务编号并发异常

## Bug 是什么
并发创建工单时，业务编号可能重复，记录数量和时间线也会随竞争窗口失真。该题只交付诊断结论，不修改生产代码。

## 如何触发
让多个请求同时创建工单，使编号统计、编号生成和记录写入交错执行；重复编号会在相同初始数据下稳定出现。

## 根因
涉及 internal/handler/ticket_handler.go、internal/service/ticket_service.go、internal/repository/ticket_repo.go、internal/model/ticket.go。相关符号是 CreateTicket、nextTicketNo、TicketRepository.Create、Ticket.TicketNo。并发竞态发生在编号生成与写入之间，原子边界和唯一性约束没有共同保护持久化结果。

## 运行指令
```text
go test -v ./internal/repository -run '^TestConcurrentTicketCreationKeepsBusinessNumberUnique$' -race -count=20
```

## 错误信息
```text
duplicate business number persisted
```

## 错误堆栈
```text
--- FAIL: TestConcurrentTicketCreationKeepsBusinessNumberUnique (0.01s)
internal/repository/concurrent_ticket_number_verification_test.go:42
```
