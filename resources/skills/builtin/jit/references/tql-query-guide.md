# TQL 与 Q 表达式参考

当 `jit model tql --help` 或 `jit model query --help` 还不够时，阅读本文件。

## 先确认元数据

- 在写复杂查询前先刷新 app 缓存：`jit app refresh`
- 在猜字段前先检查目标模型：`jit model get <fullModelName>`
- 使用完整模型名，不要只写简称、标题或业务昵称
- 写查询前先确认字段的 `name`，不要把字段 `title` 当作查询字段名

## 命令选择

- 已经有完整 TQL 语句时，使用 `jit model tql '<TQL>'`
- 可以表达成“模型 + Q 表达式筛选”时，使用 `jit model query <fullModelName> --filter '<Q expression>' --page <n> --size <n>`
- 不确定字段、模型、继承来的元素时，先用 `jit model ls|get` 检查，不要直接猜

## TQL 基本认知

**TQL 不是 SQL，也不是 ORM、LINQ。** 这是围绕 JIT 数据模型构建的独立查询语法，不要把你熟悉的 SQL/ORM 惯性直接带进来。

| 你可能想写的 | TQL 正确写法 |
| --- | --- |
| `SELECT * FROM table` | `Select([F("field1"), F("field2")], From(["models.Model"]))` |
| `Model.query.filter()` | `Select([F("id"), F("name")], From(["models.Model"]), Where(...))` |
| `table.filter().order_by()` | `Select(..., From(...), Where(...), OrderBy(...))` |
| `FROM table WHERE cond` | `From([...]), Where(Q(...))` |
| `WHERE MONTH(date) = 3` | `Where(Q("date", "month", 3))` |
| `M("model") \| Q(...) \| Limit(5)` | `Select(..., From(...), Where(...), Limit(0, 5))` |
| `UPPER(name)` | 不支持，TQL 没有大小写转换函数 |

最小模板：

```python
Select(
  [F("id"), F("name")],
  From(["models.Customer"]),
  Where(Q("name", "=", "Alice")),
  Limit(0, 10),
)
```

常见结构：

```python
Select(
  [F("id"), F("name"), F("createTime")],
  From(["models.Customer"]),
  Where(Q("status", "=", "active")),
  OrderBy((F("createTime"), -1)),
  Limit(0, 20),
)
```

```python
Select(
  [F("deptId"), F(Formula("COUNT(F('id'))"), "cnt")],
  From(["models.UserModel"]),
  GroupBy(F("deptId")),
  Having(Q("cnt", ">", 5)),
  Limit(0, 50),
)
```

## TQL 核心规则

五条铁律：

1. `From(["model"])` 的参数必须是列表，不能是字符串
2. `Select(...)` 里的字段必须显式写成 `F("fieldName")`
3. `Q(...)` 必须放在 `Where(...)` 或 `Having(...)` 中，不能直接链到 `From(...)`
4. `Limit(offset, size)` 需要两个参数
5. 只使用本文档明确列出的函数、结构和操作符，没写出来的不要假设支持

常见不支持函数：

| 类别 | 不支持 | 替代方案或说明 |
| --- | --- | --- |
| 数学 | `SQRT`, `LOG`, `LN`, `CEIL`, `FLOOR` | 平方根可用 `POWER(x, 0.5)` |
| 文本 | `UPPER`, `LOWER`, `REVERSE`, `SUBSTRING` | TQL 没有大小写转换 |
| 聚合 | `PERCENTILE`, `CORR`, `COVAR` | 仅使用本文档列出的聚合函数 |
| 日期 | `DATEDIFF`, `TIMESTAMPDIFF` | 用 `DATEDELTA` |
| 其他 | `CAST`, `CONVERT`, `COALESCE`, `NULLIF` | 用 `DEFAULTVALUE`、`IF` 等 |

常见错误对照：

