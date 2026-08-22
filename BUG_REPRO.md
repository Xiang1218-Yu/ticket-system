# 看板状态聚合异常

## Bug 是什么
数据变化后，看板仍可能带出已经消失的状态数量，并把旧值传播给后续并发请求。该题只交付诊断结论，不修改生产代码。

## 如何触发
先读取状态聚合结果，再删除或改变对应状态，同时并发发起另一轮读取；共享聚合结果未失效时会把旧状态重新带回响应。

## 根因
涉及 internal/handler/stats_handler.go、internal/service/stats_service.go、internal/repository/ticket_repo.go、internal/model/ticket.go。相关符号是 Stats、Dashboard、CountByStatus、TicketRepository.statusCache。状态污染和并发竞态共同作用：缓存没有失效策略或同步保护，缺少的状态又被旧聚合结果补回。

## 运行指令
```text
go test -v ./internal/repository -run '^TestStatusAggregationDropsDeletedStatusDuringConcurrentReads$' -race -count=20
```

## 错误信息
```text
deleted status remained
```

## 错误堆栈
```text
--- FAIL: TestStatusAggregationDropsDeletedStatusDuringConcurrentReads (0.03s)
internal/repository/stats_cache_verification_test.go:51
```
