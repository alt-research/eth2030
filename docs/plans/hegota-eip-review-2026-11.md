# Hegotá：逐项 EIP 与讨论审阅登记

> 截点：2026-11-25（Asia/Shanghai）。与[宏观计划](hegota-macro-plan-2026-11.md)及[来源清单](hegota-sources-2026-11.json)配套使用。
> 本登记共 72 项：Meta 58 项 + Forkcast 补充 11 项 + EF 评级补充 7973 + Frames 邻接提案 8288/8504。

## 阅读规则

- **文档状态**（Draft/Review/Stagnant/Withdrawn 等）与**某次升级纳入阶段**（PFI/CFI/SFI/DFI）是两条轴；下文分开记录。
- **Meta** 是截点读到的 8081 登记；**阶段演进**来自 Forkcast 的带日期记录并与重点会议交叉核查，可能较 Meta 更新。未登记日期/会议时保留原始空缺，不补造决定。
- EIP 正文以来源清单的 SHA 固定。尚未合入的正文链接到 PR head；开放 PR 不等于将进入 Hegotá。
- 每项说明依次覆盖机制、公开讨论的关键演进、ETH2030 的工程处置。工程优先级/建议不是官方纳入决定。
- 公开主题抓取包括完整 post stream；审阅按关键回复和后续会议梳理。回复数指抓到的 stream 对象，可能含删除占位或跳号，不能理解为活跃支持人数。主题只有首帖时明确标注。
- 未逐句审计全部历史 PR、完整视频或私有讨论；没有以帖子热度或首帖年代判断是否批准。对每条规范的代码/向量符合性是后续 M0 工作。

## 本轮需要特别纠正的阶段差异

| EIP | Meta | 较晚记录 | 规划含义 |
| --- | --- | --- | --- |
| 8015 | CFI | 10-29 ACDC #190：SFI | CL 已计划项 |
| 8148 / 8365 / 8383 | PFI | 分别已 CFI | 保持候选，不能漏掉或当必交 |
| 8198 | CFI | 11-12 ACDC #191：DFI，作者撤回 | 不再承诺 Quick Slots |
| 8142 / 8205 | PFI | 分别已 DFI | 从本版本新开发范围排除 |
| 8334 / 8411 / 8483 | 未列 | 分别 CFI / CFI / PFI | 补入跨层观察与评估 |
| 8369 | PFI | EF 重视，但当前仍是 Informational 模型 | 必须补执行性规范才能强制共识 |
| 7805 | SFI | 11-24 讨论是否延期，尚无撤销决定 | SFI 与范围复议同时保留 |
| 8504 | 未列 | PR #12672 为 draft，尚未据此认定 PFI | 最小 PQ 接口观察项 |

下列记录按核心依赖优先，随后列其他候选与已拒绝/撤回项；排序仅为阅读方便。

## EIP-8141 — Frame Transaction

