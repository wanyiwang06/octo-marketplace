# CL-01：Admin 引用式容器（ZIP-only）、CAS 与只读依赖预览（B 模型）

> 决策依据：[00-decisions](../00-decisions.md) B 模型、D3（无同名共存/不改名/不改 frontmatter）、D6（scope 矩阵）、D7（Admin 无资产审核）、D8（CAS/relations）、C06（Admin ZIP-only + A25 Key/CAS/resolved）。Admin 只做客户端接入与现有操作适配，不新增发布矩阵或生命周期 UI。
>
> **范围与基线：** 本 issue 只改 `octo-admin` 客户端。`eec5223` 是 `octo-marketplace` 的基线，不是 `octo-admin` 的 commit；`octo-admin` 是外部依赖仓库，须以自身 commit/日期登记基线，本 workspace 无其 checkout，源码断言实施前须复核。

## 目标（Goal）

- 在现有专家/专家团上传、重传、详情及 Skill/MCP 管理界面完成新模型接入，不另造操作面。
- **硬依赖：MP-02**（包含 MP-01 的版本/发布基础）。可并行协作：MP-04 编写 CLI 转换、CL-02/CL-03 改各自客户端；迁移异常只引导到 MP-04 的 CLI 冻结计划流程。
- 接口参考：[02 API 契约](../02-api-contracts.md) 的 A03/A10（解析目标最新发布身份）、A21～A27（Admin 读写、版本、容器）、§4（schema2）及 §8（错误）；仅接入现有 Admin 操作，不新增发布接口。A22 只做精确 `version_id` 读取，不承担最新解析。以下 URL 均省略网关前缀 `/market/api/v1`。

## 背景（Background）

Expert/Team 只通过 schema2 ZIP 维护内容与“目标 ID + 可选审计版本”的关系；A25 强制 `Idempotency-Key`、`expected_edit_revision` 与只读 `resolved[]`，A24 拒绝 Expert/Team 的 JSON 内容写。B 模型下 `resolved[]` 回显的是**本次导入时解析到的目标最新发布版**，不是权威 pin；同 key 回放只回放首次 `resolved[]`，目标后来发新版必须重新用 A03/A10 解析（或换新 key 做新导入）才能看到，不能靠 A25 同 key 重放。下次读取/安装会重新解析，Admin 预览不能把当前解析结果展示成永久固定版本。A22 是精确 `version_id` 版本读取，不承担 latest resolver；目标最新身份来自 A03/A10 或 A25 导入时 `resolved[]`，正文再用 A22 精确版本、Connector 用 A21 effective 读取。

普通安装保留市场原名/原字节；不生成 `mk1`、不改写 `SKILL.md` frontmatter、不让页面覆盖 ZIP 清单。原 Admin 鉴权、写入动作、上下架和归属语义保持不变；**Admin 无资产审核**，创建/更新/重传直接生效，不进入 A14/A16 或 pending review；只有 Tenant（Web/Space）按原 Space 策略审核。不新增上架按钮、发布矩阵或治理 UI。

## 承重行为（Load-bearing）

