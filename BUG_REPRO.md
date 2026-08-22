# 附件路径边界异常

## Bug 是什么
特殊名称的附件可能被保存到预期资源根目录之外，下载时仍可能被读取。该题只交付诊断结论，不修改生产代码。

## 如何触发
上传包含路径分隔语义的特殊名称，再通过下载入口读取生成的附件路径；名称清洗、路径保存和访问校验的组合会形成越界结果。

## 根因
涉及 internal/handler/ticket_handler.go、internal/service/ticket_service.go、internal/model/attachment.go、internal/repository/attachment_repo.go。相关符号是 SaveUpload、sanitizeFilename、DownloadAttachment、Attachment.Path。错误传播跨越上传保存和下载访问边界：名称清洗保留了可逃逸部分，最终路径又未被严格限制在资源根目录。

## 运行指令
```text
go test -v ./internal/service -run '^TestSaveUploadKeepsSpecialNamesInsideUploadRoot$' -count=1
```

## 错误信息
```text
attachment escaped upload root
```

## 错误堆栈
```text
--- FAIL: TestSaveUploadKeepsSpecialNamesInsideUploadRoot (0.00s)
internal/service/attachment_path_verification_test.go:36
```
