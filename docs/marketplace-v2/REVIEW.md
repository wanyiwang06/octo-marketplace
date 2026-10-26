# Marketplace v2 方案评审指南（2026-10-26，P 模型）

> **2026-10-26 权威修订：P 模型（上架冻结依赖）取代 B 模型。** 权威文档：[decisions-20261026-pin.md](decisions-20261026-pin.md)。Expert→Skill、Team→Expert 在源发布/提交时冻结 `target_version_id`，读取/安装只读 pin；Connector 仍按 ID + effective。本文件已按 P 改写；B/latest 评审结论作废，保留的历史评审记录不能作为验收依据。

> **执行基线更新（2026-10-26）：** 已重新查询 GitHub 并 fetch：当前 main 为 `eec5223`，已合入 #124/#125/#128；#126 已关闭，#127 仍开放。下文旧状态表保留核验历史，不作为本轮施工基线。后续工作从已合并代码增量改造，既有版本/release/refs/attempt/migration 代码复用；任务及 PR 追踪见 execution.md。

**状态：P 修订已落到执行文档与核心 00/01/02，待实现验收。** 本目录 issues/00–04、issues/README、plan.json、README/REVIEW、执行记录已按 P 改写；核心 00/01/02 已按 P 改写，与执行文档一致。GitHub 仍只保留 #131 一个后端 tracking issue（详细规格保留 8 份）；文档闭合不代表源码已实现或运行验收通过。

## 0. 评审目标

请重点确认三件事：

1. **P 模型是否准确落实已确认决策**：Expert→Skill、Team→Expert 在源发布/提交时冻结 `target_version_id`；读取/A01/A34/安装/审核只读 pin；pin 失效整体失败，不回退 latest/旧版。
2. **12 条决策是否被正确落到 API/技术/issue**：尤其是 pin 冻结、A15 只校验 pin/effective、删除精确 pin、private 同 owner、A09 不做、无共存、A01 四字段、CLI 迁移 + 存量回填、Connector effective、CAS、A32 同 key/新 key。
3. **有没有会导致返工的契约矛盾**：同一语义在 P 决策/00/01/02/issues 是否一致；错误码/幂等/审核窗口/隐私是否可执行。

## 1. 阅读顺序

| 顺序 | 文件 | 看什么 |
|---|---|---|
| 1 | 本文件 | 评审范围、关键问题 |
| 2 | [decisions-20261026-pin.md](decisions-20261026-pin.md) | **P 权威**：pin 冻结、读取、审核、删除、private、存量回填 |
| 3 | [00-decisions.md](00-decisions.md) | D1–D10、C01–C08、A01–A09、迁移/审核/安装口径（已按 P） |
| 4 | [02-api-contracts.md](02-api-contracts.md) | wire 唯一权威：A01/A34/A31/A32/A03/A10/A11、DTO、错误、幂等（已按 P） |
| 5 | [01-technical-design.md](01-technical-design.md) | 内部模型、事务、pin 图、锁、审核、安装、迁移（已按 P） |
| 6 | [issues/README.md](issues/README.md) + [00-backend-delivery](issues/00-backend-delivery.md) | 唯一后端 tracking、A–D 工作流、详细规格（00–04 已按 P；05–08 待同步） |
| 7 | [plan.json](plan.json) | 机器可读依赖/所有权/API 映射（已按 P） |
| 8 | [execution.md](execution.md) | 当前 PR 拆分、远端 issue 映射、执行顺序 |
| 9 | [decisions-20261026.md](decisions-20261026.md) | 用户决策记录与理由（含 P 取代 B 的修订节） |
| 10 | [review/](review/) | 决策前的代码事实审计、PR 差距、#131–#138 issue 评审（历史背景，结论以 P 决策为准） |

## 2. 已锁定的 12 条决策

