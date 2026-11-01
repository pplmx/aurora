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

## 端点索引

全部路由按模块分组；列表只给方法和路径，参数、请求体、响应与错误码见后面对应小节。
带 key 的端点都需要 `X-API-Key`；三个健康检查与两个指标端点不需要。

**健康与指标**

- `GET /health`
- `GET /healthz`
- `GET /metrics`
- `GET /metrics/oracle`
- `GET /readyz`

**抽奖**

- `POST /api/v1/lottery/create`
- `GET /api/v1/lottery/history`
- `GET /api/v1/lottery/{id}`
- `GET /api/v1/lottery/{id}/verify`

**投票**

- `GET /api/v1/voting/candidates`
- `POST /api/v1/voting/register/candidate`
- `POST /api/v1/voting/register/voter`
- `GET /api/v1/voting/results/{id}`
- `POST /api/v1/voting/session`
- `GET /api/v1/voting/session/{id}`
- `POST /api/v1/voting/session/{id}/end`
- `POST /api/v1/voting/session/{id}/start`
- `GET /api/v1/voting/sessions`
- `POST /api/v1/voting/vote`
- `GET /api/v1/voting/voters`

**NFT**

- `POST /api/v1/nft/burn`
- `GET /api/v1/nft/list`
- `POST /api/v1/nft/mint`
- `POST /api/v1/nft/transfer`
- `GET /api/v1/nft/{id}`
- `GET /api/v1/nft/{id}/history`

**代币**

- `GET /api/v1/token/allowance`
- `POST /api/v1/token/approve`
- `GET /api/v1/token/balance`
- `POST /api/v1/token/burn`
- `POST /api/v1/token/create`
- `GET /api/v1/token/history`
- `GET /api/v1/token/info`
- `GET /api/v1/token/list`
- `POST /api/v1/token/mint`
- `POST /api/v1/token/transfer`
- `POST /api/v1/token/transfer_from`

**预言机**

- `POST /api/v1/oracle/fetch`
- `GET /api/v1/oracle/health`
- `GET /api/v1/oracle/latest`
- `GET /api/v1/oracle/query`
- `GET /api/v1/oracle/sources`
- `POST /api/v1/oracle/sources`
- `DELETE /api/v1/oracle/sources/{id}`
- `PATCH /api/v1/oracle/sources/{id}`
- `GET /api/v1/oracle/templates`

**区块链**

- `GET /api/v1/blockchain/blocks`
- `GET /api/v1/blockchain/verify`

## 鉴权与中间件

- **`X-API-Key`**：密钥来自 `AURORA_API_KEY` 或 `api.key`。开发模式下未设置会
  生成随机密钥并打印在启动日志；生产环境要求至少 32 个不含空白的可见 ASCII
  字符，并拒绝已知弱密钥（`ErrInsecureAPIKey`），可用 `openssl rand -hex 32`
  生成。
  `/healthz`、`/readyz`、`/health`、`/metrics`、`/metrics/oracle` 不需要 key。
- **没有 Web 界面**：网关不提供静态文件，也不注入任何脚本；`/` 与任何未注册路径
  都返回 404，并带与其他错误相同的 JSON 信封（`code` 为 `NOT_FOUND`）。界面只有
  桌面端的原生窗口。
下列上限与开关作用在每个请求上，除注明外都在鉴权之前生效：

| 项     | 默认值             | 超出或未开启时                                                                                           |
| ------ | ------------------ | -------------------------------------------------------------------------------------------------------- |
| 请求体 | 4 MiB              | `413`，在鉴权之前判定                                                                                    |
| 请求头 | 64 KiB（含请求行） | `431`，由 HTTP 服务在路由和鉴权前返回                                                                    |
| 限流   | 关闭               | 由 `[api.rateLimit]` 开启，按客户端固定窗口计数；只有 `trustedProxies` 中的来源才采信 `X-Forwarded-For`  |
| CORS   | 只允许同源         | 由 `[api.cors] allowedOrigins` 加入白名单；预检允许 `GET`、`POST`、`PUT`、`PATCH`、`DELETE` 与 `OPTIONS` |

正常的 API 调用不需要接近请求头上限。

