---
name: jit
description: 通过内置 jit CLI 检查或操作 JIT 后端，涵盖登录、app 元数据、开发态元素源码、模型、服务、原始 API 与明细/分析查询；适用于用户提到 jit、JIT 后端、auth、profile、app refresh、element、model、service、api、`jit model analyze` 或 `jit model query` 时。
---

# jit

当用户需要通过内置 CLI 检查或操作 JIT 后端时，使用 `jit`。

先查本地 help

- 在猜测命令名、参数名或参数格式之前，先读本地 help。
- 先看 `jit --help`，再根据问题缩小到 `jit auth --help`、`jit app --help`、`jit element --help`、`jit model --help`、`jit service --help` 或 `jit api --help`。
- 如果还不确定，就继续看精确命令的 help，例如 `jit element get --help`、`jit model query --help` 或 `jit service call --help`。
- 只要 help 已经回答了问题，就优先相信 help，而不是重复依赖本 skill。

按问题选择正确的命令族

- `jit auth ...` 用于登录、profile 管理和身份检查。
- `jit app ...` 用于 app 元数据刷新、检查和当前应用全量构建。
- `jit element ...` 用于开发态元素源码读取、保存、构建、检索和知识描述读取。
- 在写查询前，用 `jit model ls|get` 检查模型和字段。
- `jit model query` 用于读取模型明细数据，支持 `--filter`、`--fields`、`--order`、`--page`、`--size` 和 `--level`。
- `jit model create|update|delete` 用于原子写操作。
- `jit model analyze` 用于直接执行 TQL 统计与分析查询。
- `jit service ...` 用于服务发现和服务调用。
- `jit api ...` 用于原始后端 API 调用。

开发态元素规则

- 修改元素源码前，先用 `jit element get <fullName>` 读取当前资源，不要凭本地猜测覆盖远端。
- 保存单个元素资源用 `jit element save <fullName> --resources <json>`；同时保存声明和多个元素时用 `jit element apply --data <json>`。
- 只构建少量元素用 `jit element build <fullName>...`；全量构建当前 app 用 `jit app build`。
- 查找源码引用用 `jit element search <pattern>`；理解元素职责先试 `jit element knowledge <fullName>`。
- 裸删除、重命名、批量替换等破坏性文件操作没有默认命令；除非用户明确要求，否则不要通过 `jit api` 调这些低层接口。

查询场景补充规则

- 如果任务涉及编写或修复 TQL 或 Q 表达式，在构造命令之前先读 `references/tql-query-guide.md`。
- 如果不确定模型名或字段名，先执行 `jit app refresh`、`jit model ls`、`jit model get <fullModelName>` 再写查询。
- 不要凭记忆臆造 SQL、ORM 或 LINQ 风格语法。

工作方式

- 除非用户明确要求修改远端状态，否则优先选择只读命令。
- `jit` 是 JSON-first CLI，payload 和 shell quoting 要保持精确。
- 如果上下文不明确，显式传 `--app <org/app>` 和 `--profile <name>`，不要猜。
- 如果输出或可用元数据看起来过期，先执行 `jit app refresh` 再重试。