| 错误写法 | 正确写法 | 原因 |
| --- | --- | --- |
| `From("models.UserModel")` | `From(["models.UserModel"])` | From 参数必须是列表 |
| `Limit(10)` | `Limit(0, 10)` | Limit 需要两个参数 |
| `Formula("COUNT(*)")` | `Formula("COUNT(F('id'))")` | COUNT 必须指定字段 |
| `Q("a", ">", 1) & Q("b", "<", 2)` | `Q(Q("a", ">", 1), Q.AND, Q("b", "<", 2))` | 用 `Q.AND` 组合，不是 `&` |
| `From([...]).Q(...)` | `Select([...], From([...]), Where(Q(...)))` | Q 必须放在 Where 中 |
| `OrderBy([(F("field"), 1)])` | `OrderBy((F("field"), 1))` | OrderBy 参数直接用元组 |
| `OrderBy(F("field"), desc=True)` | `OrderBy((F("field"), -1))` | 不支持 `desc` 关键字 |
| `OrderBy((Formula("COUNT(...)"), 1))` | 不支持 | OrderBy 只能用 `F()` 字段引用 |
| `Formula("SQRT(F('x'))")` | `Formula("POWER(F('x'), 0.5)")` | 不支持 `SQRT`，用 `POWER` 代替 |
| `Select(["field"], ...)` | `Select([F("field")], ...)` | 字段必须用 `F()` 包装 |
| `SELECT * FROM table` | 改成 `Select(...)`、`From([...])` 结构 | TQL 不是 SQL |

常见报错与修正方向：

| 错误信息 | 原因 | 修正方向 |
| --- | --- | --- |
| `First parameter of From must be list type` | From 参数用了字符串 | 改成 `From(["model"])` |
| `Invalid star expression` | `COUNT(*)` 不支持 | 改成 `COUNT(F('id'))` |
| `'From' object has no attribute 'Q'` | 把 Q 直接链在 From 上了 | 把 Q 放进 `Where(...)` |
| `name 'Eq' is not defined` | 使用了不存在的函数 | 改用 `Q(...)` 表达式 |
| `OrderBy.__init__() got an unexpected keyword argument 'desc'` | OrderBy 用了 `desc` 关键字 | 改为 `OrderBy((F("field"), -1))` |
| `'str' object has no attribute 'fieldId'` | 字段没用 `F()` 包装 | 改成 `F("field")` |
| `KeyError: 'XXX'` | 使用了不存在的函数 | 只用本文档列出的函数 |
| `Child[0] must be str` | Q 表达式第一个参数不是字符串 | 改成 `Q("field", "=", value)` |
| `tuple index out of range` | 函数参数个数不对 | 回头检查函数参数数量和顺序 |
| `'NoneType' object is not callable` | Q 操作符未注册或写错 | 回头检查操作符表 |
| `Argument must be Node` | Q 组合参数类型不对 | 改成 `Q(Qt1, Q.AND, Qt2)` 这类 Qt 组合 |
| `Function 'XXX' is not supported` | 用了不支持的函数 | 只用本文档列出的函数 |

## 字段表达式 F

字段必须显式列出，不支持 `Select("*", ...)`。

```python
F("fieldName")
F("t1.fieldName", "alias")
F(Formula("COUNT(F('id'))"), "cnt")
```

示例：

```python
Select([F("id"), F("name")], From(["models.Customer"]))
```

错误示例：

```python
Select("*", From(["models.Customer"]))
Select(["name"], From(["models.Customer"]))
```

## 数据源 From 与 Join

单模型：

```python
From(["models.Customer"])
From(["models.Customer", "t1"])
```

Join 示例：

```python
From(
  ["models.Order", "t1"],
  LeftJoin("models.Customer", "t2"),
  On([F("t1.customerId"), "=", F("t2.id")]),
)
```

注意：

- `From(...)` 的第一个参数是主模型描述
- Join 条件里的字段引用继续使用 `F(...)`
- `On(...)` 中的字段仍然要用 `F(...)`
- 关联字段不确定时，先用 `jit model get` 查模型定义

## Where / GroupBy / Having / Limit

过滤：

```python
Where(Q("status", "=", "active"))
```

分组与聚合筛选：

```python
GroupBy(F("deptId"))
Having(Q("cnt", ">", 5))
```

分页：

```python
Limit(0, 50)
```

注意：