1. **ZIP-only。** Expert/Team 的创建/更新只走 `POST /admin/plugins/import` 与 `POST /admin/plugins/container_reupload/{plugin_id}`（multipart 原 ZIP + `container_schema_version="2"` + 契约允许元数据）；A24 明确拒绝 Expert/Team 的 JSON 内容/关系/元数据覆盖（400 `zip_only_required`），不另设 metadata PATCH。
2. **A25 幂等/CAS 与 receipt。** 同一意图同一 `Idempotency-Key`；新建省略 `expected_edit_revision`，重传必带；同 key 同 ZIP/字段/CAS 由服务端 receipt 回放原结果（**含首次 `resolved[]`，不因目标后来发新版重算**），不同输入 409；revision 过期 409 `edit_conflict`。receipt 由服务端拥有，客户端只复用同一 key 回放，不另建本地去重账本、不按文件名/名称猜成功。要看目标后来发布的新版本，必须重新走 A03/A10，或使用新 key 做新导入；不能依赖 A25 同 key 回放。`resolved[]` 只读回显本次解析到的目标身份，不作为客户端 pin。
3. **B 预览。** 页面只展示清单声明与服务端 `resolved[]`；依赖目标的**最新发布身份**先由 A03/A10 解析（或采用本次 A25 导入返回的 `resolved[]` 观察值），再用 A22 按精确 `version_id` 读取正文、用 A21 `?view=effective` 读取 Connector 有效配置；A22 不承担 latest resolver，不能用 current/editing 或同 key 回放的旧 `resolved[]` 代替最新解析。明确标注“本次解析版本”。不提供依赖选择器、pin 编辑器、页面关系覆盖；目标后来发新版时，下一次预览/读取必须重新调用 A03/A10（或新 key 新导入）显示新版本，不能把旧 `resolved[]` 当固定值。
4. **关系写入。** 清单只表达目标 ID（Expert skills/connectors、Team members 的成员 ID/`member_key`/role/is_leader/sort_order）；`version_id` 可省略，若出现仅作审计：提供时必须属于目标且已 release，否则 400 `dependency_version_mismatch`；与本次最新解析版本不同不报错、不改变解析；Connector 完全省略版本，带版本 400 `connector_version_forbidden`。Team 的成员角色/Leader/顺序由 Team 边决定，成员 Expert 内容跟随最新。
5. **D8 写路径 CAS。** 独立 Skill 元数据编辑/重传、MCP/Connector 表单编辑、Expert/Team 重传都从同一次授权 editing 读取取得原完整内容与 `edit_revision`，提交 `expected_edit_revision`；不在提交前偷偷读取最新 revision 给陈旧表单续签。A24 的 Skill/Connector 关系为空，不要求 `relations`；`relations` 必填只适用于 A12 完整 Write，不外溢到 A24/A25/A26。
6. **Connector effective。** 详情/预览读 `A21 ?view=effective` 展示当前有效配置；编辑表单读默认 editing，不能把未批准编辑冒充已生效；Connector 不显示历史版本选择，不新增业务版本。
7. **Admin 无资产审核，沿用既有动作。** Admin 的创建、更新、ZIP 重传与系统 Connector 写入都直接生效，并在成功事务内同步 release/effective；不创建 A14 review、不进入 A16 批准、不产生 pending review/`review_id` 或“提交审核/待审”状态。只有 Tenant（Web/Space）继续既有 Space 审核策略。评分、投放、上下架、删除均走原入口与权限；评分/投放不生成内容版本；default 投放不可隐藏；非默认场景既有可见性保留。
8. **无共存/无改名。** 不实现 `mk1`、不重写 frontmatter/BOM、不改目录名；同名不同内容由安装阶段冲突失败；Admin 不提供“改名后共存”的旁路。

## 非目标（Out of scope）

依赖选择器、pin 编辑器、草稿页、独立发布编辑按钮或安装入口；AI 猜 ID、覆盖 ZIP 清单、自动执行生产迁移；**Admin 资产审核/提交/待审/批准（A14/A16）**、A09 restore、A29/A30 HTTP、新发布矩阵、新生命周期 UI、同名共存、Fleet/Daemon 改造。Admin 只做直接生效写入，只有 Tenant 使用审核。

## 源码与所有权