- **请求关联**：每个响应都带 `X-Request-ID`。调用方提供的 ID 只接受最长
  128 字符的可见 ASCII，否则服务端会生成一个，避免未鉴权请求把超长 ID
  放大进访问日志；访问日志中的 HTTP 方法和路径也会分别截断到 32/1024
  字节，CORS 白名单同时允许发送和读取该头。
- **禁止缓存**：所有响应带 `Cache-Control: no-store`。API 使用 `X-API-Key`
  而不是 `Authorization`，因此不能依赖 HTTP 缓存对鉴权请求的默认保护；创建
  响应与预言机配置还可能包含一次性私钥或上游请求头。
- **浏览器加固**：所有响应（含 `/healthz`、`/metrics` 与 404）同样带
  `X-Content-Type-Options: nosniff`、`X-Frame-Options: DENY` 和
  `Referrer-Policy: no-referrer`。

## 错误响应

失败时返回一个文档：

```json
{"error": "not found", "code": "NOT_FOUND"}
```

`code` 是稳定的机器可读值，脚本按它分支；`error` 是给人看的说明，措辞可能
调整，不要解析。未被分类的失败返回 `500` 与 `INTERNAL_ERROR`。

状态码约定：

- 客户端错误（参数缺失、密钥长度不对、越界）→ 4xx；服务端故障 → 5xx。
- 未授权返回 401，请求体超限返回 413，限流返回 429。
- 未知资源（NFT id、预言机 source、token_id）统一 404，不用空数组冒充成功。

中间件层另有 `BODY_TOO_LARGE`（413）、`INVALID_REQUEST`（400）、
`NOT_FOUND`（404）、`METHOD_NOT_ALLOWED`（405）与 `RATE_LIMITED`（429）。

同一个 `code` 可能用于多个模块（如 `INVALID_SIGNATURE`），下表按首次出现的模块
列出。

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
| 预言机 | `FETCH_FAILED`                 | 502  |
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

`NOT_TOKEN_OWNER`（403）、`DUPLICATE_TRANSFER`（409）与 `INVALID_SIGNATURE`
（401）在上表中有定义，服务端当前不会产生：前两个哨兵还没有用例返回它们，
`INVALID_SIGNATURE` 则因为 token 与 nft 把验签结果表达成布尔值而不是错误
（`NFTService.VerifyTransfer` 返回 `(false, nil)`），只有投票用它。列出是为了
让调用方在它们启用时不必改解析逻辑；按「已返回」处理会写出永远不触发的分支。
`internal/api/handler/classification_test.go` 会检查这一点：表里每一项都必须有
代码返回其哨兵，或登记在上面的保留名单里。

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
  `aurora_http_requests_by_module_status_total{module,code}`；`module` 只取固定路由
  集合（`token`/`nft`/`voting`/`lottery`/`oracle`/`blockchain`），未知前缀一律计入
  `other`，所以随便请求 `/api/v1/<任意词>/…` 也不会让标签基数增长——实测一次
  `/api/v1/zzz-attacker-1/x` 得到 `module="other"`；
- `aurora_http_request_duration_seconds`（直方图）；
- `aurora_outbox_pending_events`（gauge）：审计 outbox 中等待重投的事件数，
  与 `aurora doctor` 的 outbox 行同源；统计失败时本次抓取省略该行，不写 0；
- `aurora_outbox_oldest_pending_seconds`（gauge）：outbox 中最老的待投递事件已等待
  的秒数——数量在繁忙节点上可能正常，只有这个年龄持续增长才说明补偿循环卡住；
  空 outbox 或读取失败时省略该行；
- `aurora_ledger_blocks`（gauge）：本地账本区块数（含创世块），随每次抓取更新；
- `aurora_build_info{go,version[,build_time]}`：构建身份（恒为 1），用来看某个实例
  跑的是哪个版本；链接器未注入构建时间时省略该标签。

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

| 方法 | 路径                          | 说明                                        |
| ---- | ----------------------------- | ------------------------------------------- |
| POST | `/api/v1/lottery/create`      | 创建抽奖；`winner_count` 省略时用配置默认值 |
| GET  | `/api/v1/lottery/history`     | 分页历史（`limit`、`offset`、`q` 过滤）     |
| GET  | `/api/v1/lottery/{id}`        | 单条记录，未知 id 返回 404                  |
| GET  | `/api/v1/lottery/{id}/verify` | 复验 VRF 证明与中奖集合，返回完整校验报告   |

