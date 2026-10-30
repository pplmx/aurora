# REST API 参考

`aurora-api` 是独立的 REST 服务二进制（`./aurora` 不带监听端口）。它只服务
`/api/v1`、健康检查与指标接口，没有静态页面；桌面端通过进程内调用复用同一套
应用层，因此下面的路径与桌面端窗口看到的行为一致。

```bash
just api                                          # 构建 ./aurora-api（注入 git 版本）
AURORA_API_KEY="replace-with-at-least-32-random-characters" ./aurora-api
./aurora-api --host 127.0.0.1 --port 9090
./aurora-api --config /etc/aurora.toml            # 指定配置文件
AURORA_SERVER_HOST=127.0.0.1 AURORA_SERVER_PORT=9090 ./aurora-api
```

监听地址优先级：**flag（`-H`/`-p`） > `AURORA_SERVER_HOST`/`AURORA_SERVER_PORT` >
`[server]` 配置 > 默认 `0.0.0.0:8080`**。端口必须在 1–65535，越界在启动前拒绝；
未知参数同样在启动前报错并提示用法。

配置文件默认按 `$HOME/aurora.toml` → `./config/aurora.toml` 查找，与 CLI 一致；
`--config` 指定后只读该文件，文件缺失或无法解析会在绑定端口前失败，不会退回默认
配置。

## 鉴权与中间件

- **`X-API-Key`**：密钥来自 `AURORA_API_KEY` 或 `api.key`。开发模式下未设置会
  生成随机密钥并打印在启动日志；生产环境要求至少 32 个不含空白的可见 ASCII
  字符，并拒绝已知弱密钥（`ErrInsecureAPIKey`），可用 `openssl rand -hex 32`
  生成。
  `/healthz`、`/readyz`、`/health`、`/metrics`、`/metrics/oracle` 不需要 key。
- **没有 Web 界面**：网关不提供静态文件，也不注入任何脚本；`/` 与任何未注册路径
  都返回 404，并带与其他错误相同的 JSON 信封（`code` 为 `NOT_FOUND`）。界面只有
  桌面端的原生窗口。
- **请求体上限**：4 MiB，超出返回 `413`，在鉴权之前生效。
- **请求头上限**：64 KiB（包含请求行），超出由 HTTP 服务在路由和鉴权前返回
  `431`；正常 API 调用不需要接近该上限。
- **限流**：默认关闭；`[api.rateLimit]` 可开启，按客户端固定窗口计数，只有
  `trustedProxies` 中的来源才采信 `X-Forwarded-For`（否则无法绕过）。
- **CORS**：默认只允许同源，`[api.cors] allowedOrigins` 加入白名单；预检允许
  `GET`、`POST`、`PUT`、`PATCH`、`DELETE` 与 `OPTIONS`。
- **请求关联**：每个响应都带 `X-Request-ID`。调用方提供的 ID 只接受最长
  128 字符的可见 ASCII，否则服务端会生成一个，避免未鉴权请求把超长 ID
  放大进访问日志；访问日志中的 HTTP 方法和路径也会分别截断到 32/1024
  字节，CORS 白名单同时允许发送和读取该头。
- **禁止缓存**：所有响应带 `Cache-Control: no-store`。API 使用 `X-API-Key`
  而不是 `Authorization`，因此不能依赖 HTTP 缓存对鉴权请求的默认保护；创建
  响应与预言机配置还可能包含一次性私钥或上游请求头。

## 错误响应

失败时返回 `{"error": "...", "code": "..."}`：`code` 是稳定的机器可读值，
`error` 是给人看的说明，措辞可能调整；未被分类的失败返回 `500` 与
`INTERNAL_ERROR`。中间件层另有 `BODY_TOO_LARGE`（413）、`INVALID_REQUEST`
（400）、`NOT_FOUND`（404）与 `RATE_LIMITED`（429）。同一个 `code` 可能
用于多个模块（如 `INVALID_SIGNATURE`），下表按首次出现的模块列出。