- 仓库 `octo-admin`：`src/api/{marketplace,expert,skill,mcp,plugin}.ts`；`src/pages/ExpertMarket/{parseContainer,aiPrompt,importRating}.ts`、`src/pages/ExpertMarket/{UploadModal,ReuploadButton,ExpertDetailDrawer,SquadDetailDrawer,detailParts}.tsx`。
- 独立 Skill 入口：`src/pages/SkillMarket/{SkillUploadModal,SkillDetailDrawer,SkillTab}.tsx` 及 `src/api/skill.ts::{updateAdminSkill,commitAdminSkillReupload,getAdminSkillMd}`；MCP 入口：`src/pages/SystemMcp/{FormModal,DetailDrawer,index}.tsx` 及 `src/api/mcp.ts::{toConnectorUpsert,updateSystemMcp}`。同步各入口所属的 `src/i18n/locales/{zh-CN,en-US}/{expertMarket,skillMarket,systemMcp}.json` 与已有 common 文案；实施前以仓库实际 namespace 注册为准。
- 本 issue 只拥有客户端 adapter、类型、现有 UI 与使用说明；不写数据库。版本/release/CAS 和 Tenant 审核接口的接入由 MP-01/MP-02 实现，Admin 不接 A14/A16；转换与映射由 MP-04 CLI 所有。
- **源码与基线（外部依赖）：**`octo-admin` 不在本 workspace，`eec5223` 只是 `octo-marketplace` 基线，不能当 admin SHA。本 issue 的 admin 文件路径与行为断言按契约和历史记录列出，实施前须以 `octo-admin` 自身 commit/日期复核；未复核前不得把“已核源码”当作实现证据。
- **待复核源码现状（本 workspace 未核对 octo-admin）：**`expert.ts::importExpertContainer/reuploadContainer` 及 `ReuploadButton` 已经提交原始 ZIP，旧浏览器逐 Skill 上传链早已删除；本次替换的是 `parseContainer.ts` 仍解析 bundled/inline 的 schema 和服务端导入语义。`resolveSkillRefs/getSystemSquad` 仍用 `allSettled` 丢弃失败依赖，`resolveSquadMember` 仍从 `expert/context.json` 补角色并读成员当前 `mcp.json`，`detailParts::loadSkillMd` 仍有 404 转 inline stub，均须明确切走。

## 实施步骤

