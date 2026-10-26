# 规则裁定与最终决策（2026-10-26 · P 模型）

> **P 模型取代 B：** 本文已按《[决策修订：P 模型](decisions-20261026-pin.md)》改写。Expert→Skill、Team→Expert 以已发布源快照中的 `target_version_id` **pin** 为权威；公共读取/A01/A03/A34/安装/审核观察只读 pin，不再解析目标最新版。Connector 保持 ID + `plugin_connector_effective`。历史段落中的 B/latest 表述仅为历史记录，均被 P 取代。**当前 PR #139 是 B 模型实现，必须按 P 重做，不能按现状评审或合并。**

状态：**最终决策已锁定（2026-10-26 · P 模型）**。用户在 PR #139 评审后明确确认：Expert 的技能版本、Team 的成员 Expert 版本都按**上架时版本**冻结；private 保持原有语义，允许 private Expert 引用同 owner 的 private Skill。本文替代 B 模型的“读时最新解析”，也替代早期固定 pin 讨论；D1–D10 中与依赖版本无关的部分保持有效，与本文冲突的旧段落以本文为准。

## 0. 决策清单

| ID | 决策 |
|---|---|
| D1 | 保留原有删除能力。删除保护 = live incoming（含草稿）∪ 未删除来源当前 `published_version_id` 的 published refs（按**精确 `target_version_id` pin**）∪ Team 间接路径（已发布 Team 快照 → 成员 Expert pin → 该 **pinned Expert 版本**的 refs）。Expert 发新版本移除旧 Skill 不自动释放 Team 保护：只要 Team 仍 pin 旧 Expert 版本，旧版本 refs 仍保护；Team 重新发布 pin 到移除该 Skill 的 Expert 版本（或删除边）后才释放。没有保护引用的曾发布资源也可删。移除 `rejectPublishedMutation` 对删除的总禁令。只改删除，不改“已发布资源收窄 visibility”的现有行为。 |
| D2 | 不做 A09 restore-to-draft；从 MP-01/API/plan/CL-02 删除 A09。 |
| D3 | 不允许同名/多版本共存；同名冲突失败；普通安装保留市场原名/原字节；不改 Fleet/Daemon。 |
| D4 | A01 `dependency_availability` 四字段；`evaluated_view/evaluated_version_id` 指 Expert/Team 自己的 published/editing；`target.version_id` 是**已发布快照里的 pin**，不是目标最新版；可用性按 pin 判定；名称只取 pinned released snapshot/effective 内容，不披露其他 owner 的 editing name；无权依赖只匿名不可披露部分。 |
| D5 | 生产迁移用 CLI + 离线授权；不做 A29/A30 HTTP；CLI 两次 preview/冻结摘要；回滚必须数据安全。**生产切换时对存量已发布 Expert/Team 执行一次 pin 回填（方案 A）：合法且仍 release 的已有 pin 保留，缺失/无效的冻结为迁移时目标当前正式发布版，无可用目标则该根 blocked。** |
| D6 | private 来源可引用已发布 system、同 Space 的 space、**同 owner 的 private**；private Expert→private Skill、private Team→private Expert 是合法主路径。space/system 来源不得引用 private；跨 Space 禁止；同 Space 不同 owner 的 private 仍禁止。public 列表隐藏 pin 不可用的 Expert/Team，A34/安装整体失败；owner 只能看诊断，不能安装坏图。**private 同 owner private→private 合法，不做全面禁止。** |
| D7 | Admin 不用审核，写入直接生效并同步 release/effective；Admin 写入时解析并冻结缺失 pin。Tenant 沿用既有审核策略；审核观察 = **pin 观察** + Connector effective；目标后来发新版**不** 409；只有 pin 版本失效/目标不可用或 Connector effective 漂移才 409 `dependency_changed`；安装/A34 读 pin/effective。 |
| D8 | `edit_revision` + `expected_edit_revision`；更新必带、新建省略；缺失/过期 409；`relations` 必填只适用于 A12 完整 Write（Skill import 不接 relations，A25 关系来自 ZIP 清单）；Admin/Web/CLI/Bot 都改。 |
| D9 | 保留 A32 URL；A31/A32 同一引擎；同 key 同输入重放原 plan；**上游目标发新版不阻断**；解析出的根/依赖版本只属于冻结 plan，不进 `intent_digest`/`input_hmac`；**新 key 新安装按当前源发布快照（含其 pin）**，不按目标最新；A31/A32 单 `X-Workspace-Id`，A32 header=body；显式安装输入变化 409，使用新 key 新安装；仅首次成功计数。 |
| D10 | 新改造完成后旧市场执行器不可达；部署门禁包含撤销旧 CLI 的 DB 账号/DSN/权限并移除或 deny 旧本地二进制，不只网关/网络隔离；回滚接受兼容二进制 + 新版本恢复；不恢复指针/legacy。 |
| P | **Expert→Skill、Team→Expert 在发布/提交时把目标当前正式发布版冻结为源快照中的 `target_version_id` pin；公共读取/A01/A03/A34/安装/审核观察只读 pin，不解析目标最新版。** 草稿可空；发布/提交解析缺失 pin；显式 pin 必须归属目标且为正式 release。Connector 按 ID + effective，关系 `target_version_id` 必须 NULL。Team 的 `member_key/role/is_leader/sort_order` 仍由 Team 边决定。 |
| 审核 | 沿用现有 Space 策略：默认自动，Space 可关；自动模式内部重试一次。**提交冻结 pin 观察 + Connector effective；批准只校验 pinned 版本/effective；目标发新版不 409；只有 pin 版本失效/目标不可用或 effective 漂移返回 409 `dependency_changed`；批准后 pin 不变，重新发布源版本才刷新 pin。** |

