# JIT CLI 一期落地方案（v2）

## 定位

非交互式 CLI，面向 AI Agent 和脚本调用，基于 JIT PAT 认证。二进制名：`jit`。

## 技术栈

| 维度 | 选型 | 理由 |
|------|------|------|
| 语言 | Go 1.23+ | 单二进制分发、交叉编译、lark-cli/dws 验证过的路线 |
| CLI 框架 | spf13/cobra | 行业标准，两个参考项目都用 |
| JSON 过滤 | itchyny/gojq | 同上 |
| 密钥存储 | zalando/go-keyring | OS 原生 keychain，同上 |
| 构建发布 | goreleaser | 同上 |
| 输出终端着色 | fatih/color | 轻量，仅用于 stderr 提示 |

不引入 charmbracelet 系列（bubbletea/lipgloss/huh），一期不做 TUI。

---

## 路由模型（核心修正）

### 后端实际路径格式

JIT 后端路由不是简单的 `/api/{elementFullName}/{functionName}`，而是：

```
/api/{org}/{app}/{element/path/parts}/{functionName}
```

证据链：
- Http interceptor (`JitService/interceptors/Http/backend/interceptor.py:19-23`)：
  `names = [item for item in request.path.split("/") if item]`，然后 `names[3:i]` 取元素全名。
  过滤空字符串后 names = `['api', '{org}', '{app}', ...elementParts..., '{func}']`，
  `names[3:]` 跳过前 3 段（api/org/app）。
- Permission interceptor (`JitService/interceptors/Permission/interceptor.py:21-22`)：
  `path.replace("/api", "").strip("/").replace("/", ".")` → `{org}.{app}.{elementParts}.{func}`，
  再 `.replace(app.appId, "")` 去掉 `{org}.{app}` 前缀。
- 前端实际调用 (`JitAi/commons/AssistantRender/index.tsx:36`)：
  `apiUrl.path = page.app.apiPath`，SSE 调用拼接为 `{apiPath}/services/JitAISvc/sendMessage`。
- apiAuth (`JitService/apiAuths/NormalType/base.py:38`)：
  `self.names = self.request.path.split("/")[4:]`（含前导空字符串，等效于跳过 `/api/{org}/{app}`）。

### CLI 路径映射

`jit api` 必须感知 app 上下文。完整请求 URL：

```
{server}/api/{org}/{app}/{elementPath}/{functionName}
```

示例：
```
https://demo.jit.cn/api/wanyun/JitAi/services/JitAISvc/sendMessage
https://demo.jit.cn/api/wanyun/JitAuth/auths/loginTypes/services/AuthSvc/listCliTokens
https://demo.jit.cn/api/erp_demo/ErpApp/models/Customer/query
```

---

## Profile 模型（核心修正）

### 问题

JIT 是私有化部署的，不同客户实例地址不同，同一实例上可能有多个 app。
单一 `~/.jit/credentials` + 单一 `config.json` 无法隔离多环境凭证。

### 设计：按 server 隔离 profile

凭证隔离键：`server`（实例域名）。一个 profile 绑定一个 server + 一个 PAT + 一个默认 app。

```
~/.jit/
├── config.json              # 全局配置（当前 profile、默认输出格式）
└── profiles/
    ├── demo/                # profile 名（用户自定义，默认取域名）
    │   ├── config.json      # server URL、默认 app（org/app）
    │   └── credentials      # PAT（文件权限 0600，keychain 优先）
    └── staging/
        ├── config.json
        └── credentials
```

全局 `config.json`：
```json
{
  "current_profile": "demo",
  "default_format": "json"
}
```

Profile `config.json`：
```json
{
  "server": "https://demo.jit.cn",
  "default_app": "wanyun/JitAi"
}
```

### 凭证存储策略

1. 优先 OS keychain（go-keyring），key = `jit-cli:{profile_name}`
2. fallback 到 `~/.jit/profiles/{name}/credentials`（文件权限 0600）

---

## 命令面设计

### 三层结构

