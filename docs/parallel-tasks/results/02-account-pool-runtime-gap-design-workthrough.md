# 02-account-pool-runtime-gap-design · Workthrough

## 1. 任务目标

补齐并冻结 `02-account-pool-runtime-gap-design` 文档,把它从“缺口描述”升级为后续并行 agent 可直接引用的运行时设计约束文档。

## 2. 已完成工作

- 新建 `docs/parallel-tasks/plans/02-account-pool-runtime-gap-design.md`
- 明确 image-only 目标下的共享契约:
  - 只保留图片网关
  - 网关鉴权收敛为实例级静态 Bearer Token
  - 图片运行时不再以全局账号列表调度为主逻辑
- 冻结账号池运行时主链路:
  - `DispatchRoute`
  - `ResolvedPool`
  - 主池优先
  - fallback 条件
- 冻结成员字段在 Phase 1 的语义:
  - `enabled`
  - `priority`
  - `weight`
  - `max_parallel`
  - `sticky_ttl_sec`
- 补齐 Phase 1 / 2 / 3 的分阶段目标与 MVP 验收标准

## 3. 验证与自审

- 文档交付任务,未修改 Go / 前端运行时代码
- 已人工对照任务拆分说明与 image-only 边界要求,确认本文档:
  - 不再把“全局账号列表调度”当作 MVP 主路径
  - 明确了主池 / fallback 契约
  - 明确了后续 Task 4 / Task 5 可直接引用的运行时接口边界

## 4. 最终合并 PR

- PR: `#2`
- URL: `https://github.com/SwimmingLiu/gpt2api/pull/2`
- 目标分支: `main`
- 说明: 本 PR 即本任务的最终合并 PR

## 5. 交付文件

- `docs/parallel-tasks/plans/02-account-pool-runtime-gap-design.md`
- `docs/parallel-tasks/results/02-account-pool-runtime-gap-design-workthrough.md`