1. 更新公共 DTO：`view`、`selected_version_id`、`version_label`、`current_version_id`、`has_unpublished_changes`、`is_published`、`effective_revision`、`effective_digest`、`declared_visibility`、`revision_no`、`edit_revision`；关系 `relation_id`/`target_plugin_id`/可选 `target_version_id`（仅审计）/`sort_order`/`data`；历史 `is_released`、只读 `resolved[]`；Connector 不增加业务版本字段，但必返 `effective_revision/effective_digest`。**Admin 运营 DTO 必须完全省略 `published_version_id`（包括 null 占位）**，不公开线上指针；Admin 默认详情是 current，不从 current 推断线上内容；`view` 只允许默认 editing/current 与 Connector effective，非法组合 400 `details.field=view`（`details.reason=view_invalid`）。
2. 替换 `parseContainer.ts` 的旧 bundled/inline 解析，保留现有原 ZIP 整体上传链。ZIP 根只能有一个 `expert.json` 或 `squad.json`，`schema_version:"2"`；上传原始 ZIP 到 A25，multipart 含 `file`、`container_schema_version="2"` 和契约允许的元数据，不提交 `relations`/`manifest` 覆盖字段。`buildContainerForm`、import/reupload adapter 与两个 drawer 调用点一起补 schema、候选 version 和重传 CAS，不仅修改文件选择器。
3. 清单严格按 02 §4 完整字段表：两类均 schema_version/kind/name、非空 summary 或 description、instruction，tags 可选；Expert 显式 skills/connectors 数组，Team 为 members 目标 ID/`version_id`（可选审计）/`member_key`/role/is_leader/sort_order。`version_id` 可省略；提供时必须属于目标且已 release，否则 400 `dependency_version_mismatch`；与本次最新解析版本不同不报错、不改变解析。Skill/Connector 各 ≤20、成员 1～30 且唯一 Leader/`member_key`/目标；`role` 非空且 ≤500 字符、`member_key` 1～64 字符并匹配 `[a-z][a-z0-9_-]*`、`sort_order` ≥0；非 Leader 禁保留 `role=leader`；解析后成员名称必须唯一，冲突返回 400 `team_member_name_conflict`。ZIP 仅一个根清单，拒绝额外文件（skills ZIP、`mcp.json`、`expert/context.json` 等）；重复 JSON 键、缺目标 ID、Connector 带版本、未知/旧团字段及内联能力明确拒绝，不静默丢指令；版本创建缺省 1.0.0，重传沿 current，需正式新标签时提示用户明确更高版本、不自动 bump。文件/归档预算以 02 为准。
4. 本地预览仅展示清单声明；服务端 `resolved[]` 是**本次导入时**每条最终直接关系的只读身份回显，使用 02 定义的 `source_plugin_id/relation_id/relation_type/target_plugin_id/target_version_id?`，Connector 完全省略版本。预览必须标注目标版本是“本次解析”，不得在产品文案中承诺固定；不把该数组臆造为含名称/配置的完整图。需要展示目标当前最新发布身份时，必须用 A03/A10 重新解析（或只采用本次 A25 返回的 `resolved[]` 观察值），再用 A22 按精确 `version_id` 读取该目标版本正文、用 A21 `?view=effective` 读取 Connector 有效配置；**A22 不是 latest resolver**，不得用 A21 current/editing 或 A25 同 key 回放的旧 `resolved[]` 冒充最新。显示精确 ID/版本并与清单对照。无权/不存在或版本资格失败不得删边后提交；用户修 ZIP 后重传。旧包 `legacy_container_format` 指向 MP-04 的 CLI 显式转换流程，不能前端拆包悄悄创建独立资源。AI 帮助保留 ZIP 生成，但只生成清单自身内容并使用操作者提供的真实 ID，不猜 ID 或用 UI 覆盖清单。
5. 导入/重传同一意图持有同一 `Idempotency-Key`，网络重试不换 key，文件或元数据改变属于新意图；编辑/重传携带契约规定的 `expected_edit_revision`。`UploadModal.ImportEntry` 分别保存每个文件的 key 和成功 plugin ID；中止/超时进入原 uncertain 状态时保留 key，重试同 ZIP/元数据回放 A25，不能按名称猜成功再换 key。保留 `importRating.ts::importThenRate` 的“导入成功、评分失败只重试评分”，批量上传逐行显示真实结果，不宣称跨文件原子。409 按 reason 展示输入/CAS 冲突，读回差异后重新确认，不自动覆盖。新建不带 expected revision。
6. 详情关系保留 `relation_id`/类型/目标 ID/审计版本/data/sort_order；`resolveSkillRefs/resolveSquadMember/getSystemSquad` 先按关系的目标 ID 用 A03（轻量图）或 A10（候选/最新发布身份）取得该目标**本次最新发布版本 ID**，再以 A22 精确 `version_id` 读取版本正文/详情（A22 必须带精确 ID 并做归属检查，不能只传 `plugin_id` 自动取 latest/current；A22 对 Connector 返回 400 `unsupported_type`）；Connector 预览用 A21 `?view=effective` 沿原 Admin 鉴权读取有效配置，MCP 编辑表单仍用默认 editing，不能冒充已生效。Team 成员的二跳 Skill/Connector 同样先解析身份再精确读取；不再读成员 current 本地 `mcp.json` 作为依赖内容。删除 `expert/context.json` 角色补全及成员旧 `mcp.json` 数据源。Admin Skill 正文/下载使用 A23：默认 current 且必须标实际 `version_id`，可显式传 `version_id`；公共 published 读取使用 A06/A07，不得把 A06/A07 与 A23 混用或互相兜底。`SkillMdSource` 与两 drawer 的选择状态携带资源+版本，移除 `loadSkillMd` 的 404 转 stub。错误不丢边或变成“无依赖”，不沿用缓存/inline 配置；缓存按资源+版本和身份上下文分离。历史版本自身正文/附件不可变，但其依赖目标按当前最新解析展示并标注，不把最新内容说成旧快照内容。Connector 不显示历史版本选择。
7. 状态提示消费现有响应；**Admin 无资产审核**：AdminCreate/AdminUpdate、ZIP 导入/重传直接生效并同步 release/effective，不调用 A14、不产生 A16 pending/approval，不进入审核队列；只有 Tenant（Web/Space）继续既有发布审核与 Space 策略。沿用上下架和删除动作及权限检查，不建立新保存/即时发布矩阵，不增加重新上架菜单、按钮或接口。仅将关系/解析身份与完整图错误接入已有操作；不能借重传新增上线副作用。
8. 保留现有评分/投放界面和非默认场景可见性，default 投放不能隐藏；不增加新隐藏治理流程，评分/投放不生成内容版本。
9. 统一展示 `error.code/details.field/details.reason/hint`，如 `skills[0].target_plugin_id`、`members[1].id`；鉴权、跨 Space、依赖占用、依赖变化与 CAS 冲突保留可定位错误，不把失败项包装成成功；源版本/refs 自相矛盾为 500 `relation_index_inconsistent`，refs 一致但目标缺失才是正常 `dependency_unavailable`，不得吞成零依赖。同步 AI 提示和现有使用文案，删除“ZIP 被拆分上传”“UI 覆盖关系”等旧指引。
10. 独立 Skill 和 Connector 的写调用同步 CAS，不能只改 Expert/Team ZIP：`updateAdminSkill`、`updateSystemMcp` 从同一次授权 editing 读取取得原完整内容及 `edit_revision`，提交 `expected_edit_revision`；A24 的 Skill/Connector 关系为空，不要求 `relations`（显式 `[]` 也可），不把 A12 的 relations 必填外溢到 A24；`commitAdminSkillReupload` 及其 UI 调用链向 A26 传 `expected_edit_revision`（不是任意版本号/文件名），parse_task_id 完整文件继续由服务端保存，不从 readme 重建包。CAS 必须保护用户打开/确认的那一版。A24 拒绝 Expert/Team create/update（400 `zip_only_required`），其内容及内容元数据统一 A25 ZIP；Expert/Team 无现有 JSON 内容编辑调用，不新增 metadata PATCH 绕过 ZIP。评分和 placement 独立原入口保留，不带内容 CAS 或新建内容版本。
11. 不实现同名共存相关 UI/命名：不生成 `mk1`、不展示/写入改写后的 frontmatter、不提示“改名后可共存”；同名不同内容冲突交给安装错误提示，不提供 Admin 旁路。

