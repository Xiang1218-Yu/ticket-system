# 指派组别隔离异常

## Bug 是什么
两组人员并发指派时，共享查询状态会造成错误拒绝或跨组成功。该题只交付诊断结论，不修改生产代码。

## 如何触发
同时发起属于不同处理组的指派请求，并让两次组别查询重叠；其中一个请求覆盖另一个请求的可变查询结果后即可稳定观察到污染。

## 根因
涉及 internal/service/assign_service.go、internal/repository/user_repo.go、internal/model/user.go、internal/model/ticket.go。相关符号是 Assign、FindByGroup、User.Group、Ticket.AssignmentGroup。并发竞态来自仓储复用可变共享对象保存查询结果，组别状态跨请求泄漏到权限判断和写入。

## 运行指令
```text
go test -v ./internal/service -run '^TestConcurrentAssignmentsKeepGroupsIsolated$' -race -count=20
```

## 错误信息
```text
IT handler was read as group
```

## 错误堆栈
```text
--- FAIL: TestConcurrentAssignmentsKeepGroupsIsolated (0.02s)
internal/service/concurrent_assignment_verification_test.go:58
```
