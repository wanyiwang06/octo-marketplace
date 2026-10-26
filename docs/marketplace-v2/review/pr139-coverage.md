# PR #139 对 #131–#138 的覆盖评审

评审对象：`Mininglamp-OSS/octo-marketplace#139`，draft，head `codex/wt-marketplace-v2-foundation` @ `0c7cc1f`，base `eec5223`。68 files，+2509/−362。PR 正文明确“单一后端实现 PR、draft、不关闭任何 tracking issue”；CI `verify` **FAIL**，另有 `codex-review`/lint/secret-scan 通过。

## 0. 总判断

PR139 是 **#131/#132/#133 的地基切片**：CAS、Connector effective、A34 主路径、latest graph、删除保护、冻结安装计划已有实现。但它**没有完成 #131/#132/#133 的全部验收**，**基本没有做 #134**，**#135–#137 是外部客户端**（PR 只有后端部分），**#138 未做**。因此不能据此关闭任何 issue。

| Issue | 覆盖 | 结论 |
|---|---|---|
| #131 MP-01 | **部分** | edit_revision CAS、Connector effective、Admin direct 落地；A01 DTO、A24 Expert/Team 拒绝、完整 effective_digest/回填、A21–A23 DTO 未完成 |
| #132 MP-02 | **部分** | latest graph（A34/install）、删除保护、DB refs 可空落地；HTTP refs 仍强制 version、A01 公共 SQL、A15 observations、A25 resolved 未完成 |
| #133 MP-03 | **部分** | A34 主路径、冻结 plan/HMAC/lease/UNKNOWN 落地；A34 409 矩阵、EXPIRED、Fleet key=attempt_id、成功本地回放、A31 统一、frontmatter 未完成 |
| #134 MP-04 | **基本未做** | 旧 stockmigration/plugin-backfill 仍在；无新 CLI/离线授权/resume/去指针回滚；CI 迁移测试失败 |
| #135 CL-01 | **少量后端** | A26 CAS、Admin direct；A22/A25/A24/DTO 缺口；客户端不在本 PR |
| #136 CL-02 | **少量后端** | A32 新路由、A34；A01/A15/A32 workspace UUID/header=body/错误码缺口；客户端不在本 PR |
| #137 CL-03 | **少量后端** | A34；Buddy 客户端不在本 PR |
| #138 REL-01 | **未做** | 无部署门禁、DB/旧二进制撤销、旧执行器退役、数据安全回滚、A34 失败矩阵证据 |

## 1. 已落地（可对应 issue）

### 1.1 #131：edit_revision + Connector effective

- `migrations/sql/20261026-02-plugin-edit-revision.sql`：`plugins.edit_revision`，默认 1，CHECK > 0。
- `internal/repository/plugin/edit_revision.go`：`GetEditRevision` + `checkEditRevision`；`write.go` 在 `Update`/`RebuildGraph` 行锁内 CAS，成功后 `edit_revision+1`。
- `internal/api/handler/plugin/handler.go`：`pluginResponse/listItemResponse` 返回 `edit_revision`；`upsertRequest` 接受 `expected_edit_revision`；错误映射 409 `edit_conflict`。
- Admin Create 拒绝 `expected_edit_revision`；Admin Update/A25/A26/A12 在 `versionManagement` 下要求 CAS；A26 重传解析 `expected_edit_revision`。
- `migrations/sql/20261026-03-connector-effective.sql` + `internal/repository/plugin/connector_effective.go`：effective 表、写入、读取 overlay、Admin direct 生效、审核批准写 effective。
- 结论：**CAS 主路径和 Admin direct effective 已落地**；但 **effective_digest 只覆盖 `manifest_hash + plugin_hash`**，不是方案要求的完整有效行（name/tags/visibility/附件等变化不改变 digest/ revision）；回填只覆盖 `status=1 AND published` 的 Connector。见 §2.1。