- 不写筛选时可以省略 `Where(...)`
- 不写排序时可以省略 `OrderBy(...)`
- 如果只需要取前 N 条，仍然写 `Limit(0, N)`
- 使用聚合函数时，非聚合字段应进入 `GroupBy(...)`

## 排序 OrderBy

语法格式：`OrderBy((字段, 方向), ...)`

- `1` 表示升序
- `-1` 表示降序
- 参数必须是元组
- 排序字段优先用 `F(...)`
- 不要使用 `desc=True`
- 不支持 `Formula(...)` 直接排序

```python
OrderBy((F("createTime"), 1))
OrderBy((F("createTime"), -1))
OrderBy((F("status"), -1), (F("createTime"), 1))
```

错误示例：

```python
OrderBy(F("field"), desc=True)
OrderBy(F("field"))
OrderBy([(F("field"), 1)])
OrderBy((Formula("COUNT(F('id'))"), 1))
```

如果需求是“按聚合结果排序”或“TopN 聚合排行”，不要在同一层对聚合表达式排序，而是改成外层子查询后二次排序。

## Formula 速查

Formula 用于在 TQL 中嵌入计算逻辑。当前实现里，本文档按聚合、数学、日期、文本、逻辑、高级、地址七类整理了常用函数；字段引用统一写成 `F('fieldName')`。

```python
Formula("COUNT(F('id'))")
Formula("DATEADD(TODAY(), -30, 'D')")
Formula("POWER(F('score'), 0.5)")
```

高频提醒：

- `Formula("...")` 外层用双引号包整个表达式，字段引用写成 `F('field')`
- 统计记录数时，优先用 `COUNT(F('id'))`
- 如果你下意识写了 SQL 风格 `COUNT(*)`，改写为 `COUNT(F('id'))`
- `SQRT(...)` 不支持时，用 `POWER(x, 0.5)`
- 时间偏移常见写法：`DATEADD(TODAY(), -30, 'D')`

### 聚合函数

TQL 里要区分“行级计算”和“列聚合”：

| 用途 | 行级计算 | 列聚合 |
| --- | --- | --- |
| 求和 | `SUM(F('a'), F('b'))` | `COLSUM(F('amount'))` |
| 平均 | `AVG(F('a'), F('b'))` | `COLAVG(F('amount'))` |
| 最大 | `MAX(F('a'), F('b'))` | `COLMAX(F('amount'))` |
| 最小 | `MIN(F('a'), F('b'))` | `COLMIN(F('amount'))` |
| 计数 | 不适用 | `COUNT(F('id'))` |

常用聚合函数：

| 函数 | 用法 | 说明 |
| --- | --- | --- |
| `COUNT` | `Formula("COUNT(F('id'))")` | 计数 |
| `DISTINCT` | `Formula("DISTINCT(F('field'))")` | 去重计数 |
| `COLSUM` | `Formula("COLSUM(F('field'))")` | 列求和 |
| `COLAVG` | `Formula("COLAVG(F('field'))")` | 列平均值 |
| `COLMIN` | `Formula("COLMIN(F('field'))")` | 列最小值 |
| `COLMAX` | `Formula("COLMAX(F('field'))")` | 列最大值 |
| `FILL` | `Formula("FILL(F('field'))")` | 非空计数 |
| `NOTFILL` | `Formula("NOTFILL(F('field'))")` | 空值计数 |
| `SELECTED` | `Formula("SELECTED(F('field'))")` | 选中计数 |
| `NOTSELECTED` | `Formula("NOTSELECTED(F('field'))")` | 未选中计数 |
| `FIRSTROW` | `Formula("FIRSTROW(F('field'))")` | 首行 ID |
| `LASTROW` | `Formula("LASTROW(F('field'))")` | 末行 ID |
| `ROWID` | `Formula("ROWID()")` | 行号 |
| `MEDIAN` | `Formula("MEDIAN(F('field'))")` | 中位数 |
| `STDDEV` | `Formula("STDDEV(F('field'))")` | 标准差 |

说明：

- `MEDIAN`、`STDDEV` 的可用性可能受底层数据库类型影响
- 使用聚合函数时，如果没有 `GroupBy(...)`，非聚合字段可能返回空值