- 状态：正文 `Draft`；Meta `SFI`；最近跟踪阶段 **SFI**。[规范](https://raw.githubusercontent.com/ethereum/EIPs/ade9e9cab7e148bba4cd293f916a4a189cd7a8aa/EIPS/eip-8141.md) · [主讨论](https://ethereum-magicians.org/t/frame-transaction/27617)
- 阶段演进：2026-03-26 CFI（[acde/233](https://forkcast.org/calls/acde/233/)） → 2026-08-27 SFI（[acde/244](https://forkcast.org/calls/acde/244/)）。
- 机制与讨论：原生可编程验证、执行与付款；当前正文已有 flags、双维 limits、signature list、聚合 fee 与原子批次。主讨论由 VERIFY/APPROVE 设计演进到 EOA 支持、signature hash 可塑性、paymaster 和 gas 隔离。#163–170 讨论签名字节与长度承诺，#178–186 涉及 sponsor 支付及 8151 代码依赖；11 月会议又在讨论 proof frame 与哈希方案。SFI 不表示接口冻结。
- ETH2030：P0 首项：旧编码/scopes 与当前规范差异很大；固定组合后再更新 types、VM、processor、txpool、RPC 和收据。
- 讨论覆盖：抓取 181/181 个 stream 对象；最新回复创建时间 2026-11-20 UTC。

## EIP-8250 — Keyed Nonces for Frame Transactions

- 状态：正文 `Draft`；Meta `CFI`；最近跟踪阶段 **CFI**。[规范](https://raw.githubusercontent.com/ethereum/EIPs/ade9e9cab7e148bba4cd293f916a4a189cd7a8aa/EIPS/eip-8250.md) · [主讨论](https://ethereum-magicians.org/t/eip-8250-keyed-nonces-for-frame-transactions/28437)
- 阶段演进：2026-07-02 PFI（[acde/240](https://forkcast.org/calls/acde/240/)） → 2026-09-24 CFI（[acde/246](https://forkcast.org/calls/acde/246/)）。
- 机制与讨论：当前模型是有界、严格递增的 nonce_keys 集合与 nonce_seq；空集合选择 legacy nonce，非空集合可有多个独立 key。主题首帖仍描述旧的单 key 模型；10-07 回复讨论 NONCE_MANAGER 与未来树迁移，10-23 #6 明确提醒 [] 与 [0] 不同。当前正文还以 MAX_NONCE_SEQ 表示 single-use，不能沿用早期二维打包假设。
- ETH2030：P0/P1 与 Frames 一起校准；覆盖重叠/不重叠 key 集、支付时消费、失败后持久性、替换、CREATE 地址与 BAL 记录。
- 讨论覆盖：抓取 5/5 个 stream 对象；最新回复创建时间 2026-10-23 UTC。

## EIP-8272 — Recent Roots for Frame Transactions

- 状态：正文 `Draft`；Meta `CFI`；最近跟踪阶段 **CFI**。[规范](https://raw.githubusercontent.com/ethereum/EIPs/ade9e9cab7e148bba4cd293f916a4a189cd7a8aa/EIPS/eip-8272.md) · [主讨论](https://ethereum-magicians.org/t/eip-8272-recent-roots-for-frame-transactions/28621)
- 阶段演进：2026-07-02 PFI（[acde/240](https://forkcast.org/calls/acde/240/)） → 2026-09-24 CFI（[acde/246](https://forkcast.org/calls/acde/246/)）。
- 机制与讨论：通过 canonical VERIFY frame 检查 (source_id, slot, root)，在账号验证前建立近期 root 依赖；当前不改 FrameTx envelope。07-15/17 的 #2/#3 比较 envelope 字段与 canonical frame，作者指出顺序和检查时机的取舍。当前正文选择 frame，并要求 slotNumber；不能从旧首帖或固定 12 秒 timestamp 推导实现。
- ETH2030：P0/P1；测试同 slot 覆盖、下一 slot 可引用、8191 边界、来源隔离、重组及交易池过期；同步 8369 的旧表述。
- 讨论覆盖：抓取 3/3 个 stream 对象；最新回复创建时间 2026-07-17 UTC。

## EIP-8504 — Proof frames for leanSPHINCS signatures

- 状态：正文 `Draft`；Meta `未列`；最近跟踪阶段 **观察/未确认纳入阶段**。[规范](https://raw.githubusercontent.com/lambdaclass/EIPs/654c0c3b2b5e066073012284fa93b2fea0522763/EIPS/eip-8504.md) · [开放 PR](https://github.com/ethereum/EIPs/pull/12672) · [主讨论](https://ethereum-magicians.org/t/eip-8504-proof-frames-for-leansphincs-signatures/30448)
- 阶段演进：未发现对应的 Forkcast Hegotá 阶段历史；以 Meta/会议及 PR 原始记录为边界。
- 机制与讨论：PR #12672 的最小 leanSPHINCS proof frame：声明依赖，签名 witness 独立传输，直接验证而不做完整聚合。11-20 #2/#3 明确要复用 8288 的 mode/layout/zero data_hash 约定，使未来格式兼容。当前 PR 仍列 witness transport、block encoding/validation、gas 和参数为未完成，且为 draft PR。
- ETH2030：P0 跟踪/P1 受控原型；最终编码、参数、承诺和预算固定后实施；不标作 SFI 或已经 PFI。
- 讨论覆盖：抓取 3/3 个 stream 对象；最新回复创建时间 2026-11-20 UTC。

## EIP-8288 — In-mempool signature and proof aggregation

- 状态：正文 `Draft`；Meta `未列`；最近跟踪阶段 **观察/未确认纳入阶段**。[规范](https://raw.githubusercontent.com/ethereum/EIPs/ade9e9cab7e148bba4cd293f916a4a189cd7a8aa/EIPS/eip-8288.md) · [主讨论](https://ethereum-magicians.org/t/eip-8288-frame-type-for-pq-sig-and-stark-aggregation/28723)
- 阶段演进：未发现对应的 Forkcast Hegotá 阶段历史；以 Meta/会议及 PR 原始记录为边界。
- 机制与讨论：以依赖集合与递归 STARK 聚合签名/证明，要求聚合声明绑定区块实际交易依赖。06 月讨论 hash-based 最小依赖；09 月关注 deps_hash、授权 verification key 与真实交易集合绑定；11-01 #14 提议更通用的 verifier 路径。11 月会议倾向先交付最小兼容接口，完整聚合另排。
- ETH2030：架构约束与后续研究；Hegotá 不默认承诺完整聚合。测试缺失/额外依赖、排序去重、错误 VK 和 witness 传输。
- 讨论覆盖：抓取 15/15 个 stream 对象；最新回复创建时间 2026-11-16 UTC。

## EIP-8298 — SETCODEFROM Code Reuse Instruction

- 状态：正文 `Draft`；Meta `CFI`；最近跟踪阶段 **CFI**。[规范](https://raw.githubusercontent.com/ethereum/EIPs/ade9e9cab7e148bba4cd293f916a4a189cd7a8aa/EIPS/eip-8298.md) · [主讨论](https://ethereum-magicians.org/t/eip-8298-setcodefrom-code-reuse-instruction/28779)
- 阶段演进：2026-07-02 PFI（[acde/240](https://forkcast.org/calls/acde/240/)） → 2026-10-08 CFI（[acde/247](https://forkcast.org/calls/acde/247/)）。
- 机制与讨论：让账号采用已有代码，服务代码复用与从 7702 delegation 到真实代码账号的迁移。讨论涉及 initcode 是否允许、代码何时生效及 SELFDESTRUCT 生命周期；11-24 #23 提出 EXTCODEHASH 与当前执行代码不一致可能破坏授权假设，并建议延迟生效。该建议不能当作已合入规范。
- ETH2030：P1 与 8151 配套；先解决采用时机，测试 reentrancy、EXTCODEHASH/CALL、创建/销毁、代码依赖重验证与旧密钥退出。
- 讨论覆盖：抓取 23/23 个 stream 对象；最新回复创建时间 2026-11-24 UTC。

## EIP-8151 — Account Code Restricted ecRecover

- 状态：正文 `Draft`；Meta `CFI`；最近跟踪阶段 **CFI**。[规范](https://raw.githubusercontent.com/ethereum/EIPs/ade9e9cab7e148bba4cd293f916a4a189cd7a8aa/EIPS/eip-8151.md) · [主讨论](https://ethereum-magicians.org/t/eip-8151-account-code-restricted-ecrecover/27690)
- 阶段演进：2026-06-04 PFI（[acde/238](https://forkcast.org/calls/acde/238/)） → 2026-10-08 CFI（[acde/247](https://forkcast.org/calls/acde/247/)）。
- 机制与讨论：ecRecover 在恢复地址后读取该地址代码分类；真实代码账号不再通过原有 0x01 授权，empty/delegation 情形按规范处理。02 月指出 OP Stack fault-proof 依赖 L1 ecRecover 的兼容风险；07 月规则推广到 raw code；10 月澄清 gas/BAL 和空返回数据。11-22 作者说明新地址 precompile 无法保护硬编码 0x01 的旧合约；11-24 仍在请求 OP Stack 参与。
- ETH2030：P1 高风险候选；做 permit、multisig、fault-proof 回放与 signer-code 失效索引，未有生态兼容结论不能发布。
- 讨论覆盖：抓取 30/30 个 stream 对象；最新回复创建时间 2026-11-24 UTC。

## EIP-7906 — Transaction Assertions via State Diff Opcode

- 状态：正文 `Draft`；Meta `CFI`；最近跟踪阶段 **CFI**。[规范](https://raw.githubusercontent.com/ethereum/EIPs/ade9e9cab7e148bba4cd293f916a4a189cd7a8aa/EIPS/eip-7906.md) · [主讨论](https://ethereum-magicians.org/t/eip-restricted-behavior-transaction-type/23130)
- 阶段演进：2026-07-30 PFI（[acde/242](https://forkcast.org/calls/acde/242/)） → 2026-09-24 CFI（[acde/246](https://forkcast.org/calls/acde/246/)）。
- 机制与讨论：通过 POST_TX/状态差异观察进行交易结果断言，并定义失败后的回滚边界。讨论由地址/slot 级约束扩展到事件和已触及状态。10 月 #32–35 指出断言回滚执行体可能同时抹去 paymaster 的 ERC-20 补偿，但其 ETH 付款保留；11 月会议仍讨论可行边界。
- ETH2030：P1 条件扩展；先闭合 sponsor 补偿与 rollback，再测试 false-return token、失败重试、批次和 BAL。
- 讨论覆盖：抓取 38/38 个 stream 对象；最新回复创建时间 2026-11-23 UTC。

## EIP-7805 — Fork-choice enforced Inclusion Lists (FOCIL)

- 状态：正文 `Draft`；Meta `SFI`；最近跟踪阶段 **SFI**。[规范](https://raw.githubusercontent.com/ethereum/EIPs/ade9e9cab7e148bba4cd293f916a4a189cd7a8aa/EIPS/eip-7805.md) · [主讨论](https://ethereum-magicians.org/t/eip-7805-committee-based-fork-choice-enforced-inclusion-lists-focil/21578)
- 阶段演进：2025-12-11 CFI（[acdc/171](https://forkcast.org/calls/acdc/171/)） → 2026-02-19 SFI（[acdc/175](https://forkcast.org/calls/acdc/175/)）。
- 机制与讨论：由 includer 委员会收集 IL，经执行侧判定缺失交易是否仍可加入，并由 CL fork choice 强制执行。早期讨论集中在匿名 includer、IL 选择多样性、满块豁免和按 validator/按 stake 选人的取舍；11-21 的 #18 又提出 EIP-7934 字节上限使遗漏检查不闭合。11-24 专题会继续讨论 PQ 下的容量及是否延期，尚未撤销 SFI。
- ETH2030：P0 规范与 EL 接线；跨层交付设范围门。测试满 gas/满 bytes、余额与 nonce 级联、无效字节、晚到/equivocation、ePBS 下 fork choice。
- 讨论覆盖：抓取 17/17 个 stream 对象；最新回复创建时间 2026-11-21 UTC。

## EIP-8369 — VOPS Profiles for FOCIL Eligibility

- 状态：正文 `Draft`；Meta `PFI`；最近跟踪阶段 **PFI**。[规范](https://raw.githubusercontent.com/ethereum/EIPs/ade9e9cab7e148bba4cd293f916a4a189cd7a8aa/EIPS/eip-8369.md) · [主讨论](https://ethereum-magicians.org/t/eip-8369-vops-profiles-for-focil-eligibility/29298)
- 阶段演进：未发现对应的 Forkcast Hegotá 阶段历史；以 Meta/会议及 PR 原始记录为边界。
- 机制与讨论：定义 Base VOPS 与 AA-VOPS 的资格、有限状态面和预算；Profile 2 用 builder-claimed index 检查遗漏。它是 Informational。09-07/09 的 #2–4 讨论能否取消 index 选择；作者强调单个 nonce 的稳定性不代表整个验证结果稳定，共享 payer/code/storage 仍会变化。正文明确要求后续执行性扩展，并指出 ePBS post-reveal 职责未闭合。
- ETH2030：P0 澄清、P1 条件实现；不得直接激活候选预算。覆盖状态重建、精确交易身份、index 承诺、完整 VERIFY 与代码加载上限。
- 讨论覆盖：抓取 4/4 个 stream 对象；最新回复创建时间 2026-09-09 UTC。

## EIP-3298 — Remove storage-clear refund and refund cap

- 状态：正文 `Draft`；Meta `CFI`；最近跟踪阶段 **CFI**。[规范](https://raw.githubusercontent.com/ethereum/EIPs/ade9e9cab7e148bba4cd293f916a4a189cd7a8aa/EIPS/eip-3298.md) · [主讨论](https://ethereum-magicians.org/t/eip-3298-removal-of-refunds/5430)
- 阶段演进：2026-08-13 PFI（[acde/243](https://forkcast.org/calls/acde/243/)） → 2026-09-24 CFI（[acde/246](https://forkcast.org/calls/acde/246/)）。
- 机制与讨论：取消 storage-clear refund 与 refund cap，保留当前正文规定的 net-metered STORAGE_WRITE reversal。2021 年旧主题围绕退款/GasToken 的利益与兼容性；2026-08-31 #31 提醒剩余 reversal 仍需 refund counter、journal、Frames 与 floor 交互。不能读成“删掉所有 refund 代码”。
- ETH2030：P2 单项；与 8037/8038/7778、Frames 和 floor 组合回归，不以代码删除量估工作。
- 讨论覆盖：抓取 32/32 个 stream 对象；最新回复创建时间 2026-09-05 UTC。

## EIP-4758 — Deactivate SELFDESTRUCT

- 状态：正文 `Stagnant`；Meta `CFI`；最近跟踪阶段 **CFI**。[规范](https://raw.githubusercontent.com/ethereum/EIPs/ade9e9cab7e148bba4cd293f916a4a189cd7a8aa/EIPS/eip-4758.md) · [主讨论](https://ethereum-magicians.org/t/eip-4758-deactivate-selfdestruct/8710)
- 阶段演进：2026-05-21 PFI（[acde/237](https://forkcast.org/calls/acde/237/)） → 2026-10-08 CFI（[acde/247](https://forkcast.org/calls/acde/247/)）。
- 机制与讨论：SELFDESTRUCT 改为仅转走 ETH 的 SENDALL，停止删除 code/storage/nonce。旧主题关注 force-send、生命周期和已有合约用途；最近的范围争议主要在 ACDT #104：它已在 devnet-2，而 TCREATE 未上网，需决定成对加入或撤下。
- ETH2030：P2 配套评估 8360；验证同交易创建/销毁、余额转移与 8298，不把旧 Stagnant 当成已经 DFI。
- 讨论覆盖：抓取 40/40 个 stream 对象；最新回复创建时间 2023-09-20 UTC。

## EIP-5920 — PAY opcode

- 状态：正文 `Draft`；Meta `CFI`；最近跟踪阶段 **CFI**。[规范](https://raw.githubusercontent.com/ethereum/EIPs/ade9e9cab7e148bba4cd293f916a4a189cd7a8aa/EIPS/eip-5920.md) · [主讨论](https://ethereum-magicians.org/t/eip-5920-pay-opcode/11717)
- 阶段演进：2026-08-13 PFI（[acde/243](https://forkcast.org/calls/acde/243/)） → 2026-10-08 CFI（[acde/247](https://forkcast.org/calls/acde/247/)）。
- 机制与讨论：PAY 转账但不调用接收方代码，降低转账触发执行的耦合。05-13 #61/#62 对 forced balance change 是否妨碍形式化验证有明确分歧；主张者指出已有其他 force-send 路径。10 月 ACD 已 CFI，后续仍受范围收敛约束。
- ETH2030：P2；测试新账号计费、拒收合约、余额守恒、重入缺失语义与 Frames 组合。
- 讨论覆盖：抓取 61/61 个 stream 对象；最新回复创建时间 2025-05-13 UTC。

## EIP-7668 — Remove bloom filters

- 状态：正文 `Stagnant`；Meta `CFI`；最近跟踪阶段 **CFI**。[规范](https://raw.githubusercontent.com/ethereum/EIPs/ade9e9cab7e148bba4cd293f916a4a189cd7a8aa/EIPS/eip-7668.md) · [主讨论](https://ethereum-magicians.org/t/eip-7653-remove-bloom-filters/19447)
- 阶段演进：2026-07-30 PFI（[acde/242](https://forkcast.org/calls/acde/242/)） → 2026-09-24 CFI（[acde/246](https://forkcast.org/calls/acde/246/)）。
- 机制与讨论：区块与 receipt 中的 bloom 变为空字节，改变编码及日志查询相关实现。讨论比较零长度、固定 256 字节全零/全一和压缩效果；较晚回复也关注大 gas limit 下的过滤效果，不能视作“只删计算”。
- ETH2030：P2；需要 RPC/索引/收据根与历史兼容回归，先测查询退化成本。
- 讨论覆盖：抓取 17/17 个 stream 对象；最新回复创建时间 2026-02-20 UTC。

## EIP-7709 — Read BLOCKHASH from Storage and Update Cost

- 状态：正文 `Draft`；Meta `CFI`；最近跟踪阶段 **CFI**。[规范](https://raw.githubusercontent.com/ethereum/EIPs/ade9e9cab7e148bba4cd293f916a4a189cd7a8aa/EIPS/eip-7709.md) · [主讨论](https://ethereum-magicians.org/t/eip-7709-read-blockhash-opcode-from-storage-and-adjust-gas-cost/20052)
- 阶段演进：2026-05-07 PFI（[acde/236](https://forkcast.org/calls/acde/236/)） → 2026-10-08 CFI（[acde/247](https://forkcast.org/calls/acde/247/)）。
- 机制与讨论：BLOCKHASH 从历史系统合约 storage 读取并按 cold/warm 访问收费。公开主主题很短，最近回复主要是 Hegotá PFI 材料；10 月会议支持其 BAL/轻客户端用途。短讨论不构成风险已清零。
- ETH2030：P2；覆盖 fork 边界、serve window、系统写入、BAL 读集和 gas。
- 讨论覆盖：抓取 2/2 个 stream 对象；最新回复创建时间 2026-05-05 UTC。

## EIP-7907 — Meter Contract Code Size

- 状态：正文 `Draft`；Meta `PFI`；最近跟踪阶段 **PFI**。[规范](https://raw.githubusercontent.com/ethereum/EIPs/ade9e9cab7e148bba4cd293f916a4a189cd7a8aa/EIPS/eip-7907.md) · [主讨论](https://ethereum-magicians.org/t/eip-remove-contract-size-limit/23156)
- 阶段演进：2026-10-08 PFI（[acde/247](https://forkcast.org/calls/acde/247/)）。
- 机制与讨论：对超出初始代码范围的代码加载收费；当前正文基于已提高的 code/initcode 上限，不是无限增大代码。主题讨论 code reuse、lazy inclusion 与 gas model；10-28 回复强调 ZK RAM/EXTCODESIZE 的资源压力。10 月重提 PFI，尚未正式纳入。
- ETH2030：P2；用 adversarial code touch 和证明端成本评估，不沿用旧“移除大小限制”标题。
- 讨论覆盖：抓取 46/46 个 stream 对象；最新回复创建时间 2026-10-28 UTC。

## EIP-7973 — Warm Account Write Metering

- 状态：正文 `Draft`；Meta `未列`；最近跟踪阶段 **观察/未确认纳入阶段**。[规范](https://raw.githubusercontent.com/ethereum/EIPs/ade9e9cab7e148bba4cd293f916a4a189cd7a8aa/EIPS/eip-7973.md) · [主讨论](https://ethereum-magicians.org/t/eip-7973-warm-account-write-metering/25907)
- 阶段演进：未发现对应的 Forkcast Hegotá 阶段历史；以 Meta/会议及 PR 原始记录为边界。
- 机制与讨论：对 warm account writes 等成本计量进行调整，属于状态/账号写入 gas 设计。主讨论 #2 提醒 account 的 storage_root 也应计入脏状态判断，以免先写 storage 再转 ETH 被重复收费。这是 EF 9 月评级提及的提案，不在本次 Meta/Forkcast Hegotá 关联并集中；不能把 EF 的反对评级映射为正式 ACD DFI。
- ETH2030：观察/不承诺；需重新提出、明确范围与数据依据后才排开发。
- 讨论覆盖：抓取 2/2 个 stream 对象；最新回复创建时间 2025-11-19 UTC。

## EIP-7979 — Call and Return Opcodes for the EVM

- 状态：正文 `Review`；Meta `CFI`；最近跟踪阶段 **CFI**。[规范](https://raw.githubusercontent.com/ethereum/EIPs/ade9e9cab7e148bba4cd293f916a4a189cd7a8aa/EIPS/eip-7979.md) · [主讨论](https://ethereum-magicians.org/t/eip-7979-call-and-return-opcodes-for-the-evm/24615)
- 阶段演进：2026-06-04 PFI（[acde/238](https://forkcast.org/calls/acde/238/)） → 2026-09-24 CFI（[acde/246](https://forkcast.org/calls/acde/246/)）。
- 机制与讨论：增加显式子程序 call/return 与受保护 return stack，涉及目的地验证。初期讨论 ENTERSUB/目的地标记；09-26 #43/#44 明确纠正“原型已经独立验证”的说法，语言与客户端维护者审查仍必要。9 月从默认 DFI 倾向转 CFI。
- ETH2030：P2；要求编译器/语言团队消费和独立性能证据；测试返回栈边界、非法目的地与已有 bytecode。
- 讨论覆盖：抓取 43/43 个 stream 对象；最新回复创建时间 2026-09-26 UTC。

## EIP-8015 — Remove `deposit` and `eth1data` fields

- 状态：正文 `Draft`；Meta `CFI`；最近跟踪阶段 **SFI**。[规范](https://raw.githubusercontent.com/ethereum/EIPs/ade9e9cab7e148bba4cd293f916a4a189cd7a8aa/EIPS/eip-8015.md) · [主讨论](https://ethereum-magicians.org/t/eip-8015-remove-legacy-deposit-and-eth1data-fields/25401)
- 阶段演进：2026-07-23 PFI（[acdc/183](https://forkcast.org/calls/acdc/183/)） → 2026-09-17 CFI（[acdc/187](https://forkcast.org/calls/acdc/187/)） → 2026-10-29 SFI（[acdc/190](https://forkcast.org/calls/acdc/190/)）。
- 机制与讨论：EIP-6110 完全生效后删除遗留 deposit/eth1data 的 CL 字段。主主题只有首帖；较晚证据来自 ACDC #190，已 SFI，Meta 仍 CFI。不能因主主题安静而漏掉会议升级。
- ETH2030：CL 已计划项；Lighthouse 主责，EL 检查 deposit execution requests 的前置与 fork 配置。
- 讨论覆盖：抓取 1/1 个 stream 对象；最新回复创建时间 2025-09-09 UTC。

## EIP-8077 — eth/XX - announce transactions with nonce

- 状态：正文 `Draft`；Meta `CFI`；最近跟踪阶段 **CFI**。[规范](https://raw.githubusercontent.com/ethereum/EIPs/ade9e9cab7e148bba4cd293f916a4a189cd7a8aa/EIPS/eip-8077.md) · [主讨论](https://ethereum-magicians.org/t/eip-8077-eth-xx-add-nonce-and-source-to-transactions-announcement/26505)
- 阶段演进：2026-08-13 PFI（[acde/243](https://forkcast.org/calls/acde/243/)） → 2026-10-08 CFI（[acde/247](https://forkcast.org/calls/acde/247/)）。
- 机制与讨论：交易公告增加 source/nonce 等元数据，减少无效交易下载；最终 wire 字段需看规范。2025-12/2026-01 回复曾推动公告 fees；08-19 #4 又提 upfront cost 与取整。当前正文/标题与历史建议不能直接拼接成消息格式。
- ETH2030：P2 网络协商候选；特别检查 Frames 多 key nonce、费用表示、谎报元数据与旧 peer。
- 讨论覆盖：抓取 4/4 个 stream 对象；最新回复创建时间 2026-08-19 UTC。

## EIP-8116 — Replace cumulative receipt fields

- 状态：正文 `Draft`；Meta `CFI`；最近跟踪阶段 **CFI**。[规范](https://raw.githubusercontent.com/ethereum/EIPs/ade9e9cab7e148bba4cd293f916a4a189cd7a8aa/EIPS/eip-8116.md) · [主讨论](https://ethereum-magicians.org/t/eip-8116-replace-cumulative-receipt-fields/27359)
- 阶段演进：2026-07-30 PFI（[acde/242](https://forkcast.org/calls/acde/242/)） → 2026-10-08 CFI（[acde/247](https://forkcast.org/calls/acde/247/)）。
- 机制与讨论：receipt 改用每笔 gasUsed，logIndex 语义也有变化，取消累计字段依赖。主主题只有首帖；ACDE #247 将其置于低优先级 CFI。没有充分公开回复可据以宣称生态一致。
- ETH2030：P2；需 receipt root、RPC、日志排序、索引与历史读取全链路兼容。
- 讨论覆盖：抓取 1/1 个 stream 对象；最新回复创建时间 2025-12-30 UTC。

## EIP-8131 — Unified Transaction Content Floor

- 状态：正文 `Draft`；Meta `CFI`；最近跟踪阶段 **CFI**。[规范](https://raw.githubusercontent.com/ethereum/EIPs/ade9e9cab7e148bba4cd293f916a4a189cd7a8aa/EIPS/eip-8131.md) · [主讨论](https://ethereum-magicians.org/t/eip-8131-add-auth-data-to-eip-7623-floor/27567)
- 阶段演进：2026-06-04 PFI（[acde/238](https://forkcast.org/calls/acde/238/)） → 2026-09-24 CFI（[acde/246](https://forkcast.org/calls/acde/246/)）。
- 机制与讨论：将多类用户可控交易字节纳入统一 content floor，覆盖 calldata/auth/access-list/blob hash 等。主主题只有一条简短支持回复；最近实质争论在 devnet 范围会议，与 8279 被作为资源上界组合讨论。
- ETH2030：P1 候选；测试 Frames witness/fee、refund、零/非零字节与最坏带宽，不能凭主帖点赞视为批准。
- 讨论覆盖：抓取 2/2 个 stream 对象；最新回复创建时间 2026-01-22 UTC。

## EIP-8148 — Custom sweep threshold for validators

- 状态：正文 `Draft`；Meta `PFI`；最近跟踪阶段 **CFI**。[规范](https://raw.githubusercontent.com/ethereum/EIPs/ade9e9cab7e148bba4cd293f916a4a189cd7a8aa/EIPS/eip-8148.md) · [主讨论](https://ethereum-magicians.org/t/eip-8148-custom-sweep-threshold-for-validators/27669)
- 阶段演进：2026-05-28 PFI（[acdc/179](https://forkcast.org/calls/acdc/179/)） → 2026-10-29 CFI（[acdc/190](https://forkcast.org/calls/acdc/190/)）。
- 机制与讨论：允许 0x02 validator 指定 sweep threshold，涉及 EL 请求与 CL 提款逻辑。主讨论检验 MaxEB、自动提款与 consolidation 的提前流动性风险；09 月 EF 反对评级后，10-29 ACDC 仍将其 CFI。
- ETH2030：CL 主责的条件项；EL 评估请求合约/编码，测试阈值、聚合与边界，不能按 EF 评级判为正式 DFI。
- 讨论覆盖：抓取 25/25 个 stream 对象；最新回复创建时间 2026-11-16 UTC。

## EIP-8163 — Reserve `EXTENSION (0xae)` opcode

- 状态：正文 `Review`；Meta `CFI`；最近跟踪阶段 **CFI**。[规范](https://raw.githubusercontent.com/ethereum/EIPs/ade9e9cab7e148bba4cd293f916a4a189cd7a8aa/EIPS/eip-8163.md) · [主讨论](https://ethereum-magicians.org/t/eip-8163-reserve-0xae-extension-opcode/27756)
- 阶段演进：2026-04-23 PFI（[acde/235](https://forkcast.org/calls/acde/235/)） → 2026-09-24 CFI（[acde/246](https://forkcast.org/calls/acde/246/)）。
- 机制与讨论：为其他 EVM 保留 0xae 扩展前缀；Ethereum L1 上继续 exceptional halt。主题有 L2/其他 EVM 需求与尾部 0xae 行为澄清；9 月会议从拟 DFI 转 CFI，与 EF 的反对评级不同。
- ETH2030：P2 低集成候选；区分 L1 保留与新增可执行 opcode，测试末尾字节和历史 bytecode。
- 讨论覆盖：抓取 12/12 个 stream 对象；最新回复创建时间 2026-10-05 UTC。

## EIP-8173 — Foundations of EVM Control Flow

- 状态：正文 `Review`；Meta `PFI`；最近跟踪阶段 **PFI**。[规范](https://raw.githubusercontent.com/ethereum/EIPs/ade9e9cab7e148bba4cd293f916a4a189cd7a8aa/EIPS/eip-8173.md) · [主讨论](https://ethereum-magicians.org/t/eip-8173-foundations-of-evm-control-flow/27855)
- 阶段演进：未发现对应的 Forkcast Hegotá 阶段历史；以 Meta/会议及 PR 原始记录为边界。
- 机制与讨论：EVM 静态控制流方向的总体设计/基础说明，涉及其他具体 opcode 提案。主题讨论动态跳转是否妨碍 JIT/AOT、分析复杂度与 legacy 代码；作者明确没有提议给 legacy code 增加新限制。
- ETH2030：PFI 背景/设计项；不把总体愿景当独立实现工作量，按具体消费提案验收。
- 讨论覆盖：抓取 8/8 个 stream 对象；最新回复创建时间 2026-06-17 UTC。

## EIP-8253 — Bump nonce of zero-nonce storage accounts

- 状态：正文 `Draft`；Meta `CFI`；最近跟踪阶段 **CFI**。[规范](https://raw.githubusercontent.com/ethereum/EIPs/ade9e9cab7e148bba4cd293f916a4a189cd7a8aa/EIPS/eip-8253.md) · [主讨论](https://ethereum-magicians.org/t/eip-8253-upgrade-pre-spurious-dragon-accounts/28505)
- 阶段演进：2026-05-07 PFI（[acde/236](https://forkcast.org/calls/acde/236/)） → 2026-09-24 CFI（[acde/246](https://forkcast.org/calls/acde/246/)）。
- 机制与讨论：fork 开始时对链配置指定的 empty-code/zero-nonce/nonempty-storage 账号将 nonce 设为 1。主主题只有首帖；10-22 已列入 devnet-2 增量范围。关键是每链名单由实际 pre-state 得出，不能从测试网名单复制。
- ETH2030：P2；审计每链名单、fork 一次性执行、重组及创建碰撞，提供状态证据。
- 讨论覆盖：抓取 1/1 个 stream 对象；最新回复创建时间 2026-05-12 UTC。

## EIP-8279 — Block Access List Byte Floor

- 状态：正文 `Draft`；Meta `CFI`；最近跟踪阶段 **CFI**。[规范](https://raw.githubusercontent.com/ethereum/EIPs/ade9e9cab7e148bba4cd293f916a4a189cd7a8aa/EIPS/eip-8279.md) · [主讨论](https://ethereum-magicians.org/t/eip-8279-block-access-list-byte-floor/28662)
- 阶段演进：2026-06-04 PFI（[acde/238](https://forkcast.org/calls/acde/238/)） → 2026-09-24 CFI（[acde/246](https://forkcast.org/calls/acde/246/)）。
- 机制与讨论：运行时计量 BAL 增长字节并纳入交易 floor，限制低价状态访问制造超大 BAL。07-10 回复明确要求评估 AA 影响；11-23 仍有客户端主张加入 devnet，而整体尚未同意扩大范围。
- ETH2030：P1 与 8131 配套；测试 revert、重复读写、Frames 单独预算与最大 BAL，而非只测平均交易。
- 讨论覆盖：抓取 2/2 个 stream 对象；最新回复创建时间 2026-07-10 UTC。

## EIP-8334 — Bundled Attestation Propagation

- 状态：正文 `Draft`；Meta `未列`；最近跟踪阶段 **CFI**。[规范](https://raw.githubusercontent.com/sukunrt/EIPs/f7f0cfb3a7550dbb5b77fe97fbc86a04f13efe04/EIPS/eip-8334.md) · [开放 PR](https://github.com/ethereum/EIPs/pull/11905) · [主讨论](https://ethereum-magicians.org/t/eip-8334-bundled-attestation-propagation/29008)
- 阶段演进：2026-07-23 PFI（[acdc/183](https://forkcast.org/calls/acdc/183/)） → 2026-10-29 CFI（[acdc/190](https://forkcast.org/calls/acdc/190/)）。
- 机制与讨论：用 waggle bundle 去重 attestation data 并协调 peers，属于网络传播改动。主主题只有首帖；规范尚在 PR #11905，但 ACDC #190 已 CFI。收益数字是模拟而非本仓库实测。
- ETH2030：CL 条件项；Lighthouse 做 mixed-peer、去重、非法签名及尾延迟测试；EL 不承担核心算法。
- 讨论覆盖：抓取 1/1 个 stream 对象；最新回复创建时间 2026-07-14 UTC。

## EIP-8360 — TCREATE Opcode

- 状态：正文 `Draft`；Meta `CFI`；最近跟踪阶段 **CFI**。[规范](https://raw.githubusercontent.com/ethereum/EIPs/ade9e9cab7e148bba4cd293f916a4a189cd7a8aa/EIPS/eip-8360.md) · [主讨论](https://ethereum-magicians.org/t/eip-8360-tcreate-opcode/29258)
- 阶段演进：2026-09-10 PFI（[acde/245](https://forkcast.org/calls/acde/245/)） → 2026-10-08 CFI（[acde/247](https://forkcast.org/calls/acde/247/)）。
- 机制与讨论：TCREATE 提供临时合约部署与交易末尾清理机制。08 月讨论临时账号使用 CREATE 的 nonce/地址复用；09 月作者同意澄清地址连续性不代表代码/授权连续性。11 月仍与 4758 的兼容补偿成对讨论。
- ETH2030：P2 条件项；先决定配套范围，覆盖跨交易复用、CREATE 子合约、余额、存储与代码消亡。
- 讨论覆盖：抓取 11/11 个 stream 对象；最新回复创建时间 2026-09-29 UTC。

## EIP-8365 — Disallow new 0x00 validators

- 状态：正文 `Draft`；Meta `PFI`；最近跟踪阶段 **CFI**。[规范](https://raw.githubusercontent.com/ethereum/EIPs/ade9e9cab7e148bba4cd293f916a4a189cd7a8aa/EIPS/eip-8365.md) · [主讨论](https://ethereum-magicians.org/t/eip-8365-bls-withdrawal-credential-retirement/29284)
- 阶段演进：2026-08-06 PFI（[acdc/184](https://forkcast.org/calls/acdc/184/)） → 2026-10-01 CFI（[acdc/188](https://forkcast.org/calls/acdc/188/)）。
- 机制与讨论：当前正文只阻止创建新的 0x00 validator；既有 validator 与 BLS-to-execution 变更仍保留。主题讨论 0x00 退休方向；10-29 会议对被拒 deposit 的资金处置与激活门仍有问题。标题已从广义 retirement 收窄，不能自动扩展到烧毁既有余额。
- ETH2030：CL 条件项；EL 保持现行请求语义，联测 fork 前后 deposit 与现存 key 迁移。
- 讨论覆盖：抓取 6/6 个 stream 对象；最新回复创建时间 2026-08-20 UTC。

## EIP-8368 — CPSB Recalibration for New Gas Limit

- 状态：正文 `Draft`；Meta `PFI`；最近跟踪阶段 **PFI**。[规范](https://raw.githubusercontent.com/ethereum/EIPs/ade9e9cab7e148bba4cd293f916a4a189cd7a8aa/EIPS/eip-8368.md) · [主讨论](https://ethereum-magicians.org/t/eip-8368-cpsb-recalibration-for-new-gas-limit/29293)
- 阶段演进：2026-08-13 PFI（[acde/243](https://forkcast.org/calls/acde/243/)）。
- 机制与讨论：按新参考 gas limit 重算 CPSB；正文仍明确为 placeholder，关键参数 TBD。主主题后续内容有限；ACDE #246 要求与 8372 等待 Glamsterdam 主网数据后一起判断。
- ETH2030：P2 数据门；不填拍脑袋常量，记录状态增长、成本、目标与替代方案。
- 讨论覆盖：抓取 2/2 个 stream 对象；最新回复创建时间 2026-08-19 UTC。

## EIP-8372 — Normalized state gas limit

- 状态：正文 `Draft`；Meta `PFI`；最近跟踪阶段 **PFI**。[规范](https://raw.githubusercontent.com/ethereum/EIPs/ade9e9cab7e148bba4cd293f916a4a189cd7a8aa/EIPS/eip-8372.md) · [主讨论](https://ethereum-magicians.org/t/eip-8372-normalized-state-gas-limit/29332)
- 阶段演进：2026-08-13 PFI（[acde/243](https://forkcast.org/calls/acde/243/)）。
- 机制与讨论：缩放 raw state-gas limit，并归一化区块 gasUsed，与 CPSB 选择联动。主主题只有首帖；较晚会议明确与 8368 联评，不能从“多维 gas 路线图”推出已批准。
- ETH2030：P2 数据门；回归 basefee、Frames state budget、reservoir 和最坏状态增长。
- 讨论覆盖：抓取 1/1 个 stream 对象；最新回复创建时间 2026-08-06 UTC。

## EIP-8383 — Reduce CL Block Retention Window

- 状态：正文 `Draft`；Meta `PFI`；最近跟踪阶段 **CFI**。[规范](https://raw.githubusercontent.com/ethereum/EIPs/ade9e9cab7e148bba4cd293f916a4a189cd7a8aa/EIPS/eip-8383.md) · [主讨论](https://ethereum-magicians.org/t/eip-8383-reduce-cl-block-retention-window/29449)
- 阶段演进：2026-08-20 PFI（[acdc/185](https://forkcast.org/calls/acdc/185/)） → 2026-10-29 CFI（[acdc/190](https://forkcast.org/calls/acdc/190/)）。
- 机制与讨论：缩短 CL 最低 block-serving window，依赖弱主观性与 checkpoint sync 参数。主主题只有首帖；10-29 ACDC 升为 CFI，Meta 仍 PFI。具体窗口属于 CL 保留义务，不是 EL 历史可任意删除。
- ETH2030：CL 条件项；EL 审核 ePBS 下同步与历史服务契约，测试长离线恢复。
- 讨论覆盖：抓取 1/1 个 stream 对象；最新回复创建时间 2026-08-17 UTC。

## EIP-8411 — Fast Execution Payload Broadcast

- 状态：正文 `Draft`；Meta `未列`；最近跟踪阶段 **CFI**。[规范](https://raw.githubusercontent.com/ethereum/EIPs/ade9e9cab7e148bba4cd293f916a4a189cd7a8aa/EIPS/eip-8411.md) · [主讨论](https://ethereum-magicians.org/t/eip-8411-fast-execution-payload-broadcast/29613)
- 阶段演进：2026-09-17 PFI（[acdc/187](https://forkcast.org/calls/acdc/187/)） → 2026-10-29 CFI（[acdc/190](https://forkcast.org/calls/acdc/190/)）。
- 机制与讨论：把 execution payload envelope 分片，bid 承诺 chunk Merkle root，每片可独立验证。主主题回复指向作者传播设计文章；10-29 ACDC 升 CFI。与 8146 sidecar 或 7807 SSZ 迁移均不等价。
- ETH2030：CL 主责跨层候选；EL 定义 assemble/import/backpressure 与失片恢复契约，测试恶意 chunk。
- 讨论覆盖：抓取 2/2 个 stream 对象；最新回复创建时间 2026-09-17 UTC。

## EIP-8483 — Post-Quantum Material Registration

- 状态：正文 `Draft`；Meta `未列`；最近跟踪阶段 **PFI**。[规范](https://raw.githubusercontent.com/potuz/EIPs/d62cd56be2b898ec187d4d215d20c07b052a78bb/EIPS/eip-8483.md) · [开放 PR](https://github.com/ethereum/EIPs/pull/12609) · [主讨论](https://ethereum-magicians.org/t/eip-8483-post-quantum-material-registration/30186)
- 阶段演进：2026-11-12 PFI（[acdc/191](https://forkcast.org/calls/acdc/191/)）。
- 机制与讨论：通过执行请求让 validator/builder 注册 PQ material，由 CL 保存供后续迁移。11-12 获 PFI；11-24 主讨论针对 DVT 的 VRF/XMSS 树容量、注册/更新和多层设计提出问题。PR 从早期 32-byte 描述继续演进，当前正文区分 validator 64 bytes 与 builder 32 bytes。
- ETH2030：P1 跨层影响评估；请求类型、所有权、覆盖更新、fork 转换与系统合约版本固定后再实现。
- 讨论覆盖：抓取 3/3 个 stream 对象；最新回复创建时间 2026-11-24 UTC。

## EIP-2488 — Deprecate the CALLCODE opcode

- 状态：正文 `Stagnant`；Meta `DFI`；最近跟踪阶段 **DFI**。[规范](https://raw.githubusercontent.com/ethereum/EIPs/ade9e9cab7e148bba4cd293f916a4a189cd7a8aa/EIPS/eip-2488.md) · [主讨论](https://ethereum-magicians.org/t/eip-2488-deprecate-the-callcode-opcode/3957)
- 阶段演进：2026-06-18 PFI（[acde/239](https://forkcast.org/calls/acde/239/)） → 2026-09-10 DFI（[acde/245](https://forkcast.org/calls/acde/245/)）。
- 机制与讨论：弃用 CALLCODE，涉及旧 bytecode 兼容。旧主题比较 stub 与维持原实现成本，2025 回复仍支持弃用；Hegotá 的较晚阶段为 DFI。
- ETH2030：本版本排除；不因清理容易而绕过范围流程。
- 讨论覆盖：抓取 4/4 个 stream 对象；最新回复创建时间 2025-12-14 UTC。

## EIP-7645 — Alias ORIGIN to SENDER

- 状态：正文 `Stagnant`；Meta `DFI`；最近跟踪阶段 **DFI**。[规范](https://raw.githubusercontent.com/ethereum/EIPs/ade9e9cab7e148bba4cd293f916a4a189cd7a8aa/EIPS/eip-7645.md) · [主讨论](https://ethereum-magicians.org/t/eip-7645-alias-origin-to-sender/19047)
- 阶段演进：2026-06-18 PFI（[acde/239](https://forkcast.org/calls/acde/239/)） → 2026-09-10 DFI（[acde/245](https://forkcast.org/calls/acde/245/)）。
- 机制与讨论：将 ORIGIN 与 sender/caller 语义调整，影响依赖 tx.origin 的旧合约。主讨论关注 AA 与原有授权模式；2026-06 的倡议不代表最后阶段，9 月已 DFI。
- ETH2030：排除；Frames 自身的 ORIGIN 语义按 8141 单独实现。
- 讨论覆盖：抓取 13/13 个 stream 对象；最新回复创建时间 2026-06-17 UTC。

## EIP-7666 — EVM-ify the identity precompile

- 状态：正文 `Draft`；Meta `DFI`；最近跟踪阶段 **DFI**。[规范](https://raw.githubusercontent.com/ethereum/EIPs/ade9e9cab7e148bba4cd293f916a4a189cd7a8aa/EIPS/eip-7666.md) · [主讨论](https://ethereum-magicians.org/t/eip-7561-evm-ify-the-identity-precompile/19445)
- 阶段演进：2026-07-30 PFI（[acde/242](https://forkcast.org/calls/acde/242/)） → 2026-10-08 DFI（[acde/247](https://forkcast.org/calls/acde/247/)）。
- 机制与讨论：用 EVM bytecode 代替 identity precompile。01 月回复指出原来线性计费与 memory expansion 二次成本并不等价；04 月仍有 PUSH0 前置修订。
- ETH2030：排除；任何未来 EVMification 都需逐输入行为与 gas 等价证明。
- 讨论覆盖：抓取 3/3 个 stream 对象；最新回复创建时间 2025-04-20 UTC。

## EIP-7716 — Anti-Correlation Attestation Penalties

- 状态：正文 `Draft`；Meta `DFI`；最近跟踪阶段 **DFI**。[规范](https://raw.githubusercontent.com/ethereum/EIPs/ade9e9cab7e148bba4cd293f916a4a189cd7a8aa/EIPS/eip-7716.md) · [主讨论](https://ethereum-magicians.org/t/eip-7716-anti-correlation-attestation-penalties/20137)
- 阶段演进：2026-04-16 PFI（[acdc/177](https://forkcast.org/calls/acdc/177/)） → 2026-10-15 DFI（[acdc/189](https://forkcast.org/calls/acdc/189/)）。
- 机制与讨论：对相关性较高的 attestation 故障加重惩罚。公开主主题只有首帖；阶段历史是 10-15 DFI，未找到可视为已解决经济模型与多客户端停机问题的共识。
- ETH2030：排除；不触碰 EL 或擅改惩罚模型。
- 讨论覆盖：抓取 1/1 个 stream 对象；最新回复创建时间 2024-05-27 UTC。

## EIP-7807 — SSZ execution blocks

- 状态：正文 `Draft`；Meta `DFI`；最近跟踪阶段 **DFI**。[规范](https://raw.githubusercontent.com/ethereum/EIPs/ade9e9cab7e148bba4cd293f916a4a189cd7a8aa/EIPS/eip-7807.md) · [主讨论](https://ethereum-magicians.org/t/eip-7807-ssz-execution-blocks/21580)
- 阶段演进：2026-03-12 DFI（[acde/232](https://forkcast.org/calls/acde/232/)） → 2026-07-30 PFI（[acde/242](https://forkcast.org/calls/acde/242/)） → 2026-09-10 DFI（[acde/245](https://forkcast.org/calls/acde/245/)）。
- 机制与讨论：SSZ execution blocks 统一数据结构、哈希和跨层表示。07/08 月反馈要求明确移除字段、双维 gas 与 bloom 的规范位置；曾重新 PFI，9 月再次 DFI。
- ETH2030：排除广泛格式迁移；与 Frames 当前正在讨论的哈希选择分开追踪。
- 讨论覆盖：抓取 4/4 个 stream 对象；最新回复创建时间 2026-08-19 UTC。

## EIP-7819 — SETDELEGATE instruction

- 状态：正文 `Draft`；Meta `DFI`；最近跟踪阶段 **DFI**。[规范](https://raw.githubusercontent.com/ethereum/EIPs/ade9e9cab7e148bba4cd293f916a4a189cd7a8aa/EIPS/eip-7819.md) · [主讨论](https://ethereum-magicians.org/t/eip-7819-create-delegate/21763)
- 阶段演进：2026-07-30 PFI（[acde/242](https://forkcast.org/calls/acde/242/)） → 2026-09-10 DFI（[acde/245](https://forkcast.org/calls/acde/245/)）。
- 机制与讨论：SETDELEGATE 为 code delegation 引入显式指令。讨论委托设置/撤销权限应归 factory 还是被委托代码，以及新 opcode 与扩展 CREATE 的取舍；9 月 DFI。
- ETH2030：排除；不与 8298 的真实 code adoption 混同。
- 讨论覆盖：抓取 36/36 个 stream 对象；最新回复创建时间 2026-09-22 UTC。

## EIP-7851 — Code-Controlled EOA Delegation

- 状态：正文 `Draft`；Meta `DFI`；最近跟踪阶段 **DFI**。[规范](https://raw.githubusercontent.com/ethereum/EIPs/ade9e9cab7e148bba4cd293f916a4a189cd7a8aa/EIPS/eip-7851.md) · [主讨论](https://ethereum-magicians.org/t/eip-7851-code-controlled-eoa-delegation/22344)
- 阶段演进：2026-06-04 PFI（[acde/238](https://forkcast.org/calls/acde/238/)） → 2026-09-10 DFI（[acde/245](https://forkcast.org/calls/acde/245/)）。
- 机制与讨论：通过 code-controlled delegation 退出旧 EOA 权限的一种替代设计。主题讨论 3607、deactivation marker 和 7702 相容；旧版本又影响过 8151，但 8151 后续已推广规则。
- ETH2030：排除旧替代机制；仅保留设计历史，按最终 Frames/8298 组合推进。
- 讨论覆盖：抓取 34/34 个 stream 对象；最新回复创建时间 2026-06-10 UTC。

## EIP-7862 — Delayed State Root

- 状态：正文 `Draft`；Meta `DFI`；最近跟踪阶段 **DFI**。[规范](https://raw.githubusercontent.com/ethereum/EIPs/ade9e9cab7e148bba4cd293f916a4a189cd7a8aa/EIPS/eip-7862.md) · [主讨论](https://ethereum-magicians.org/t/eip-7862-delayed-execution-layer-state-root/22559)
- 阶段演进：2026-07-02 PFI（[acde/240](https://forkcast.org/calls/acde/240/)） → 2026-09-10 DFI（[acde/245](https://forkcast.org/calls/acde/245/)）。
- 机制与讨论：延迟 EL state root，为执行/证明争取窗口。主主题后续主要说明研究与沟通渠道；其实现牵涉广泛 state-root 假设，Hegotá 已 DFI。
- ETH2030：排除；本地异步/证明原型不等于当前升级所需。
- 讨论覆盖：抓取 3/3 个 stream 对象；最新回复创建时间 2025-04-23 UTC。

## EIP-7923 — Linear, Page-Based Memory Costing

- 状态：正文 `Draft`；Meta `DFI`；最近跟踪阶段 **DFI**。[规范](https://raw.githubusercontent.com/ethereum/EIPs/ade9e9cab7e148bba4cd293f916a4a189cd7a8aa/EIPS/eip-7923.md) · [主讨论](https://ethereum-magicians.org/t/eip-linearize-memory-costing/23290)
- 阶段演进：2026-08-13 PFI（[acde/243](https://forkcast.org/calls/acde/243/)） → 2026-09-10 DFI（[acde/245](https://forkcast.org/calls/acde/245/)）。
- 机制与讨论：将 memory gas 调整为线性、分页模型。09 月回复要求充分比较多种 memory 设计及 DoS 面；不是常量小改，阶段 DFI。
- ETH2030：排除；保留性能研究，避免在 Frames 计费校准时额外改变 memory 模型。
- 讨论覆盖：抓取 34/34 个 stream 对象；最新回复创建时间 2026-09-21 UTC。

## EIP-8025 — Optional Execution Proofs

- 状态：正文 `Draft`；Meta `DFI`；最近跟踪阶段 **DFI**。[规范](https://raw.githubusercontent.com/ethereum/EIPs/ade9e9cab7e148bba4cd293f916a4a189cd7a8aa/EIPS/eip-8025.md) · [主讨论](https://ethereum-magicians.org/t/eip-optional-execution-proofs/25500)
- 阶段演进：2026-05-14 PFI（[acdc/178](https://forkcast.org/calls/acdc/178/)） → 2026-10-15 DFI（[acdc/189](https://forkcast.org/calls/acdc/189/)）。
- 机制与讨论：允许执行证明和验证工作流，推进 stateless/zkVM 互通。主讨论询问证明系统与 guest program 版本如何显式标识；EF A 评级之后，ACDC #189 仍 DFI。
- ETH2030：排除本 fork；未来单独定义可信 guest、VK、故障恢复与验证矩阵。
- 讨论覆盖：抓取 2/2 个 stream 对象；最新回复创建时间 2026-05-18 UTC。

## EIP-8094 — eth/vhash - Blob-Aware Mempool

- 状态：正文 `Draft`；Meta `DFI`；最近跟踪阶段 **DFI**。[规范](https://raw.githubusercontent.com/ethereum/EIPs/ade9e9cab7e148bba4cd293f916a4a189cd7a8aa/EIPS/eip-8094.md) · [主讨论](https://ethereum-magicians.org/t/eip-8094-eth-vhash-blob-aware-mempool/26834)
- 阶段演进：2026-08-13 PFI（[acde/243](https://forkcast.org/calls/acde/243/)） → 2026-09-10 DFI（[acde/245](https://forkcast.org/calls/acde/245/)）。
- 机制与讨论：blob-aware mempool / eth-vhash 传播与验证跟踪。主主题指向验证跟踪澄清 PR；没有证据将其与本版本 Frames 公共池规则合并批准。
- ETH2030：排除；blobpool 既有代码另做当前前置规范审计。
- 讨论覆盖：抓取 2/2 个 stream 对象；最新回复创建时间 2025-12-10 UTC。

## EIP-8105 — Universal Enshrined Encrypted Mempool

- 状态：正文 `Draft`；Meta `未列`；最近跟踪阶段 **撤回**。[规范](https://raw.githubusercontent.com/ethereum/EIPs/ade9e9cab7e148bba4cd293f916a4a189cd7a8aa/EIPS/eip-8105.md) · [主讨论](https://ethereum-magicians.org/t/eip-8105-universal-enshrined-encrypted-mempool/27201)
- 阶段演进：日期未登记 PFI → 2026-02-23 撤回。
- 机制与讨论：协议内置、可插拔 key provider 的 encrypted mempool。后续回复讨论 timelock/受信集合与 CR，但阶段历史记录 02-23 撤回。主文件仍 Draft 与本 fork 撤回不矛盾。
- ETH2030：本版本排除；保留研究，不把尚有主文件当仍在升级队列。
- 讨论覆盖：抓取 3/3 个 stream 对象；最新回复创建时间 2026-08-14 UTC。

## EIP-8115 — Batch priority fees at end of block

- 状态：正文 `Draft`；Meta `DFI`；最近跟踪阶段 **DFI**。[规范](https://raw.githubusercontent.com/ethereum/EIPs/ade9e9cab7e148bba4cd293f916a4a189cd7a8aa/EIPS/eip-8115.md) · [主讨论](https://ethereum-magicians.org/t/eip-8115-batch-priority-fees-at-end-of-block/27358)
- 阶段演进：2026-07-30 PFI（[acde/242](https://forkcast.org/calls/acde/242/)） → 2026-09-10 DFI（[acde/245](https://forkcast.org/calls/acde/245/)）。
- 机制与讨论：把 priority fee 结算移到 block 末尾。主主题只有首帖；较晚范围 DFI。可能改变执行中观察余额，不能当纯实现优化。
- ETH2030：排除；遵守现行 fee recipient 结算顺序。
- 讨论覆盖：抓取 1/1 个 stream 对象；最新回复创建时间 2025-12-30 UTC。

## EIP-8142 — Block-in-Blobs (BiB)

- 状态：正文 `Draft`；Meta `PFI`；最近跟踪阶段 **DFI**。[规范](https://raw.githubusercontent.com/ethereum/EIPs/ade9e9cab7e148bba4cd293f916a4a189cd7a8aa/EIPS/eip-8142.md) · [主讨论](https://ethereum-magicians.org/t/eip-8142-block-in-blobs-bib/27621)
- 阶段演进：2026-08-06 PFI（[acdc/184](https://forkcast.org/calls/acdc/184/)） → 2026-09-21 DFI。
- 机制与讨论：Block-in-Blobs 将 execution payload 数据可用性绑定到 blob commitments，为不下载执行体的验证准备 DA。主主题只有首帖；较晚阶段历史已 DFI，而 Meta 仍 PFI。不能把证明正确性直接等同数据可用性。
- ETH2030：Hegotá 排除；保留未来 DA/证明研究，不把本地 BiB 原型计入版本完成率。
- 讨论覆盖：抓取 1/1 个 stream 对象；最新回复创建时间 2026-01-29 UTC。

## EIP-8146 — Block Access List Sidecars

- 状态：正文 `Draft`；Meta `DFI`；最近跟踪阶段 **DFI**。[规范](https://raw.githubusercontent.com/ethereum/EIPs/ade9e9cab7e148bba4cd293f916a4a189cd7a8aa/EIPS/eip-8146.md) · [主讨论](https://ethereum-magicians.org/t/eip-8146-block-access-list-sidecars/27757)
- 阶段演进：2026-07-02 PFI（[acde/240](https://forkcast.org/calls/acde/240/)） → 2026-09-17 DFI（[acdc/187](https://forkcast.org/calls/acdc/187/)）。
- 机制与讨论：把 BAL 作为独立 sidecar 传播，增加可用性与时限约束。07-10 回复探索复用 BiB/PeerDAS 传播而不是新 topic；EF B 评级后正式阶段已 DFI。
- ETH2030：排除；保留本地 BAL 实现与 sidecar 激活的边界。
- 讨论覆盖：抓取 2/2 个 stream 对象；最新回复创建时间 2026-07-10 UTC。

## EIP-8182 — Private ETH and ERC-20 Transfers

- 状态：正文 `Review`；Meta `DFI`；最近跟踪阶段 **DFI**。[规范](https://raw.githubusercontent.com/ethereum/EIPs/ade9e9cab7e148bba4cd293f916a4a189cd7a8aa/EIPS/eip-8182.md) · [主讨论](https://ethereum-magicians.org/t/eip-8182-private-eth-and-erc-20-transfers/27889)
- 阶段演进：2026-05-21 PFI（[acde/237](https://forkcast.org/calls/acde/237/)） → 2026-09-10 DFI（[acde/245](https://forkcast.org/calls/acde/245/)）。
- 机制与讨论：协议内置 ETH/ERC-20 隐私转账池与特定密码学设施。09 月讨论是否应先在应用层部署、谁承担 setup/audit/维护成本；作者坚持协议级路径有独特作用，但范围已 DFI。
- ETH2030：排除；隐私基础通过 Frames/nonce/roots 评估，不承诺具体池。
- 讨论覆盖：抓取 32/32 个 stream 对象；最新回复创建时间 2026-09-16 UTC。

## EIP-8184 — LUCID encrypted mempool

- 状态：正文 `Draft`；Meta `未列`；最近跟踪阶段 **DFI**。[规范](https://raw.githubusercontent.com/ethereum/EIPs/ade9e9cab7e148bba4cd293f916a4a189cd7a8aa/EIPS/eip-8184.md) · [主讨论](https://ethereum-magicians.org/t/eip-8184-lucid-encrypted-mempool/28017)
- 阶段演进：2026-02-12 PFI（[acde/230](https://forkcast.org/calls/acde/230/)） → 2026-03-12 DFI（[acde/232](https://forkcast.org/calls/acde/232/)）。
- 机制与讨论：LUCID 用先承诺后揭示的加密交易路径降低 MEV 信息泄露。08 月作者指出 PQ crypto 与 sender reveal optionality 的阻碍；09 月有替代观点，未构成重新纳入决定。
- ETH2030：本版本排除；主题在 fork 阶段登记与 EIP 创建时间上还有历史先后差异，只保留来源原值。
- 讨论覆盖：抓取 3/3 个 stream 对象；最新回复创建时间 2026-09-07 UTC。

## EIP-8188 — Last-Written Block for Accounts and Slots

- 状态：正文 `Draft`；Meta `DFI`；最近跟踪阶段 **DFI**。[规范](https://raw.githubusercontent.com/ethereum/EIPs/ade9e9cab7e148bba4cd293f916a4a189cd7a8aa/EIPS/eip-8188.md) · [主讨论](https://ethereum-magicians.org/t/eip8188-state-tiering-by-write-age/28234)
- 阶段演进：2026-05-21 PFI（[acde/237](https://forkcast.org/calls/acde/237/)） → 2026-09-10 DFI（[acde/245](https://forkcast.org/calls/acde/245/)）。
- 机制与讨论：为账号/slot 记录 last-written 信息，支持冷热 state 分层与收费。05 月回复质疑 never-initialized 写入与频繁只读旧数据如何定价；状态生命周期问题未因此解决。
- ETH2030：排除；等待未来 state/trie 方案，不把现有 pruner 视为已实现本提案。
- 讨论覆盖：抓取 6/6 个 stream 对象；最新回复创建时间 2026-05-29 UTC。

## EIP-8198 — Quick Slots

- 状态：正文 `Draft`；Meta `CFI`；最近跟踪阶段 **DFI**。[规范](https://raw.githubusercontent.com/ethereum/EIPs/ade9e9cab7e148bba4cd293f916a4a189cd7a8aa/EIPS/eip-8198.md) · [主讨论](https://ethereum-magicians.org/t/eip-8198-quick-slots/28057)
- 阶段演进：2026-08-06 PFI（[acdc/184](https://forkcast.org/calls/acdc/184/)） → 2026-10-01 CFI（[acdc/188](https://forkcast.org/calls/acdc/188/)） → 2026-11-12 DFI（[acdc/191](https://forkcast.org/calls/acdc/191/)）。
- 机制与讨论：Quick Slots 改变 slot 时长与依赖时序参数，影响 CL/EL/费用及生态。主讨论比较 schedule 与固定 fork-local 参数；11-12 ACDC #191 及作者留言撤回其 Hegotá 推进，以聚焦 PQ。Meta 的 CFI 已落后。
- ETH2030：本版本不新增承诺；保留既有研究，移除其作为 Frames/roots 实施前提的假设。
- 讨论覆盖：抓取 6/6 个 stream 对象；最新回复创建时间 2026-09-24 UTC。

## EIP-8200 — EVMification

- 状态：正文 `Draft`；Meta `DFI`；最近跟踪阶段 **DFI**。[规范](https://raw.githubusercontent.com/ethereum/EIPs/ade9e9cab7e148bba4cd293f916a4a189cd7a8aa/EIPS/eip-8200.md) · [主讨论](https://ethereum-magicians.org/t/eip-8200-evmification/28036)
- 阶段演进：2026-07-30 PFI（[acde/242](https://forkcast.org/calls/acde/242/)） → 2026-09-10 DFI（[acde/245](https://forkcast.org/calls/acde/245/)）。
- 机制与讨论：以 EVM 代码替代部分 precompile 的 EVMification 框架。主题讨论是否应保留常数计费及按真实复杂度收费；规范/实现等价与证明端成本需独立处理。
- ETH2030：排除；避免与 Frames 增加的验证方案一并扩大范围。
- 讨论覆盖：抓取 6/6 个 stream 对象；最新回复创建时间 2026-06-25 UTC。

## EIP-8205 — Withdrawal credentials preregistration

- 状态：正文 `Draft`；Meta `PFI`；最近跟踪阶段 **DFI**。[规范](https://raw.githubusercontent.com/ethereum/EIPs/ade9e9cab7e148bba4cd293f916a4a189cd7a8aa/EIPS/eip-8205.md) · [主讨论](https://ethereum-magicians.org/t/eip-8205-withdrawal-credentials-preregistration/28084)
- 阶段演进：2026-04-16 PFI（[acdc/177](https://forkcast.org/calls/acdc/177/)） → 2026-10-29 DFI（[acdc/190](https://forkcast.org/calls/acdc/190/)）。
- 机制与讨论：为 withdrawal credentials 预注册以防 deposit front-running，新增请求与 CL 处理。主题讨论 BLS 依赖与后量子方向，11 月仍有社区支持；但 ACDC #190 已决定把问题交给 PQ deposit 重设计，Meta 的 PFI 落后。
- ETH2030：Hegotá 排除；保留安全问题与未来设计关联，不因新回复自动重新纳入。
- 讨论覆盖：抓取 15/15 个 stream 对象；最新回复创建时间 2026-11-18 UTC。

## EIP-8219 — Checked Arithmetic Opcodes

- 状态：正文 `Draft`；Meta `DFI`；最近跟踪阶段 **DFI**。[规范](https://raw.githubusercontent.com/ethereum/EIPs/ade9e9cab7e148bba4cd293f916a4a189cd7a8aa/EIPS/eip-8219.md) · [主讨论](https://ethereum-magicians.org/t/eip-8219-checked-arithmetic-opcodes/27913)
- 阶段演进：2026-08-13 PFI（[acde/243](https://forkcast.org/calls/acde/243/)） → 2026-09-10 DFI（[acde/245](https://forkcast.org/calls/acde/245/)）。
- 机制与讨论：添加 checked arithmetic 指令或相应执行机制。07 月讨论独立 opcode 与模式 flag、条件分支和 gas 成本；尚不能认定语言工具链已统一。
- ETH2030：排除；控制 VM 新语义和编译器联动面。
- 讨论覆盖：抓取 9/9 个 stream 对象；最新回复创建时间 2026-07-03 UTC。

## EIP-8237 — Independent CL/EL Sync

- 状态：正文 `Draft`；Meta `DFI`；最近跟踪阶段 **DFI**。[规范](https://raw.githubusercontent.com/ethereum/EIPs/ade9e9cab7e148bba4cd293f916a4a189cd7a8aa/EIPS/eip-8237.md) · [主讨论](https://ethereum-magicians.org/t/eip-8237-independent-cl-el-sync/28331)
- 阶段演进：2026-04-23 PFI（[acde/235](https://forkcast.org/calls/acde/235/)） → 2026-09-17 DFI（[acdc/187](https://forkcast.org/calls/acdc/187/)）。
- 机制与讨论：CL/EL 独立同步以减少重复 payload 下载。09 月主题深入讨论 commitment accumulator、payload 服务责任与 Engine 绑定；已有想法不等于同步安全模型已通过。
- ETH2030：排除；同步与历史恢复作为正确性基线继续测试，不擅自丢弃数据。
- 讨论覆盖：抓取 10/10 个 stream 对象；最新回复创建时间 2026-09-24 UTC。

## EIP-8243 — Batching Attestations at Source

- 状态：正文 `Draft`；Meta `DFI`；最近跟踪阶段 **DFI**。[规范](https://raw.githubusercontent.com/ethereum/EIPs/ade9e9cab7e148bba4cd293f916a4a189cd7a8aa/EIPS/eip-8243.md) · [主讨论](https://ethereum-magicians.org/t/eip-8243-batching-attestations-at-source/28606)
- 阶段演进：2026-06-11 PFI（[acdc/180](https://forkcast.org/calls/acdc/180/)） → 2026-09-17 DFI（[acdc/187](https://forkcast.org/calls/acdc/187/)）。
- 机制与讨论：在发送端批量传播 attestations。06 月回复质疑用 opt-in 解法是否解决既有 opt-in 采用慢的问题；与 8334 bundled propagation 是不同提案。
- ETH2030：排除；CL 网络候选按 8334 的独立阶段跟进。
- 讨论覆盖：抓取 2/2 个 stream 对象；最新回复创建时间 2026-06-30 UTC。

## EIP-8304 — Trustless log and transaction index

- 状态：正文 `Draft`；Meta `DFI`；最近跟踪阶段 **DFI**。[规范](https://raw.githubusercontent.com/ethereum/EIPs/ade9e9cab7e148bba4cd293f916a4a189cd7a8aa/EIPS/eip-8304.md) · [主讨论](https://ethereum-magicians.org/t/eip-8304-trustless-log-and-transaction-index/28824)
- 阶段演进：2026-06-18 PFI（[acde/239](https://forkcast.org/calls/acde/239/)） → 2026-09-24 DFI（[acde/246](https://forkcast.org/calls/acde/246/)）。
- 机制与讨论：为 log/transaction 提供可验证索引。11-22 作者给出验证开销估计并称可先作为本地 Geth 功能发布；这是实现者观点，未重新 CFI。
- ETH2030：排除协议激活；本地索引优化可另开性能任务，不算本 fork 内容。
- 讨论覆盖：抓取 19/19 个 stream 对象；最新回复创建时间 2026-11-22 UTC。

## EIP-8321 — Hash-Chain RANDAO

- 状态：正文 `Draft`；Meta `DFI`；最近跟踪阶段 **DFI**。[规范](https://raw.githubusercontent.com/ethereum/EIPs/ade9e9cab7e148bba4cd293f916a4a189cd7a8aa/EIPS/eip-8321.md) · [主讨论](https://ethereum-magicians.org/t/eip-8321-hash-chain-randao/28942)
- 阶段演进：2026-07-23 PFI（[acdc/183](https://forkcast.org/calls/acdc/183/)） → 2026-09-17 DFI（[acdc/187](https://forkcast.org/calls/acdc/187/)）。
- 机制与讨论：以 hash chain 增强 RANDAO 抗量子方向。11 月仍有存储与阈值方案询问，不能把恢复讨论当作重新纳入；正式阶段 DFI。
- ETH2030：排除；等待完整 PQ 共识设计，不局部抢跑。
- 讨论覆盖：抓取 8/8 个 stream 对象；最新回复创建时间 2026-11-18 UTC。

## EIP-8333 — Align Checkpoint with Epoch Boundary Block

- 状态：正文 `Withdrawn`；Meta `DFI`；最近跟踪阶段 **DFI**。[规范](https://raw.githubusercontent.com/ethereum/EIPs/ade9e9cab7e148bba4cd293f916a4a189cd7a8aa/EIPS/eip-8333.md) · [主讨论](https://ethereum-magicians.org/t/eip-8333-align-checkpoint-with-epoch-boundary-block/29003)
- 阶段演进：2026-08-06 PFI（[acdc/184](https://forkcast.org/calls/acdc/184/)） → 2026-10-01 DFI（[acdc/188](https://forkcast.org/calls/acdc/188/)）。
- 机制与讨论：调整 checkpoint 与 epoch 边界 block 的对应。主题讨论 ePBS 下最后 payload 与 consensus block 的 finalized 关系；当前 EIP 文档为 Withdrawn，Hegotá 阶段 DFI。
- ETH2030：排除；不把它与本项目已有 finality 模块关联成版本要求。
- 讨论覆盖：抓取 7/7 个 stream 对象；最新回复创建时间 2026-08-25 UTC。

## EIP-8341 — Partial Execution Payload Commitments

- 状态：正文 `Draft`；Meta `未列`；最近跟踪阶段 **DFI**。[规范](https://raw.githubusercontent.com/nerolation/EIPs/7c0a835f2be7a1678990473a20212fb97e3ed017/EIPS/eip-8341.md) · [开放 PR](https://github.com/ethereum/EIPs/pull/11936) · [主讨论](https://ethereum-magicians.org/t/eip-8341-partial-execution-payload-commitments/29030)
- 阶段演进：2026-07-23 PFI（[acdc/183](https://forkcast.org/calls/acdc/183/)） → 2026-09-17 DFI（[acdc/187](https://forkcast.org/calls/acdc/187/)）。
- 机制与讨论：ePBS bid 的部分 payload commitment 排除 state root，让 builder 延后计算它。主主题只有首帖，设计作为 7862 替代；PR 仍开放不表示 Hegotá 未拒绝，ACDC #187 已 DFI。
- ETH2030：排除；不要与近期 Glamsterdam payload_root 安全修订混同。
- 讨论覆盖：抓取 1/1 个 stream 对象；最新回复创建时间 2026-07-17 UTC。

## EIP-8355 — Precompiles for ML-DSA Verification

- 状态：正文 `Draft`；Meta `DFI`；最近跟踪阶段 **DFI**。[规范](https://raw.githubusercontent.com/ethereum/EIPs/ade9e9cab7e148bba4cd293f916a4a189cd7a8aa/EIPS/eip-8355.md) · [主讨论](https://ethereum-magicians.org/t/eip-8355-precompiles-for-ml-dsa-verification/29211)
- 阶段演进：2026-08-13 PFI（[acde/243](https://forkcast.org/calls/acde/243/)） → 2026-10-08 DFI（[acde/247](https://forkcast.org/calls/acde/247/)）。
- 机制与讨论：ML-DSA verification precompiles，特定 PQ 签名方案的原生入口。11 月回复讨论 FIPS/上下文字符串及 JOSE/COSE 的兼容；主主题继续活跃，但 10-08 阶段已 DFI。
- ETH2030：排除本 fork 预编译承诺；保持 Frames 的方案可编程性，标准化与验证器性能另评。
- 讨论覆盖：抓取 20/20 个 stream 对象；最新回复创建时间 2026-11-11 UTC。

## EIP-8358 — Net Gas Metering for Account Changes

- 状态：正文 `Draft`；Meta `未列`；最近跟踪阶段 **DFI**。[规范](https://raw.githubusercontent.com/ethereum/EIPs/ade9e9cab7e148bba4cd293f916a4a189cd7a8aa/EIPS/eip-8358.md) · [主讨论](https://ethereum-magicians.org/t/eip-8358-net-gas-metering-for-account-changes/29304)
- 阶段演进：2026-08-13 PFI（[acde/243](https://forkcast.org/calls/acde/243/)） → 2026-09-10 DFI（[acde/245](https://forkcast.org/calls/acde/245/)）。
- 机制与讨论：账号余额变化 net metering，调整 transfer stipend 等细节。08 月 #2–6 对 stipend 最低转发量、unused gas 与 base charge 有实质澄清；正式阶段 DFI。
- ETH2030：排除；避免再扩大 Frames/state gas 的计费变更。
- 讨论覆盖：抓取 6/6 个 stream 对象；最新回复创建时间 2026-08-13 UTC。

## EIP-8359 — Beacon Block Reporting Field

- 状态：正文 `Draft`；Meta `未列`；最近跟踪阶段 **DFI**。[规范](https://raw.githubusercontent.com/hanniabu/EIPs/06ee26f2b86f36c839f9c41d99fcf49ab8323e5d/EIPS/eip-8359.md) · [开放 PR](https://github.com/ethereum/EIPs/pull/12063) · [主讨论](https://ethereum-magicians.org/t/eip-8359-beacon-block-reporting-field/29224)
- 阶段演进：2026-08-06 PFI（[acdc/184](https://forkcast.org/calls/acdc/184/)） → 2026-10-15 DFI（[acdc/189](https://forkcast.org/calls/acdc/189/)）。
- 机制与讨论：新增 CL reporting 字段供 validator 自报客户端与 setup。主题仅有一条支持回复，自报信息的真实性与隐私不能据此保证；ACDC #189 已 DFI。
- ETH2030：排除；运维可观测性使用链下工具另做。
- 讨论覆盖：抓取 2/2 个 stream 对象；最新回复创建时间 2026-09-07 UTC。

## EIP-8363 — Tapered Issuance Burn

- 状态：正文 `Draft`；Meta `DFI`；最近跟踪阶段 **DFI**。[规范](https://raw.githubusercontent.com/ethereum/EIPs/ade9e9cab7e148bba4cd293f916a4a189cd7a8aa/EIPS/eip-8363.md) · [主讨论](https://ethereum-magicians.org/t/eip-8363-tapered-issuance-burn/29263)
- 阶段演进：2026-08-06 PFI（[acdc/184](https://forkcast.org/calls/acdc/184/)） → 2026-10-01 DFI（[acdc/188](https://forkcast.org/calls/acdc/188/)）。
- 机制与讨论：通过 issuance/burn 机制改变质押回报与总量路径。长主题持续争论安全预算、solo staker、集中化与决策合法性，10 月末仍有强烈反对；不能把大量讨论当技术通过。
- ETH2030：排除；经济政策需更广泛生态过程，不由 EL 开发计划自行裁决。
- 讨论覆盖：抓取 228/228 个 stream 对象；最新回复创建时间 2026-10-31 UTC。

## EIP-8367 — Balance sunset for retired BLS validators

- 状态：正文 `Draft`；Meta `未列`；最近跟踪阶段 **DFI**。[规范](https://raw.githubusercontent.com/ensi321/EIPs/c55245baa104ff8f6e02285cb9e58d66640cf829/EIPS/eip-8367.md) · [开放 PR](https://github.com/ethereum/EIPs/pull/12099) · [主讨论](https://ethereum-magicians.org/t/eip-8367-balance-sunset-for-retired-bls-validators/29299)
- 阶段演进：2026-08-06 PFI（[acdc/184](https://forkcast.org/calls/acdc/184/)） → 2026-09-17 DFI（[acdc/187](https://forkcast.org/calls/acdc/187/)）。
- 机制与讨论：为退休 BLS validator 设置逐步余额 sunset。08 月回复争论丢失 withdrawal mnemonic、原始 depositor 是否等于 owner，以及是否应销减余额；作者指出操作方不必然是资金所有者。
- ETH2030：排除；不得从 8365 停止新建推导出清零旧余额。
- 讨论覆盖：抓取 5/5 个 stream 对象；最新回复创建时间 2026-08-20 UTC。

## EIP-8371 — RowDAS - Distributed Blob Reconstruction

- 状态：正文 `Draft`；Meta `DFI`；最近跟踪阶段 **DFI**。[规范](https://raw.githubusercontent.com/ethereum/EIPs/ade9e9cab7e148bba4cd293f916a4a189cd7a8aa/EIPS/eip-8371.md) · [主讨论](https://ethereum-magicians.org/t/eip-8371-rowdas-distributed-blobspace-reconstruction/29320)
- 阶段演进：2026-08-06 PFI（[acdc/184](https://forkcast.org/calls/acdc/184/)） → 2026-10-15 DFI（[acdc/189](https://forkcast.org/calls/acdc/189/)）。
- 机制与讨论：RowDAS 分布式重建 blob，改善数据重建负载。10-15 作者补充测量文章，存在实测资料不改变 ACDC #189 的 DFI；成本收益与当前瓶颈需另算。
- ETH2030：排除；不把 blob 吞吐增长自动视为 Hegotá 已定。
- 讨论覆盖：抓取 2/2 个 stream 对象；最新回复创建时间 2026-10-15 UTC。

## EIP-8374 — Persist Warm Access Sets Across Reverts

- 状态：正文 `Draft`；Meta `未列`；最近跟踪阶段 **DFI**。[规范](https://raw.githubusercontent.com/ethereum/EIPs/ade9e9cab7e148bba4cd293f916a4a189cd7a8aa/EIPS/eip-8374.md) · [主讨论](https://ethereum-magicians.org/t/eip-8374-persist-warm-access-sets-across-reverts/29341)
- 阶段演进：2026-08-13 PFI（[acde/243](https://forkcast.org/calls/acde/243/)） → 2026-10-08 DFI（[acde/247](https://forkcast.org/calls/acde/247/)）。
- 机制与讨论：revert 后保留已 warm 的地址与 storage access sets。主主题只有首帖；EF 评级把它与 net metering 联系，但 ACD 阶段已 DFI。改变 revert gas 语义需独立兼容分析。
- ETH2030：排除；Frames 批次回滚沿选定的现行规则实现。
- 讨论覆盖：抓取 1/1 个 stream 对象；最新回复创建时间 2026-08-07 UTC。

## EIP-8375 — ePBS Mandatory Burn of Execution Rewards

- 状态：正文 `Draft`；Meta `未列`；最近跟踪阶段 **DFI**。[规范](https://raw.githubusercontent.com/benaadams/EIPs/b2bb0ef69e4b236703c30468d5773e896410a679/EIPS/eip-8375.md) · [开放 PR](https://github.com/ethereum/EIPs/pull/12130) · [主讨论](https://ethereum-magicians.org/t/eip-8375-ember-epbs-mandatory-burn-of-execution-rewards/29380)
- 阶段演进：2026-08-13 PFI（[acde/243](https://forkcast.org/calls/acde/243/)） → 2026-09-17 DFI（[acdc/187](https://forkcast.org/calls/acdc/187/)）。
- 机制与讨论：ePBS 下对 execution rewards 与 builder bid 执行 burn，并涉及竞争 floor。主主题只有首帖；PR 在完善 floor/credit/coalition 风险，并未改变 DFI。协议收益与权利分配不能只当结算优化。
- ETH2030：排除；不改 proposer/builder 奖励或经济政策。
- 讨论覆盖：抓取 1/1 个 stream 对象；最新回复创建时间 2026-08-10 UTC。

## EIP-8379 — Top-up Sync

- 状态：正文 `Draft`；Meta `DFI`；最近跟踪阶段 **DFI**。[规范](https://raw.githubusercontent.com/ethereum/EIPs/ade9e9cab7e148bba4cd293f916a4a189cd7a8aa/EIPS/eip-8379.md) · [主讨论](https://ethereum-magicians.org/t/eip-8379-top-up-sync/29405)
- 阶段演进：2026-08-20 PFI（[acdc/185](https://forkcast.org/calls/acdc/185/)） → 2026-10-01 DFI（[acdc/188](https://forkcast.org/calls/acdc/188/)）。
- 机制与讨论：Top-up Sync 方向的 CL 同步改动。主主题只有首帖；10-01 阶段 DFI，缺少可据以宣布后续共识反转的公开回复。
- ETH2030：排除；未来根据同步设计重新评估。
- 讨论覆盖：抓取 1/1 个 stream 对象；最新回复创建时间 2026-08-12 UTC。

## 无编号方向与前置背景

**Partial Reconstruction and 2D PeerDAS** 仍出现在 Forkcast 的 pending proposals 中，没有在本登记虚构 EIP 编号或通过阶段。[原始提案主题](https://ethereum-magicians.org/t/hegota-headliner-partial-reconstruction-and-2d-peerdas/27652)。它与 RowDAS 的成本收益有关，但不因此变成本 fork 必交；需要单独的规范、范围决定及实测。

过程与前置背景另查：

- [EIP-7723](https://eips.ethereum.org/EIPS/eip-7723)：升级阶段定义；主讨论涉及正式状态与各客户端准备程度如何区分。
- [EIP-8081](https://eips.ethereum.org/EIPS/eip-8081)：范围索引；[主讨论](https://ethereum-magicians.org/t/eip-8081-hegota-network-upgrade-meta-thread/26876)包含 EF 架构、Lido、Serenita、Ethlabs、Lodestar、Prysm 等不同立场；不能把任一团队 tier 当总体共识。
- [EIP-7773](https://eips.ethereum.org/EIPS/eip-7773)：Glamsterdam 前置；Hegotá 基于其最终规则，而非仓库早期实验命名。
- [EIP-7732](https://eips.ethereum.org/EIPS/eip-7732)：ePBS 时序、builder bid、payload reveal 与 fork choice，是 FOCIL 集成的重要约束。
- [EIP-7928](https://eips.ethereum.org/EIPS/eip-7928)：BAL 的记录、排序、验证与状态重建；有 commitment 不等于已证明 diff 正确。

以上五项的正文版本、主题链接与抓取覆盖也记录在来源清单中，但不重复计入 72 项候选研究数。