```bash
curl -s -H "X-API-Key: $KEY" -H 'Content-Type: application/json' \
  -d '{"participants":"A,B,C","seed":"demo","winner_count":1}' \
  http://localhost:8080/api/v1/lottery/create
```

`GET /lottery/history` 默认返回最新 50 条；`limit` 上限 100，`offset` 上限
10000。分页在 SQLite 查询中完成，并按
`timestamp DESC, block_height DESC, id` 使用复合索引，因此同一秒创建或导入的
记录也有稳定顺序，不会在两次翻页之间重复或漏行。`q` 在 id、seed、参与者和
中奖者上做不区分大小写的子串匹配（`%` 与 `_` 按字面匹配，不当通配符），过滤与
分页都在存储层完成——先在当前页过滤只能返回恰好落在该页的匹配。命中集内同样按
上面的顺序分页：

```json
[{"id":"c2421dfb187c8ee9","seed":"aurora-e2e-seed","winners":["Dev","Carol"],"block_height":1},
 {"id":"2f7d1a90c4e8b6d2","seed":"dev-draw","winners":["Dev"],"block_height":4}]
```

单条记录（长哈希与密钥以 … 省略）：

```json
{"id":"c2421dfb187c8ee9","block_height":1,"seed":"aurora-e2e-seed",
 "participants":["Alice","Bob","Carol","Dev"],"winners":["Dev","Carol"],
 "winner_addresses":["0x9c24f45a7ea9e4668ee31dc18bd0a9153f1413ce","0xb2dd7d8a70567a0e23308a6a77b38d603eaf2bac"],
 "vrf_proof":"d44a91a9…f4e9f408","vrf_output":"a3101302…9c60c333",
 "vrf_public_key":"/QC9j6g0…LR0Q=","timestamp":1793378358,"verified":true}
```

`GET /lottery/{id}/verify` 返回完整校验报告：
`{"record":{"id":"c2421dfb187c8ee9",…},"checks":[{"status":"ok","message":"…"}],"integrity_ok":true,"proof_verified":true}`。
`{id}` 按下面的顺序解析，与 CLI `lottery verify` 一致：

| 顺序 | 形式              | 说明                             |
| ---- | ----------------- | -------------------------------- |
| 1    | 精确抽奖 ID       | 命中即返回                       |
| 2    | 整数区块高度      | 同一高度有多条时取其中最新的一条 |
| 3    | ID 或 seed 的子串 | 多个命中时取最新的一条           |

判定失败的依据与 HTTP 状态无关：证明或中奖集合对不上时**仍返回 200**，
`integrity_ok` 为 `false`，`checks` 里 `status:"fail"` 的条目说明原因。脚本应读
`integrity_ok`，而不是只看状态码。

## 投票

| 方法 | 路径                                | 说明                                                     |
| ---- | ----------------------------------- | -------------------------------------------------------- |
| POST | `/api/v1/voting/register/voter`     | 注册选民                                                 |
| POST | `/api/v1/voting/register/candidate` | 注册候选人                                               |
| POST | `/api/v1/voting/session`            | 创建会期                                                 |
| POST | `/api/v1/voting/session/{id}/start` | 开始会期                                                 |
| POST | `/api/v1/voting/session/{id}/end`   | 结束会期                                                 |
| POST | `/api/v1/voting/vote`               | 投票（Ed25519 签名），回执带 `session_id`/`candidate_id` |
| GET  | `/api/v1/voting/candidates`         | 候选人列表，默认 20，支持 `limit`/`offset`               |
| GET  | `/api/v1/voting/sessions`           | 会期列表，默认 20，支持 `limit`/`offset`                 |
| GET  | `/api/v1/voting/voters`             | 选民列表，默认 20，支持 `limit`/`offset`                 |
| GET  | `/api/v1/voting/session/{id}`       | 单个会期                                                 |
| GET  | `/api/v1/voting/results/{id}`       | 计票结果                                                 |

