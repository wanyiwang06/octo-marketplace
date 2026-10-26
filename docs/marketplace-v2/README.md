# Marketplace 技术方案与可执行开发 issues（P 模型）

> **2026-10-26 权威修订：P 模型（上架冻结依赖）取代 B 模型。** 权威文档：[decisions-20261026-pin.md](decisions-20261026-pin.md)。Expert→Skill、Team→Expert 在源资源发布/提交时冻结 `target_version_id`，公共读取/A01/A03/A34/安装/审核只读 pin；Connector 仍按 ID + effective。本目录 issues/00–04、issues/README、plan.json、REVIEW、执行记录已按 P 改写；核心 00/01/02 已按 P 改写，与本文一致。

> **执行基线更新（2026-10-26）：** 已重新查询 GitHub 并 fetch：当前 main 为 `eec5223`，已合入 #124/#125/#128；#126 已关闭，#127 仍开放。下文旧状态表保留核验历史，不作为本轮施工基线。后续工作从已合并代码增量改造，既有版本/release/refs/attempt/migration 代码复用；任务及 PR 追踪见 execution.md。

**状态：2026-10-26 最终决策已锁定，并按晚间修订切换为 P。** 引用模型为 **P：Expert→Skill、Team→Expert 在源资源发布/提交时冻结精确 pin**；公共读取/A01/A34/安装/审核只读 pin，不解析目标最新版，也不回退/自动切换。Connector 仍按 ID + effective。A09 restore、同名共存、A29/A30 HTTP 迁移已明确不做或改为 CLI；存量已发布资产的 pin 在切换时由迁移 CLI 回填。完整决策见 [P 决策](decisions-20261026-pin.md) 与 [决策记录](decisions-20261026.md)（第 9 节 B 模型为历史，已被取代）。

A34 单请求完整内容读取仍为确认范围，按 P 返回“pin 图 + 全部原始文件”；Admin ZIP、Fleet 现有幂等 POST、原权限审核及市场动作保持；同名/多版本共存明确拒绝，普通安装保留市场原名/原字节，不改 Fleet/Daemon。

> **评审入口：** 先读 [REVIEW.md](REVIEW.md)，里面列了 P 评审重点、已确认决策、已知实现默认和“不要按现状合并”的 PR。P 决策见 [decisions-20261026-pin.md](decisions-20261026-pin.md)，决策记录见 [decisions-20261026.md](decisions-20261026.md)，决策前代码审计见 [review/](review/)，#131–#138 的逐项评审见 [review/issue-review-131-138.md](review/issue-review-131-138.md)，关键阻塞的案例解释见 [review/blockers-explained.md](review/blockers-explained.md)。

## 阅读顺序

0. [REVIEW · 评审指南](REVIEW.md)：评审范围、重点、默认项、PR 基线。
1. [P · 上架冻结依赖决策](decisions-20261026-pin.md)：取代 B 的权威；pin 语义、审核、删除、private、存量回填。
2. [00 · 最终决策与规则裁定](00-decisions.md)：D1–D10、C01–C08、A01–A09、迁移/审核/安装口径（已按 P）。
3. [02 · 跨服务 API 契约](02-api-contracts.md)：A01/A34/A31/A32、DTO、错误、幂等与恢复（wire 唯一权威，已按 P）。
4. [01 · 完整技术方案](01-technical-design.md)：领域、数据、状态、权限、读取、安装、迁移、回滚、验收（已按 P）。
5. [开发规格与依赖图](issues/README.md)：1 个后端 tracking（#131）+ 详细规格；客户端/REL 已关闭或外部，05–08 客户端/REL 规格待按 P 同步。
6. [plan.json](plan.json)：唯一后端 issue、仓库、API/表 owner。
7. [execution.md](execution.md)：当前 PR 拆分、远端 issue 映射、执行顺序。
8. [decisions-20261026.md](decisions-20261026.md)：用户决策记录与 P 取代 B 的修订节。
9. [review/issue-review-131-138.md](review/issue-review-131-138.md)：历史 #131–#138 评审与修正建议。
10. [review/blockers-explained.md](review/blockers-explained.md)：8 个关键阻塞 + 其他重要修正的案例解释。
11. [review/pr139-coverage.md](review/pr139-coverage.md)：PR #139 对后端 issue 的覆盖与缺口。