### 数学函数

| 函数 | 用法 | 说明 |
| --- | --- | --- |
| `ABS` | `Formula("ABS(F('field'))")` | 绝对值 |
| `ROUND` | `Formula("ROUND(F('field'), 2)")` | 四舍五入 |
| `TRUNCATE` | `Formula("TRUNCATE(F('field'), 2)")` | 截断 |
| `MOD` | `Formula("MOD(F('a'), F('b'))")` | 取模 |
| `POWER` | `Formula("POWER(F('field'), 2)")` | 幂运算 |
| `RANDOM` | `Formula("RANDOM()")` | 随机数 |

行级计算示例：

```python
Formula("F('price') * F('quantity')")
Formula("F('col1') + F('col2') + 100")
```

### 日期函数

| 函数 | 用法 | 说明 |
| --- | --- | --- |
| `YEAR` | `Formula("YEAR(F('date'))")` | 年初日期 |
| `YEARMONTH` | `Formula("YEARMONTH(F('date'))")` | 月初日期 |
| `YEARMONTHDAY` | `Formula("YEARMONTHDAY(F('date'))")` | 当日日期 |
| `YEARQUARTER` | `Formula("YEARQUARTER(F('date'))")` | 季度初日期 |
| `YEARWEEK` | `Formula("YEARWEEK(F('date'))")` | 周初日期 |
| `NOW` | `Formula("NOW()")` | 当前时间 |
| `TODAY` | `Formula("TODAY()")` | 今天 |
| `DATE` | `Formula("DATE(2026, 3, 21)")` | 构造日期 |
| `DATESTR` | `Formula("DATESTR(F('date'))")` | 日期字符串 |
| `DATEDELTA` | `Formula("DATEDELTA(F('start'), F('end'), 'D')")` | 日期差 |
| `DATEADD` | `Formula("DATEADD(F('date'), 7, 'D')")` | 日期加减 |
| `STRTODATE` | `Formula("STRTODATE('2026-01-01')")` | 字符串转日期 |
| `STRTODATETIME` | `Formula("STRTODATETIME('2026-01-01 12:00:00')")` | 字符串转时间 |
| `EXTRACT` | `Formula("EXTRACT(F('date'), 'Y')")` | 提取日期部分 |
| `MONTHDAYS` | `Formula("MONTHDAYS(F('date'))")` | 月天数 |
| `MONTHSTART` | `Formula("MONTHSTART(F('date'))")` | 月初 |
| `MONTHEND` | `Formula("MONTHEND(F('date'))")` | 月末 |
| `DAYOFYEAR` | `Formula("DAYOFYEAR(F('date'))")` | 年中第几天 |
| `WEEKOFYEAR` | `Formula("WEEKOFYEAR(F('date'))")` | 年中第几周 |
| `WEEKDAYNUM` | `Formula("WEEKDAYNUM(F('date'))")` | 星期几数字 |
| `WEEKDAYSTR` | `Formula("WEEKDAYSTR(F('date'))")` | 星期几字符串 |
| `TIMESTAMPFORMAT` | `Formula("TIMESTAMPFORMAT(F('ts'))")` | 时间戳转日期时间 |

`DATEDELTA` / `DATEADD` / `EXTRACT` 常见操作符：

| 操作符 | 说明 |
| --- | --- |
| `Y` | 年 |
| `Q` | 季度 |
| `M` | 月 |
| `D` | 日 |
| `W` | 周 |
| `H` | 小时 |
| `I` | 分钟 |
| `S` | 秒 |

这些操作符使用大写字母，避免写成小写或其他变体。

### 文本函数

