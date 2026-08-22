# 重复评价异常

## Bug 是什么
同一工单的并发评价可能同时通过重复检查并写入多条记录，竞争落败方也无法得到稳定清晰的反馈。

## 如何触发
让两个相同评价请求在检查完成后、写入开始前交错执行，两个请求都可能看到尚无评价并继续持久化。

## 根因
涉及 internal/model/review.go、internal/repository/review_repo.go、internal/service/review_service.go、internal/handler/ticket_handler.go。相关符号是 Review、CreateReview、HasReviewForTicket、ReviewHandler。并发竞态位于重复检查与写入之间，缺少原子约束使一单一评依赖请求时序。

## 运行指令
```text
go test -v ./internal/service -run '^TestConcurrentReviewsLeaveExactlyOneRecord$' -race -count=20
```

## 错误信息
```text
parallel reviews created
```

## 错误堆栈
```text
--- FAIL: TestConcurrentReviewsLeaveExactlyOneRecord (0.02s)
internal/service/concurrent_review_verification_test.go:63
```