| 模块   | code                           | HTTP |
| ------ | ------------------------------ | ---- |
| 代币   | `TOKEN_NOT_FOUND`              | 404  |
| 代币   | `TOKEN_EXISTS`                 | 409  |
| 代币   | `INSUFFICIENT_BALANCE`         | 400  |
| 代币   | `INSUFFICIENT_ALLOWANCE`       | 400  |
| 代币   | `INVALID_SIGNATURE`            | 401  |
| 代币   | `AMOUNT_MUST_BE_POSITIVE`      | 400  |
| 代币   | `NOT_TOKEN_OWNER`              | 403  |
| 代币   | `ZERO_OWNER`                   | 400  |
| 代币   | `TOKEN_NOT_MINTABLE`           | 400  |
| 代币   | `TOKEN_NOT_BURNABLE`           | 400  |
| 代币   | `UNAUTHORIZED`                 | 401  |
| 代币   | `TRANSFER_TO_ZERO`             | 400  |
| 代币   | `INVALID_AMOUNT`               | 400  |
| 代币   | `AMOUNT_TOO_LARGE`             | 400  |
| 代币   | `INVALID_DECIMALS`             | 400  |
| 代币   | `DUPLICATE_TRANSFER`           | 409  |
| 代币   | `TOKEN_NAME_REQUIRED`          | 400  |
| 代币   | `TOKEN_NAME_TOO_LONG`          | 400  |
| 代币   | `TOKEN_SYMBOL_REQUIRED`        | 400  |
| 代币   | `TOKEN_SYMBOL_TOO_LONG`        | 400  |
| 代币   | `PUBLIC_KEY_REQUIRED`          | 400  |
| 代币   | `INVALID_PUBLIC_KEY_LENGTH`    | 400  |
| 代币   | `PRIVATE_KEY_REQUIRED`         | 400  |
| 代币   | `INVALID_PRIVATE_KEY_LENGTH`   | 400  |
| 代币   | `INVALID_BASE64`               | 400  |
| 代币   | `AUDIT_PUBLISH_FAILED`         | 500  |
| NFT    | `NFT_NOT_FOUND`                | 404  |
| NFT    | `NAME_REQUIRED`                | 400  |
| NFT    | `OWNER_REQUIRED`               | 400  |
| NFT    | `NOT_NFT_OWNER`                | 403  |
| NFT    | `KEY_MISMATCH`                 | 401  |
| NFT    | `INVALID_PRIVATE_KEY`          | 400  |
| NFT    | `INVALID_PUBLIC_KEY`           | 400  |
| NFT    | `ZERO_ADDRESS`                 | 400  |
| NFT    | `NAME_TOO_LONG`                | 400  |
| NFT    | `DESCRIPTION_TOO_LONG`         | 400  |
| NFT    | `IMAGE_URL_TOO_LONG`           | 400  |
| NFT    | `TOKEN_URI_TOO_LONG`           | 400  |
| 抽奖   | `LOTTERY_NOT_FOUND`            | 404  |
| 抽奖   | `EMPTY_PARTICIPANT_NAME`       | 400  |
| 抽奖   | `PARTICIPANT_NAME_TOO_LONG`    | 400  |
| 抽奖   | `INVALID_PARTICIPANT_NAME`     | 400  |
| 抽奖   | `SEED_TOO_SHORT`               | 400  |
| 抽奖   | `SEED_TOO_LONG`                | 400  |
| 抽奖   | `NO_PARTICIPANTS`              | 400  |
| 抽奖   | `TOO_MANY_PARTICIPANTS`        | 400  |
| 抽奖   | `DUPLICATE_PARTICIPANT`        | 400  |
| 抽奖   | `WINNER_COUNT_NOT_POSITIVE`    | 400  |
| 抽奖   | `TOO_MANY_WINNERS`             | 400  |
| 抽奖   | `WINNERS_EXCEED_PARTICIPANTS`  | 400  |
| 预言机 | `INVALID_SOURCE`               | 400  |
| 预言机 | `SOURCE_NOT_FOUND`             | 404  |
| 预言机 | `SOURCE_DISABLED`              | 400  |
| 预言机 | `RATE_LIMITED`                 | 429  |
| 投票   | `SESSION_NOT_FOUND`            | 404  |
| 投票   | `SESSION_NOT_STARTED`          | 400  |
| 投票   | `SESSION_ENDED`                | 400  |
| 投票   | `VOTER_NOT_REGISTERED`         | 404  |
| 投票   | `CANDIDATE_NOT_FOUND`          | 400  |
| 投票   | `CANDIDATE_NOT_IN_SESSION`     | 400  |
| 投票   | `ALREADY_VOTED`                | 409  |
| 投票   | `INVALID_VOTE_SIGNATURE`       | 401  |
| 投票   | `VOTER_NAME_REQUIRED`          | 400  |
| 投票   | `CANDIDATE_NAME_REQUIRED`      | 400  |
| 投票   | `SESSION_TITLE_REQUIRED`       | 400  |
| 投票   | `INVALID_SESSION_TIME`         | 400  |
| 投票   | `CANDIDATES_REQUIRED`          | 400  |
| 投票   | `DUPLICATE_CANDIDATE`          | 400  |
| 投票   | `VOTER_NAME_TOO_LONG`          | 400  |
| 投票   | `CANDIDATE_NAME_TOO_LONG`      | 400  |
| 投票   | `CANDIDATE_PARTY_TOO_LONG`     | 400  |
| 投票   | `CANDIDATE_PROGRAM_TOO_LONG`   | 400  |
| 投票   | `SESSION_TITLE_TOO_LONG`       | 400  |
| 投票   | `SESSION_DESCRIPTION_TOO_LONG` | 400  |