| 函数 | 用法 | 说明 |
| --- | --- | --- |
| `CONCAT` | `Formula("CONCAT(F('a'), F('b'))")` | 连接字符串 |
| `LEFT` | `Formula("LEFT(F('field'), 3)")` | 左截取 |
| `RIGHT` | `Formula("RIGHT(F('field'), 3)")` | 右截取 |
| `MID` | `Formula("MID(F('field'), 1, 3)")` | 中间截取 |
| `LEN` | `Formula("LEN(F('field'))")` | 长度 |
| `REPLACE` | `Formula("REPLACE(F('field'), 'a', 'b')")` | 替换 |
| `INSERT` | `Formula("INSERT(F('field'), 1, 2, 'xx')")` | 插入字符串 |
| `TRIM` | `Formula("TRIM(F('field'))")` | 去空格 |
| `LOCATE` | `Formula("LOCATE('a', F('field'))")` | 查找位置 |
| `TOSTRING` | `Formula("TOSTRING(F('field'))")` | 转字符串 |
| `TONUMBER` | `Formula("TONUMBER(F('field'))")` | 转数字 |
| `IDCARDBIRTHDAY` | `Formula("IDCARDBIRTHDAY(F('idcard'))")` | 身份证生日 |
| `IDCARDSEX` | `Formula("IDCARDSEX(F('idcard'))")` | 身份证性别 |

### 逻辑函数

| 函数 | 用法 | 说明 |
| --- | --- | --- |
| `IF` | `Formula("IF(F('a') > 0, 'yes', 'no')")` | 条件判断 |
| `IFS` | `Formula("IFS(F('a')=1, '一', F('a')=2, '二')")` | 多条件判断 |
| `AND` | `Formula("AND(F('a')>0, F('b')>0)")` | 与 |
| `OR` | `Formula("OR(F('a')>0, F('b')>0)")` | 或 |
| `ISEMPTY` | `Formula("ISEMPTY(F('field'))")` | 是否为空 |
| `ISNOTEMPTY` | `Formula("ISNOTEMPTY(F('field'))")` | 是否非空 |
| `EMPTY` | `Formula("EMPTY(F('field'))")` | 空值判断 |
| `EMPTYSTR` | `Formula("EMPTYSTR(F('field'))")` | 空字符串判断 |
| `DEFAULTVALUE` | `Formula("DEFAULTVALUE(F('field'), 0)")` | 默认值 |

### 高级函数

| 函数 | 用法 | 说明 |
| --- | --- | --- |
| `ACC` | `Formula("ACC(F('amount'))")` | 累计 |
| `GROUPACC` | `Formula("GROUPACC(F('amount'))")` | 分组累计 |
| `RANK` | `Formula("RANK(F('score'))")` | 排名 |
| `GROUPRANK` | `Formula("GROUPRANK(F('score'))")` | 分组排名 |
| `CHAINRATIO` | `Formula("CHAINRATIO(F('amount'))")` | 环比 |
| `CHAININCREASE` | `Formula("CHAININCREASE(F('amount'))")` | 环比增量 |
| `CHAINPERIOD` | `Formula("CHAINPERIOD(F('amount'))")` | 环比周期 |
| `SAMERATIO` | `Formula("SAMERATIO(F('amount'))")` | 同比 |
| `SAMEINCREASE` | `Formula("SAMEINCREASE(F('amount'))")` | 同比增量 |
| `SAMEPERIOD` | `Formula("SAMEPERIOD(F('amount'))")` | 同比周期 |

### 地址函数

| 函数 | 用法 | 说明 |
| --- | --- | --- |
| `PROVINCE` | `Formula("PROVINCE(F('address'))")` | 提取省份 |
| `PROVINCECITY` | `Formula("PROVINCECITY(F('address'))")` | 提取省市 |
| `PROVINCECITYDISTRICT` | `Formula("PROVINCECITYDISTRICT(F('address'))")` | 提取省市区 |

## Q 表达式基础

Q 表达式用于构建筛选条件。

单条件：

```python
Q("field_name", "operator", value)
```

组合条件：

```python
Q(Qt1, Q.AND, Qt2)
Q(Qt1, Q.OR, Qt2)
```

语法要点：

- `field_name` 用字段 `name`，不是字段标题
- `field_name` 和 `operator` 必须写字符串
- 字符串值使用引号，数值不需要引号
- 关联字段可用双下划线连接，例如 `customer__dept__name`
- `Q.AND` 表示与，`Q.OR` 表示或

## Q 表达式包裹规则

单条件标准写法：

```python
Q("field", "=", value)
```

组合写法：