```
jit auth          — PAT 管理 + profile 管理
jit api           — 通用 API 网关（透传后端响应）
jit <shortcut>    — 高频业务动作的快捷命令
```

### 第一层：jit auth

```bash
# Profile + PAT 管理
jit auth login --server <url> --app <org/app> [--token <pat>] [--profile <name>]
    # 创建/更新 profile，写入 PAT
    # --token 省略时从 stdin 读取（不回显）
    # --profile 省略时自动从 server 域名生成

jit auth status [--profile <name>]
    # 验证 PAT 有效性 + 输出当前用户/企业信息
    # 实现：调用 corps/services/MemberSvc/getCurrUserInfo（Bearer PAT）
    # 成功 → PAT 有效，返回 {user, member, corpFullName}
    # 失败 → PAT 无效或过期

jit auth logout [--profile <name>]
    # 清除指定 profile 的 PAT

jit auth list
    # 列出所有 profile（名称、server、默认 app、PAT 状态）

jit auth use <profile>
    # 切换当前活跃 profile
```

`auth status` 的验收标准明确为：
1. 调用 `corps/services/MemberSvc/getCurrUserInfo`（`JitAuth/corps/services/MemberSvc/service.py:19`）
2. 解析 JSON 响应，按 errcode 判定：errcode == 0 → PAT 有效，输出 `{user, member, corpFullName}` 到 stdout
3. errcode != 0 → PAT 无效或过期，stderr 报错并提示重新 login（不依赖 HTTP 401/403，JIT 认证失败走 errcode 而非 HTTP 状态码）

### 第二层：jit api

通用 API 网关，**透传后端原生响应**，不再包 CLI 信封。

```bash
# 基本用法（使用当前 profile 的 server + default_app）
jit api <elementPath>/<functionName> [--data '{}']

# 显式指定 app（覆盖 default_app）
jit api <elementPath>/<functionName> --app <org/app> [--data '{}']

# 示例（app 上下文来自 profile.default_app 或 --app）
jit api services/JitAISvc/sendMessage --data '{"assistantId":"xxx","chatId":"yyy","message":"hello"}'
jit api auths/loginTypes/services/AuthSvc/listCliTokens --app wanyun/JitAuth
jit api models/services/ModelSvc/getModelList --app wanyun/JitORM
jit api models/services/ModelSvc/getModelsMeta --app wanyun/JitORM
jit api models/Customer/query --data '{"filter":{},"page":1,"size":10}' --app erp_demo/ErpApp

# 全局 flags
--data <json>         请求体 JSON（也可从 stdin 读取：echo '{}' | jit api ... --data @-）
--app <org/app>       指定目标应用（覆盖 profile 默认值）
--profile <name>      指定 profile（覆盖当前活跃 profile）
--jq <expr>           jq 表达式过滤输出
--dry-run             只打印完整请求 URL + headers + body，不执行
```

路径拼装逻辑：
```
{profile.server}/api/{org}/{app}/{elementPath}/{functionName}
                      ^^^^^^^^^^^
                      来自 --app 或 profile.default_app
```

### 输出规范（核心修正）

**`jit api` 默认透传后端原生响应**，不包 CLI 信封。理由：
- "直接映射后端接口，零学习成本" 和 "CLI 再包一层信封" 是互相矛盾的
- Agent 拿到的就是后端文档里描述的响应格式，无需二次适配
- 后端已有自己的 errcode/data/details 契约

CLI 自身的错误（网络超时、PAT 无效、参数缺失等）：
- 输出到 stderr（JSON 格式，方便 Agent 解析）
- 设置非零 exit code

```bash
# 成功：stdout = 后端原生响应，exit 0
$ jit api models/services/ModelSvc/getModelList --app wanyun/JitORM
{"errcode": 0, "data": [...], "errmsg": "success"}

# 后端业务错误：stdout = 后端原生响应，exit 1
$ jit api models/services/ModelSvc/getModelInfo --data '{"fullName":"nonexist.Model"}' --app wanyun/JitORM
{"errcode": 40001, "errmsg": "模型不存在", "data": null}

# CLI 自身错误：stderr = CLI 错误，exit 2
$ jit api models/services/ModelSvc/getModelList --profile nonexist
# stderr: {"error": "profile_not_found", "message": "profile 'nonexist' does not exist"}
```

