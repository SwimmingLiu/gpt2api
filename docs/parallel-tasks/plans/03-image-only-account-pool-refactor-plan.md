# 03. Image-Only Account Pool Refactor Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 把当前仓库收缩成“仅保留管理后台 + `gpt-image-2` 网关 + 账号池调度”的系统,并补齐账号池运行时主链路。

**Architecture:** 先做产品边界收缩,把用户 SaaS、计费、充值、运营后台、文本聊天从运行时剥离;再把图片请求路径改造成“模型/默认池 -> 池成员 -> 调度器 -> 图片 runner”的单一主链路。实现时优先保证边界清晰、写集分离、可并行推进,最后再做总集成与清库。

**Tech Stack:** Go 1.26, Gin, sqlx, Redis, Vue 3 + Vite, MySQL migration

---

## 0. 执行原则

### 0.0 共享契约摘要（后续实现统一引用）

- **最终保留的对外接口**
  - `POST /v1/images/generations`
  - `GET /v1/models`
  - `GET /p/img/:task_id/:idx`
- **最终保留的后台模块**
  - 管理员登录
  - 账号
  - 代理
  - 账号池
  - 池路由
  - 最小设置
- **最终网关鉴权**
  - `/v1` 固定为实例级静态 Bearer Token
  - 不再保留“用户拥有 API Key”的运行时模型
- **必须停止初始化的模块**
  - billing / recharge / usage / audit / backup
  - user 自助菜单 / personal / playground
  - chat 路由与其依赖链

### 0.1 任务分组原则

- 先做**边界冻结**任务,再并行做代码改造
- 并行任务必须尽量使用**不重叠写集**
- `cmd/server/main.go` 与 `internal/server/router.go` 统一由**集成任务**收口,避免多个 agent 冲突

### 0.2 推荐执行顺序

1. Task 1 先完成
2. Task 2 / Task 3 / Task 4 可并行
3. Task 5 在 Task 2 与 Task 4 基础上执行
4. Task 6 最后执行

### 0.3 冻结后的共享契约入口

后续 agent 在动代码前,先统一引用以下冻结文档:

- **产品边界与路由清单**
  - `01-image-only-admin-slimming-design.md` 第 `0` 节与第 `6` 节
- **账号池运行时契约与验收口径**
  - `02-account-pool-runtime-gap-design.md` 第 `4.4` 节与第 `8` 节
- **禁止重新初始化的模块**
  - `internal/billing`
  - `internal/recharge`
  - `internal/audit`
  - `internal/backup`
  - `internal/usage`
  - `internal/user`
  - `internal/rbac`
  - 文本聊天主链路
- **固定保留的对外接口**
  - `POST /v1/images/generations`
  - `GET /v1/models`
  - `GET /p/img/:task_id/:idx`
- **固定鉴权模型**
  - 后台管理员 JWT
  - 网关静态 Bearer Token

---

### Task 1: 冻结目标边界与共享契约

**Owner:** 单独 1 个 agent,不要并行修改其他代码

**Files:**
- Modify: `/Users/swimmingliu/data/github-proj/gpt2api/README.md`
- Modify: `/Users/swimmingliu/data/github-proj/gpt2api/docs/parallel-tasks/plans/01-image-only-admin-slimming-design.md`
- Modify: `/Users/swimmingliu/data/github-proj/gpt2api/docs/parallel-tasks/plans/02-account-pool-runtime-gap-design.md`
- Create or Modify: `/Users/swimmingliu/data/github-proj/gpt2api/docs/parallel-tasks/plans/03-image-only-account-pool-refactor-plan.md`

- [ ] 明确最终保留的对外接口:
  - `POST /v1/images/generations`
  - `GET /v1/models`
  - `GET /p/img/:task_id/:idx`

- [ ] 明确最终保留的后台模块:
  - 管理员登录
  - 账号
  - 代理
  - 账号池
  - 最小设置

- [ ] 明确网关鉴权方案:
  - 推荐固定为实例级静态 Bearer Token
  - 不再保留用户拥有 API Key 的模型

- [ ] 在文档中冻结“账号池 MVP”的验收口径:
  - 请求可命中主池
  - 池成员参与调度
  - fallback 生效

- [ ] 输出一页“共享契约摘要”,供后续 agent 引用:
  - 删除哪些路由
  - 保留哪些表
  - 哪些模块必须不能再被初始化

**Verification:**
- 文档内容与当前目标一致
- 所有后续任务都能直接引用该任务产出的边界说明

