# Marketplace v2 后端交付（合并 MP-01–MP-04）

> **2026-10-26 权威修订：P 模型（上架冻结依赖）取代 B 模型。** 新决策见 [decisions-20261026-pin.md](../decisions-20261026-pin.md)：Expert→Skill、Team→Expert 在源资源发布/提交时冻结 `target_version_id`，公共读取/A01/A34/安装/审核只读 pin，不再解析 latest、不回退最新；Connector 仍按 ID + effective。本文件与 01–04、issues/README、plan.json、README/REVIEW 已按 P 改写；旧 B/latest 段落只作历史，不再作为实现依据。**核心文档 00/01/02 已按 P 改写（见 §4）。**
>
> 决策依据：[decisions-20261026-pin](../decisions-20261026-pin.md)（P 唯一新增权威）、[00-decisions](../00-decisions.md) D1–D10、[02 API 契约](../02-api-contracts.md)（wire 唯一权威）、[01 技术方案](../01-technical-design.md)。本文把原 #131–#134 合并为 **octo-marketplace 唯一后端 tracking issue**；#132–#138 不再单独跟踪。

## 0. 状态与范围

- 基线：`origin/main eec5223`（#124/#125/#128 已合入；#126 关闭；#127 开放）。
- 初始实现：PR #139（draft，head `0c7cc1f`）当前是 **B 模型实现**：latest resolver、A01 latest SQL、A15 批准时重解析、A34 动态依赖、安装 latest，且 `target_version_id` 仅审计/可空。**这些与 P 直接冲突，PR #139 不能按现状评审/合并，必须按 §2 重做后再验收。** 只有与依赖版本解析无关的底座（CAS、Connector effective、A34 HTTP 骨架、attempt/冻结 plan、删除保护并集骨架）可复用。
- 范围：**仅 `octo-marketplace` 后端**。Admin/Web/CLI/Buddy 客户端、Fleet/Daemon 改动不在本 issue；它们由各仓库自行实现，不在此跟踪。
- 不做：A09 restore、A29/A30 HTTP 迁移、同名/多版本共存、`mk1` 别名、`SKILL.md` frontmatter 改写、`installation_name`、跨 Space 私有引用。

## 1. 目标

在已合入的 #124/#125/#128 基础上，完成 P 模型的统一后端：

1. 版本不可变、`current/published/release`、`edit_revision` CAS；
2. Connector `effective` 与完整有效行摘要/回填；
3. Expert→Skill、Team→Expert 在源资源发布/提交时冻结 `target_version_id`，已发布快照 pin 为权威；
4. A01 pin 可用性过滤与四字段诊断、A15 pin/effective 审核观察；
5. A34 按 pin 图返回完整内容，同一源版本 `bundle_digest` 稳定；
6. A31/A32 统一安装 Coordinator、同 key 原 plan/新 key 按当前源发布快照、UNKNOWN/租约；
7. MP-04 CLI 冻结计划/离线授权/恢复/数据安全回滚，并完成存量已发布资产的 pin 回填；
8. 全量 MySQL 集成、OpenAPI、迁移测试通过。

## 2. PR #139 可复用基础与必须重做项

### 2.1 可复用（不等于满足 P 验收）

- `plugins.edit_revision` + 行锁 CAS；A12/Admin/A25/A26 的 `expected_edit_revision`；409 `edit_conflict`。
- `plugin_connector_effective` 表、Admin direct 写入、审核批准写 effective、读取 overlay。
- A34 `GET /plugins/{plugin_id}/distribution` 的路由/认证/`no-store`/16MiB/零 Fleet/零写骨架。
- 删除保护并集骨架：live incoming ∪ 未删除来源当前 `published_version_id` refs；移除 `rejectPublishedMutation` 删除总禁令。
- `plugin_install_attempts` 的 `frozen_plan_json`/HMAC/`intent_digest`/state/lease/UNKNOWN；A32 新路由；A26 CAS。
- `SkillInstallAdapterVersion="original-v1"`，新安装不生成 `mk1`。

