> **历史记录（B 模型）：** 本文写于 P 模型（上架冻结依赖）修订之前，保留作评审证据；B/latest 结论已被 [decisions-20261026-pin.md](../decisions-20261026-pin.md) 取代。

# Issue #131–#138 评审（B 模型）

评审时间：2026-10-26 16:25 +08。对象：octo-marketplace #131–#138（MP-01…MP-04、CL-01…CL-03、REL-01）。基线：`origin/main eec5223`（#124/#125/#128 已合入；#126 关闭未合入；#127 开放未合入；本地 `main` 仍为 `99cc261`）。

> **总判断：方向基本正确，但不能按现状直接进入实现。** #131–#138 是本地 `issues/01…08` 的近乎逐字复制，新增的 header/链接/基线声明不构成新的决策漂移；问题主要来自**方案自身**的契约缺口和 #128 合入后的代码基线未清理。应先修 00/02/01/issues，再改 GitHub 正文，最后写代码。

## 0. 逐 issue 结论

| Issue | 结论 | 关键原因 |
|---|---|---|
| **#131 MP-01** | 方向正确，**不可直接实现** | A01 DTO 在 MP-01/MP-02 之间“定义但不填充却必返”；refs 可空与现有 CHECK 冲突；仍写“唯一创建 release/refs”但基线已有；缺 Connector effective 回填；A21/A22/A23/A26 wire 细节缺失 |
| **#132 MP-02** | **不可直接实现** | D1 验收写“Team 旧 Skill 仍被保护”，在 B 下不成立；间接保护链写乱；解析器输入 `plugin_id+view` 无法表达 A03/A14 的显式根版本；SQL 谓词与“唯一解析器”冲突；refs CHECK；A15 观察字段缺失；未明确移除 `rejectPublishedMutation` |
| **#133 MP-03** | **不可直接实现** | 输入身份含“规范版本”，会让上游正常发新版破坏同 key 重试；未明确移除 #128 的 pinned/mk1 执行器；attempt 表 PK/旧行迁移、EXPIRED/租约转移、A31/A32 细节、`fleet_key=attempt_id`、同名冲突错误码缺失 |
| **#134 MP-04** | 方向正确，**不可直接实现** | 漏掉已合入的 `marketplace-stock-migration` 及其 `--apply/--rollback` 禁止语义；旧 CLI 仍是未授权旁路；CLI 用 HTTP 409；幂等键/resume 存储/Connector reuse drift/`source_digest` 未定义 |
| **#135 CL-01** | 方向正确，**有 2 个阻塞契约错误** | 把 A22 当“最新解析”读取器（A22 要精确 version_id）；接口范围排除 A03/A10；Admin “沿用审核/无新增审核”与 D7“Admin 无审核”冲突；octo-admin 基线缺失；A25 replay 与“下次预览显示新版本”冲突 |
| **#136 CL-02** | 作为客户端 issue 基本正确，**scope/baseline 错** | 缺“本轮只改 marketplace”的 banner；A01～A10 含已删除 A09；A32 缺 `X-Workspace-Id` header/body 一致；A34 session cache 与 no-store 冲突；A15 缺 `resolved_dependency_observations`；Connector candidate 缺 effective；错误码把“配置/依赖变化”写成 `idempotency_input_mismatch` |
| **#137 CL-03** | **基本正确**，少量修正 | 服务端窗口措辞、同名冲突 reason、MP-04 依赖措辞、“原 plan”客户无法持有；A34/本地边界/D9 主体正确 |
| **#138 REL-01** | 方向正确，**不可直接实现** | A34 失败矩阵漏 409/400；部署门禁挡不住本地 CLI/DB 直写；未要求移除 `--rollback` 指针回滚；pinned readiness SQL 仍按旧图；D1 published refs 范围未限定 |

## 1. 跨 issue 阻塞项

### C1 · 方案基线仍写过时状态（已部分修正）

`README.md`、`REVIEW.md`、`issues/README.md`、`plan.json` 之前仍写 `59ca051`、#126/#127/#128 open；实际 `origin/main=eec5223`，`#128` 已合入、`#126` 关闭、`#127` 开放。评审者会按旧基线误判 #128。已同步 branch 的 baseline banner、`execution.md`、`plan.json` 的 `baseline`/`issue_url`；仍需更新 `REVIEW.md` 的旧表格（已改）和任何残留的“#128 未合并”表述。

### C2 · #128 已合入代码的清理没有 owner

