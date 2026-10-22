# REST API 参考

`aurora-api` 是独立的 API + Web 服务二进制（`./aurora` 不带监听端口）。桌面端复用
同一套路由，通过 `aurora://localhost/` 在进程内加载，因此下面的路径对桌面端同样
成立（页面内部调用）。

```bash
just api                                          # 构建 ./aurora-api（注入 git 版本）
AURORA_API_KEY="your-strong-key" ./aurora-api     # 默认 0.0.0.0:8080
./aurora-api --host 127.0.0.1 --port 9090
AURORA_SERVER_HOST=127.0.0.1 AURORA_SERVER_PORT=9090 ./aurora-api
```

监听地址优先级：**flag（`-H`/`-p`） > `AURORA_SERVER_HOST`/`AURORA_SERVER_PORT` >
`[server]` 配置 > 默认 `0.0.0.0:8080`**。端口必须在 1–65535，越界在启动前拒绝；
未知参数同样在启动前报错并提示用法。

## 鉴权与中间件

- **`X-API-Key`**：密钥来自 `AURORA_API_KEY` 或 `api.key`。开发模式下未设置会
  生成随机密钥并打印在启动日志；生产环境拒绝弱密钥（`ErrInsecureAPIKey`）。
  `/healthz`、`/readyz`、`/health`、`/metrics`、`/metrics/oracle` 与静态页面
  不需要 key。
- **Web 页面同源注入**：`window.AURORA_API_KEY` 由带 nonce 的 bootstrap 脚本写入，
  CSP 为 `connect-src 'self'` / `script-src 'self' 'nonce-…'`，页面请求无需手动
  带 header；桌面端剥掉 CSP 但传输层仍是进程内 scheme（无监听、无外部 origin）。
- **请求体上限**：4 MiB，超出返回 `413`，在鉴权之前生效。
- **限流**：默认关闭；`[api.rateLimit]` 可开启，按客户端固定窗口计数，只有
  `trustedProxies` 中的来源才采信 `X-Forwarded-For`（否则无法绕过）。
- **CORS**：默认只允许同源，`[api.cors] allowedOrigins` 加入白名单。

## 健康与指标

| 方法 | 路径              | 说明                                   |
| ---- | ----------------- | -------------------------------------- |
| GET  | `/healthz`        | 存活检查                               |
| GET  | `/readyz`         | 就绪检查（探测数据库）                 |
| GET  | `/health`         | `/healthz` 的别名                      |
| GET  | `/metrics`        | Prometheus 文本：请求计数与按模块统计  |
| GET  | `/metrics/oracle` | 各预言机源的抓取健康度                 |

## 抽奖

| 方法 | 路径                          | 说明                                              |
| ---- | ----------------------------- | ------------------------------------------------- |
| POST | `/api/v1/lottery/create`      | 创建抽奖；`winner_count` 省略时用配置默认值       |
| GET  | `/api/v1/lottery/history`     | 分页历史（`limit`、`offset`）                     |
| GET  | `/api/v1/lottery/{id}`        | 单条记录，未知 id 返回 404                        |
| GET  | `/api/v1/lottery/{id}/verify` | 复验 VRF 证明与中奖集合，返回 `{id,valid,reason}` |

```bash
curl -s -H "X-API-Key: $KEY" -H 'Content-Type: application/json' \
  -d '{"participants":"A,B,C","seed":"demo","winner_count":1}' \
  http://localhost:8080/api/v1/lottery/create
```

## 投票

| 方法 | 路径                                | 说明                     |
| ---- | ----------------------------------- | ------------------------ |
| POST | `/api/v1/voting/register/voter`     | 注册选民                 |
| POST | `/api/v1/voting/register/candidate` | 注册候选人               |
| POST | `/api/v1/voting/session`            | 创建会期                 |
| POST | `/api/v1/voting/session/{id}/start` | 开始会期                 |
| POST | `/api/v1/voting/session/{id}/end`   | 结束会期                 |
| POST | `/api/v1/voting/vote`               | 投票（Ed25519 签名）     |
| GET  | `/api/v1/voting/candidates`         | 候选人列表               |
| GET  | `/api/v1/voting/sessions`           | 会期列表                 |
| GET  | `/api/v1/voting/session/{id}`       | 单个会期                 |
| GET  | `/api/v1/voting/results/{id}`       | 计票结果                 |