## 逐项验收（Acceptance）

- [ ] schema2 Expert/Team 原 ZIP 上传成功；清单目标 ID、成员角色与顺序和服务端详情逐项一致，无选择器、无额外依赖写入、无 frontmatter 改名；`role`≤500、`member_key` 1～64 正则、`sort_order`≥0，解析后名称冲突返回 `team_member_name_conflict`，额外文件拒绝。
- [ ] 缺目标 ID、旧内联格式、重复目标/Leader、越限均定位原字段；失败无部分成功提示。system/space/private 范围按已确认规则验收（private 来源可引用已发布 system、同 Space 的 space、同 owner private；space/system 引用 private 拒绝），权限错误不泄露目标存在性。A24 对 Expert/Team JSON 写返回 `zip_only_required`；A26 重传缺/过期 `expected_edit_revision` 拒绝。
- [ ] Admin DTO 含 `view/selected_version_id/version_label/current_version_id/has_unpublished_changes/is_published/effective_revision/effective_digest/declared_visibility/revision_no`，且完全省略 `published_version_id`（包括 null 占位）；A22 只按精确 `version_id` 读取并做归属检查，A03/A10 提供最新解析身份，Admin Skill 内容用 A23（默认 current 且标实际版本），公共 A06/A07 用 published，不混用或互相兜底。
- [ ] schema2 `version_id` 可省略；提供时须属于目标且已 release（否则 400 `dependency_version_mismatch`），与本次最新解析不同不报错；Connector 带版本返回 `connector_version_forbidden`。
- [ ] 同 key 重试回放原结果（含**首次** `resolved[]`，不因目标后来发新版重算），换文件/字段复用 key 冲突；过期 `edit_revision` 不覆盖。A24 Skill/Connector 关系为空，不要求 `relations`；A12 完整 Write 才要求显式 relations（省略/null 400 `relations_required`，`[]` 才清空）；A25 只收 ZIP，不接受 HTTP relations/manifest 覆盖，不能要求上传额外 relations 数组。
- [ ] 目标后来发新版时，下一次预览/读取通过新的 A03/A10 调用（或新 key 新导入）显示新版本，不把同 key 回放的旧 `resolved[]` 当权威 pin；页面不因目标更新报来源保存失败（除非来源自身 CAS/校验失败）。
- [ ] 历史版本自身正文/附件不变；其依赖目标按当前最新解析展示并有“本次解析”标识；`is_released=false` 不被宣称为正式发布；Connector 无版本 UI。
- [ ] ZIP 上传/重传不提供依赖选择器或页面关系覆盖；Admin 写入、直接生效与上下架行为及权限不变，不新增 Admin 审核队列/上架动作或治理 UI。
- [ ] 评分/非默认场景投放沿用现有行为，default 不可隐藏；删除只删除引用式来源及自身关系、不删共享目标，live∪published∪Team→最新发布 Expert 间接保护的拒绝均可读且不泄露敏感信息。
- [ ] 批量每 ZIP 独立 key；网络中止/响应丢失重试同 key 不重复导入，评分失败不重导入。Skill 元数据编辑/重传与 MCP 表单编辑全部携带原 CAS；旧表单并发保存被拒绝，不用最新 revision 强行通过。
- [ ] Admin 管理 current 详情、历史版本与 Connector effective 配置明确区分；失败关系、404 正文和旧 context/mcp 附件均不静默 fallback，原管理行/删除入口仍可用且不冒充完整图。
- [ ] 不出现 `mk1`、frontmatter/BOM 改写、目录名改写或“Admin 改名共存”入口；旧包只提示 MP-04 CLI 转换。