Exit code 约定：
- 0 = 请求成功且后端 errcode == 0
- 1 = 请求成功但后端 errcode != 0（业务错误）
- 2 = CLI 自身错误（网络、认证、参数等）

**`jit model` 等 shortcut 可以做轻量格式化**（因为 shortcut 本身就是抽象层），但 `jit api` 必须保持透传。

### 第三层：shortcuts（一期少量）

一期 shortcut 聚焦模型操作，已验证后端接口存在：

模型元数据接口（`JitORM/models/services/ModelSvc/e.json` 确认）：
- `getModelList` — 获取所有模型列表（loginRequired: 0）
- `getModelsMeta` — 获取所有模型元数据（loginRequired: 0）
- `getModelInfo(fullName)` — 获取单个模型定义
- `aiSelect(tql, limit, offset)` — TQL 查询

模型数据 CRUD（`JitORM/models/NormalType/backend/model.py` 确认，通过模型 fullName 路径直接调用）：
- `query(filter, fieldList, orderList, page, size, level)` — 分页查询
- `get(filter, orderList, level)` — 获取单条
- `create(rowData, triggerEvent)` — 创建记录
- `updateByPK(pkList, updateData, triggerEvent)` — 按主键更新
- `deleteByPK(pkList, triggerEvent)` — 按主键删除
- `createOrUpdateMany(rowDataList, triggerEvent)` — 批量创建/更新

注意：ModelSvc 位于 JitORM 应用下，模型数据 CRUD 通过各业务 app 下的模型 fullName 路径调用。
CLI 需要处理两种 app 上下文：
- `jit model list/meta/info` → 打 JitORM 的 ModelSvc
- `jit model query/create/update/delete` → 打业务 app 下的具体模型

```bash
# 模型元数据（默认打 JitORM 应用）
jit model list                                    # → ModelSvc/getModelList
jit model meta                                    # → ModelSvc/getModelsMeta
jit model info <fullName>                         # → ModelSvc/getModelInfo

# 模型数据 CRUD（需指定业务 app 或使用 profile 默认 app）
jit model query <fullName> [--filter '{}'] [--page 1] [--size 10] [--app <org/app>]
    # → {app}/models/{fullName}/query
jit model create <fullName> --data '{}' [--app <org/app>]
    # → {app}/models/{fullName}/create
jit model update <fullName> --pk '[]' --data '{}' [--app <org/app>]
    # → {app}/models/{fullName}/updateByPK
jit model delete <fullName> --pk '[]' [--app <org/app>]
    # → {app}/models/{fullName}/deleteByPK

# TQL 查询（打 JitORM 的 ModelSvc）
jit model select <tql> [--limit 50] [--offset 0]  # → ModelSvc/aiSelect

# 用户信息
jit whoami [--profile <name>]
```

shortcut 内部调用同一套 HTTP client，路径拼装逻辑一致。

---

## 项目结构

```
wanyun/JitCli/
├── cmd/
│   ├── root.go              # cobra root command + 全局 flags（--profile, --app, --jq, --format）
│   ├── auth/
│   │   └── auth.go          # login/logout/status/list/use
│   ├── api/
│   │   └── api.go           # 通用 API 网关
│   └── model/
│       └── model.go         # model shortcut
├── internal/
│   ├── profile/
│   │   ├── profile.go       # profile CRUD（~/.jit/profiles/）
│   │   └── store.go         # keychain + file fallback 凭证存储
│   ├── client/
│   │   └── client.go        # HTTP client（PAT Bearer auth + 路径拼装）
│   ├── config/
│   │   └── config.go        # 全局配置（~/.jit/config.json）
│   ├── output/
│   │   ├── json.go          # JSON 输出（透传 + pretty print）
│   │   └── jq.go            # --jq 过滤
│   └── build/
│       └── version.go       # 版本信息（ldflags 注入）
├── main.go
├── go.mod
├── go.sum
├── Makefile
├── .goreleaser.yml
└── README.md
```

