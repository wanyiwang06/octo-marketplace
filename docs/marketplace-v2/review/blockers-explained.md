> **历史记录（B 模型）：** 本文写于 P 模型（上架冻结依赖）修订之前，保留作评审证据；B/latest 结论已被 [decisions-20261026-pin.md](../decisions-20261026-pin.md) 取代。

# #131–#138 关键阻塞与修正（案例版）

本文把 [issue-review-131-138.md](issue-review-131-138.md) 里的阻塞项展开成具体场景，便于产品/后端/客户端对齐。基线与决策仍以 `00-decisions.md`、`02-api-contracts.md`、`decisions-20261026.md` 为准。

## 1. #132 D1：旧 Skill 到底还保不保护？

**一句话：** 在 B 下，Team 引用的是 Expert ID，间接依赖取该 Expert **当前正式发布版**；Expert 新版本移除的旧 Skill，不再被这个 Team 保护。

**案例：**

1. Expert `E` 发布 `e1`，`e1` 引用 Skill `S`。
2. Team `T` 发布 `t1`，`t1` 引用 `E`。
3. `E` 发布 `e2`，`e2` 删除了 `S`。
4. 此时 `T` 的公共图是 `T -> E(ID) -> e2`，`e2` 已没有 `S`；`S` 不属于 `T` 的依赖。
5. 如果 `S` 没有其他 live 引用、没有其他来源的 current published refs、没有别的 Team 间接引用，`S` **应该可以删除**。

**当前 issue 的问题：** #132 验收写“t1→e 的已发布/间接依赖仍保护相关目标；不再把 t1 固定到 e1 作为判定依据”。这句话前后矛盾：既然不固定 e1，就不应保护 e1 里的旧 Skill；但“仍保护相关目标”容易被读成继续保护 S。

**正确写法：**

- `t1` 保护 `E`（ID）以及 `e2` 的**当前**发布依赖；
- 旧 `S` 失去 `t1` 的保护，并集为空时可删；
- `t1` 的审计 pin `e1` 永远不作为保护依据；
- 如果 `E` 的 current 草稿重新引用 `S`，`S` 仍受 **live incoming** 保护；
- 如果另一个来源的 current published refs 引用 `S`，`S` 仍受 published refs 保护。

**验收：** 构造 `T→E`、`E e2` 删除 `S`，断言 `S` 可删；再让 `E` 的草稿重新引用 `S`，断言 `S` 又被 live incoming 保护。这样才能区分“历史发布保护超集”和“当前 live/published 引用”。

## 2. #132 间接保护链写乱了

**当前文本：** “published Team→成员 Expert→该 Expert 发布依赖的 Team→最新发布 Expert→Skill/Connector”。

**问题：** 中间多了一个“该 Expert 发布依赖的 Team”跳。Expert 的关系只有 Skill/Connector，不引用 Team；这条链会把不存在的 Team 递归写进来，导致过度保护、循环或漏保护。

**正确链：**

```text
published Team 快照
  -> 成员 Expert ID（Team 边）
  -> 该 Expert 当前 published_version_id 的 refs
  -> Skill / Connector
```

只有**一跳成员 Expert**，然后直接读该 Expert 的 current published refs；不再展开第二个 Team，不读 e1 历史，不读 current 草稿（草稿由 live incoming 单独保护）。

**案例：** `T` 有成员 `E1`、`E2`；`E1` 有 Skill `S1`，`E2` 有 Skill `S2`。删 `S1` 时，保护路径应是 `T -> E1 -> S1`；不应该从 `E1` 再找“E1 依赖的 Team”。如果写成多跳，可能把 `E2` 或无关 Team 的依赖也算进来。

## 3. #132 解析器不能只用 `plugin_id + view`

**案例 A（A03 历史根）：** `E` 当前 published v2，用户查看历史版本 `v1` 的图。`plugin_id + view=published` 会解析成 v2，返回错误的根；A03 需要显式 `version_id=v1`。

**案例 B（A13/A14 精确提交）：** `E` current v3，用户发布/提交 `v2`。如果解析器只按 `plugin_id + published`，会解析成 v3，审核/发布错内容；必须用请求里的 `version_id=v2`，并校验它归属 `E`。

**正确输入：**

```text
{root_plugin_id, root: published | explicit version_id | editing}
```

- A01/A34：`published`；
- A03：显式 `version_id`（历史根）；
- A13/A14：请求提交的 `version_id`；
- 编辑视图：`editing`；
- 对显式 version **绝不回读 current**。

**SQL vs 唯一解析器：** A01 必须在 COUNT/排序/分页前过滤，所以需要 SQL 谓词；但“所有读取只调用一个 Go resolver”与手写 SQL 会变成两套 latest 逻辑。正确做法是把 SQL 谓词定义为同一语义核心（同 owner、同一规则、生成/校验、等价测试），或明确它是唯一允许的共享实现，禁止其他手写 latest。

