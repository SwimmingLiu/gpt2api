# 01. gpt2api 图片账号池化精简改造需求文档

**日期**: 2026-04-26

## 0. 共享契约摘要

本节作为后续并行任务的**权威边界摘要**。若其他实现文档、旧 README、旧代码注释与本节冲突,以本节为准。

### 0.1 最终保留面

- **对外接口**
  - `POST /v1/images/generations`
  - `GET /v1/models`
  - `GET /p/img/:task_id/:idx`
- **后台能力**
  - 管理员登录
  - 账号管理
  - 代理管理
  - 账号池 / 池路由管理
  - 最小系统设置
- **核心运行时**
  - `gpt-image-2` 图片生成主链路
  - ChatGPT 上游逆向客户端
  - 账号导入 / 编辑 / 刷新 / 额度探测
  - 池内调度与图片代理下载

### 0.2 最终移除面

- **OpenAI 兼容接口**
  - `POST /v1/chat/completions`
  - `POST /v1/images/edits`
  - `GET /v1/images/tasks/:id`
- **后台 / 用户侧**
  - `/api/auth/register`
  - `/api/me/**`
  - `/api/keys/**`
  - `/api/recharge/**`
  - `/api/admin/users/**`
  - `/api/admin/credits/**`
  - `/api/admin/groups/**`
  - `/api/admin/usage/**`
  - `/api/admin/recharge/**`
  - `/api/admin/audit/**`
  - `/api/admin/system/backup/**`
  - `/api/admin/models/**`
  - `/api/admin/account-import-sources/**`

### 0.3 运行时硬约束

- **鉴权模型**
  - 后台只保留管理员 JWT 登录
  - `/v1/*` 统一使用实例级静态 Bearer Token
  - 不再保留“普通用户 -> API Key -> quota -> billing”链路
- **必须保留的数据表**
  - `oai_accounts`
  - `oai_account_cookies`
  - `account_proxy_bindings`
  - `proxies`
  - `account_pools`
  - `account_pool_members`
  - `model_pool_routes`
  - `image_tasks`
  - `system_settings`
- **主入口必须停止初始化的模块**
  - `internal/billing`
  - `internal/recharge`
  - `internal/audit`
  - `internal/backup`
  - `internal/usage`
  - `internal/user`
  - `internal/rbac`
  - 文本聊天主链路
  - 远程导入源管理（若本轮按建议移除）
- **账号池 MVP 验收口径**
  - 图片请求能命中主池
  - 调度器只在池成员内选账号
  - 主池无可用成员时 fallback 生效

## 1. 背景

当前仓库的定位仍然是“OpenAI 兼容 SaaS 网关 + 运营后台”,而不是“面向单一模型的账号池服务”。

现状包含的大块能力有:

- 图片网关与图片任务链路
- 文本聊天链路
- 用户注册 / 登录 / JWT / RBAC
- 用户 API Key、自助面板、在线体验、接口文档
- 积分计费、充值支付、账单流水
- 全站运营后台: 用户、统计、审计、备份、系统设置
- 账号池、代理池、调度器、远程导入源

后续目标已经明确收敛为:

1. 只保留 `gpt-image-2` 生成调用
2. 只保留账号管理、代理管理、账号池、调度
3. 不再做用户侧 SaaS 运营,只保留管理系统
4. 最终形态更接近 `sub2api` 风格的“账号池服务”,而不是多租户售卖平台

因此需要对当前项目做一次明确的产品级收缩,避免后续账号池改造继续背着大量无关模块前进。

## 2. 目标

### 2.1 总目标

把当前项目收缩成一个**仅服务图片生成的管理后台 + 网关服务**:

- 对外只暴露最小图片 API
- 对内只保留管理员管理能力
- 删除 SaaS 计费、充值、用户运营、文本聊天等无关模块
- 为后续账号池运行时改造腾出更清晰的代码边界

### 2.2 本轮必须保留

本轮必须保留以下能力:

- `gpt-image-2` 图片生成主链路
- ChatGPT 上游逆向客户端
- 账号导入 / 编辑 / 刷新 / 额度探测
- 代理池与账号绑定
- 调度器
- 管理员登录
- 管理后台
- 图片代理下载 `/p/img/...`

### 2.3 本轮必须移除

根据当前目标,以下能力属于明确非目标:

1. **计费系统**
   - 积分余额
   - 预扣 / 结算 / 退款
   - 账变流水
   - 分组倍率