---

### Task 2: 后端 SaaS 能力剥离

**Owner:** Agent A

**Write Set:**
- `/Users/swimmingliu/data/github-proj/gpt2api/internal/billing/**`
- `/Users/swimmingliu/data/github-proj/gpt2api/internal/recharge/**`
- `/Users/swimmingliu/data/github-proj/gpt2api/internal/audit/**`
- `/Users/swimmingliu/data/github-proj/gpt2api/internal/backup/**`
- `/Users/swimmingliu/data/github-proj/gpt2api/internal/usage/**`
- `/Users/swimmingliu/data/github-proj/gpt2api/internal/user/**`
- `/Users/swimmingliu/data/github-proj/gpt2api/internal/rbac/**`
- `/Users/swimmingliu/data/github-proj/gpt2api/internal/gateway/chat.go`
- `/Users/swimmingliu/data/github-proj/gpt2api/internal/gateway/playground.go`

**Do Not Touch:**
- `/Users/swimmingliu/data/github-proj/gpt2api/cmd/server/main.go`
- `/Users/swimmingliu/data/github-proj/gpt2api/internal/server/router.go`
- `/Users/swimmingliu/data/github-proj/gpt2api/internal/scheduler/**`
- `/Users/swimmingliu/data/github-proj/gpt2api/internal/accountpool/**`

- [ ] 删除聊天主链路实现,保留图片链路所需公共代码
- [ ] 删除计费、充值、统计、审计、备份相关 service / handler / dao 初始化依赖
- [ ] 删除普通用户域模型与用户分组依赖
- [ ] 把图片生成代码从 `billing`、`usage`、`group ratio` 中解耦
- [ ] 清理图片 handler 中仅服务 SaaS 的字段与流程

**Expected Output:**
- 相关包不再被图片运行时依赖
- 图片请求链路不再调用积分、充值、用户分组、usage 统计逻辑

**Verification:**
- `rtk rg -n "billing|recharge|usage|audit|backup|group" /Users/swimmingliu/data/github-proj/gpt2api/internal/gateway /Users/swimmingliu/data/github-proj/gpt2api/internal/image`
- 结果中不应再出现图片主链路对这些模块的运行时依赖

---

### Task 3: 前端精简为纯管理后台

**Owner:** Agent B

**Write Set:**
- `/Users/swimmingliu/data/github-proj/gpt2api/web/src/**`

**Do Not Touch:**
- 后端 Go 文件

- [ ] 删除 `/personal/**` 全部页面与相关 API 调用
- [ ] 删除用户管理、积分管理、充值、用量统计、审计、备份、模型配置页面
- [ ] 把路由收缩为:
  - `/login`
  - `/admin/accounts`
  - `/admin/proxies`
  - `/admin/account-pools`（如需新增）
  - `/admin/account-pool-routes`（如需新增）
  - `/admin/settings`（最小版）
- [ ] 调整后台布局与菜单,移除用户侧入口
- [ ] 如果账号池页面尚未完整,补成独立后台页面,不要再挤在账号页的临时逻辑里

**Expected Output:**
- 一个只有管理员视角的后台前端

**Verification:**
- `cd /Users/swimmingliu/data/github-proj/gpt2api/web && npm run build`
- 构建成功
- 页面菜单中不再出现 personal / billing / usage / audit / backup / user 等入口

---

### Task 4: 账号池运行时内核补齐

**Owner:** Agent C

**Write Set:**
- `/Users/swimmingliu/data/github-proj/gpt2api/internal/accountpool/**`
- `/Users/swimmingliu/data/github-proj/gpt2api/internal/scheduler/**`
- `/Users/swimmingliu/data/github-proj/gpt2api/internal/account/dao.go`
- `/Users/swimmingliu/data/github-proj/gpt2api/internal/account/importcore/**`
- `/Users/swimmingliu/data/github-proj/gpt2api/sql/migrations/**`（仅在确需补字段/索引时）

**Do Not Touch:**
- `/Users/swimmingliu/data/github-proj/gpt2api/cmd/server/main.go`
- `/Users/swimmingliu/data/github-proj/gpt2api/internal/server/router.go`
- `/Users/swimmingliu/data/github-proj/gpt2api/web/src/**`

- [ ] 定义运行时需要的 `DispatchRoute` / `ResolvedPool` 契约
- [ ] 让调度器支持“按池成员取候选账号”,而不是全局取候选
- [ ] 让成员 `enabled` 进入真实过滤逻辑
- [ ] 实现主池 -> fallback pool 的调度回退
- [ ] 明确 `priority`、`weight`、`max_parallel` 的 Phase 1 语义
- [ ] 保证导入到 `TargetPoolID` 的账号能被新调度器看到

