# JIT CLI

`jit` 是一个面向 JIT 的非交互式命令行工具，主要服务于 AI Agent 和脚本场景。
一期默认使用 PAT（`jit_pat_*`）作为鉴权方式，通过 `Authorization: Bearer <token>` 直接复用现有 JIT 后端接口。

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
jit auth status
jit whoami
```

常用 profile 操作：

```bash
jit auth list
jit auth use demo
jit auth logout --profile demo
```

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
jit app info
jit app elements
```

服务快捷命令：

```bash
jit service list
jit service list --all
jit service list --filter attendance
jit service exec corps.services.AttendanceSvc getAttendanceColumns --data '{"corpFullName":"corps.Default"}'
```

模型相关示例：

```bash
jit app refresh
jit model list
jit model list --all
jit model info wanyun.crm.Customer
jit model query wanyun.crm.Customer --filter 'Q("name", "=", "Alice")' --page 1 --size 10 --app erp_demo/ErpApp
```

`jit model` 说明：

- `jit model` 始终使用解析出的业务 app：若传入 `--app <org/app>` 则优先使用，否则回退到 profile 的 `default_app`。
- CLI 不再自行推导 `JitAuth`、`JitORM` 这类兄弟应用，共享服务由后端继承机制负责解析。
- `jit model list` 读取本地缓存的 `appInfo.js` 结果，默认只返回当前 app 自身的非 private 模型元素。
- 传入 `jit model list --all` 时，会把 `extendApps` 中集成进来的模型也一起列出。
- 切换 app 或后端元素定义发生变化后，建议重新执行 `jit app refresh`。
- `jit model info` 仍然调用 `ModelSvc/getModelInfo`，完整字段定义以后端接口为准。
- `jit model query --filter` 传的是 Q 表达式字符串；省略时会按空过滤查询，不再把过滤条件当作 JSON 对象发送。

`jit service` 说明：

- `jit service list` 读取本地 `appInfo.js` 缓存，默认只列出当前 app 自身中非 private 且带有 `functionList` 的元素。
- 传入 `jit service list --all` 时，会把 `extendApps` 中集成进来的服务也一起列出。
- 某些实际可调用的服务不会出现在列表里，例如来源于继承链、且在源 app 中被标记为 `private` 的服务；这类服务仍可以通过 `jit service exec` 或 `jit api` 直接调用。
- `jit service exec` 仅在元素命中缓存时校验 `functionName`；如果元素不在缓存中，则跳过预校验，最终以后端返回结果为准。
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
