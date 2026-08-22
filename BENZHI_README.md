# ticket-system

项目用途：基于 **Go + Gin + GORM + SQLite** 实现的轻量级内部工单管理系统，适用于 IT 报修、行政采购、人事申请、后勤维修等服务请求场景。项目源代码、依赖描述和评测专用 Docker 文件共同构成自包含任务；不依赖本机预编译二进制。

## 标准构建、运行和测试命令

```bash
go build ./...
go run .
go test ./...
```
## 评测容器

评测专用 Dockerfile 为 `benzhi.Dockerfile`，构建脚本为 `build_benzhi_docker.sh`。

```bash
chmod +x build_benzhi_docker.sh
./build_benzhi_docker.sh my-go-task linux/arm64
./build_benzhi_docker.sh my-go-task linux/amd64
docker run -it my-go-task:latest
```
