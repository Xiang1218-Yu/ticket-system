# 内部工单流转与处理进度追踪系统

基于 **Go + Gin + GORM + SQLite** 实现的轻量级内部工单管理系统，适用于 IT 报修、行政采购、人事申请、后勤维修等服务请求场景。

员工可提交、跟踪并评价工单；处理组组长可将工单指派给组内成员；处理人可推进状态、补充处理备注；管理员可查看全量数据及统计看板。

## 功能概览

- **用户与认证**：JWT 登录认证；支持普通员工、处理人、管理员三种角色。
- **工单提交**：支持标题、描述、类型、紧急程度与多附件提交；系统按类型自动分配处理组并生成工单编号。
- **工单处理**：组长或管理员可指派组内处理人；工单按固定流程推进状态并记录系统时间线。
- **工单查询**：提供我提交的、待我处理的、我处理过的、全部工单和超时工单列表；支持分页、状态、类型、紧急程度和关键词筛选。
- **工单详情**：展示提交人、处理人、类型、状态、附件、评价及完整处理时间线。
- **评价反馈**：提交人在工单关闭后可从处理速度、处理质量、沟通体验三个维度进行 1～5 星评价。
- **统计看板**：展示分类统计、状态统计、今日/本周新增、待处理数、超时数、平均处理时长和处理人工作量。
- **超时提醒**：普通工单超过 48 小时、紧急工单超过 12 小时仍未完成时，自动标记为超时；前端以浏览器通知模拟提醒。

## 技术栈

| 层级 | 技术 |
| --- | --- |
| 后端框架 | Gin |
| ORM | GORM |
| 数据库 | SQLite（`glebarez/sqlite`，无需 CGO） |
| 身份认证 | JWT (`github.com/golang-jwt/jwt/v5`) |
| 配置管理 | Viper |
| 日志 | Zap |
| 前端 | 原生 HTML / CSS / JavaScript |

## 目录结构

```text
.
├── api/
│   └── router.go                 # API 路由注册
├── config/
│   └── config.go                 # 配置加载
├── internal/
│   ├── database/                 # SQLite 连接、迁移与初始化数据
│   ├── handler/                  # HTTP 参数解析与响应
│   ├── middleware/               # JWT、CORS、日志中间件
│   ├── model/                    # GORM 数据模型
│   ├── repository/               # 数据访问层
│   └── service/                  # 工单、指派、认证、统计等业务规则
├── web/                          # 前端页面与静态资源
├── uploads/                      # 上传附件目录（运行时生成）
├── config.yaml                   # 应用配置
├── main.go                       # 程序入口
└── go.mod
```

## 环境要求

- Go **1.26** 或更高版本（以 `go.mod` 为准）
- 无需单独安装 SQLite 或配置 CGO

## 快速开始

在项目根目录执行：

```bash
# 安装依赖
# Go 会在构建或测试时自动下载依赖；如需预下载可执行：
go mod download

# 运行测试
go test ./...

# 构建
go build -o ticket-system ./main.go

# 启动服务
./ticket-system
```

服务默认监听 `http://localhost:8080`。启动后可在浏览器打开：

```text
http://localhost:8080
```

> 程序会自动创建/迁移 SQLite 数据库、上传目录和日志目录，并在空数据库中初始化默认分类与演示账号。

## 配置说明

默认配置位于 `config.yaml`：

```yaml
server:
  port: 8080
  mode: debug

database:
  driver: sqlite
  dsn: "ticket.db"

jwt:
  secret: "change-me-in-production-please-use-a-long-random-string"
  expire_hours: 24

log:
  level: info
  dir: "logs"
  filename: "app.log"

upload:
  dir: "uploads"
  max_size_mb: 20
```

生产部署时请至少修改：

1. `jwt.secret`：替换为高强度随机密钥。
2. `server.mode`：改为 `release`。
3. `database.dsn`、`log.dir`、`upload.dir`：配置为持久化且权限受控的位置。

## 默认账号

