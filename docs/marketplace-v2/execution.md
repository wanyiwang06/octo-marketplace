# Marketplace v2 执行跟踪

基线：`origin/main eec5223`（#124/#125/#128 已合并；#126 关闭；#127 开放）。PR #139 是 draft 初始后端实现，未完成 #131 验收。

## 任务收敛

GitHub 只保留 **#131** 作为 `octo-marketplace` 唯一后端 tracking issue：

- 原 #132/#133/#134 合并进 #131；
- #135/#136/#137 因实现仓库是 `octo-admin`/`octo-web`/`octo-cli`/`octo-buddy-client` 已关闭，不在本仓库跟踪；
- #138 因跨仓 REL 已关闭；marketplace 侧验证并入 #131 的 Definition of Done。

详细验收见 [issues/00-backend-delivery.md](issues/00-backend-delivery.md)；接口 wire 以 [02-api-contracts.md](02-api-contracts.md) 为准。

## 工作流

| 切片 | 内容 | 前置 | 验收边界 |
|---|---|---|---|
| A | 版本/current/published/release、edit_revision CAS、Connector effective、A21–A26 读投影 | 无 | CAS 全写路径、完整 effective digest/回填、Admin 无审核 |
| B | refs 可空、latest resolver、A01 四字段/SQL、A15 observations、删除并集/D6/D8 | A | HTTP refs 可空、公共 latest 过滤、负例 |
| C | A34 最新完整内容、A31/A32 统一 Coordinator、D9 重试/UNKNOWN/租约 | B | A34 409 矩阵、同 key 原 plan/新 key 最新、无 mk1/frontmatter |
| D | MP-04 CLI 冻结计划、离线授权、preview/apply/resume、数据安全回滚 | B | 旧 CLI 退役、无指针回滚、迁移测试 |
| V | MySQL/OpenAPI/迁移/回归与 CI | A–D | `verify` 绿；不用文档/单测代替真实接口 |

工作流可在同一后端 PR 内按可验证提交推进，也可拆后续小 PR；**不强制一 issue 一 PR**。不在 main 临时放通不完整入口；不自动合并、不执行生产迁移。未完成前 #131 保持 open。

## 远端任务映射

| 切片 | Issue | 状态 |
|---|---|---|
| MP-BACKEND（A–D + V） | https://github.com/Mininglamp-OSS/octo-marketplace/issues/131 | 进行中；PR #139 为初始实现，未完成 |
| CL-01/02/03 | 已关闭；各自仓库外部跟踪 | 不在本仓库 |
| REL-01 | 已关闭；marketplace 侧验证并入 #131 | 跨仓证据外部完成 |