| ID | 决策 |
|---|---|
| P | Expert→Skill、Team→Expert 在源发布/提交时冻结 `target_version_id`；读取/安装只读 pin；目标发新版不影响；Connector 按 ID 读 effective；Team 角色/Leader/顺序仍由 Team 边决定 |
| D1 | 保留原删除能力；保护 = live incoming（含草稿）∪ published 精确 pin refs（保护目标 plugin + pin 版本）∪ Team→pinned 成员 Expert 版本间接；无“曾发布即禁止删除” |
| D2 | A09 restore 不做；无 `restored_from_version_id` |
| D3 | 不允许同名/多版本共存；同名冲突失败；无 `mk1`/frontmatter 改写；不改 Fleet/Daemon |
| D4 | A01 `dependency_availability` 四字段；evaluated_view/version 指来源自己的 published/editing；target.version_id 用 pin；无权匿名 |
| D5 | 生产迁移用 CLI + 离线授权；无 A29/A30 HTTP；两次 preview/冻结 plan_digest；数据安全回滚；cutover 回填存量已发布 pin |
| D6 | 同 owner private→private 允许（private Expert→private Skill、private Team→private Expert）；space/system 不得引用 private；pin 依赖下架/删除/失去 release 导致隐藏 + A34/安装失败 |
| D7 | Connector effective + 审核观察；变化 409 `dependency_changed`；安装/A34 读 effective |
| D8 | `edit_revision` + `expected_edit_revision`；relations 缺失 400；Admin/Web/CLI/Bot 都改 |
| D9 | A32 保留 URL；A31/A32 同一引擎；同 key 同输入原 plan，目标发新版不阻断；显式安装输入变化 409；新 key 按当前源发布快照新安装；仅首次计数 |
| D10 | 部署门禁 + 老执行器不可达；兼容二进制 + 新版本恢复；不指针回滚 |
| 审核 | 默认自动（Space 可关）；自动内部重试；人工批准只校验 pin/effective，目标发新版不 409；批准后 pin 不变 |

## 3. 评审重点（建议逐项打勾）

### P 模型

- [ ] pin 在 Expert/Team 发布/提交时写入已发布快照；草稿可空；提交/发布冻结缺失 pin；显式 pin 必须 official release 且归属正确；已 release 同一源版本重复发布不刷新 pin。
- [ ] 读取/A01/A03/A34/安装/审核只读 pin，不解析目标 latest；pin 版本不可用时整体失败，**不回退更新版/旧版，也不自动切换**。
- [ ] Team 成员固定为 Team 快照 pin 的 Expert 版本；该版本的 Skill 也按 Expert 自己的 pin；Team 角色/Leader 仍由 Team 边决定。
- [ ] 目标发新版不改变已发布图与 A34 `bundle_digest`；只有源重新发布才刷新 pin 图；同 key 重试仍使用首次 plan，新 key 按当前源发布快照。
- [ ] 存量已发布 Expert/Team 在切换时完成 pin 回填（合法保留、缺失冻结、无可冻结则 blocked），且读取不懒回填。
- [ ] private 同 owner→private 正例（Expert→Skill、Team→Expert）与 space/system→private 负例都有覆盖。

### 删除与生命周期（D1）

- [ ] 删除保护 = live incoming（含草稿）∪ published 精确 pin refs（保护目标 plugin + pin 版本）∪ Team→pinned 成员 Expert 版本 refs，是否覆盖真实引用。
- [ ] 无“曾发布即禁止删除”后，已安装用户可能因删除受影响；是否接受（当前决策接受）。
- [ ] 只改删除、不改“已发布资源收窄 visibility”的现有行为，是否明确。

### A01 诊断（D4）

- [ ] `evaluated_view/evaluated_version_id` 指来源自己的 published/editing，而不是依赖版本；`target.version_id` 是 pin；是否清楚。
- [ ] 已发布来源始终诊断 published，仅从未发布才 editing；是否满足“草稿修好但线上仍坏”的场景。
- [ ] 无权依赖合并一条匿名 `dependency_unavailable`，不泄漏名称/ID/数量；是否足够。
- [ ] mine 仅 editing、公共仅 published、Connector effective；非法组合 400 `details.field=view`；是否与 Web 计划一致。

