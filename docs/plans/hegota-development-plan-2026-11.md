# ETH2030 Hegotá 完整开发计划

> 版本：v1.1；制定日期与研究截点：2026-11-25（Asia/Shanghai）；A 组实施记录已补齐。
> 本文是后续实施的主计划，覆盖规范、实现、兼容性、测试、跨客户端联调、发布与维护。A 组 7 项基线任务已交付并验证，其余功能任务仍按下述规范和验收门实施。
> 依据：[宏观研究](hegota-macro-plan-2026-11.md)、[72 项 EIP 与讨论登记](hegota-eip-review-2026-11.md)、[固定来源与版本](hegota-sources-2026-11.json)。A 组的当前交付见 [实施报告](hegota-a/README.md)。

## 1. 交付目标与完成定义

交付一个基于**选定官方规范组合**、可由原生 ETH2030 节点完整运行的 Hegotá EL 版本。它必须从公开交易入口完成交易传播、构块、Engine API 导入、状态执行、收据查询、同步、重组和重启恢复，并与独立 EL/CL 实现得到一致结果。

验收对象是实际发布二进制与实际容器入口；helper、实验模块、geth 包装路径的通过结果分别记录。版本完成需要同时满足：

1. 所有进入目标网络范围的 EIP 都有固定规范版本、逐规则符合性记录、实现位置和测试向量。
2. Frames 核心及纳入的伴随提案完成公开入口端到端验证；协议有效性、公开 mempool 策略、FOCIL 资格不混用。
3. FOCIL 按该目标网络的正式范围执行；若保留，必须验证 IL 约束实际进入 CL fork choice；若正式延期，提供更新后的范围记录与对应配置。
4. 与独立参考实现比较 state root、receipt root、gas、BAL 和有效/无效分类；没有未解释的共识差异。
5. 必需测试全部执行通过，缺 fixture、超时、跳过、没有交易或只检查区块高度均不计作通过。
6. fork 边界、数据库恢复、历史查询、资源上限、可重现构建、发布说明与故障处置齐备。

不以仓库包数量、旧测试总数或历史路线图百分比作为完成指标。

## 2. 规范与范围管理

### 2.1 当前基线

本计划复用 2026-11-25 的研究快照：EIPs `ade9e9cab7e148bba4cd293f916a4a189cd7a8aa`，consensus-specs `d2021d2719235f0df4726a1d330502cb15aa8ea3`。尚未合入的提案以来源清单中的 PR head SHA 为准。实际实施前，HGT-001 还必须选定相容的 EELS、EEST、execution-apis、CL、测试工具和 devnet 配置版本。

计划编制时，本地 `refs/EIPs` 与 `refs/consensus-specs` 还是 2 月快照。2026-11-25 已按用户指令分别更新到上游 `master` 的 `ade9e9cab7e148bba4cd293f916a4a189cd7a8aa` 与 `d2021d2719235f0df4726a1d330502cb15aa8ea3`，与本计划采用的研究版本一致。参考正文继续只读；后续引用更新记录 SHA/校验值，不修改上游正文来迁就现有实现。历史研究清单中的旧本地引用保留为当时基线。

