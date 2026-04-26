# gpt2api 账号池分阶段改造设计

**日期**: 2026-04-25

## 1. 背景

当前仓库已经有基础账号池能力,但本质上还是“单一全局账号列表 + 统一调度”:

- 账号存储、导入、刷新、额度探测已经存在,见 `/Users/swimmingliu/data/github-proj/gpt2api/internal/account/`
- 调度器当前只会从全体 `healthy` 账号里挑选可用账号,见 `/Users/swimmingliu/data/github-proj/gpt2api/internal/scheduler/scheduler.go`
- 模型只有“对外 slug -> 上游 slug”映射,没有“模型 -> 账号池”路由,见 `/Users/swimmingliu/data/github-proj/gpt2api/internal/model/model.go`
- 后台账号页已有导入、刷新、探测、代理绑定,但没有“账号池 / 分组 / 路由治理”维度,见 `/Users/swimmingliu/data/github-proj/gpt2api/web/src/views/admin/Accounts.vue`

这意味着:

1. 图片号、聊天号、Codex 号无法稳定隔离
2. 不同模型不能固定命中指定账号池
3. “高质量号 / 廉价号 / 测试号 / 保活号”之类运营策略无法表达
4. 现有调度只能基于账号状态和冷却,不能做池级治理

目标是把现有“账号列表”升级成接近 sub2api 风格的“可治理账号池”,但不推翻现在的 Go 服务结构。

## 2. 目标

### 2.1 本次总目标

在保留现有账号导入、刷新、额度探测、代理绑定能力的前提下,新增一层显式的账号池治理能力,让系统支持:

- 账号池定义
- 账号加入多个池
- 模型固定路由到指定池
- 池内调度策略扩展
- 池维度后台管理和统计
- 批量导入/批量分配/批量治理
- 会话粘性

### 2.2 非目标

本轮不做:

- 多平台统一资源池(OpenAI API Key / Claude / Gemini 混合调度)
- 重写整个 gateway / billing / proxy 架构
- 推翻现有 `oai_accounts` 表和刷新逻辑

## 3. 方案比较

### 方案 A: 在现有账号体系上增加“账号池 + 路由 + 调度扩展”层（推荐）

做法:

- 保留 `oai_accounts`
- 新增账号池表、池成员表、模型到池的路由表
- 调度器从“全局选账号”改成“先解析目标池,再在池内选账号”
- 后台新增账号池管理页和模型路由能力

优点:

- 兼容当前结构,风险最低
- 能分阶段上线
- Phase 1 就能得到明显收益

缺点:

- 调度器会比现在复杂
- 需要补一组新的管理 API 和前端页面

### 方案 B: 复用现有字段硬塞“池”概念

做法:

- 给账号表直接加一个 `pool` 字段
- 给模型表直接加一个 `pool` 字段
- 一个账号只能属于一个池

优点:

- 开发速度最快

缺点:

- 一账号多用途场景直接被锁死
- 后续做高质量池 / 兜底池 / 交叉池会很痛苦
- 很快会演变成技术债

### 方案 C: 直接重写成通用资源调度中心

做法:

- 单独抽象 provider / account / capability / route / scheduler
- 先统一资源模型,再让 gateway 全量迁移

优点:

- 理论扩展性最强

缺点:

- 超出当前任务范围
- 改动面过大,不适合在现有仓库里一步到位

### 推荐

采用 **方案 A**。它最符合“先把当前项目改造成可治理账号池,再逐步增强”的目标。

## 4. 总体设计

### 4.1 核心建模

新增三个核心实体:

1. **账号池** `account_pools`
   - 定义一个可调度资源池
   - 例如: `image-main`, `chat-main`, `codex-main`, `image-premium`, `image-fallback`

2. **池成员** `account_pool_members`
   - 定义账号和池的关系
   - 支持一个账号属于多个池
   - 在成员层承载池内调度参数,例如权重、优先级、是否启用

3. **模型路由** `model_pool_routes`
   - 把模型绑定到一个主池
   - 可选兜底池
   - Gateway 收到请求后先解析模型路由,再进入调度