### Connector effective（D7）

- [ ] Tenant 需审核的 Connector 编辑批准后生效；审核 observations 变化 409 后按原权限取消并重提新 review_id，不修改旧基线。
- [ ] Admin 不用审核（用户已确认）；既有写入直接生效并同步 effective，仍验证权限、CAS 和依赖范围。
- [ ] 已安装实例不受新 effective 影响；新安装用新 effective；是否清楚。

### CAS（D8）

- [ ] `edit_revision` 从 1 开始；更新必带、新建省略；缺失/过期 409；relations 缺失 400；四端都改。
- [ ] 灰度/旧客户端 400 的迁移策略是否可接受。

### 安装（D9）

- [ ] A32 URL 保留、内部接同一 Coordinator、要求 Key/用户/UUID；旧客户端升级前 400。
- [ ] 同 key 同输入只重放首次冻结 plan；目标发新版不阻断，显式输入变化 409；新 key 按当前源发布快照新安装；仅首次成功计数。
- [ ] 状态机/UNKNOWN/lease 是否足够；A32 不再走 legacy create/bind/delete。

### 迁移（D5）

- [ ] CLI + 离线授权；两次 preview/冻结 `plan_digest`；apply 只认冻结计划。
- [ ] cutover 对存量已发布 Expert/Team 回填 pin：合法保留、缺失/无效冻结当前正式发布版、无可冻结则 blocked；读取不懒回填。
- [ ] 数据安全回滚：不恢复 current/published 指针、不删行/文件，forward-fix 新版本；是否满足“回滚必须数据安全”。
- [ ] 新版本走现有审核（默认自动/人工 waiting_review）；不直接插 release。
- [ ] 迁移后各节点独立可见；completed 条件是否足够。

### 共存/权限/兼容

- [ ] 同名/多版本共存拒绝；普通安装保留市场原名/原字节；不改 Fleet/Daemon；是否满足产品预期。
- [ ] 同 owner private→private 允许；space/system 不得引用 private；是否与现有 private 资产兼容。
- [ ] A09/A29/A30 完全移出范围；是否还有客户端/文档依赖。

### 错误/幂等/隐私

- [ ] `dependency_changed`/`idempotency_input_mismatch`/`installation_result_unknown` 等错误码是否覆盖真实场景。
- [ ] 无权目标不泄漏存在性/名称/数量；private/删除/下架匿名规则是否一致。
- [ ] A25 receipt 的 7 天回放/tombstone/同事务；A31/A32 同 key 域。

## 4. 已确认边界与实现检查

1. **Admin 不用审核**，直接生效并同步 effective；Tenant 沿用原审核策略。依赖更新**仅审计展示**，无主动通知。
2. **A03 历史根**可选 version_id，但依赖读取该根快照的 pin；A34 不允许根版本覆盖。Team 成员同名按 pinned Expert 版本的名称报 team_member_name_conflict。
3. **删除间接保护**只取 Team 快照 pin 的成员 Expert 版本及其 pin refs，不扩大到全部历史；live 草稿引用仍保护；下架 pin 目标不自动解保护。
4. **人工审核恢复**取消旧申请后重提新 review_id；版本化源可同版本，Connector 重验当前 CAS/内容；旧观察值不可变。
5. **离线授权**格式、签名、信任来源、有效期、撤销与同 run 续签恢复固定按 02 §5.1；命令帮助可细化，但不能改变上述契约。

## 5. 代码/PR 基线（评审时不要误判）