2. **充值支付**
   - 套餐
   - EPay 回调
   - 用户充值页
   - 管理员充值订单页

3. **运营后台**
   - 审计日志
   - 用量统计
   - 数据备份恢复
   - SMTP 测试 / 站点宣传文案 / 面向外部的站点展示配置

4. **用户系统**
   - 开放注册
   - 普通用户登录态
   - 个人中心
   - 用户管理、用户分组、用户积分
   - 用户拥有自己的 API Key、自助控制台、自助文档

5. **文本聊天**
   - `/v1/chat/completions`
   - 文字模型 UI
   - 文字 Token 限流与结算逻辑

### 2.4 本轮建议一并移除

为了和“只做图片账号池”目标保持一致,本轮建议一并移除:

- `/v1/images/edits`
- `reference_images` 图生图扩展
- 图片异步查询对外接口 `/v1/images/tasks/:id`
- 后台模型 CRUD
- 远程 `sub2api` / CPA 导入源管理

说明:

- 这些能力不是绝对不能存在,而是**不再属于 MVP 必需能力**
- 若后续确实需要图生图或异步任务,应作为独立 Phase 回补,而不是留在当前精简基线上

## 3. 目标产品形态

精简后的产品形态应当是:

### 3.1 管理面

只保留管理员使用的后台:

- 管理员登录
- 账号管理
- 代理管理
- 账号池管理
- 最小必要的系统参数配置

不再存在:

- 用户侧菜单
- 个人中心
- 用户充值 / 账单 / 接口文档 / Playground

### 3.2 网关面

对外只保留最小接口集:

- `POST /v1/images/generations`
- `GET /v1/models`（可简化为只返回 `gpt-image-2`）
- `GET /p/img/:task_id/:idx`

可选保留:

- 内部健康检查 `/healthz`、`/readyz`

不再保留:

- `POST /v1/chat/completions`
- `POST /v1/images/edits`
- `GET /v1/images/tasks/:id`

### 3.3 鉴权模型

本轮产品不再是多租户 SaaS,因此鉴权模型应收缩为:

1. **后台鉴权**
   - 保留管理员登录
   - 不再区分普通用户与管理员

2. **网关鉴权**
   - 不再依赖“用户拥有 API Key”
   - MVP 推荐改为**实例级静态 Bearer Token**或**系统级 API Key**
   - 不再需要 `users -> api_keys -> quota -> billing` 这一整套链路

本轮推荐方案:

- 后台保留最小管理员 JWT 登录
- 网关使用配置项中的静态服务令牌

原因:

- 实现成本最低
- 最符合“只保留管理系统”的当前目标
- 避免为了保留 `/v1` 鉴权而继续维持一整套用户体系

## 4. 保留模块与移除模块

### 4.1 建议保留的后端模块

- `/Users/swimmingliu/data/github-proj/gpt2api/internal/upstream/chatgpt/`
- `/Users/swimmingliu/data/github-proj/gpt2api/internal/image/`
- `/Users/swimmingliu/data/github-proj/gpt2api/internal/account/`
- `/Users/swimmingliu/data/github-proj/gpt2api/internal/proxy/`
- `/Users/swimmingliu/data/github-proj/gpt2api/internal/scheduler/`
- `/Users/swimmingliu/data/github-proj/gpt2api/internal/accountpool/`
- 最小化后的 `/Users/swimmingliu/data/github-proj/gpt2api/internal/auth/`
- 最小化后的 `/Users/swimmingliu/data/github-proj/gpt2api/internal/settings/`

### 4.2 建议移除的后端模块

- `/Users/swimmingliu/data/github-proj/gpt2api/internal/billing/`
- `/Users/swimmingliu/data/github-proj/gpt2api/internal/recharge/`
- `/Users/swimmingliu/data/github-proj/gpt2api/internal/audit/`
- `/Users/swimmingliu/data/github-proj/gpt2api/internal/backup/`
- `/Users/swimmingliu/data/github-proj/gpt2api/internal/usage/`
- `/Users/swimmingliu/data/github-proj/gpt2api/internal/user/`
- `/Users/swimmingliu/data/github-proj/gpt2api/internal/rbac/`
- `/Users/swimmingliu/data/github-proj/gpt2api/internal/gateway/chat.go`
- `/Users/swimmingliu/data/github-proj/gpt2api/internal/accountsource/`（若本轮不保留远程导入）

### 4.3 建议保留的前端页面