历史比对：[GitHub #114–#123 与固定 pin 方案的差异记录](03-github-issues-114-123-diff.md) 保留为历史；其中与 P 冲突的段落不再代表当前方案。

## 方案权威

P 修订决策 [decisions-20261026-pin.md](decisions-20261026-pin.md) → [00-decisions](00-decisions.md) → [API 契约](02-api-contracts.md) → [技术方案](01-technical-design.md) → issue 执行说明。核心 00/01/02 已按 P 改写；任何内部矛盾必须先修文档，不能在代码里自行降级或创造第二套规范。

## P 模型要点

- **pin 在源发布/提交时冻结**：Expert→Skill、Team→Expert 的 `target_version_id` 在 Expert/Team 发布/提交时写入已发布快照，成为权威 pin；草稿可空，发布/提交动作解析并冻结。
- **只读 pin**：A01/A03/A34/安装/审核读取源快照 pin；不解析目标当前发布版；pin 版本失效（删除/下架/失去 release）或目标不可用时整体失败，不回退更新版/旧版，也不自动切换。
- **Team**：成员 Expert 固定为 Team 快照 pin 的版本；Team 的角色/Leader/顺序仍由 Team 边决定；成员 Expert 发新版不影响已发布 Team。
- **稳定**：同一源版本内容与 A34 `bundle_digest` 稳定；目标发新版不改变；只有源资源重新发布才刷新 pin 图。
- **审核**：提交冻结 pin 与 Connector effective 观察；人工批准只校验 pin/effective，目标发新版不 409；批准后 pin 不变。
- **Connector**：按 ID 读 effective，无业务版本 pin。
- **存量**：生产切换时迁移 CLI 回填现有已发布 Expert/Team 的 pin；无可冻结版本则 blocked。
- **private**：同 owner private→private 仍合法；space/system 不得引用 private；跨 Space 禁止。
- **代价**：Skill/Expert 发新版不会自动传到已发布引用方；owner 需编辑并重新发布源资源才能更新 pin。

## 已确定不做

- A09 restore-to-draft。
- 同名/多版本共存、`mk1` 别名、frontmatter 改写。
- A29/A30 HTTP 迁移（改为 CLI + 离线授权）。
- “曾发布即禁止删除”的总禁令。
- 新生命周期状态、新审核角色、Fleet/Daemon 改造。

## 源码核验依据

源码按项目 AGENTS 约定读取；本轮以 `origin/main 59ca051`（#124/#125 已合入）、PR #126 `e7ffc79`、PR #127 `016baef`、替代候选 #128 `3467a39` 为基线。核验的是本地/远端 checkout，不推断未验证的运行时行为。

| 事实 | 主要证据 |
|---|---|
| 两安装 URL | Marketplace `internal/api/handler/plugin/handler.go`、`install.go`、`installations.go` |
| 当前 Approve/发布事务 | `internal/repository/plugin/review.go`、`version_review.go`、`version_management.go` |
| 删除保护现状 | `internal/repository/plugin/references.go:254`、`write.go:838-1019` |
| 旧图按 plugin_id/current | `internal/service/plugin/service.go`、`internal/repository/plugin/read.go` |
| Fleet 现有能力 | 方案已核验的 `pkg/capability/validation.go`、`143_skill_owner_unique` 等；本轮未重新拉取最新 Fleet 源码 |
| 锁/refs 现状 | `internal/repository/plugin/references.go`、`review.go` |

## 逐文档审查与编码就绪边界

本目录 issues、plan.json 已完成 P 对齐；核心 00/01/02 的 P 同步尚未完成，编码前必须联读 [P 决策](decisions-20261026-pin.md) 与 [02](02-api-contracts.md)，冲突处以 P 决策为准并先修核心文档。本目录运行验证只覆盖文档链接/JSON 样例/依赖 DAG；服务、MySQL、Fleet、Daemon、客户端 smoke 是实现 issue 的验收要求，不冒充已通过。

| 规格/Issue | P 模型下的直接编码依据 | 状态 |
|---|---|---|
| **[MP-BACKEND #131](issues/00-backend-delivery.md)** | **唯一后端 tracking：A 版本/CAS/effective、B pin 冻结/refs/A01/A15、C A34/安装、D 迁移 CLI + 存量回填** | **进行中；PR139 为 B 实现，需按 P 重做** |
| [MP-01](issues/01-version-publication.md) | 版本/文件/current-published/release、CAS、读投影、A01 底座；无 A09 | 详细规格 |
| [MP-02](issues/02-dependency-governance.md) | pin 冻结/图读取、A01 四字段诊断、删除精确 pin、Connector effective、A25 receipts、A03/A10/A11 | 详细规格 |
| [MP-03](issues/03-installation-fleet.md) | A34 pin 内容、A31/A32 同一 Coordinator、同 key 原 plan、新 key 按源发布快照、状态机、无共存 | 详细规格 |
| [MP-04](issues/04-migration.md) | CLI 两次 preview/冻结计划、可恢复 apply、数据安全回滚、独立可见、存量 pin 回填；无 A29/A30 HTTP | 详细规格 |
| [CL-01](issues/05-admin.md) | schema2 ZIP、A24 拒绝 Expert/Team、A25 Key/CAS/resolved、固定预览 | #135 已关闭（admin） |
| [CL-02](issues/06-web-cli.md) | 显式引用/审核/版本读取/A34 一次完整内容/mine 四字段诊断/CAS/新 key 重装 | #136 已关闭（web/cli） |
| [CL-03](issues/07-buddy.md) | 单 A34、无依赖补取、本地边界、IPC 错误/pending | #137 已关闭（buddy） |
| [REL-01](issues/08-cutover.md) | 真实四端/数据层/Fleet/Daemon、部署门禁、旧执行器不可达、数据安全回滚 | #138 已关闭（跨仓 REL） |

编码时发现真实协议差异，先改 00/02，再改代码；禁止客户端猜字段或运行时降级。OpenAPI 生成及服务/页面/数据库 smoke 仍在各 issue 实现阶段执行，本轮只验证文档与契约样例。

## 五组评审修订状态

P 决策、issues/00–04、issues/README、plan.json、README/REVIEW、执行记录已同步 P 修订，状态为“执行文档修订完成，待开发与实现验收”；核心 00/01/02 仍需按 P 同步。Admin 不用审核；通知仅审计展示；人工审核观察冲突走取消后新一轮提交；安装原 key 不因上游正常更新而改变；删除不扩大到历史关系超集。离线迁移授权以 02 §5.1 为准。

PR 基线为 `origin/main eec5223`（#124/#125/#128 已合入；#126 关闭；#127 开放）。PR #139 是 draft 的 B 模型后端实现，与 P 冲突，需按 P 重做；实施前重新核对 head，不把 PR139 当前状态当作验收通过。
