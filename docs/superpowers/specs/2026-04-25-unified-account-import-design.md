# gpt2api 统一账号导入设计

**日期**: 2026-04-25

## 1. 背景

当前仓库已经有完整的 ChatGPT 账号管理基础设施:

- 账号存储、加密、刷新、额度探测位于 `/Users/swimmingliu/data/github-proj/gpt2api/internal/account/`
- 图片执行链路位于 `/Users/swimmingliu/data/github-proj/gpt2api/internal/image/` 与 `/Users/swimmingliu/data/github-proj/gpt2api/internal/upstream/chatgpt/`
- 后台账号页位于 `/Users/swimmingliu/data/github-proj/gpt2api/web/src/views/admin/Accounts.vue`
- 账号池基础表与 API 已经存在,位于 `/Users/swimmingliu/data/github-proj/gpt2api/internal/accountpool/`

但当前“导入账号”能力仍然是来源分散、业务分散的状态:

1. 手动新增走 `/api/admin/accounts`
2. 文本 token 导入走 `/api/admin/accounts/import-tokens`
3. JSON / 文件导入走 `/api/admin/accounts/import`
4. 各入口虽然最终都落到 `oai_accounts`,但来源解析、身份补全、upsert、后处理没有统一的内核边界

同时,目标能力已经明确:

- 支持 `access token` 文本导入
- 支持 CPA 文件导入
- 支持 sub2api 导入
- 支持手动新增导入
- 导入后统一进入当前项目的账号池、刷新器、额度探测器、图片 API 网关体系

本设计要解决的核心问题不是“新增一个导入按钮”,而是把“多来源账号进入系统”的路径统一成一套可维护、可扩展、可测试的导入主流程。

## 2. 目标

### 2.1 总目标

在不推翻当前账号数据库模型、不破坏现有后台路由兼容性的前提下,建立一套**统一导入内核 + 多来源适配器**体系,让以下来源都能稳定导入到当前项目:

- `access token` 文本 / TXT
- CPA 文件
- sub2api 本地 JSON
- 手动新增

并在导入成功后统一支持:

- email 维度 upsert
- 敏感字段加密入库
- 默认代理绑定
- 导入到指定账号池
- kick AT 刷新
- kick 图片额度探测

### 2.2 非目标

本轮不做:

- 远程 CPA 浏览导入
- 远程 sub2api 浏览导入
- 导入任务中心 / 导入历史表
- 大批量异步任务框架
- 完整迁移 `chatgpt2api` 的浏览器指纹字段(`fp` / `impersonate` / `sec-ch-ua`)
- 重写现有 `oai_accounts` / `refresher` / `quota prober` 体系

这些能力应作为 Phase 2 或 Phase 3 增量功能,不进入本轮 MVP。

## 3. 方案比较

### 方案 A: 每种来源各写一套独立导入逻辑

做法:

- `access token` 文本单独一套
- CPA 单独一套
- sub2api 单独一套
- 手动新增继续独立

优点:

- 初期改动看起来最小

缺点:

- 去重、身份补全、upsert、加池、后处理规则会重复实现
- 后续扩展远程来源时会继续发散
- 任何导入规则修改都要同步改多处

不推荐。

### 方案 B: 统一导入内核 + 多来源适配器（推荐）

做法:

- 每种来源只负责“解析原始输入 → 统一候选结构”
- 统一导入内核负责“标准化 → 身份补全 → 去重 → upsert → 后处理”
- 保留现有 HTTP 路由,内部全部改走统一内核

优点:

- 规则集中,后续维护成本最低
- 最适合当前仓库已经存在的账号、代理、账号池、刷新器结构
- 远程来源后续只要多写 adapter,不用再碰核心导入规则

缺点:

- 首次重构需要抽出边界,不是简单补代码

推荐采用。

### 方案 C: 一步到位引入导入任务系统

做法:

- 先做统一导入,同时上导入任务表、异步执行、进度轮询、失败重试