## 测试与可观测 smoke（Verification）

- 实施时扩展 `parseContainer.test.ts`、`src/api/expert.test.ts`、相关 drawer/upload 测试，覆盖 schema2、HTTP 字段/幂等头、版本解析标识及失败路径；本设计交付不运行检查。
- 隔离环境验证原 ZIP 导入/重传、目标 ID/`resolved[]` 与列表/详情回显，记录 endpoint/脱敏 key/版本 ID；确认无逐 Skill 创建请求、Admin 写入无 A14/A16 pending，Tenant 审核/上下架操作与鉴权保持，不为新治理动作安排 smoke。
- 用一次依赖错误、一次“目标已发新版”的预览和一次过期 revision 重传，界面保持原数据并展示字段原因/解析版本，不把目标新版误报为来源失败；证据不得含 Token、Secret 或 MCP 原文。

## 部署前置

- MP-01/MP-02 的 Admin DTO、schema2、CAS 与幂等能力先行；旧包转换提示只有在 MP-04 CLI 可用后才能提供执行入口。Admin 无资产审核；直接生效写入与 Tenant 发布审核操作保持，不新增生命周期部署前置。旧上传执行链必须移除，不能保留失败 fallback；不部署 A09/A29/A30。

## 本轮评审修订验收

- [ ] Admin 无资产审核：创建、更新、重传直接生效并同步 effective，界面和调用链无“提交审核/待审/审核批准”，不调用 A14/A16、不创建 pending review；只有 Tenant（Web/Space）按原策略审核。失败 CAS/依赖校验不得部分生效。
- [ ] 依赖目标的最新身份由 A03/A10（或 A25 本次 `resolved[]`）解析，再用 A22 精确 `version_id` 读正文、A21 effective 读 Connector；A22 不承担 latest resolver。A23 默认 current 并标实际版本，公共 A06/A07 只读 published，不混用。
- [ ] DTO 含本轮新增 `view/selected_version_id/version_label/current_version_id/has_unpublished_changes/is_published/effective_revision/effective_digest/declared_visibility/revision_no`；**Admin DTO 完全省略 `published_version_id`（包括 null 占位）**。
- [ ] A25 同 key 回放首次 `resolved[]`；要看目标后来新版必须重新 A03/A10 或新 key 新导入，不能靠同 key 回放。A24/A25 均不创建 pending review。
- [ ] schema2 `version_id` 仅审计（归属+release，否则 400 `dependency_version_mismatch`；差异不报错），Connector 版本 400；manifest role/member_key/sort_order/解析后名称唯一/额外文件校验齐全；A24 `zip_only_required`、A26 `expected_edit_revision` 有负例。
- [ ] `octo-admin` 以自身 commit/日期登记基线；`eec5223` 仅作 marketplace 基线，源码断言在 admin 仓库复核前不作为证据。依赖预览展示本次解析版本；更新仅审计展示，无主动通知；管理权限不绕过依赖范围，也不代替生产迁移的计划执行授权。