```python
Q(Q("field1", "=", value1), Q.AND, Q("field2", "=", value2))
Q(Q("status", "=", "pending"), Q.OR, Q("status", "=", "shipped"))
```

嵌套写法：

```python
Q(
  Q(Q("status", "=", "TRADE_SUCCESS"), Q.AND, Q("amount", ">", 100)),
  Q.OR,
  Q("status", "=", "TRADE_FINISHED"),
)
```

不推荐写法：

```python
Q(Q("field", "=", value))
```

Q 的常见组合规则：

| 写法 | 参数个数 | 说明 |
| --- | --- | --- |
| `Q()` | 0 | 返回空 Qt |
| `Q("f", "op", v)` | 3 | 单条件标准写法 |
| `Q(Qt)` | 1 | 直接返回该 Qt |
| `Q(Qt1, Qt2)` | 2 | 两个 Qt 做 AND 组合 |
| `Q(Qt1, Q.AND, Qt2)` | 3 | 显式 AND |
| `Q(Qt1, Q.OR, Qt2)` | 3 | 显式 OR |

## Q 操作符

| 类别 | 操作符 | 说明 | 示例 |
| --- | --- | --- | --- |
| 比较 | `=` | 等于 | `Q("status", "=", "active")` |
| 比较 | `!=` | 不等于 | `Q("status", "!=", "deleted")` |
| 比较 | `>` | 大于 | `Q("age", ">", 18)` |
| 比较 | `>=` | 大于等于 | `Q("score", ">=", 60)` |
| 比较 | `<` | 小于 | `Q("price", "<", 100)` |
| 比较 | `<=` | 小于等于 | `Q("stock", "<=", 10)` |
| 成员 | `in` | 在列表中 | `Q("status", "in", ["a", "b"])` |
| 成员 | `nin` | 不在列表中 | `Q("status", "nin", ["x", "y"])` |
| 模糊 | `like` | 包含 | `Q("name", "like", "张")` |
| 模糊 | `nlike` | 不包含 | `Q("name", "nlike", "test")` |
| 模糊 | `likeany` | 包含任一 | `Q("tags", "likeany", ["a", "b"])` |
| 模糊 | `nlikeany` | 不包含任一 | `Q("tags", "nlikeany", ["x"])` |
| 前后缀 | `startswith` | 以...开头 | `Q("code", "startswith", "ORD")` |
| 前后缀 | `endswith` | 以...结尾 | `Q("email", "endswith", "@qq.com")` |
| 范围 | `range` | 区间范围 | `Q("age", "range", [18, 60])` |
| 空值 | `isnull` | 是否为空 | `Q("deletedAt", "isnull", 1)` |
| 日期 | `year` | 年份匹配 | `Q("createTime", "year", 2025)` |
| 日期 | `month` | 月份匹配 | `Q("createTime", "month", 3)` |
| 日期 | `day` | 日期匹配 | `Q("createTime", "day", 15)` |
| 日期 | `week` | 周数匹配 | `Q("createTime", "week", 10)` |
| 地址 | `province` | 省份匹配 | `Q("address", "province", "广东")` |
| 地址 | `city` | 城市匹配 | `Q("address", "city", "深圳")` |
| 地址 | `district` | 区县匹配 | `Q("address", "district", "南山")` |
| 归属 | `belong` | 地址属于 | `Q("address", "belong", {"province": "广东"})` |
| 归属 | `nbelong` | 地址不属于 | `Q("address", "nbelong", {"province": "广东"})` |

说明：

- `isnull` 中 `1` 表示空，`0` 表示非空
- 使用未注册操作符时，可能出现 `'NoneType' object is not callable`

## Q 表达式常见模式

单条件：

```python
Q("age", ">", 18)
```

IN 列表：

```python
Q("status", "in", ["active", "pending"])
```

NOT IN：

```python
Q("status", "nin", ["deleted", "cancelled"])
```

数值范围：

```python
Q("age", "range", [18, 60])
```

时间范围：

必须使用完整格式 `YYYY-MM-DD HH:MM:SS`。

```python
Q("createTime", "range", ["2026-01-01 00:00:00", "2026-12-31 23:59:59"])
```

最近 30 天：

