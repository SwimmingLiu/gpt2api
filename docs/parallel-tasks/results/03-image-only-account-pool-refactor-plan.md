# 03. Image-Only Account Pool Refactor · Workthrough

## 概述

本次子任务把仓库运行时收口到「管理员后台 + `gpt-image-2` 网关 + 账号池调度」基线,并补齐了图片账号池主链路。

对应最终合并 PR:

- PR: `#3`
- URL: [https://github.com/SwimmingLiu/gpt2api/pull/3](https://github.com/SwimmingLiu/gpt2api/pull/3)
- Base: `codex/unified-account-import`

## 已完成工作

### 1. 运行时边界收口

- `/v1` 改为实例级静态 Bearer Token 鉴权
- `/v1` 只保留:
  - `GET /v1/models`
  - `POST /v1/images/generations`
- 保留图片代理:
  - `GET /p/img/:task_id/:idx`
- 后台 API 收缩为:
  - 管理员登录
  - `/api/admin/accounts`
  - `/api/admin/proxies`
  - `/api/admin/account-pools`
  - `/api/admin/account-pool-routes`
  - `/api/admin/settings`
- `/api/auth/register` 仅保留空库 bootstrap admin 能力,默认注册开关改为关闭

### 2. 账号池运行时补齐

- 新增 `DispatchRoute` / `ResolvedPool` 契约
- 调度器改为按池成员选账号,不再走图片主链路的全局候选集
- 成员 `enabled` 真正参与过滤
- 支持主池不可用时切 fallback pool
- Phase 1 语义冻结:
  - `priority` 越小越优先
  - `weight` 同优先级内按值从大到小稳定排序
  - `max_parallel` 当前仍受一号一锁约束
- 图片 runner 已接入池路由 -> 调度器的新主链路

### 3. 前端精简

- 路由收缩为:
  - `/login`
  - `/admin/accounts`
  - `/admin/proxies`
  - `/admin/account-pools`
  - `/admin/account-pool-routes`
  - `/admin/settings`
- 删除 personal 区入口和注册入口
- 调整后台菜单为纯管理员视角
- 新增独立页面:
  - `AccountPools.vue`
  - `AccountPoolRoutes.vue`
- 设置页收缩为最小标签集,移除失效的邮件测试入口

### 4. 文档与脚本

- README 增补 image-only 共享契约摘要
- `01` / `02` / `03` 计划文档补充冻结边界与 MVP 验收口径
- `scripts/smoke.mjs` 重写为 image-only / admin-only 冒烟脚本

## 验证结果

通过:

- `rtk go vet ./...`
- `rtk go test ./...`
- `cd web && rtk npm run build`
- `rtk make build`
- `rtk bash deploy/build-local.sh`
- `rtk node --check scripts/smoke.mjs`

受环境限制未完成:

- `docker compose -f deploy/docker-compose.yml up -d --wait`
- `cd scripts && npm run smoke:docker`
  - 原因: 当前执行环境缺少 `docker`
- 完整浏览器视觉回归
  - 已做前端构建与本地预览尝试
  - 但当前桌面浏览器会话被外部活动复用,未能稳定完成整套截图取证

## 产物

- 本地二进制: `/Users/swimmingliu/.config/superpowers/worktrees/gpt2api/image-only-account-pool-refactor/bin/gpt2api`
- 部署二进制: `/Users/swimmingliu/.config/superpowers/worktrees/gpt2api/image-only-account-pool-refactor/deploy/bin/gpt2api`
- Goose: `/Users/swimmingliu/.config/superpowers/worktrees/gpt2api/image-only-account-pool-refactor/deploy/bin/goose`
- 前端构建: `/Users/swimmingliu/.config/superpowers/worktrees/gpt2api/image-only-account-pool-refactor/web/dist/`