## NFT

| 方法 | 路径                       | 说明                              |
| ---- | -------------------------- | --------------------------------- |
| POST | `/api/v1/nft/mint`         | 铸造（可选 `image`、`token_uri`） |
| POST | `/api/v1/nft/transfer`     | 转让（签名）                      |
| POST | `/api/v1/nft/burn`         | 销毁（签名）                      |
| GET  | `/api/v1/nft/{id}`         | 单个 NFT，未知 id 404             |
| GET  | `/api/v1/nft/{id}/history` | 操作历史，未知 id 404             |
| GET  | `/api/v1/nft/list`         | 按 owner 分页列表                 |

## 代币

| 方法 | 路径                          | 说明                                       |
| ---- | ----------------------------- | ------------------------------------------ |
| POST | `/api/v1/token/create`        | 创建代币，`decimals` 0..127                |
| POST | `/api/v1/token/mint`          | 增发                                       |
| POST | `/api/v1/token/transfer`      | 转账                                       |
| POST | `/api/v1/token/approve`       | 授权                                       |
| POST | `/api/v1/token/burn`          | 销毁                                       |
| POST | `/api/v1/token/transfer_from` | 使用授权转账                               |
| GET  | `/api/v1/token/balance`       | 余额（`token_id`、`owner`）                |
| GET  | `/api/v1/token/allowance`     | 授权额度（`token_id`、`owner`、`spender`） |
| GET  | `/api/v1/token/history`       | 流水，默认页大小 50，支持 `limit`/`offset` |
| GET  | `/api/v1/token/info`          | 代币元数据                                 |

未知 `token_id` 的读接口返回 `404 TOKEN_NOT_FOUND`，与 `info` 一致——不会用
`200 {"amount":"0"}` 或 `200 []` 把“代币不存在”伪装成“没有活动”。

## 预言机

| 方法   | 路径                            | 说明                                        |
| ------ | ------------------------------- | ------------------------------------------- |
| GET    | `/api/v1/oracle/sources`        | 数据源列表                                  |
| POST   | `/api/v1/oracle/sources`        | 新建数据源（URL/type/method/path/interval） |
| PATCH  | `/api/v1/oracle/sources/{id}`   | 启用/停用                                   |
| DELETE | `/api/v1/oracle/sources/{id}`   | 删除，未知 id 404                           |
| GET    | `/api/v1/oracle/templates`      | 内置模板                                    |
| POST   | `/api/v1/oracle/fetch`          | 立即抓取一次；限流时返回 `429 RATE_LIMITED` |
| GET    | `/api/v1/oracle/query`          | 历史数据；缺少 `source` 参数返回 400        |
| GET    | `/api/v1/oracle/latest`         | 最新数据点                                  |
| GET    | `/api/v1/oracle/health`         | 抓取健康统计                                |

数据源的 URL 中的请求头/来源在写入前校验；CR/LF 会被拒绝（`INVALID_SOURCE`），
`extractByPath` 允许合法的空字符串值，但对象/数组/null 叶子会失败而不是把 Go
map 存成链上数据。

## 区块链

| 方法 | 路径                          | 说明                     |
| ---- | ----------------------------- | ------------------------ |
| GET  | `/api/v1/blockchain/verify`   | 校验本地区块链完整性     |

## 错误格式

```json
{"error": "not found", "code": "NOT_FOUND"}
```

- 客户端错误（参数缺失、密钥长度不对、越界）→ 4xx；服务端故障 → 5xx。
- 未授权返回 401，请求体超限返回 413，限流返回 429。
- 未知资源（NFT id、预言机 source、token_id）统一 404，不用空数组冒充成功。

## 生命周期与可观测性

服务启动时同时启动预言机调度器（按每个源的 `interval`）与审计事件 outbox
补偿循环；收到 SIGINT/SIGTERM 后 15s 内优雅停机并释放 SQLite 句柄。日志路径由
`[log]` 控制，默认 `./logs/`（桌面端默认落在数据库旁的 `logs/`）。