**案例 C（分页不一致）：** 第 1 页 SQL 过滤隐藏了 `E`，COUNT 仍按未过滤算，`total` 多 1；或列表显示 `E`，A34 用另一套 latest 逻辑却失败。必须 SQL 与 resolver 同口径。

## 4. #131 refs 可空与现有 CHECK 冲突

**案例：** 新建 Expert，关系 `expert_skill -> S`，但不传 `target_version_id`（B 允许，仅审计）。当前：

- `chk_plugin_version_ref_type` 要求 `expert_skill`/`expert_team_expert` 的 `target_version_id` 非空；
- `ValidateReferenceMetadata` 对 nil 返回 `ErrInvalidRelation`；
- 保存会 400 或 DB 失败。

**结果：** 实现者为了“通过”，可能把当前 latest 写进 `target_version_id`，把 B 又变回固定 pin；或者 B 的保存路径直接不可用。

**正确修复：**

- 增量 ALTER 放宽/删除 `chk_plugin_version_ref_type`，只要求 Connector 为 NULL；
- `ValidateReferenceMetadata` 对 Expert→Skill/Team→Expert 允许 nil；
- refs 插入允许 NULL；
- 把 `internal/repository/plugin/references.go`、`internal/model/plugin_snapshot.go` 纳入 MP-01/MP-02 owner；
- 测试“省略 version_id 合法；显式 version_id 必须属于目标且已 release”。

## 5. #133 输入身份不能含“规范版本”

**案例：**

1. 首次安装 `E`，解析到 `S v1`，Fleet 请求已发出，网络超时 → UNKNOWN。
2. 之后 `S v2` 正常发布。
3. 用户用同一个 Idempotency-Key 重试。
4. 按 D9，应重放首次冻结的 `v1` plan，不能因 `v2` 发布变成 409 或重建 `v2`。
5. 如果 attempt 的 input identity 含“规范版本”，`v2` 会让输入摘要变化 → 409 或重建，直接违反 D9。

**正确：**

- `intent_digest` 只含 root plugin ID、Space、Workspace、runtime、resource_name；
- `custom_env` 只在完整 `input_hmac`；
- 解析出的 root/依赖版本只属于**冻结 plan**，不属于输入身份；
- 同 key 重放原 plan；显式输入变化才 409 `idempotency_input_mismatch`；
- 想用最新版：新 key 新安装，重新解析。

**补充：** 已知 SUCCEEDED 回放不要求重填 Secret；非成功重试必须能重建完整请求（含原 Secret），否则安全失败/保留 UNKNOWN，不能一律报 input mismatch。

## 6. #128 已合入代码的清理没有 owner

**案例 A（两套执行器）：** 新代码加了 `InstallCoordinator`，但 main 仍有 `pinnedInstallation`/`installPinnedLegacy`；flag on 时 A31/A32 走旧 pinned 路径，flag off 走新路径。测试只测 flag off，就宣称“一个引擎”，生产 flag on 仍是旧逻辑。

**案例 B（mk1）：** main 的 `SkillInstallAdapterVersion="mk1"`、`PreparePinnedSkills` 仍会给每个 Skill 生成 `mk1` 别名并改写 `SKILL.md` frontmatter。D3 说“不允许共存、不改 frontmatter”，但旧代码还在，安装结果仍被改写。

**案例 C（指针回滚）：** main 的 `marketplace-stock-migration --rollback` 仍执行“restore root pointers”。REL-01 做数据安全回滚演练时，如果调用它，就违反 D5/D10；但 issue 没有要求删除/禁用。

**案例 D（pinned readiness）：** `scripts/pinned-release-readiness.sql` 仍按 `target_version_id` 遍历；B 下这是可空审计值，SQL 会漏掉浮动依赖或验证错图。

**正确：** 在 MP-02/03/04/REL 的 owner/验收里明确：

- 删除 `pinnedInstallation`/`installPinnedLegacy`/`SkillInstallAdapterVersion`/`PreparePinnedSkills`/`adapter_version`；
- A31/A32 在 flag on/off 都只进 `InstallCoordinator`；
- 删除 `--rollback` 指针回滚、直接插 release、`--g4` 自证批准；
- 重写 readiness SQL；
- 测试“旧路径不可达/旧字段为 0”。

## 7. #135 把 A22 当“最新解析器”

**案例：** CL-01 预览 Expert ZIP，清单引用 Skill `S`。页面要显示“S 本次解析到的最新发布版”。

- A22 是 `GET /admin/plugins/{id}/versions/{version_id}`：必须给精确 version_id，并做归属检查；它不是 latest resolver。
- 如果 CL-01 不传 version_id，A22 无法解析；如果拿 A21 current/editing，可能显示 S 的草稿或旧版本，而不是最新 published。
- 正确：先通过 A03/A10（或 A25 `resolved[]` 的 import 时观察）拿到目标最新发布身份，再用 A22 精确版本读正文/A21 effective 读 Connector。

