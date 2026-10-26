# 决策修订：P 模型（上架冻结依赖，2026-10-26）

状态：**取代 B 模型**。用户在 PR #139 评审后明确要求：Expert 的技能版本、Team 的成员 Expert 版本都按**上架时的版本号**冻结，不再跟市场当前最新版本走；private 保持原有语义，允许 private Expert 引用 private Skill。

本文是新的权威方案。`00-decisions.md`、`01-technical-design.md`、`02-api-contracts.md` 及 issue 中与本文冲突的 B/latest 段落均需按本文修订。历史评审记录可保留，但必须标注“B 模型历史记录，已被 P 模型取代”。

## 1. 定义

### P1 Expert→Skill

- Expert 发布/上架时，服务端为每条 `expert_skill` 关系写入**当时 Skill 的当前正式发布版本** `target_version_id`。
- 已发布 Expert 快照中的 `target_version_id` 是权威 pin；公共读取、A01、A03、A34、安装、审核观察都读这个 pin。
- Skill 之后发新版、下架新版、删除新版，都不改变已发布 Expert 的 pin；Expert 仍读旧版本。
- Skill 的 pinned 版本被删除/下架/失去正式 release → Expert 的依赖不可用；**不回退到更新版本，也不自动切换**。Expert owner 必须编辑并重新发布，重新冻结到当时的当前正式发布版本。
- Expert 重新发布时，所有 `expert_skill` 关系重新冻结到发布时的当前正式发布版本（即使 target plugin id 没变）。

### P2 Team→Expert

- Team 发布/上架时，为每条 `expert_team_expert` 关系写入**当时成员 Expert 的当前正式发布版本** `target_version_id`。
- 已发布 Team 快照中的 pin 是权威；公共读取、A34、安装都读该 Expert 版本，不再取 Expert 最新版。
- 该 Expert 版本内部的 Skill pin 沿用 Expert 自己发布时的快照，不因 Team 读取而二次解析最新。
- Team 重新发布时，所有成员 Expert 关系重新冻结到发布时的当前正式发布版本。

### P3 Connector

- 本次不改 Connector：仍按 **plugin id + `plugin_connector_effective`** 读取当前 effective 修订；Connector 没有业务版本 pin。
- Expert/Team 发布时记录 Connector 的 effective revision/digest 观察值；审核/安装按 effective 校验。若后续明确要求 Connector 也冻结，再单独立项。

### P4 发布快照

- 权威 pin 存在**源资源已发布版本的 relations 快照**里，不是只存在 live relation。
- `target_version_id` 从“审计值/可空”改为“已发布快照必填、权威”；草稿/提交前可空，发布动作负责解析并写入。
- A12 完整 Write 允许客户端显式给 `target_version_id`：必须是目标的正式 release 且归属正确；不给时由发布动作解析当前正式发布版本。
- A25 schema2 ZIP 的 `version_id` 同理：可显式给，也可省略；Admin 直接生效路径在写入时解析并冻结。
- Connector 关系仍必须 NULL。

### P5 读取与图

- 根选择不变：公共/A34 用根 `published_version_id`；A01 mine 用 `editing`；A03 用显式根 `version_id`。
- 依赖边不再调用 latest resolver；直接读取源快照的 `target_version_id`。
- Expert 根：读取 Expert 快照的 Skill pins。
- Team 根：读取 Team 快照的成员 Expert pins；每个成员 Expert 读取**该 pinned Expert 版本**的 Skill pins。
- Team 的 `member_key/role/is_leader/sort_order` 仍由 Team 边决定；成员 Expert 的 pin 不改变这些字段。
- 同一源版本、同一组 pin 的图是稳定的；A34 `bundle_digest` 不再因目标发新版而变化。
- 只有源资源重新发布才会改变 pin 图；新 key 安装也按当前源发布快照，不按目标最新。

### P6 A01 公共过滤与 mine 诊断

- 公共列表/总数/category/tag facet：只保留根已发布且其 pinned 依赖图可用的 Expert/Team。
- “可用”= 每个 pin 的 target version 存在、已正式 release、目标 plugin `status=1`/未删除/未 embedded/`listing_state=published`/installAccess+referenceAudience 通过；Connector 还需 effective 行。
- 不再按目标当前 `published_version_id` 判断；pin 的旧版本只要仍 released 且目标可读，就可用，即使目标后来发了新版。
- mine 四字段保留：`status/evaluated_view/evaluated_version_id/reasons`；`target.version_id` 是 pin；原因码仍 `dependency_delisted/deleted/unavailable`。
- 不披露其他 owner 的 editing name；名称取 pin 的 released snapshot/effective 内容；无授权则匿名/空名。
- A01 的 SQL 谓词改为“按源快照 pin 校验”，不再需要 latest 解析等价测试；需要列 pin 版本可用性的 MySQL 测试。

### P7 A15 审核

