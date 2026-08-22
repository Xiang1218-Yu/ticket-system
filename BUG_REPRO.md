# 平均处理时长口径异常

## Bug 是什么
仍在处理的工单只因残留完成时间就被纳入平均处理时长，统计结果与生命周期状态不一致。

## 如何触发
准备一条未结束但带有完成时间的记录，再读取平均处理时长；该记录不应参与聚合，空数据场景仍应返回既有含义。

## 根因
涉及 internal/model/ticket.go、internal/repository/ticket_repo.go、internal/service/stats_service.go、internal/handler/stats_handler.go。相关符号是 Ticket.Status、AvgProcessingMinutes、Stats、Dashboard。状态污染发生在聚合仅依赖时间字段而忽略生命周期状态，异常结果随后跨层传到看板响应。

## 运行指令
```text
go test -v ./internal/repository -run '^TestAverageDurationExcludesOpenTicketsWithCompletedTimestamp$' -count=1
```

## 错误信息
```text
included an open ticket
```

## 错误堆栈
```text
--- FAIL: TestAverageDurationExcludesOpenTicketsWithCompletedTimestamp (0.00s)
internal/repository/avg_status_verification_test.go:39
```
