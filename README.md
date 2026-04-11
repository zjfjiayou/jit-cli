# JIT CLI

`jit` 是一个面向 JIT 的非交互式命令行工具，主要服务于 AI Agent 和脚本场景。
一期默认使用 PAT（`jit_pat_*`）作为鉴权方式，通过 `Authorization: Bearer <token>` 直接复用现有 JIT 后端接口。

## 内置 Agent Skill

仓库内置了一份给 Agent 使用的 `jit` skill，路径如下：

```text
resources/skills/builtin/jit/SKILL.md
```

这份 skill 面向已经安装好 `jit` CLI 的 Agent 运行环境，约束 Agent 先查本地 help，再按 `auth`、`app`、`model`、`service`、`api` 等命令族选择正确命令，避免凭记忆臆造参数或查询语法。

如果任务涉及 TQL 或 Q 表达式，skill 还会引用同目录下的参考文档：

```text
resources/skills/builtin/jit/references/tql-query-guide.md
```

如果你要把 JitCli 集成到自己的 Agent/技能系统，可以直接复用这套目录结构。

## 安装

### 1. 从 Release 下载二进制

GoReleaser 预期产物如下：

- `jit-darwin-amd64.tar.gz`
- `jit-darwin-arm64.tar.gz`
- `jit-linux-amd64.tar.gz`
- `jit-linux-arm64.tar.gz`
- `jit-windows-amd64.zip`
- `jit-windows-arm64.zip`

解压后执行：

```bash
chmod +x jit
mv jit ~/.local/bin/jit
```

### 2. 使用安装脚本

Linux/macOS：

```bash
curl -fsSL https://raw.githubusercontent.com/zjfjiayou/jit-cli/main/scripts/install.sh | sh
```

Windows PowerShell：

```powershell
irm https://raw.githubusercontent.com/zjfjiayou/jit-cli/main/scripts/install.ps1 | iex
```

可选安装环境变量：

- `JIT_CLI_REPO`：仓库名，默认 `zjfjiayou/jit-cli`
- `JIT_CLI_VERSION`：版本号，默认 `latest`
- `JIT_CLI_INSTALL_DIR`：安装目录，默认 Unix 为 `~/.local/bin`，PowerShell 为 `~/.local/bin`

## 认证（PAT + Profile）

先在 JIT Web 个人中心创建 PAT，再执行登录：

```bash
jit auth login --server https://demo.jit.cn --app wanyun/JitAi --token jit_pat_xxx_yyy
```

也可以通过 stdin 传入 token：

```bash
printf '%s' 'jit_pat_xxx_yyy' | jit auth login --server https://demo.jit.cn --app wanyun/JitAi
```

查看当前身份：

```bash
jit auth whoami
jit whoami
```

常用 profile 操作：

```bash
jit auth ls
jit auth use demo
jit auth use 0
jit auth logout --profile demo
jit auth rm demo
```

- `jit auth logout`：只删除 profile 对应的 PAT，保留 profile 配置
- `jit auth rm`：删除整个 profile，包括配置、PAT 和本地缓存
- `jit auth ls` 会按当前输出顺序给每个 profile 标注 `index`，可直接用于 `jit auth use <index>`

## API 用法

原始 API 网关调用，默认使用当前 profile 的 `default_app`：

```bash
jit api services/JitAISvc/sendMessage --data '{"assistantId":"a","chatId":"c","message":"hello"}'
```

显式指定 app：

```bash
jit api auths/loginTypes/services/AuthSvc/listCliTokens --app wanyun/JitAi
```

AppInfo 缓存相关命令：

```bash
jit app refresh
jit app get
jit app ls
```

服务快捷命令：

```bash
jit service ls
jit service ls --all
jit service ls --filter attendance
jit service call corps.services.AttendanceSvc getAttendanceColumns --data '{"corpFullName":"corps.Default"}'
```

模型相关示例：

```bash
jit app refresh
jit model ls
jit model ls --all
jit model get wanyun.crm.Customer
jit model query wanyun.crm.Customer --filter 'Q("name", "=", "Alice")' --fields '["id","name"]' --order '[["id",-1]]' --page 1 --size 20 --app erp_demo/ErpApp
jit model create wanyun.crm.Customer --data '{"name":"Alice"}'
jit model update wanyun.crm.Customer --filter 'Q("id","=",1)' --data '{"name":"Bob"}'
jit model delete wanyun.crm.Customer --filter 'Q("id","=",1)'
jit model analyze 'Select([F("id"), F("name")], From(["models.Customer"]), Limit(0, 10))'
```

`jit model` 说明：

- `jit model` 始终使用解析出的业务 app：若传入 `--app <org/app>` 则优先使用，否则回退到 profile 的 `default_app`。
- CLI 不再自行推导 `JitAuth`、`JitORM` 这类兄弟应用，共享服务由后端继承机制负责解析。
- `jit model ls` 读取本地缓存的 `appInfo.js` 结果，默认只返回当前 app 自身的非 private 模型元素。
- 传入 `jit model ls --all` 时，会把 `extendApps` 中集成进来的模型也一起列出。
- 切换 app 或后端元素定义发生变化后，建议重新执行 `jit app refresh`。
- `jit model get` 仍然调用 `ModelSvc/getModelInfo`，完整字段定义以后端接口为准。
- `jit model query` 调用模型的 `aiQuery`，适合读取明细列表；`--filter` 传的是 Q 表达式字符串，`--fields` 和 `--order` 传 JSON 数组字符串。
- `jit model create`、`jit model update`、`jit model delete` 分别调用 `aiCreate`、`aiUpdate`、`aiDelete`。
- `jit model analyze` 通过 `ModelSvc/aiSelect` 执行 TQL 统计与分析查询。

`jit service` 说明：

- `jit service ls` 读取本地 `appInfo.js` 缓存，默认只列出当前 app 自身中非 private 且带有 `functionList` 的元素。
- 传入 `jit service ls --all` 时，会把 `extendApps` 中集成进来的服务也一起列出。
- 某些实际可调用的服务不会出现在列表里，例如来源于继承链、且在源 app 中被标记为 `private` 的服务；这类服务仍可以通过 `jit service call` 或 `jit api` 直接调用。
- `jit service call` 仅在元素命中缓存时校验 `functionName`；如果元素不在缓存中，则跳过预校验，最终以后端返回结果为准。
- 切换 app 或后端元素定义发生变化后，建议重新执行 `jit app refresh`。

全局参数：

- `--profile <name>`
- `--app <org/app>`
- `--jq <expr>`
- `--format json`：一期仅支持 JSON
- `--dry-run`

## 退出码

- `0`：请求成功，且后端 `errcode == 0`
- `1`：请求成功，但后端 `errcode != 0`，属于业务错误
- `2`：CLI 侧错误，例如网络错误、鉴权失败、参数非法、profile 不存在等

`jit api` 会将后端原始响应输出到 stdout。
CLI 自身错误会以 JSON 形式输出到 stderr。

## 构建与发布

本地构建：

```bash
make build
```

运行测试：

```bash
make test
```

生成 GoReleaser snapshot：

```bash
make snapshot
```

## 说明

- 一期以 JSON 输出和脚本友好为主。
- 一期刻意不提供 `--format table`。
- `jit api` 不会再额外包一层 CLI 自定义响应结构。
