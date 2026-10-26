# Marketplace v2 后端交付（合并 MP-01–MP-04）

> 决策依据：[00-decisions](../00-decisions.md) B 模型、D1–D10；[02 API 契约](../02-api-contracts.md) 是 wire 唯一权威；[01 技术方案](../01-technical-design.md) 是内部实现约束。本文把原 #131–#134 合并为 **octo-marketplace 唯一后端 tracking issue**；#132–#138 不再单独跟踪。

## 0. 状态与范围

- 基线：`origin/main eec5223`（#124/#125/#128 已合入；#126 关闭；#127 开放）。
- 初始实现：PR #139（draft，head `0c7cc1f`）已提供 CAS、Connector effective、A34 主路径、latest graph、删除保护、冻结安装计划等基础；**它没有完成本 issue 的验收，不能关闭本 issue**。
- 范围：**仅 `octo-marketplace` 后端**。Admin/Web/CLI/Buddy 客户端、Fleet/Daemon 改动不在本 issue；它们由各仓库自行实现，不在此跟踪。
- 不做：A09 restore、A29/A30 HTTP 迁移、同名/多版本共存、`mk1` 别名、`SKILL.md` frontmatter 改写、`installation_name`、跨 Space 私有引用。

## 1. 目标

在已合入的 #124/#125/#128 基础上，完成 B 模型的统一后端：

1. 版本不可变、`current/published/release`、`edit_revision` CAS；
2. Connector `effective` 与完整有效行摘要/回填；
3. Expert→Skill、Team→Expert 的最新发布解析；
4. A01 四字段诊断、公共列表 SQL 过滤、A15 审核观察；
5. A34 单请求最新完整内容；
6. A31/A32 统一安装 Coordinator、同 key 原 plan/新 key 最新、UNKNOWN/租约；
7. MP-04 CLI 冻结计划/离线授权/恢复/数据安全回滚；
8. 全量 MySQL 集成、OpenAPI、迁移测试通过。

## 2. PR #139 已提供的基础（可复用，不等于完成）

- `plugins.edit_revision` + 行锁 CAS；A12/Admin/A25/A26 的 `expected_edit_revision`；409 `edit_conflict`。
- `plugin_connector_effective` 表、Admin direct 写入、审核批准写 effective、读取 overlay。
- `ResolveLatestGraph`（published + release、Team 深度 ≤2、Connector effective、D6 audience）。
- A34 `GET /plugins/{plugin_id}/distribution`：原始文件、JCS `bundle_digest`、`no-store`、16MiB、零 Fleet/零写。
- 删除保护：live incoming ∪ 未删除来源当前 `published_version_id` refs；移除 `rejectPublishedMutation` 删除总禁令。
- `plugin_install_attempts` 的 `frozen_plan_json`/HMAC/`intent_digest`/state/lease/UNKNOWN；A32 新路由；A26 CAS。
- `SkillInstallAdapterVersion="original-v1"`，新安装不生成 `mk1`。

**PR139 已知未完成项：**

- `buildRelations` 仍强制非 Connector `target_version_id`，A12 HTTP 不能保存浮动引用；
- `effective_digest` 只算 `manifest_hash + plugin_hash`，不是完整有效行；回填只覆盖 published Connector；
- A34 未发布/下架依赖返回 404/500，缺 409 `dependency_unavailable`/`dependency_changed`；
- 安装 completed 仍调用 Fleet，不是本地回放；无 EXPIRED；Fleet key 仍是客户端 key；同名冲突 400；`PreparePinnedSkills` 仍改写 frontmatter；
- A31 legacy `/plugins/install` 未与 A32 统一；
- A01/A15/A21–A25 的 DTO/view/observations 未完成；
- MP-04 迁移 CLI/离线授权/恢复未做，旧 stockmigration/plugin-backfill 仍在；
- CI `verify` 的 4 个迁移测试失败。

## 3. 工作流

### A. 版本、读投影与 Connector effective（原 MP-01）