- 登录页
- 管理后台布局
- 账号管理页
- 代理管理页
- 后续新增或完善的账号池管理页

### 4.4 建议移除的前端页面

- `/personal/**` 全部页面
- 用户管理
- 积分管理
- 充值订单
- 用量统计
- 审计日志
- 数据备份
- 用户分组
- 模型配置

## 5. 数据层收缩要求

### 5.1 建议保留的数据表

- `oai_accounts`
- `oai_account_cookies`
- `account_proxy_bindings`
- `proxies`
- `account_pools`
- `account_pool_members`
- `model_pool_routes`（若本轮仍保留“池路由”建模）
- `image_tasks`（若继续使用图片代理与任务结果落库）
- `system_settings`（仅保留最小字段）

### 5.2 建议停止依赖或逐步删除的数据表

- `users`
- `user_groups`
- `api_keys`
- `credit_transactions`
- `recharge_packages`
- `recharge_orders`
- `admin_audit_logs`
- `backup_files`
- `usage_logs`
- `account_import_sources`

说明:

- 本轮优先目标是**停止运行时依赖**
- 数据库层是否立刻删表,可按迁移风险分两步做:
  1. 先断开代码引用
  2. 再做清库 migration

## 6. 路由收缩要求

### 6.1 后台 API 最终应保留

- `/api/auth/login`
- `/api/auth/refresh`（若后台继续使用 refresh token）
- `/api/admin/accounts/**`
- `/api/admin/proxies/**`
- `/api/admin/account-pools/**`
- `/api/admin/account-pool-routes/**`
- `/api/admin/settings/**`（仅最小设置集）

### 6.2 后台 API 应移除

- `/api/auth/register`
- `/api/me/**`
- `/api/keys/**`
- `/api/recharge/**`
- `/api/public/epay/notify`
- `/api/admin/users/**`
- `/api/admin/credits/**`
- `/api/admin/groups/**`
- `/api/admin/usage/**`
- `/api/admin/recharge/**`
- `/api/admin/audit/**`
- `/api/admin/system/backup/**`
- `/api/admin/models/**`
- `/api/admin/account-import-sources/**`

### 6.3 OpenAI 兼容 API 最终应保留

- `POST /v1/images/generations`
- `GET /v1/models`
- `GET /p/img/:task_id/:idx`

### 6.4 OpenAI 兼容 API 应移除

- `POST /v1/chat/completions`
- `POST /v1/images/edits`
- `GET /v1/images/tasks/:id`

## 7. 最小系统设置范围

精简后仅保留与下面主题直接相关的设置:

- 网关上游超时
- 调度最小间隔
- 429 冷却
- 排队等待上限
- 账号刷新开关 / 扫描间隔
- 额度探测开关 / 扫描间隔
- 代理探测开关 / 间隔 / 超时
- 后台登录 JWT 基本参数
- 网关静态服务令牌

以下设置分类应整体删除或冻结:

- billing
- recharge
- mail
- 面向外部站点展示
- 用户注册开关

## 8. 迁移策略

### 8.1 推荐迁移顺序

1. 先冻结目标边界
2. 再删除文本聊天与用户侧页面
3. 再移除计费 / 充值 / 运营后台依赖
4. 再改造网关鉴权模型
5. 最后清理数据库与残留配置

### 8.2 原则

- 先断业务路径,后删代码
- 先让系统运行在“精简模式”,再物理移除旧模块
- 所有删除必须附带路由与依赖清单,避免残留死引用

## 9. 验收标准

满足以下条件时,可判定“功能精简”完成:

1. 前端不再出现任何用户侧页面
2. 后端不再暴露聊天、充值、计费、审计、备份、用户运营相关接口
3. 图片请求链路不再依赖 `users`、`billing`、`recharge`、`usage`
4. `POST /v1/images/generations` 在仅配置管理员后台 + 静态服务令牌的情况下可正常出图
5. 管理后台仍可完成:
   - 登录
   - 导入 / 编辑账号
   - 绑定代理
   - 查看 / 编辑账号池
6. 代码主入口 `cmd/server/main.go` 不再初始化被移除模块
7. 文档与部署配置不再宣传 SaaS 能力

## 10. 非目标提醒

本需求文档解决的是“删什么、保留什么、系统最终长什么样”。

它**不等价于**“账号池已经实现到 sub2api 模式”。

账号池运行时缺口仍需单独设计与实现,应作为下一份文档的主题。