首次启动、且用户表为空时，会自动创建以下账号。**所有默认账号密码均为 `123456`，仅用于本地演示，部署前务必修改。**

| 用户名 | 姓名 | 角色 | 处理组 | 说明 |
| --- | --- | --- | --- | --- |
| `admin` | 系统管理员 | 管理员 | - | 可查看所有工单、跨组指派与关闭工单 |
| `it_leader` | IT组组长 | 处理人 | IT组 | IT 组组长 |
| `it_staff` | IT组组员 | 处理人 | IT组 | IT 组成员 |
| `xz_leader` | 行政组组长 | 处理人 | 行政组 | 行政组组长 |
| `hr_leader` | 人事组组长 | 处理人 | 人事组 | 人事组组长 |
| `hq_leader` | 后勤组组长 | 处理人 | 后勤组 | 后勤组组长 |
| `staff` | 普通员工 | 普通员工 | - | 普通提交人 |

## 角色与权限

| 操作 | 普通员工 | 处理人 | 组长 | 管理员 |
| --- | :---: | :---: | :---: | :---: |
| 提交工单 | ✓ | ✓ | ✓ | ✓ |
| 查看自己提交的工单 | ✓ | ✓ | ✓ | ✓ |
| 查看所属组工单详情 | - | ✓ | ✓ | ✓ |
| 查看所有工单 | - | - | 所属组 | ✓ |
| 指派处理人 | - | - | 所属组 | ✓ |
| 推进为“处理中” | - | 被指派人 | 被指派人 | ✓ |
| 推进为“已完成” | - | 被指派人 | 所属组 | ✓ |
| 关闭工单 | 提交人 | 提交人 | 提交人 | ✓ |
| 添加备注/附件 | 自己提交的工单 | 所属组工单 | 所属组工单 | ✓ |
| 提交评价 | 自己关闭的工单 | 自己关闭的工单 | 自己关闭的工单 | 自己关闭的工单 |

## 工单类型与自动分组

| 工单类型 | 自动分配处理组 |
| --- | --- |
| IT报修 | IT组 |
| 行政采购 | 行政组 |
| 人事申请 | 人事组 |
| 后勤维修 | 后勤组 |
| 其他 | IT组 |

## 状态流转与超时规则

工单状态只能按以下顺序单向流转：

```text
待处理（pending） → 处理中（processing） → 已完成（done） → 已关闭（closed）
```

- 提交工单后初始状态为 `pending`。
- 被指派处理人可将工单更新为 `processing`。
- 被指派处理人、所属组组长或管理员可将工单更新为 `done`。
- 提交人或管理员可将工单更新为 `closed`。
- `done`、`closed` 状态不计入超时。
- `normal` 工单超过 **48 小时**未完成即超时。
- `urgent` 工单超过 **12 小时**未完成即超时。

## API 概览

除注册和登录外，所有接口均需携带：

```http
Authorization: Bearer <JWT_TOKEN>
```

成功响应统一为：

```json
{
  "code": 0,
  "message": "ok",
  "data": {}
}
```

### 认证

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `POST` | `/api/v1/auth/register` | 注册普通员工或处理人 |
| `POST` | `/api/v1/auth/login` | 登录并获取 JWT |
| `GET` | `/api/v1/auth/me` | 获取当前用户 |