## 1. P 引用模型

### 1.1 定义

- **pin** = 已发布 Expert/Team 版本快照中每条 `expert_skill`/`expert_team_expert` 关系的 `target_version_id`。它是公共读取、A01、A03、A34、安装和审核观察的唯一依赖版本依据。
- 草稿/未提交快照的 `target_version_id` 可为 NULL。发布/提交动作把缺失 pin 解析为目标**当时的当前正式发布版**并写入快照；显式给出的 pin 必须属于目标且为正式 release，否则 400 `dependency_version_mismatch`。pin 只由发布/提交动作写入，发布后不变。
- 目标之后发新版、下架新版、删除新版，都不改变已发布来源的 pin；来源仍读旧 pin。**不回退到更新版本，也不自动切换。**
- **Expert→Skill**：读取 Expert 已发布版本快照的 Skill pins。pin 的 Skill 版本被删除/下架/失去正式 release，或目标 plugin 不可读/已删除/embedded/未上架/受众不通过 → 依赖不可用；Expert owner 必须编辑并发布新的源版本，重新冻结 pin。
- **Team→Expert**：读取 Team 已发布版本快照的成员 Expert pins；每个成员 Expert 使用**该 pinned Expert 版本**。该 Expert 版本的 Skill 依赖沿用**它自己快照中的 pins**，不因 Team 读取二次解析。Team 的 `member_key/role/is_leader/sort_order` 仍由 Team 边决定。
- **Expert→Connector**：按 ID 读 `plugin_connector_effective` 的当前有效修订，Connector 无版本 pin。发布时记录 effective 观察值，审核/安装按 effective 校验；若后续明确要求 Connector 冻结，再单独立项。
- **重新发布** = 发布新的源版本快照。发布事务重新冻结该版本所有关系的 pin：缺失 pin 解析为发布时目标当前正式发布版；显式 pin 校验后保留。因此重新发布默认刷新所有未显式指定的 pin。对已 release 的同一源版本重复发布是幂等操作，不刷新 pin。
- 已发布快照的 `target_version_id` 是权威 pin；Connector 关系必须 NULL。

### 1.2 pin 解析与一致性

- A01、A03、A34、安装、审核观察复用同一个 **`PinnedDependencyResolver`（pin 解析服务）**，不得各写一套，也不得回退调用任何 latest resolver。
- 根选择不变：A01 公共与 A34 用根 `published_version_id`；A01 `mode=mine`/编辑视图用 `editing`；A03 历史根用显式 `version_id`；A13/A14 用请求提交的 `version_id`（Connector 用当前编辑修订）。显式 version 必须归属根且为 release；解析不得回读 current。
- 依赖边不再解析目标当前版本，直接读源快照的 `target_version_id`。Expert 根读 Skill pins；Team 根读成员 Expert pins，再读该 pinned Expert 版本的 Skill pins/Connector effective。
- 同一请求在同一数据库一致性快照内解析完整 pin 图；同一源版本 + 同一组 pin 的图稳定。只有源资源重新发布才改变 pin 图。
- 任一 pin 不可用即整图不可用；不裁剪、不跳文件、不删边、不补 latest、不换用更新的 pin。
- 图/文件预算仍按 API §2.4/§6.1：1500 节点、2000 边、16MiB 等，按 pin 后的资源+版本计数。
- A01 的 SQL 谓词按源快照 pin 校验可用性，与 resolver 同一语义核心（同 owner、同可用定义、同 Team 一跳展开），必须有等价测试；不得手写 latest，也不得在 Service/前端事后过滤。
- refs/版本索引缺失、`relations_json` 非数组、漏/多 refs、等数量错目标等属完整性错误，公共过滤必须 fail closed（不进入结果/500 `relation_index_inconsistent`），不得当成零依赖或 available。