### 1.2 #132：latest graph、删除保护、DB refs 可空

- `internal/repository/plugin/latest_graph.go`：`ResolveLatestGraph` 在 repeatable-read 只读事务内按 `published_version_id + release` 解析；不回退旧版；Team→Expert→Skill 深度 ≤2；Connector 走 effective；audience 校验；节点/边上限。
- `internal/service/plugin/distribution.go` + handler/router：A34 使用该 graph。
- `references.go`：`ValidateReferenceMetadata` 允许 Expert→Skill/Team→Expert 的 `target_version_id=nil`；`migrations/sql/20261026-04-floating-reference-audit.sql` 放宽 CHECK；新增 `rejectPublishedIncomingRelations`，只按未删除来源当前 `published_version_id` 的 refs 保护，不再 `rejectPublishedMutation` 总禁令；`write.go` Delete/DeleteGraph 调用 live + published incoming。
- D6 `referenceAudience`：private→system/same-Space space/same-owner private；space/system→private 拒绝；跨 Space 拒绝。
- 结论：**删除并集、D6、latest graph 的 A34/install 路径已落地**；但 **A12 HTTP 保存仍要求非 Connector 的 `target_version_id` 非空**（见 §2.2），所以“浮动引用可保存”未端到端完成；A01 公共过滤、A15 observations、A25 resolved 未做。

### 1.3 #133：A34 主路径 + 冻结安装计划

- `internal/service/plugin/distribution.go`：A34 单 GET；root 仅 Expert/Team；latest graph；原始文件校验；`bundle_digest` 用 JCS + SHA-256；`maxCapabilityBytes` 16MiB；`Cache-Control: no-store`；无 Fleet/写操作；Connector 用 effective，`files:[]`，`mcp_config`。
- `migrations/sql/20261026-05-install-frozen-plans.sql` + `install_attempt.go`/`install_hmac.go`：
  - `frozen_plan_json`、HMAC key id、`intent_digest`、`attempt_state`、`had_unknown`、lease/retry；
  - `intent_digest` 只含 root/Space/Workspace/runtime/name，不含解析版本；
  - `input` HMAC 含 custom_env；`request` HMAC 绑定冻结 Fleet 请求；
  - `AcquireInstallDispatch`/`SetUnknown`/`Finish`/`Fail` + lease；DISPATCHING→UNKNOWN；晚到成功 CAS；同 key 冻结 plan。
- `install_primitives.go`：`SkillInstallAdapterVersion = "original-v1"`，无 `mk1`；同安装名不同内容返回 `same_install_identity_different_content`。
- 结论：**A34 主路径和“冻结 plan + 未知结果”主路径已落地**；但错误矩阵、EXPIRED、Fleet key、成功回放、A31 统一、frontmatter 仍未完成。见 §2.3。

### 1.4 #135/#136 的后端小切片

- A26 `expected_edit_revision`；A32 新路由 `POST /plugins/{plugin_id}/installations`，单 `X-Workspace-ID` header、单 `Idempotency-Key`、拒绝 Bot、3MiB body、runtime UUID、key 1–200 可见 ASCII、custom_env 限制。
- A34 返回 `selected_version_id`、`version_label`、`effective_revision/digest`、graph/assets。
- Admin `pluginResponse` 增加 `edit_revision/effective_revision/effective_digest`。
- 这些只覆盖客户端契约的一部分，不能替代 #135–#137 的客户端实现。

## 2. 关键缺口（按严重度）

### 2.1 #131：effective_digest 不是“完整有效行”

`connector_effective.go`：

```go
digest := sha256.Sum256([]byte(p.ManifestHash + "\x00" + p.PluginHash))
```

`distribution.go` 也按同一公式校验。结果是：

