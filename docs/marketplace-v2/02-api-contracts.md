# 跨服务 API 权威契约（P 模型）

> **P 模型取代 B：** 本文已按《[决策修订：P 模型](decisions-20261026-pin.md)》改写。Expert→Skill、Team→Expert 以上架时冻结在源快照中的 `target_version_id` **pin** 为权威；公共读取/A01/A03/A34/安装/审核观察只读 pin，不再解析目标最新版；Connector 保持 ID + effective。历史段落中的 B/latest 字样仅作历史对照，均以 P 为准。**当前 PR #139 是 B 模型实现，必须按 P 重做，不能按现状评审或合并。**

状态：**2026-10-26 最终决策已锁定，引用模型为 P（上架冻结依赖）。** Expert→Skill、Team→Expert 在发布/提交时把目标当前正式发布版冻结为源快照中的 `target_version_id` pin；此后公共读取、A03、A34、安装、审核观察只读 pin，目标发新版不改变已发布来源。草稿 pin 可为 NULL；发布/提交解析缺失 pin、校验显式 pin；Connector 关系必须 NULL，仍按 ID 读 `plugin_connector_effective`。A09 restore-to-draft、A29/A30 HTTP 迁移、同名/多版本共存本期均不做；生产迁移改 CLI + 离线授权（§5），并对存量已发布 Expert/Team 做 cutover pin 回填（方案 A）。A34 一次返回 pin 图及全部原始内容，客户端不得按依赖补请求；`bundle_digest` 对同一源版本 + 同一组 pin 稳定。Admin ZIP、Fleet 既有重试、system/Space/范围、原审核和市场动作保持。不得遇错降级、不得回退/切换到其他版本、不得把同 key 重试静默改成新源发布快照。模型见[技术方案](01-technical-design.md)与[最终决策](00-decisions.md)。

## 1. 通用规则

### 1.1 路径、身份和包络

- 下表 Marketplace 路径为相对服务前缀；用户实际 URL 为 `/market/api/v1` + 路径，服务现有注册为 `/api/v1` + 路径。保留既有网关，不新增路径探测。
- Fleet 对外文档路径为 `/v1` + 路径；网关为 `/fleet/api/v1` + 路径；内部 client 按配置选择唯一 base path。swag Router 写相对路径，不重复 `/v1`。
- Tenant：有效用户/Bot token（`Token`或Bearer，不能两者矛盾）、`X-Space-Id`；owner/Space从权威身份得出，不信body。User/Bot的现有发布、提交与管理权限不变，owner提交及既有审核者校验继续有效；安装及同键重试仅允许用户身份。
- Admin：本人session、既有superAdmin/marketAdmin鉴权及跨Space管理读不变；不能绕过已确认的system/Space依赖矩阵。不新增生命周期管理权限或运营按钮。
- 安装/Fleet：只允许用户身份；单个 `X-Workspace-Id` UUID，不接受 slug/query 兜底。Marketplace 转发原用户 token/Space，不能使用平台管理员 token 代装。
- 单体成功 `{data:{...}}`，列表 `{data:[...],pagination:{page,page_size,total}}`，空成功 `{data:{}}`；日期 RFC3339 UTC；page 默认1、page_size默认20/最大100。新字段 snake_case，布尔 is_/has_/can_。
- JSON 请求拒绝未知字段、多个根值、重复键和错误类型；所有写入缺少必需字段明确400。**`relations` 显式数组仅对 A12 完整 Write 必需**：缺失或 null 为 400 `relations_required`（只有显式 `[]` 才表示清空）；A12 Skill parse-task import 不接受 `relations`，A25 的关系来自 ZIP schema2 清单，不由 HTTP `relations` 覆盖。HTTP header 大小写不敏感，约定写法不影响现有客户端。
- JSON 错误统一 `{error:{code,message,details,hint}}`，code 只用既有12项。文件下载200为ZIP字节流，是显式包络豁免；失败仍JSON。所有内容/历史文件读取鉴权，不能返回 bucket/object key 或 storage_uri。
- **P pin（唯一权威）：** 已发布 Expert/Team 源快照的每条 `expert_skill`/`expert_team_expert` 关系携带 `target_version_id` pin。公共读取/A01/A03/A34/安装/审核观察只读该 pin：pin 的 target version 必须存在、已正式 release，目标 plugin `status=1`、`deleted_at IS NULL`、未 embedded、`listing_state='published'`、installAccess/referenceAudience 通过；Connector 还需 effective 行。**不读取目标当前 `published_version_id`，不解析目标最新版，不回退到更新或旧版本。** 目标后来发新版不影响已发布来源；来源重新发布新版本才刷新 pin。**Team 的成员 Expert 使用 Team 快照 pin 的 Expert 版本；该版本再读它自己快照中的 Skill pins，不因 Team 读取二次解析。**
- **冻结规则：** 草稿 `target_version_id` 可为 NULL；发布/提交（A13/A14/A25 Admin 写入）把缺失 pin 解析为目标**当时的当前正式发布版**并写入快照；显式 pin 必须属于目标且已 release，否则 400 `dependency_version_mismatch`；显式指向非正式版本 400；无可用目标 409 `dependency_unavailable`。发布后 pin 不变；已 release 的同一版本重复发布幂等，不刷新 pin。Connector 必须完全省略（NULL）。
- **唯一解析入口与根选择：** A01、A03、A34、安装准备、审核观察复用同一个 **pin 解析服务**，不能各写一套。解析器输入为 `{root_plugin_id, root: published | explicit version_id | editing}`：A01 公共/A34 的根取来源当前 `published_version_id`；A03 的显式 `version_id` 表示历史根；A13/A14 的根取请求提交的 `version_id`（Connector 为提交时 `edit_revision`）；编辑投影取 `editing`。**显式版本根只读取该 `version_id`，绝不回读 current；依赖边只读该根快照的 pin，不再解析 latest。** A01 公共过滤的 SQL 谓词是上述 resolver 的同一语义核心（同一资格/状态/owner 范围与同一 pin 规则），仅为在 COUNT/排序/分页前过滤而作为共享 SQL 实现的特例（explicit carve-out），不得另写一套 latest；SQL 谓词与 resolver 必须有等价测试（含 Team pinned 成员链、Connector effective、缺失/非法 refs），其他手写 latest 查询视为契约违例。
- **一致性：** 一次请求在同一个数据库一致性快照内读取完整 **pin 图**；同一源版本 + 同一组 pin 的图稳定，只有源资源重新发布才改变。不同源版本/不同 pin 的两次请求可不同，但不是同版本漂移。
- **P 的稳定性与代价（必须写进客户端行为）：** 同一 Expert/Team 版本在 pin 不变时读取/安装得到相同依赖内容；A34 `bundle_digest` 对同一源版本 + 同一组 pin 稳定；目标发新版不影响引用方；要采用新版必须由 source owner 发布新源版本；审核冻结提交时的 pin 图，批准后不漂移；安装可复现性为“同 key 重试重放首次冻结 plan”，新 key 按当前源发布快照（含 pin），不按目标 latest。
- **target_version_id：** Expert→Skill、Team→Expert 的**权威 pin**；草稿可省略/为 null，已发布快照必填；显式提供时必须属于目标且已 release，否则 400 `dependency_version_mismatch`；发布/提交时缺失 pin 由服务端解析，不由客户端 latest 推导。Connector 必须完全省略，null/空字符串 400 `connector_version_forbidden`。source/target `plugin_id` 是关系身份，`target_version_id` 是冻结的目标版本。
- 市场与运行态分开：市场关系使用目标的`plugin_id`（Expert/Team 关系带权威 `target_version_id` pin；Connector 无版本），不接受本地路径或Fleet skill_id。纯本地组合不调用Marketplace，工作空间本地Skill直接推Fleet；不创建private市场中间层。system仅引用已发布system；space仅引用已发布system/同Space的space；private来源可引用已发布system、同Space的space及**同owner private**（private Expert→private Skill、private Team→private Expert 合法）；**space/system 来源不得引用 private，跨 Space 禁止；同 Space 不同 owner 的 private 仍禁止。private 同 owner private→private 合法。** Team递归按 pinned 成员 Expert 同样校验。private 目标不扩大受众，A01 公共列表、A34 与安装只对同 owner 可见。访问与上下架资格沿用现有检查，不新增豁免。

### 1.2 本方案的 breaking changes

| 调整 | 原因 | 必须同步的调用者 |
|---|---|---|
| Expert→Skill、Team→Expert 的 `target_version_id` 改为**权威 pin**：草稿可空、发布/提交冻结、已发布必填；Connector 仍禁止该字段 | P：上架冻结依赖，读取不解析 latest | Admin/Web/CLI及Bot工具，Buddy只读 |
| 删除 A09 restore-to-draft（端点、DTO、客户端动作） | D2：不做历史还原为草稿 | 全部 Admin/Web/CLI/Bot 客户端 |
| 撤回 A29/A30 HTTP，生产迁移改为 CLI 两次 preview/冻结计划（§5） | D5：CLI + 离线授权，不做 HTTP 迁移 | Admin/CLI/运维；CL-02 不接 A29/A30 |
| 更新必须 `expected_edit_revision`；**仅 A12 完整 Write** 的 relations 必须显式数组（A24 Skill/Connector 不要求） | D8：防并发覆盖；Skill parse-task import 使用其独立请求、服务端生成空关系 | Admin/Web/CLI及Bot工具 |
| 市场默认 published，编辑显式 `view=editing`；mine 仅 editing，公共仅 published | 草稿隔离 | 所有客户端 |
| A32 保留 URL 但接入同一安装引擎，必须 Idempotency-Key、用户身份、Workspace UUID | D9：同一原子安装边界，不保留 legacy 行为 | Web/CLI；Buddy已有幂等入口 |
| detail_graph 改为按根快照 pin 的 nodes/edges | P：节点版本是 pin，不是目标 latest | 当前未发现四端消费；未来调用者按新schema，不据此新增本地版本注册表 |
| A34 单次返回 pin 图的完整专家/团队内容；`bundle_digest` 对同一源版本 + 同一组 pin 稳定 | P：一次读取、消除客户端扇出与混版 | Buddy；Web完整内容界面；CLI完整内容查询 |
| Connector 无业务版本；审核观察 effective 修订/摘要 | D7：无版本但不能审核漂移 | Admin/Web/CLI审核与展示 |
| Admin Expert/Team 停止通用 JSON 写入；A24 拒绝这两类，A25 ZIP-only；Admin 写入时冻结/校验 pin | C06/P：ZIP 是内容/关系权威 | Admin原ZIP导入/重传及所有实际JSON调用者 |
| 同名/多版本不共存；同名不同内容安装冲突失败，不生成别名、不改写 frontmatter | D3：本期拒绝共存，不改 Fleet/Daemon | 安装客户端及迁移校验 |

已确认范围按统一契约同步调用者，不保留旧字段推断、旧数据自动升级或错误 fallback。既有权限、审核和市场动作不重新设计，Fleet 协议不变。**pin 是唯一显式依赖版本规则；没有“失败后补 latest、切 legacy、删边、跳文件、自动换 pin”的隐藏分支。** 同名/多版本共存明确不做，不阻止完整内容查询按资源＋pin 版本身份返回数据，也不构成安装隔离承诺。

## 2. 公共 DTO（规范性字段定义）

下列 `{}` 表示合法开放领域JSON对象而非未定义实现。所有 ID 是非空字符串，服务生成的 version_id/attempt_id 为 UUID；既有 plugin_id 按原契约兼容。所有版本化字段在 Connector 响应省略，不能填假值。

### 2.1 RelationWire / RelationSnapshot

核心wire约束：expert_skill、expert_team_expert 写入目标 `plugin_id`；`target_version_id` 是**权威 pin**——写入时可省略/为 null，发布/提交时冻结为目标当时的当前正式发布版并写入源快照；显式值必须属于目标且已 release。公共读取、A03、A34、安装、审核一律读该 pin，不解析目标最新版。expert_connector 仅目标 ID。以下接口变化只服务独立引用与 P pin 模型，不新增市场治理动作。

| 字段 | 类型与必需性 | 规则 |
|---|---|---|
| relation_id | string，新建可省略，更新已有边应回传 | 来源关系身份；容器重传按类型+目标匹配保留 |
| source_plugin_id | string，写可省略，读必返 | 写时如有必须等于当前来源 |
| target_plugin_id | string，必填 | 关系权威身份，不允许名称替代 |
| target_version_id | string，可选（草稿）；已发布 Expert/Team 快照必填；仅 expert_skill/expert_team_expert 可给 | **权威 pin。** 显式提供时必须属于目标且有 release，否则 400 `dependency_version_mismatch`；省略/为 null 时由发布/提交动作解析为目标当时的当前正式发布版；发布后不再改。公共读取/A03/A34/安装/审核观察只读该值，不回退/切换到目标最新版。Connector 必须完全省略，null/空字符串 400 `connector_version_forbidden` |
| relation_type | enum，必填 | expert_skill / expert_connector / expert_team_expert |
| sort_order | integer，默认0 | >=0 |
| data | object，默认{} | team仅member_key/role/is_leader且三项必填；member_key正则/长度、role非空≤500字符、唯一Leader及非Leader禁保留role=leader见技术§5.1；其余关系只能{} |