### 1.3 审核

- 默认自动审核：沿用 `plugin_review_policies`（无 Space 配置时 `IsAutoApproveEnabled=true`）；Space owner/admin 可关闭。Admin 写入直接生效，不进入 pending review。
- 提交/写入冻结 `resolved_dependency_observations`（Expert/Team 的 pins，Team 含成员 Expert pins 及每个 pinned Expert 版本的 Skill pins）与 `connector_observations`（`effective_revision`/`effective_digest`）。观察值是本次提交的 pin 图快照，不是 latest 解析结果。
- 人工审核批准时只校验：每个 pin 的 target version 仍存在且正式 release；目标仍可读、未删除、未 embedded、已上架、受众通过；Connector effective 未变化。
- **目标后来发布新版本不构成 `dependency_changed`**；只有 pinned 版本失效/目标不可用或 Connector effective 变化才 409。
- 自动模式：提交→冻结 pin→批准；并发完整性/提交竞争可重新读取同一 pin 图并重试一次，正常不把 409 暴露给用户。
- 人工模式（Space 显式关闭自动审核）：批准时 pin/effective 校验失败返回 409 `dependency_changed`；由有权取消者取消旧申请，再由有权提交者重新提交并生成新的 review_id。重新提交同一源版本沿用已冻结 pin；缺失 pin 在提交时补齐，不刷新已存在 pin。
- 批准不改变 pin；**重新发布源版本才刷新 pin**。Skill/Expert 发新版不需要重新审核引用它的 Expert/Team，引用方也不会自动改用新版。

### 1.4 代价与客户端影响

- 同一个 Expert/Team 版本在 pin 不变时读取/安装得到相同依赖内容；A34 `bundle_digest` 对同一源版本 + 同一组 pin 稳定，可作为该冻结图的校验值。源版本重新发布后 pin 图变化，是新的源版本，不是同一版本漂移。
- 要使用依赖的新版本，source owner 必须编辑并发布新的源版本；依赖发新版本身不影响已发布来源。
- 审核冻结 pin 图；批准后 pin 不变，不会在批准后漂移。
- 安装可复现性：同 key 重试重放首次冻结 plan；**新 key 新安装按当前源发布快照（含其中 pin）**，不按目标最新。
- 本期仅记录/展示 pin 解析结果，不因目标发新版主动发送通知，不新增通知任务或推送协议。
- “pin”是唯一显式规则，不是失败回退：pin 不可用即整体失败/隐藏，不补 latest、不切历史版本、不删边、不降级。

## 2. 裁定原则

1. 保留独立市场资产、完整审核、可验证安装、显式历史迁移；依赖版本在**上架时冻结为 pin**。
2. 不新增生命周期治理、不新增审核角色、不新增 Fleet 协议；沿用现有上下架/删除/投放/审核策略。
3. 技术选择在本文作出；字段、算法、错误包络不再转交产品决定。
4. 没有“失败后补 latest、切 legacy、删边、跳文件、空配置、自动换 pin”的隐藏分支；pin 是唯一显式规则。
5. 生产迁移仍需人工批准计划；CLI 不是绕过授权。

## 3. 冲突与裁定