优点:

- 扩展性最好

缺点:

- 超出当前任务范围
- 数据库、前端、任务编排复杂度一起抬高
- 远程来源还没落地前就上任务框架,投入产出比不高

不作为当前实现方案。

## 4. 总体设计

### 4.1 设计原则

1. **来源解析与业务处理解耦**
   - 来源只负责把原始数据解析成统一结构
   - 统一内核负责真正业务处理

2. **数据库仍是唯一事实源**
   - 所有导入来源最终统一进入 `oai_accounts`
   - 不把 `chatgpt2api` 的 JSON 存储模式引入当前系统

3. **email 继续作为账号实体唯一键**
   - 当前项目是“账号实体池”,不是“纯 token 池”
   - upsert 继续按 email 做

4. **手动新增也走统一导入链路**
   - 不再保留一套完全平行的账号创建规则

5. **Phase 1 不新增数据库表**
   - 直接复用已有的 `oai_accounts` / `account_proxy_bindings` / `account_pool_members`

### 4.2 模块边界

新增两个目录:

- `/Users/swimmingliu/data/github-proj/gpt2api/internal/account/importcore/`
- `/Users/swimmingliu/data/github-proj/gpt2api/internal/account/importsource/`

职责划分:

- `importsource`: 原始输入解析
- `importcore`: 导入编排、标准化、补全、upsert、后处理
- `account.Service`: 继续负责账号领域的加密 / 解密 / 基础 CRUD
- `refresher` / `quota prober`: 继续负责运行态修复和额度探测
- `accountpool.Service`: 继续负责账号池成员维护

## 5. 统一中间模型

### 5.1 ImportCandidate

所有导入来源最终都要转换成统一候选结构:

```go
type ImportCandidate struct {
    SourceType       string
    SourceRef        string

    AccessToken      string
    RefreshToken     string
    SessionToken     string

    Email            string
    ClientID         string
    ChatGPTAccountID string
    AccountType      string
    PlanType         string

    TokenExpiresAt   *time.Time
    OAISessionID     string
    OAIDeviceID      string
    Cookies          string
    Notes            string
}
```

约束:

- `AccessToken` 是本轮 Phase 1 的强必填
- `SourceType` 必须明确,用于结果展示和调试
- `SourceRef` 必须带来源定位信息,例如 `line:3` / `file:foo.json`

### 5.2 ImportOptions

统一导入内核需要一套公共选项:

```go
type ImportOptions struct {
    UpdateExisting bool
    DefaultProxyID uint64
    TargetPoolID   uint64

    ResolveIdentity bool
    KickRefresh     bool
    KickQuotaProbe  bool
}
```

默认值:

- `UpdateExisting = true`
- `ResolveIdentity = true`
- `KickRefresh = true`
- `KickQuotaProbe = true`

### 5.3 ImportResult

统一返回批量结果:

```go
type ImportResult struct {
    Total   int
    Created int
    Updated int
    Skipped int
    Failed  int

    Results []ImportLineResult
}

type ImportLineResult struct {
    Index      int
    SourceType string
    SourceRef  string
    Email      string
    Status     string
    Reason     string
    ID         uint64
}
```

`Status` 仅允许:

- `created`
- `updated`
- `skipped`
- `failed`

## 6. 多来源适配器设计

### 6.1 Access Token 文本 / TXT

新增:

- `/Users/swimmingliu/data/github-proj/gpt2api/internal/account/importsource/access_token.go`

输入:

- 多行文本
- `.txt` 文件内容

规则:

- 按 `\r?\n` 拆行
- trim 后过滤空行
- 每行构造一个 `ImportCandidate`
- `SourceType = "access_token_text"`
- `SourceRef = "line:<n>"`

输出字段:

- 只填 `AccessToken`

### 6.2 CPA 文件

新增:

- `/Users/swimmingliu/data/github-proj/gpt2api/internal/account/importsource/cpa.go`