**Expected Output:**
- 一个不依赖全局账号列表的池内调度核心

**Verification:**
- 为 `internal/accountpool/**`、`internal/scheduler/**` 增加或更新单测
- 至少覆盖:
  - 主池命中
  - 成员禁用过滤
  - fallback 命中
  - 无可用成员时失败

---

### Task 5: 网关与鉴权收口集成

**Owner:** Agent D

**Write Set:**
- `/Users/swimmingliu/data/github-proj/gpt2api/cmd/server/main.go`
- `/Users/swimmingliu/data/github-proj/gpt2api/internal/server/router.go`
- `/Users/swimmingliu/data/github-proj/gpt2api/internal/gateway/images.go`
- `/Users/swimmingliu/data/github-proj/gpt2api/internal/gateway/images_proxy.go`
- `/Users/swimmingliu/data/github-proj/gpt2api/internal/apikey/**` 或其替代实现
- `/Users/swimmingliu/data/github-proj/gpt2api/internal/middleware/**`
- `/Users/swimmingliu/data/github-proj/gpt2api/internal/model/**`
- `/Users/swimmingliu/data/github-proj/gpt2api/configs/**`

**Depends On:**
- Task 2
- Task 4

- [ ] 把 `/v1` 鉴权改成实例级静态 Bearer Token或等价的最小系统级 token 方案
- [ ] 删除 `/v1/chat/completions`
- [ ] 删除 `/v1/images/edits`
- [ ] 删除 `/v1/images/tasks/:id`
- [ ] 把 `/v1/models` 收缩为只返回 `gpt-image-2`
- [ ] 在图片网关层接入 Task 4 产出的池路由与调度契约
- [ ] 确保图片代理 `/p/img/...` 在新链路下仍可工作

**Expected Output:**
- 网关运行时只剩图片主链路
- 请求进入后能真正命中账号池调度

**Verification:**
- `go test ./...`
- 用最小配置启动后,以下命令成功:
  - `curl /v1/models`
  - `curl /v1/images/generations`
- 以下命令应返回 404 或不再注册:
  - `/v1/chat/completions`
  - `/v1/images/edits`
  - `/v1/images/tasks/:id`

---

### Task 6: 总集成、清库与交付验收

**Owner:** Integrator

**Write Set:**
- 全仓集成性修改
- 文档、部署脚本、README、迁移文件

**Depends On:**
- Task 2
- Task 3
- Task 4
- Task 5

- [ ] 清理未再使用的 import、依赖初始化、菜单、配置项
- [ ] 评估并执行“停止依赖但暂不删表”与“直接删表”的数据库策略
- [ ] 更新 README 与部署文档,去掉 SaaS 宣传与无关启动说明
- [ ] 补齐最终 smoke 验证步骤
- [ ] 输出一份“迁移后系统使用说明”

**Verification:**
- `go vet ./...`
- `go test ./...`
- 若前端仍保留: `cd /Users/swimmingliu/data/github-proj/gpt2api/web && npm run build`
- 若部署链路仍保留:
  - `docker compose -f /Users/swimmingliu/data/github-proj/gpt2api/deploy/docker-compose.yml up -d --wait`
  - `cd /Users/swimmingliu/data/github-proj/gpt2api/scripts && npm run smoke:docker`

---

## 并行执行建议

### 可并行组 A

- Agent A 执行 Task 2
- Agent B 执行 Task 3
- Agent C 执行 Task 4

原因:

- 三者写集基本可分离
- 后端业务剥离、前端精简、调度核心补齐可以同时推进

### 串行收口组 B

- Agent D 执行 Task 5
- Integrator 执行 Task 6

原因:

- `main.go`、`router.go`、图片网关入口是天然集成点
- 不适合多个 agent 同时改

---

## 交付门槛

只有同时满足以下条件,整个计划才算完成:

1. 系统不再包含用户 SaaS 运行时
2. 系统只保留图片生成主链路
3. 后台只保留管理系统
4. 账号池真正进入请求调度主路径
5. `gpt-image-2` 出图链路从入口到调度到图片代理全链路可用

---

## 给后续 agent 的一句话摘要

这不是“在旧 SaaS 项目里加一点账号池”,而是**先把项目收缩成图片账号池服务,再让账号池真正进入运行时**。