创建会期返回 `draft`。`/vote` 同时检查状态与时间窗口——只满足其中一项不够，
因为新会期是草稿，而操作员也可以在名义结束时间之前手动结束：

| 判定顺序 | 情形                                              | 返回                  |
| -------- | ------------------------------------------------- | --------------------- |
| 1        | 会期不存在                                        | `SESSION_NOT_FOUND`   |
| 2        | 状态已是 `ended`                                  | `SESSION_ENDED`       |
| 3        | 早于 `start_time`                                 | `SESSION_NOT_STARTED` |
| 4        | 晚于 `end_time`                                   | `SESSION_ENDED`       |
| 5        | 状态不是 `active`（含 `draft`，即使时间在窗口内） | `SESSION_NOT_STARTED` |

第 5 行是容易误解的一处：草稿态会期**在窗口内也不接受选票**，必须先调用
`/session/{id}/start`。

投票写接口的请求体字段：

| 端点                        | 字段                                                                                                                                   |
| --------------------------- | -------------------------------------------------------------------------------------------------------------------------------------- |
| `voting/register/voter`     | `name`（非空，≤100 字符）                                                                                                              |
| `voting/register/candidate` | `name`（非空，≤100）、`party`（可选，≤100）、`program`（可选，≤1000）                                                                  |
| `voting/session`            | `title`（非空，≤200）、`description`（可选，≤1000）、`candidate_ids`（非空且不重复）、`start_time`/`end_time`（unix 秒，后者必须更晚） |

会期的**请求字段是 `candidate_ids`**（候选人 id 数组），响应里对应字段叫
`candidates`；两者不是同一个名字，按响应结构回填请求会拿到空名单。

`GET /voting/session/{id}` 与 `POST /voting/session`、`POST /session/{id}/start`、
`/end` 返回与 `/voting/sessions` 完全相同的字段（snake_case），可以用同一个结构
解析：

```json
{"id":"95f640cd-cc8e-4178-b5fa-8e9b89073125","title":"Budget 2026",
 "description":"","status":"draft","candidates":["5974ffa1-2d14-4b95-96c0-8d4c2d211f27"],
 "start_time":1793376742,"end_time":1793463142,"created_at":1793376742}
```

`GET /voting/results/{id}` 的计票结果：

```json
{"session_id":"95f640cd-cc8e-4178-b5fa-8e9b89073125","total_votes":1,
 "candidates":[{"id":"5974ffa1-…","name":"Alice","party":"Greens","vote_count":1},
               {"id":"8a2f19c4-…","name":"Bob","party":"Reds","vote_count":0}]}
```

`total_votes` 等于下列各行 `vote_count` 之和（每个 `(session_id, voter_pk)` claim
计一张）；`candidates` 按会期名单顺序返回，名单里已删除的候选人仍占一行、票数为
0 且其历史票数不再计入——数组顺序与 `session.candidates` 一致而不是按票数排序，
要排名请在客户端排序。

`POST /voting/vote` 的请求体（键均为 base64）：

```json
{"voter_public_key":"<base64 公钥：注册选民的响应>",
 "candidate_id":"5974ffa1-2d14-4b95-96c0-8d4c2d211f27",
 "private_key":"<base64 私钥：注册时保存的那一份>","session_id":"95f640cd-cc8e-4178-b5fa-8e9b89073125"}
```

签名由服务端用请求里的私钥完成（消息是 `voter_public_key|candidate_id|unix 秒`），
再用登记在册的公钥复验：私钥与公私钥对不匹配时返回 `401 INVALID_VOTE_SIGNATURE`，
base64 或长度不合法分别返回 `INVALID_BASE64` / `INVALID_PRIVATE_KEY`（400）。

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

集合端点按创建时间从新到旧排列，并在同一秒内用唯一键（`id` / `public_key`，
或写入顺序）稳定次序，因此同一页在两次请求之间不会换位：代币、候选人、会期、
选民、数据源、NFT 列表及 NFT 操作历史都是如此。预言机读数按时间从新到旧，
同一秒内保持写入顺序，`/oracle/latest` 返回的就是该列表的第一条。

## NFT