| ID | 主题 | 最终裁定 |
|---|---|---|
| C01 | private 目标 | **private 来源可引用已发布 system、同 Space 的 space、同 owner 的 private（含 private Expert→private Skill、private Team→private Expert）；space/system 来源不得引用 private；跨 Space 禁止；同 Space 不同 owner 的 private 仍禁止。** Admin 可见不绕过。私有资产作为依赖时不扩大受众；A34/安装只对同 owner 可见。**private 同 owner private→private 合法，不做全面禁止。** |
| C02 | 展示位 | 保留现有非默认场景可见性；default 不能隐藏；候选接口独立于场景列表。 |
| C03 | 生命周期 | 保留现有删除/下架/访问/投放动作。删除保护 = live incoming（active 来源、未删除、不过滤 listing、含草稿）∪ published refs（仅未删除来源的当前 `published_version_id`，按**精确 `target_version_id` pin**）∪ Team 间接路径（已发布 Team 快照的成员 Expert pin → 该 **pinned Expert 版本**的 refs）。**没有保护引用的曾发布资源可删除**；历史发布版本不是保护超集；下架 pinned 目标版本/plugin 不自动解保护，引用方重新发布/删除边后才释放；下架依赖不获得新增安装豁免。移除 `rejectPublishedMutation` 对删除的总禁令。 |
| C04 | Connector 版本 | 不新增业务 Version；采用 `plugin_connector_effective`（有效修订/摘要），编辑与 effective 分离；安装/A34 读 effective；存量 Connector 切换前回填 revision 1 + digest，`effective_digest` 覆盖完整有效行。**Admin 系统 Connector 的既有直接生效写入保留（同步更新 effective，属既有平台操作）；Tenant Connector 编辑必须审核；Expert/Team 审核观察 effective。Connector 不 pin。** |
| C05 | Connector 审核漂移 | 提审冻结全部 Connector 的 effective_revision/digest；批准时变化 409 `dependency_changed`；只改 editing 不误判。**目标 Expert/Skill 发新版不触发该错误。** |
| C06 | Admin 依赖 | Admin Expert/Team 只上传/重传 schema2 ZIP；A24 拒绝这两类 JSON 内容写；A25 强制 Idempotency-Key、CAS、resolved[]；Admin 直接写入时解析并冻结缺失 pin。 |
| C07 | 同名/多版本 | **本期不允许共存；同名不同内容安装冲突失败（409 `same_install_identity_different_content`）**；不改 `SKILL.md` frontmatter、不生成 `mk1` 别名、不改 Fleet/Daemon。 |
| C08 | Admin 发布矩阵 | 保留现有 AdminCreate/AdminUpdate 行为：创建即发布+default；更新保留 listing/owner/Space/visibility，只接版本/pin/effective 底座；不新增矩阵、不新增重新上架动作。 |

## 4. 原缺口与最终处理

| ID | 主题 | 最终处理 |
|---|---|---|
| A01 | 保存/发布双指针 | 保留 current/published/release；读默认 published，编辑显式 editing；A01 增加 `dependency_availability`（P 语义，按 pin 判定）。 |
| A02 | 删除保护 | 按 C03 并集；不新增总禁令；published refs 使用精确 pin，Team 用 pinned 成员 Expert。 |
| A03 | 图版本身份 | 图节点按 pin 的 `plugin_id@version_id`（Connector `plugin_id@effective_revision`）；Expert 根用其 Skill pins，Team 根用成员 Expert pins + pinned Expert 版本的 Skill pins；A03 轻量图，A34 完整内容。 |
| A04 | 安装引擎 | 一个 InstallCoordinator；A31/A32 只做 adapter；删除市场 legacy create/bind/delete 补偿；独立 `/experts`、`/squads` 保留共享代码。 |
| A05 | 安装重试 | 首次冻结无凭据 plan/请求 + HMAC；同 key 只重放原 plan；同 key 显式输入变化返回 409；上游目标发新版仍重放原 plan；**新 key 按当前源发布快照（含 pin）**；状态机/UNKNOWN；仅首次成功计数。 |
| A06 | 迁移 | CLI 生产迁移（D5）：两次 preview/冻结计划、显式标签/映射、可恢复 run、数据安全回滚；存量已发布 Expert/Team 在 cutover 做 pin 回填；不做 A29/A30 HTTP。 |
| A07 | 重新上架 | 撤回；不新增按钮/接口。 |
| A08 | 历史 | 旧快照/字节/记录保留；P 后发布快照按自身 pin 读取；回填前的历史发布快照由 cutover 回填补齐 pin；不伪造 pin/release；历史读取仍按 release/受众。 |
| A09 | restore-to-draft | **删除，不做。** 历史读取 A04–A08 保留。 |

## 5. 生产迁移（CLI）