- [ ] A12/Admin/A25/A26 全部写路径使用 `edit_revision` CAS；Create 拒绝 `expected_edit_revision`；冲突 409 `edit_conflict`。
- [ ] A21：默认 `editing/current`；Connector 详情可显式 `view=effective`；Admin DTO 完全省略 `published_version_id`；返回 `edit_revision`。
- [ ] A22：精确 `version_id` + 归属/权限；Connector 拒绝；不得当 latest resolver。
- [ ] A23：`skill_md/download` 默认 current 并返回/标注实际 `version_id`；公共 A06/A07 只读 published，不互相兜底。
- [ ] A24：仅 Skill/Connector 完整 Write；Expert/Team 一律 400 `zip_only_required`；Skill/Connector 不要求 `relations`。
- [ ] A25：schema2 ZIP-only；`version_id` 可选审计，必须属于目标且已 release；`resolved[]` 记录本次导入解析；同 key 回放首次 `resolved[]`。
- [ ] A26：重传必须带 `expected_edit_revision`；不得用版本号/文件名代替。
- [ ] Connector `effective_digest` 覆盖完整有效行（manifest/package/name/tags/visibility/attachments）；revision 只在完整 digest 变化时 +1；存量 Connector 回填 revision 1，未发布/无有效内容者明确不伪造。
- [ ] Admin 无资产审核：A24/A25/A26 直接生效；仅 Tenant 走 A14/A16；不产生 pending review。

### B. 引用、最新解析、A01 与删除治理（原 MP-02）

- [ ] 唯一 `LatestPublishedResolver` 输入 `{root_plugin_id, root: published | explicit version_id | editing}`；A01/A34 published，A03 显式历史根，A13/A14 提交版本，编辑 editing；显式根绝不回读 current。
- [ ] A01 公共列表/详情在 COUNT/排序/facets 前按 latest 过滤；SQL predicate 与 resolver 同一语义核心（同 owner、生成/校验、等价测试），不写第二套 latest。
- [ ] A01 `dependency_availability` 完整 DTO：`status/evaluated_view/evaluated_version_id/reasons[]`，reason 含 `code/target?/via?`；mine Expert/Team 必返，公共省略；`available` 时 `reasons=[]`；匿名只隐藏不可披露部分。
- [ ] A12 HTTP `relations`：Expert→Skill/Team→Expert 的 `target_version_id` 可空/仅审计；Connector 必须 NULL；显式值必须属于目标且已 release；错误码 `dependency_version_mismatch`/`connector_version_forbidden`；修复 `buildRelations` 与 DB/validator 一致。
- [ ] refs 索引完整性：源版本/refs 不一致 500 `relation_index_inconsistent`；refs 一致但目标缺失为正常 `dependency_unavailable`。
- [ ] 删除保护：live incoming（active、含草稿、不过滤 listing）∪ 未删除来源当前 `published_version_id` refs ∪ 一跳 Team→成员 Expert→该 Expert 当前 published refs；不读历史 pin；无“曾发布禁删”；移除 `rejectPublishedMutation` 删除总禁令；可见性收窄行为不变。
- [ ] D6：private 来源可引用 published system / 同 Space space / 同 owner private；space/system 不得引用 private；跨 Space 禁止。
- [ ] D8：`relations` 仅 A12 完整 Write 必填；A12 Skill import 不接收 relations；A25 关系来自 ZIP。
- [ ] A15：审核观察同时返回 `resolved_dependency_observations`（Skill/Expert 最新解析）与 `connector_observations`；`dependency_diff` 不把审计 `target_version_id`/最新版本变化当差异。
- [ ] view 矩阵：公共仅 published；mine 仅 editing；Connector effective；非法组合 400 `view_invalid`（`details.field=view`）；A15 审核观察按 published 根。

### C. 安装与 A34（原 MP-03）

- [ ] A34：Expert/Team 单 GET；latest graph + 全部原始文件；JCS `bundle_digest`；`no-store`；16MiB；零 Fleet/零写；不返回 `installation_name`、不改 frontmatter。
- [ ] A34 失败矩阵：400 path/type/binary；404 根/目标不可见；409 最新依赖未发布/下架/删除/状态不合法（`dependency_unavailable`/`dependency_changed`）；413 预算；500 文件/refs 完整性。补“S v2 下架 → 409”测试。
- [ ] A31/A32 共用唯一 `InstallCoordinator`；flag on/off 只控制 Coordinator 是否启用，不切换执行器；删除 `pinnedInstallation`/`installPinnedLegacy`/`adapter_version`/`transform_variant`。
- [ ] A32：单 `X-Workspace-Id` UUID header（或明确大小写约定）且与 body `workspace_id` 一致；单 `Idempotency-Key` 1–200 可见 ASCII；3MiB body/16MiB Fleet；用户身份、拒绝 Bot；缺失/多值/非 UUID/不相等 400。
- [ ] D9：`intent_digest` 只含 root/Space/Workspace/runtime/resource_name；解析版本不进输入身份；同 key 重放首次冻结 plan；显式输入变化 409 `idempotency_input_mismatch`；市场依赖变化 409 `dependency_changed`；新 key 才解析最新。
- [ ] `attempt_state` 含 `PREPARED/DISPATCHING/SUCCEEDED/FAILED/UNKNOWN/EXPIRED`；lease owner 才能 dispatch；DISPATCHING 到期→UNKNOWN；晚到成功 CAS；Fleet key=`attempt_id`。
- [ ] 已成功本地回放，不重发 Fleet；成功回放不要求重填 Secret；非成功重试必须能重建完整请求，否则安全失败。
- [ ] 同安装名不同内容 409 `same_install_identity_different_content`；安装保留市场原名/原字节，不生成 `mk1`、不改 `SKILL.md` frontmatter；仅首次成功计数。
- [ ] attempt PK/旧行迁移：目标唯一键去掉不必要的 Space 维度；旧 `completed` 仅按已验证结果映射 SUCCEEDED，否则 UNKNOWN/人工。