**另一个冲突：** A25 同 key 回放会回放**首次** `resolved[]`；因此“目标后来发新版时下一次预览显示新版本”不能靠 A25 同 key 回放，必须走新的 A03/A10 读取或新 key/新 import。

**正确接口范围：** CL-01 需要 A03/A10 才能满足“预览最新解析”；plan.json 的 `consumes_api_ids` 要补上，不能只列 A21–A27。

## 8. #138 A34 失败矩阵、部署门禁与指针回滚

### 8.1 A34 失败矩阵漏 409/400

**案例：** Team `T` 的成员 Expert `E` 的最新版 `e2` 引用的 Skill `S v2` 被下架。B 下 `T` 的公共图不可用，A34 应该 fail closed。

当前 REL-01 只列 413/500/404 和“missing/unauthorized→404”，没有要求 409 `dependency_unavailable`/`dependency_changed`。测试者可能接受 404/500，就无法证明 B 的核心失败语义。

**正确：**

- 根/目标不可见 → 404；
- 最新依赖未发布/下架/状态不合法 → 409 `dependency_unavailable`/`dependency_changed`；
- path/type/binary 不合法 → 400；
- 预算 → 413；文件/refs 完整性 → 500；
- 必须把“S v2 下架 → A34 409”作为 B fail-closed 验收。

### 8.2 部署门禁挡不住本地 CLI/DB

**案例：** 旧 `plugin-backfill` 和 `marketplace-stock-migration` 是本地二进制，用 DSN 直连 MySQL。网关/网络隔离/停前端按钮都挡不住操作者在本机执行 `--all-spaces --apply`。REL-01 如果只验证“网关拒绝”，旧路径仍可写。

**正确：**

- 撤销旧 CLI 的 DB 账号/DSN/权限；
- 移除/deny 旧二进制；
- 枚举 A14–A19/placement 等全入口；
- 验证旧 `plugin-backfill` 和 `marketplace-stock-migration --apply` 都不能创建/切换/写 release；
- 记录命令、操作者、制品、拒绝证据。

### 8.3 指针回滚仍保留

**案例：** 运维在回滚演练中执行 `marketplace-stock-migration --rollback`，它把 `current/published` 指针改回旧值。REL-01 的验收却写“不把 current/published 改回旧值”，但没要求删除这个模式；演练可以“通过”却用了禁止路径。

**正确：** 删除/禁用 `--rollback`；回滚只允许兼容二进制 + 从历史生成新版本；用测试断言该 flag/路径不存在。

### 8.4 D1 published refs 范围

**案例：** 一个资源只有历史发布引用、当前 `published_version_id` 已不再引用它；如果保护集合取“全部历史发布”，它永远删不掉，回到“曾发布禁删”。正确只取**未删除来源的 current `published_version_id`** 的 refs；历史发布不是保护超集。

## 9. 其他重要修正（简表）

