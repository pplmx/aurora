# Aurora — 仓库约定

产品介绍、安装方式、功能清单与完整的 `just` 清单在 [README.md](README.md)。本文件
只记录在这个仓库改代码时要遵守的约束；同一份说明写两遍会各自过期，产品正文不再
复制到这里。

## 入口与正文

- 任务入口：[justfile](justfile)，没有 Makefile。
- 设计正文：桌面 [docs/desktop.md](docs/desktop.md)、CLI [docs/cli.md](docs/cli.md)、
  REST [docs/api.md](docs/api.md)、分层契约
  [docs/architecture.md](docs/architecture.md)。
- 配置查找顺序：`--config` → `$HOME/aurora.toml` → `./config/aurora.toml`，各节
  语义见 [config/aurora.toml](config/aurora.toml)。
- 历史记录不是现行指令：`docs/superpowers/`（早期设计）与 `.planning/`
  （里程碑）保留原样；以代码和上面的正文为准。

## 门槛

改到哪一层就跑哪一层的门槛，跑过才算完成：

| 改动     | 命令                                                                     |
| -------- | ------------------------------------------------------------------------ |
| 根模块   | `just check` + `just test`                                               |
| 桌面模块 | `just desktop-check`；行为测试再跑 `just desktop-test`                   |
| lint     | `just lint`（errcheck 关闭）；桌面模块的 lint 在 `just desktop-check` 里 |
| 界面     | `just desktop-shot`                                                      |

- 真实窗口截图按需加环境变量：有数据 `AURORA_E2E_SEED=1`、浅色
  `AURORA_E2E_THEME=light`、窄窗 `AURORA_E2E_SIZE=900x600`。
- 覆盖率底线：核心模块 Domain 与 App 都在 80% 以上；数字只在 README 维护。
- 不删测试、不跳测试、不降低断言或阈值换绿灯；修不好就回滚并说明原因。

## 代码约定

- 分层方向：`domain` 不导入其他层；`app` 编排用例；`infra` 实现外部依赖；
  `api` 与 `desktop` 是两个表示层，共用 `app` 层，不各自实现业务规则。
- 默认 `CGO_ENABLED=0`；不引入 C 依赖。
- 桌面是 MyGo 原生控件（`desktop/internal/nativeui`），没有 HTML/JavaScript；
  窗口文案走 `tr()`，中文表在 `desktop/internal/nativeui/messages_zh.go`，缺项
  回退英文。
- 破坏性命令一律要求 `-y/--confirm`，不保留旧别名。
- 文档与命令、路由、`just` 配方的一致性由 `internal/repocheck` 与
  `internal/api/docs_test.go` 锁定；改这些表面时同步改测试。

## 提交

- 直接在 `main` 开发；一次提交只放一个自洽的逻辑单元，代码与它的测试同提交。
- 禁止 `git commit --amend`（hook 自动改过且未推送时除外）。
- 不推送：`git push` 由维护者决定。