### 2.2 必须按 P 重做/替换

1. `internal/repository/plugin/latest_graph.go` → `PinnedDependencyResolver`：读源已发布快照的 `target_version_id`，不调用 latest、不按目标 `published_version_id` 解析。
2. A01 `dependency_predicate.go` / `dependency_availability.go` → pin 可用性：pin 版本仍 official release 且目标可读即可用；目标后来发新版不影响。
3. A15 review observations → pin 观察；提交冻结 pins + Connector effective，批准只校验 pin/effective，不再重新解析 latest。
4. A34/安装 → 按 pin 图读取，同一源版本内容稳定；新 key 也读当前源发布快照，不读目标 latest。
5. A12/A25 发布路径 → 草稿允许 `target_version_id` 空，提交/发布时解析当前正式发布版并冻结；显式 pin 必须校验归属/official release。
6. A25 `resolved[]` → 记录本次写入/冻结的 pin，不做 latest 解析回显。
7. 删除保护/索引 → 精确 pin（目标 plugin 或 pin 版本），Team 间接走 pinned 成员 Expert 版本。
8. 存量 pin 回填 CLI：对既有已发布 Expert/Team 冻结当前正式发布目标版本，无法冻结则 `blocked`。
9. 测试：pin 稳定性、目标发新版不影响、pin 失效隐藏、private 同 owner 引用、Team pin Expert、Expert pin Skill、A15 新版本不 409、A34 digest 稳定、存量回填 blocked。

### 2.3 其余已知缺口（沿用）

- `buildRelations` 仍强制非 Connector `target_version_id`，A12 不能保存缺 pin 草稿；发布冻结逻辑不存在。
- `effective_digest` 只算 `manifest_hash + plugin_hash`，不是完整有效行；回填只覆盖 published Connector。
- A34 失败映射缺 409 `dependency_unavailable`/`dependency_changed`。
- 安装 completed 仍调用 Fleet，不是本地回放；无 EXPIRED；Fleet key 仍是客户端 key；同名冲突 400；`PreparePinnedSkills` 仍改写 frontmatter。
- A31 legacy `/plugins/install` 未与 A32 统一。
- A01/A15/A21–A25 的 DTO/view/observations 未完成。
- MP-04 迁移 CLI/离线授权/恢复/pin 回填未做，旧 stockmigration/plugin-backfill 仍在。
- CI `verify` 的 4 个迁移测试失败。

## 3. 工作流

### A. 版本、读投影与 Connector effective（原 MP-01）

- [ ] A12/Admin/A25/A26 全部写路径使用 `edit_revision` CAS；Create 拒绝 `expected_edit_revision`；冲突 409 `edit_conflict`。
- [ ] A21：默认 `editing/current`；Connector 详情可显式 `view=effective`；Admin DTO 完全省略 `published_version_id`；返回 `edit_revision`。
- [ ] A22：精确 `version_id` + 归属/权限；Connector 拒绝；不得当 latest resolver。
- [ ] A23：`skill_md/download` 默认 current 并返回/标注实际 `version_id`；公共 A06/A07 只读 published，不互相兜底。
- [ ] A24：仅 Skill/Connector 完整 Write；Expert/Team 一律 400 `zip_only_required`；Skill/Connector 不要求 `relations`。
- [ ] A25：schema2 ZIP-only；`version_id` 可选，显式值必须是目标 official release，省略时提交/直接生效时解析并冻结到当前正式发布版；`resolved[]` 记录本次冻结/写入的 pin；同 key 回放首次 `resolved[]`。
- [ ] A26：重传必须带 `expected_edit_revision`；不得用版本号/文件名代替。
- [ ] Connector `effective_digest` 覆盖完整有效行（manifest/package/name/tags/visibility/attachments）；revision 只在完整 digest 变化时 +1；存量 Connector 回填 revision 1，未发布/无有效内容者明确不伪造。Connector 关系 `target_version_id` 必须 NULL，发布只观察 effective revision/digest。
- [ ] Admin 无资产审核：A24/A25/A26 直接生效；仅 Tenant 走 A14/A16；不产生 pending review。