输入:

- 单文件单 JSON
- 多文件批量 JSON

优先提取字段:

- `access_token`
- `accessToken`
- `refresh_token`
- `session_token`
- `email`
- `client_id`
- `chatgpt_account_id`
- `account_id`
- `oai_session_id`
- `oai_device_id`
- `cookies`

映射规则:

- `SourceType = "cpa_file"`
- `SourceRef = "file:<filename>"`

未提供 email 时允许继续导入,由统一内核后续补全。

### 6.3 sub2api 本地 JSON

新增:

- `/Users/swimmingliu/data/github-proj/gpt2api/internal/account/importsource/sub2api_json.go`

输入:

- 顶层对象包含 `accounts[]`

优先提取字段:

- `credentials.access_token`
- `credentials.refresh_token`
- `credentials.session_token`
- `credentials.client_id`
- `credentials.chatgpt_account_id`
- `extra.email`
- `name`
- `platform`

规则:

- `SourceType = "sub2api_json"`
- `SourceRef = "account:<index>"`
- `AccountType` 继续沿用当前仓库对 `name/platform` 的推断逻辑

### 6.4 手动新增

新增:

- `/Users/swimmingliu/data/github-proj/gpt2api/internal/account/importsource/manual.go`

输入:

- 现有后台表单 body

规则:

- 直接包装成一个 `ImportCandidate`
- `SourceType = "manual"`
- `SourceRef = "admin_form"`

手动新增不再直接自己写库,而是调用统一导入内核。

## 7. 统一导入内核执行流程

### 7.1 阶段 1: 标准化

职责:

- trim 全部字段
- 过滤空 token
- 统一 `account_type` 和 `plan_type` 的缺省值
- 为每条候选保留原始索引

标准化阶段只做格式清洗,不访问数据库,不打远程请求。

### 7.2 阶段 2: 身份补全

职责:

- 优先信任来源里已有的 `Email` / `ChatGPTAccountID` / `TokenExpiresAt`
- 如果为空,尝试从 JWT claims 解析:
  - `email`
  - `chatgpt_account_id`
  - `exp`
- 仍然拿不到 `Email` 时,才访问上游补全:
  - `/backend-api/me`
  - 必要时 `/backend-api/conversation/init`

规则:

- 导入入库前必须拿到 `Email`
- 拿不到 email 的记录标记为 `failed`

### 7.3 阶段 3: 冲突检查与同批去重

规则:

1. 同一批记录按最终 `email` 去重
2. 同邮箱保留最后一条
3. 如果来源提供的 `email` 与 JWT/远程补全出来的 `email` 不一致:
   - 当前记录直接 `failed`
   - 不自动覆盖

### 7.4 阶段 4: 数据库 upsert

规则:

- 以 `email` 查询现有账号
- 不存在:
  - `created`
- 存在且 `UpdateExisting = false`:
  - `skipped`
- 存在且 `UpdateExisting = true`:
  - `updated`

字段覆盖策略:

- 新值非空 → 覆盖旧值
- 新值为空 → 保留旧值

必须继续复用当前项目的加密逻辑:

- `auth_token`
- `refresh_token`
- `session_token`
- `cookies`

状态规则:

- 已有账号为 `dead` 或 `suspicious`
- 本次导入了新的有效 AT
- 更新后恢复为 `healthy`

### 7.5 阶段 5: 后处理

导入成功后统一支持:

1. 默认代理绑定
   - `DefaultProxyID > 0` 时为新建账号绑定默认代理
   - 已有账号默认不覆盖现有代理绑定

2. 导入到账号池
   - `TargetPoolID > 0` 时自动写入 `account_pool_members`
   - 默认成员参数:
     - `enabled = true`
     - `weight = 100`
     - `priority = 100`
     - `max_parallel = 1`

3. kick 后台任务
   - `KickRefresh = true` → `refresher.Kick()`
   - `KickQuotaProbe = true` → `prober.Kick()`