| 方法 | 路径                       | 说明                                                        |
| ---- | -------------------------- | ----------------------------------------------------------- |
| POST | `/api/v1/nft/mint`         | 铸造（可选 `image_url`、`token_uri`）                       |
| POST | `/api/v1/nft/transfer`     | 转让（签名）                                                |
| POST | `/api/v1/nft/burn`         | 销毁（签名）                                                |
| GET  | `/api/v1/nft/{id}`         | 单个 NFT，未知 id 404                                       |
| GET  | `/api/v1/nft/{id}/history` | 操作历史，默认 50，支持 `limit`/`offset`；未知 id 404       |
| GET  | `/api/v1/nft/list`         | NFT 列表，可选 `owner` 过滤；默认 20，支持 `limit`/`offset` |

NFT 操作历史按最新优先从存储层分页，`limit` 上限 100，`offset` 上限 10000。
`/nft/list` 省略 `owner` 时列出全部 NFT（新资产在前）；强制 owner 会让没有参与
铸造的调用方只能靠 id 才能读到资产。列表的 `offset` 上限是 1000。带 `owner`
过滤时同样新资产在前，所以开关过滤不会把读者刚看的那一页倒过来。

```json
[{"id":"adcbadaf-a4d8-45dd-98b6-bd421ec6ae18","name":"Star","description":"First asset",
  "image_url":"","token_uri":"","owner":"q83vEjRWeJAq…","creator":"q83vEjRWeJAq…",
  "block_height":2,"timestamp":1793378358}]
```

NFT 写接口的字段：

| 端点           | 字段                                                       |
| -------------- | ---------------------------------------------------------- |
| `nft/mint`     | `name`、`description`、`image_url`、`token_uri`、`creator` |
| `nft/transfer` | `nft_id`、`from`、`to`、`private_key`                      |
| `nft/burn`     | `nft_id`、`owner`、`private_key`                           |

铸造只登记 `creator` 公钥、不要求签名，创建者即首位持有者；转让与销毁由服务端
用请求里的私钥校验：私钥与 `from`/`owner` 公钥不配对返回 `401 KEY_MISMATCH`，
编码或长度不合法返回 `INVALID_PUBLIC_KEY` / `INVALID_PRIVATE_KEY`（400），
公钥配对但已不是当前持有者返回 `403 NOT_NFT_OWNER`。

`/nft/{id}/history` 的每一项：`{"id":"5040267b-a6e9-40f8-b3db-bc27178acac2",
"nft_id":"adcbadaf-…","type":"mint","from":"","to":"q83vEjRWeJAq…",
"block_height":2,"timestamp":1793378358}`。
已销毁 NFT 的审计记录仍保留；读取超出末页会返回 `200 []`，而不是把空页误判为
未知 id。

## 代币

| 方法 | 路径                          | 说明                                                                            |
| ---- | ----------------------------- | ------------------------------------------------------------------------------- |
| POST | `/api/v1/token/create`        | 创建代币，`decimals` 0..127                                                     |
| POST | `/api/v1/token/mint`          | 增发                                                                            |
| POST | `/api/v1/token/transfer`      | 转账                                                                            |
| POST | `/api/v1/token/approve`       | 授权                                                                            |
| POST | `/api/v1/token/burn`          | 销毁                                                                            |
| POST | `/api/v1/token/transfer_from` | 使用授权转账                                                                    |
| GET  | `/api/v1/token/balance`       | 余额（`token_id`、`owner`）                                                     |
| GET  | `/api/v1/token/allowance`     | 授权额度（`token_id`、`owner`、`spender`）                                      |
| GET  | `/api/v1/token/history`       | 流水（`token_id`，省略 `owner` 读取全代币流水），默认 50，支持 `limit`/`offset` |
| GET  | `/api/v1/token/info`          | 代币元数据（`token_id`）                                                        |
| GET  | `/api/v1/token/list`          | 代币列表，存储层分页；默认 20，支持 `limit`/`offset`                            |

写接口的请求体字段（金额是十进制字符串，公钥/私钥是 base64；签名由服务端用
请求里的私钥完成，客户端不必自己签名）：