### 4.2 调度责任拆分

调度路径改成:

1. Gateway 根据请求模型查出目标池
2. 调度器只在该池的成员账号里选可用账号
3. 池内再叠加当前已有规则:
   - 账号状态
   - cooldown
   - token 过期
   - 今日限额
   - 最小间隔
4. Phase 2 再加:
   - 池内权重
   - 池内优先级
   - 会话粘性
   - 成员级并发上限

### 4.3 与现有模块的边界

- `/internal/account/` 继续负责账号本体、导入、刷新、探测
- 新增 `/internal/accountpool/` 负责池、成员、路由
- `/internal/scheduler/` 负责“按池调度”
- `/internal/gateway/` 负责“从模型解析目标池”
- `/web/src/views/admin/` 新增账号池与路由管理页面

## 5. 数据模型设计

### 5.1 account_pools

建议字段:

- `id`
- `code` 唯一编码,供内部路由使用
- `name`
- `pool_type` 例如 `chat` / `image` / `codex` / `mixed`
- `description`
- `enabled`
- `dispatch_strategy` 先只支持 `least_recently_used`,后续扩展
- `sticky_ttl_sec`
- `created_at`
- `updated_at`
- `deleted_at`

### 5.2 account_pool_members

建议字段:

- `id`
- `pool_id`
- `account_id`
- `enabled`
- `weight`
- `priority`
- `max_parallel`
- `note`
- `created_at`
- `updated_at`

约束:

- `(pool_id, account_id)` 唯一

### 5.3 model_pool_routes

建议字段:

- `id`
- `model_id`
- `pool_id`
- `fallback_pool_id`
- `enabled`
- `created_at`
- `updated_at`

约束:

- `model_id` 唯一

### 5.4 为什么不用直接给账号表加 pool_id

因为这会把“一账号多池”直接堵死。后续一旦出现:

- 某账号同时可跑 `chat` 和 `codex`
- 某高质量账号既在 `image-main` 又在 `image-premium`
- 某账号暂时只从一个池禁用,但在另一个池继续可用

单字段方案就不够用了。

## 6. API 与后台设计

### 6.1 后端 API

新增 `/api/admin/account-pools`:

- `GET /api/admin/account-pools`
- `POST /api/admin/account-pools`
- `GET /api/admin/account-pools/:id`
- `PATCH /api/admin/account-pools/:id`
- `DELETE /api/admin/account-pools/:id`
- `GET /api/admin/account-pools/:id/members`
- `POST /api/admin/account-pools/:id/members`
- `PATCH /api/admin/account-pools/:id/members/:memberId`
- `DELETE /api/admin/account-pools/:id/members/:memberId`

新增模型路由 API:

- `GET /api/admin/account-pool-routes`
- `PUT /api/admin/account-pool-routes/:modelId`
- `DELETE /api/admin/account-pool-routes/:modelId`

扩展现有账号 API:

- 账号列表支持 `pool_id` 过滤
- 批量导入时支持默认加入某个池
- 后续批量操作支持“加入池 / 移出池 / 切换成员启停状态”

### 6.2 前端后台

新增页面:

- `账号池管理`
- `模型池路由`

扩展现有 `Accounts.vue`:

- 支持按池筛选
- 展示账号所属池
- 从账号侧快速加入/移出池
- 批量导入时可指定默认池

## 7. 运行时行为设计

### 7.1 请求路由

新链路:

1. 请求进入 `/v1/chat/completions` 或 `/v1/images/generations`
2. 根据 `model.slug` 查询 `model_pool_routes`
3. 若模型未配置池:
   - Phase 1 默认走“全局兼容模式”
   - 即从所有账号里调度,保持旧行为
4. 若模型配置了池:
   - 仅从该池成员里调度
5. 若主池没有可用账号且配置了兜底池:
   - 再尝试兜底池

### 7.2 与现有账号状态机的关系

账号健康状态仍然保留在账号本体层:

