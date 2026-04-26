# 01. 图片账号池化精简改造 Task 1 Workthrough

**日期**: 2026-04-26

## 1. 完成结果

本次交付完成了“边界冻结与共享契约”收口,把后续并行任务需要遵循的产品边界、运行时契约和交付入口统一写进了仓库文档。

## 2. 已完成工作

### 2.1 README 收口

- 把仓库顶部定位改成“`gpt-image-2` 图片账号池网关”
- 新增“当前分支冻结边界摘要”,明确:
  - 保留的 3 个对外接口
  - 保留的 5 类后台能力
  - 静态 Bearer Token 鉴权方案
  - 账号池 MVP 验收口径
  - 禁止重新初始化的历史 SaaS 模块

### 2.2 产品边界冻结

在 `docs/parallel-tasks/plans/01-image-only-admin-slimming-design.md` 新增“共享契约摘要”,冻结了:

- 最终保留面
- 最终移除面
- 运行时硬约束
- 必须保留的数据表
- 主入口必须停止初始化的模块

### 2.3 账号池运行时契约冻结

在 `docs/parallel-tasks/plans/02-account-pool-runtime-gap-design.md` 新增 “Phase 1 共享运行时契约”,明确了:

- `DispatchRoute` 最小字段
- `ResolvedPool` 最小字段
- primary -> fallback 的选择顺序
- 网关只接图片主链路时的前提条件
- sticky 等扩展位在 Phase 1 的处理方式

### 2.4 并行实施入口对齐

在 `docs/parallel-tasks/plans/03-image-only-account-pool-refactor-plan.md` 增加“冻结后的共享契约入口”,让后续 agent 能直接按章节引用:

- 产品边界
- 路由清单
- 运行时契约
- 禁止初始化模块
- 固定鉴权模型

## 3. 变更文件

- `README.md`
- `docs/parallel-tasks/plans/01-image-only-admin-slimming-design.md`
- `docs/parallel-tasks/plans/02-account-pool-runtime-gap-design.md`
- `docs/parallel-tasks/plans/03-image-only-account-pool-refactor-plan.md`
- `docs/parallel-tasks/results/01-image-only-admin-slimming-design.md`

## 4. 验证情况

- 已执行: `go vet ./...`
- 已执行: `go test ./...`
- 未执行: `cd web && npm run build`
  - 原因: 本次提交不修改 `web/` 或任何前端静态资源
- 未执行: `docker compose -f deploy/docker-compose.yml up -d --wait`
- 未执行: `cd scripts && npm run smoke:docker`
  - 原因: 本次提交为纯文档边界冻结,不涉及运行时代码改动
- 未执行: 像素级 / 视觉回归
  - 原因: 本次提交不涉及 UI、样式、页面结构或渲染结果

## 5. PR 与合并状态

- 基线分支: `codex/unified-account-import`
- 任务分支: `codex/parallel-01-image-only-admin-slimming-design`
- PR: 待创建,创建后回填
- 合并状态: 待创建,合并后回填