| 端点                  | 字段                                                                  |
| --------------------- | --------------------------------------------------------------------- |
| `token/create`        | `name`、`symbol`、`total_supply`、`owner`、`decimals`（0..127，可选） |
| `token/mint`          | `token_id`、`to`、`amount`、`private_key`                             |
| `token/transfer`      | `token_id`、`from`、`to`、`amount`、`private_key`                     |
| `token/approve`       | `token_id`、`owner`、`spender`、`amount`、`private_key`               |
| `token/transfer_from` | `token_id`、`owner`、`to`、`amount`、`spender`、`spender_key`         |
| `token/burn`          | `token_id`、`from`、`amount`、`private_key`                           |

创建只登记 `owner` 公钥、不要求签名；之后的每次写都必须由该公钥对应的私钥
（或 `spender_key`）签名，私钥与登记公钥不匹配时返回 `401 INVALID_SIGNATURE`。

未知 `token_id` 的读接口返回 `404 TOKEN_NOT_FOUND`，与 `info` 一致——不会用
`200 {"amount":"0"}` 或 `200 []` 把“代币不存在”伪装成“没有活动”。
余额、授权和历史中的非空 `owner`/`spender` 必须解码为 32 字节 Ed25519 公钥；
长度错误返回 `400 INVALID_PUBLIC_KEY_LENGTH`，不会被当作空账户读取。
`token/history` 省略 `owner` 时读取整个代币的转账流水。它的 `limit` 上限 100、
`offset` 上限 **1000**（一次请求能翻多深是有限的，`/nft/list` 同为 1000），
`/token/list` 的 `offset` 上限则是 10000。`/token/info` 与
`/token/list` 还带 `mintable`/`burnable`（当前创建即两者皆可）。

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
`/oracle/query` 默认返回该源最新 10 条读数，`limit` 上限 100；它没有 `offset`，
更早的读数请改用桌面端或直接读库。CLI 的 `oracle data` 用同一组数字（默认 10、
上限 100），所以同一个 `--limit` 在两种界面上拿到同样长的列表。
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

数据源写接口的字段：`POST /oracle/sources` 接受 `name`、`url`、`type`、
`method`、`headers`、`path`、`interval`；`PATCH /oracle/sources/{id}` 只接受
`enabled`（布尔），用于启用/停用；`DELETE /oracle/sources/{id}` 没有请求体。

`POST /api/v1/oracle/sources` 的 `headers` 接受 JSON 对象，例如
`{"Authorization":"Bearer ...","X-API-Key":"..."}`。`method` 必须是 HTTP token，
请求头名称也必须是 token，值不能包含除 tab 外的控制字符；JSON、CR/LF 和长度都
在写入前校验，不合法一律返回 `INVALID_SOURCE`。

**有的头不能由数据源设置。** 分三种原因：

| 原因         | 头                                                                                                           | 说明                                      |
| ------------ | ------------------------------------------------------------------------------------------------------------ | ----------------------------------------- |
| 由传输层派生 | `Content-Length`、`Transfer-Encoding`、`Trailer`                                                             | 写入会与连接实际状态矛盾                  |
| 逐跳头       | `Connection`、`Keep-Alive`、`Proxy-Authenticate`、`Proxy-Authorization`、`Proxy-Connection`、`TE`、`Upgrade` | 只对单跳连接有效，不属于某个数据源        |
| 代理凭据     | —                                                                                                            | 放在 `HTTP_PROXY`/`HTTPS_PROXY` 的 URL 里 |

仅大小写不同的重复头名同样被拒绝，否则同一份配置会随 map 遍历顺序产生不同
请求。`Host` 会覆盖实际请求的虚拟主机，并在同源重定向中保留、跨源时清除。

数据源 URL 必须带非空 hostname；写出端口时必须在 `1..65535`，不能是只有冒号的
空值。`extractByPath` 接受合法的空字符串值，但对象、数组与 null 叶子会失败——
否则 Go map 会被当成链上数据记录下来。

**抓取拒绝内部地址。** URL 校验、重定向与实际拨号三处都查同一份策略，因此
DNS 在两次检查之间改指向也不会被绕过。被拒绝的是这些类别：