- `healthy`
- `warned`
- `throttled`
- `suspicious`
- `dead`

池成员层只表达“这个账号在这个池里是否可参与调度”,不替代账号本体状态。

### 7.3 会话粘性（Phase 2）

粘性键建议这样构造:

- 优先取显式 header,例如后续可支持 `X-Session-Sticky-Key`
- 否则按 `api_key_id + user_id + model + conversation_id/task_id`

Redis 记录:

- `sticky:{pool_id}:{key} -> account_id`

行为:

- 账号仍可用就继续命中
- 账号不可用则自动失效并重选

## 8. 分阶段实施

### Phase 1: 账号池基础设施

目标:

- 先把“账号池”做成一等公民
- 让模型可以固定路由到账号池
- 让调度器按池选账号

范围:

- 表结构: `account_pools` / `account_pool_members` / `model_pool_routes`
- 后端: DAO / Service / Handler / Router
- 调度器: 支持 pool 约束
- Gateway: 按模型解析目标池
- 前端: 账号池管理页 + 模型池路由页 + 账号列表池筛选

交付后效果:

- 图片模型走图片池
- 聊天模型走聊天池
- Codex 模型走 Codex 池
- 现有账号导入/刷新/探测逻辑继续可用

### Phase 2: 池内智能调度

目标:

- 把“能用”升级成“好用”

范围:

- 成员级权重、优先级、最大并发
- 会话粘性
- 最近失败计数与自动冷却
- 池健康度统计
- 批量启停 / 批量加池 / 批量移池

交付后效果:

- 同一会话尽量固定账号
- 高质量账号优先被特定模型或特定流量命中
- 坏号自动降权/熔断,不拖垮整池

### Phase 3: 运营与导入增强

目标:

- 把池变成可运营对象

范围:

- 导入时携带池信息
- 导入后批量归池
- 池快照导出/导入
- 池维度统计视图
- 审计日志补充池变更明细

交付后效果:

- 新号导入后可以快速归类
- 运营和运维动作可以围绕“池”来做,不是围绕“单号”硬操作

## 9. 错误处理与兼容策略

### 9.1 兼容旧逻辑

如果模型没有配置账号池路由,默认继续走旧调度逻辑。这保证:

- 升级后不会立刻影响现网
- 可以分模型、分池渐进迁移

### 9.2 主池不可用

优先级:

1. 主池可用账号
2. 兜底池可用账号
3. 返回 `no_available_account`

### 9.3 池被禁用

- 后台禁止把模型路由到禁用池
- 运行期若路由配置指向禁用池,视为无可用池

## 10. 测试策略

### 10.1 后端

- DAO 测试:
  - 池 CRUD
  - 成员绑定/解绑
  - 模型路由查询
- Service 测试:
  - 唯一性和参数校验
  - 批量加池/移池
- Scheduler 测试:
  - 仅从指定池选账号
  - 主池失败兜底池
  - 非成员账号不可被命中
- Gateway 测试:
  - 模型命中正确池
  - 无路由回退旧行为

### 10.2 前端

- 账号池页面基础交互
- 模型路由页面保存/回显
- 账号列表按池筛选

### 10.3 回归

必须验证:

- 现有账号导入不退化
- 现有刷新/探测不退化
- 图片生成主流程不退化
- 聊天主流程不退化

## 11. 推荐执行顺序

按下面顺序推进:

1. **Phase 1**
2. **Phase 1 全量测试 + 前端构建 + smoke**
3. **Phase 2**
4. **Phase 2 全量测试 + 前端构建 + smoke**
5. **Phase 3**

理由:

- 没有 Phase 1,后面的“粘性 / 权重 / 批量治理”都没有落点
- Phase 1 完成后,项目就已经从“账号列表”升级成“账号池”

## 12. 本次执行决策

本轮按用户要求,不再反复征求阶段确认,直接执行推荐路线:

- 先完成 Phase 1
- Phase 1 通过后再进入 Phase 2
- 每一阶段结束后做完整验证,再继续下一阶段