RelationSnapshot 与上述同字段，source_plugin_id 可由所属版本推导；不允许客户端写 observed_*。**pin `target_version_id` 不进入 `content_digest`**（草稿为空、发布时才补齐），但**显式给出的 pin 属于 RelationSnapshot 身份，参与新版本/复用判定**；发布时补齐缺失 pin 不单独产生新版本。pin 是已发布快照的一部分，进入 pin 图、可用性、删除保护、A34 与安装。目标发新版不改变已发布关系的 pin、不要求来源发新版、不触发引用方重新审核；来源发布新版本才刷新未显式指定的 pin。Review 独立返回 `connector_observations:[{plugin_id,effective_revision,effective_digest}]` 和 pin 观察 `resolved_dependency_observations`，这些是服务端冻结值，不是 latest。

来源仍使用既有 private 本人草稿/发布时：private 来源可引用已发布 system、同 Space 的 space 及**同 owner private**（private Expert→private Skill、private Team→private Expert 合法）；**space/system 来源不得引用 private；跨 Space 禁止；同 Space 不同 owner 的 private 仍禁止。private 同 owner private→private 合法。** private 资产作为依赖时不扩大受众，A34/安装只对同 owner 可见。纯本地组合不因此纳入市场。

### 2.2 PluginProjection / Detail / Version

本节投影用于读取来源自身版本、隔离未发布编辑，并表达 P 的 pin 结果；沿用现有owner/Admin鉴权，不另设取证模式或新增内容治理权限。历史受众检查不得因资源后来公开而泄露旧私有内容。

PluginProjection 保留既有身份、plugin_name/type、category/tags/publisher/icon、owner_id/space_id/visibility、listing_state/display_status、评分与计数；新增：

- `view`: published/editing/effective，必返，Connector市场为effective。
- `selected_version_id`、`version_label`：**来源自身** view 对应的实际版本；published视图必返，为来源当前正式发布版（根版本），不随依赖更新改变；Connector省略。
- `published_version_id`：Tenant版本化资源必返、允许null；不能用current填充。**Admin运营DTO完全省略该字段（包括null占位）**，与原具体方案一致。
- `current_version_id`、`edit_revision`、`has_unpublished_changes`：仅授权编辑者/管理者的editing或mine响应返回；current可null，edit_revision从1开始。版本化资源的has_unpublished_changes为current非空且与published不同；两者均空为false。Connector省略current_version_id，以完整editing内容及拟定受众与effective内容比较has_unpublished_changes，不比较两个独立计数器是否相等。公开详情不得泄漏草稿ID或编辑修订。
- `effective_revision`、`effective_digest`：Connector的有效内容必返；没有有效内容时，仅授权editing可见并以null表示。
- `manifest_json`、`plugin_json`：详情返回；列表保留既有轻量manifest投影，不返回Package。字段内容与view选根一致，不能名称是current、Package是published。
- `dependency_availability`：仅A01 `mode=mine`中的Expert/Team必返，供作者诊断；其他类型及公共市场列表省略。独立于listing_state/display_status，不改变发布、审核或上下架状态，结构及安全规则见§2.2.1。

Detail=`{plugin:PluginProjection,relations:RelationWire[],relation_result?:{created:string[],updated:string[],deleted:string[]}}`。Connector relations为空。Detail 的 relations 表达来源自己的关系集合与已保存的 pin（草稿可为 null）；目标 pin 版本的内容由 A03/A34/A10 表达，不在 Detail 里改写 pin。

Version=`{plugin_id,version_id,revision_no,version_label,declared_visibility,is_released,is_published,manifest_json,plugin_json,relations,files,content_digest,created_at,created_by}`；`files:[{path,mime_type,content_size,content_hash}]`不含内部key。VersionSummary明确省略manifest_json、plugin_json、relations、files，只保留身份/序号/标签/受众/状态/摘要/创建信息，不能每行返回完整关系/文件图。is_published为当前发布，is_released为曾正式发布。历史读取受众取当前有效范围与该版本declared_visibility交集，后来公开不追溯公开旧私有历史；不明历史仅owner/Admin按既有权限管理读取。A09 restore 已删除，因此 Version/VersionSummary 不再有 `restored_from_version_id`。

A34完整内容读取已确认，沿用既有鉴权、可用性和历史受众检查，不增加安全状态或专用取证入口。A02/A03供详情/轻量图展示，A05～A08供独立历史/文件工具；不得用“先查A03，再按目标ID/版本逐个补正文/文件”替代一次完整专家内容查询。

#### 2.2.1 A01：公共市场过滤与“我的发布”依赖诊断（P 语义）

继续使用`GET /market/api/v1/plugins`，不新增接口或依赖检查query开关。公共列表以来源**当前正式发布版**的 relations 为关系集合（不用 current/草稿替代），逐条按**该快照的 `target_version_id` pin** 校验；只展示自身符合原上架/权限条件、且完整 pin 图均可用的资源：任一 Skill、Connector 或成员 Expert 的 pin 不可用（pin 版本缺失/下架/删除/失去 release、目标 plugin 不可读/已删除/未 embedded/未上架/受众不通过；Connector 无 effective 行），依赖方不出现在列表；Team 递归为 Team pin 的成员 Expert 版本 → 该 pinned Expert 版本的 Skill pins/Connector，不能只查直接关系。**不解析目标最新版，不得用新草稿/current 替代 pin 目标，也不得在 pin 不可用时回退或切换到其他版本。目标发布更新版本不会影响 pin 到旧版本的来源。**

该条件在SQL分页、排序及COUNT之前生效；`pagination.total`、搜索/分类筛选、`GET /plugin_categories`的plugin_count与公共`GET /plugin_tags`复用同一 pin 可用性判定。只查refs、版本和资源状态，不逐条调用A34或读取内容文件，也不做目标 latest 解析。目标缺失不能因INNER JOIN丢边而被当成“无依赖”。