### 7.6 阶段 6: 结果汇总

每条记录都必须写入 `ImportLineResult`,包括:

- 成功创建
- 成功更新
- 邮箱已存在被跳过
- token 缺字段失败
- email 冲突失败
- 远程补全失败

不能因为单条记录失败而中断整批导入。

## 8. 接口设计

### 8.1 保留并扩展现有接口

继续保留:

- `POST /api/admin/accounts`
- `POST /api/admin/accounts/import`
- `POST /api/admin/accounts/import-tokens`

内部统一改走 `importcore.Import(...)`。

### 8.2 手动新增

`POST /api/admin/accounts` 继续返回单个 `Account`,但内部流程改为:

1. 把表单 body 转成单个 `ImportCandidate`
2. 走统一导入内核
3. 返回创建或更新后的账号对象

可新增可选字段:

- `update_existing`
- `target_pool_id`
- `kick_refresh`
- `kick_quota_probe`

### 8.3 Token 文本导入

`POST /api/admin/accounts/import-tokens` 继续支持:

- `mode = at | rt | st`
- `tokens = string | string[]`

扩展可选字段:

- `default_proxy_id`
- `target_pool_id`
- `kick_refresh`
- `kick_quota_probe`
- `resolve_identity`

返回体继续兼容当前 `ImportSummary` 风格。

### 8.4 本地 JSON / 文件导入

`POST /api/admin/accounts/import` 继续支持:

- `application/json`
- `multipart/form-data`

扩展可选字段:

- `source_kind = auto | cpa_file | sub2api_json | token_file`
- `target_pool_id`
- `kick_refresh`
- `kick_quota_probe`
- `resolve_identity`

## 9. 前端设计

### 9.1 统一导入弹窗

在 `/Users/swimmingliu/data/github-proj/gpt2api/web/src/views/admin/Accounts.vue` 中保留一个“导入账号”入口,打开统一弹窗。

弹窗拆分为以下 pane:

- `Access Token`
- `CPA 文件`
- `sub2api`
- `手动新增`

### 9.2 共用高级选项

无论当前选择哪种导入来源,都显示同一组高级选项:

- 邮箱已存在则更新
- 默认代理
- 导入到账号池
- 导入后立即刷新
- 导入后立即探测额度

### 9.3 结果展示

导入完成后统一展示:

- 成功 / 失败总数
- 结果列表
- 失败原因
- 仅查看失败项

前端不应该按来源分裂出不同结果组件。

## 10. 文件改造范围

### 10.1 新增后端文件

- `/Users/swimmingliu/data/github-proj/gpt2api/internal/account/importcore/types.go`
- `/Users/swimmingliu/data/github-proj/gpt2api/internal/account/importcore/service.go`
- `/Users/swimmingliu/data/github-proj/gpt2api/internal/account/importcore/identity.go`
- `/Users/swimmingliu/data/github-proj/gpt2api/internal/account/importcore/persist.go`
- `/Users/swimmingliu/data/github-proj/gpt2api/internal/account/importcore/postprocess.go`
- `/Users/swimmingliu/data/github-proj/gpt2api/internal/account/importsource/access_token.go`
- `/Users/swimmingliu/data/github-proj/gpt2api/internal/account/importsource/cpa.go`
- `/Users/swimmingliu/data/github-proj/gpt2api/internal/account/importsource/sub2api_json.go`
- `/Users/swimmingliu/data/github-proj/gpt2api/internal/account/importsource/manual.go`

### 10.2 重构后端文件

- `/Users/swimmingliu/data/github-proj/gpt2api/internal/account/handler.go`
- `/Users/swimmingliu/data/github-proj/gpt2api/internal/account/importer.go`
- `/Users/swimmingliu/data/github-proj/gpt2api/internal/account/importer_tokens.go`
- `/Users/swimmingliu/data/github-proj/gpt2api/cmd/server/main.go`