---

## 认证流程

```
用户在 JIT Web 个人中心创建 PAT
         ↓
jit auth login --server https://demo.jit.cn --app wanyun/JitAi --token jit_pat_xxx_yyy
         ↓
CLI 解析 token 格式（jit_pat_{tokenId}_{secret}），格式不对直接报错
         ↓
CLI 调用 corps/services/MemberSvc/getCurrUserInfo（Bearer PAT）验证有效性并获取用户信息
         ↓
有效 → 创建 profile，PAT 存入 keychain（fallback 文件），输出 {user, member, corpFullName}
无效 → 报错，提示检查 token 或 server 地址
         ↓
后续所有请求自动从当前 profile 读取 server + app + PAT
```

注意：不再使用 `isTokenExpire` 做验证（该接口只返回 status/msg，无用户详情）。
直接调 `corps/services/MemberSvc/getCurrUserInfo` 一步到位：能拿到数据说明 PAT 有效，拿不到就是无效。

---

## 分发策略

优先级从高到低：

1. **GitHub Release 二进制包**（goreleaser 生成，darwin/linux/windows × amd64/arm64）
   - 配套 install.sh / install.ps1 一键安装脚本
2. **Homebrew / Scoop**（goreleaser 自动生成 formula/manifest）
3. **npm wrapper**（仅作安装入口，postinstall 下载对应平台二进制）
   - 参考 lark-cli 的 package.json + scripts/install.js 模式
   - CLI 主体始终是 Go 二进制，npm 只是分发渠道

---

## 一期里程碑（含测试验收门）

| 阶段 | 内容 | 验收标准 | 预估 |
|------|------|----------|------|
| M1 | 项目骨架 + profile 模型 + auth login/status/logout/list/use | profile CRUD 正确、per-server 凭证隔离、keychain 存取、PAT 格式校验 | 3 天 |
| M2 | jit api 通用网关 + --jq + --dry-run | 路径拼装正确（`/api/{org}/{app}/{path}`）、Bearer PAT 注入、后端响应透传、--dry-run 不发请求、stdout/stderr 分流、exit code 三级约定 | 3 天 |
| M3 | jit model shortcut + jit whoami | model CRUD 映射正确、--app 覆盖生效、whoami 输出用户/企业信息 | 2 天 |
| M4 | goreleaser + install 脚本 + npm wrapper | 三平台六架构构建通过、install.sh smoke test、npm postinstall 下载正确二进制 | 2 天 |
| M5 | 错误处理完善 + README | CLI 错误 JSON 格式化到 stderr、后端 errcode 透传到 exit code、README 覆盖安装/认证/使用 | 1 天 |

每个里程碑的测试要求：
- 单元测试：路径拼装、profile 读写、PAT 解析、output 格式化、jq 过滤
- 集成测试（可 mock HTTP）：auth login 全流程、api 请求全流程、--dry-run 输出验证
- 手动验收：对真实 JIT 实例执行 auth login → api 调用 → model query 全链路

总计约 11 天，一个人可完成。

---

## 对 Review Findings 的逐条回应

### v1 Review

#### Finding 1（高风险）：路由模型写错

已修正。v2 方案的路由模型基于对 Http/Permission/apiAuth 三个 interceptor 的源码分析，
完整路径格式为 `/api/{org}/{app}/{elementPath}/{functionName}`。
CLI 通过 profile 的 `default_app`（格式 `org/app`）+ 命令行 `--app` 覆盖来提供 app 上下文。
详见「路由模型」章节。

#### Finding 2（中高风险）：存储模型是单实例思维

已修正。v2 引入 profile 模型，凭证按 server 隔离。
每个 profile 绑定 server + PAT + default_app，支持 `jit auth use` 切换。
详见「Profile 模型」章节。

#### Finding 3（中风险）：auth status 契约比后端现状更强

v2 中改为调用 `MemberSvc.getCurrentMember`，但接口名仍然是错的。v3 已修正，见下方。