登录示例：

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"staff","password":"123456"}'
```

### 工单

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `GET` | `/api/v1/categories` | 获取工单类型 |
| `POST` | `/api/v1/tickets` | 创建工单（`multipart/form-data`） |
| `GET` | `/api/v1/tickets` | 全部工单；管理员查看全部，组长仅查看所属组 |
| `GET` | `/api/v1/tickets/my` | 我提交的工单 |
| `GET` | `/api/v1/tickets/pending` | 待我处理的工单 |
| `GET` | `/api/v1/tickets/processed` | 我处理过的工单 |
| `GET` | `/api/v1/tickets/overdue` | 超时工单；管理员查看全部，组长仅查看所属组 |
| `GET` | `/api/v1/tickets/:id` | 工单详情 |
| `PUT` | `/api/v1/tickets/:id` | 更新标题或描述 |
| `PUT` | `/api/v1/tickets/:id/status` | 更新工单状态 |
| `GET` | `/api/v1/users/handlers?group=<处理组>` | 获取指定组处理人 |
| `PUT` | `/api/v1/tickets/:id/assign` | 指派处理人 |
| `POST` | `/api/v1/tickets/:id/comments` | 添加处理备注 |
| `POST` | `/api/v1/tickets/:id/attachments` | 补传单个附件 |
| `GET` | `/api/v1/attachments/:id` | 下载附件 |
| `POST` | `/api/v1/tickets/:id/review` | 提交评价 |
| `GET` | `/api/v1/stats` | 获取统计看板 |

### 创建工单

请求为 `multipart/form-data`，字段如下：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | :---: | --- |
| `title` | string | ✓ | 标题，最多 200 字符 |
| `description` | string | - | 描述，最多 10000 字符 |
| `category_id` | integer | ✓ | 工单类型 ID |
| `urgency` | string | - | `normal` 或 `urgent`，默认 `normal` |
| `attachments` | file[] | - | 附件；每个文件最大 20 MB |

示例：

```bash
curl -X POST http://localhost:8080/api/v1/tickets \
  -H "Authorization: Bearer <JWT_TOKEN>" \
  -F 'title=办公电脑无法开机' \
  -F 'description=按下电源键后屏幕无显示。' \
  -F 'category_id=1' \
  -F 'urgency=urgent' \
  -F 'attachments=@./screenshot.png'
```

### 列表筛选参数

以下参数可用于工单列表接口：

| 参数 | 说明 |
| --- | --- |
| `page` | 页码，默认 `1` |
| `page_size` | 每页数量，默认 `10`，最大 `100` |
| `status` | `pending`、`processing`、`done`、`closed` |
| `category` | 工单类型名称，例如 `IT报修` |
| `urgency` | `normal` 或 `urgent` |
| `search` | 按工单编号、标题或提交人姓名搜索 |

### 常用 JSON 请求体

更新标题/描述：

```json
{
  "title": "更新后的标题",
  "description": "更新后的描述"
}
```

更新状态：

```json
{
  "status": "processing"
}
```

指派处理人：

```json
{
  "assignee_id": 2
}
```

添加备注：

```json
{
  "content": "已检查网络和电源，正在更换配件。"
}
```

提交评价：

```json
{
  "speed": 5,
  "quality": 5,
  "communicate": 4,
  "comment": "处理及时，沟通清楚。"
}
```

## 前端页面

| 页面 | 路径 |
| --- | --- |
| 首页 | `/` |
| 登录 | `/login` |
| 注册 | `/register` |
| 工单列表 | `/tickets` |
| 新建工单 | `/tickets/new` |
| 工单详情 | `/tickets/detail?id=<id>` |
| 我的工单 | `/my-tickets` |
| 待处理工单 | `/pending` |
| 统计看板 | `/dashboard` |
| 超时工单 | `/overdue` |

## 测试与静态检查

```bash
# 如默认 Go 构建缓存目录不可写，可临时指定可写缓存目录
GOCACHE=/tmp/gocache go test ./...

GOCACHE=/tmp/gocache go vet ./...
```

当前测试覆盖：

- 工单状态合法流转。
- 普通/紧急工单的超时阈值。
- 数据库层超时筛选及分页。
- 处理组范围筛选。

## 安全与部署注意事项

- 不要在生产环境使用默认 JWT 密钥或默认账号密码。
- 建议在反向代理层启用 HTTPS，并设置可信来源的 CORS 策略。
- 附件下载需通过认证和工单访问权限校验；上传文件单文件大小上限为 20 MB。
- SQLite 适合轻量、单机或低并发场景；高并发部署可将仓储层切换至 PostgreSQL 等数据库。
- 请定期备份 SQLite 数据库文件及 `uploads/` 附件目录。