### 10.3 重构前端文件

- `/Users/swimmingliu/data/github-proj/gpt2api/web/src/views/admin/Accounts.vue`
- `/Users/swimmingliu/data/github-proj/gpt2api/web/src/api/accounts.ts`

建议新增前端组件目录:

- `/Users/swimmingliu/data/github-proj/gpt2api/web/src/components/admin/account-import/`

## 11. Phase 1 实现范围

### 11.1 必做

- 统一导入内核
- Access Token 文本 / TXT
- CPA 文件
- sub2api 本地 JSON
- 手动新增复用统一导入
- 默认代理绑定
- 导入到账号池
- kick 刷新 / 探测

### 11.2 不做

- 远程 CPA
- 远程 sub2api
- 导入历史
- 导入任务系统
- 指纹字段完整迁移

## 12. 测试策略

### 12.1 单元测试

必须覆盖:

- access token 文本解析
- CPA JSON 解析
- sub2api JSON 解析
- JWT claims 补全
- email 冲突处理
- `UpdateExisting = true / false`
- `TargetPoolID` 导入加池

### 12.2 Handler 测试

必须覆盖:

- `POST /api/admin/accounts`
- `POST /api/admin/accounts/import`
- `POST /api/admin/accounts/import-tokens`

并验证旧返回字段没有丢失。

### 12.3 集成验证

必须验证:

1. 导入后 `oai_accounts` 有数据
2. 指定代理后 `account_proxy_bindings` 有数据
3. 指定池后 `account_pool_members` 有数据
4. 导入后能进入调度器候选集

### 12.4 仓库级验证

如果进入实现,默认要跑:

- `rtk go vet ./...`
- `rtk go test ./...`
- `rtk sh -lc 'cd /Users/swimmingliu/data/github-proj/gpt2api/web && npm run build'`

如果最终改动形成真实交付状态,再补:

- `rtk sh -lc 'cd /Users/swimmingliu/data/github-proj/gpt2api && docker compose -f deploy/docker-compose.yml up -d --wait'`
- `rtk sh -lc 'cd /Users/swimmingliu/data/github-proj/gpt2api/scripts && npm run smoke:docker'`

## 13. MVP 验收标准

本轮 MVP 完成标准:

1. `/Users/swimmingliu/Downloads/chatgpt账号.txt` 这类一行一个 AT 的文本可以导入
2. 导入后可以指定加入账号池
3. CPA 文件和 sub2api 本地 JSON 可以通过统一弹窗导入
4. 手动新增不再绕开统一导入规则
5. 导入后账号可被刷新 / 探测
6. 至少一条导入账号能够通过当前图片 API 闭环跑通:
   - `POST /v1/images/generations`

## 14. 风险与缓解

### 风险 1: 部分 access token 无法本地解析 email

缓解:

- 先用 JWT 补全
- 不足时再走上游补全
- 仍然失败则只失败该条,不影响整批

### 风险 2: 手动新增改走统一导入后影响旧前端

缓解:

- 保持 `POST /api/admin/accounts` 路由与返回体不变
- 增加 handler 兼容测试

### 风险 3: 导入加池失败导致整批回滚逻辑复杂

缓解:

- 把“账号入库成功”和“后处理失败”分开记录
- 后处理失败写 `warning`,不否定主导入结果

### 风险 4: Phase 1 做太满导致交付延迟

缓解:

- 严格限制 Phase 1 只做本地 4 来源
- 远程来源全部后置

## 15. 推荐结论

本轮应按 **方案 B: 统一导入内核 + 多来源适配器** 实施。

先完成本地四来源统一导入与账号池接入,把“来源兼容问题”与“核心导入业务逻辑”彻底解耦。这样后续要支持 `chatgpt2api` 风格的远程来源时,只需新增来源适配器与浏览接口,不需要重新碰导入内核、账号加密、账号池后处理和图片 API 主链路。