`eec5223` 的 main 仍含：

- `pinnedInstallation`/`installPinnedLegacy`（`install.go`）、`SkillInstallAdapterVersion="mk1"`/`PreparePinnedSkills`（`install_primitives.go`）；
- `plugin_install_attempts` 旧 PK `(space_id, caller_uid, workspace_id, key_digest)`、`adapter_version/completed`，无 state/lease/plan/HMAC；
- `plugin_relations.target_version_id` 的 CHECK 仍要求 expert_skill/expert_team_expert 非空；
- `marketplace-stock-migration` 的 `--rollback` 指针回滚、直接 `INSERT plugin_version_releases`、`--g4...` 自证批准；
- `scripts/pinned-release-readiness.sql` 按 `target_version_id` 遍历。

Issue 文本都说“无 mk1/无指针回滚/浮动引用”，但没有把**已有代码的移除**写成明确 owner/文件/验收；MP-03/04/REL-01 必须显式承接，否则新 Coordinator 加了、旧执行器仍在，flag on/off 都能“通过”。

### C3 · 一 PR/跨仓范围冲突

#131–#133 header 写“全部后端在 1 个代码 PR、Admin/Web/CLI/Buddy 不在本次范围”，但：

- #131 自身验收和 `issues/README` 仍要求 MP-01→MP-02→MP-03 独立合并；
- #135/#136/#137 的实现仓库是 octo-admin/web/cli/buddy；
- `plan.json`/`issues/README` 明确按仓库拆分、不强制一 issue 一 PR。

需要统一：要么保留“1 个后端 PR”并把“独立合并/四端同批”改成外部依赖；要么删掉“1 个 PR”并恢复按仓库 PR。否则评审/执行无法判断边界。

### C4 · A01 四字段 DTO 的 MP-01/MP-02 边界

#131 说 MP-01 “只定义 DTO”但又写“A01 mine Expert/Team 必返”，而 `status/reasons/target/via` 由 MP-02 计算；MP-01 无法真实填充。必须二选一：

- MP-01 只交付 Go 类型/序列化槽位，不写“必返”；MP-02 的验收写“mine 必返且 available 时 reasons=[]”；或
- 给 MP-01 一个明确的 interim 值（会修改 00/02，不推荐）。

同时 #131 未定义 `DependencyTarget`/`DependencyReason` 的 `target/via` 形状；只在 02 §2.2.1 有，MP-01 的 A01 DTO 必须完整引用。

### C5 · refs 可空与现有 CHECK/validator

B 要求 Expert→Skill、Team→Expert 的 `target_version_id` 可空、仅审计；但基线：

- `chk_plugin_version_ref_type` 要求这两类非空；
- `ValidateReferenceMetadata` 对 nil 返回 `ErrInvalidRelation`；
- `references.go`/`plugin_snapshot.go` 不在 #131/#132 的 owner 列表。

必须指定增量 ALTER 放宽 CHECK、修改 validator，并把这些文件纳入 MP-01/MP-02 的 owner；否则保存浮动引用会直接 400/DB 失败。

### C6 · Connector effective 回填

#131 没有回填现有 Connector 到 `plugin_connector_effective`；一旦切换，所有旧 Connector 没有 effective 行，A34/安装/引用会失败。必须加 migration/backfill（revision 1 + digest），并定义 `effective_digest` 覆盖完整有效行（manifest/package/declared visibility）。

### C7 · #128 的旧 migration CLI 是未授权旁路

#134 把源码路径写成 `cmd/plugin-backfill`，但 main 已有 `cmd/marketplace-stock-migration` + `internal/backfill/stockmigration` + `plugin_stock_migration_batches/roots`。旧 CLI 已有 `--all-spaces --apply`、`--rollback` 指针回滚、直接插 release、`--g4...` 自证批准。若 #134 不列它、不退役它，验收可以在“旧 CLI 仍可写”的情况下通过。必须：

- 把旧 CLI/表纳入 owner；
- 决定 evolve 或 replace；
- 删除/禁用 `--rollback` 指针回滚、直接 release、自证批准、固定 pin；
- REL-01 验证两个旧入口（plugin-backfill + marketplace-stock-migration）都不可达。

### C8 · 错误码语义

