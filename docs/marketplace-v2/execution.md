# Marketplace v2 执行跟踪

> **2026-10-26 权威修订：P 模型（上架冻结依赖）取代 B 模型。** 权威文档：[decisions-20261026-pin.md](decisions-20261026-pin.md)。Expert→Skill、Team→Expert 在源发布/提交时冻结 `target_version_id`，读取/安装只读 pin；Connector 仍按 ID + effective。本执行计划已按 P 更新。

基线：`origin/main eec5223`（#124/#125/#128 已合并；#126 关闭；#127 开放）。PR #139 是 draft 的 **B 模型**后端实现（latest resolver/A15 重解析/A34 动态依赖/安装 latest），与 P 冲突，需按 [issues/00 §2](issues/00-backend-delivery.md) 重做后才能验收。

## 任务收敛

GitHub 只保留 **#131** 作为 `octo-marketplace` 唯一后端 tracking issue：

- 原 #132/#133/#134 合并进 #131；
- #135/#136/#137 因实现仓库是 `octo-admin`/`octo-web`/`octo-cli`/`octo-buddy-client` 已关闭，不在本仓库跟踪；
- #138 因跨仓 REL 已关闭；marketplace 侧验证并入 #131 的 Definition of Done。

详细验收见 [issues/00-backend-delivery.md](issues/00-backend-delivery.md)；接口 wire 以 [02-api-contracts.md](02-api-contracts.md) 为准。

## 工作流

| 切片 | 内容 | 前置 | 验收边界 |
|---|---|---|---|
| A | 版本/current/published/release、edit_revision CAS、Connector effective、发布冻结 pin 的事务边界、A21–A26 读投影 | 无 | CAS 全写路径、完整 effective digest/回填、发布缺 pin 解析、Admin 无审核 |
| B | refs 可空草稿/发布必填、pin 图读取、A01 pin 四字段/SQL、A15 pin observations、删除精确 pin/D6/D8 | A | 草稿缺 pin 合法、发布冻结、公共 pin 过滤、负例 |
| C | A34 pin 完整内容、A31/A32 统一 Coordinator、D9 重试/UNKNOWN/租约 | B | A34 409 矩阵、同 key 原 plan/新 key 按源发布快照、无 mk1/frontmatter |
| D | MP-04 CLI 冻结计划、离线授权、preview/apply/resume、数据安全回滚、存量 pin 回填 | B | 旧 CLI 退役、无指针回滚、回填 blocked、迁移测试 |
| V | MySQL/OpenAPI/迁移/回归与 CI | A–D | `verify` 绿；不用文档/单测代替真实接口 |

工作流可在同一后端 PR 内按可验证提交推进，也可拆后续小 PR；**不强制一 issue 一 PR**。不在 main 临时放通不完整入口；不自动合并、不执行生产迁移。未完成前 #131 保持 open。PR #139 的 B 实现必须整体替换，不保留 latest 灰度兜底。核心 00/01/02 已按 P 改写；冲突处按 00/P 决策执行。

## 远端任务映射

| 切片 | Issue | 状态 |
|---|---|---|
| MP-BACKEND（A–D + V） | https://github.com/Mininglamp-OSS/octo-marketplace/issues/131 | 进行中；PR #139 为 B 实现，需按 P 重做 |
| CL-01/02/03 | 已关闭；各自仓库外部跟踪 | 不在本仓库 |
| REL-01 | 已关闭；marketplace 侧验证并入 #131 | 跨仓证据外部完成 |