```python
Q("createTime", ">=", F(Formula("DATEADD(TODAY(), -30, 'D')")))
```

checkbox 选中 / 未选中：

```python
Q("is_checked", "isnull", 0)
Q("is_checked", "isnull", 1)
```

其中 `0` 表示非空，通常可视为已选中；`1` 表示空，通常可视为未选中。

关联字段：

```python
Q("customer__address__city", "=", "北京市")
```

## `jit model query` 示例

```bash
jit model query wanyun.crm.Customer --filter 'Q("name", "=", "Alice")' --page 1 --size 10
```

```bash
jit model query wanyun.crm.Customer --filter 'Q(Q("status", "=", "active"), Q.AND, Q("level", ">=", 3))' --page 1 --size 20
```

```bash
jit model query wanyun.crm.Customer --filter 'Q("createTime", "range", ["2026-01-01 00:00:00", "2026-01-31 23:59:59"])' --page 1 --size 20
```

## `jit model tql` 示例

简单查询：

```python
Select(
  [F("id"), F("name")],
  From(["models.Customer"]),
  Where(Q("name", "=", "Alice")),
  Limit(0, 10),
)
```

时间筛选：

```python
Select(
  [F("id"), F("title")],
  From(["models.Task"]),
  Where(Q("createTime", ">=", F(Formula("DATEADD(TODAY(), -30, 'D')")))),
  Limit(0, 20),
)
```

聚合统计：

```python
Select(
  [F("deptId"), F(Formula("COUNT(F('id'))"), "cnt")],
  From(["models.UserModel"]),
  GroupBy(F("deptId")),
  Having(Q("cnt", ">", 5)),
  Limit(0, 50),
)
```

聚合后外层排序：

```python
Select(
  [
    F("agg.deptId"),
    F("agg.totalAmount"),
  ],
  From([
    Select(
      [
        F("deptId"),
        F(Formula("COLSUM(F('amount'))"), "totalAmount"),
      ],
      From(["models.OrderModel"]),
      GroupBy(F("deptId")),
    ),
    "agg",
  ]),
  OrderBy((F("agg.totalAmount"), -1)),
  Limit(0, 10),
)
```

## 常见错误

| 错误写法 | 正确写法 |
| --- | --- |
| `SELECT * FROM table` | 改成 `Select(...)`、`From([...])` 这类 TQL 结构 |
| `From("models.Customer")` | `From(["models.Customer"])` |
| `Limit(10)` | `Limit(0, 10)` |
| `Select(["name"], ...)` | `Select([F("name")], ...)` |
| `Q("a", "=", 1) & Q("b", "=", 2)` | `Q(Q("a", "=", 1), Q.AND, Q("b", "=", 2))` |
| 在 `--filter` 里塞原始 SQL | 传入 Q 表达式字符串 |
| `OrderBy(F("field"), desc=True)` | `OrderBy((F("field"), -1))` |
| `OrderBy(F("field"))` | `OrderBy((F("field"), 1))` |
| `From([...]).Q(...)` | `Where(Q(...))` |
| `COUNT(*)` | `COUNT(F('id'))` |
| `OrderBy((Formula("COUNT(...)"), 1))` | 外层子查询后二次排序 |

常见报错与修正方向：

- `First parameter of From must be list type`
  改成 `From(["model"])`
- `Invalid star expression`
  把 `COUNT(*)` 改成 `COUNT(F('id'))`
- `'From' object has no attribute 'Q'`
  把 Q 放进 `Where(...)`
- `OrderBy.__init__() got an unexpected keyword argument 'desc'`
  用 `OrderBy((F("field"), -1))`
- `'str' object has no attribute 'fieldId'`
  检查是否忘了用 `F("field")`
- `'NoneType' object is not callable`
  检查 Q 操作符是否写错

## 推荐工作流

1. 先执行 `jit app refresh`
2. 再用 `jit model get <fullModelName>` 检查模型和字段
3. 从一个字段、一个条件开始
4. 先确认最小查询能跑通
5. 基础查询正确后，再增加字段、筛选、排序、聚合或 limit
6. 如果需求涉及聚合排序或 TopN，优先改成外层子查询后二次排序