- #136 “配置/依赖变化 409 `idempotency_input_mismatch`” 错：显式输入变化才是 `idempotency_input_mismatch`；市场 Connector/节点变化应是 `dependency_changed`（或 `dependency_unavailable`）。否则客户端会把“市场更新”当“用户输入错”。
- #138 A34 失败矩阵漏 409 `dependency_unavailable`/`dependency_changed` 和 400 path/type/binary；只写 404 会让 B 的核心 fail-closed（S 最新版下架）无法验收。
- #133 输入身份含“规范版本”会让上游正常发新版产生输入 HMAC 差异，破坏同 key 原 plan；应只把 root ID/Space/Workspace/runtime/resource_name 放进 `intent_digest`，完整请求/HMAC 只在非成功重试或显式提供 Secret 时校验。

### C9 · Admin 审核措辞

D7 已锁定：Admin 无资产审核、直接生效并同步 effective；Tenant 才走审核。#135 反复写“沿用发布/审核/无新增审核步骤”，会被实现成保留旧审核或让 Admin 走 A14/A16；#138 也写“Admin 发布审核不变”。必须统一为：“Admin create/update/reupload 直接生效，无 pending review；Tenant Connector/Expert/Team 才走审核”。

## 2. 逐 issue 需要改的地方

### #131 MP-01

**正确部分：** current/published/release、CAS/relations、Connector effective、view 规则、A01 DTO 底座、无 A09/mk1；代码事实（published+space 旧闸门、`redactVersionRelations`、`retainedAttachmentKeys`、OpenAPI 扫描缺 `skill.go`）基本准确。

**必须修：**

1. A01 DTO 边界（C4）。
2. refs 可空/CHECK/validator（C5）。
3. 删除“唯一创建 `plugin_version_releases`/refs”的表述：这些表已存在，MP-01 只提交增量字段（`edit_revision`、`declared_visibility`、`version_label`、`revision_no`、`content_digest`、effective、review freeze）；不要再 `CREATE TABLE`。
4. Connector effective 回填（C6）。
5. A21/A22/A23/A26 wire：Admin DTO 必须省略 `published_version_id`；A21 默认 editing/current，Connector 可显式 effective；A22 精确 version_id + 归属；A23 默认 current 并标注实际版本；A26 重传带 `expected_edit_revision`；补 `zip_only_required`/`connector_version_forbidden`/`dependency_version_mismatch`/`view_invalid`/`relation_index_inconsistent`。
6. “应有版本/refs 缺失”要区分：源版本/refs 不一致 = 500 `relation_index_inconsistent`；refs 一致但目标 Plugin 缺失 = 正常 `dependency_unavailable`。
7. A02 Connector 默认 effective；A06/A07 默认 published 不取 current；A08 仅 UTF-8；A21/A22/A23 验收。
8. `published+space` 旧保存闸门改成 editing 的行为变更要写明 Connector 范围，并与 00/01 的“只改删除”协调。

### #132 MP-02

**正确部分：** B 解析、D4 四字段、D6 同 owner private/space-system 禁止、D7 observations/409、D8 CAS 意图、Team 最新/角色不变、无回退/不重审引用方。

**必须修：**

1. **D1 旧 Skill 保护（高）**：`Team t1 引用 Expert e; e 发 e2 去掉旧 Skill` 后，t1 的发布边指向 e ID，间接解析 e2；**旧 Skill 不再被 t1 保护**，只有在其他 live 引用或另一来源 current published refs 时才保护。当前验收写“t1→e 的已发布/间接依赖仍保护相关目标”，会被读成保护旧 Skill，等同于“历史发布保护超集”。改为：t1 保护 e 和 e2 的当前依赖；旧 Skill 失去 t1 保护，若并集为空则可删；t1 的审计 pin e1 永远不是依据。
2. **间接链写乱（高）**：`published Team→成员 Expert→该 Expert 发布依赖的Team→最新发布 Expert→Skill` 多了 Team/Expert 跳；正确是“published Team 快照 → 成员 Expert ID → 该 Expert 当前 published 快照 refs”，仅一跳成员 Expert，不再展开 Team。
3. **解析器根选择（高）**：`LatestPublishedResolver` 输入 `plugin_id+view` 无法表达 A03 历史根、A13/A14 提交的精确 version_id；应为 `{root_plugin_id, root: published | explicit version_id | editing}`，A01/A34 用 published，A03 用显式 version，A13/A14 用提交 version，不得对显式 version 重读 current。
4. **SQL vs 唯一解析器（高）**：A01 必须在 COUNT/分页前过滤，需要 SQL；但“所有 A01/A03/A34/install/review 只调用一个 resolver”会与手写 SQL 冲突。应把 SQL 谓词定义为解析器的同一语义核心（同 owner、生成/校验、等价测试），或明确它是唯一允许的共享实现，禁止其他手写 latest。
5. refs CHECK/owner（C5）。
6. D6 矩阵补：private 来源可引用已发布 system、同 Space space、同 owner private；space/system 不得引用 private；跨 Space 禁止；补 private→public 正例。
7. D8 relations 必填只适用 A12 完整 Write；A12 Skill import 不接受 relations；A25 关系来自 ZIP；不要写成所有入口都传 relations。
8. “四端同批改”与“仅 marketplace”banner 冲突；后端只交付 CAS，客户端是外部依赖。
9. A15 加 `resolved_dependency_observations`（Skill/Expert 最新解析版本）+ `connector_observations`。
10. 明确移除 `rejectPublishedMutation` 总禁令（Delete/DeleteGraph/visibility 调用），只保留 live 保护再补并集；否则曾发布资源仍不可删。
11. 补 A01 view 矩阵/非法 400 `details.field=view`；显式 `target_version_id` 必须属于目标且已 release；匿名聚合只合并不可披露部分，其他可见原因仍可列出。