- 生产路径：`marketplace-stock-migration` CLI + 离线授权（命令名/子命令以实现时确认为准）；不新增 A29/A30 HTTP。
- **存量 pin 回填（方案 A）：** 生产切换时，迁移 CLI 对现有已发布 Expert/Team 执行一次 pin 回填：
  - 已有合法 `target_version_id` 且该版本仍是正式 release → 保留；
  - 没有 pin/无效 → 冻结到迁移时目标当前正式发布版；
  - 找不到可冻结版本 → 该根标记 `blocked`，不静默取最新/旧版，不懒回填。

  回填在离线授权/冻结计划下执行，产出 `plan_digest`；不得在普通读取时懒回填。回填后源资源不因回填自动重新审核；下一次用户重新发布时按 P1/P2/P4 重新冻结。回填 CLI 仍需 preview/apply/resume、数据安全回滚、去指针回滚；system versioned/Space/Tenant Connector 的 blocked 状态保留到对应路径补齐。
- CLI 契约：apply/resume 不使用 HTTP 状态码；`plan_stale`/`authorization_invalid` 以 CLI reason + 非零退出表达，已有 run 保持 `blocked`。不定义幂等键参数；run 由 `(source_scope, source_identity, plan_digest, root)` 仲裁，同一冻结计划重跑恢复原 run。
- 存储：冻结 plan（或其不可变对象 key）、run 状态、map、authorization→batch/run 绑定均需增量 DDL 与明确 owner，`resume --run-id` 才能执行；字段细节见 01 §3.1/§8.2。
- 两次只读 preview：
  1. discovery：来源 inventory、候选、关系、缺失选择；
  2. 冻结：显式标签/映射/Connector server 分组/归属，写 `plan_digest`；apply 只认冻结计划，不接受手改摘要。
- 来源身份：数据库 `plugin_id@version_id`；ZIP 原字节 SHA-256（若 CLI 支持 ZIP，必须显式；本期可只支持数据库来源，ZIP 另行授权）。`source_digest` 统一映射 02 的 `input_digest`，不定义第二套摘要。
- create/reuse/update_source：root 用 `update_source` 保持 ID；子节点 create/reuse；不做自动目标/最新版本猜测。迁移生成的新 Expert/Team 在发布时按 P 冻结 pin；复用目标若在计划中显式给 `target_version_id`，apply 必须校验其仍为正式 release，否则 `plan_stale`/`blocked`。
- 标签：preview 可建议 `patch+1`，但必须写入冻结计划并由人批准；apply 不自行换标签；冲突重新 preview。
- Connector：默认保留原 server 集合；只有冻结计划显式拆分才拆，server key 不重复/不丢。system 来源按现有 Admin 直接生效并同步 effective；Tenant 来源走审核，批准后才产生 effective；run 在人工审核期间 `waiting_review`。reuse 的 `target_effective_revision/digest` 是批准观察值，apply 时漂移返回 `plan_stale`，不写成关系 pin。
- 审核：CLI 生成的新版本走现有发布/审核（默认自动可自动批准；人工则 waiting_review）；CLI run 状态 `completed|waiting_review|blocked`；不直接插 release 绕过审核。
- 旧入口与门禁：删除 `marketplace-stock-migration` 的 `--rollback` 指针回滚、直接 `INSERT plugin_version_releases`、`--g4...` 自证批准和固定 pin 旧逻辑；旧无授权 `--all-spaces --apply` 模式与 `plugin-backfill` 不得保留可写旁路。部署门禁包含撤销旧 CLI/旧模式的 DB 账号/DSN/权限、移除或 deny 旧 `plugin-backfill` 二进制，并由 REL-01 验证旧入口不能创建/切换/写 release，不只网关/网络。
- 回滚（数据安全）：
  - 不删表/不删文件/不覆盖旧快照；
  - 不把 `current_version_id/published_version_id` 改回旧值；
  - 需要停止新分发时用下架/部署隔离，不用指针回切；
  - 内容恢复从旧历史生成新版本（forward-fix）；
  - 不调用/恢复旧 `--rollback` 指针回滚模式；
  - 保留 batch/mapping/attempt/audit。
- 独立可见：迁移后的 Skill/Expert 必须在对应市场独立可发现；只有全部节点发布/列表资格就绪才 completed。

## 6. 同名/多版本

- 本期 **拒绝共存**：Expert/Team 同一依赖名不同内容时安装失败；不生成稳定别名、不改写 `SKILL.md`、不改目录名。
- 普通无冲突安装使用市场原名/原字节。
- 若未来要做，需另立决策并补真实 Fleet/Daemon 加载证据；不得以 `mk1` 等未批准算法上线。