| 类别                | 例子                                                                 | 为什么                                |
| ------------------- | -------------------------------------------------------------------- | ------------------------------------- |
| 回环与未指定        | `127.0.0.0/8`、`::1`、`::`                                           | 数据源不该读本机服务                  |
| 私有与保留          | `10.0.0.0/8`、`172.16.0.0/12`、`192.168.0.0/16`、`0.0.0.0/8`         | 集群内网与占位地址                    |
| 链路本地            | `169.254.0.0/16`、`fe80::/10`                                        | 含云厂商元数据端点                    |
| 运营商级 NAT        | `100.64.0.0/10`                                                      | 运营商内网                            |
| 组播与保留          | `224.0.0.0/4`、`240.0.0.0/4`、`ff00::/8`                             | 不是单播目的地                        |
| 文档与基准网段      | `192.0.2.0/24`、`198.51.100.0/24`、`203.0.113.0/24`、`2001:db8::/32` | 示例地址不是真实服务                  |
| IPv6 转换前缀       | `64:ff9b::/96`、`2002::/16`                                          | 会翻译成内嵌的 IPv4，需按该 IPv4 判定 |
| 已废弃的 site-local | `fec0::/10`                                                          | 仍可能被解析到                        |
| IETF 协议分配块     | `2001::/23`（含 Teredo、ORCHID）                                     | 不指向可公开访问的主机                |

公开 IPv4 的 mapped IPv6 形式按对应 IPv4 判定，不会被误拒。带方括号且省略端口
的 IPv6 字面量先规范化再判定。完整的 30 条前缀清单在
`internal/netguard/blocked.go`，本表按类别归纳，新增网段只需改那一处。

重定向链最多跟随 10 跳，跨源重定向不继承数据源配置的请求头——否则
`X-API-Key` 等自定义凭据会泄漏给第三方主机。

数据源配置的 `User-Agent` 与 `Accept` 按“是否出现”覆盖 Aurora 默认值：显式空
`User-Agent` 表示不发送该头，空 `Accept` 也会保留；其他自定义头原样发送。
`Content-Type` 只在 POST/PUT/PATCH 且为空时补 `application/json`，
`X-Request-ID` 则由客户端为每一跳重新生成。

抓取失败时只返回 HTTP 状态或底层网络原因，不回显完整数据源 URL；userinfo、
path 与 query 都可能包含凭据，因此 `/oracle/health` 的 `last_error` 与调度器
日志不把它们作为诊断上下文。用 `source_id` 关联具体数据源。

## 区块链

| 方法 | 路径                        | 说明                                                |
| ---- | --------------------------- | --------------------------------------------------- |
| GET  | `/api/v1/blockchain/verify` | 校验本地区块链完整性                                |
| GET  | `/api/v1/blockchain/blocks` | 审计记录分页（默认 20，`limit` 上限 100，`q` 过滤） |

```json
{"valid":true,"length":3,"head":"0ab2857b160a…","first_broken_index":-1}
```

`first_broken_index` 为 `-1` 表示整条链通过；否则指向第一处哈希断开的区块下标，
`valid` 为 `false`。`head` 是头区块哈希（hex），只要链上有区块就会返回——链断开
时同样带着，两台机器对照记录时先比头哈希。

`/blockchain/blocks` 按最新在前返回创世块之外的记录，`payload` 是模块写入的原始
文本；`total` 不含创世块，`offset` 以条目计而不是区块高度。`q` 在原始文本
**和解码后的记录句子**上做不区分大小写的子串匹配（查 `Draw`、`Winners` 或模块名
都能命中，与 CLI 的 `-s`、桌面账本搜索一致），此时 `total` 为匹配条数：

```json
{"blocks": [{"height": 2, "timestamp": 1793385489, "payload": "b8dfb283-1b87-449e-a0aa-271b441e5d54|Star|AQIDBAUGBwgJCgsMDQ4PEBESExQVFhcYGRobHB0eHyA="}], "total": 2, "limit": 20, "offset": 0}
```

## 生命周期与可观测性

服务启动时同时启动预言机调度器（按每个源的 `interval`）与审计事件 outbox
补偿循环；收到 SIGINT/SIGTERM 后 15s 内优雅停机并释放 SQLite 句柄。日志路径由
`[log]` 控制，默认 `./logs/`（桌面端默认落在数据库旁的 `logs/`）。Unix-like
系统上日志目录固定为 `0700`、`aurora.log` 固定为 `0600`；旧版本留下的宽松
权限会在下一次启动时自动收紧。