- 提交时冻结 `resolved_dependency_observations`：Expert/Team 的 pins、Connector 的 effective revision/digest。
- 人工审核批准时只校验：
  - pin 的 target version 仍存在且正式 release；
  - 目标仍可读/未删除/未 embedded/已上架；
  - Connector effective 未变化。
- 目标后来发布**新版本**不构成 `dependency_changed`；只有 pin 的版本失效/目标不可用/Connector effective 变化才 409。
- 自动审核仍可重试一次；不再“批准时重新解析 latest”。
- 批准后 pin 不变；源资源重新发布才会改变依赖。

### P8 删除保护

- 保护来源 = live incoming（含草稿）∪ 未删除来源当前 `published_version_id` 的 published refs。
- Expert/Team 的 published refs 现在是精确 `target_version_id` pin；删除目标 plugin 或 pin 的版本时必须被保护。
- Team 间接保护：Team → pinned 成员 Expert → 该 Expert pinned version 的 refs；不再取成员最新版。
- 没有保护引用的曾发布资源仍可删除（D1 保留）。
- 下架 pinned 目标版本/plugin 不自动解保护；引用方重新发布/删除边后才释放。

### P9 private 矩阵

保留原有语义，不做“private 一律不可被引用”：

- private Expert 可以引用**同 owner 的 private Skill**；
- private Team 可以引用**同 owner 的 private Expert**；
- private 来源还可引用 system、同 Space 的 space、同 owner 的 private；
- space/system 来源不得引用 private；跨 Space 禁止；
- Admin 可见性不绕过；A34/安装只对同 owner 可见 private 图。
- 同 owner 判定使用 Space + owner uid；private Expert 引用同 owner private Skill 是合法主路径，必须有回归。

## 2. 存量迁移

用户在本次修订中未单独指定存量策略；按以下默认执行，若产品改口再调整：

- 生产切换时，迁移 CLI 对现有已发布 Expert/Team 执行一次**pin 回填**：
  - 已有合法 `target_version_id` 且该版本仍是正式 release → 保留；
  - 没有 pin/无效 → 冻结到迁移时目标当前正式发布版本；
  - 找不到可冻结版本 → 该根标记 blocked，不静默取最新/旧版。
- 回填在离线授权/冻结计划下执行，产出 `plan_digest`；不得在普通读取时懒回填。
- 回填后，源资源不因回填自动重新审核；下一次用户重新发布时按 P4/P1/P2 重新冻结。
- 迁移 CLI 仍需支持 preview/apply/resume、数据安全回滚、去指针回滚；system versioned/Space/Tenant Connector 的 blocked 状态保留到对应路径补齐。

## 3. API/错误变化

- 已发布 Expert/Team relations 的 `target_version_id` 必填；草稿可空；Connector 必须 NULL。
- 发布时缺失 pin 由服务端解析；显式 pin 必须通过归属/release/audience 校验；显式指向非正式版本 → 400。
- 读取/A34/安装只用 pin；pin 不可用 → 404/409 依赖错误，不回退最新。
- A15 新版本不再 409；pin 失效/Connector effective 变化才 409 `dependency_changed`。
- A01 mine `dependency_availability.target.version_id` = pin；不再写 latest。
- A34 `bundle_digest` 对同一源版本稳定；不同源版本可不同。
- 安装 plan 的 `resolved`/冻结根/依赖版本都用 pin；同 key 重放原 plan，新 key 仍按当前源发布快照。

## 4. 对 PR #139 的影响

当前 PR #139 的实现是 B 模型：

- latest resolver / A01 latest SQL / A15 批准重解析 / A34 动态依赖 / 安装 latest；
- `target_version_id` 仅审计、可空；
- 这些都与本修订冲突，**不能按现状评审/合并**。

需要重做/替换的后端切片：

1. `internal/repository/plugin/latest_graph.go` → pinned resolver（按源快照 pin）。
2. A01 `dependency_predicate.go` / `dependency_availability.go` → pin 可用性。
3. A15 review observations → pin 观察；批准只校验 pin/effective。
4. A34/install → 按 pin 图读取，不再 latest。
5. A12/A25 发布路径 → 发布时冻结缺失 pin。
6. A25 `resolved[]` → 记录本次写入的 pin（不是 latest 解析）。
7. 删除保护/索引 → 精确 pin。
8. 存量 pin 回填 CLI。
9. 测试：pin 稳定性、目标发新版不影响、pin 失效隐藏、private 同 owner 引用、Team pin Expert、Expert pin Skill、A15 不发新版 409、A34 digest 稳定。

## 5. 待确认的默认

以下按推荐默认写入方案；若产品不认可，改动只影响对应段落：

1. 存量：迁移时冻结当前正式发布版本（方案 A）。
2. private：严格同 owner 的 private→private；同 Space 不同 owner 仍禁止。
3. 重新发布：刷新所有关系 pin 到发布时当前正式发布版本。
4. Connector：不 pin，继续 effective。