## 7. A34 完整内容读取（P 语义）

- `GET /plugins/{plugin_id}/distribution`，只 path ID + 既有认证/Space；无 workspace/runtime/custom_env/幂等键/根版本选择。
- 每次按 **pin 图**读取完整内容：Expert/Team 自身当前发布版本 + 其快照中的依赖 pins；Team 的成员 Expert 用 Team 快照 pin 的版本，再读该 Expert 版本的 Skill pins + 有效 Connector。不解析目标最新版，不回退其他版本。
- 返回 `DistributionBundle{schema_version,root,graph,assets,bundle_digest}`；每个 node_key 恰一 asset；文件原字节/hash；Connector `files:[]` + 有效公开配置/占位符。
- `bundle_digest` 对**同一源版本 + 同一组 pin** 稳定；源版本重新发布（pin 图变化）才是新输入。服务端不得跨请求缓存；`no-store`；零 Fleet/零写入。
- 名称取 pinned released snapshot/effective 内容，不从 current/editing 名称填充。
- 16MiB 完整 JSON、图/文件预算；缺节点/文件/权限/完整性整体失败；不裁剪/分页/补 latest。错误矩阵固定：路径/类型/二进制 400，根或目标不可见 404，pin 依赖未发布/下架/删除/状态不合法 409 `dependency_unavailable`/`dependency_changed`，预算 413，文件/索引完整性 500；不新增顶层错误码。
- 不生成 `installation_name`、不改写 frontmatter、不产安装副本；读取成功不代表 Fleet 安装/共存成功。
- CL-02/CL-03 完整内容只消费 A34；A03 只做轻量图展示。

## 8. A01 公共过滤与 mine 诊断（P 语义）

- 公共 A01：Expert/Team 的**pin 图**任一依赖不可用 → 来源不进入公共列表；不修改来源 `listing_state`。
- “可用”= 每个 pin 的 target version 存在、已正式 release、目标 plugin `status=1`/未删除/未 embedded/`listing_state=published`/installAccess+referenceAudience 通过；Connector 还需 effective 行。**不按目标当前 `published_version_id` 判断**；pin 的旧版本只要仍 released 且目标可读，就可用，即使目标后来发了新版。
- SQL 在 COUNT/排序/分页前按 pin 过滤；分类 `plugin_count`、公共 tags 同口径；一次一致性快照。
- mine/Admin 保留原管理范围全部非删除资产与 total；不新增回收站。
- 仅 mine 的 Expert/Team 必返：
  `dependency_availability={status,evaluated_view,evaluated_version_id,reasons}`；
  `evaluated_view/version` 指来源自己的 published/editing（有 published 诊断 published，未发布诊断 current/editing）；
  `reasons[].code` 固定 `dependency_delisted|dependency_deleted|dependency_unavailable`；
  `target.version_id` 是 **pin**，不是目标最新版；Team 的 `via` 是 Team pin 的成员 Expert（含 pin 版本）。
- **从未发布草稿的 pin 可为空：** mine 对空 pin 关系返回 `unavailable` + `dependency_unavailable`（不带 `target.version_id`，可披露的 target ID/名称仍可给），提示“依赖尚未冻结，发布时校验”；不得用目标当前正式版冒充 pin。已发布快照缺 pin 属完整性错误（500 `relation_index_inconsistent`），不得降级成普通 unavailable。
- 目标及整条 via 逐项有权才给具体身份；否则只把不可披露部分合并为一条匿名 `dependency_unavailable`，不带 ID/名称/类型/版本/数量；其他可披露原因仍可分别列出。
- 名称取 pin 的 released snapshot/effective 内容；不授权/无 released 内容则匿名或空名，不得使用其他 owner 的 editing name。
- 源版本/refs 自相矛盾（版本缺失、非数组、漏/多 refs、等数量错目标）为 500 `relation_index_inconsistent`；refs 一致但目标 Plugin 缺失才是正常 `dependency_unavailable`，不得吞成零依赖/available。缺失 refs 索引 fail closed。
- 四字段 DTO 的 Go 类型/序列化槽位由 MP-01 定义，但 MP-01 不填 `status/reasons/target/via`、不得伪造 available；MP-02 负责 pin 可用性 predicate、诊断计算与 reason 授权。
- `view`：mine 仅 editing，公共仅 published，Connector 可 effective；非法组合 400 `details.field=view`。
- 只读，不改 listing/审核/投放；pin 图只在来源重新发布后变化；不新增推送/轮询/状态列。