| Issue | 修正 | 案例/原因 |
|---|---|---|
| #131 | A01 DTO 边界 | MP-01 只定义 Go 类型/序列化槽位，不写“mine 必返”；MP-02 才计算 status/reasons。否则 MP-01 无法真实填充，可能伪造 available |
| #131 | 不要再 CREATE release/refs | main 已有 `plugin_version_releases`、refs、索引；只提交增量字段（edit_revision/declared_visibility/version_label/revision_no/content_digest/effective/review freeze） |
| #131 | Connector effective 回填 | 旧 Connector 没有 effective 行，切换后 A34/安装/引用失败；回填 revision 1 + digest |
| #131 | A21/A22/A23/A26 wire | Admin DTO 必须省略 published_version_id；A21 默认 editing/current；A22 精确 version；A23 默认 current 并标实际版本；A26 重传 expected_edit_revision；补错误码 |
| #131 | 源 refs 与目标缺失分开 | 源版本/refs 不一致 = 500 relation_index_inconsistent；refs 一致但目标 Plugin 缺失 = 正常 dependency_unavailable |
| #132 | D6 矩阵 | private 来源可引用已发布 system、同 Space space、同 owner private；space/system 不得引用 private；补 private→public 正例 |
| #132 | D8 relations | 只有 A12 完整 Write 要求 relations；Skill import 不接受 relations；A25 关系来自 ZIP |
| #132 | A15 observations | 加 `resolved_dependency_observations`（Skill/Expert）+ `connector_observations`；否则审核看不到冻结依赖版本 |
| #132 | view 矩阵 | mine 仅 editing；公共仅 published；Connector effective；非法 400 details.field=view；补负例 |
| #132 | 移除总禁令 | 明确移除 `rejectPublishedMutation`，只保留 live 保护再补并集；否则曾发布仍不可删 |
| #132 | 匿名 only hidden | 只合并不可披露部分；其他可见原因仍可列出，不泄漏隐藏数量 |
| #133 | 状态机 | 补 EXPIRED（或 retry_until 派生）；DISPATCHING 租约到期→UNKNOWN；lease owner 才能 dispatch；晚到成功 CAS |
| #133 | attempt 表迁移 | PK 去掉 space_id、加 state/lease/plan/HMAC/retry_until；旧 completed 行只按已验证结果映射 succeeded，否则 unknown/人工 |
| #133 | A31/A32 细节 | X-Workspace-Id 单头 UUID、A32 header=body、key 1–200 可见 ASCII、3MiB/16MiB、拒绝 Bot、fleet_key=attempt_id |
| #133 | 同名冲突 | 映射 409 same_install_identity_different_content；用 Fleet 现有冲突，不新增预检查询 |
| #133 | #126/variant | 引用 #128，不引用 #126；删除 `variant`/`transform_variant` 残留；凭据检查走既有 400 或客户端控制 |
| #134 | 旧 CLI | 纳入 `marketplace-stock-migration`/stockmigration/batch roots；删除 `--rollback`/直接 release/`--g4`/固定 pin；REL 验证旧入口不可达 |
| #134 | CLI 契约 | `plan_stale`/`authorization_invalid` + 非零退出/blocked，不要 409；apply/resume 无幂等键，run 由 plan_digest/root 仲裁；补 resume/frozen plan/run DDL |
| #134 | reuse/source_digest | reuse 漂移 apply 时 `plan_stale`；`source_digest` 映射 `input_digest`；补 JCS/排序/plan_digest 向量 |
| #135 | Admin 审核 | Admin 无任何提交/待审/批准；直接生效；仅 Tenant 审核；A24/A25 不创建 pending review |
| #135 | octo-admin 基线 | eec5223 是 marketplace 基线；补 admin commit/date 或标外部依赖 |
| #135 | DTO/版本 | 补 view/selected_version_id/version_label/current_version_id/has_unpublished_changes/is_published/effective_revision/digest；Admin DTO 省略 published_version_id |
| #135 | A25 replay | 同 key 回放首次 resolved；下次预览最新要走 A03/A10 或新 key/import |
| #135 | A23 vs A06/A07 | Admin 当前内容用 A23，默认 current 并标实际 version；public published 用 A06/A07；不混用 |
| #135 | schema2/manifest | 字段是 `version_id`（非 target_version_id），提供时须属于目标且已 release；补 role/member_key/sort_order/resolved-name 唯一/额外文件拒绝 |
| #136 | scope/baseline | 补“仅 marketplace 后端”banner；octo-web/cli 外部依赖与各自基线；不要把 eec5223 当客户端基线 |
| #136 | A01 范围/A09 | A01～A10 去掉 A09；改 A01–A08/A10/A12–A19/A32/A34 |
| #136 | A32/A34 | 补 X-Workspace-Id + header=body；A34 不 session cache，no-store，刷新重新 GET |
| #136 | A15/candidate | 补 resolved_dependency_observations；Candidate Connector 用 effective_revision，不要只筛 published |
| #136 | 错误码 | 显式输入变化才 idempotency_input_mismatch；市场配置/依赖变化用 dependency_changed/dependency_unavailable |
| #137 | 小修 | 服务端出站强制 retry_until；补 same_install_identity_different_content；MP-04 不是 CL-03 依赖；“原 plan”改“原请求/完整规范化参数” |
| #138 | 门禁/A34/回滚 | 见第 8 节；另补 Admin 无审核措辞、D1 current published 范围、删除无出处的数量上限、Admin 预览改本地 manifest + post-import resolved |

## 10. 建议修复顺序

1. **00/02 契约**：A01 DTO 边界、refs 可空/CHECK、Connector effective 回填、解析器根选择、D9 输入身份/错误码、A34 失败矩阵、CLI 契约。
2. **01 + 8 issue**：移除 #128 pinned/mk1/pointer rollback/direct release/pinned readiness 的 owner/验收；补 A21–A26、A15 observations、refs CHECK、Connector backfill。
3. **GitHub #131–#138**：按修正稿更新正文（它们目前是本地 issue 的逐字复制）。
4. **代码**：MP-01 → MP-02 → MP-03/MP-04/CL-01 → CL-02/CL-03 → REL-01。

## 11. 边界

- 本文是评审解释，不是实现；未运行服务/MySQL/Fleet/Daemon/客户端。
- 基线代码用 `git show eec5223:` 核对；本地工作区 `99cc261` 不代表当前 main。
- #135–#137 的客户端源码断言未在 workspace 复核（没有 octo-admin/web/cli/buddy checkout）。