- `plugin_name`、`tags_json`、`visibility`、`attachment_keys` 等变化不会改变 `effective_digest`；
- `writeConnectorEffective` 的 `ON DUPLICATE KEY` 在 digest 相同但 name/tags 变化时**不增加 effective_revision**；
- 与方案“effective_digest 覆盖完整有效行”不一致，审核观察/客户端无法发现这些变化。

回填 SQL 只选 `status=1 AND deleted_at IS NULL AND listing_state='published'`；未发布/草稿 Connector 没有 effective 行。是否允许它们后续发布由 Admin direct 补行，需要明确。

### 2.2 #132：HTTP 仍强制 `target_version_id`

`internal/service/plugin/service.go` 的 `buildRelations` 仍写：

```go
} else if r.TargetVersionID == nil || strings.TrimSpace(*r.TargetVersionID) == "" {
    return nil, &ReviewFieldError{Field: field, Reason: "released_version_required", ...}
}
```

而 PR 同时：

- 放宽 DB CHECK 允许 NULL；
- 放宽 `ValidateReferenceMetadata` 允许 NULL；
- 迁移测试/仓库层测试用 NULL。

结果是 **A12 HTTP 写入不能保存浮动引用**；DB 层“可空”只在绕过 service 的测试/内部路径成立。这是 #132 的核心验收缺口。

另：

- `buildRelations` 对 Connector 返回 reason `connector_version_not_allowed`，契约是 `connector_version_forbidden`；
- `released_version_required` 不是契约的 `dependency_version_mismatch`；
- 未看到 `view_invalid`/`relation_index_inconsistent` 的 HTTP mapping。

### 2.3 #133：A34 错误矩阵与安装重试缺口

**A34 错误矩阵：**

- `ResolveLatestGraph` 对未发布/下架/删除的目标返回 `ErrNotFound`（404）；`ErrVersionBaselineUnavailable` 走默认 500。
- `writeServiceError` 没有 A34 的 409 `dependency_unavailable`/`dependency_changed`；`ErrDependencyHidden` 映射 403（安装路径），不是 409。
- A34 handler 文档写了 `@Failure 409`，但服务错误没有对应分支；S v2 下架场景不会稳定返回 409。

**安装：**

- `install_attempt.go` 对 `completed` 的成功回执仍调用 Fleet（注释说“Until local Workspace authorization is wired”），不是本地回放；不满足“已成功回放不重发”。
- `completed` 路径仍要求 custom_env HMAC 匹配；成功回放不带 Secret 会冲突。
- Fleet 幂等键使用 `p.IdempotencyKey`（客户端 key），不是 `attempt_id`。
- 状态 CHECK 不含 `EXPIRED`；超过 `retry_until` 返回 `ErrConflict`，没有 EXPIRED 状态。
- 同安装名冲突由 `PreparePinnedSkills` 返回 `InstallationValidationError` → 400，不是 409 `same_install_identity_different_content`。
- `PreparePinnedSkills` 仍调用 `rewriteInstallFrontmatter`，会改写 `SKILL.md` frontmatter 的 `name`，与“保留原字节/不改 frontmatter”冲突；PR 正文“preserves original bytes”只对 A34 内容成立，不对安装成立。
- A31 legacy `/plugins/install` 仍存在，A32 新路由未与 A31 统一到同一 Coordinator；`pinned_install.go` 仍是主要路径。
- A32 的 `X-Workspace-ID` 没有 UUID 校验（`NormalizeInstallationParams` 只校验 1–200 可打印字符）；handler 不接受 body `workspace_id`（unknown field 400），与 #136 的“header=body workspace_id”不一致。

### 2.4 #134：迁移 CLI 基本未做