## 9. 发布/审核/安装/数据

- 发布审核职责、owner 提交、Space 自动审核默认、人工审核入口沿用；自动/人工都执行同一 pin 图校验。
- 保存/发布：Skill/Expert/Team 内容、关系集合、标签/受众变化写新版本；`edit_revision` CAS；`relations` 显式数组（仅 A12 完整 Write；Skill import 不接 relations，A25 从 ZIP 清单取关系）。
- **发布/提交冻结 pin**：缺失 pin 解析为目标当时当前正式发布版；显式 pin 校验归属/release/受众；显式指向非正式版本 → 400；无可用目标版本 → 409 `dependency_unavailable`。发布事务写入 pin 并切 published；已 release 版本不再改 pin。
- **审核**：提交冻结 pins + Connector effective；批准只校验 pinned 版本/effective；目标发新版不 409；pin 失效/目标不可用/effective 漂移 409 `dependency_changed`；批准不改变 pin。
- Connector：编辑/审核/effective 分离；不新增业务 Version；存量 Connector 回填 revision 1 + digest，`effective_digest` 覆盖完整有效行；Connector 关系 `target_version_id` 必须 NULL。
- Admin 读取：DTO 省略 `published_version_id`；A21 默认 editing/current（仅 Connector 可显式 effective）；A22 精确 `version_id`；A23 默认 current 并标实际版本；A26 重传 `expected_edit_revision`。Admin 写入时解析并冻结缺失 pin。
- 安装：A31/A32 一个引擎；同 key 原 plan；**新 key 按当前源发布快照（含 pin）**；Secret 只在内存；仅首次成功计数；状态机含 `EXPIRED`，`DISPATCHING` 租约到期 → `UNKNOWN`；同名冲突 `same_install_identity_different_content`。
- 数据：`plugin_version_releases`、`plugins.published_version_id`、`plugin_relations`（live 编辑关系可空，非权威）、`plugin_version_relation_refs`（**已发布快照的 Expert/Team 关系 pin 必填；草稿可空；Connector 必为 NULL**）、`plugin_connector_effective`、`plugin_install_attempts`、`plugin_mutation_receipts`、migration batch/run/map/authorization binding。
- 历史：published 快照不可变；pin 是发布快照的一部分，不是读取时解析值；A34/安装对同一源版本 + 同一组 pin 返回稳定图。

## 10. 不变核心与非目标

不变：独立资产、完整审核、跨 Space 隔离、不保存运行时 Secret、Marketplace 展开/Fleet 原子安装、历史字节保留、生产迁移另行授权。

非目标：同名共存、A09 恢复、A29/A30 HTTP、新生命周期状态、新审核角色、Fleet/Daemon 改造、自动执行生产迁移、读取时 latest 解析。

## 11. 验收新增重点