### #133 MP-03

**正确部分：** A34 最新解析/JCS/no-store/16MiB/全失败/零 Fleet/零写入/无 installation_name/frontmatter；D3 无共存；D9 同 key/新 key 方向；A31/A32 同引擎意图。

**必须修：**

1. **输入身份含“规范版本”（高）**：正常发新版不能让同 key 重试 409。`intent_digest` 只含 root plugin ID、Space、Workspace、runtime、resource_name；`custom_env` 只在完整 input_hmac；解析出的版本不进 intent_digest/input_hmac。补“S v2 发布后同 key 仍重放 v1 plan”的验收。
2. **未明确移除 #128 pinned/mk1（高）**：ownership 要列 `pinned_install.go`、`install_primitives.go`、`install_attempt.go`；验收要在 flag on/off 下 A31/A32 都进 `InstallCoordinator`，无 mk1/`adapter_version`，旧路径不可达。
3. attempt 表迁移：PK 去掉 `space_id`、加 state/lease/plan/HMAC/retry_until；旧 completed 行只按已验证结果映射 succeeded，否则 unknown/人工，不得自动重试。
4. 状态机补 `EXPIRED`（或声明为 retry_until 派生）并统一 01/02 的 5/6 状态；补 DISPATCHING 租约到期→UNKNOWN、lease owner 才能 dispatch、晚到成功 CAS。
5. `custom_env` 规则：intent_digest 不含 Secret；成功回放可不重填；非成功重试必须能重建完整请求，否则安全失败，不要一律 409 input mismatch。
6. “不重读当前市场”改为“不重新解析/不重选版本；出站前仍按原 plan 重验权限/节点资格，失效 409 dependency_changed”。
7. 明确 `fleet_key=attempt_id`；A31/A32 补 `X-Workspace-Id` 单头 UUID、A32 header=body、key 1–200 可见 ASCII、3MiB/16MiB、拒绝 Bot。
8. 同名冲突映射 `409 same_install_identity_different_content`，用 Fleet 现有冲突，不新增预检查询。
9. 引用 #128 而不是 #126；把 review 里“改造 mk1”改成“移除”。
10. 删除 `variant`/`transform_variant` 残留；明确“疑似凭据”走既有 400 reason 或客户端控制，不新增扫描器。

### #134 MP-04

**正确部分：** D5 CLI-only/no A29/A30、两次只读 preview、冻结 plan_digest、apply 只认冻结计划、B 审计 ID、data-safe rollback 方向、independent visibility、waiting_review/blocked/completed；§5.1 离线授权（Ed25519/JCS、部署信任、actor、有效期/撤销）与 02 一致。

**必须修：**