## 健康与指标

| 方法 | 路径              | 说明                                   |
| ---- | ----------------- | -------------------------------------- |
| GET  | `/healthz`        | 存活检查                               |
| GET  | `/readyz`         | 就绪检查（探测数据库）                 |
| GET  | `/health`         | `/healthz` 的别名                      |
| GET  | `/metrics`        | Prometheus 文本：请求计数与按模块统计  |
| GET  | `/metrics/oracle` | 各预言机源的抓取健康度                 |

`/readyz` 正常时返回 `{"status":"ok","checks":{"database":"ok"}}`；数据库探测
失败时返回 `503` 与 `{"status":"unhealthy","checks":{"database":"fail"}}`。

## 指标

`/metrics` 输出 Prometheus 文本，全部指标带 `aurora_` 前缀：

- `aurora_http_requests_total`；
- `aurora_http_requests_by_status_total{code}`；
- `aurora_http_requests_by_module_total{module}` 与
  `aurora_http_requests_by_module_status_total{module,code}`；
- `aurora_http_request_duration_seconds`（直方图）。

`/metrics/oracle` 输出每个数据源的抓取健康度：
`aurora_oracle_fetch_attempts_total{source}`、
`aurora_oracle_fetch_successes_total{source}`、
`aurora_oracle_fetch_failures_total{source}`；调度器启动前该端点返回空内容。

```text
# HELP aurora_http_requests_total Total HTTP requests handled.
# TYPE aurora_http_requests_total counter
aurora_http_requests_total 3
# TYPE aurora_http_requests_by_status_total counter
aurora_http_requests_by_status_total{code="200"} 2
aurora_http_requests_by_status_total{code="404"} 1
```

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

`GET /lottery/history` 默认返回最新 50 条；`limit` 上限 100，`offset` 上限
10000。分页在 SQLite 查询中完成，并按
`timestamp DESC, block_height DESC, id` 使用复合索引，因此同一秒创建或导入的
记录也有稳定顺序，不会在两次翻页之间重复或漏行。

单条记录（长哈希与密钥以 … 省略）：

```json
{"id":"c2421dfb187c8ee9","block_height":1,"seed":"aurora-e2e-seed",
 "participants":["Alice","Bob","Carol","Dev"],"winners":["Dev","Carol"],
 "winner_addresses":["0x9c24f45a7ea9e4668ee31dc18bd0a9153f1413ce","0xb2dd7d8a70567a0e23308a6a77b38d603eaf2bac"],
 "vrf_proof":"d44a91a9…f4e9f408","vrf_output":"a3101302…9c60c333",
 "vrf_public_key":"/QC9j6g0…LR0Q=","timestamp":1793378358,"verified":true}
```

`GET /lottery/{id}/verify` 返回 `{"id":"c2421dfb187c8ee9","valid":true}`；证明或
中奖集合对不上时 `valid` 为 `false`，并带 `reason` 说明。

## 投票

