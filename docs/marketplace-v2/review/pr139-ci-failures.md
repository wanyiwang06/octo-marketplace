# PR #139 CI 失败根因

对象：`Mininglamp-OSS/octo-marketplace#139`，head `0c7cc1f`，base `eec5223`。
失败 job：`verify`（run `40293406824` / job `121473548779`），MySQL 8.0 integration；`lint`/`secret-scan`/`dependency-review`/`code-review` 通过。

## 0. 结论

四个失败都在 `internal/db` 的 MySQL 集成测试，分为两类：

1. **测试未随新契约更新**（CAS、迁移顺序、旧断言）；
2. **真实迁移缺口**（migrated Connector 没有 `plugin_connector_effective`，新 latest graph/install 无法读取）。

不是 flaky，也不是 MySQL 环境问题；都是确定性的。

## 1. `TestStockMigrationRemainsEditableAndPublishable`

**错误：**

```text
plugin_stock_migration_test.go:125: edit after migration: plugin edit conflict
```

**位置：** `internal/db/plugin_stock_migration_test.go:112-125`。

**根因：**

测试在迁移后直接调用：

```go
req := pluginsvc.WriteRequest{...}
updated, err := svc.Update(ctx, owner, id, req)
```

`req` 没有 `ExpectedEditRevision`；而 PR139 在 `versionManagement` 开启时：

- `Update`/`RebuildGraph` 走 `checkEditRevision`；
- `RequireEditRevision = s.versionManagement`；
- `ExpectedEditRevision == nil` 时返回 `ErrEditConflict`。

迁移本身给 `plugins.edit_revision` 默认 1，数据没问题；失败是**测试没有传迁移后读到的 revision**。

**性质：** 测试/调用方未适配 CAS。若这是真实老客户端，会从 200 变成 409；这是 CAS 的预期行为，但需要升级客户端/测试。

**修复：**

```go
p := detail.Plugin
req := pluginsvc.WriteRequest{
    ExpectedEditRevision: &p.EditRevision,
    ...
}
```

并补一条“缺少 expected_edit_revision → 409 edit_conflict”的负例，而不是让测试绕过。

## 2. `TestInstallMigrationIndexesUpgradePreservesDataAndScanPlans`

**错误：**

```text
plugin_install_migration_schema_test.go:96: Error 1054 (42S22):
Unknown column 'frozen_plan_json' in 'field list'
```

**位置：** `internal/db/plugin_install_migration_schema_test.go:93-99`。

**根因：**

测试先把库迁到 `20261024-04-reference-index-state.sql`（旧 schema），然后**在应用 `20261026-05-install-frozen-plans.sql` 之前**调用：

```go
repo := pluginrepo.New(database)
attempt := pluginrepo.InstallAttempt{...}
repo.ClaimInstallAttempt(ctx, tenantScope(), attempt)
```

PR139 的 `ClaimInstallAttempt` INSERT 已经包含 `frozen_plan_json/hmac_key_id/intent_digest/attempt_state/had_unknown` 等列，而旧 schema 还没有这些列，所以直接 `Unknown column`。

**性质：** 测试顺序错误。它想验证“升级保留旧 receipt”，但用了新代码在旧 schema 上写新 receipt，因果反了。

另外该测试还有第二个过期断言：

```go
if n, err := migrate.Exec(...); err != nil || n != 1 { ... }
```

现在待应用迁移是 `20261026-01/02/03/04/05`，`n` 应为 5，不再是 1。

**修复：**

1. 在旧 schema 上用**旧列 raw SQL** 插入一条 receipt（不要用新 `ClaimInstallAttempt`）；
2. 应用全部迁移；
3. 用新 `GetInstallAttempt` 读回，验证 receipt 保留；
4. `n != 1` 改为实际待应用数（当前 5）；
5. 索引检查保持。

## 3. `TestReferenceToInstallationDeliveryMigrationCatchupMySQL`

**错误：**

```text
plugin_delivery_upgrade_test.go:28:
Table '...plugin_install_attempts' doesn't exist
handling 20261026-05-install-frozen-plans.sql
```

**位置：** `internal/db/plugin_delivery_upgrade_test.go:20-33`。

**根因：**

这个测试模拟“PR2 已部署、PR3 的早期迁移还没跑”的 catch-up：

```go
for _, m := range all {
    if m.Id == "20261024-02-plugin-install-attempts.sql" ||
       m.Id == "20261024-03-stock-migration-batches.sql" ||
       m.Id == "20261026-01-install-migration-access-indexes.sql" {
        continue
    }
    prior = append(prior, m)
}
```