分类接口仍返回既有active配置行，不因plugin_count=0而新增隐藏规则；前端是否隐藏空分类保持原行为。完整性检查与公共COUNT/列表/分类/tags使用同一数据库一致性快照：将relations_json的relation_id/type/target_plugin_id/**target_version_id**经JSON_TABLE投影，与refs双向比对（含source_plugin_id归属），并核对数组类型及数量。**已发布快照的 pin 缺失、refs 索引缺失必须 fail closed：返回 500 `relation_index_inconsistent`，不得进入公共结果，也不得被依赖不可用过滤吞成零依赖；** 合法空数组与零refs可用；版本缺失、非数组、漏/多refs或等数量错target均500 `relation_index_inconsistent`。refs一致但目标Plugin缺失才按正常不可用处理。`target_version_id` 是 pin，不是可空审计列；其缺失或与目标状态不符按 pin 不可用/完整性错误处理。

`mode=mine`保留原owner/Space及自身未删除条件，包含原有草稿、待审、已发布、下架等资源；**不得因依赖异常删除行或减少total**。mine缺省及唯一允许的`view`为`editing`；显式`mode=mine&view=published`返回400 `VALIDATION_ERROR`（`details.field=view`、`details.reason=view_invalid`），不能忽略参数、隐藏从未发布的草稿或回退current冒充published。公共模式缺省及允许的`view`为`published`；仅`plugin_type=connector`可显式使用`effective`，响应按有效态归一。公共模式显式`editing`同样400；非法组合统一400，`details.field=view`、`details.reason=view_invalid`。Web“我的发布”的全部/专家/专家团列表显式请求`mode=mine&view=editing`，保留编辑投影及原操作入口，不增加回收站或已删除来源展示。mine标签统计与A21 Admin管理列表也不得套用公共市场的依赖失效隐藏条件。

每个mine Expert/Team返回`DependencyAvailability={status:"available"|"unavailable",evaluated_view:"published"|"editing",evaluated_version_id:string,reasons:DependencyReason[]}`：

| 字段 | 规则 |
|---|---|
| status | 只表示本次 **pin 图**（来源评估版本及其快照全部 pin）的元数据可用性；available时reasons为空，unavailable时至少一条原因。不代表根自身已上架、A34文件完整或Fleet安装成功 |
| evaluated_view | 指**来源自己**的视图：来源存在published_version_id时必须为published，即使本行展示editing草稿；只有从未发布的来源才以current草稿评估并返回editing。reasons 始终基于该评估版本自身的 relations/pins，不用展示行的草稿关系替代 |
| evaluated_version_id | 与上述评估根严格一致的UUID，必填；不能用草稿修复状态覆盖线上版本诊断。应有版本/refs/pin缺失或数据库失败按既有完整性/服务错误处理，不伪造available |
| reasons | `DependencyReason={code,target?:DependencyTarget,via?:DependencyTarget[]}`；code仅 dependency_delisted／dependency_deleted／dependency_unavailable；`DependencyTarget={plugin_id,plugin_type,plugin_name,version_id?}`，类型为skill/connector/expert，Connector省略version_id |
| target.version_id | **该依赖在评估快照中的 pin**；目标从未发布、已删除、pin 缺失或无权确认时省略；不是目标最新版 |
| target/via | 仅当目标及整条中间路径均可按既有授权读取时返回。直接原因省略via；Team的间接原因via给 Team pin 的**成员 Expert**（含 pin 版本），不包含根和目标本身 |

**从未发布草稿的 pin 可为空：** mine 对空 pin 的关系返回 `status=unavailable`、`reasons=[{code:"dependency_unavailable",target?}]`（不返回 `target.version_id`，可披露的 target ID/name 仍可给），提示“依赖尚未冻结，发布时校验”；**不得用目标当前正式发布版冒充 pin**。已发布快照缺 pin 属完整性错误，按 500 `relation_index_inconsistent` 处理，不得降级成普通 unavailable。

无法查看目标、目标缺失无法确认、或已删元数据不可授权读取时，**只把这些不可披露项合并为一条**`{"code":"dependency_unavailable"}`（不按每个隐藏项各返一条，也不把可披露项一并吞掉）；不返回隐藏ID、名称、类型、版本、路径或隐藏数量，不通过旧缓存/快照名称绕过现有权限，也不暴露它究竟下架还是删除。**名称只取 pin 的 released snapshot/effective 内容；无权或没有 released 内容时匿名/空名，不使用其他 owner 的 editing name。** **若同时存在可披露原因，`dependency_delisted`/`dependency_deleted` 及 target/via 仍按授权逐条保留**；混合列表只隐藏不可披露部分，不能因存在匿名原因就把整组变成匿名。普通下架/删除依赖是正常诊断数据，mine仍成功返回该行；真正查询/完整性故障不吞成正常原因或空数组。

P 下典型返回（pin 是唯一依据）：

| 场景 | 返回 |
|---|---|
| E 已发布 v1，其 pin 的 S-v1 被下架 | `status=unavailable, evaluated_view=published, evaluated_version_id=E-v1, reason=dependency_delisted target=S-v1`；Web 显示“线上发布版异常”，不是“已恢复” |
| E 从未发布，current v1 的 pin 的 S-v1 被下架 | `status=unavailable, evaluated_view=editing, evaluated_version_id=current-v1, reason=dependency_delisted target=S-v1`；Web 显示“草稿依赖异常（尚未发布）” |
| E 已发布 v1 pin S-v1；S 后来发布 v2，S-v1 仍 released/目标可读 | `status=available, reasons=[]`（目标发新版不影响 pin） |
| E 已发布 v1 pin S-v1；S-v1 被下架，但 S 已发布 v2 | `status=unavailable`，`target.version_id=S-v1`（不回退/不切换到 v2） |
| 你无权看 S（或整条 via 中任一项无权） | `status=unavailable, evaluated_view=published/editing, reasons=[{"code":"dependency_unavailable"}]`，无 target/via/数量 |

以下是同一mine列表行新增字段的两个片段，不是新增API包络。可授权的下架依赖（`target.version_id` 为 pin）：

```json
{
  "dependency_availability": {
    "status": "unavailable",
    "evaluated_view": "published",
    "evaluated_version_id": "00000000-0000-4000-8000-000000000101",
    "reasons": [
      {
        "code": "dependency_delisted",
        "target": {
          "plugin_id": "demo-skill",
          "plugin_type": "skill",
          "plugin_name": "部署检查",
          "version_id": "00000000-0000-4000-8000-000000000102"
        }
      }
    ]
  }
}
```

无权或无法安全确认具体目标时，只返回匿名原因：

```json
{
  "dependency_availability": {
    "status": "unavailable",
    "evaluated_view": "published",
    "evaluated_version_id": "00000000-0000-4000-8000-000000000101",
    "reasons": [{"code": "dependency_unavailable"}]
  }
}
```

Web以固定code映射中英文i18n，不由服务端返回任意HTML或整句文案。unavailable显示独立“依赖异常”标记，hover/键盘focus的Tooltip说明原因；保留原发布/审核标记和操作。published提示可用“Skill「部署检查」的固定版本已下架，当前发布版本暂不在市场展示”；间接原因说明成员专家路径；匿名提示“存在已失效或不可访问的依赖，请检查依赖配置”。editing提示必须写“草稿依赖异常”（空 pin 时提示“依赖尚未冻结，发布时校验”），不声称该草稿已从市场隐藏。全部tab经getMySkills/MineRow/MineTable及专家/专家团tab均须展示，不仅修改市场卡片。

不改变来源listing_state，不持久化新治理状态；pin 版本恢复合法上架，或来源发布**移除/替换该依赖的新版本**后，下一次列表查询重算。只保存修复草稿不能清除已发布版本异常；来源发布新版本后，公共市场按新版本的 pin 图读取，旧版本仍按自身 pin 诊断。沿用既有刷新/缓存失效，不新增推送/轮询。前端不能过滤mine异常行或为标记/Tooltip逐卡请求依赖、A34或完整文件；缺少必需诊断字段不能默认为available。

### 2.3 Write / Publish / Review

发布审核职责、owner提交、User/Bot鉴权、Space自动审核开关及管理权限均不变。自动/人工批准执行同一 **pin 图**校验及发布隔离；**提交冻结 pin，批准只校验 pin/effective，批准后 pin 不变，不冻结未来内容也不需要未来内容**。**ZIP-only仅适用于Admin Expert/Team的内容、内容元数据与依赖写入（A25）；Tenant Expert/Team仍使用A12完整Write+relations。** A24拒绝Admin Expert/Team创建/更新，评分及placement保持原专用入口，不新增metadata PATCH旁路或保存/发布矩阵。**Connector 例外：Admin 系统 Connector 的既有直接生效写入保留（同步更新 effective，属既有平台操作）；Tenant Connector 编辑必须审核；Expert/Team 审核观察 effective。** **Admin 不做资产审核：A24/A25/A26 的 create/update/reupload 直接生效并同步 published/effective，不创建 pending review、不产生 review_id、不进入 A15 队列；只有 Tenant 的 A13/A14 按现有 Space 策略自动或人工审核；Admin 写入时解析并冻结缺失 pin。Admin 的迁移执行授权不等同于资产审核，也不得被实现成新审核矩阵。**

Write 保留现有 `{plugin:{plugin_id?,plugin_name,plugin_type,visibility,version?,manifest_json,plugin_json,category_id?,tags,publisher?,icon?},relations:[RelationWire],expected_edit_revision?:integer}`。

- **A12 完整 Write** 中 plugin_name/type/visibility/Manifest/Package/relations 显式必需；其他已有可选元数据按原语义保留。新建省略plugin_id和expected_edit_revision；更新二者必填（Admin path ID可作为plugin_id，但body如有必须相等）。缺失/过期 `expected_edit_revision` 为 409 `edit_conflict`；`relations` 缺失/null 为 400 `relations_required`。**relations 中的 `target_version_id` pin 可省略/为 null；显式给出时按 RelationWire 校验归属/release，省略的 pin 在发布/提交时冻结。** 此要求的 `relations` 仅指本条完整 Write；A12 Skill import、A25 容器导入均不接收 HTTP relations（分别见下与 §4）。
- `version` 是候选SemVer标签，更新省略沿用current标签；正式发布时与release验证，不靠保存序号判断。Connector仅保留原合法展示标签兼容，无版本选择语义。
- `relations:[]`明确清空；不是PATCH语义。返回Detail。
- owner/Space不可写；Tenant写入visibility是编辑拟定受众，保存到declared_visibility，不立即改变线上visibility。发布时按候选图校验outbound并检查incoming后原子切换；Admin更新的归属/范围保持原值。
- relations 存目标 `plugin_id` 与 pin `target_version_id`；草稿可省略/为 null，发布/提交补齐，显式 pin 校验后保留。保存只记录目标身份与显式 pin，不解析 latest，也不把缺失 pin 猜成目标当前版本。

同一content_digest不能直接判为重复保存：候选version_label、declared_visibility或完整RelationSnapshot身份（含关系身份、排序、data 及**显式给出的 pin**）变化也生成新不可变Version；只有上述全部冻结输入一致才可复用current。content_digest只由服务端统一codec生成，客户端视为opaque并原样传递，不自行用JSON.stringify或Go默认编码推导；它不等于公开可重算的A34 bundle_digest。**pin 不进入 content_digest**（草稿可空、发布时才补齐）；显式 pin 变化属于关系身份变化，应生成新快照；发布时补齐缺失 pin 不改变内容摘要、不单独产生新版本。目标发新版不改变已发布来源的摘要/pin，也不要求来源发新版本。

**A12的既有Skill包导入：**`POST /plugins/import`保留JSON `{parse_task_id,plugin_id?,plugin_name?,name?,description?,category_id?,tags?,visibility?,icon?,version?,changelog?,expected_edit_revision?}`及200 `{data:Detail}`。新建必parse_task_id和明确plugin_name，省略plugin_id/CAS；更新必parse_task_id、plugin_id、expected_edit_revision，其他元数据省略沿用原值，category_id可显式null清空，tags显式[]清空。name仅为既有别名，同时给plugin_name时不允许值矛盾。parse task须属于当前UID/Space、成功且未消费；完整SKILL.md、原ZIP及支持文件由服务端保存，不能只用readme重建Package。该请求不接relations，Skill最终关系为[]；成功生成editing快照及selected/current/version_label/edit_revision，不自动发布。CAS或持久化失败释放task消费权且保留旧版本/文件；前后并发和消费必须与内容提交一致。A14不再接parse_task_id，审核必须在保存后提交准确version_id。MP-01负责该既有端点及OpenAPI扫描/生成，MP-02不另造上传协议。

版本化Publish=`{plugin_id,version_id,version,changelog?}`，version必须等于指定快照version_label，version_id必填，不猜current；**version_id 选择来源自己的待发布快照，不是依赖 pin**。发布事务内为该版本关系补齐缺失 pin（解析为发布时目标当前正式发布版）、校验显式 pin（归属/ release/受众）、读取 Connector effective；pin 写入快照/refs 后登记 release/切 published。显式指向非正式版本 400 `dependency_version_mismatch`；任一 pin 目标不可用 409 `dependency_unavailable`，不回退/不切换版本、不删边、不静默替换。Connector Publish=`{plugin_id,expected_edit_revision,changelog?}`，不传version_id或业务version。提交审核同样使用确定版本或Connector编辑修订，并在提交时冻结缺失 pin，防并发保存后发布错内容。

PublishResult保留`{plugin_id,listing_state,display_status,review_id?}`，Tenant版本资源另返published_version_id；Admin不返该指针。A13沿用既有成功HTTP 200，以listing_state/display_status/review_id区分待人工审核与已发布，不新增202分支。Space按现有策略自动批准或进入人工审核：**默认自动审核**（无 Space 配置时 `IsAutoApproveEnabled=true`），自动路径提交→冻结 pin→批准，若提交/批准间 pin/effective 校验失败或并发冲突则按同一 pin 图重试一次，正常不把 409 暴露给用户；**目标发布新版本不触发 409**。Space 显式关闭自动审核后进入人工审核。private既有直接发布不变，同 owner private→private 依赖允许，但不扩大受众。

Review提交=`{plugin_id,version_id,changelog?}`；Connector为`{plugin_id,expected_edit_revision,changelog?}`，不另传Manifest/Package/relations/parse_task_id。approve/cancel空body，reject=`{reason}`；实际批准复用唯一发布事务。A15保留现有review_id/plugin_id/space_id/target_scope/status/kind/version/changelog及申请者、审核者、自动批准/决策来源和时间字段；以下是明确的详情增量，不在队列列表逐行加载重内容：

- `source_version_id`：版本化资源必返；`source_edit_revision`与`frozen_content_digest`必返；Connector省略source_version_id和业务版本字段。
- `frozen_version:Version`：版本化源的完整冻结版本（**只冻结来源自身内容**），其 Expert/Team relations 携带提交时冻结的 pin；Connector省略；`frozen_connector:{manifest_json,plugin_json,declared_visibility}`仅Connector必返。两者清除内部对象key，正文来自冻结内容，不向current详情扇出补齐。
- 保留详情`readme_content`及`frozen_relations:RelationSnapshot[]`，与冻结Version完全一致；合法无关系返[]，快照缺失/解析失败500 relation_index_inconsistent，不能沿用旧decode失败吞空。详情失败不改变审核状态，仍可按已有权限拒绝/取消。
- `resolved_dependency_observations:[{plugin_id,plugin_type:"skill"|"expert",version_id}]`：提交时冻结的 **pin 观察**，覆盖来源快照图中全部 Skill/Expert 目标（不含来源根；Team 含 pinned 成员 Expert 及其 pinned Expert 版本的 Skill pins），按 plugin_id 去重；`version_id` 是 pin，不是目标最新版。观察值是该提交的冻结依赖快照，批准只校验这些 pin，不重新解析 latest。Connector 仍由 `connector_observations` 单独表达（D7）；两者合起来覆盖全部 pin/effective 依赖。
- `connector_observations:[{plugin_id,effective_revision,effective_digest}]`记录 pin 图中全部 Connector 当前 effective；Connector自身提审也记录其提交时effective基线，首次无有效行可为null/null。仅该首次根允许null。
- 批准只校验 pin/effective：人工审核窗口内任一 pinned target version 失效（删除/下架/失去 release）、目标不可读/已删除/未 embedded/未上架/受众不通过，或 `connector_observations.effective_*` 变化，返回409 `dependency_changed`，由有权取消者取消旧申请，再由有权提交者重新提交并生成新的 review_id；审核人查看新申请后批准；自动审核内部重试一次，不把日常 409 暴露给用户；重试后仍失败则不发布并返回409 `dependency_changed`，不静默批准。**目标发布新版本不构成 `dependency_changed`；只改 editing 不误判；批准后 pin 不变，Skill/Expert 发新版不要求重新审核引用它的 Expert/Team。**
- `baseline_version_id:string|null`及`dependency_diff:[{change:"added"|"removed"|"changed",before?:RelationSnapshot,after?:RelationSnapshot,before_version_label?,after_version_label?}]`在提交时按当时published根冻结，Connector的baseline_version_id为null、关系diff为空。added仅after、removed仅before、changed二者皆有且确有 target ID／type／sort_order／data 等变化；按relation_type+target_plugin_id对应边。**pin 变化不单独生成 `changed`，目标发新版不参与 diff；** 目标可附 pin 的真实 version_label（仅提示），Connector不附业务标签。基线、diff和观察值由服务端存入审核快照，不随current或后来published漂移；列表省略这些重字段。沿用审核读取授权和历史受众规则，无权依赖不泄漏身份、缺图不伪造完整预览。

**人工审核冲突恢复（不新增端点）：** 409 `CONFLICT` / `details.reason=dependency_changed` 不修改旧 pending 申请或冻结观察值，GET 刷新也不更新基线。有权取消者使用既有 cancel 取消旧申请；取消与批准竞争按 pending CAS 仲裁，若已批准则读取原结果，不继续代提。有权提交者随后使用既有 submit 生成新的 review_id 和新 pin 观察值。Skill/Expert/Team 可重新提交同一不可变 source_version_id，沿用该版本已冻结的 pin（缺失 pin 才在提交时补齐），仍须通过当前权限、标签、发布基线及依赖资格检查；不自动改用 current 新草稿，也不把目标最新版填入 pin。Connector 提交必须携当前 expected_edit_revision，重新冻结当前编辑内容，若编辑已变化须展示并明确重新提交，不冒充原冻结内容。旧记录永久保留；新记录进入原自动/人工策略；批准空 body 不变。客户端不得自动取消、重提或连续重试 approve。迁移 run 的原 review 被取消后保持 blocked，不自动换绑新 review；另行修订迁移计划和授权处理，不通过普通页面重提绕过迁移恢复约束。

### 2.4 Graph / Candidate / Dependents

Graph=`{root_node_key,nodes,edges}`。**依赖节点是所选根快照 pin 的版本；节点版本就是关系 pin，不是目标 latest。** A03 显式选择历史根时，根节点是所选根自身版本，依赖按该历史根快照自身的 pin；A34 不提供根版本选择。

- VersionNode=`{node_key,plugin_id,version_id,plugin_type,version_label,plugin_name,manifest_json,files}`；ConnectorNode=`{node_key,plugin_id,plugin_type:"connector",plugin_name,manifest_json,effective_revision,effective_digest}`，其manifest_json来自本次选定完整effective行，不读editing。
- node_key为版本化plugin_id@version_id（pin 版本）或Connector plugin_id@effective_revision；VersionNode的files为`[{path,mime_type,content_size,content_hash}]`，ConnectorNode不承载文件清单，其ContentAsset.files必须[]。节点ID唯一；每条边的source_version_id等于source节点version_id，target ID及pin版本/有效修订等于target节点，root身份与根节点逐字段一致。**节点 `plugin_name`/标签取 pin 的 released snapshot/effective，不读 current 或其他 owner 的 editing name。** 所有节点从根可达，不能塞入无边的图外资产；同一依赖插件在一次 pin 图中只出现其 pin 的一个版本，不按名称合并、也不并列其他版本。
- 图统一最多1500个节点（含根，按资源＋pin 版本/有效修订去重）、2000条边；A03与A34使用同一计数规则。Marketplace原detail_graph的630子节点限制和旧install的500目标限制须在对应 pin 图路径更新，不能误把Connector新增节点当成已兼容；Fleet现有30成员/600 Skill及其他限制不变。读取边/节点时增量计数，超限413 budget_exceeded，不能全量构建后裁剪。
- edge=`{relation_id,source_node_key,target_node_key,source_version_id,target_plugin_id,target_version_id?,resolved_effective_revision?,relation_type,sort_order,data}`；`target_version_id` 是**关系 pin**（节点身份），不是目标 latest；Connector省略target_version_id并返回本次选定的resolved_effective_revision。所有端点必须在nodes，缺目标不能裁剪后200。
- A03的Graph是轻量版本图，files仅文件清单，不含正文；node_key不是读权限凭据。完整专家/团队查询直接取A34，其graph给 pin 身份及映射，assets已含全部实际内容，客户端不再补取Version/文件。独立历史工具仍可单独调用A05～A08，历史版本自身内容不可变，其依赖按该历史快照自身的 pin（回填后），不恢复/不解析 latest。

Candidate保留Plugin身份/发布者/scope，新增`published_version_id`、`version_label`（Skill/Expert **当前正式发布版**）、`effective_revision`（Connector **当前有效修订**）、`can_select`、`reason?:string`、`is_already_bound`。**Skill/Expert 的这两个身份字段只取当前正式发布版，绝不用 current/editing 版本或标签填充；从未发布时 `published_version_id=null`、不可选。** 目标无权限根本不返回；有权限但不满足来源受众/状态时可返回安全的不可选原因。候选展示的是**选择时的目标当前正式发布版**，选择动作本身不写 pin；发布/提交时由服务端冻结为当时的正式发布版（显式 pin 以显式值为准）。后续读取/安装只读冻结后的 pin。

Dependents为标准列表，每行`{source_plugin_id,source_version_id,relation_type,target_plugin_id,target_version_id?,is_indirect,via?:[{plugin_id,version_id}]}`，另有顶层`meta:{has_hidden_dependents:boolean}`。集合是删除保护的live边＋published直接/间接路径并集：live使用该来源current_version_id；published**只取未删除来源的当前 `published_version_id`**（历史发布版不进入保护集合），直接按 refs 的 `target_version_id` pin，间接展开“Team 当前发布快照 → pin 的成员 Expert 版本 → 该 pinned Expert 版本的 refs”，使用 Team 快照里的 pin，不取成员最新版，也不展开第二个 Team。source_version_id指路径根版本，relation_type是最终指向被查询目标的边类型，target_version_id是**该路径的 pin**，不是目标最新版。相同根版本、最终边、目标 plugin_id、完整 via pin 身份只返一次，不同路径保留；按source_plugin_id/source_version_id/is_indirect/via/target_plugin_id/relation_type/target_version_id稳定排序后分页。目标是Connector时省略target_version_id；直接行省略via。仅完整授权路径进入data及total，隐藏路径仅置has_hidden_dependents，不返回其ID/数量，不能对隐藏路径分页计数。

## 3. Marketplace 接口清单

表内ID用于issues/manifest关联。未列出的分类、评分、解析任务、Space审核策略等既有能力保持原协议，除内容读取必须遵循统一投影。operationId保持现有者不因本文重新命名；新增者在表中给出。

### 3.1 读取和历史

读取沿用现有权限，并补准确的正式/编辑版本选择与 **P pin 读取**。A09 restore-to-draft 已删除（D2），不提供“历史还原为草稿”动作；历史读取 A04～A08 保留，历史版本自身快照不可变，依赖按该快照自身的 pin（cutover 回填后），不解析 latest、不伪造 release/pin。

| ID | 方法/路径 | 请求 | 返回/权限 | 所属issue |
|---|---|---|---|---|
| A01 | GET /plugins | scene_code必填；plugin_type公共必填、mine可省；保留category_id/tag/q/mode/sort/page/page_size，新增view；mine缺省且仅editing；公共缺省published，Connector可显式effective；不合法组合400 `details.field=view`/`details.reason=view_invalid` | 公共按 **pin 图**在COUNT/排序/分页前过滤；mine保留原非删除行及Expert/Team四字段诊断；Admin列表不套用公共过滤 | MP-01底座；MP-02判定扩展 |
| A02 | GET /plugins/detail | plugin_id、include_relations默认true、view默认published | Detail；editing只owner/有权管理者，Connector默认effective；relations 为目标ID＋已保存 pin（草稿可空），不改写/伪造依赖 pin | MP-01 |
| A03 | GET /plugins/detail_graph | plugin_id、view默认published；可给version_id选择根自身已发布版本（历史展示），与view=editing互斥；**显式version_id只按该版本读取，不回读current** | 轻量Graph；依赖节点为该根快照 pin，根按version_id或published，不含正文；历史根仅有权且曾发布 | MP-02 |
| A04 | GET /plugins/versions | 保留plugin_id/page/page_size；scope=released默认；scope=all只owner/管理者 | VersionSummary列表；Connector400 unsupported_type | MP-01 |
| A05 | GET /plugins/{plugin_id}/versions/{version_id} | path IDs | Version，所属检查；普通人只能正式版本；operationId=plugin.version.get | MP-01 |
| A06 | GET /plugins/skill_md | 保留plugin_id；可选version_id，不给时取published | `{content,version_id}`；仅Skill，不能取current兜底 | MP-01 |
| A07 | GET /plugins/download | 保留plugin_id；可选version_id，不给取published | Skill ZIP，文件缺失明确失败，不静默改取其他版本/最新包 | MP-01 |
| A08 | GET /plugins/{plugin_id}/versions/{version_id}/files/content | query path必填，相对路径 | `{path,content,mime_type,content_hash}`，仅UTF-8文本；operationId=plugin.file.get | MP-01 |
| A10 | GET /plugins/dependency_candidates | source_plugin_id可选；新建必传source_type/source_visibility；target_type/q/page；existing来源可给拟定visibility但要验证权限 | Candidate列表（选择时的目标当前正式发布版），非场景列表；不写 pin，发布/提交才冻结；operationId=plugin.dependency.list | MP-02 |
| A11 | GET /plugins/{plugin_id}/dependents | page/page_size | Dependents；只资源owner/有权治理者；operationId=plugin.dependent.list | MP-02 |

历史文本附件可由A08独立读取；不新增任意对象URL接口。Admin读Skill用A23 skill_md/download，Expert指令从A22 Version.plugin_json中的原始AGENTS或合法受控内容接口取得；不能读取storage引用自行下载。A03默认published；Connector的缺省市场投影规范化为effective，显式editing仅有权管理者，不能给Connector伪造version_id。完整专家/团队内容只在§6的A34一次取得，不用A03+A05～A08拼装。

### 3.2 写入、审核、生命周期

以下发布/审核/下架/删除均复用既有动作、权限与入口，只接 **P pin 冻结/校验**与引用完整性校验。不新增下架类别、治理状态或运营流程。删除保留既有 live incoming 保护，并补 published refs 及间接引用；**published 保护只取未删除来源的当前 `published_version_id`，按精确 `target_version_id` pin判定，不以全部历史发布版作为保护超集**；不能因改用 pin 而放开原本阻止删除的草稿/live 引用。**移除 `rejectPublishedMutation` 对 `Delete`/`DeleteGraph` 的删除总禁令；已发布资源收窄 visibility 的既有行为不变，不把删除并集复用到收窄。**

| ID | 方法/路径 | 请求与语义 | 返回 | 所属issue |
|---|---|---|---|---|
| A12 | POST /plugins/upsert、POST /plugins/import | 前者完整Write+relations（CAS 必带）；relations 的 pin 可选、显式校验；后者Skill parse_task包导入，独立请求见§2.3；更新均CAS，只推进editing | Detail，包含实际selected/current版本与修订；Connector无版本 | MP-01底座；MP-02依赖接入 |
| A13 | POST /plugins/publish | Publish，显式version_id选择来源自身快照，**只按该version_id发布、不回读current**；锁内补齐缺失 pin/校验显式 pin并读取 Connector effective | PublishResult；pin 目标不可用409 dependency_unavailable，显式 pin 非正式400 | MP-01发布；MP-02规则 |
| A14 | POST /plugins/review_requests | Review提交，锁定请求提交的来源自身version_id/Connector编辑修订，**只按该提交版本记录/审核、不回读current**；提交时冻结缺失 pin 与 effective 观察 | 201 ReviewDetail | MP-02 |
| A15 | GET /plugins/review_requests 及 GET /plugins/review_requests/{review_id} | 保留既有过滤/分页/本人或Space审核者权限 | 列表/ReviewDetail（含 pin 观察 resolved_dependency_observations/connector_observations） | MP-02 |
| A16 | POST /plugins/review_requests/{review_id}/approve | 空body；**批准只校验 pinned 版本/目标可用性与 Connector effective**；pin/effective 变化409 dependency_changed，目标发新版不 409；自动模式内部重试；批准释放冻结快照（含 pin），不重新解析 | PublishResult，重复同请求返回已批准结果不重新发布 | MP-02 |
| A17 | POST /plugins/review_requests/{review_id}/reject 或 /cancel | reject必需reason；cancel空body | `{}` | MP-02 |
| A18 | POST /plugins/delist | 保留原`{plugin_id,reason}`、鉴权与下架语义，不另设下架类别或依赖豁免；已发布资源收窄 visibility 的现有行为不变 | 原listing状态结果 | MP-02 |
| A19 | POST /plugins/delete | 保留原`{plugin_id}`和软删除；删除保护 = live incoming（含草稿、不过滤 listing）∪ published refs（**仅未删除来源的当前 `published_version_id`**，按**精确 `target_version_id` pin**，含 Team→pinned 成员 Expert 版本→其 refs 的间接路径；历史发布版不构成保护超集）；无保护引用的曾发布资源可删除，不新增“曾发布即禁止删除” | 原删除结果，不删除共享独立依赖 | MP-02 |

现有场景投放可见性接口原样保留；非默认场景可设is_visible，default场景现有禁止隐藏的约束不变。不将场景可见性扩成资产级隐藏状态，也不新增平台治理API。A09 不恢复；历史读取与既有附件保留。

### 3.3 Admin 保留接口及必要增量

Expert/Team沿用A25 ZIP上传/重传，清单是关系集合的权威（目标 ID；`version_id` 为可选显式 pin）。其余Admin写入保持原鉴权、归属、listing及内容生效策略，仅适配 **P pin 冻结/校验**、并发校验和完整关系，不增加独立发布/重新上架入口。**Admin 创建/更新/重传（A24/A25/A26）直接生效并同步 `published_version_id`/effective，不创建 pending review、不产生 `review_id`、不进入 A15 队列；写入时解析缺失 pin、校验显式 pin。只有 Tenant 提交走 A13/A14 的现有自动/人工审核。A24 的 Skill/Connector 关系为空，不要求 `relations`；A12 完整 Write 的 `relations` 必填不外溢到 Admin 写入。** **存量 Connector 回填：** 切换前把现有 Connector 内容回填为 `effective_revision=1` + `effective_digest`（覆盖完整有效行：manifest/package/declared visibility），否则 A21/A34/安装找不到有效态；回填由 MP-01 增量 migration 执行，重复执行幂等。**存量已发布 Expert/Team 的 pin 回填见 §5（方案 A）。**

| ID | 方法/路径 | 请求与语义 | 返回 | 所属issue |
|---|---|---|---|---|
| A21 | GET /admin/plugins、GET /admin/plugins/{plugin_id} | 保留分页过滤；版本化资源默认editing/current；**仅Connector详情允许显式view=effective供依赖预览**，且不得用current冒充effective，沿原Admin鉴权；不套用公共依赖隐藏 | 原列表/Detail，**完全省略`published_version_id`（包括null占位）**；Admin保留原范围行（mine 见 A01） | MP-01 |
| A22 | GET /admin/plugins/{plugin_id}/versions/{version_id} | 按 version_id 精确读取/归属检查，Admin鉴权；Connector禁止 | Version；operationId=admin_plugin.version.get | MP-01 |
| A23 | GET /admin/plugins/{plugin_id}/skill_md、/download | 可选version_id，缺省current，**响应必须标出实际返回的version_id**（不能只给标签或暗示latest） | 文本结果/ZIP；仅Skill，原缺省保持 | MP-01 |
| A24 | POST /admin/plugins、PATCH /admin/plugins/{plugin_id} | 仅Skill/Connector完整Write；PATCH必CAS；**relations 不作为必填（空关系）**。Expert/Team全部内容及内容元数据只A25，A24拒绝这两类（400 zip_only_required）。**Admin写入直接生效、无pending review** | Detail（无review_id）；评分/placement走原专用入口 | MP-01底座；MP-02校验 |
| A25 | POST /admin/plugins/import、POST /admin/plugins/container_reupload/{plugin_id} | schema2容器，见§4；强制Idempotency-Key；重传必expected_edit_revision；**直接生效、无pending review**；**写入时冻结缺失 pin、校验显式 pin** | Detail + resolved数组（本次写入的 pin）；Connector不含version_id | MP-02 |
| A26 | POST /admin/plugins/skill_import、POST /admin/plugins/skill_reupload/{plugin_id} | 保留JSON parse_task_id、name/category/tags/version等；重传**必带**expected_edit_revision（新建省略）；**直接生效、无pending review** | Detail；Skill自身上传不因禁止bundled而删除 | MP-01 |
| A27 | GET /admin/plugins/{plugin_id}/dependents | 与A11同包络，Admin可跨Space；operationId=admin_plugin.dependent.list | Dependents | MP-02 |

**A29/A30 本期不实现。** `POST /admin/plugins/legacy_import/preview` 与 `POST /admin/plugins/legacy_import/apply` 不注册路由、不出现在 OpenAPI、不分配客户端、不进入本发布验收；编号保留仅为避免复用。生产迁移一律走 §5 的 CLI + 离线授权。Admin DELETE、rating、分类/场景/投放保留原入口与语义；只给现有删除保护补版本化引用。先前新建的治理及 Admin 上架接口从当前目录撤回，不作为前置任务，不重排其他编号。

## 4. 引用式容器与导入幂等

沿用具体方案schema2字段，不另造容器格式。multipart：file与container_schema_version="2"必填；version/category_id可选，重传expected_edit_revision为必填十进制整数。创建省version默认1.0.0，重传省略沿用current候选标签；内容/关系有变化且原动作直接发布时，复用已release标签409 version_label_conflict，要求明确更高标签，不自动bump。ZIP仅一个根expert.json或squad.json，不允许额外文件（尤其skills ZIP、mcp.json、expert/context.json）。两类公共字段如下：

schema2压缩multipart请求继续使用部署MaxArchiveBytes()（MAX_UPLOAD_MB）上限；解压仅一个清单且≤1MiB，最终AGENTS.md同样≤1MiB。保留路径穿越、重复路径、软链接和解压预算检查；不因ZIP只有引用而跳过归档安全。§5 的 CLI preview 处理旧多文件包时沿原归档上限和完整文件校验，不误套schema2单文件限制。

| 字段 | schema2规则 |
|---|---|
| schema_version / kind | 必须分别为字符串"2"及agent/squad，与文件名和multipart一致 |
| name | 必填，trim后非空且UTF-8≤128字节 |
| summary / description | 至少一个trim后非空，优先summary，否则既有description别名；最终摘要UTF-8≤512字节，不截断 |
| tags | 可选字符串数组，trim、去空及去重后≤20项；缺省[]，不是页面覆盖字段 |
| instruction | 两类均必填非空UTF-8文本，≤1MiB；生成该来源唯一AGENTS.md，Team不能沿用旧leader/strategies渲染器丢掉instruction |
| skills / connectors | 仅agent，显式数组各≤20；skills每项仅`{id,version_id?}`、connectors每项仅`{id}`；`version_id`为可选显式 pin（提供时须属于目标且有release，冻结后写入快照；省略由 Admin 写入动作解析为写入时目标当前正式发布版），Connector永不带版本 |
| members | 仅squad，1～30，每项仅`{id,version_id?,member_key,role,is_leader,sort_order?}`；`version_id`同为可选显式 pin（Team pin 的成员 Expert 版本）；sort_order缺省数组下标，其他wiring遵守§2.1及技术§5.1 |

未知字段、重复JSON键、内联能力及旧Team的leader/strategies/dependencies/permission均拒绝；旧结构返回400 legacy_container_format引导显式迁移，不静默丢弃。明确是schema2但字段错误使用400 VALIDATION_ERROR并指清单路径，不因严格解码重写原始指令字节。

```json
{"schema_version":"2","kind":"agent","name":"示例专家","summary":"引用市场Skill的专家","instruction":"示例指令","skills":[{"id":"skill-id"}],"connectors":[{"id":"connector-id"}]}
```

```json
{"schema_version":"2","kind":"squad","name":"示例团队","summary":"引用市场专家的团队","instruction":"团队指令","members":[{"id":"expert-id","member_key":"lead","role":"负责人","is_leader":true,"sort_order":0}]}
```

Expert的skills/connectors显式数组可为空；Team members为1～30。ZIP用id是原具体方案的领域格式，HTTP Relation仍target_plugin_id；只有容器parser负责转换，不要求改通用Lib。`version_id` 若给出是**显式 pin**：**不进入 content_digest**（发布时补齐导致摘要稳定），但写入已发布快照并成为读取/安装依据；省略时由写入动作解析为写入时目标当前正式发布版；不属于目标或未发布才 400 `dependency_version_mismatch`。重传按relation_type+target_plugin_id保留relation_id，关系集合/data/角色/Leader/顺序/**显式 pin**变化才生成新版本（`content_digest` 仍不含 pin）；目标发新版不计入。不能接受UI的relations/manifest覆盖字段。

A25返回`{data:{plugin,relations,relation_result?,resolved:ResolvedDependency[]}}`，Detail基础字段保持不变。`ResolvedDependency={source_plugin_id,relation_id,relation_type,target_plugin_id,target_version_id?}`，每条最终直接关系恰好一行，按sort_order/relation_id稳定排列，无依赖为[]；`target_version_id` 是**本次写入冻结的 pin**，Connector完全省略；它同时写入关系快照，是后续读取/安装唯一依据。不内含目标名称/配置或审核观察值，不把resolved误作完整图；Admin名称/正文预览使用A21有效Connector或A22 Version。

导入key唯一域为actor＋operation＋target_plugin_id（新建固定new）＋key，摘要覆盖原ZIP及schema/version/category/CAS。先核验当前Admin权限、查已提交receipt，再执行首次CAS/依赖校验；同key同输入回放原Detail+resolved（resolved按首次写入的 pin 原样回放，不因目标后来发新版重算），不因自己的提交已推进修订而重做旧CAS。不同输入409，不能再创建或补当前内容。receipt与结果同事务，7天完整回放后保留tombstone拒绝复用；对象先校验上传，失败只清理本次新建且无引用对象。

同 key 回放返回首次导入时冻结的 `Detail + resolved[]`；它只用于幂等重试，**不是目标最新版预览**。需要查看目标当前正式发布版时，使用 A03/A10 重新读取（或新 key/新导入），**不得用 A25 同 key 回放刷新 pin/latest**；`resolved[]` 也不写回关系或作为后续读取/安装依据。A25 为 Admin 动作，直接生效、不创建 pending review。

## 5. 生产迁移：CLI + 离线授权（不做 A29/A30 HTTP）

旧市场资产内嵌的 Skill 及 Expert（含专家团成员及其内嵌 Skill）拆成独立、市场可见资产，原依赖方改为目标 `plugin_id` 引用，并在新版本发布/提交时按 P 冻结 `target_version_id` pin；**存量已发布 Expert/Team 在 cutover 做一次 pin 回填（方案 A）**。按所属市场来源的 owner 及既定 system/Space 受众、现有发布审核和列表规则展示，不能仅去掉 embedded 关系却继续隐藏目标或创建 private 依赖。**生产执行必须由持有离线授权的操作者批准冻结计划；CLI 不是绕过授权。** 不新增生命周期规则或 HTTP API。

**存量 pin 回填（方案 A）：** 生产切换时，迁移 CLI 对现有已发布 Expert/Team 执行一次 pin 回填：已有合法 `target_version_id` 且该版本仍正式 release → 保留；缺失/无效 → 冻结为回填时目标当前正式发布版；找不到可冻结版本 → 该根 `blocked`，不静默取最新/旧版。回填在离线授权/冻结计划下执行，产出 `plan_digest`；不得在普通读取时懒回填；回填不自动重新审核，下一次用户重新发布按 P 重新冻结。

**传输方式（D5）：** 本期生产迁移使用 `marketplace-stock-migration` CLI + 离线授权，**不实现 A29/A30 HTTP**。CLI 与领域服务复用同一转换/校验实现及同一计划 schema；A29/A30 编号保留但不注册路由、不分配客户端。任何自动 apply、猜测映射或绕过批准的路径都不得作为实现捷径。

**旧入口退役（D10）：** 现基线中的 `cmd/plugin-backfill`，以及 `marketplace-stock-migration` 旧模式（`--all-spaces --apply`、`--rollback` 指针回滚、直接 INSERT release、`--g4...` 自证批准、固定 pin；`internal/backfill/stockmigration` 与 `plugin_stock_migration_batches/roots` 同属旧路径），必须在本期删除/禁用，不得与新 apply 并存；`plugin_stock_migration_batches/roots` 作为 batch/root 审计底座复用/演进，不再保留第二套 run/写入口，具体增量 DDL 由 MP-04 提交。新 CLI 只接受 §5.1 的离线授权制品；旧 CLI 的 DB 账号/DSN/权限要撤销，旧二进制要 deny/移除，runbook 同步更新。REL-01 需验证 `plugin-backfill` 与 `marketplace-stock-migration` 旧 apply 两条入口均不可达；仅网关/网络隔离不能作为证据。

**来源身份：** 数据库来源为 `plugin_id@version_id`，并核对 `expected_edit_revision` CAS；ZIP 来源为原始字节 SHA-256（CLI 支持 ZIP 时必须显式，本期可只支持数据库来源，ZIP 另行授权）。不得凭名称/内容相似自动选目标，也不得把**未校验的旧 pin** 当权威；存量回填校验通过后才成为 pin。

**两次只读 preview（冻结摘要）：**

1. `preview --discover`（discovery）：完整枚举来源 inventory、建议候选、关系及缺失选择；只读，不写业务对象、审核、receipt、映射或 OSS；`can_apply=false`，不把建议或唯一候选当批准。
2. `preview --freeze --proposed-plan <file>`（frozen）：操作者补全显式标签、映射、Connector server 分组、归属/受众后再次 preview；preview 可建议 `patch+1` 标签，但必须写入冻结计划并由人批准，apply 不自行换标签，标签冲突重新 preview。CLI 重取同一来源并核验完整映射、拓扑、标签、归属/受众及文件，输出冻结 `plan.json` 与 `plan_digest`；任何无法确认项仍 discovery、不可 apply。修改任意映射/分组/范围/关系/标签必须重新 freeze，**不允许手改冻结计划后沿用摘要**。

两次 preview 均只读数据库与对象存储；`MigrationPreview` 与 `MigrationPlan` 是 CLI 输出/输入文件格式，不是 HTTP 端点。

`MigrationPreview={phase:"discovery"|"frozen",input_digest,source_kind,source_identity,source_scope,source_plugin_id?,source_version_id?,expected_edit_revision?,resources:DiscoveryResource[],relations:MigrationPlanRelation[],exceptions:[{source_node_key,reason,can_retry}],can_apply,plan?:MigrationPlan,plan_digest?:string}`。只有frozen可同时给plan/plan_digest及can_apply=true。DiscoveryResource为`{source_node_key,plugin_type,source_content_digest,plugin_name,proposed_owner_id,proposed_space_id,proposed_visibility,suggested_targets:[{plugin_id,version_id?,version_label?}],mcp_server_keys?}`；候选均经授权，Connector不自动给候选或版本，最多输出server身份/脱敏descriptor fingerprint，不含配置秘密。

`MigrationPlan={schema_version:"1.0",input_digest,source_kind,source_identity,source_scope,source_plugin_id?,source_version_id?,expected_edit_revision?,resources:MigrationPlanResource[],relations:MigrationPlanRelation[]}`。冻结plan使用以下完整字段，后续执行不能再凭当前名称/内容猜映射：

source_kind仅`plugin_version|zip`；数据库来源必须source_plugin_id/source_version_id/expected_edit_revision，ZIP必须省略。root的source_node_key和origin_node_key均为root；proposed_plan使用同一MigrationPlan字段形状，不含plan_digest/approval_reason，缺选择不得冻结。冻结前把mcp_server_keys按UTF-8字节序排序，资源及关系使用下述稳定排序。

| 类型/字段 | 必需性与执行规则 |
|---|---|
| MigrationPlanResource | `{source_node_key,origin_node_key,plugin_type,source_content_digest,action,owner_id,space_id,visibility,category_id?,version_label?,target_plugin_id?,target_version_id?,mcp_server_keys?,target_effective_revision?,target_effective_digest?}`；space_id允许null，其余省略规则如下 |
| source_node_key / origin_node_key | root恰好一个；其余为来源inventory的稳定节点。非Connector二者相同且每源节点恰好覆盖一次。合法MCP拆组可多条独立source_node_key指向同一origin_node_key，各组server key须无重复且完整覆盖原集合；除此不漏节点、不加图外能力 |
| action=create | 创建独立目标，省略target IDs。Skill/Expert/Team必明确新version_label；Connector省略版本标签/ID。源内容、名称、指令和文件由冻结origin输入确定，category_id显式给合法分类或null；不能用只有fingerprint的计划还原配置 |
| action=reuse | 必target_plugin_id；**不要求也不写死依赖版本**：迁移生成的新版本在发布时按 P 冻结 pin；`target_version_id` 若给出是**显式 pin**，apply 必须校验其仍为正式 release，否则 `plan_stale`/`blocked`。省略新version_label/category_id，不修改目标内容/归属。Connector省略target_version_id，记录preview时的target_effective_revision/digest作为批准观察值，apply变化则plan_stale，不把它们写成关系pin |
| action=update_source | 仅数据库来源的root，target_plugin_id必须等于source_plugin_id，省略target_version_id，明确新version_label。生成同一Plugin的新引用式快照，不创建第二个根，不覆盖历史；ZIP根用create。禁止子节点以该动作改写共享目标 |
| owner/Space/visibility | 保持所属市场来源及已确认system/Space受众，不能靠人工plan扩大受众或改变owner；ZIP根采用已鉴权Admin允许的新建范围。每次由服务端核验，Admin可见不豁免依赖矩阵，private历史不能因本次确认被自动公开 |
| MigrationPlanRelation | `{source_node_key,target_node_key,relation_type,sort_order,data}`；两端必须在resources，类型/wiring依统一关系契约。父边引用create/update_source输出时从提交map取得真实Plugin ID（显式 pin 可选）；reuse按target_plugin_id引用，Connector只ID。完整保留来源成员/Skill关系及原角色/Leader/顺序，禁止自动补latest、猜测映射或丢边 |
| mcp_server_keys | 仅Connector必填非空，明确来自origin的server集合；默认整组，不按URL/名称猜映射。批准拆组必须完整覆盖、无重复/额外能力，所有配置从同一已授权输入取得，秘密/来源不明先阻断 |

复用Expert时，其实际关系必须与计划中对应子节点的目标ID及Connector身份一致；**依赖版本按 P 在发布时冻结，计划显式给出的 pin 必须校验**。对应子节点仍为create且真实ID未知时，不能确认复用该Expert，可由操作者选择精确复用其既有子目标或新建Expert。不能为了匹配计划覆写共享Expert，也不把同内容不同目标身份偷偷合并。

摘要统一由服务端/CLI计算，客户端按opaque值回传：ZIP的input_digest与source_identity均为原ZIP字节SHA-256；数据库的source_identity=plugin_id@source_version_id，input_digest覆盖源图完整快照、文件原字节摘要及原归属。source_content_digest标识该源节点的原始内容，不冒充尚未分配目标ID的新Version摘要。source_scope=SHA-256(JCS({owner_id,space_id,visibility}))，取root并由服务端核验。plan_digest=SHA-256(JCS(MigrationPlan))小写hex；resources按source_node_key，relations按source_node_key/relation_type/sort_order/target_node_key排序，字符串用UTF-8字节序、序号用整数。摘要绑定每个新建标签、映射、server分组及关系；正式Version的content_digest在实际ID绑定后由唯一版本codec生成。**字段命名：** 迁移计划/预览中的 `input_digest` 是唯一来源摘要；issue 或旧 CLI 中的 `source_digest` 只是同一值的旧名，必须映射为 `input_digest`，不得并存两套字段或两套计算（旧字段移除方式实现时确认）。

**apply（只认冻结计划）：** `apply --plan <plan.json> --plan-digest <digest> --authorization <authorization.json>` 必须同时满足：计划为frozen、摘要与重算一致、来源/归属/权限/输入未变、操作者持有离线授权。CLI只执行冻结计划，不自行换标签、不重选目标、不猜映射；摘要或输入变化返回 `plan_stale`，要求重新 preview/批准。数据库root首次切换还须CAS原source_version_id/expected_edit_revision；ZIP不伪造这两项。actor来自认证/运行记录，审计不能信任手工`approved_by`。**存量 pin 回填（方案 A）作为 cutover 计划的一次性步骤：在放通新读取前完成，产物写入 plan/run 审计；回填失败的根 `blocked`，不得进入普通读取；不得在读取时懒回填。** **CLI 失败语义不使用 HTTP 409，也不新增 HTTP 错误码：`plan_stale`、`authorization_invalid` 等 reason 返回非零退出码，已有 run 保持 `blocked`，尚未开始的失败不创建业务对象；只有 HTTP 端点使用通用错误包络。apply/resume 不接受 `Idempotency-Key`，也不得引入未定义的迁移幂等键；run 唯一性只由 `(source_scope, source_identity, plan_digest, source_node_key=root)` 仲裁，apply 与 resume 共用该身份。**

apply按 Skill/Connector → Expert → Team 的拓扑执行；root用`update_source`保持原Plugin ID，子节点create/reuse。新版本走现有发布/审核：Space默认自动审核可自动批准；人工模式为 `waiting_review`，不直接插入 release 绕过审核。`MigrationRun={run_id,status:"completed"|"waiting_review"|"blocked",plan_digest,mappings:[{source_node_key,target_plugin_id?,target_version_id?,step_state}],exceptions:[{source_node_key,reason}]}`；step_state仅`reserved|created|waiting_review|published|source_switched|blocked`，未提交target省略；**Expert/Team 的 `target_version_id` 是该关系发布时冻结的 pin，Connector 永无target_version_id**。全部节点/来源切换、发布和**独立列表资格**核验就绪才`completed`；等待审核为`waiting_review`；已部分提交后失败为`blocked`，保留run/mapping并给出安全原因，不假报整体完成。迁移后的Skill/Expert必须在对应市场独立可发现；只有全部节点发布/列表资格就绪才completed。

**新建 Connector 的生效与审核：** 迁移 plan 中的 Connector 节点按来源范围沿用既有两条路径：system 来源创建 system Connector，走 Admin 既有直接生效写入并同步 `plugin_connector_effective`，不创建 pending review；Tenant 来源创建 Tenant Connector，必须走既有 Connector 审核，apply 后 run 为 `waiting_review`，批准前没有 effective 行、不能被 Expert/Team 解析。CLI 不得直接写 `plugin_connector_effective` 或 release 绕过审核；迁移操作者的 system 资格及由谁触发直接生效**实现时确认**，不得新增审核角色或旁路。

**可恢复与幂等：** run以`(source_scope,source_identity,plan_digest,source_node_key=root)`唯一保留；同一冻结计划重跑恢复原run，不重复创建Version/提审，也不重做最初CAS。map的`source_commit_json={output_version_id,output_edit_revision,review_id?,frozen_content_digest,frozen_pins}`在来源切换时同事务记录，标识自己的已提交输出及该版本 pin 图。后续current新编辑不覆盖、不自动按旧计划重新套用；原冻结审核正常批准且输出已正式发布时可完成，不能因current已有新草稿误判失败；审核拒绝/取消保持blocked。新计划或输入改变必须重新preview+批准，不借旧map跨范围复用；恢复重验原review/release/effective及全部目标资格，不自动重提已拒绝审核。迁移不承诺多个目标/来源一个大事务，也不是`/_batch`。

**持久化：** 冻结 `plan.json`/`plan_digest`、授权绑定（`authorization_id`→batch/run/plan_digest）、`MigrationRun`、`plugin_dependency_migration_map`、`source_commit_json` 及授权更替审计必须持久化；计划表名为 `plugin_dependency_migration_runs`、`plugin_migration_authorizations`（与 plan.json/#134 一致；若实现改名，两处同步），不得只有本地文件。**归属：MP-04 拥有 run/frozen plan/map/authorization binding 的增量 DDL；MP-01 提供既有 release/refs/attempt 底座，不重复建表。** `resume --run-id` 只从该持久化存储读取原计划摘要/步骤，不接受操作者另交 plan 覆盖；缺少原计划/授权绑定时安全失败为 `blocked`，不能重新 apply 猜测。**字段别名：** `source_node_key=root` 与 issue/plan 中的 `root_node_key` 是同一节点键；本文统一用 `source_node_key=root`，实现/文档不得同时保留两套键。

**回滚（数据安全，D5/D10）：**

- 不删表、不删文件、不覆盖旧快照/历史版本；
- **不把 `current_version_id`／`published_version_id` 改回旧值**，不做指针回切；
- 需要停止新分发时用下架或部署隔离，不用指针回切；
- 内容恢复从旧历史生成新版本（forward-fix），走现有发布/审核；
- 保留 batch/mapping/attempt/audit，回滚动作可审计；
- 不恢复 legacy 执行器/旧入口，不重新产生 embedded/inline；回滚接受兼容二进制 + 新版本恢复（D10）。

以下为数据库Expert根更新、内嵌Skill新建的冻结`MigrationPlan`示例（CLI文件组件，不是HTTP包络）。来源摘要使用文档fixture，并非真实生产数据；按上述排序/JCS可核验`plan_digest=d1bedf7b9724c941503338ebf94bdee85bc2eb1610ee43b983f6b2755e9538f2`：

```json
{
  "schema_version": "1.0",
  "input_digest": "8899750166a83bdf613df738f61d617028a9e3e1f802e84d59ded2a882f64c97",
  "source_kind": "plugin_version",
  "source_identity": "demo-expert@00000000-0000-4000-8000-000000000101",
  "source_scope": "068efbd59f64782094bec61e08e0633f63b7aea760da24c27271c1e9fc1ed52c",
  "source_plugin_id": "demo-expert",
  "source_version_id": "00000000-0000-4000-8000-000000000101",
  "expected_edit_revision": 1,
  "resources": [
    {"source_node_key":"root","origin_node_key":"root","plugin_type":"expert","source_content_digest":"45e59bae59bf2f21a5aa71a7d0015e9c638d3f3e0369143809c62b0f4b592f6f","action":"update_source","owner_id":"demo-owner","space_id":"demo-space","visibility":"space","version_label":"2.0.0","target_plugin_id":"demo-expert"},
    {"source_node_key":"root/skills/0","origin_node_key":"root/skills/0","plugin_type":"skill","source_content_digest":"c2131fff66b00a0b446a94ebae9d11b59f28d7c113ad2878041bed0429150461","action":"create","owner_id":"demo-owner","space_id":"demo-space","visibility":"space","category_id":null,"version_label":"1.0.0"}
  ],
  "relations": [{"source_node_key":"root","target_node_key":"root/skills/0","relation_type":"expert_skill","sort_order":0,"data":{}}]
}
```

MP-04 负责 CLI、冻结计划/摘要、拓扑创建/复用、run 恢复、独立可见核验与数据安全回滚；MP-02 的 receipt 底座提供 operation 扩展。生产 apply 仍须离线授权；CLI 不得自动执行生产迁移。

### 5.1 离线授权制品与恢复契约

`apply --plan <plan.json> --plan-digest <digest> --authorization <authorization.json>`；`resume --run-id <run_id> --authorization <authorization.json>` 使用同一校验器。授权文件为 `{payload,signature}`，signature 为 Ed25519 对 UTF-8 JCS(payload) 的签名（base64url、无 padding）；不接受文件自带公钥或算法覆盖。payload 必需字段为 `{schema_version:"1.0",authorization_id,key_id,issuer_id,actor_id,environment_id,source_scope,source_identity,plan_digest,batch_id,issued_at,expires_at}`；ID/范围非空，时间为 UTC RFC3339，issued_at ≤ 当前时间 < expires_at；期限由签发者明确填写，无无限期默认。

可信配置由部署管理提供：environment_id、key_id→公钥/issuer_id、签发者可授权范围和启停状态、撤销的 key_id/authorization_id；不从 plan、授权附件或操作者临时参数导入信任。actor_id 必须匹配执行环境已认证主体，不能由 `--actor` 或手写 approved_by 声明；不能可靠解析主体或加载信任配置时拒绝写入。逐次验证签名、环境、范围、计划摘要、来源身份、操作者及有效期，不因是 Admin 跳过迁移执行授权。

authorization_id 唯一绑定一个批次和原 run（同冻结计划恢复返回同一 run），首次占用与 run 关联原子提交；不能用于其他环境、范围、计划或创建第二个 run。同 run 恢复仍重验当前权限、授权有效期及撤销状态，不重做已提交步骤。过期/撤销后暂停写入并保留原 run/map；重新签发的新 authorization_id 仅在 actor/environment/source/plan_digest/batch 完全匹配时绑定原 run，记录授权更替审计，不能创建替代 run。修改计划必须重新 freeze/批准，不能用换授权掩盖变化。

轮换时新增可信 key_id 并停用旧钥签发；历史公钥保留供审计，执行是否接受由启停/撤销状态决定。授权失效或不匹配使用 CLI reason `authorization_invalid` 并非零退出，既有 run 保持 blocked；未开始不得创建业务对象。日志仅记授权 ID、签发者、主体、计划摘要和安全 reason，不含私钥/凭据。CLI reason 不新增 HTTP 顶层错误枚举（R2）。只读 inventory/preview/verify 不因持有授权而获得额外读权限。

## 6. 安装与窗口内安全重试（客户端 → Marketplace）

| ID | 方法/路径 | 请求 | 返回 | 所属issue |
|---|---|---|---|---|
| A31 | POST /plugins/{plugin_id}/installations | 单个X-Workspace-Id、必填Idempotency-Key；JSON `{runtime_id,resource_name?,custom_env?}` | 200 `{agent_id}`或`{squad_id}`，保留现有命名 | MP-03 |
| A32 | POST /plugins/install | 原JSON `{plugin_id,workspace_id,runtime_id,custom_env?}`；必填Idempotency-Key，workspace_id必须UUID | 与原成功结果一致，调用同一原子引擎，不走legacy create/bind/delete | MP-03 |
| A34 | GET /plugins/{plugin_id}/distribution | path ID＋既有认证/Space上下文；无body，无Workspace/runtime/custom_env/幂等键，不接受根version覆盖 | 200 `{data:DistributionBundle}`；按源发布快照 **pin 图**返回完整原始内容，无分页/后续内容请求；operationId=plugin.distribution.get | MP-03 |

**A31/A32 都必须使用用户身份（拒绝 Bot）/Space 上下文、必填 `Idempotency-Key` 和单个 `X-Workspace-Id` UUID，共用同一个 `InstallCoordinator`；A32 只保留 URL，不保留 legacy create/bind/delete 行为。** A31/A32输入归一化后相同唯一域；不能同一key通过换URL创建另一attempt。A32 的 Workspace header 与 body 的 `workspace_id` 均必填且必须相等，否则400。runtime_id UUID；resource_name最多128字符；custom_env为map<string,string>，最多100项、key最多128字节、单值32KiB、合计256KiB。**`Idempotency-Key` 必须是单个 header、1～200 个可见 ASCII 字符（不接受重复/多值或空白）；缺失/非法 400。** 安装JSON沿用3MiB上限，无压缩body、禁止redirect；最终Fleet请求16MiB上限。

安装attempt只用于Marketplace本地关联与重试控制，不新增安装结果查询接口。**首次构建plan时冻结一次请求可见的完整结果：当前源发布快照的根版本、该快照的完整 pin 图、Connector effective、物化字节、命名、runtime和bindings**；记录首次可能发出Fleet请求的时间及不晚于其后24h的retry_until。attempt状态为`prepared|dispatching|succeeded|failed|unknown|expired`（EXPIRED/UNKNOWN 为明确终态，不与其他结果混同）：仅`prepared`及窗口内`unknown`可按原plan重试；`succeeded`回放；`failed`不自动重试；超窗`expired`。已有attempt先按原用户/Space/Workspace及原输入核对，再决定是否需要构建首次plan；**重试不得因目标发新版、Connector或其他live内容变化重建plan、重新读取源发布快照、重选版本/pin或修改出站请求**；仅这些正常发布变化不触发 409，原 key 继续原 plan；显式输入变化、pin 节点下架/删除或权限失效按下文分别处理。

调用者须重新提供与首次相同的运行时Secret；Marketplace结合原plan重建完整请求，校验包含Secret的完整规范化请求一致后，才可在现有24h窗口内以原Fleet幂等键重试F01。Secret只在处理请求的内存中使用，不写入客户端本地记录、Marketplace attempt/plan/日志。仅有幂等键或不含Secret的plan不足以重发；应用重启后若无法重建原参数，不自动POST，也不承诺无凭据自动恢复。

已知SUCCEEDED先验证原caller/Space/Workspace访问及`intent_digest`（归一化root/Space/Workspace/runtime/resource_name，不含custom_env），可回放本地验证结果，不要求根仍上架或重填Secret；若本次显式给custom_env仍校验完整input_hmac。不同非Secret意图即使同key也409 idempotency_input_mismatch，不能回放另一根/Runtime的结果。回放是历史提交结果，不保证资源仍存在。无本地记录、后续4xx或资源名称均不能证明此前未提交。

**同 key、配置与依赖变化（用户本轮确认）：**

- 同 key 同输入网络重试只重放首次冻结 plan；上游根、Skill、成员 Expert 发新版或 Connector 正常更新 effective，不阻断重试、不重选版本/pin。
- 本次显式输入（root/Workspace/runtime/resource_name/custom_env 等）与首次不一致 → 409 `idempotency_input_mismatch`，不重建 plan。**不得把市场依赖/pin/配置变化写成 `idempotency_input_mismatch`；它只表示显式输入不同。**
- 原 plan 的市场配置/依赖资格变化（根或 pin 节点下架/删除、Connector effective 失效、权限失效等）返回 409 `dependency_changed`（权限前置仍按 401/403/404），不重建 plan、不重选版本；错误不是先前安装未提交的证明，已存在 UNKNOWN 时保留未知结果和原 attempt。
- 想使用源发布快照的更新 pin 必须新 key 新安装，首次按当时当前源发布快照冻结计划。新 key 不用于自动绕过未知旧结果，旧结果先核对。
- 审核窗口变化仍按 §2.3 返回 `dependency_changed`，与正常上游发新版后的原安装重试分开处理。
- **安装计数只在首次 `succeeded` 时增加一次**；同 key 回放、UNKNOWN、failed 均不计数。

POST准备失败按§8错误；dispatching竞争409 installation_in_progress；上游结果未知503 installation_result_unknown，details仅含安全attempt_id/retry_until/can_retry_same_key，不含Secret或原响应。can_retry_same_key只表示仍有窗口且能重建同完整请求，不承诺查询。超窗返回409 installation_attempt_expired，无法重建或输入不同时不重发。后续4xx或资源状态变化不能证明先前UNKNOWN未提交，必须保留未知并提示人工核对，不换key、删记录或fallback。POST成功带Cache-Control:no-store，只有明确回放证据才带Idempotency-Replayed:true。

### 6.1 A34：单请求完整专家与依赖内容（P 语义）

网关URL为`GET /market/api/v1/plugins/{plugin_id}/distribution`。此处的distribution仅指完整内容读取，不是安装、Fleet结果查询或市场运营动作。沿用现有用户/Bot读取鉴权和Space上下文；认证Token仍必需，但不要求运行时凭据。请求只有path plugin_id，无body，不要求Workspace/runtime/custom_env/Idempotency-Key，不提供历史根版本选择器或editing入口。

在同一个短只读一致性事务内，选择根当前正式发布版或有效 Connector，**按该根快照的 pin 读取全部版本化依赖**（Expert 根读 Skill pins；Team 根读成员 Expert pins，再读该 pinned Expert 版本的 Skill pins），以及全部 Connector 完整 effective 行（含revision/digest/payload）复制到内存，核对投影及完整有效内容摘要后退出事务，再读不可变OSS并校验。不得先记旧revision后晚读新配置。根或任一 pin 依赖不可用时整图失败，**不回退/不切换到其他版本**；根与依赖必须当前可发布/可访问且通过既有访问/上下架及历史受众检查。**A34 每次都按 pin 图读取；同一源版本 + 同一组 pin 返回相同依赖内容与相同 `bundle_digest`，目标发新版不影响；只有源版本重新发布导致 pin 图变化才不同。** 依赖不可用时 owner 只能在 A01 mine 看到诊断，不能通过 A34/安装绕过。Skill/Connector单根沿用现有本地入口，不新增产品操作。

`DistributionBundle={schema_version:"1.0",root:{plugin_id,plugin_type,plugin_name,selected_version_id?,effective_revision?},graph:Graph,assets:ContentAsset[],bundle_digest:string}`，包装在通用`data`中：

| 字段/类型 | 必需性及不变量 |
|---|---|
| root | 版本化根必有selected_version_id（根当前正式发布版）；Connector根必有effective_revision且省略selected_version_id；与graph.root_node_key对应节点一致 |
| graph | 根快照的完整 **pin 图**与角色/Leader/顺序/绑定关系；节点按资源＋pin 版本去重，Connector按ID＋有效修订区分，不按名称合并；同一依赖插件只出现其 pin 的一个版本 |
| assets | 每个graph node_key恰好一项，无遗漏、重复或图外项；`ContentAsset={node_key,market_name,files:ContentFile[],mcp_config?}` |
| ContentFile | `{path,mime_type,content,content_hash}`全部必填；path为该节点内的安全相对路径，content为完整原始UTF-8文本，content_hash为原字节SHA-256小写hex，不归一化换行 |
| files | Expert/Team仅自身AGENTS.md；Skill含SKILL.md及全部支持文件，完全对应graph文件清单的路径/mime/大小/hash。Connector固定[]，不额外返回mcp.json或缺清单文件 |
| mcp_config | 仅Connector必填，对应本次选中有效修订的完整公开声明/Secret占位符；其他节点省略，不含运行时秘密、不读Expert旧附件。服务端按完整effective payload验证effective_digest；客户端不从仅mcp_config伪重算整行摘要 |
| market_name | 必须等于对应graph节点plugin_name，来自 **pin 的 released snapshot/effective** 投影，不读 current/其他 owner editing name，不用安装副本名替代 |
| bundle_digest | 排除自身后完整data按RFC8785 JCS计算SHA-256小写hex，无sha256:前缀。先将nodes/assets按node_key、edges按source_node_key/relation_type/sort_order/relation_id、files按path排序；字符串比较用UTF-8字节字典序，sort_order用整数比较。再执行JCS（对象key按其UTF-16规则），不依赖Go/JS默认JSON编码。**同一源版本 + 同一组 pin 的 digest 稳定；目标发新版或重新 GET 不改变它。** |

同一节点内容只在assets存一份，通过node_key与graph关联；同一依赖插件在一次响应中只有其 pin 的一个版本，不会同时出现其他版本。`bundle_digest` 对同一源版本 + 同一组 pin 稳定，可用于校验该冻结图；源版本重新发布后才是新 digest。 A34 不生成 `installation_name`、不改写 `SKILL.md` frontmatter、不产安装副本，返回市场原名/原字节；读取成功不代表 Fleet 安装或同名共存成功。

读取复用 pin 解析 resolver、版本文件及对象完整性校验，不在服务端环回调用A03/A05/A08拼HTTP。现有capability_definition.go及capability_assets.go可复用文件读取/预算/校验能力，但先拆出不依赖Workspace/runtime/Secret的内容解析，不能原样暴露按ID/current或旧inline MCP的安装builder。远端安装在读取结果之后另行序列化为Fleet请求。

全量成功才200：校验所有节点/文件、无重复安全路径、UTF-8和摘要、有效配置及既有预算后才能发送成功。JSON响应总量最多16MiB（包含包络、元数据及转义后的UTF-8编码），沿用单文本1MiB、Skill支持文件50/全图500及图预算；超限413 budget_exceeded，不能分页、截断、跳文件或返回下载链接让客户端补齐。**错误矩阵固定：根或目标不可见 → 404（按既有安全隐匿，不泄露存在性）；路径/类型/二进制不合法 → 400；pin 依赖未发布/下架/删除/状态不合法 → 409 `dependency_unavailable` 或 `dependency_changed`；图/文件预算 → 413 `budget_exceeded`；文件或索引完整性故障 → 500 `artifact_integrity`/`relation_index_inconsistent`。不返回残缺图后200，不补latest、不回退/不切换到其他版本，也不因错误改用其他读取接口。**

返回`Cache-Control:no-store`。**服务端不得跨请求缓存 pin 图或 `bundle_digest`，客户端也不得跨请求/会话缓存 A34 响应、assets 或 digest；每次打开、刷新、切换目标或重试都重新发起 A34 GET，失败时不得回退到会话中的旧 bundle。** 本GET不调用Fleet、不创建安装attempt/业务资源、不改变市场发布状态；查询成功不能作为安装成功或共存支持证据。Buddy/Web/CLI完整内容入口一次A34取得全部数据，客户端只校验和消费，不再调用依赖detail/versions/skill_md/download/files补内容，不在A34失败后改用其他接口拼装。

以下是专家＋依赖Skill的完整示例。Skill 的版本是**该 Expert 发布快照 pin 的版本**，不是目标最新版；只要源版本与 pin 不变，重复请求返回相同图与 `bundle_digest`，只有 Expert 重新发布导致 pin 变化才不同。文件摘要与`bundle_digest`按上述规则可计算校验；Connector及Team按类型规则扩展，不是线上响应实测：

```json
{
  "data": {
    "schema_version": "1.0",
    "root": {"plugin_id":"demo-expert","plugin_type":"expert","plugin_name":"demo-expert","selected_version_id":"00000000-0000-4000-8000-000000000101"},
    "graph": {
      "root_node_key": "demo-expert@00000000-0000-4000-8000-000000000101",
      "nodes": [
        {"node_key":"demo-expert@00000000-0000-4000-8000-000000000101","plugin_id":"demo-expert","version_id":"00000000-0000-4000-8000-000000000101","plugin_type":"expert","version_label":"1.0.0","plugin_name":"demo-expert","manifest_json":{"$schema":"cowork-plugin-manifest-2.0.json","plugin_name":"demo-expert","plugin_type":"expert","name":"demo-expert","description":"Demonstration expert","labels":[]},"files":[{"path":"AGENTS.md","mime_type":"text/markdown","content_size":59,"content_hash":"9a9f93c879ba2ccf6c23c9354ffde2bd709dc71121c9e74dfef749dba9cab329"}]},
        {"node_key":"demo-skill@00000000-0000-4000-8000-000000000102","plugin_id":"demo-skill","version_id":"00000000-0000-4000-8000-000000000102","plugin_type":"skill","version_label":"1.0.0","plugin_name":"demo-skill","manifest_json":{"$schema":"cowork-plugin-manifest-2.0.json","plugin_name":"demo-skill","plugin_type":"skill","name":"demo-skill","description":"Demonstration skill","labels":[]},"files":[{"path":"SKILL.md","mime_type":"text/markdown","content_size":99,"content_hash":"7d33231a4649f18dd3c82f6b3ba980fce8e89e1dda636f49bbc708681b912128"},{"path":"references/guide.txt","mime_type":"text/plain","content_size":49,"content_hash":"1bbb3c80d153d9a5d810f64ed30ae9873b6670108fd2467981236d9215255269"}]}
      ],
      "edges": [
        {"relation_id":"demo-relation","source_node_key":"demo-expert@00000000-0000-4000-8000-000000000101","target_node_key":"demo-skill@00000000-0000-4000-8000-000000000102","source_version_id":"00000000-0000-4000-8000-000000000101","target_plugin_id":"demo-skill","target_version_id":"00000000-0000-4000-8000-000000000102","relation_type":"expert_skill","sort_order":0,"data":{}}
      ]
    },
    "assets": [
      {"node_key":"demo-expert@00000000-0000-4000-8000-000000000101","market_name":"demo-expert","files":[{"path":"AGENTS.md","mime_type":"text/markdown","content":"# Demo expert\nUse the pinned skill and its reference file.\n","content_hash":"9a9f93c879ba2ccf6c23c9354ffde2bd709dc71121c9e74dfef749dba9cab329"}]},
      {"node_key":"demo-skill@00000000-0000-4000-8000-000000000102","market_name":"demo-skill","files":[{"path":"SKILL.md","mime_type":"text/markdown","content":"---\nname: demo-skill\ndescription: Demonstration skill\n---\nRead references/guide.txt before acting.\n","content_hash":"7d33231a4649f18dd3c82f6b3ba980fce8e89e1dda636f49bbc708681b912128"},{"path":"references/guide.txt","mime_type":"text/plain","content":"This reference belongs to the pinned demo-skill.\n","content_hash":"1bbb3c80d153d9a5d810f64ed30ae9873b6670108fd2467981236d9215255269"}]}
    ],
    "bundle_digest": "63e4e7d2012d3407262301bbaf182dad4007f74c02cee4466877d291ffd42a5a"
  }
}
```

示例中的 `bundle_digest` 只作格式示意，不构成回归基线；实现必须按上述 JCS 规则对当次完整 `data` 重算。

## 7. Marketplace → Fleet

### 7.1 原子安装主体保留

F01=`POST /v1/capabilities/install`，仅消费现有`{definition,bindings}`协议，不修改Fleet协议、实现或部署，不是Marketplace Relation API。

```json
{"definition":{"schema_version":"1.0","name":"marketplace:demo-expert","skills":[{"name":"demo-skill","description":"Demonstration skill","content":"---\nname: demo-skill\ndescription: Demonstration skill\n---\nRead references/guide.txt before acting.\n","config":{},"files":[{"path":"references/guide.txt","source":{"type":"inline_text","content":"This reference belongs to the pinned demo-skill.\n"}}]}],"experts":[{"name":"demo-expert","description":"Demonstration expert","instructions":"Use the pinned skill.","skill_names":["demo-skill"],"custom_env":{}}]},"bindings":{"experts":[{"expert_name":"demo-expert","runtime_id":"00000000-0000-4000-8000-000000000001"}]}}
```

示例为普通无命名冲突安装，使用市场原名/原字节。单Expert省略expert_team；Team为`{name,description,instructions,members:[{expert_name,role,is_leader}]}`，每Expert绑定相同runtime_id，custom_env属于expert而非bindings。成员按市场sort_order/relation_id输出，member_key及顺序映射仅留本地plan，不能加进Fleet JSON。Leader出站role="leader"，非Leader保留原合法role；市场/A34原role/member_key/sort_order不改。无MCP省略mcp_config，非空才发送，空mcpServers会被现有Fleet拒绝。Definition不新增市场ID/pin或Runtime绑定接口。

**命名与共存（D3）：** 本期**明确不允许同名/多版本共存**。同一个 Definition 内同名不同完整内容安装冲突失败（409 `same_install_identity_different_content`）；不生成 `mk1` 或其他稳定别名、不改写 `SKILL.md` frontmatter、不改目录名、不改 Fleet/Daemon。普通无冲突安装保留市场原名/原字节；Fleet 现有 `Definition.Skill.Name` 唯一约束和 workspace+owner+name 冲突行为保持不变，实际触发现有冲突就明确失败。未来若要做共存，需另立决策并补真实 Fleet→PostgreSQL→Daemon 联调证据，不得从本示例推导已启用。

### 7.2 消费既有24h幂等窗口，不改Fleet

Fleet现有F01以workspace+认证owner+key及规范化完整请求判定幂等，在既有24h窗口内，同键同请求可回放原结果。Marketplace使用原attempt_id作为Fleet幂等键；首次plan一旦冻结，任何重试都必须沿用该plan并由调用者重新提供相同运行时Secret，不能按当前源发布快照重建内容，也不能通过新key规避冲突。**同 key 不重新读取源发布快照/pin；上游目标正常发新版仍重放原 plan。显式输入变化或原 pin 节点资格失效才按 §6 拒绝；想使用更新 pin 则新 key 新 attempt。**

Fleet仍用其规范化请求SHA-256作为内部重复比较；Marketplace本地`fleet_request_hmac`只校验自己重建的完整出站请求，包括运行时Secret，不冒充Fleet归一化指纹，两种摘要不跨服务比较。HMAC和不含Secret的冻结plan不具备还原Secret的能力。这里“不落Secret”专指客户端本地记录与Marketplace的attempt/plan/日志，不改变Fleet既有运行态持有职责。

Marketplace只在自首次可能发出请求起不超过24h的保守窗口内允许自动重试；一旦超期，即使Fleet自身可能允许过期key重新占用，Marketplace也不得再次自动POST。本方案不改变Fleet过期重占行为、cleanup、数据库、receipt保留期限或任何其他实现，不承诺远端永久去重。

重试直接调用现有F01，不以先查询receipt为前提，不新增Fleet或Marketplace安装恢复GET，也无新增查询DTO/operationId。请求缺少原Secret、无法重建完整参数、输入不同、超过窗口，或安全重试后仍无法确认结果时，保留未知并明确提示人工核对；不得自动换key、删attempt或fallback安装。窗口内再次重试仍须满足原plan、同键及同完整请求条件。应用重启仅在仍有原plan、窗口和用户重新提供的相同完整参数时具备重试条件，不承诺自动恢复。

F01返回仍须执行§7.1完整性验证；有效窗口内不同请求的冲突沿用现有Fleet行为，不把所有409当同键冲突，也不将4xx当作先前未知请求未提交的证据。本地缺少attempt或当前资源名称相同/不同，都不能用于判断未知提交。REL-01只收集既有Fleet的真实联调证据；旧pending/UNKNOWN无法证明结果时保留未决并人工核对，不将其迁移成“未执行的新尝试”。

### 7.3 工作空间定制组合：直接消费既有 Fleet 资源能力

本节是职责说明和现有能力核验，不新增本次Marketplace接口或Fleet改造任务。公开Fleet服务已提供`POST /v1/skills`、`PATCH /v1/skills/{skill_id}`、`PUT /v1/experts/{expert_id}/skills`（全量替换）、`POST /v1/experts/{expert_id}/skills`（增量添加）；使用原用户/Workspace鉴权。

本地Skill推送时读取真实SKILL.md与支持文件，创建请求沿用`{name,description,content,config,files:[{path,content}]}`，其中content为SKILL.md正文；获得Fleet Skill ID后，以`{skill_ids:[...]}`绑定工作空间实例。只传本地目录路径无法让远端读取。全量PUT必须包含需要保留的既有绑定，不能误删其他Skill；同名或共享资源冲突不能自动覆盖，必要时显式创建实例使用的副本。

此链路没有市场发布、审核或private Skill步骤。市场下载后对实例追加本地Skill只是运行态定制，不改变已发布市场版本；将组合再发布市场必须另走独立Skill正式发布和Expert引用，Expert的依赖按发布/提交时冻结的 pin 读取，不解析 target latest。上传与绑定失败分别明确呈现，不声称两次既有调用整体原子或推断已运行会话立即热更新。这里确认服务端底座可用，不声称Buddy现有UI已实现完整本地选择推送。

## 8. 错误、无降级与版本隔离

| HTTP/code | reason | 行为 |
|---|---|---|
| 400 VALIDATION_ERROR | relations_required / dependency_version_mismatch / dependency_embedded / connector_version_forbidden / team_wiring_invalid / team_member_name_conflict / unsupported_type / zip_only_required / dependency_scope_mismatch / view_invalid | 明确字段；草稿可省略 `target_version_id`，提供时必须属于目标/已 release，否则 `dependency_version_mismatch`；已发布快照缺 pin 由发布校验拒绝（非 API reason）；不补current、不把Admin JSON旁路接入ZIP内容；space/system 引用 private 目标按 `dependency_scope_mismatch` 拒绝且不扩大受众；view 非法组合 `details.field=view`/`details.reason=view_invalid` |
| 404 NOT_FOUND | dependency_unavailable / version_not_found | 无权目标不泄漏ID/名称/存在性；private/跨Space/隐藏路径匿名；不删边后成功；A34/读取路径不回退或切换到其他版本 |
| 409 CONFLICT | dependency_changed / dependency_unavailable / dependency_in_use / edit_conflict / version_label_conflict | 阻止变更/发布；pin 的版本失效/目标不可用时不替换 pin、不回退/不切换版本；人工审核窗口 pin/effective 变化或原安装 pin 节点下架/删除等资格失效用 `dependency_changed`；上游正常发新版不阻断原 plan 重试、也不 409 |
| 409 CONFLICT | installation_in_progress / idempotency_input_mismatch / installation_attempt_expired | 保留原尝试/计划；同 key 不静默改为新源发布快照、不换key、不伪造完成 |
| 409 CONFLICT | same_install_identity_different_content | 同名不同内容安装冲突失败；不清空依赖、不随机重命名、不生成别名 |
| 400 VALIDATION_ERROR | mcp_server_name_conflict / unsupported_binary / incompatible_skill_path / legacy_container_format | 整体失败；旧容器引导显式转换，不自动转换 |
| 413 PAYLOAD_TOO_LARGE | budget_exceeded | 返回安全max/actual，不能截断文件/成员 |
| 500 INTERNAL_ERROR | artifact_integrity / relation_index_inconsistent | 内容/索引不一致是服务数据故障；不回退live/其他版本；已发布快照 pin/refs 缺失 fail closed |
| 503 UPSTREAM_UNAVAILABLE | installation_result_unknown / fleet_unavailable | 不回显上游Secret原文；仅在既有24h窗口内以原plan、同键和同完整请求重试，否则保留未知/人工核对 |

权限前置失败使用既有401/403；其他人的目标按404隐匿。**任何错误都不向无权者返回 private 目标的存在性、名称、ID 或数量；** `dependency_scope_mismatch`只在有权查看目标但来源受众不覆盖（含 space/system→private 且目标对操作者可见）时400，不透露隐藏路径。`dependency_version_required` 等旧 reason 已删除，不再作为请求/响应 reason 或错误码出现；`dependency_version_mismatch` 仅表示“提供了显式 pin 但不属于目标/未 release”。**pin 解析失败的错误映射：** 读取/A34 目标不可见或 pin 目标不可读 → 404 `dependency_unavailable`；发布/批准/A34 pin 不可用 → 409 `dependency_unavailable` 或 `dependency_changed`；原安装 pin 节点资格失效 → 409 `dependency_changed`（权限错误仍按 401/403/404）；目标正常发新版不触发该错误；不新增顶层 code，也不回退/切换到其他版本。A29/A30 的 `plan_stale`/`migration_incomplete` 不出现在 HTTP 错误表，CLI 迁移使用 §5 的计划/run 状态与退出码。所有handler需声明401/403/404/409/413/500/503中实际可达项，按原固定code枚举，不为reason新造顶层code。

## 9. 实施责任与验证口径

MP-01拥有版本/文件/CAS及A01唯一owner，负责A01/A02/A04～A08/A12/A13/A21～A24/A26的版本底座；**A09 已删除，不在任何 issue/客户端范围**。MP-02扩展A01依赖可用性（**pin 可用性**、四字段诊断、公共COUNT/facets、mine/Admin保留行）、共享 **pin 解析服务**、关系/图/refs、Connector effective 审核适配、删除并集，拥有A03/A10/A11/A14～A19/A25/A27。MP-03仍唯一拥有A31/A32安装与A34完整读取，A31/A32同一InstallCoordinator，A34复用 **pin resolver**，Fleet只既有安装联调、不改代码；A34不依赖Fleet在线。MP-04负责§5的CLI生产迁移、冻结计划/摘要、run恢复、**存量 pin 回填**、独立可见核验与数据安全回滚；**A29/A30 HTTP 已撤回，不属于本发布，也不属于任何实现issue的交付物**。CL-01消费A03/A10/A21～A27（A22只用于精确版本正文，pin 身份来自A03/A10或A25 resolved）；CL-02负责Web/CLI的A01～A08/A10/A12～A19/A32/A34、mine四字段、CAS与新key重装；CL-03消费A01/A02/A03/A05～A08/A31/A34，单次A34、不补依赖；REL-01验收真实四端、数据层、Fleet/Daemon、部署门禁、旧执行器不可达及数据安全回滚。8issue依赖保持；**A34仍属MP-03，A09/A29/A30不在任何范围**；生成物、客户端字段形状须同步，不靠前端扇出/过滤或运行时兼容补救。

实现端点时将本文DTO落到Go struct与swag及OpenAPI3.1；每个拥有API的issue运行check/diff并保存真实结果。这里只生成设计，不运行未实现API，不将旧main OpenAPI校验结果当作新协议通过证据。最终REL-01按真实四端流量逐个核对A/F接口和失败行为。
