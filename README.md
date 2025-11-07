# NatsumeAI

一个基于 go-zero 的微服务电商，集成了权限认证，CDC，agent，分布式事务等技术。

待完善...

## 文档

[apifox 接口文档](https://u6pteyxjh0.apifox.cn/)

## 前置环境

一台电脑

Go 1.25.2

docker 以及 compose **插件**

## 快速开始

1. 构建项目：

```bash
go mod tidy
chmod +x ./build.sh
./build.sh
```

2. 启动依赖
```bash
make dependency-prep
make dependency
make devops
```

3. 引入 sql 文件，执行 ./tools/ 下的 casbin 脚本

4. 启动程序
```bash
make app
```

## 项目总览

NatsumeAI 是一个基于 go-zero 的微服务电商示例，集成了认证鉴权（JWT + Casbin）、CDC（Canal → Kafka）、搜索/向量检索（Elasticsearch + Embedding）、智能导购 Agent（CloudWeGo Eino + Tool Calling）、分布式事务（DTM）、异步任务（Asynq）等能力。

- 代码语言与版本：Go 1.25.2（`go.mod`）
- 微服务通信：gRPC（zrpc，Consul 注册/发现）
- 网关与路由：Traefik（`manifest/deploy/dependency.yaml`）
- 存储与中间件：MySQL、Redis、Kafka、Elasticsearch、Asynq（独立 Redis）
- 事务与 CDC：DTM（TCC/SAGA）、Canal（MySQL → Kafka）
- 权限与认证：Auth 服务 + go-jwt、Casbin（DB 持久化 + Redis Watcher）
- 智能导购：Agent 服务调用大模型进行意图识别与工具调用，实现电商推荐对话

## 目录结构

顶层关键目录与文件：

- `app/api/*`：各业务的 HTTP API（go-zero rest），含 `*.api` 定义与 `etc/*.yaml` 配置
- `app/services/*`：各业务的 RPC 服务（zrpc，gRPC/Protobuf + Consul 注册）
- `app/dal/*`：数据访问层（goctl model 由 `manifest/sql/*/*.sql` 生成）
- `app/common/*`：通用常量、错误码、Middlewares、工具方法
- `manifest/deploy/*.yaml`：Docker Compose 编排（依赖、应用、DevOps 工具）
- `manifest/sql/*/*.sql`：各领域初始化 SQL
- `manifest/casbin/*`：Casbin 模型与策略（CSV）
- `tools/casbinimport`：导入 Casbin 策略到 MySQL 的小工具
- `build.sh`：一键构建所有 API/RPC 可执行文件
- `Makefile`：goctl 生成、Compose 一键启动

## 核心服务与端口

HTTP API（见各自 `etc/*.yaml`）：

- user-api：`0.0.0.0:10001`（`app/api/user/etc/user-api.yaml`）
- inventory-api：`0.0.0.0:10002`（`app/api/inventory/etc/inventory-api.yaml`）
- product-api：`0.0.0.0:10003`（`app/api/product/etc/product-api.yaml`）
- cart-api：`0.0.0.0:10004`（`app/api/cart/etc/cart-api.yaml`）
- order-api：`0.0.0.0:10005`（`app/api/order/etc/order-api.yaml`）
- coupon-api：`0.0.0.0:10006`（`app/api/coupon/etc/coupon-api.yaml`）
- agent-api：`0.0.0.0:10007`（`app/api/agent/etc/agent-api.yaml`）

RPC 服务（容器内监听端口，Consul 注册，见各自 `etc/*.yaml`）：

- auth.rpc：`12000`（JWT 颁发/校验/刷新）
- user.rpc：`12001`
- inventory.rpc：`12002`
- product.rpc：`12003`
- cart.rpc：`12004`
- order.rpc：`12005`
- coupon.rpc：`12006`
- payment.rpc：`12006`（与 coupon.rpc 端口数值一致但位于不同容器，不冲突）
- agent.rpc：`12007`
- indexer：无 gRPC，对 Kafka/ES 工作

辅助面板与工具：

- Consul UI：`http://localhost:8500`
- Traefik Dashboard：`http://localhost:8081`
- Kafka UI：`http://localhost:58080`
- Asynqmon：`http://localhost:8083`

提示：Compose 使用外部网络 `application`，若首次启动失败，请先创建：

```bash
docker network create application
```

## 运行依赖与数据流

- 服务发现：Consul（注册键如 `user.rpc`、`product.rpc` 等）
- 关系数据库：MySQL（`manifest/deploy/dependency.yaml` 配置为 `Natsume` 数据库）
- 缓存与队列：Redis（业务）+ Redis（Asynq 专用）
- 事件总线：Kafka（Canal → Kafka → Indexer）
- 全文/向量检索：Elasticsearch（索引名默认 `products`）
- 分布式事务：DTM（`dtm` 容器，go-zero 驱动）

CDC → 搜索链路：

1. MySQL 表 `Natsume.product*` 由 Canal 订阅（`manifest/sql/canal/canal.sql`、`dependency.yaml` 环境变量）。
2. Canal 将变更写入 Kafka Topic（如 `natsume_products`、`natsume_product_categories`）。
3. Indexer（`app/services/indexer`）消费 Kafka，生成/更新 ES 文档；若配置了 Embedding，则构建向量并写入 `dense_vector` 字段（`EmbeddingDimension` 默认 2048）。
4. product-api / product-rpc 支持关键字/类目/销量排序等检索；Agent 的工具调用也走此链路。

## 智能导购 Agent（agent-api / agent.rpc）

- API：`POST /api/v1/agent`（`app/api/agent/agent.api`），入参 `query` 文本，返回自然语言 `answer` 与 `items` 推荐列表。
- 流程：
  - 意图识别：使用 CloudWeGo Eino 绑定工具的对话模型，对用户 `query` 做意图分类（`recommend|no_need|outfit` 等），结构化预算/类目/关键词（`app/services/agent/internal/agent/intent/classifier.go`）。
  - 工具调用：模型触发 `search_products` 工具（`app/services/agent/internal/agent/tools`），由执行器转发到 `product.rpc` 的 `SearchProducts`（`executor.go`）。
  - 多轮工具：遵循约束（避免重复、最多 3 次调用、参数差异化），聚合多批结果（`chat/workflow.go`）。
  - 结果生成：严格基于工具 JSON 结果生成总结回答，拒绝幻觉，必要时返回无结果提示。
- 配置：
  - ChatModel 与 Embedding 基于火山方舟 ARK（`app/services/agent/etc/agent.yaml`），请以自有 Key/Model 覆盖默认示例值。

## 鉴权与权限（Auth + Casbin）

- 认证：
  - `auth.rpc` 提供 `GenerateToken/ValidateToken/RefreshToken`（`app/services/auth/auth.proto`）。
  - API 侧通过通用中间件注入用户信息：`app/common/middleware/authmiddleware.go`，读取 Cookie/Header：`access_token`、`refresh_token`，自动刷新并下发新 Cookie。
- 授权：
  - Casbin 模型：`manifest/casbin/model.conf`（`keyMatch2` 支持 REST 风格）。
  - 策略示例：`manifest/casbin/policy.csv`（`user`、`merchant` 等角色）。
  - 导入工具：`tools/casbinimport`，示例：

```bash
go run ./tools/casbinimport \
  -dsn    "root:***@tcp(localhost:3306)/Natsume?charset=utf8mb4&parseTime=True&loc=Local" \
  -model  "manifest/casbin/model.conf" \
  -policy "manifest/casbin/policy.csv" \
  -truncate=true
```

## 配置说明（通用字段）

所有服务的 `etc/*.yaml` 基本包含：

- `Name`/`ListenOn`：服务名与监听地址（RPC 服务）
- `Consul`：注册中心 `Host`/`Key`/`Meta`/`Tag`
- `MysqlConf.datasource`：MySQL 连接串（统一库 `Natsume`）
- `CacheConf`、`RedisConf`：go-zero 缓存与 Redis 直连配置
- `KafkaConf`：`Broker`/`Group`/`Topic` 等
- `AsynqConf` 与 `AsynqServerConf`：异步任务（订单、支付）
- `DtmConf`：DTM 服务器地址与业务回调
- `LogConf.Level`：`info`/`error` 等
- 产品相关：`ElasticConf`（地址、索引、维度）、`Embedding`（模型与 Key）
- Agent 相关：`ChatModel`、`Embedding`、`Rerank` 等

安全提示：仓库中的 Key 值仅为演示用途，请务必在本地/生产环境覆盖为自己的密钥，并避免提交到版本库。

## 数据库与模型

- 初始化 SQL：见 `manifest/sql/*/*.sql`，例如：
  - 商品与类目：`manifest/sql/product/product.sql`
  - 订单：`manifest/sql/order/order.sql`
  - 用户与地址/商家：`manifest/sql/user/user.sql`
  - 购物车/库存/券/支付：对应子目录内
- DAL 生成：`make model`（基于 goctl + DDL）会输出到 `app/dal/<domain>`

## API 概览（节选）

- 用户（`app/api/user/user.api`）
  - 注册/登录；地址 CRUD；商家申请/查询
- 商品（`app/api/product/product.api`）
  - 详情/Feed/搜索；商家创建/更新/删除/加类目
- 购物车（`app/api/cart/cart.api`）
  - 列表、增、改、删
- 订单（`app/api/order/order.api`）
  - 预下单、下单、取消、详情、列表
- 库存（`app/api/inventory/inventory.api`）
  - 查询、调整（商家）
- 优惠券（`app/api/coupon/coupon.api`）
  - 领取、我的券；发布（管理端）
- Agent（`app/api/agent/agent.api`）
  - 对话推荐：`POST /api/v1/agent`

完整 API 文档见 Apifox（见上文链接）。

## 构建与开发

- 一键构建：`./build.sh`（支持指定目标与 `GOOS/GOARCH/OUTPUT_DIR`）
- 生成代码：
  - API：`make api` 或 `make api-<module>`（goctl api）
  - RPC：`make rpc` 或 `make rpc-<module>`（protoc + zrpc）
  - Model：`make model` 或 `make model-<domain>`（基于 `manifest/sql`）
- 启动顺序建议：`make dependency-prep` → `make dependency` → 导入 SQL/Casbin → `make devops`（可选）→ `make app`

## 常见问题（FAQ）

- 外部网络不存在：执行 `docker network create application`
- ES 索引报向量映射不兼容：调整 `EmbeddingDimension` 或清理索引后由 Indexer 重新创建（`app/services/indexer/internal/es/index_initializer.go`）。
- 访问 9999 网关：Traefik 已将各 API 路由按前缀暴露（例如 `/api/v1/products` 指向 product-api）。
- Agent 返回“未调用工具”：检查 `agent.yaml` 中的 `ChatModel` 与 `ProductRpc` 是否正确配置并可用。