- **P pin 冻结：** S 发布 v2 后，已发布 E 仍按自身快照 pin 的 S v1 读取；E/Team 公共列表/A34/安装不变，不需要发新版本；只有 E 重新发布新源版本才刷新 pin。Team pin 成员 E 版本，E 之后发新版不影响 Team。
- **不回退/不追新：** pin 版本下架/删除/失去 release 时依赖不可用，公共隐藏、A34/安装整体失败；不静默用更新版或旧版、不自动换 pin。
- **重新发布刷新：** 发布新源版本时缺失 pin 解析为发布时目标当前正式发布版，显式 pin 校验后保留；pin 图变化导致 A34 `bundle_digest` 变化。
- **审核：** 自动模式无人工步骤；人工模式提交冻结 pin 观察 + effective；**目标发新版不 409**；只有 pin 失效/目标不可用/effective 漂移 409；批准后 pin 不变；Admin 无 pending review。
- **D4：** mine 的 `target.version_id` 是 pin、名称取 pinned released snapshot、只匿名不可披露部分、view 非法 400；MP-01 只定义 DTO 槽位，MP-02 计算 status/reasons/target/via。
- **删除：** live 草稿引用保护；未删除来源当前 published refs 按精确 pin 保护；Team 按 pinned 成员 Expert 版本保护；只要 Team 仍 pin 旧 Expert 版本，E 发新版本移除 S 也不释放该 Team 保护；Team 重新发布/删边后才释放；无“曾发布总禁令”，移除 `rejectPublishedMutation`。
- **refs/Connector/CHECK：** 已发布 Expert/Team 关系的 `target_version_id` 必填；草稿可空；Connector 必须 NULL；CHECK 只强制 Connector NULL，发布 validator 强制已发布非空；`dependency_version_required` 不恢复；存量 Connector 回填 revision 1 + digest。
- **A34：** 单 GET pin 图；`bundle_digest` 对同一源版本 + 同一组 pin 稳定；no-store 且不跨请求缓存；400/404/409/413/500 失败矩阵见 §7。
- **安装：** 同 key 原 plan；同 key 显式输入变化 409；上游目标发新版仍重放原 plan；新 key 当前源发布快照；仅首次计数；`intent_digest`/`input_hmac` 不含 pin 版本；A31/A32 单 `X-Workspace-Id`，A32 header=body。
- **迁移：** 两次 preview/冻结计划；存量 pin 回填（方案 A）；CLI `plan_stale`/`authorization_invalid` + 非零退出/blocked，不用 HTTP 409；无幂等键参数；数据安全回滚；不指针回切；审核/独立可见。
- **#128 清理：** 删除 `pinnedInstallation`/`installPinnedLegacy`/`SkillInstallAdapterVersion`/`PreparePinnedSkills`/`adapter_version`/`transform_variant`；A31/A32 在 flag on/off 都只进 `InstallCoordinator`；迁移 CLI 删除 `--rollback`/直接 release/`--g4...`；`scripts/pinned-release-readiness.sql` 重写为 **pin readiness/结构检查**，不按 latest 解析。
- **D6：** private 来源可引用已发布 system、同 Space space、同 owner private（含 private Expert→private Skill）；space/system 不得引用 private；跨 Space 禁止；同 Space 不同 owner private 仍禁止；private 同 owner private→private 合法。
- **D8：** relations 必填仅 A12 完整 Write；Skill import 不接；A25 从 ZIP。
- **PR #139：** 当前实现是 B 模型（latest resolver/A01 latest SQL/A15 批准重解析/A34 动态依赖/安装 latest、`target_version_id` 仅审计可空），与 P 冲突；必须按 `decisions-20261026-pin.md` §4 重做，不能以“已实现”评审通过。

## 评审修订裁定（2026-10-26，P 语义）

- **人工审核恢复：** 批准只校验 pin 版本/目标可用性/Connector effective；目标发新版不 409。冲突时旧申请与 pin 观察值保持不变；使用原取消/提交权限取消旧申请并重新提交，生成新 review_id、新 pin 观察值；版本化资源可重提同一不可变来源版本，沿用其已冻结 pin；Connector 按当前编辑修订重新冻结并重新审阅，不把旧冻结内容替换为当前内容。无新增刷新基线接口，不自动取消或代提。
- **删除范围：** published refs 仅从未删除来源的当前 `published_version_id` 进入，按精确 pin 判定；Team 间接链仅一跳：已发布 Team 快照 → pinned 成员 Expert 版本 → 该版本的 refs；不展开第二个 Team，不扫描历史发布版。live 草稿引用仍保护；下架 pin 目标不自动解保护。
- **一致性与重试：** 每次请求内部一致；同一源版本 + 同一组 pin 的图稳定。原 key 重放原计划；目标正常发新版不触发冲突；pin 失效/显式输入变化/权限失效分别校验。新 key 按当前源发布快照构建新计划。
- **Admin 与通知：** Admin 不用审核，沿用直接生效动作和权限，仍做 CAS、pin 解析/校验及范围校验。依赖更新仅审计展示，不主动通知。
- **迁移执行授权：** 采用 API §5 的签名授权制品，绑定计划、操作者、范围和批次；信任配置、有效期、撤销及同 run 恢复均须验证。存量 pin 回填属于生产切换计划的一部分，须离线授权。此不表示用户已授权实际生产迁移。
- **契约边界（2026-10-26 评审，P 修订）：** A01 DTO 的 MP-01/MP-02 分工、refs CHECK/validator、发布/提交 pin 冻结、A34 失败矩阵与稳定 digest、CLI `plan_stale`/`authorization_invalid` 契约、pin readiness SQL 及 #128 旧执行器清理，均以 01 的对应章节为实现口径；这些是工程契约补齐，不改变 P/D1–D10 的产品决策。