1. **旧 stockmigration CLI 未列/未退役（高）**：把 `cmd/marketplace-stock-migration`、`internal/backfill/stockmigration`、`plugin_stock_migration_batches/roots` 纳入 owner；决定复用/替换；删除 `--rollback` 指针回滚、直接 release、`--g4...` 自证批准、固定 pin；更新 runbook。
2. **旧 `--apply` 仍是旁路（高）**：验收只 fence `plugin-backfill`；必须 fence `marketplace-stock-migration --all-spaces --apply`（撤销 DB 权限/移除二进制），并由 REL-01 验证。
3. CLI 错误用 `plan_stale`/`authorization_invalid` + 非零退出/`blocked`，不要 409。
4. 删除/定义“幂等键”：02 §5.1 apply/resume 没有 key 参数，run 由 `(scope,identity,plan_digest,root)` 仲裁；并发测试改成同 plan/batch。
5. 补 apply-time Connector reuse 漂移 `plan_stale`；不要把 review-window `dependency_changed` 当 apply 错误。
6. `source_digest` 映射到 02 的 `input_digest`；补 JCS/排序/`plan_digest` 测试向量。
7. resume/frozen plan/run/batch/auth-binding 存储 DDL 与 owner；否则 `resume --run-id` 不可执行。
8. 新 Connector（旧 inline MCP）要写清 system/Tenant、审核路径、effective 观察值、waiting_review 顺序。
9. 验收不要要求 MP-03 的安装/A34；MP-04 只验 A01/CLI graph，A34/install 归 REL。
10. 命令名可细化，但 02 §5.1 apply/resume 参数语义不能变。
11. reuse 验收补“目标未改/未省略版本不被当权威”；ZIP identity 按 DB-only 延期条件 gate。

### #135 CL-01

**正确部分：** ZIP-only/A25 Key/CAS/resolved、A24 拒绝 Expert/Team、无选择器/pin UI、无 mk1/frontmatter、rating/placement 保留、scope 矩阵意图。

**必须修：**

1. **A22 当最新解析器（阻塞）**：A22 是 `GET /admin/plugins/{id}/versions/{version_id}`，要精确 version_id + 归属，不是 latest resolver。预览目标的最新解析身份应来自 A03/A10（或 A25 `resolved[]` 的 import 时观察），正文再用 A22 精确版本 / A21 Connector effective；不能用 Admin current/editing 当依赖解析。
2. **接口范围缺 A03/A10（阻塞）**：CL-01 要显示“本次解析版本/下次预览跟随新版”，必须消费 A03/A10；补 plan.json `consumes_api_ids`。
3. Admin 审核措辞：改成“Admin 无任何提交/待审/批准；直接生效；仅 Tenant 审核”；A24/A25 不创建 pending review。
4. octo-admin 基线：`eec5223` 是 marketplace 基线，不是 admin SHA；补 admin commit/date，或标外部依赖。
5. DTO 补 `view/selected_version_id/version_label/current_version_id/has_unpublished_changes/is_published/effective_revision/digest/declared_visibility/revision_no`；`declared_version`→`version_label`；Admin DTO 必须省略 `published_version_id`。
6. A25 same-key replay 回放首次 `resolved[]`；“下次预览显示新版本”必须走 A03/A10 或新 key/新 import，不能靠 A25 同 key。
7. A23 `/admin/.../skill_md|download` 默认 current 并标实际 version；A06/A07 是 public published；不要混用。
8. schema2 `version_id`（不是 target_version_id）可选审计；提供时必须属于目标且已 release，与最新解析不同不报错；Connector 带版本 400。
9. 补 manifest 校验：role≤500、member_key 1–64 正则、sort_order≥0、resolved-name 唯一 `team_member_name_conflict`、额外文件拒绝。
10. 命名 `zip_only_required`；确保 A26 重传 `expected_edit_revision` 在 #131 实现。

### #136 CL-02

**正确部分：** A34 单 GET/无 BFS/ZIP/frontmatter、CAS、A13/A14 version_id、A32 新 key、无 mk1/A29/A30；B 跟随/团队角色不变。

**必须修：**

1. scope/baseline：补“仅 marketplace 后端”banner，标 octo-web/octo-cli 外部依赖与各自基线；不要把 eec5223 当客户端基线。
2. A01 接口范围 `A01～A10` 含已删除 A09；改为 A01–A08/A10/A12–A19/A32/A34。
3. A32 补 `X-Workspace-Id` 单头 UUID、header=body `workspace_id`、key 1–200 可见 ASCII；负例。
4. A34 不要 session cache：`no-store`，只在当前 view/request 持有，刷新/切换重新 GET；不能跨请求复用旧 `bundle_digest`。
5. A15 补 `resolved_dependency_observations`；`dependency_diff` 只含 relation ID/type/sort_order/data/role/order，审计 target version 只展示，不参与 diff/触发重审。
6. Candidate：Skill/Expert 用 published_version_id/version_label；Connector 用 effective_revision；不要只筛 published。
7. Tooltip 文案改为“最新发布版已下架”，不要暗示整个 Skill 下架。
8. **错误码**：“配置/依赖变化”不应写 `idempotency_input_mismatch`；显式输入变化才用它，市场依赖/Connector 变化用 `dependency_changed`/`dependency_unavailable`。