它故意把 `20261024-02`（创建 `plugin_install_attempts`）排除在 prior 之外，但 **没有排除新的 `20261026-05`**。于是 05 在 prior 阶段先执行，执行 `ALTER TABLE plugin_install_attempts ...` 时表还不存在，直接失败。

同时测试仍期望：

```go
if n, err := migrate.Exec(...); err != nil || n != 3
```

加入 04/05 后，待应用数已不是 3。

**性质：** 测试的 catch-up 集合没有随新迁移更新；05 依赖 02 的表，不能在 02 之前执行。

**修复：**

- prior 排除 `20261026-05`（因为 05 依赖 02 的 attempts 表）；
- 让完整 source 按 ID 顺序应用 `02 → 03 → 01 → 05`；
- `n != 3` 改为实际待应用数（当前 4）；
- 若要验证“05 单独执行”的健壮性，另加显式依赖/前置检查，而不是让 catch-up 测试承担。

## 4. `TestMigratedGraphFeedsPinnedInstallationMySQL`

**错误：**

```text
plugin_migration_installation_test.go:65:
migrated installation: <nil> version history migration required calls=0
```

**位置：** `internal/db/plugin_migration_installation_test.go:60-66`。

**根因（真实代码缺口）：**

- 测试用 `stockmigration.ApplyRoot` 迁移一个带 MCP Connector 的 Expert/Team；
- `stockmigration.writeNode` 对 Connector 只写 `plugins/plugin_relations`，**直接 return，不写 `plugin_connector_effective`**；
- PR139 的 `LatestGraph` 在 `visit` 到 Connector 时调用 `overlayConnectorEffective`；
- 没有 effective 行 → `ErrVersionBaselineUnavailable`（错误文本 `version history migration required`）；
- `resolvePinnedInstallation` 因此失败，`remote.calls=0`。

这不是测试断言问题，而是 **#134 迁移 CLI 没有为迁移生成的 Connector 写 effective**。迁移 03 只回填“已存在且 published”的 Connector，回填不到 stockmigration 之后新建的 Connector。

即使补上 effective，这个测试还有过期断言：

```go
if !strings.Contains(name, "-mk1-") || !strings.Contains(d.Skills[0].Content, "name: "+name) { ... }
```

新契约要求保留市场原名/原字节、无 `mk1`、不改 frontmatter；应改为断言原名、原 `SKILL.md` 字节。

**修复：**

1. `stockmigration.writeNode` 为 Connector 写 `plugin_connector_effective`（revision 1 + 完整 effective_digest；至少要覆盖 manifest/package/visibility）；
2. 或在迁移结束时统一补 effective；
3. 更新测试断言：无 `-mk1-`、内容原字节、`name` 不被改写；
4. 用 `plugin_connector_effective` 的迁移/回填测试覆盖“迁移新建 Connector”场景。

## 5. 四个失败的共同点

| 失败 | 类型 | 是否代码 bug |
|---|---|---|
| StockMigration edit conflict | 测试未传 `expected_edit_revision` | 否（测试/调用方适配） |
| InstallMigration unknown `frozen_plan_json` | 测试在 05 前用新 repo 写旧表 | 否（测试顺序） |
| Delivery catch-up table missing | prior 未排除 05，05 依赖 02 | 否（测试顺序/迁移依赖） |
| Migrated graph version baseline | 迁移未写 Connector effective | **是**（#134 迁移缺口；测试断言也过期） |

## 6. 建议修复顺序

1. 修 `stockmigration.writeNode` 的 Connector effective（真实功能），同时更新 `TestMigratedGraph...` 的 `-mk1-`/frontmatter 断言。
2. 修 `TestInstallMigration...`：旧 receipt 用 raw SQL 插入，应用全部迁移后再读；更新待应用数。
3. 修 `TestReference...`：prior 排除 05，确保 02 先于 05；更新待应用数。
4. 修 `TestStockMigration...`：传 `ExpectedEditRevision`，补缺失 CAS 的负例。
5. 重新跑 `verify`，确认 4 个 DB 失败消失；同时检查 `go test ./...` 是否还有非 DB 的隐藏失败。

## 7. 备注

- 失败 job 是 `verify`，不是 lint/secret-scan；因此不是格式或密钥扫描问题。
- CI 里 MySQL 初始化正常（容器启动、数据库创建），失败发生在测试 SQL/迁移阶段。
- PR 正文称本地 `go test ./...` 通过；当前 CI 的 MySQL 集成测试失败，说明本地未覆盖/未按 CI 顺序执行。