| 方法 | 路径                                | 说明                                       |
| ---- | ----------------------------------- | ------------------------------------------ |
| POST | `/api/v1/voting/register/voter`     | 注册选民                                   |
| POST | `/api/v1/voting/register/candidate` | 注册候选人                                 |
| POST | `/api/v1/voting/session`            | 创建会期                                   |
| POST | `/api/v1/voting/session/{id}/start` | 开始会期                                   |
| POST | `/api/v1/voting/session/{id}/end`   | 结束会期                                   |
| POST | `/api/v1/voting/vote`               | 投票（Ed25519 签名）                       |
| GET  | `/api/v1/voting/candidates`         | 候选人列表，默认 20，支持 `limit`/`offset` |
| GET  | `/api/v1/voting/sessions`           | 会期列表，默认 20，支持 `limit`/`offset`   |
| GET  | `/api/v1/voting/voters`             | 选民列表，默认 20，支持 `limit`/`offset`   |
| GET  | `/api/v1/voting/session/{id}`       | 单个会期                                   |
| GET  | `/api/v1/voting/results/{id}`       | 计票结果                                   |

创建会期返回 `draft`；只有先调用 `/session/{id}/start` 把状态切到 `active`
且当前时间位于会期窗口内时，`/vote` 才会接收选票。`ended`、未知状态或窗口外
的请求都会以 `SESSION_NOT_STARTED` / `SESSION_ENDED` 拒绝。

`GET /voting/session/{id}` 与 `POST /session/{id}/start`、`/end` 返回与
`/voting/sessions` 完全相同的字段（snake_case），可以用同一个结构解析：

```json
{"id":"95f640cd-cc8e-4178-b5fa-8e9b89073125","title":"Budget 2026",
 "description":"","status":"draft","candidates":["5974ffa1-2d14-4b95-96c0-8d4c2d211f27"],
 "start_time":1793376742,"end_time":1793463142,"created_at":1793376742}
```

每个会期对同一选民只接收一票；同一次投票的 claim、选票记录和候选人计数在同一个
SQLite 事务内提交。上一会期的选票不会阻止该选民参加下一会期。计票只统计每个
`(session_id, voter_pk)` claim 对应的那一张选票；迁移保留的历史重复审计记录不会
重复计入结果。

选民列表只返回姓名、公钥和 `has_voted`，不返回注册时一次性生成的私钥；
`has_voted` 表示该选民是否曾经投过票，不表示当前会期是否已经投过。客户端如果
需要替选民签名，必须使用注册响应里当时保存的私钥。

会期对象带 `start_time` / `end_time`（unix 秒），即选票被接受的时间窗：窗口之外
的投票会被拒绝，调用方据此可以在提交前解释原因。
候选人、选民与会期列表都在存储层分页，`limit` 上限 100，`offset` 上限 10000；
缺失或非法参数使用默认页，超出上限的 `limit`/`offset` 会被截断，不会被解释成
无界读取。

## NFT

| 方法 | 路径                       | 说明                                                  |
| ---- | -------------------------- | ----------------------------------------------------- |
| POST | `/api/v1/nft/mint`         | 铸造（可选 `image`、`token_uri`）                     |
| POST | `/api/v1/nft/transfer`     | 转让（签名）                                          |
| POST | `/api/v1/nft/burn`         | 销毁（签名）                                          |
| GET  | `/api/v1/nft/{id}`         | 单个 NFT，未知 id 404                                 |
| GET  | `/api/v1/nft/{id}/history` | 操作历史，默认 50，支持 `limit`/`offset`；未知 id 404 |
| GET  | `/api/v1/nft/list`         | 按 owner 分页列表                                     |

NFT 操作历史按最新优先从存储层分页，`limit` 上限 100，`offset` 上限 10000。

```json
[{"id":"adcbadaf-a4d8-45dd-98b6-bd421ec6ae18","name":"Star","description":"First asset",
  "image_url":"","token_uri":"","owner":"q83vEjRWeJAq…","creator":"q83vEjRWeJAq…",
  "block_height":2,"timestamp":1793378358}]
```

`/nft/{id}/history` 的每一项：`{"id":"5040267b-a6e9-40f8-b3db-bc27178acac2",
"nft_id":"adcbadaf-…","type":"mint","from":"","to":"q83vEjRWeJAq…",
"block_height":2,"timestamp":1793378358}`。
已销毁 NFT 的审计记录仍保留；读取超出末页会返回 `200 []`，而不是把空页误判为
未知 id。