[EIP-8081](https://eips.ethereum.org/EIPS/eip-8081) 仍登记 7805、8141 为 SFI；较晚会议对 8015 等存在更新，详见逐项登记。2026-11-26 ACDC #192 尚未举行，不能把其可能结果用于本计划的已决事项。[会议议程](https://github.com/ethereum/pm/issues/2305)

### 2.2 执行分组

| 分组 | 内容 | 执行方式 |
| --- | --- | --- |
| 基础主线 | 规范锁定、真实调用链、fork 配置、Frames、状态/收据/BAL、mempool/RPC、恢复与验收 | 可立即准备；改变共识行为前须固定对应规范 |
| 当前 SFI 的 FOCIL | 7805 的 EL/CL 联调与 fork-choice 验证 | 完整列入计划；按当前 SFI 准备，明确延期决定出现后才调整交付范围 |
| Frames 伴随候选 | 8250、8272 | 优先设计与向量准备；按最终组合/目标 devnet 范围实施与激活 |
| PQ 与账号迁移候选 | 8504、8298、8151、8483 | 设置接口与兼容性门；条件不成熟时可开发受控原型，不对外声称已激活 |
| FOCIL AA 资格模型 | 8369 | 先形成可执行规范与验证职责；Informational 正文不能直接当共识激活依据 |
| 其他未定候选 | 7906、资源 floor、EVM/收据/网络/CL 清理等 | 每项有独立任务、成本与准入结论，禁止整批默认启用 |
| 已拒绝/撤回或远期 | 逐项登记中的 DFI/撤回项、完整 8288 聚合等 | 记录处置；重新纳入需要新范围证据，研究工作不计入本版本完成率 |

**发布范围判定：** 以“网络 + fork 名 + 激活点 + 规范组合 + EIP 集合”描述目标。不把实验 feature flag 当成任意改变公共网络规则的开关。FOCIL 当前的 SFI 不能仅因讨论延期就自行关闭，也不要求上游重复确认已有决定后才能开展独立工作。

### 2.3 必须先解决的规范决策

| 决策编号 | 问题 | 负责角色 / 影响 | 关闭条件 |
| --- | --- | --- | --- |
| D01 | 8141、8250 的费用与 envelope 组合；8272 的 canonical frame 如何组合 | 规范/EL；编码、哈希、RPC、gas | 同一版本的组合 schema 与有效/无效向量固定 |
| D02 | signature list / proof frame / witness 与未来 8288 的接口 | 规范/PQ；types、P2P、区块承诺 | PR/规范冻结到可实现程度；未采用的变体不混入主路径 |
| D03 | FOCIL 是否保留 Hegotá；采用哪个 devnet 组合 | 版本/CL；联合 devnet 与发布范围 | 记录有效 ACD 决定；没有新决定时保留既有 SFI 基线 |
| D04 | 8369 的 claimed-index、预算、状态面与 ePBS post-reveal 职责 | 规范/CL/EL；Profile 2 | Standards Track 执行性规则、传输承诺和向量齐全 |
| D05 | 8298 code adoption 生效时机及 8151 兼容性 | VM/生态；旧授权与 fault proof | 固定语义、实际应用回放、风险与准入结论 |
| D06 | 7906 回滚执行体与 paymaster 补偿 | VM/txpool；交易断言 | 补偿/费用/nonce/日志回滚边界有规范和测试 |
| D07 | 4758 与 8360 是否成对纳入 | 版本/VM；临时合约兼容 | 正式范围与迁移路径一致；不只实现其中一半并宣称配套完整 |
| D08 | 8368/8372 的 CPSB 与归一化参数 | 规范/性能；状态增长与费用 | 上游数据与选择明确；不在客户端内自行定参数 |

这些问题的来源、公开分歧与正文不同步处已列入[宏观计划 §5.1](hegota-macro-plan-2026-11.md#51-已发现的规范组合风险)及[逐项登记](hegota-eip-review-2026-11.md)。关闭决策时保存来源、日期、采用版本、被替代版本及受影响任务。

## 3. 实施基线与架构边界

### 3.1 首批必须核实的真实路径

本轮补查基线为提交 `3b63d69`，协议代码与上一轮基线相同。以下直接影响验收设计：

| 观察 | 本地证据 | 开发计划中的处理 |
| --- | --- | --- |
| 两套运行入口 | [原生入口](../../pkg/cmd/eth2030/main.go)、[geth 包装入口](../../pkg/cmd/eth2030-geth/node.go) | 原生入口作为本版本主要验收对象；geth 包装路径单独记录支持范围 |
| 容器入口与旧 README 不一致 | [Dockerfile](../../pkg/Dockerfile) 用 `ENTRYPOINT ["eth2030"]`，`geth` 也链接到原生二进制；[devnet README](../../pkg/devnet/kurtosis/README.md)仍有 geth 包装说明 | HGT-004 查每个目标配置最终命令；HGT-090 修正文档与运行身份记录 |
| 真实 processor 与 helper 分离 | [processor.go](../../pkg/core/execution/processor.go)重建 FrameTx/VM context，再调用 helper | 必须验证字段未丢失、费用/nonce 不重复处理，实际 processor 是共识验收入口 |
| Engine 存在独立实现 | [engine backend](../../pkg/engine/backend.go)、[node backend](../../pkg/node/backend/engine_backend.go) | 查明节点实际注入与路由；同一 wire 协议在生产路径上必须行为一致 |
| 旧 Frame smoke test 不足 | [test-frametx.sh](../../pkg/devnet/kurtosis/scripts/test-frametx.sh)仍使用旧 APPROVE scope，并将部分不可用情形计 PASS | HGT-080 改为真实编码交易、执行状态与余额/nonce/收据断言 |
| FOCIL smoke test 仅看出块/交易 | [verify-focil.sh](../../pkg/devnet/kurtosis/scripts/features/verify-focil.sh)没有构造“可加入但被遗漏”的 payload | HGT-047 必须有故意遗漏、合法豁免和 fork choice 的相反用例 |
| 汇总器可能掩盖 assertoor 失败 | [run-feature-tests.sh](../../pkg/devnet/kurtosis/scripts/run-feature-tests.sh)把非零结果标作 TIMEOUT，且不作为最终 FAIL 条件 | HGT-080 统一完整判定，真实失败不能降级成超时或 PASS |
| fixture 可缺失后跳过 | [native runner tests](../../pkg/core/eftest/state_test_runner_test.go)、[eftest README](../../pkg/core/eftest/README.md)同时涉及 native/geth runner | HGT-006/081 记录实际 backend、fixture 数、skip 数；必需语料缺失阻断验收 |
| CI 与容器工具链不同 | [CI](../../.github/workflows/ci.yml)为 Go 1.26；容器 builder 为 Go 1.25，module 最低版本也需纳入 | HGT-082/091 固定并验证支持矩阵，不能只看宿主构建 |
| 兄弟仓库未在约定路径检出 | 本轮 `/Users/fy/Projects/{lighthouse,dora,spamoor}` 均不存在 | M0 准备独立检出与来源版本；当前不把本地缺失说成协议不支持 |

### 3.2 目标调用链

| 链路 | 起点 → 中间环节 → 终点 | 必须观察的结果 |
| --- | --- | --- |
| 用户交易 | RPC → typed decode → admission simulation → txpool → P2P → builder | 精确原始字节、hash、sender/payer、费用和依赖保持一致 |
| 区块执行 | Engine newPayload → 节点 backend → block/processor → VM/state → receipts/BAL → DB | 不能因某个 helper 单测成功而绕过 processor；导入结果可持久化重放 |
| 构块 | Engine payload attributes → builder → 状态执行 → getPayload → CL | 与区块导入共用规则，避免“本机可构、独立节点拒绝” |
| FOCIL | EL 选 IL → CL 签名/gossip/store → builder 更新 → satisfaction → CL fork choice | execution validity 与 IL satisfaction 分开；按选定 ePBS 时序执行 |
| 运行恢复 | DB 打开/同步 → canonical head → reorg → pool revalidation → 查询 | 状态/nonce/roots/依赖缓存恢复到同一 canonical 分支 |

协议字段转换点都需 round-trip 和丢字段测试；特别检查 `types.Transaction → message → FrameTx → VM context → receipt/RPC`。公共网络 activation 不能由本地实验标志隐式打开。

**前置规则也属于基线验收。** HGT-006 必须核对所选组合依赖的 ePBS/BAL（7732/7928）、执行请求与 deposit（7685/6110）、Glamsterdam 计费（8037/8038/2780/7778/7976）、交易/区块限制（7825/7934）、slotNumber（7843）及 7702 等现有能力。具体依赖以所选 EIP `requires` 和规范组合为准，不能因为目录存在就认定已符合。发现缺口时建立同级修复任务与回归向量，先补齐被主线消费的前置；新增工作计入 M0 重估，不隐藏在后续集成测试里。

### 3.3 仓库责任

| 责任角色 | 代码与仓库 | 交付边界 |
| --- | --- | --- |
| EL 核心 | `pkg/core/{types,config,eips,vm,execution,state,chain,rawdb}`、`pkg/bal` | 编码、执行、费用、状态根、收据、fork/DB |
| EL 接入 | `pkg/txpool`、`pkg/rpc`、`pkg/p2p`、`pkg/engine`、`pkg/node`、`pkg/cmd` | 公开入口、传播、构块、Engine API、真实节点 |
| CL / 联调 | 独立 `../lighthouse` 仓库 | Heze state/gossip/fork choice、CL 候选及运行版本 |
| 测试 / 性能 | `pkg/core/eftest`、`pkg/devnet/kurtosis`、CI；独立 `../spamoor` | 向量、差分、负载、联合 devnet 与资源证据 |
| 观察 / 运维 | `pkg/metrics`、`pkg/log`；独立 `../dora` | 展示实际 Frames/IL/PQ 状态，不能以 UI 替代协议验证 |
| 规范 / 版本维护 | `docs/plans`、配置与版本记录 | scope、ADR、符合性矩阵、准入与发布门 |

角色是职责分工，不假定已有具体人员。兄弟仓库的修改各自读取指令、构建、测试并独立提交；本计划不授权向任何团队发消息或直接发布。

## 4. 工作分解：77 项可执行任务

### 4.1 任务字段与状态

下表“依赖”列只列直接前置任务；还同时受所属范围的决策门约束。范围未定的任务先做分析/测试准备，协议实现按选定 devnet/发布范围启用。已有代码需先审计再修订，任务不默认要求全部重写。

每个任务进入开发前补齐：owner、采用规范条款/SHA、修改文件、正向与反向用例、估工、PR/commit、产物路径。统一状态为 `planned → ready → in_progress → review → verified`；另外使用 `blocked_by_spec`、`deferred`、`out_of_scope` 并写明解除条件。`verified` 必须指向可复现证据。

HGT-001/002/003/004 保留宏观计划中的编号与职责；其他任务在此细化。最初 77 项均为 `planned`。2026-11-25 已完成 A 组 7 项（HGT-001/004/005/006/007/008/009）；[任务状态登记](../../specs/hegota/tasks.json)与[验证证据](../../evidence/hegota-a/verification.json)逐项对应。其余任务仍未因本轮基线工作被认定实现，条件项尚未承诺为最终发布范围。

### A. 规范、基线与运行路径

| ID | 工作与主责 | 依赖 | 交付物与验收 |
| --- | --- | --- | --- |
| HGT-001 | 固定规范组合；规范/版本 | 无 | EIP/EELS/EEST/CL/Engine/devnet 的 SHA、组合矩阵、差异与决策登记；D01/D02 等未决部分显式隔离 |
| HGT-004 | 原生/包装/Engine 真实调用链；EL 接入 | 无 | 二进制→服务→backend→processor/VM 路由表；每个测试结果能指认运行路径 |
| HGT-005 | fork 配置、命名与历史兼容；EL 核心 | HGT-001、HGT-004 | 配置→rules→VM/processor/API 激活矩阵；fork 前拒绝新类型、fork 后按范围接受，旧配置迁移有测试 |
| HGT-006 | 原生执行、Glamsterdam 前置与历史 fork 基线；测试/核心 | HGT-004 | 前置协议符合性与缺口修复清单、当前构建/fixture/skipped 清单；native 与 geth 参考结果分开，提供最小差分 runner |
| HGT-007 | 组合 schema 与语义 ADR；规范/VM | HGT-001 | 编码/哈希、approval、gas、rollback、nonce/roots 的组合 ADR 与 goldens；不从互相冲突的草案拼凑规则 |
| HGT-008 | 生产 stub 与实验路径清理；EL 核心/接入 | HGT-004、HGT-005 | 零签名、可选 verifier、PQ stub、unsupported backend 的调用审计；真实路径无测试放行，预期失败有负向测试 |
| HGT-009 | 范围登记与候选准入；规范/版本 | HGT-001 | 72 项逐一映射到实施/条件/观察/排除；每次上游变化保存差异、影响任务、准入或延期理由 |

### B. Frames、状态与区块执行

代码重点：`pkg/core/types`、`core/config`、`core/eips`、`core/vm`、`core/execution`、`core/state`、`core/chain`、`core/rawdb`、`bal`。规范主依据：[8141](https://eips.ethereum.org/EIPS/eip-8141)、[8250](https://eips.ethereum.org/EIPS/eip-8250)、[8272](https://eips.ethereum.org/EIPS/eip-8272)及其明确前置规则。

| ID | 工作与主责 | 依赖 | 交付物与验收 |
| --- | --- | --- | --- |
| HGT-002 | FrameTx 编码/哈希/常量；types | HGT-007、HGT-005 | canonical encode/decode、tx hash 与 signing hash、边界 goldens；未知模式/flags、非规范 RLP、链 ID/整数溢出拒绝 |
| HGT-003 | APPROVE 授权与付款；VM/state | HGT-002、HGT-007 | scope/flags、ADDRESS/target、sender/payer、nonce 消费及最大费用结算；重复/越权授权、嵌套调用、revert 有明确结果 |
| HGT-010 | Frame context 与 introspection；VM | HGT-002 | 完整 context 与参数指令；未来结构/过去结果可见性、非法索引、非 Frame 上下文与 calldata 边界符合向量 |
| HGT-011 | 双维 gas、fee/floor/refund；VM/费用 | HGT-002、HGT-007 | execution/state/blob 的预算与最大付费不变量；不足、溢出、跨 frame 隔离、frame entry 与 gas pool 回补验证 |
| HGT-012 | journal、原子批次与 transient storage；state/VM | HGT-003、HGT-010、HGT-011 | frame/批次/整交易的 snapshot 语义；approval、付款、nonce、logs、warm sets、storage 各自按规范回滚 |
| HGT-013 | 原生 processor 与构块/导入统一；execution | HGT-012、HGT-019、HGT-006 | typed transaction→message→FrameTx/context 无字段丢失；消费的前置规则通过，相同 pre-state 上构块与导入 post-state 一致，无重复扣费/nonce |
| HGT-014 | Frame receipt、日志与索引；types/execution | HGT-013 | payer/frame 结果、gas 与 logs 的规范编码；失败分类、receipt root、索引及查询往返一致 |
| HGT-015 | BAL 与区块级验证；execution/BAL | HGT-013、HGT-014 | 执行与 published BAL 一致；重放验证 root、gas、字节上界及 system updates；错误 BAL/receipt 不得过验证 |
| HGT-016 | 交易/收据持久化与重放；DB/chain | HGT-014、HGT-015 | raw bytes 与 schema 版本保存；重启/历史读/区块重放相同；旧数据库可读，必要迁移有回退说明 |
| HGT-017 | keyed nonce 状态与消费；state/execution | HGT-003、HGT-007 | NONCE_MANAGER 的部署前置、读写/BAL、key set 规范化、single-use 与 rollback；空集/零 key、交集冲突与 CREATE 语义可复现 |
| HGT-018 | recent roots 与 canonical verifier；state/VM | HGT-010、HGT-007 | 系统合约部署/code hash、source/slot/root、expiry、slotNumber；空 slot、覆盖、窗口及 reorg 向量一致 |
| HGT-019 | 当前 signature schemes 与默认账号验证；types/VM | HGT-002、HGT-010 | 所选 8141 的 signature validation/default code、域分离、长度承诺及可塑性检查；任意签名入口不被误认为协议已验证 |

8250/8272 若尚未进入选定网络范围，HGT-017/018 的状态与向量可在受控组合上完成；不得把它们的实验状态写入公共网络数据库。

### C. 公开交易池、传播、RPC 与节点接入

代码重点：`pkg/txpool`、`pkg/p2p`、`pkg/rpc`、`pkg/node/backend`、`pkg/engine`。Frames 的 mempool 策略以所选 8141 版本为依据，不沿用旧的 conservative/aggressive 名称推定规则。

| ID | 工作与主责 | 依赖 | 交付物与验收 |
| --- | --- | --- | --- |
| HGT-020 | validation prefix 模拟与 trace 规则；txpool | HGT-013、HGT-019 | 各类认可 prefix、停止点、访问矩阵与预算；block-valid 但不允许公开传播的交易可清晰区分 |
| HGT-021 | 代码/状态依赖索引与重验证；txpool | HGT-020 | sender/payer/code/delegation/nonce/root 依赖；canonical change、code adoption、reorg 仅触发必要重验，无缓存误用 |
| HGT-022 | keyed nonce/roots 的 pool 处理；txpool | HGT-017、HGT-018、HGT-021 | 同 sender 不同 key 的替换/队列策略按规范；依赖过期、重叠 key、回滚驱逐，不擅自放宽公开池并发上限 |
| HGT-023 | paymaster reservation 与批量失效防护；txpool | HGT-020、HGT-021 | 余额预留、计数、释放、非 canonical payer 限额；攻击不能导致无限接纳或无界重验 |
| HGT-024 | 费用排序、替换、限额与驱逐；txpool | HGT-011、HGT-022、HGT-023 | 真实可支付成本、underpriced/replacement、多维预算与内存上限；用户替换行为可预测 |
| HGT-025 | typed 交易 P2P 传播与 peer 兼容；P2P | HGT-002、HGT-020、HGT-024 | 原始交易字节传播/重取、能力协商、无效公告/大包拒绝；不能仅在本机 RPC 接受 |
| HGT-026 | builder 的 Frame 选择与再验证；node/engine | HGT-015、HGT-024 | 同一状态快照/slot 的选择、执行与 root；失效交易不污染 payload，构建取消/重建不遗留 reservation |
| HGT-027 | 提交、查询与 receipt RPC；RPC | HGT-014、HGT-016、HGT-020 | schema/hex/错误分类、hash 与 payer/frame 映射；实际节点公开入口提交后可查询并核对 bytes |
| HGT-028 | call/simulate/estimate/trace；RPC/VM | HGT-013、HGT-027 | 符合选定 API 的模拟上下文、付款与双维成本；估 gas 失败解释、状态不落盘、trace 不改变执行结果 |
| HGT-029 | 原生节点 lifecycle 与端到端；node/测试 | HGT-025、HGT-026、HGT-027、HGT-028、HGT-016 | 外部客户端提交→跨 peer 传播→构块→独立导入→收据→重启/重组回放闭环，实际运行入口固定 |

### D. PQ 最小接口、账号迁移与交易断言

这些任务受 D02/D05/D06 及范围准入约束。规范依据：[8504 草案 PR](https://github.com/ethereum/EIPs/pull/12672)、[8288](https://eips.ethereum.org/EIPS/eip-8288)、[8298](https://eips.ethereum.org/EIPS/eip-8298)、[8151](https://eips.ethereum.org/EIPS/eip-8151)、[7906](https://eips.ethereum.org/EIPS/eip-7906)。只交付所选方案，完整递归聚合不自动进入当前版本。

| ID | 工作与主责 | 依赖 | 交付物与验收 |
| --- | --- | --- | --- |
| HGT-030 | PQ 接口与聚合边界 ADR；规范/PQ | HGT-007、HGT-009 | scheme/dependency/witness/哈希的固定版本；8504 与完整 8288 边界、未定参数和准入条件明确 |
| HGT-031 | proof frame 与 witness 编码；types/PQ | HGT-030、HGT-002 | declaration、零 hash 约定、排序去重、sidecar/witness 的规范字节向量；错误绑定和多余依赖拒绝 |
| HGT-032 | verifier、gas 与参数；crypto/VM | HGT-031、HGT-011 | 与独立已核查实现的签名向量一致；code/binding 来源、参数、CPU 路径和成本上界固定，stub 不作为验收 |
| HGT-033 | witness 的交易池/网络可用性；txpool/P2P | HGT-031、HGT-032、HGT-025 | witness 获取、校验、缓存、去重、失配与逐出；仅有 tx body 无 witness 不被误报完整验证 |
| HGT-034 | block dependency/commitment 与导入；engine/execution | HGT-031、HGT-032、HGT-015、HGT-033 | declaration union 与 witness/承诺精确绑定；缺失/多余/错误 scheme、hash/VK 都有负向区块 |
| HGT-035 | SETCODEFROM 与迁移执行；VM/state | HGT-012、HGT-021、HGT-009 | code adoption 时机、codehash、reentrancy、创建/销毁与 BAL；迁移前后旧私钥权限按规范变化 |
| HGT-036 | ecRecover code restriction；VM/txpool | HGT-035、HGT-021 | empty/delegation/真实 code 分类、返回数据、gas、BAL、signer 代码依赖；Permit/multisig/OP fault-proof 有兼容准入报告 |
| HGT-037 | POST_TX 断言与 paymaster 回滚；VM/state | HGT-012、HGT-015、HGT-023、HGT-009 | 可观察 diff/event 边界与失败回滚；nonce/付款保留和 token 补偿按最终规则，false-return token/重试/批次测试通过 |

HGT-036 若发现不可接受的生态兼容风险，阻断该候选的发布准入；不能关闭已有安全检查来强行通过。HGT-034 的直接验证也不等于实现完整 8288 聚合。

### E. FOCIL 与 AA-VOPS

规范依据：[7805](https://eips.ethereum.org/EIPS/eip-7805)、[8369](https://eips.ethereum.org/EIPS/eip-8369)、固定版本 Heze、Engine API 与 ePBS 前置。Profile 1 和 Profile 2 分开验收，不能用 sender balance 或“VERIFY target 有代码”替代真实资格验证。

| ID | 工作与主责 | 依赖 | 交付物与验收 |
| --- | --- | --- | --- |
| HGT-040 | FOCIL 范围、时序与状态 ADR；规范/CL/EL | HGT-001、HGT-004、HGT-009 | D03、IL 身份/分支/时限、execution validity vs satisfaction、Engine 调用职责固定；无新决定时维持当前 SFI 计划 |
| HGT-041 | IL 类型、存储与裁剪；EL/CL | HGT-040、HGT-005 | slot/dependent root 身份、精确 bytes、重复/equivocation/晚到/重组/保留窗口；两个节点/backend 行为可解释 |
| HGT-042 | Engine wire 协议与能力协商；engine | HGT-040、HGT-005 | get IL、payload attributes/update/import 与返回值按所选 API；schema contract tests，未知版本正确拒绝 |
| HGT-043 | builder IL 更新与构块；node/engine | HGT-041、HGT-042、HGT-026 | append/rebuild 采用明确语义；并发更新、取消、交易级联有效性和截止时间可重复，不引发死锁 |
| HGT-044 | 基础 IL satisfaction；EL execution | HGT-041、HGT-015 | 准确的存在性、有效性与 gas/bytes 约束；7934/RLP 前缀边界、无效签名与费用不足等负向场景闭合 |
| HGT-045 | 真实 node backend 接线；node | HGT-041、HGT-042、HGT-043、HGT-044 | 非空 IL 真正存储、参与构块/导入并回传结果；不把 unsupported 或旁路 backend 当已接通 |
| HGT-046 | AA-VOPS/Profile 2 执行性扩展；规范/EL/CL | HGT-017、HGT-018、HGT-020、HGT-044、HGT-007 | D04 关闭后实现 claims 编码/承诺、状态/BAL 重建、完整 prefix、预算与 ePBS 检查职责；否则明确未激活 |
| HGT-047 | gossip→fork choice 联合测试；CL/测试 | HGT-045、HGT-008、HGT-080、HGT-070 | 故意遗漏的可加入交易触发约束；满块/无效/迟到/分支例外按规范放行，CL 可观察决策与状态一致 |
| HGT-048 | PQ IL 容量与资源评估；性能/CL | HGT-047、HGT-034 | 若纳入 PQ，测 raw signatures/witness 的 IL 可容纳性、传播尾延迟和验证预算；不擅自增大 committee/IL 上限 |

HGT-046 启动前要求 D04 真正闭合；仅有 Informational 参考模型不够。FOCIL IL 本身的 PQ 签名替换与完整证明传播若另有新 EIP，必须新增范围与估工，不能隐藏在 HGT-048 性能任务里。

### F. 其他 EL 候选：逐项准入与实施

所有任务依赖 HGT-009 的准入记录，并以对应 EIP 的固定版本为依据；尚未纳入目标范围的条目可完成评估后标 `deferred`，不要求全部进入发布。编号/主讨论见[逐项登记](hegota-eip-review-2026-11.md)。

| ID | EIP / 工作与主责 | 依赖 | 交付物与验收 |
| --- | --- | --- | --- |
| HGT-050 | 8131/8279：content/BAL floor；gas/BAL | HGT-011、HGT-015、HGT-009 | 统一 byte 计量、运行时 BAL 上界、revert/floor/refund 与 Frames/PQ 组合；最坏负载资源有界 |
| HGT-051 | 3298：storage-clear refund/cap；gas | HGT-011、HGT-012、HGT-009 | 保留规范要求的 reversal；SSTORE 恢复、跨 frame 回滚、floor 计费和旧合约回放一致 |
| HGT-052 | 4758/8360：SENDALL/TCREATE；VM | HGT-012、HGT-035、HGT-009 | D07 关闭后实施；临时合约的创建/清理、余额、nonce、CREATE 子合约与旧模式迁移测试 |
| HGT-053 | 5920：PAY；VM | HGT-011、HGT-012、HGT-009 | 转账不触发接收方执行；费用、新账号/余额/BAL 与 Frames 上下文一致 |
| HGT-054 | 7709：BLOCKHASH storage；VM/state | HGT-005、HGT-015、HGT-009 | 历史系统写入/读窗口、cold/warm gas、fork 边界、BAL 和缺历史情形一致 |
| HGT-055 | 7668/8116：bloom/receipt 格式；types/RPC | HGT-014、HGT-016、HGT-027、HGT-009 | 分别准入、按组合测试；根、gas/logIndex、历史 receipt、getLogs/索引与生态消费者不混版 |
| HGT-056 | 7979；参考 8173：call/return；VM/编译器 | HGT-010、HGT-012、HGT-009 | return stack/目的地验证/legacy 代码与编译器消费明确；设计背景不冒充额外共识 opcode |
| HGT-057 | 8163：EXTENSION 保留；VM | HGT-005、HGT-009 | L1 继续 exceptional halt；尾字节、非法执行与历史字节码一致 |
| HGT-058 | 8253：fork nonce 修复；state/config | HGT-005、HGT-006、HGT-009 | 每链 pre-state 生成名单、精确条件、一次性转换与重组；不复制其他链名单 |
| HGT-059 | 7907：代码加载计量；VM/性能 | HGT-011、HGT-009 | 固定 code/initcode 上限前置；代码触及、嵌套/重用、EXTCODE* 及证明端资源边界 |
| HGT-060 | 8077：交易公告元数据；P2P/txpool | HGT-022、HGT-025、HGT-009 | source/nonce/fees 的最终 wire schema、Frames 多 key 表达、谎报验证及旧 peer 兼容 |
| HGT-061 | 8368/8372：参数与归一化；gas/性能 | HGT-011、HGT-050、HGT-009 | D08 的数据报告与选择；若准入才更新参数，basefee/gasUsed/reservoir/Frames 回归 |

### G. CL 主责与跨仓库配套

依赖 HGT-001/004 准备的固定、独立 CL checkout 与本地指令。CL 内部实现由对应仓库任务交付，ETH2030 负责接口、数据与联合验收；当前缺少 checkout 不妨碍先完成 EL 独立工作。

| ID | EIP / 工作与主责 | 依赖 | 交付物与验收 |
| --- | --- | --- | --- |
| HGT-070 | 8015：legacy deposit/eth1data 清理；CL | HGT-005、HGT-009 | 按较晚 SFI 基线交付；6110 完成条件、Heze state/fork 转换与 EL 请求处理正常 |
| HGT-071 | 8365：阻止新 0x00 validator；CL | HGT-070、HGT-009 | 新/既有 validator、fork 前待处理 deposit、资金处置与 BLS-to-execution 规则符合选定版本 |
| HGT-072 | 8334：bundled attestations；CL 网络 | HGT-009、HGT-080 | mixed-peer、invalid signature、去重/流控与资源报告；模拟收益与本地实测分开 |
| HGT-073 | 8383：CL retention；CL/同步 | HGT-009、HGT-006 | weak-subjectivity/checkpoint-sync 与保留窗口一致；EL 数据服务契约不随意缩减 |
| HGT-074 | 8411：payload chunk broadcast；CL/EL 接口 | HGT-015、HGT-042、HGT-009 | chunk root/proof、乱序/失片/重复、完整性、backpressure 与导入限额；不能导入未组装完成的 payload |
| HGT-075 | 8148：sweep threshold；CL/请求 | HGT-070、HGT-009 | EL 请求/合约→CL threshold→withdrawals；consolidation、MaxEB 与权限/金额边界可复现 |
| HGT-076 | 8483：PQ material registration；CL/请求/PQ | HGT-070、HGT-030、HGT-009 | validator/builder 的请求类型、owner、数据长度、覆盖更新、顺序、合约 code hash 和 CL 存储一致 |

独立 `spamoor` 工作由 HGT-080/084/085 驱动，负责有效与恶意 Frames、nonce/roots、迁移/PQ/IL 负载。独立 `dora` 工作由 HGT-087/090 驱动，负责显示真实字段与验证状态；都必须固定各自提交并独立验证。

### H. 验证体系、性能与恢复

| ID | 工作与主责 | 依赖 | 交付物与验收 |
| --- | --- | --- | --- |
| HGT-080 | 修正测试判定与交易生成；测试/工具 | HGT-004、HGT-006 | 显式 PASS/FAIL/TIMEOUT/SKIP/UNSUPPORTED；Frame 真交易与 FOCIL 负向场景可生成，汇总器不吞 assertoor 失败 |
| HGT-081 | EELS/EEST 差分与符合性；测试/核心 | HGT-006、HGT-007 | 按 native backend 跑固定 corpus；每条规则有向量/错误分类，缺语料或无执行不算通过，goldens 不由被测实现单独生成 |
| HGT-082 | CI 与必需语料门；测试/版本 | HGT-080、HGT-081 | 分层 CI、固定 toolchain/语料哈希/数量、测试清单与结构化结果；必需 fixture 零 skip，条件项按 release scope 计算 |
| HGT-083 | fuzz/race/属性检查；测试/核心 | HGT-012、HGT-021、HGT-042 | 编码与费用不变量、snapshot/revert、并发 payload/IL/重验压力；固定 seed、可最小化复现，无未解释 race |
| HGT-084 | 跨客户端互通矩阵；测试/CL/EL | HGT-029、HGT-047、HGT-082 | 至少一个独立 EL + 外部 CL；按 scope 增补候选，包含发送/构块/导入角色互换及负向区块，根与判定一致 |
| HGT-085 | 对抗负载与资源预算；性能 | HGT-024、HGT-026、HGT-082 | 底线硬件、数据集、p50/p95/p99、CPU/RSS/IO/带宽；共享 payer/代码失效/大签名/超大 BAL 的成本有上界 |
| HGT-086 | fork/sync/长 reorg/恢复；测试/chain | HGT-016、HGT-029、HGT-082 | fork 前后、重复导入、crash/restart、磁盘问题、空 slot、深 reorg、sync/re-execution 均与 canonical state 一致 |
| HGT-087 | 长稳与可观测性；测试/运维 | HGT-084、HGT-085、HGT-086 | 每个范围项有交易与覆盖计数，按阶段长稳；记录告警、停机/恢复与资源漂移，Dora/metrics 可关联到实际请求 |

HGT-081/082 的框架可在功能实现并行准备，但某个 EIP 的“符合性通过”必须等待该 EIP 的实际执行链路。HGT-084 的 FOCIL 前置仅在正式范围裁剪后才可由有证据的替代验收门取代。

### I. 发布、迁移与后续维护

| ID | 工作与主责 | 依赖 | 交付物与验收 |
| --- | --- | --- | --- |
| HGT-090 | 文档、配置与升级/恢复手册；版本/运维 | HGT-009、HGT-029、HGT-086 | 网络 scope、activation、DB/CLI 兼容、系统合约、支持/不支持项、故障脚本与旧文档修订；复现不依赖隐含本机设置 |
| HGT-091 | 可复现二进制/镜像与 RC；版本 | HGT-082、HGT-090、HGT-008 | commit/version/toolchain/dependency lock、binary/image digest、license/依赖清单；容器真实入口通过目标测试 |
| HGT-092 | RC 升级演练与外部网络验证；测试/运维 | HGT-087、HGT-091 | 目标版本 RC 的从旧库升级、fork 转换、回退演练与冻结报告；上游测试网可用时验证，未可用则保留该外部门 |
| HGT-093 | 发布包与维护交接；版本/运维 | HGT-092 | release scope/已知限制/证据索引、升级命令、问题分级与值守/runbook；外部发布另按明确授权执行 |

## 5. 依赖、里程碑与交付顺序

### 5.1 主依赖链

主线按 `HGT-001/004 → HGT-007/005/006 → HGT-002 → HGT-003/010/011/019 → HGT-012/013 → HGT-014/015/016 → HGT-020…029 → 联合验证 → RC` 推进。HGT-080/081 的验证框架从基线阶段就开始，不能等全部实现后再建设。

FOCIL 从 HGT-040 开始，与 Frames 的稳定底座并行；后接 HGT-041…045、070、047。Profile 2 另依赖 HGT-017/018 和 D04；PQ witness 另依赖 HGT-030…034。任何候选加入都要回到组合测试，不能绕过主链直接进入 RC。

### 5.2 阶段验收门

| 里程碑 | 主要任务 / 可展示产物 | 通过条件 | 不通过时仍可推进 |
| --- | --- | --- | --- |
| M0 基线就绪 | 001、004…009、080/081 框架；规范/运行路径/fixture 清单 | 支持的组合可复现，必需路径与缺口明确，首批任务有测试 oracle | 独立调用链、旧测试判定和语料整理 |
| M1 核心可执行 | 002、003、010…016、019 | native processor 的编码/授权/计费/回滚/收据向量与 EELS 一致 | RPC schema、pool/CL harness 与候选评估 |
| M2 公开入口闭环 | 017/018 按 scope、020…029 | 外部用户可提交真实 Frame tx，经独立 peer/导入端查询，重启和 reorg 通过 | FOCIL 独立基础与 PQ 接口研究 |
| M3 跨层一致 | 040…047、070；046/048 按范围 | 当前 scope 的 IL 约束实际进入 fork choice，执行与 IL 状态分类正确 | 非 FOCIL 的 Frames/恢复/性能工作；正式延期需改范围再替换此门 |
| M4 候选范围冻结 | 准入的 030…037、050…061、071…076 | 每项完成实现/兼容/向量/角色矩阵，未准入项有明确处置 | 核心主线长稳与发布材料 |
| M5 发布候选 | 082…092 与全部 scope 实现 | 下文发布检查全部通过，RC binary/image 与证据一致 | 仅阻断点修复，冻结后不再随意增加功能 |
| M6 可交付发布包 | 093 | 升级/恢复命令、已知限制与证据索引齐备 | 后续维护按 bug/上游变化另开任务 |

M1/M2 是内部开发里程碑；它们不表示已实现完整 Hegotá。目标网络保留的 SFI 与最终范围项缺一不可。

## 6. 人员假设、排期与容量

### 6.1 排程假设

为形成可执行的先后顺序，初始按 **2 名 EL 工程师 + 1 名 CL/联调工程师 + 共享测试/性能支持** 排程；不表示已经分配人员。每两周一个迭代。M0 完成后按真实差异、上游规范成熟度、可用人手和参考客户端支持重新估工。

下表是项目内部的初始区间估算，不是 EF 时间表，也不承诺主网日期。规范持续变化、参考客户端尚未支持、重大 DB/计费重构会延长它；若仅一名开发者，不沿用三人并行日历，改成按依赖串行并在 M0 重排。

| 迭代窗口（相对启动） | 主要结果 | 可并行工作 |
| --- | --- | --- |
| I1–I2 | M0：固定组合、真实入口、原生基线、严格测试判定；明确 D01/D02 初始采用版 | CL checkout/版本、fixture harness、候选准入分析 |
| I2–I5 | M1：Frame codec、signature/APPROVE、双维 gas/journal、processor、receipt/BAL/DB | fork 配置、FOCIL wire/store、RPC schema、nonce/roots 向量 |
| I4–I8 | M2：公开池依赖/预算、P2P、builder、RPC、nonce/roots 组合与 E2E | FOCIL 实际 node 接线、CL 8015、PQ/迁移兼容分析 |
| I6–I10 | M3：FOCIL 正反向联合用例与 fork choice；Profile 2 按成熟度推进 | 准入的少量 EL/CL 候选与资源负载 |
| I8–I12 | M4：最终选择的 PQ/迁移/候选完成组合验证，冻结范围 | 独立客户端矩阵、fuzz、长 reorg、升级与性能 |
| I10–I14 | M5/M6：长稳、RC、升级/恢复演练和发布包 | 文档修订、证据审阅与后续维护准备 |

**初始容量区间：约 9–14 个迭代（18–28 周）**，适用于上述人员假设、核心范围和少量已成熟候选。所有候选全部纳入、新 PQ 共识机制、完整 8288 聚合不在该区间内。I1–I2 必须给出按任务重估的工时及关键链，收敛后替换本区间；不可将此估算用于对外发布日期。

### 6.2 容量取舍

- EL 核心与 EL 接入分工协作，以固定 schema/向量衔接；同时改 `FrameTx`、processor、VM 的 PR 必须协调合并顺序。
- FOCIL/CL 工程师优先处理 Heze/ePBS 版本、Engine contract tests 和 8015；不要把 EL reviewer 同时分配给所有候选。
- 每个候选只有在说明“新增集成面 + 测试面 + 是否延长关键路径”后才进入迭代。技术互相关联的 8131/8279、4758/8360 按组合审查，但分别记录纳入状态。
- 上游未决导致某任务阻塞时，继续无依赖的测试、旧路径审计、恢复和性能工作；不为赶进度自行定义不兼容的规则。

## 7. 测试与证据计划

### 7.1 分层测试矩阵

| 层级 | 必测场景 | Oracle / 输出 | 阻断条件 |
| --- | --- | --- | --- |
| 编码与哈希 | frame flags/limits/value、签名/依赖、canonical integer、长度/溢出、nonce set | 固定规范向量与独立工具，byte-by-byte 比较 | 任一字节/哈希不符或实现自生成 oracle 未独立核查 |
| VM 与状态 | scopes、target/address、default/verify/sender、batch、transient、异常与回滚 | pre/post state、logs、nonce、payer、费用守恒；EELS 对照 | 负余额、重复消费、越权授权、预算串用或回滚残留 |
| gas/BAL/区块 | 双维与 blob、floor/refund、system updates、最大 block bytes、BAL diff | roots、gasUsed、BAL 字节、有效/无效分类 | 执行与 published diff 不一致、gas/byte 上界误判 |
| mempool | prefix trace、canonical paymaster、reservations、code/root/nonce 依赖、替换与失效 | admission reason、trace、重验计数与资源数据 | 公开池误接纳/误传播、预算无界或 reorg 后缓存陈旧 |
| RPC/P2P/节点 | 实际 submit/propagate/build/import/query、call/estimate/trace、重启 | 原始交易和响应、构块/导入 root、已验证运行身份 | 只在 helper/geth 参考可用、字段丢失、无交易却 PASS |
| FOCIL | 必须包含却遗漏、合法 gas/bytes/invalid 豁免、晚到/equivocation、分支差异 | EL execution/IL 状态与 CL fork-choice 决策日志 | 有效链被错误拒绝、可加入遗漏不影响应有决策 |
| PQ/迁移/断言 | witness 缺失/错 key/错 hash、code adoption、old-key authority、permit、paymaster rollback | 独立 crypto vectors、应用回放、缺失/额外依赖负向块 | stub 放行、未绑定声明、旧授权意外保留或应用损失未评估 |
| 恢复与历史 | fork 前后、空 slot、crash/DB reopen、sync、长 reorg、旧 receipt/log 查询 | canonical state 与 pool 依赖重建、历史结果 | 拒绝合法恢复、state/receipt/BAL 漂移或 DB 不可恢复 |

历史基础功能必须跑原生 backend；geth 作为参考执行可增加对照，不能替代被测实现。任何测试都记录 fixture 来源、SHA、选取规则、总数、实际执行数、失败和跳过数。

### 7.2 跨客户端角色矩阵

| 场景 | Builder / 交易发送侧 | 导入/验证侧 | 目的 |
| --- | --- | --- | --- |
| 原生自洽 | ETH2030 | 第二个 ETH2030 + 外部 CL | 揭示构块与导入路径差异 |
| 外部导入 | 独立 EL | ETH2030 + 外部 CL | 排除“仅接受自己构造的块” |
| 外部构块验证 | ETH2030 | 独立 EL + 外部 CL | 排除本地规则自洽但不符合规范 |
| 混合交易来源 | 外部 transaction generator/peer | 两个不同 EL 的 mempool | 检验传播、替换与 admission 差异 |
| FOCIL 负向 | 有意忽略可加入 IL 的受控 builder | CL + ETH2030/独立 EL | 直接检查约束与豁免；不能只观察出块 |
| fork/restart | fork 前数据库与老节点 | 新版 ETH2030/参考节点 | 验证升级、重组和恢复 |

独立 EL/CL 的具体名称与版本在 M0 固定，只有实际支持目标组合才纳入该矩阵；某实现暂不支持记 `UNSUPPORTED`，不算失败也不算通过，但不能满足必需的独立互通验收。

### 7.3 资源与长稳

先在 M0/M1 记录底线硬件及历史 fork 的处理基线，再通过 ADR 固定 slot/build/import/replay 的预算和安全余量。至少测量 p50/p95/p99、峰值 RSS、CPU、disk read/write、wire bytes 和队列长度；gas 不能直接代替代码加载、网络和磁盘成本。

必需负载包括最大合法 Frames、失效前缀、共享 payer、nonce/key/root 冲突、代码变更导致重验、大 PQ witness、超大 BAL/接近区块字节上限、IL 重复/晚到与慢 peer。性能结果附完整硬件、参数、负载 seed 和版本，平均值不能掩盖尾延迟。

项目内部初始长稳门建议为 **24 小时预检 + 冻结后的 72 小时联合负载**，这是本项目建议，不是 EIP 要求。必须跨过实际用例所需的 fork/expiry/轮换边界；单纯运行 72 小时没有覆盖这些边界不算通过。变更触及共识、计费、并发或恢复时重跑相关长稳门。

### 7.4 运行命令与结果判定

以下命令是实施过程中现有模块的验证入口；本轮只运行构建，未来任务按修改范围执行相应测试：

```bash
cd /Users/fy/Projects/eth2030/pkg
env -u GOROOT go build ./...
env -u GOROOT go test ./core/types ./core/config ./core/eips ./core/vm ./core/execution ./core/state/... ./core/chain/... ./bal/...
env -u GOROOT go test ./txpool/... ./p2p/... ./rpc/... ./engine/... ./node/... ./cmd/...
env -u GOROOT go test -race ./txpool/... ./engine/... ./node/backend/...
env -u GOROOT go vet ./...
env -u GOROOT go test ./... -timeout=15m -count=1
```

Docker/Kurtosis 另固定 image digest、CL 版本与 package commit。先用独立测试 enclave，不复用正在运行的他人网络；清理只针对本任务创建的 enclave。现有 `full-feature` 配置要先由 HGT-004/080 校验范围和判定，不能直接作为官方 Hegotá 符合性结果。

`TIMEOUT`、`SKIP`、`UNSUPPORTED` 在必需用例上都不能关闭验收项。排除测试需写明规范条款、原因及范围决定；不得通过缺 fixture、mock 跳过真实验证或弱化断言使报告变绿。

## 8. 发布门、数据迁移与运维

### 8.1 RC 发布检查

| 检查项 | 必需证据 |
| --- | --- |
| 范围与版本 | 网络/EIP 集/activation/规范组合锁定，所有 D01–D08 对当前范围有关闭或有据排除记录 |
| 功能与符合性 | 当前范围任务 verified，逐规则向量无遗漏，root/gas/BAL/error 差分无未解释结果 |
| 跨客户端 | 独立 EL 和 CL 的角色矩阵通过，生产入口与容器 digest 可追溯 |
| 安全与资源 | 当前范围的越权/可塑性/费用/DoS/并发/恢复结果审阅完成，严重缺陷清零 |
| 运行与恢复 | 升级、fork、crash、DB、sync、reorg、历史查询与回退演练通过 |
| 语料与工具链 | 必需语料零 skip；宿主/CI/容器支持版本固定；无测试捷径进入生产 |
| 发布包 | 二进制、镜像、commit、依赖清单、校验值、已知限制、runbook 和配置示例一致 |
| 外部网络 | 官方目标测试网可用时完成接入验证；尚不可用则只能称内部 RC，不能称测试网验收完成 |

生产回退不等于关闭 fork 或重解释已导入区块。DB 格式若发生不可逆变化，必须先备份、记录 schema version，并提供读旧库或按固定数据重同步的路径；回退只能使用与目标网络共识兼容的版本。

### 8.2 指标与告警

至少覆盖 Frame admission/reject reason、验证前缀 gas/time、依赖失效与重验队列、payer reservation、builder/import 延迟、IL timely/equivocation/unsatisfied、Engine error、sync lag、reorg depth、DB 错误与资源上限。PQ witness 缓存/验证和 candidate-specific 指标按最终范围补齐。

日志应关联 tx hash、block root/hash、slot、payload ID、fork/规范组合；可观测性避免输出用户 witness 全文或其他无诊断价值的大块数据。Dora 展示来源明确的验证结果，缺数据标未知。

### 8.3 发布后处理

共识分歧、错误授权/扣款、错误 state root、无法恢复和无界资源消耗作为最高优先级处理：保存输入/版本/数据库快照，重现并做最小回归，修复后重跑受影响差分与互通门。一般 RPC/显示问题按影响另排，不能用运维配置绕过共识检查。

持续跟踪上游范围和规范变化；每次变更标出受影响任务、向量、数据库/网络兼容性与是否重启 RC 周期。对外发布或通知团队需另有明确指令，本计划本身不触发发布或消息发送。

## 9. 变更与任务管理

每个 PR 应完成一个可审阅的规则组；共享类型先合入兼容基座，再修改消费者，临时不兼容阶段只在隔离开发分支存在。分支使用 `codex/` 前缀，提交只包含本任务编辑的文件；不顺手修 unrelated 功能，不修改只读 `refs/`。

单项完成清单：

- 固定 EIP/规范条款与决定，写明和旧代码的差异。
- 更新实际生产路径及必要配置，保留该 scope 之外历史行为。
- 添加新功能的正向/负向单元测试；涉及协议语义必须有 processor/节点或规范差分测试。
- 相关构建与测试通过，新增 fixture 不被 skip，运行结果与使用的 backend 明确。
- 对字段、费用、DB、RPC、P2P、CL 的影响交代清楚；更新符合性矩阵和 `STATUS.md`。
- commit/PR 与证据可追溯；reviewer 核实关键不变量后才标 verified。

M0 的逐规则矩阵至少包含：`EIP + section + spec SHA + requirement ID + caller/file + positive vector + negative vector + oracle + target fork + test result + open question`。性能/资源项另带硬件与预算；范围排除项记录范围依据，不能伪造为已实现。

### 第一轮启动顺序

1. **先做 HGT-001 与 HGT-004。** 固定规范组合与真实运行路径；准备缺失的 CL/测试工具 checkout 版本。
2. **接着做 HGT-006、007、080、081 框架。** 锁定 oracle、纠正测试判定并关闭可独立解决的 D01/D02 部分。
3. **开始 HGT-005、002、003、010、011、019。** 按 codec→context/signature→approval/gas 的依赖合入；对应向量同步进入 CI。
4. **并行准备 HGT-040、070 和候选分析。** 只做不依赖未定语义的部分；不要先启动全部候选实现。
5. **M0 后重估任务与容量，M1 后进入公开池/RPC/联合链路。** 始终以完整目标网络范围和可复现证据判断完成。

## 10. 72 项提案的工作处置索引

本表逐项覆盖上一轮的 72 项研究对象；“排除/观察”同样是明确处置，避免既漏项又把所有 EIP 误当必交。阶段来源与细节仍保留在[逐项登记](hegota-eip-review-2026-11.md)，本表只负责映射工作。

| EIP | 当前计划分组 | 对应任务 | 处置 / 激活条件 |
| --- | --- | --- | --- |
| EIP-2488 | DFI | HGT-009 | 保留研究；无新的重新纳入决定，不排本版本实现 |
| EIP-3298 | 候选 | HGT-051 | refund/floor/Frames 回归 |
| EIP-4758 | 候选 | HGT-052 | 与 TCREATE 配套决定 |
| EIP-5920 | 候选 | HGT-053 | 无 recipient execution 的转账语义 |
| EIP-7645 | DFI | HGT-009 | 保留研究；无新的重新纳入决定，不排本版本实现 |
| EIP-7666 | DFI | HGT-009 | 保留研究；无新的重新纳入决定，不排本版本实现 |
| EIP-7668 | 候选 | HGT-055 | 单独准入，收据/RPC/日志历史回归 |
| EIP-7709 | 候选 | HGT-054 | history 系统写入/窗口/gas/BAL |
| EIP-7716 | DFI | HGT-009 | 保留研究；无新的重新纳入决定，不排本版本实现 |
| EIP-7805 | 当前 SFI；范围复议中 | HGT-040…048 | 完整列入，按正式范围变化调整 |
| EIP-7807 | DFI | HGT-009 | 保留研究；无新的重新纳入决定，不排本版本实现 |
| EIP-7819 | DFI | HGT-009 | 保留研究；无新的重新纳入决定，不排本版本实现 |
| EIP-7851 | DFI | HGT-009 | 保留研究；无新的重新纳入决定，不排本版本实现 |
| EIP-7862 | DFI | HGT-009 | 保留研究；无新的重新纳入决定，不排本版本实现 |
| EIP-7906 | 候选 | HGT-037 | 断言与 paymaster 回滚决策门 |
| EIP-7907 | PFI 候选 | HGT-059 | 代码加载计量与资源报告 |
| EIP-7923 | DFI | HGT-009 | 保留研究；无新的重新纳入决定，不排本版本实现 |
| EIP-7973 | 仅观察 | HGT-009 | 未确认本 fork 纳入，不排新实现 |
| EIP-7979 | 候选 | HGT-056 | return stack 与编译器消费 |
| EIP-8015 | 较晚 SFI | HGT-070 | CL 主责，EL 联调验收 |
| EIP-8025 | DFI | HGT-009 | 保留研究；无新的重新纳入决定，不排本版本实现 |
| EIP-8077 | 候选 | HGT-060 | 公告 wire schema 与多 key 兼容 |
| EIP-8094 | DFI | HGT-009 | 保留研究；无新的重新纳入决定，不排本版本实现 |
| EIP-8105 | 已撤回 | HGT-009 | 保留研究；无新的重新纳入决定，不排本版本实现 |
| EIP-8115 | DFI | HGT-009 | 保留研究；无新的重新纳入决定，不排本版本实现 |
| EIP-8116 | 候选 | HGT-055 | 单独准入，receipt gas/logIndex 回归 |
| EIP-8131 | 候选 | HGT-050 | 与 BAL byte floor 组合验证 |
| EIP-8141 | 主线 | HGT-002、HGT-003、HGT-010…016、HGT-019…029 | 按选定规范完整实现 |
| EIP-8142 | DFI | HGT-009 | 保留研究；无新的重新纳入决定，不排本版本实现 |
| EIP-8146 | DFI | HGT-009 | 保留研究；无新的重新纳入决定，不排本版本实现 |
| EIP-8148 | 跨层候选 | HGT-075 | 请求/threshold/withdrawals 联测 |
| EIP-8151 | 候选 | HGT-036、HGT-021 | 不可变合约/fault-proof 与重验门 |
| EIP-8163 | 候选 | HGT-057 | L1 exceptional halt 保持不变 |
| EIP-8173 | PFI；设计背景 | HGT-056、HGT-007 | 按具体消费 EIP 验收，不独立虚构功能 |
| EIP-8182 | DFI | HGT-009 | 保留研究；无新的重新纳入决定，不排本版本实现 |
| EIP-8184 | DFI | HGT-009 | 保留研究；无新的重新纳入决定，不排本版本实现 |
| EIP-8188 | DFI | HGT-009 | 保留研究；无新的重新纳入决定，不排本版本实现 |
| EIP-8198 | DFI | HGT-009 | 保留研究；无新的重新纳入决定，不排本版本实现 |
| EIP-8200 | DFI | HGT-009 | 保留研究；无新的重新纳入决定，不排本版本实现 |
| EIP-8205 | DFI | HGT-009 | 保留研究；无新的重新纳入决定，不排本版本实现 |
| EIP-8219 | DFI | HGT-009 | 保留研究；无新的重新纳入决定，不排本版本实现 |
| EIP-8237 | DFI | HGT-009 | 保留研究；无新的重新纳入决定，不排本版本实现 |
| EIP-8243 | DFI | HGT-009 | 保留研究；无新的重新纳入决定，不排本版本实现 |
| EIP-8250 | 伴随候选 | HGT-017、HGT-022 | 组合编码/nonce 与激活范围门 |
| EIP-8253 | 候选 | HGT-058 | 逐链状态名单和 fork 转换 |
| EIP-8272 | 伴随候选 | HGT-018、HGT-022 | canonical verifier、slot/root 与重组 |
| EIP-8279 | 候选 | HGT-050 | 运行时资源上界与组合计费 |
| EIP-8288 | 远期/架构约束 | HGT-030、HGT-009 | 保留兼容设计，不承诺完整聚合 |
| EIP-8298 | 候选 | HGT-035、HGT-036 | 与 8151 配套兼容准入 |
| EIP-8304 | DFI | HGT-009 | 保留研究；无新的重新纳入决定，不排本版本实现 |
| EIP-8321 | DFI | HGT-009 | 保留研究；无新的重新纳入决定，不排本版本实现 |
| EIP-8333 | DFI | HGT-009 | 保留研究；无新的重新纳入决定，不排本版本实现 |
| EIP-8334 | CL 候选；开放 PR | HGT-072 | mixed-peer 与网络资源 |
| EIP-8341 | DFI | HGT-009 | 保留研究；无新的重新纳入决定，不排本版本实现 |
| EIP-8355 | DFI | HGT-009 | 保留研究；无新的重新纳入决定，不排本版本实现 |
| EIP-8358 | DFI | HGT-009 | 保留研究；无新的重新纳入决定，不排本版本实现 |
| EIP-8359 | DFI | HGT-009 | 保留研究；无新的重新纳入决定，不排本版本实现 |
| EIP-8360 | 候选 | HGT-052 | 临时合约生命周期与兼容性 |
| EIP-8363 | DFI | HGT-009 | 保留研究；无新的重新纳入决定，不排本版本实现 |
| EIP-8365 | CL 候选 | HGT-071 | 停止新 0x00，与旧 validator 区分 |
| EIP-8367 | DFI | HGT-009 | 保留研究；无新的重新纳入决定，不排本版本实现 |
| EIP-8368 | PFI；等待数据 | HGT-061 | 与 8372 联评，参数未定不激活 |
| EIP-8369 | PFI；Informational | HGT-007、HGT-046 | 执行性扩展与 ePBS 职责闭合后实施 |
| EIP-8371 | DFI | HGT-009 | 保留研究；无新的重新纳入决定，不排本版本实现 |
| EIP-8372 | PFI；等待数据 | HGT-061 | 与 8368 联评，计费/归一化门 |
| EIP-8374 | DFI | HGT-009 | 保留研究；无新的重新纳入决定，不排本版本实现 |
| EIP-8375 | DFI | HGT-009 | 保留研究；无新的重新纳入决定，不排本版本实现 |
| EIP-8379 | DFI | HGT-009 | 保留研究；无新的重新纳入决定，不排本版本实现 |
| EIP-8383 | CL 候选 | HGT-073 | CL retention 与恢复契约 |
| EIP-8411 | 跨层候选 | HGT-074 | chunk 可用性与 payload 导入 |
| EIP-8483 | PFI；开放 PR | HGT-076 | PQ 请求与 CL 存储所有权 |
| EIP-8504 | 开放草案 | HGT-030…034 | 接口冻结后条件实现 |

## 11. 原始计划交付的验证（历史记录）

本轮不开展上述协议开发、Docker/Kurtosis 启动、跨仓库修改或外部发布。已核查原生/包装入口、processor、Engine backend、测试脚本、fixture runner 和 CI 的实际文件；本地 `env -u GOROOT go build ./...` 通过。

提交前核验全部任务编号唯一、依赖无环且前置存在、72 项 EIP 无遗漏/重复、所有本地链接存在、范围阶段未被计划优先级覆盖；更新 `CHECK.md`、`STATUS.md` 并单独提交。已有宏观研究与来源清单继续作为审阅证据。


## 12. A 组实施与验收（2026-11-25）

A 组 7 个任务的完整交付见 [A 组报告](hegota-a/README.md)，固定来源、6 个 profile、D01–D09、72 项准入、逐规则前置缺口、组合 schema/ADR、真实运行路径及安全修复均有对应产物。严格 native/upstream geth 最小差分执行 12 项，零失败/零跳过；新编码及语义 oracle 不被冒充为当前 native 的完整 Hegotá 执行通过。构建、vet、全仓库测试、相关 race 和 blst 构建标签测试通过，旧 skip 清单保留。

HGT-006-R01…R16 是本次明确登记的 M0 修复工作，消费其规则的后续任务必须先关闭它们；D01–D09 对联合发布仍有开放项，不能拼接草案或用旧原型冒充最新规范。当前没有可宣称已验证的联合 Hegotá 发布 profile。Docker/Kurtosis、兄弟 CL 仓库构建与正式 devnet 互通留在 HGT-070/080/091 的既定验收范围。