### D. 迁移 CLI 与数据安全回滚（原 MP-04）

- [ ] 在 `marketplace-stock-migration`/`internal/backfill/stockmigration`/`plugin_stock_migration_batches/roots` 上演进；退役 `plugin-backfill`；删除 `--rollback` 指针回滚、直接 `INSERT plugin_version_releases`、`--g4...` 自证批准、固定 pin；撤销旧 DSN/权限、移除/deny 旧构建，旧 `--apply` 不可写。
- [ ] 两次只读 preview（discover/freeze），显式标签/映射/Connector server 分组/归属，冻结 `plan_digest`；apply 只认冻结计划+离线授权，不接受手改摘要。
- [ ] 离线授权：Ed25519/JCS、部署信任配置、actor、有效期/撤销/续签；不把公钥放附件；CLI `plan_stale`/`authorization_invalid` 非零退出/blocked，不用 HTTP 409。
- [ ] run 由 `(source_scope, source_identity, plan_digest, root_node_key)` 仲裁，无未定义幂等键；定义 frozen plan/run/batch/authorization 增量存储；`resume --run-id` 只读原计划。
- [ ] 复用/更新目标不改内容/归属；apply 时 Connector reuse 的 effective revision/digest 漂移 → `plan_stale`；`source_digest`=`input_digest`，JCS/排序/`plan_digest` 测试向量。
- [ ] 数据安全回滚：只退兼容二进制，保留 schema/对象/attempt/map；不恢复 `current/published` 指针；内容回退从历史生成新版本并走原发布/审核；不调用 `--rollback`。
- [ ] `scripts/pinned-release-readiness.sql` 按 B 改写或降级为仅结构检查；不新增 A29/A30 HTTP；REL 验证旧 `plugin-backfill`/旧 `marketplace-stock-migration --apply` 不可达。

## 4. 交付顺序

1. A（CAS/effective/读投影）与 B（refs/latest/A01/删除）先闭合，保证写入口和公共读取同口径。
2. C（A34/安装）在 B 后实现；A34 先于安装错误矩阵验收。
3. D（迁移 CLI）与 C 可并行，但必须在生产切换前完成。
4. 最后跑全量 MySQL/OpenAPI/迁移/回归；PR139 作为初始实现继续迭代，不另起第二个后端 tracking issue。

## 5. Definition of Done

- [ ] 本 issue 的 A–D 全部勾选完成；没有“文档完成/单测通过”代替真实 MySQL/接口验收。
- [ ] `go test ./...`、MySQL 集成、迁移测试、OpenAPI `check`/`diff` 全绿；CI `verify` 通过。
- [ ] 关键负例：跨 Space/private、refs 缺失/不一致、最新依赖下架 409、同 key 重放/新 key 最新、UNKNOWN/EXPIRED、同名冲突、旧 CLI 不可达、指针回滚不存在。
- [ ] PR139 已知缺口全部关闭（见 §2），不再保留 `mk1`/frontmatter 改写/指针回滚/旧迁移旁路。
- [ ] 仅改 `octo-marketplace`；不声称 Admin/Web/CLI/Buddy/Fleet/Daemon 已完成。

## 6. 验证

```bash
go test ./...
make openapi-check
make openapi-diff
# MySQL integration: TEST_MYSQL_DSN=... go test ./internal/db -run 'Migration|Install|Reference|Latest'
```

## 7. 参考

- [01 技术方案](../01-technical-design.md)、[02 API 契约](../02-api-contracts.md)、[00 决策](../00-decisions.md)
- 详细规格：[01 版本](01-version-publication.md)、[02 引用](02-dependency-governance.md)、[03 安装](03-installation-fleet.md)、[04 迁移](04-migration.md)
- 历史评审：[PR139 覆盖](../review/pr139-coverage.md)、[#131–#138 评审](../review/issue-review-131-138.md)