### B. 引用、pin 冻结、A01 与删除治理（原 MP-02）

- [ ] 唯一 `PinnedDependencyResolver`（替代 `LatestPublishedResolver`）：输入 `{root_plugin_id, root: published | explicit version_id | editing}`；**直接读取源快照的 `target_version_id`**，不调用 latest resolver、不按目标当前 `published_version_id` 重解析。A01/A34 published，A03 显式历史根，A13/A14 提交版本，编辑 editing；显式根绝不回读 current。父节点为已发布快照时 pin 必填，editing 根允许空 pin（发布时冻结）。
- [ ] A01 公共列表/详情在 COUNT/排序/facets 前按 **pin 可用性**过滤：每个 pin 的 target version 存在、仍 official release、目标 plugin `status=1`/未删除/未 embedded/`listing_state=published`、installAccess+referenceAudience 通过；Connector 还需 effective 行。目标后来发新版不影响旧 pin 可用性；SQL 谓词直接按源快照 pin 校验，不再需要 latest 解析等价测试，但必须补 SQL pin 谓词与 resolver 的等价测试及列 pin 版本可用性的 MySQL 测试。
- [ ] A01 `dependency_availability` 完整 DTO：`status/evaluated_view/evaluated_version_id/reasons[]`，reason 含 `code/target?/via?`；mine Expert/Team 必返，公共省略；`available` 时 `reasons=[]`；`target.version_id` 是 **pin**；未发布草稿空 pin 返回 `dependency_unavailable` 且不带 version_id（可披露的 target ID/名称仍可给），发布/提交补齐后恢复；匿名只隐藏不可披露部分。
- [ ] A12 HTTP `relations`：Expert→Skill/Team→Expert 的 `target_version_id` 在草稿可空；显式值必须属于目标且为 official release，否则 400 `dependency_version_mismatch`；Connector 必须 NULL；**发布流水线（A13/A14 提交时、A25 Admin 直接写入时或发布事务内）负责把缺失 pin 解析为当时目标当前正式发布版并写入冻结/已发布快照；提交时已冻结的，批准发布只复用、不再解析**，任一 pin 不可用 409 `dependency_unavailable`；错误码 `connector_version_forbidden`；修复 `buildRelations` 与 DB/validator 一致。
- [ ] refs 索引完整性：已发布快照的 refs 必须携带精确 pin；草稿 refs 可为 NULL；源版本/refs 不一致 500 `relation_index_inconsistent`；refs 一致但目标缺失/pin 失效为正常 `dependency_unavailable`。
- [ ] 删除保护：live incoming（active、含草稿、不过滤 listing）∪ 未删除来源当前 `published_version_id` 的 published refs（精确 pin：保护目标 plugin 与 pin 的版本）∪ 一跳 Team→pinned 成员 Expert **版本**→该版本 refs 的 pin；**删除 pinned 版本或目标 plugin 必须被保护**；下架/删除 pin 目标不自动解保护，只有引用方重新发布/删除边才释放；不读历史 superset；无“曾发布禁删”；移除 `rejectPublishedMutation` 删除总禁令；可见性收窄行为不变。
- [ ] D6：private 来源可引用 published system / 同 Space space / 同 owner private；space/system 不得引用 private；跨 Space 禁止；**私有 Expert→同 owner private Skill、private Team→同 owner private Expert 是合法主路径，必须有回归**；不设“private 一律不可被引用”禁令。
- [ ] D8：`relations` 仅 A12 完整 Write 必填；A12 Skill import 不接收 relations；A25 关系来自 ZIP。
- [ ] A15：提交时冻结 `resolved_dependency_observations`（Expert/Team 的 pins）与 `connector_observations`（effective revision/digest）；人工批准只校验 pin 版本仍存在且 official release、目标仍可读/未删除/未 embedded/已上架、Connector effective 未变化；**目标后来发布新版本不构成 `dependency_changed`**；自动模式仍可重试一次；`dependency_diff` 不把 pin 变化当差异。
- [ ] view 矩阵：公共仅 published；mine 仅 editing；Connector effective；非法组合 400 `view_invalid`（`details.field=view`）；A15 审核观察按提交版本的冻结根（frozen_version）读取 pin。

