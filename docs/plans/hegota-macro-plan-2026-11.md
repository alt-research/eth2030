# ETH2030 Hegotá 宏观开发计划

> 研究截点：2026-11-25（Asia/Shanghai）。版本：v0.1，范围仍在收敛。
> 本轮交付是研究与规划；没有修改协议实现、参考子模块或兄弟仓库。
> 配套材料：[逐项 EIP 与讨论登记](hegota-eip-review-2026-11.md)、[来源与版本清单](hegota-sources-2026-11.json)。
> 后续实施入口：[完整开发计划与 77 项任务](hegota-development-plan-2026-11.md)，包含依赖、验收、排期、跨仓库责任和发布门。

## 1. 建议与当前判断

建议把 ETH2030 的 Hegotá 开发组织为“规范基线 → Frame 交易与公开交易池 → 账户迁移与 PQ 接口 → 按最终范围集成 FOCIL → 跨客户端验证”的阶段工程。先建立可复现的规范组合和差异清单，再进入功能开发；当前不应以旧文档中的“约 97% 完成”作为排期依据。

Hegotá 是 Heze（CL）与 Bogotá（EL）的组合。当前 [Meta EIP-8081](https://eips.ethereum.org/EIPS/eip-8081) 仍列 EIP-7805、EIP-8141 为 SFI，但 11 月的讨论正在重新评估范围，尤其是 FOCIL 与 PQ 共识迁移的先后顺序。SFI 是当前升级意向，不等于最终冻结、已上线或规范不再变化。

三个直接影响开发顺序的结论：

1. **Frames 是 EL 工作主轴。** EIP-8141 已 SFI；EIP-8250、8272 是重点伴随提案；EIP-8504 / PR #12672 提出的最小 leanSPHINCS proof frames 接口需优先跟踪，但仍是开放草案。完整 EIP-8288 聚合不能直接并入本版本承诺。[8141](https://eips.ethereum.org/EIPS/eip-8141)、[8504 草案 PR](https://github.com/ethereum/EIPs/pull/12672)
2. **FOCIL 保留计划，但设独立决策门。** 11 月 24 日专题会讨论将它移出 Hegotá，以集中资源推进 PQ 共识；截至截点尚不能把该讨论写成已经 DFI。先处理可复用的 EL 接口、状态验证和测试资产，大规模 CL 集成等待范围决定。[FOCIL #47 会议记录](https://ethereum-magicians.org/t/focil-breakout-47-november-24-2026/30420)
3. **增加 EIP 必须看集成与测试成本。** 11 月 23 日 ACDT 未同意当周扩展 Bogotá devnet-3 范围；这是阶段性暂停，不是所有候选永久拒绝。ETH2030 也应先校准交易格式、计费、交易池和兼容性。[ACDT #104](https://github.com/ethereum/pm/issues/2306)、[会议摘要与时间戳](https://forkcast.org/calls/acdt/104/)

## 2. 来源优先级与“最新”的定义

| 来源 | 用途 | 使用边界 |
| --- | --- | --- |
| EIP-8081 + EIP-7723 | 正式范围登记与 PFI/CFI/SFI/DFI 定义 | Meta 更新可能落后于会议；保留差异，不能静默覆盖 |
| 最新 ACDE / ACDC 的明确决定 | 补充较晚的升级范围变化 | 议程、个人意见、会议摘要中的倾向分别标注；跨层变化需相应协调 |
| EIP 正文、EELS、consensus-specs、execution-apis | 具体行为与测试依据 | 用提交 SHA 固定；不同仓库的 `master` 不自动组成一个兼容 devnet |
| EF Protocol 的优先级与评级文章 | 解释战略目标与取舍 | EF 团队意见不等于 ACD 范围决定；9 月意见需结合 11 月讨论 |
| Ethereum Magicians 主讨论、规范 PR | 设计演进、反对理由与待决项 | 一个回复或原型不代表共识；没有回复也不代表通过 |
| Forkcast 的结构化历史与会议材料 | 定位日期、会议与遗漏提案 | 区分整理稿、机器辅助内容与原始规范；核心分歧回查作者回复/会议 |
| Strawmap | 长期方向、依赖和研究主题 | 属于草案路线图，不能直接变成 Hegotá 必交清单 |

本轮覆盖 Meta 的 **58 项**：2 SFI、20 CFI、10 PFI、26 DFI；再补入 Forkcast 历史中的 11 项、EF 评级中的 EIP-7973，以及紧邻 Frames 的 EIP-8288、8504，共 **72 项提案**。另核查 EIP-7723、7773、7732、7928、8081 作为过程与前置背景。配套登记逐项提供规范、讨论、阶段演进及项目处置。

讨论研究读取公开主题的正文与回复，重点核查提出理由、关键分歧、作者澄清及较晚的范围变化；抓取的公开回复覆盖数记录在来源清单中。长主题按关键回复梳理，不宣称完成全部视频、私有 Discord、每个历史 PR 或每条技术断言的审计。本计划也不是逐条共识符合性报告。

研究固定的上游版本：

- EIPs：`ade9e9cab7e148bba4cd293f916a4a189cd7a8aa`，提交时间 2026-11-25 UTC。
- consensus-specs：`d2021d2719235f0df4726a1d330502cb15aa8ea3`，提交时间 2026-11-23 UTC。
- 尚未合入 EIPs 主分支的提案使用各 PR head SHA，见来源清单；这类草案不能伪装成已发布的规范。

## 3. EF 规划与最近变化

### 3.1 战略方向

EF Protocol 在 9 月提出 2029 年 12 月实现 L1 三层抗量子能力的目标，并讨论完整迁移与 MV-PQ 保障路径。它是研发目标与风险规划，不是对量子突破日期的预测，也不是 Hegotá 一次性交付全部 PQ 功能的承诺。[EF Protocol priorities，2026-09-07](https://blog.ethereum.org/2026/09/07/protocol-priorities)

同日 EF 将 FOCIL、Frames 置于 S 层级，并重点支持 keyed nonces、recent roots、VOPS 等伴随工作；评级体现其团队交付偏好。后续 ACD 决定可以不同，例如 Optional Execution Proofs 获 EF A 评级，但 Hegotá 阶段已 DFI。[EF Hegotá 意见与评级](https://blog.ethereum.org/2026/09/07/protocol-hegota-eips)、[8025 阶段历史](https://forkcast.org/eips/8025/)

[Strawmap](https://strawmap.org/) 提供长周期背景；二进制树、SSZ execution blocks、完整证明聚合、加密交易池等，只有经过本次升级范围流程才进入本版本承诺。研究原型仍可保留，但不计算为版本交付。

### 3.2 时间线与冲突登记

| 日期 | 观察到的变化 | 对 ETH2030 的影响 |
| --- | --- | --- |
| 2025-12-22 | EF 发布 Hegotá 提案流程及窗口；FOCIL 当时仅 CFI | 仅作流程历史，不能沿用当时范围。[官方说明](https://blog.ethereum.org/2025/12/22/hegota-timeline) |
| 2026-02-19 | ACDC #175 将 FOCIL 升为 SFI | 解释本地 2 月参考版本为何只列 FOCIL。[阶段历史](https://forkcast.org/eips/7805/) |
| 2026-03-26 → 08-27 | Frames 从 CFI 到 SFI；早期并未被选为 headliner | 区分早期评选与后续加入。[阶段历史](https://forkcast.org/eips/8141/) |
| 2026-09 至 10 月 | ACDE 批次处理候选；8250、8272 等 CFI；8151 与 8298 成对考虑 | 需要组合实现与测试，CFI 不能当最终激活清单。[ACDE #247](https://forkcast.org/calls/acde/247/) |
| 2026-10-29 | ACDC #190：8015 SFI；8334、8383、8148、8411 CFI；8205 DFI | Meta 尚有落后条目；CL 工作不能只读 Meta。[ACDC #190](https://forkcast.org/calls/acdc/190/) |
| 2026-11-12 | ACDC #191：8198 Quick Slots 转 DFI；8483 获 PFI；FOCIL/PQ 优先级未决 | 从交付承诺移除 Quick Slots；8483 做跨层影响评估。[ACDC #191](https://forkcast.org/calls/acdc/191/)、[作者撤回留言](https://github.com/ethereum/pm/issues/2291) |
| 2026-11-17 | Frames #8 讨论 8288 前向兼容、proof frame、交易哈希和 VERIFY 上限 | 接口尚未冻结；先完成差异清单，限制一次性深度重构。[议题及 PR](https://github.com/ethereum/pm/issues/2295)、[会议材料](https://forkcast.org/calls/aa/008/) |
| 2026-11-19 | ACDE #249 讨论缩小范围；不支持直接放弃 Frames；跨层 devnet 等待 FOCIL 决定 | 保持 Frames 核心推进；FOCIL 集成用决策门。[官方主题摘要](https://ethereum-magicians.org/t/all-core-devs-execution-acde-249-november-19-2026/30071) |
| 2026-11-23 | ACDT #104 暂不扩展 devnet-3，继续处理 Frames 规范；4758 与 8360 配套仍有争议 | 小 EIP 也必须独立证明收益与测试可控。[ACDT #104](https://forkcast.org/calls/acdt/104/) |
| 2026-11-24 | FOCIL #47 区分“IL 内承载 PQ 交易”与“IL 自身用 PQ 签名”；更大 proof 的传播方案仍待研究 | 不能把这两项改造及完整聚合打包假定为一个小任务。[会议主题](https://ethereum-magicians.org/t/focil-breakout-47-november-24-2026/30420) |

**尚未发生的会议不写成决定。** 2026-11-26 ACDC #192 是接下来的范围观察点；专题会提及的 12 月 10 日是拟议的后续决定节点。两者都晚于本次截点。[ACDC #192 议程](https://github.com/ethereum/pm/issues/2305)

[ethereum.org Hegotá 页面](https://ethereum.org/roadmap/hegota/) 有 2027 年的规划提示，但 Meta 的测试网和主网激活栏仍空白。本项目采用相对里程碑，不承诺一个官方尚未确定的主网日期。

## 4. 版本范围与取舍

| 工作包 | 涉及 EIP | 本项目建议 | 进入发布条件 |
| --- | --- | --- | --- |
| Frames 核心 | 8141 | P0：先规范校准，再完整执行链路 | 对选定组合的编码、签名哈希、执行、收据、计费和 RPC 均有向量 |
| 隐私/并发验证基础 | 8250、8272 | P0/P1：随 Frames 组合准备 | 接口与 8141 同版；nonce/roots 系统合约、BAL 与重组处理一致 |
| 最小 PQ 认证接口 | 8504；参考 8288 | P0 跟踪与接口设计；实现以草案冻结为门 | witness 传输、区块承诺、计费、参数、测试完整；不默认激活开放草案 |
| 旧密钥退出 | 8298、8151 | P1：成对评估；兼容性可阻断发布 | 代码采用时机、重验证依赖、Permit/OP Stack 等影响有明确结论 |
| FOCIL 基础 | 7805 | 当前 SFI，范围复议中；保留独立主线 | ACD 明确本 fork 范围；实际 EL/CL 调用路径、构块和 fork choice 联调 |
| Frames 的 FOCIL 资格 | 8369 | P0 规范问题清单；P1 条件实现 | Informational 模型转成可执行的约束；索引承诺、预算、ePBS 后检查职责明确 |
| 交易断言 | 7906 | P1 候选，先解决回滚与 paymaster 交互 | 不能让断言回滚免除已承诺给 sponsor 的补偿；正式语义及测试明确 |
| 内容与 BAL 资源上界 | 8131、8279 | P1 安全/性能候选，作为组合评估 | 与 Frames 双维计费、refund、BAL 回滚规则一致；有攻击负载数据 |
| 其他 EL 候选 | 3298、4758、5920、7668、7709、7979、8077、8116、8163、8253、8360、7907 | P2 单项准入；不批量承诺 | 有稳定规范、champion、测试向量和范围确认；4758/8360 按配套方案判断 |
| CL 与跨层候选 | 8015、8148、8334、8365、8383、8411、8483 | Lighthouse 主责；ETH2030 明确接口义务 | 8015 的较晚 SFI 单独保留；其余按当前阶段及最终 devnet 组合推进 |
| 数据驱动参数候选 | 8368、8372 | P2 等待上游数据；不自定参数 | Glamsterdam 运行数据及 ACD 选择明确 |
| 已 DFI/撤回与远期项 | 见逐项登记 | 保留研究资料，不作为 Hegotá 新开发承诺 | 只有后续正式重新纳入才变更版本范围 |

这里的 P0/P1/P2 是 **ETH2030 建议优先级**，不是官方评级或额外 SFI 声明。已有本地代码的功能也必须经过相同门槛。

## 5. 仓库基线：可以复用什么，先补什么

抽查基线为本仓库提交 `dbe10116b703f88a97a6c45f1443b489e8e3b4ce`。下表是直接代码证据，不是对全仓库的缺陷穷举。

| 领域 | 已核查的本地证据 | 与当前上游的差距 / 后续动作 |
| --- | --- | --- |
| 参考规范 | `refs/EIPs` 在 `49beda9…`（2026-02-28）；`refs/consensus-specs` 在 `14e6ce5…`（2026-02-26） | 约九个月变化未反映；本轮不修改只读 `refs/`。实施前建立外部固定版本快照及组合清单 |
| Frame 编码 | [tx_frame.go](../../pkg/core/types/tx_frame.go)：四字段 frame、旧 fee 字段、15000 intrinsic、1000 frames | 当前 8141 为带 flags/limits/value 的 frame、signature list 与聚合 fee 字段；常量也已变化。旧编码测试通过不代表兼容 |
| APPROVE | [frame_execution.go](../../pkg/core/eips/frame_execution.go)、[eip8141_opcodes.go](../../pkg/core/vm/eip8141_opcodes.go)：0/1/2 scopes | 当前为 bitmask：0 不授权、1 payment、2 execution、3 combined；需同步执行器、VM、回滚与向量 |
| Nonce | `FrameTx.Nonce` 把 upper 192 bits / lower 64 bits 作为二维 nonce | 当前 8250 是 `nonce_keys` 集合 + `nonce_seq`，`[]` 与 `[0]` 不同；不能靠字段改名宣称支持 |
| 公开交易池 | [frame_rules.go](../../pkg/txpool/frametx/frame_rules.go)：50k conservative / 200k aggressive、首帧 VERIFY | 当前 8141 有四种 validation prefix、签名成本、代码/存储依赖及不同预算；需用 trace 验证与依赖驱动重验证 |
| FOCIL 节点接线 | [engine_backend.go](../../pkg/node/backend/engine_backend.go)：非空 `ProcessInclusionList` 返回 storage unsupported | 辅助模块与另一 Engine backend 的存在不能替代生产节点调用路径；先画出并验证实际链路 |
| CL 参考差异 | 11 月 Heze 对照 2 月版本，IL 由 committee root 改为 dependent root，签名 domain、gossip/req-resp、时限与存储索引均有变化 | Lighthouse 与 EL 的 IL 表示、分支身份、缓存和签名不能沿用旧结构；按同一固定版本联调 |
| FOCIL Frames 检查 | [il_satisfaction.go](../../pkg/focil/il_satisfaction.go)：跳过 Frame sender balance，验证器可选，仅检查 VERIFY target 有代码 | 不能代替 payer/APPROVE 的真实验证前缀，更未证明满足 8369；避免错误放行或错误处罚 builder |
| Fork 配置 | [chain_config.go](../../pkg/core/config/chain_config.go)、[chain_config_forks.go](../../pkg/core/config/chain_config_forks.go)：`HogotaTime`、`GlamsterdanTime` 和实验项关联 | 需审计配置→规则→VM/processor→Engine/RPC 的激活闭环；名称迁移需考虑既有配置兼容 |
| PQ 资产 | [pubkey_registry.go](../../pkg/crypto/pqc/pubkey_registry.go) 的 registration signature 使用 stub | 不等同 8483 的 validator/builder 请求、所有权检查、CL 存储，更不能作为 PQ 上线证明 |
| ePBS 前置 | [validation.go](../../pkg/epbs/validation.go) 对零签名/零公钥有测试放行路径 | 检查生产调用是否隔离；7805 与 8369 集成前必须确认基础验证不接受测试豁免 |

**建议修正旧路线图的方式：** 后续版本以本计划及符合性矩阵跟踪；旧 `docs/ROADMAP.md` 和历史分析中的百分比仅作历史记录。不要把“有一个包”“单元测试通过”“内部 devnet 出块”“当前官方 devnet 互通”混成一个完成状态。

### 5.1 已发现的规范组合风险

这些是本轮对固定正文的交叉检查，属于待上游澄清的观察，不擅自裁定规范：

1. 8250 所展示的 `fees` 结构仍与当前 8141 聚合 `max_fee` 表述不同；必须确认最终组合编码与签名哈希。[8141 编码](https://eips.ethereum.org/EIPS/eip-8141#payload-encoding)、[8250 payload](https://eips.ethereum.org/EIPS/eip-8250#transaction-payload)
2. 8272 当前使用 canonical VERIFY frame，并明确不改 envelope；8369 的部分描述仍引用 signed-envelope recent roots。8250 当前使用 `MAX_NONCE_SEQ` 表示 single-use，而 8369 仍有 `nonce_seq == 0` 的叙述。这些变化需要同步后才能生成组合向量。[8272](https://eips.ethereum.org/EIPS/eip-8272)、[8250](https://eips.ethereum.org/EIPS/eip-8250)、[8369](https://eips.ethereum.org/EIPS/eip-8369)
3. 8369 明确是 **Informational**，本身不激活共识。Profile 2 还缺执行性扩展所需的 claimed-index 编码/承诺、最终预算与 ePBS post-reveal 职责，不能直接按候选常量上线。[8369 enforcement requirements](https://eips.ethereum.org/EIPS/eip-8369#eip-7805-extension-requirements)
4. FOCIL 主讨论 11 月 21 日提出 7934 区块字节上限可能使“可追加交易”的判定不闭合；需回归精确 RLP 边界，不能只检查剩余 gas。[作者/实现者问题](https://ethereum-magicians.org/t/eip-7805-committee-based-fork-choice-enforced-inclusion-lists-focil/21578/18)、[修复原型](https://github.com/ethereum/execution-specs/pull/4041)

上述 CL 差异来自[固定的 Heze 正文](https://github.com/ethereum/consensus-specs/tree/d2021d2719235f0df4726a1d330502cb15aa8ea3/specs/heze)。该版本已经包含部分候选代码，不能反过来把“规范主分支已有改动”推定为该 EIP 已 SFI。

## 6. 工作分解与依赖

### 阶段 A：建立实施基线与符合性矩阵

**主责：EL 规范/核心维护角色。** 不假定团队人数；一个人可以分阶段承担多个角色。

- 固定 EIPs、EELS、EEST、consensus-specs、execution-apis 和目标 devnet 的相互兼容提交；上游 head 仅作变化检测。
- 对 8141/8250/8272/7805/8369 逐条建立“规则 → 文件/调用点 → 正反向向量 → 是否集成 → 未决项”矩阵；覆盖双维 gas 与 Glamsterdam 前置规则。
- 明确实验代码与 fork 激活路径；盘点错误的旧假设、生产路径中的 stub 和测试捷径。
- 建立配置和数据库兼容策略；预先定义规范变更后哪些向量必须重生成。

**退出门 G0：** 规范组合、关键字段/常量、未决冲突清单可复现；不再用旧 API 编号或历史注释代替规范。无法闭合的部分标为 blocked-by-spec，可继续处理独立模块。

### 阶段 B：Frames 核心与交易池

**主责：EL types / VM / state processor / txpool / RPC。** 推荐顺序：类型及哈希 → gas/approval/rollback → processor 与 receipt → RPC/txpool → 真实节点接线。

- 重做/校准 frame 与交易编码、签名哈希、边界检查、introspection；链 ID、溢出、签名可塑性必须有负向向量。
- 实现 approval journal、validation prefix、atomic batch、transient storage、frame value 与 execution/state gas 隔离。
- 统一 keyed nonce 状态、支付时消费、失败回滚边界、替换/取消语义；recent roots 覆盖来源隔离、过期、空 slot 与 reorg。
- 公开交易池执行受限 trace，索引 sender/payer/code/root/nonce 依赖，有限预算重验证；区分 block validity、public mempool admission 与 FOCIL eligibility。
- RPC 覆盖提交/查询/收据/模拟/估 gas 的明确行为；估算和失败码需要跟最新 API 草案一致，不在规划里发明接口。

**退出门 G1：** 选定规范的有效/无效向量与 EELS 差分通过；从节点公开入口提交到构块、导入、收据、重启与重组的链路都能复现。

### 阶段 C：PQ 认证与账户迁移

**主责：EL 密码学接口、VM 与钱包兼容测试；CL 注册由 Lighthouse 协同。** 与阶段 B 的稳定底座部分交错推进。

- 为 proof frame / witness 接口保留明确版本边界；8504 参数、block witness、传输和 gas 尚未定稿前只做受控原型及向量。
- 区分直接 leanSPHINCS 验证与 8288 递归聚合，记录 witness 的所有权、去重、排序和区块承诺；聚合不作为基础 Frames 验证的隐含必需项。
- 8298/8151 组合覆盖旧 ECDSA 交易、7702 delegation、采用真实 code 后的 permit、代码依赖失效与不可变合约。
- 单列 OP Stack fault-proof、账户代码采用时机和 multisig 的兼容性分析。11 月主讨论仍在请求相关方参与，不能视作已有签字。[8151 讨论](https://ethereum-magicians.org/t/eip-8151-account-code-restricted-ecrecover/27690)、[8298 讨论](https://ethereum-magicians.org/t/eip-8298-setcodefrom-code-reuse-instruction/28779)
- 8483 若最终纳入，做请求类型、系统合约、所有权验证、覆盖更新、请求排序与 CL 消费的跨层测试；本地 PQ registry 不直接复用为规范实现。

**退出门 G2：** 上游范围与接口足以固定；迁移前后权限与验证成本可解释；没有以 mock/stub 或旧签名仍有效的路径冒充完整 PQ 迁移。

### 阶段 D：FOCIL 条件集成

**启动门：** ACD 对 Hegotá 的 FOCIL 范围明确；若延期，则保留独立分支/开关和测试资产，继续 Frames/PQ 主线，不擅自更改网络规则。

- 贯通 `txpool → IL 构建 → Engine → CL gossip/store → builder 更新 → payload 执行 → IL satisfaction → CL fork choice`。
- 根据选定版本确认 append/rebuild、API 版本与返回状态；ePBS 下区分执行有效性与 IL 满足性，避免把后者不满足自动解释为块执行无效。
- 覆盖非法字节、过大 IL、blob 资格、迟到与重复、equivocation、缓存/裁剪、分支差异、payload 更新竞态。
- Profile 2 只在规范允许且职责、编码、BAL 重建、代码体积与 replay 预算闭合时启用；不以“目标有代码”替代 VERIFY。
- 单独测量 raw PQ signature 对 IL 容量/带宽的影响；完整 proof 传播与 IL 签名 PQ 化分别评估。

**退出门 G3：** ETH2030 + 外部 CL/EL 的正反向用例均成立；不满足 IL 的情形真的影响 fork choice，满足的情形不会误伤有效链。

### 阶段 E：候选准入、发布与运行验证

候选 EIP 一次一个工作包，明确规范、收益、已有 devnet、集成面、回归成本和 owner。8131/8279、4758/8360 等有语义关联的组合一起审查。收据/bloom 改造要验证 RPC、日志索引和历史查询，不能只改 RLP 类型。

**退出门 G4：** 目标 devnet 的完整范围已固定；跨客户端状态根/收据/BAL 一致；恶意负载、重组、重启、同步与历史查询验证完成；才进入测试网/发布准备。

## 7. 节奏与验收

以下是本项目的相对节奏建议，不是官方日程或未经评估的人周承诺。先完成阶段 A，再依据差异矩阵估工。

| 节点 | 必交物 | 开始后续阶段的证据 |
| --- | --- | --- |
| M0：规范校准 | 固定版本清单、逐规则差异矩阵、fork 激活与节点调用图 | G0；第一批实现任务都能追溯到规范条款 |
| M1：Frames 可执行 | 核心类型/VM/processor/receipt 与状态向量 | G1 的规范差分部分通过 |
| M2：公开入口可用 | 交易池、RPC、真实节点端到端；nonce/roots 组合 | G1 全部通过；负向交易不能拖垮公开入口 |
| M3：PQ/迁移候选闭合 | 固定的 witness 接口、迁移兼容分析、成本报告 | G2；范围不确定项保留明确禁用状态 |
| M4：跨层一致 | FOCIL 若纳入则完成联合 devnet；CL 候选接口测试 | G3；或正式范围裁剪后替换该门 |
| M5：发布候选 | 固定范围、复现脚本、跨客户端结果和运行报告 | G4；不以内部 helper 测试替代互通证据 |

最低测试矩阵：

| 类别 | 必测内容 |
| --- | --- |
| 单元/向量 | 编码及哈希、scope/flags、费用上界、nonce 集合、roots 边界、frame rollback、无效 witness |
| 规范差分 | 与选定 EELS/EEST 对同一 pre-state、交易、区块比较 post-state / receipts / BAL / error |
| 调用链 | 公共提交→txpool→builder→Engine→导入→查询，含替换、驱逐、重启与 replay |
| 跨客户端 | ETH2030 与至少一个独立 EL 实现及一个外部 CL 组合互通；再扩大客户端矩阵 |
| 资源/对抗 | 批量失效、共享 payer、代码变更、最大 nonce keys、root 过期、超大签名、RLP 字节边界、IL 重复/分叉 |
| 运行恢复 | 长 reorg、DB 回滚与恢复、空 slot、同步、fork 前后交易与旧配置兼容 |

记录硬件、gas/区块参数、延迟分位数、CPU、内存、磁盘和传播数据，再设置目标。11 月 PQ-CL devnet 暴露过 key-generation 延迟与深 reorg 恢复问题，说明 EL 的恢复路径也应进入联合测试；这不是要求 EL 自己实现 CL 密钥生成。[PQ-CL #4](https://ethereum-magicians.org/t/pq-cl-breakout-4-november-24-2026/30460)

## 8. 优先处理的风险与下一批任务

| 优先级 | 风险 | 当前行动 / 决策条件 |
| --- | --- | --- |
| P0 | 2 月规范与 11 月上游分叉 | 固定版本、做组合差异矩阵，首先处理 8141 编码与 APPROVE |
| P0 | 8141/8250/8272/8369 文字不同步 | 列出最小复现与矛盾，等待/跟进上游澄清；不自行创造共识 |
| P0 | FOCIL 范围复议及 PQ 共识路径未定 | 11-26 及后续 ACD 后重新快照；将范围决定与实现进度分开 |
| P0 | Frames witness / hash / fee 继续变化 | 稳定底座与草案接口分开验收；每次换版重跑组合向量 |
| P1 | 公开交易池可被共享状态批量失效 | trace 规则、依赖索引、预算、payer reservation 与重组重验证 |
| P1 | 8151 影响不可变合约与 fault proof | 真实合约回放、受影响生态反馈、明确升级窗口；可阻断该候选 |
| P1 | FOCIL 判定只看 gas 忽略 byte 上界 | 加入 7934、RLP 长度前缀和余额/nonce 级联激活的回归 |
| P1 | stub 与实验开关进入真实路径 | 生产调用图、否定测试、默认配置与激活边界审查 |
| P2 | 候选不断加入导致测试面失控 | 单项准入；统计集成路径和测试成本，延后没有门槛证据的提案 |

建议下一批开发任务只先开四项：**HGT-001 固定规范组合与逐规则矩阵；HGT-002 FrameTx 编码/哈希/常量校准；HGT-003 APPROVE 与费用/回滚语义校准；HGT-004 真实节点的 Frames/IL 调用链审计。** HGT-002/003 的实现以 HGT-001 选择的规范组合为前提；本轮不提前把它们标成已完成。

研究后的第一轮范围复核，应核对 11 月 26 日 ACDC、下一次 ACDE/ACDT、8504/8483 PR 与 Frames 更新。无需在此阶段承诺所有候选或固定主网时间。

## 9. 本轮验证与交付边界

- `cd pkg && env -u GOROOT go build ./...`：通过。
- `cd pkg && env -u GOROOT go test ./core/types ./core/eips ./core/vm ./txpool/frametx ./focil ./engine/... ./node/backend/...`：通过。
- 文档登记编号、来源哈希、公开主题抓取覆盖、相对链接与本地证据路径另做结构核验。
- 本轮没有运行官方 Hegotá 跨客户端 devnet、没有更新规范子模块，也没有完成实现符合性审计；现有测试通过仅证明当前本地基线可运行。