- PR 没有修改 `cmd/marketplace-stock-migration`/`internal/backfill/stockmigration` 的 CLI 入口/授权/冻结/恢复；旧 `--apply`、`--rollback`、直接 release、`--g4` 仍存在（PR 只改了安装侧）。
- `cmd/plugin-backfill` 仍在文件列表/CI 中。
- 没有新的离线授权、`plan_stale`/`authorization_invalid`、`resume --run-id`、run/auth 存储实现（01 计划里的表未在 PR 迁移中出现）。
- CI 失败正是迁移链：`TestStockMigrationRemainsEditableAndPublishable`（迁移后编辑 edit conflict）、`TestInstallMigrationIndexesUpgradePreservesDataAndScanPlans`（缺 `frozen_plan_json`）、`TestReferenceToInstallationDeliveryMigrationCatchupMySQL`（表不存在）、`TestMigratedGraphFeedsPinnedInstallationMySQL`（version history migration required）。

### 2.5 #135–#138

- **#135 CL-01**：`A22` 精确读取已有，但 PR 没有 A25 `resolved[]`、A24 拒绝 Expert/Team、A21 view/DTO 全量；客户端不在 PR。`AdminCreate/Update` 仍允许 Expert/Team（`conventionType` 包含），A24 的 `zip_only_required` 未实现。
- **#136 CL-02**：A32 路由/A34 有；A01 公共过滤、A15 observations、A32 workspace UUID/header=body、`dependency_changed` 等未完成；Web/CLI 客户端不在 PR。
- **#137 CL-03**：只有 A34 后端；Buddy 客户端不在 PR。
- **#138 REL-01**：没有部署门禁、DB/DSN/旧二进制撤销、旧执行器退役、回滚演练；A34 失败矩阵/409 证据不具备；不能作为切换验收。

## 3. CI 结果

`gh pr checks 139`：`verify` **FAIL**；`lint`/`secret-scan`/`dependency-review`/`code-review` 通过。失败测试：

```text
--- FAIL: TestStockMigrationRemainsEditableAndPublishable/v1
    plugin_stock_migration_test.go:125: edit after migration: plugin edit conflict
--- FAIL: TestInstallMigrationIndexesUpgradePreservesDataAndScanPlans
    plugin_install_migration_schema_test.go:96: Unknown column 'frozen_plan_json'
--- FAIL: TestReferenceToInstallationDeliveryMigrationCatchupMySQL
    plugin_delivery_upgrade_test.go:28: Table '...plugin_install_attempts' doesn't exist
--- FAIL: TestMigratedGraphFeedsPinnedInstallationMySQL
    plugin_migration_installation_test.go:65: migrated installation: version history migration required
```

这与“迁移/旧 CLI 未改造”一致；PR 正文的本地 `go test ./...` 不能代表当前 CI 通过。

## 4. 建议

1. **先修 #132 的 HTTP 浮动引用**：`buildRelations` 允许非 Connector 的 `target_version_id=nil`；补 A12 端到端测试，而不是只测 DB/validator。
2. **修 #131 effective_digest**：对完整有效行（name/tags/visibility/manifest/package/attachments）做 JCS/SHA-256；revision 只在完整 digest 变化时递增；补回填范围说明。
3. **修 #133 A34 错误矩阵**：`ErrNotFound`/`ErrVersionBaselineUnavailable`/`ErrDependencyHidden` 在 A34 场景映射 409 `dependency_unavailable`/`dependency_changed`；补 S v2 下架测试。
4. **修安装回放**：completed 本地回放、custom_env 成功回放免 Secret、Fleet key=attempt_id、EXPIRED、同名单 409、去 frontmatter rewrite；A31 与新 A32 同引擎。
5. **#134 另开/补 PR**：新 CLI 冻结计划/离线授权/resume/去指针回滚；先修 4 个迁移测试。
6. **#135–#138** 不能由本 PR 关闭；客户端/REL 仍按 issue 验收。

## 5. 结论

PR139 可以作为 **#131/#132/#133 的地基**继续迭代，但当前是 draft、CI 红、关键契约缺口仍在；它**不解决 #134/#135–#138**，也不能关闭任何 #131–#138。建议先修上述 1–4，再评估是否把 #131/#132 的部分验收标为完成。