## 代币

| 方法 | 路径                          | 说明                                                                    |
| ---- | ----------------------------- | ----------------------------------------------------------------------- |
| POST | `/api/v1/token/create`        | 创建代币，`decimals` 0..127                                             |
| POST | `/api/v1/token/mint`          | 增发                                                                    |
| POST | `/api/v1/token/transfer`      | 转账                                                                    |
| POST | `/api/v1/token/approve`       | 授权                                                                    |
| POST | `/api/v1/token/burn`          | 销毁                                                                    |
| POST | `/api/v1/token/transfer_from` | 使用授权转账                                                            |
| GET  | `/api/v1/token/balance`       | 余额（`token_id`、`owner`）                                             |
| GET  | `/api/v1/token/allowance`     | 授权额度（`token_id`、`owner`、`spender`）                              |
| GET  | `/api/v1/token/history`       | 流水，默认页大小 50，支持 `limit`/`offset`；省略 `owner` 读取全代币流水 |
| GET  | `/api/v1/token/info`          | 代币元数据                                                              |
| GET  | `/api/v1/token/list`          | 代币列表，存储层分页；默认 20，支持 `limit`/`offset`                    |

未知 `token_id` 的读接口返回 `404 TOKEN_NOT_FOUND`，与 `info` 一致——不会用
`200 {"amount":"0"}` 或 `200 []` 把“代币不存在”伪装成“没有活动”。
余额、授权和历史中的非空 `owner`/`spender` 必须解码为 32 字节 Ed25519 公钥；
长度错误返回 `400 INVALID_PUBLIC_KEY_LENGTH`，不会被当作空账户读取。
`token/history` 省略 `owner` 时读取整个代币的转账流水。

查询串里的 base64 公钥必须百分号编码：`+` 在查询串中会被解析成空格，原样拼接
会得到 `400 INVALID_BASE64`。用 `curl -G --data-urlencode`，或任何会把 `+`
编成 `%2B`、`/` 编成 `%2F` 的客户端：

```bash
curl -s -G -H "X-API-Key: $KEY" http://localhost:8080/api/v1/token/balance \
  --data-urlencode "token_id=GLD" --data-urlencode "owner=$OWNER"
```

编码后的查询串形如 `?token_id=GLD&owner=%2B%2F%2F%2BAAECAw…%3D`：

```json
{"token_id":"GLD","owner":"+//+AAECAwQFBgcICQoLDA0ODxAREhMUFRYXGBkaGxw=","amount":"0"}
```

## 预言机

| 方法   | 路径                          | 说明                                                |
| ------ | ----------------------------- | --------------------------------------------------- |
| GET    | `/api/v1/oracle/sources`      | 数据源列表，默认 20，支持 `limit`/`offset`          |
| POST   | `/api/v1/oracle/sources`      | 新建数据源（URL/type/method/headers/path/interval） |
| PATCH  | `/api/v1/oracle/sources/{id}` | 启用/停用                                           |
| DELETE | `/api/v1/oracle/sources/{id}` | 删除，未知 id 404                                   |
| GET    | `/api/v1/oracle/templates`    | 内置模板                                            |
| POST   | `/api/v1/oracle/fetch`        | 立即抓取一次；限流时返回 `429 RATE_LIMITED`         |
| GET    | `/api/v1/oracle/query`        | 历史数据；缺少 `source` 参数返回 400                |
| GET    | `/api/v1/oracle/latest`       | 最新数据点                                          |
| GET    | `/api/v1/oracle/health`       | 抓取健康统计                                        |

`/oracle/sources` 在存储层按最新优先分页，`limit` 上限 100，`offset` 上限
10000；调度器仍在内部读取全部数据源，因为它必须检查每一个启用项。
`/oracle/health` 的每项同时带 `source_name`，健康表无需再按当前源码页反查名称。
Oracle 的读取接口是包装对象：`/oracle/sources` 返回 `{"sources": [...]}`，
`/oracle/query` 与 `/oracle/latest` 返回 `{"data": ...}`；其它模块的列表是裸数组，
解析时按各接口的示例处理。