### #137 CL-03

**基本正确。** 单 A34、无 BFS/依赖补取、无 frontmatter/BOM/ZIP 改写、`selected_version_id`+`bundle_digest`、pending/UNKNOWN、D9 同 key/新 key、本地导入边界均对。

**小修：**

1. “超窗由 Marketplace 客户端阻止”改成“服务端出站客户端强制 `retry_until`，Buddy 也在 min(startedAt+window, retry_until) 停止”。
2. 补 `same_install_identity_different_content` 分类；不要当可重试/unknown。
3. “MP-04 提供历史资源可分发状态”不是 CL-03 依赖（plan 只依赖 MP-03）；改成上下文。
4. “原 plan”改为“原请求/原完整规范化参数”（客户端不持有服务端 plan）。
5. 标注 Buddy 源码断言未在 workspace 复核。

### #138 REL-01

**正确部分：** B/D1–D10、五个直接依赖、四端 + Fleet/Daemon 证据、controlled review、UNKNOWN 保留、data-safe rollback 方向、无 A09/A29/A30/共存。

**必须修：**

1. **A34 失败矩阵（高）**：补 409 `dependency_unavailable`/`dependency_changed`（最新版下架/删除/未发布）、400 path/type/binary；404 只用于根/目标不可见。必须把“S v2 下架 → A34 409”作为 B fail-closed 验收。
2. **部署门禁挡不住本地 CLI/DB（高）**：旧 `plugin-backfill`/`marketplace-stock-migration` 是本地二进制直连 MySQL；仅网关/网络隔离不够。补 DB 账号/DSN/权限撤销、旧二进制 deny/缺失验证、A14–A19/placement 全入口清单。
3. **指针回滚仍在（高）**：main 的 `marketplace-stock-migration --rollback` 仍是“restore root pointers”；REL-01 必须要求删除/禁用，data-safe rollback 演练不得调用它。
4. pinned readiness SQL 按 `target_version_id` 遍历，B 下是审计值；改写为最新解析或仅结构检查，指定 owner。
5. D1 published refs 要限定“未删除来源的 current `published_version_id`”，不是全部历史；补“历史-only 可删/current published 阻断/live 草稿阻断”测试。
6. REL-01 不要自己删旧 adapter/路由；这是 MP-03 的交付，REL 只做退役核对。
7. “Admin 发布审核不变”改为“Admin 无资产审核；Tenant 审核/Space 策略不变”。
8. 删除无出处的“成员/直接依赖数量上限”；Admin 预览目标 ID/解析版本要明确是本地 manifest + A25 post-import resolved，不是 pre-import 服务端预览。

## 3. 建议修复顺序

1. **先修 00/02 契约**（所有 issue 的源头）：
   - A01 DTO/MP-01 vs MP-02 边界；
   - refs 可空 + CHECK/validator owner；
   - Connector effective 回填/摘要；
   - B 解析器根选择与 SQL 谓词关系；
   - D9 输入身份（去掉“规范版本”）/错误码；
   - A34 失败矩阵 400/404/409；
   - CLI `plan_stale`/`authorization_invalid`/resume 存储；
   - D1 间接保护范围（current published only）；
   - Admin 无审核措辞。
2. **再修 01 和 8 个 issue**：
   - 移除 #128 pinned/mk1/pointer rollback/direct release/pinned readiness 的 owner/验收；
   - 补 A21/A22/A23/A26、A15 observations、refs CHECK、Connector backfill；
   - 修正 #132 L99/L72、#133 输入身份、#135 A22、#136 A32/cache/error code。
3. **同步 GitHub #131–#138 正文**（它们目前是本地 issue 的逐字复制，改本地后再更新远端，避免双份漂移）。
4. **最后写代码**：按 MP-01→MP-02→MP-03/MP-04/CL-01→CL-02/CL-03→REL-01；#128 的清理在 MP-02/03/04 内完成。

## 4. 评审边界

- 只读静态评审；未运行服务/MySQL/Fleet/Daemon/客户端。
- 基线代码用 `git show eec5223:` 核对；本地工作区 `99cc261` 不代表当前 main。
- #135–#137 的客户端源码断言未在 workspace 复核（没有 octo-admin/web/cli/buddy checkout），只按契约评审。
- 本文件只给 issue 修正建议，没有修改 GitHub 正文/代码。