| PR | 状态 | 评审结论 |
|---|---|---|
| #124 | 已合入 | 保留版本/发布底座；需补 CAS、Connector effective、读投影、发布/提交 pin 冻结；不补 A09 |
| #125 | 已合入 | refs/rereview 骨架可复用；但 `target_version_id` 需从审计改为发布冻结 pin、删除保护改为精确 pin、保留 private 同 owner；需放宽草稿 CHECK 并清理 `rejectPublishedMutation` 总禁令 |
| #126 `e7ffc79` | 已关闭，未合入 | 不作为基线；其内容被 #128 替代 |
| #127 `016baef` | 开放，**不要按现状合并** | `dependency_issues` 旧 DTO、无 view/四字段/匿名/editing 根；pin 方向正确但需按 D4/P 重做诊断与 SQL 谓词 |
| #128 `3467a39` | **已合入 main `eec5223`** | 保留 inline 负校验、锁前文件冻结、readiness SQL、过期回执说明；但 main 仍含 `mk1`/旧 pin refs CHECK/指针回滚/直接 release 迁移等需按 MP-02/03/04 清理的行为。 |
| #139 `0c7cc1f` | draft，**B 模型实现** | latest graph / A01 latest SQL / A15 批准重解析 / A34 动态依赖 / 安装 latest / `target_version_id` 仅审计；**不能按现状评审/合并**，需按 P 重做（清单见 [issues/00 §2](issues/00-backend-delivery.md)）。 |

当前 remote 基线为 `origin/main eec5223`（#124/#125/#128 已合入；#126 关闭；#127 开放）。本地 checkout 可能落后，实施前先 fetch。以上 #126/#128 的关系是“替代”，不是同时合入；`mk1`/指针回滚只在 main 的既有代码中，必须显式移除，不能当作新能力保留。PR #139 的 latest 实现同样必须整体替换，不能保留为灰度兜底。

## 6. 验证状态

- 本目录内相对链接已检查；Markdown 表格/代码围栏一致；`plan.json` 为合法 JSON（schema 1.1）。
- **未运行**服务、MySQL、Fleet、Daemon、客户端 smoke；本方案不是实现验收。
- `03-github-issues-114-123-diff.md` 是历史比对，与 00 冲突处按 00 执行。


本轮修订验证：全部本地 Markdown 文件链接、代码围栏、plan JSON/8 issue 路径、任务 DAG、API 唯一所有权通过；原差异表 80 个编号仍各出现一次。`git diff --check` 通过。`make openapi-check` 通过（0 errors、10 个既有 warnings），生成物未改变。默认 `make openapi-diff` 对比 origin/main 59ca051 与较旧本地 HEAD 99cc261 报 9 errors/22 warnings（7 个路径缺失、2 个请求 oneOf 差异），属于基线差异，不能记为通过或本次接口删除；本次仅改文档。相对本地 HEAD 的生成物差异检查通过（No changes detected），不代替新契约实现验收。

## 7. 建议评审人

| 角色 | 重点 |
|---|---|
| Marketplace 后端 | 00/01/02、A01/A34/安装/锁/refs/迁移 |
| Admin/Web/CLI/Buddy | D4 诊断、A34 单次、CAS、同 key/新 key、旧入口迁移 |
| DBA/迁移 | CLI 冻结计划、数据安全回滚、map/checkpoint、独立可见 |
| 安全/隐私 | private 受众、匿名原因、A34 完整内容、Secret 边界、跨 Space |
| REL/运维 | 部署门禁、旧执行器不可达、四端证据、回滚演练 |

## 8. 评审意见怎么提

- 优先指到文件 + 章节，例如“02 §2.2.1 / A01 的 evaluated_view 语义”。
- 区分三类：**阻塞**（会改变授权/数据/安装结果）、**实现细节**（编码时确认）、**文案**。
- 若发现 P 决策与 00/01/02 冲突，先修 00/02（以 P 决策为准），再改 01/issues；不要在代码里自行降级。核心 00/01/02 已按 P 改写；发现冲突仍按 00/P 决策先修文档。
- 评审通过后，按 #131 的 A→B→C/D 工作流拆代码任务：先完成版本/effective/CAS，再完成 pin 冻结/refs/A01，随后 A34/安装与迁移 CLI + 存量回填并行；客户端 #135–#137 与跨仓 REL #138 已关闭，不在本仓库。