```json
{"sources":[{"id":"20261031003918.499636093-cadc91163ae7584d73f8f2aebe37f239","name":"btc",
 "url":"https://api.example.com/price","type":"custom","method":"GET","headers":"",
 "path":"","interval":60,"enabled":true,"created_at":1793378358}]}
```

没有读数时 `/oracle/query` 返回 `{"data":[]}`，`/oracle/latest` 返回
`{"data":null}`。

`POST /api/v1/oracle/sources` 的 `headers` 接受 JSON 对象，例如
`{"Authorization":"Bearer ...","X-API-Key":"..."}`；`method` 必须是 HTTP token，
请求头名称也必须是 token，值不能包含除 tab 外的控制字符。JSON、CR/LF 和长度
都在写入前校验（`INVALID_SOURCE`）。`Host` 会覆盖实际请求的虚拟主机并在同源
重定向中保留，跨源重定向则清除；`Content-Length`、`Transfer-Encoding` 和
`Trailer` 由 HTTP 传输层派生，写入这些头会返回 `INVALID_SOURCE`。仅大小写不同
的重复头名同样会被拒绝，避免同一配置按 map 遍历顺序产生不同请求。
`Proxy-Authorization`、`Proxy-Authenticate`、`Connection`、`Keep-Alive`、`TE`、
`Upgrade` 等代理或逐跳头也不属于单个数据源，写入会返回 `INVALID_SOURCE`；
代理凭据应放在 `HTTP_PROXY`/`HTTPS_PROXY` 的 URL 中。URL 必须带非空 hostname，
端口写出来时必须在 `1..65535` 且不能是只有冒号的空值。`extractByPath` 允许合法的空字符串
值，但对象/数组/null 叶子会失败而不是把 Go map 存成链上数据。抓取在 URL 校验、
重定向与实际拨号三处拒绝内部地址：回环、RFC1918、link-local（含云元数据）、CGNAT、
保留/基准网段、IPv4/IPv6 组播、TEST-NET 文档网段（`192.0.2.0/24`、
`198.51.100.0/24`、`203.0.113.0/24`、`3fff::/20`），以及会把内嵌 IPv4
翻译进来的 NAT64/6to4 前缀（`64:ff9b::/96`、`64:ff9b:1::/48`、
`2002::/16`）、已废弃的 IPv6 site-local（`fec0::/10`）、IPv4-compatible/
translated 区段，以及 IETF IPv6 协议分配块（`2001::/23`，含 Teredo、ORCHID）
也会被拒绝；公开 IPv4 的 mapped IPv6 形式仍按对应 IPv4 判定。重定向链最多
跟随 10 跳。带方括号且省略端口的 IPv6 字面量会先规范化再判定，不会把公网
IPv6 误判为无法解析。跨源重定向不会继承数据源配置的请求头，避免 `X-API-Key`
等自定义凭据泄漏给第三方主机。

数据源配置的 `User-Agent` 与 `Accept` 按“是否出现”覆盖 Aurora 默认值：显式空
`User-Agent` 表示不发送该头，空 `Accept` 也会保留；其他自定义头原样发送。
`Content-Type` 只在 POST/PUT/PATCH 且为空时补 `application/json`，
`X-Request-ID` 则由客户端为每一跳重新生成。

抓取失败时只返回 HTTP 状态或底层网络原因，不回显完整数据源 URL；userinfo、
path 与 query 都可能包含凭据，因此 `/oracle/health` 的 `last_error` 与调度器
日志不把它们作为诊断上下文。用 `source_id` 关联具体数据源。

## 区块链

| 方法 | 路径                          | 说明                     |
| ---- | ----------------------------- | ------------------------ |
| GET  | `/api/v1/blockchain/verify`   | 校验本地区块链完整性     |

```json
{"valid":true,"length":3,"first_broken_index":-1}
```

`first_broken_index` 为 `-1` 表示整条链通过；否则指向第一处哈希断开的区块下标，
`valid` 为 `false`。

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
`[log]` 控制，默认 `./logs/`（桌面端默认落在数据库旁的 `logs/`）。Unix-like
系统上日志目录固定为 `0700`、`aurora.log` 固定为 `0600`；旧版本留下的宽松
权限会在下一次启动时自动收紧。