### C. 安装与 A34（原 MP-03）

- [ ] A34：Expert/Team 单 GET；**pin 图** + 全部原始文件；JCS `bundle_digest` 对同一源版本稳定（目标发新版不改变）；`no-store`；16MiB；零 Fleet/零写；不返回 `installation_name`、不改 frontmatter。
- [ ] A34 失败矩阵：400 path/type/binary；404 根/目标不可见；409 pin 版本已下架/删除/失去 official release 或目标不可用（`dependency_unavailable`/`dependency_changed`）；413 预算；500 文件/refs 完整性。补“pin 的 S v1 下架 → 409、S v2 存在也不回退”测试。
- [ ] A31/A32 共用唯一 `InstallCoordinator`；flag on/off 只控制 Coordinator 是否启用，不切换执行器；删除 `pinnedInstallation`/`installPinnedLegacy`/`adapter_version`/`transform_variant`。
- [ ] A32：单 `X-Workspace-Id` UUID header（或明确大小写约定）且与 body `workspace_id` 一致；单 `Idempotency-Key` 1–200 可见 ASCII；3MiB body/16MiB Fleet；用户身份、拒绝 Bot；缺失/多值/非 UUID/不相等 400。
- [ ] D9：`intent_digest` 只含 root/Space/Workspace/runtime/resource_name；pin/解析版本不进输入身份；同 key 重放首次冻结 plan；显式输入变化 409 `idempotency_input_mismatch`；pin 节点失效 409 `dependency_changed`；**新 key 仍按当前源发布快照的 pin 图解析**，不按目标 latest。
- [ ] `attempt_state` 含 `PREPARED/DISPATCHING/SUCCEEDED/FAILED/UNKNOWN/EXPIRED`；lease owner 才能 dispatch；DISPATCHING 到期→UNKNOWN；晚到成功 CAS；Fleet key=`attempt_id`。
- [ ] 已成功本地回放，不重发 Fleet；成功回放不要求重填 Secret；非成功重试必须能重建完整请求，否则安全失败。
- [ ] 同安装名不同内容 409 `same_install_identity_different_content`；安装保留市场原名/原字节，不生成 `mk1`、不改 `SKILL.md` frontmatter；仅首次成功计数。
- [ ] attempt PK/旧行迁移：目标唯一键去掉不必要的 Space 维度；旧 `completed` 仅按已验证结果映射 SUCCEEDED，否则 UNKNOWN/人工。

### D. 迁移 CLI 与数据安全回滚（原 MP-04）

- [ ] 在 `marketplace-stock-migration`/`internal/backfill/stockmigration`/`plugin_stock_migration_batches/roots` 上演进；退役 `plugin-backfill`；删除 `--rollback` 指针回滚、直接 `INSERT plugin_version_releases`、`--g4...` 自证批准、固定 pin；撤销旧 DSN/权限、移除/deny 旧构建，旧 `--apply` 不可写。
- [ ] 两次只读 preview（discover/freeze），显式标签/映射/Connector server 分组/归属，冻结 `plan_digest`；apply 只认冻结计划+离线授权，不接受手改摘要。
- [ ] 离线授权：Ed25519/JCS、部署信任配置、actor、有效期/撤销/续签；不把公钥放附件；CLI `plan_stale`/`authorization_invalid` 非零退出/blocked，不用 HTTP 409。
- [ ] **存量 pin 回填（P 新增）**：生产切换时对现有已发布 Expert/Team 执行一次冻结：已有合法 `target_version_id` 且仍是 official release → 保留；缺失/无效 → 冻结到迁移时目标当前正式发布版；找不到可冻结版本 → 该根 `blocked`，不静默取 latest/旧版。回填在离线授权/冻结计划下执行并产出 `plan_digest`；**禁止普通读取时懒回填**；回填不自动重新审核来源，下一次用户重新发布按 P4/P1/P2 重新冻结。
- [ ] run 由 `(source_scope, source_identity, plan_digest, root_node_key)` 仲裁，无未定义幂等键；定义 frozen plan/run/batch/authorization 增量存储；`resume --run-id` 只读原计划。
- [ ] 复用/更新目标不改内容/归属；apply 时 Connector reuse 的 effective revision/digest 漂移 → `plan_stale`；显式 pin 漂移/失效 → 按 pin 冻结规则阻断；`source_digest`=`input_digest`，JCS/排序/`plan_digest` 测试向量。
- [ ] 数据安全回滚：只退兼容二进制，保留 schema/对象/attempt/map；不恢复 `current/published` 指针；内容回退从历史生成新版本并走原发布/审核；不调用 `--rollback`。
- [ ] `scripts/pinned-release-readiness.sql` 按 P 重写（校验已发布快照 pin 完整性）或降级为仅结构检查；不新增 A29/A30 HTTP；REL 验证旧 `plugin-backfill`/旧 `marketplace-stock-migration --apply` 不可达。