#### Finding 4（中风险）：jit api 透传 vs CLI 信封矛盾

已修正。`jit api` 定位为 raw API gateway，默认透传后端原生响应，不包 CLI 信封。
CLI 自身错误走 stderr + exit code 2。后端业务错误走 stdout + exit code 1。
shortcut 层（如 `jit model`）可以做轻量格式化，因为 shortcut 本身就是抽象层。
详见「输出规范」章节。

#### Finding 5（中风险）：里程碑缺测试验收门

已修正。每个里程碑增加了明确的验收标准，覆盖：
路径拼装正确、per-server 凭证隔离、--dry-run 不发请求、stdout/stderr 分流、
Bearer PAT 登录态恢复、errcode 透传、install 脚本 smoke test。
详见「一期里程碑」章节。

#### Question 回答

1. **app 上下文怎么给 CLI**：profile 配置 `default_app`（`org/app` 格式）+ 每个命令 `--app` flag 覆盖。不用全局配置单一 app，因为同一 server 上可能操作多个 app。

2. **jit api 是透传还是 CLI envelope**：透传。已裁定。

3. **table 算一期还是二期**：二期。一期只做 JSON 输出 + `--jq` 过滤。`--format` flag 一期就注册（预留），但只接受 `json`，传 `table` 报 "table format will be available in a future release"。

### v2 Review

#### Finding 1（中风险）：auth status 接口名写错

已修正。v2 中三处不一致（auth status 说明写 isTokenExpire + getCurrentMember，认证流程写 getCurrentMember，
回应写不再用 isTokenExpire）全部统一为：`corps/services/MemberSvc/getCurrUserInfo`。
依据：`JitAuth/corps/services/MemberSvc/e.json:7` 确认函数名为 `getCurrUserInfo`，
`service.py:19` 确认实现返回 `{user, member, corpFullName}`。
方案中 auth status 命令说明、认证流程、M1 验收标准三处已同步修正。

#### Finding 2（中风险）：jit model 接口契约未验证

已修正。经源码验证，接口全部存在：
- `ModelSvc` 位于 `JitORM/models/services/ModelSvc/`，`e.json` 确认 `getModelList`（L90）、
  `getModelsMeta`（L98）、`getModelInfo`（L109）、`aiSelect`（L421）等接口。
- NormalModel CRUD 方法（query/create/updateByPK/deleteByPK 等）定义在
  `JitORM/models/NormalType/backend/model.py`，通过 `loader.py:31-41` 动态注册到 `__functionList__`，
  可通过模型 fullName 路径直接调用。
- 关键区分：ModelSvc 在 JitORM 应用下，模型数据 CRUD 通过各业务 app 下的模型路径调用。
  shortcut 层已明确标注两种 app 上下文的处理方式。

### v3 Review

#### Finding 1（中风险）：auth status 失败判定按 HTTP 401/403 写，与 JIT 契约不一致

已修正。JIT 认证失败不走标准 HTTP 401/403，而是返回 HTTP 200 + JSON body 中 errcode != 0。
auth status 验收标准已改为：解析 JSON 响应，按 errcode == 0 判定成功，errcode != 0 判定失败。
依据：`JitAuth/tpsync/Meta/backend/tpac/jit/client.py:41-62` 确认 JIT 客户端按 errcode 判错。

#### Finding 2（低到中风险）：输出示例使用不存在的 ModelSvc/query 端点

已修正。示例中的 `ModelSvc/query` 替换为真实存在的 `ModelSvc/getModelInfo`。
ModelSvc 的完整接口列表已在 shortcut 章节中基于 `e.json` 逐条列出。

---

## 二期展望（不在一期范围）

- `--format table` 输出格式
- `jit ai` — 对接 AI 助理对话（SSE 流式）
- `jit element` — 元素管理 shortcut
- `jit workflow` — 工作流触发
- shell completion（bash/zsh/fish/powershell）
- 自动更新检查（参考 lark-cli 的 update 模块）
- Agent Skills 体系（参考 lark-cli/dws 的 skills 目录）