## 4. 交付顺序

1. **文档前置**：核心 `00-decisions.md`/`01-technical-design.md`/`02-api-contracts.md` 已按 [decisions-20261026-pin.md](../decisions-20261026-pin.md) 改写；本目录 issues/plan 已对齐。核心文档与 P 决策已一致；如发现冲突，以 00/P 决策为准先修文档；不得据此评审 PR #139。
2. A（CAS/effective/读投影）与 B（pin 冻结/refs/A01/删除）先闭合，保证发布写入口和公共读取同口径。
3. C（A34/安装）在 B 后实现；A34 先于安装错误矩阵验收；新 key 与同 key 都必须只消费 pin 图。
4. D（迁移 CLI + 存量 pin 回填）与 C 可并行，但必须在生产切换前完成。
5. 最后跑全量 MySQL/OpenAPI/迁移/回归；PR139 在原 PR 内按 P 重做，不另起第二个后端 tracking issue。

## 5. Definition of Done

- [ ] 本 issue 的 A–D 全部勾选完成；没有“文档完成/单测通过”代替真实 MySQL/接口验收。
- [ ] `go test ./...`、MySQL 集成、迁移测试、OpenAPI `check`/`diff` 全绿；CI `verify` 通过。
- [ ] 关键负例：跨 Space/private（含同 owner private→private 正例）、refs 缺失/不一致、pin 版本下架/删除 409 且不回退、目标发新版不改变已发布 pin、A15 新版本不 409、A34 同源版本 digest 稳定、同 key 重放/新 key 按源快照、UNKNOWN/EXPIRED、同名冲突、旧 CLI 不可达、指针回滚不存在、存量回填 blocked。
- [ ] PR139 的 B 实现已替换为 P：仓库内不再有 latest resolver/A01 latest SQL/A15 批准重解析/A34 动态依赖/安装 latest；不再保留 `mk1`/frontmatter 改写/指针回滚/旧迁移旁路。
- [ ] 仅改 `octo-marketplace`；不声称 Admin/Web/CLI/Buddy/Fleet/Daemon 已完成。

## 6. 验证

```bash
go test ./...
make openapi-check
make openapi-diff
# MySQL integration: TEST_MYSQL_DSN=... go test ./internal/db -run 'Migration|Install|Reference|Pin'
```

## 7. 参考

- [P 决策](../decisions-20261026-pin.md)、[01 技术方案](../01-technical-design.md)、[02 API 契约](../02-api-contracts.md)、[00 决策](../00-decisions.md)
- 详细规格：[01 版本](01-version-publication.md)、[02 引用](02-dependency-governance.md)、[03 安装](03-installation-fleet.md)、[04 迁移](04-migration.md)
- 历史评审：[PR139 覆盖](../review/pr139-coverage.md)、[#131–#138 评审](../review/issue-review-131-138.md)（均为 B 时代记录，结论按 P 重新判定）
