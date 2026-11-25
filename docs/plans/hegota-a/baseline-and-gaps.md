# 历史基线、前置协议与修复清单

HGT-006。验收入口是 `pkg/cmd/eth2030-baseline`，固定输入在 `pkg/core/eftest/testdata/hegota/manifest.json`。两份原始 legacytests 官方文件的 SHA、来源和 12 个 subtest 数均固定。每个用例在 native processor 和直接 upstream geth 各执行一次，对 pinned oracle 与彼此的 state root、logs hash、gas 做比较。没有使用 ETH2030 geth wrapper，也没有复用 native 结果生成 oracle。

实际覆盖：7 个旧文件用例（Frontier、Homestead、EIP150、EIP158、Byzantium、Constantinople、ConstantinopleFix）和 5 个新文件用例（Berlin、London、Paris、Shanghai、Cancun）。准确 fork 名以 [baseline.json](../../../evidence/hegota-a/baseline.json) 为准，防止用“支持的 fork 列表”冒充执行数量。这是最小历史基线，不能推导为完整历史回归，更不包含 Prague/Glamsterdam/Hegotá。

该 runner 拒绝：缺文件/错误哈希、空 corpus/缺 post、未知执行字段（例如尚未由 native GST adapter 解析的 access list/blob/auth）、未知 fork、负索引/越界索引、零或缺失 roots/logs、任意 `expectException` 通配与用例数变化。异常分类尚未引入之前，明确排除 exception fixture 的支持，不能把任意错误或 panic 记作预期异常。运行期 panic/执行错误非零退出。Go 单测用十一类破损输入验证这些失败路径。

原有宽松 runner 为兼容旧 helper 测试保留，其 `Passed` 不构成验收。它的缺失 fixture/skip、geth wrapper 与 native backend 在测试清单中分别标出。上游 geth `cmd/evm/testdata/statetest.json` 的 root/logs 全零，现以 `rejected-upstream-zero-root.json` 保存作为不可接受 oracle 的证据，未纳入 12 项必需语料。没有下载整个官方 corpus，也没有将 fixture generator 的源码数量统计为测试通过数量。

| 前置 | 已查实际调用位置 | 结论与同级修复任务 | 关闭向量 / 后续依赖 |
| --- | --- | --- | --- |
| 7732 / ePBS | `epbs/validation.go`、`engine/builder_api.go`、`engine/builder/registry.go` | `HGT-006-R01`：旧 BidHash 非 SSZ signing root，缺可信 CL domain/registry；生产提交已拒绝；完整符合性未验证 | 真实 canonical bid/domain/registry + tamper/zero/reuse；HGT-040/070/080 |
| 7928 / BAL | `core/chain.Blockchain.InsertBlock`、`core/execution`、`bal` | `HGT-006-R02`：已有记录与根检查；需固定 published BAL 序列化、实际重放和 system update 的逐规则符合性 | 正确/错误/missing BAL、重放 root；HGT-013/015 |
| 7685 requests | `core/execution.ProcessRequests`、chain requests hash | `HGT-006-R03`：当前只收 0x00–0x02，新增 builder/PQ 请求未接通；不能假定含最新 builder deposit/exit | request sort/empty/hash/limits/system calls；HGT-013/075/076 |
| 6110 deposits | `processDepositRequests` 从 storage 读取 | `HGT-006-R04`：当前存储读取不同于规范 deposit log 收集，需要修复并 oracle 对照 | valid/invalid deposit log、顺序与请求根；HGT-013/081 |
| 8037 state gas | processor reservoir 与 VM state repricing | `HGT-006-R05`：简化旧参数/模型不代表所锁规范；缺 Frames 显式二维账本 | 创建/回填/跨 frame ownership/回滚；HGT-011/012/013 |
| 8038 access pricing | `core/vm/gas_table.go`、`dynamic_gas.go`、jump table / pool | `HGT-006-R06`：需与固定版逐项比较，不能沿用旧注释常量 | warm/cold CALL/SLOAD/SSTORE/auth/access-list；HGT-006/011 |
| 2780 intrinsic/value | processor `intrinsicGasGlamst`、pool | `HGT-006-R07`：已有旧实现；当前费用与 account existence/creation 规则待核对 | EOA/合约创建/value/nonexistent、pool vs import；HGT-011/020 |
| 7778 block refund | `BlockGasUsed` / gas pool / chain receipt 检查 | `HGT-006-R08`：已有 pre-refund 分支，未证双维 fork 组合 | payer fee vs block capacity、refund 上限/floor；HGT-011/015 |
| 7976 calldata floor | processor/pool floor | `HGT-006-R09`：现有旧标量模型需对照 frame/signature elision pricing | 零/非零字节、签名长度变化、维度 reservation；HGT-011/020 |
| 7825 tx cap | block validator / processor / admission | `HGT-006-R10`：需验证 16,777,216 execution cap 与 state gas 不混用 | cap−1/cap/cap+1；HGT-002/011/015 |
| 7934 block bytes | `core/block`、FOCIL helper | `HGT-006-R11`：要覆盖整体 RLP 前缀增长，不只 IL raw bytes | limit 边界与 omitted-tx 可加入性；HGT-015/044 |
| 7843 slotNumber | types/header、Engine payload / VM SLOTNUM | `HGT-006-R12`：需从真实 Engine 保持至 header；不得从 timestamp 推槽 | skipped slot、重组、构块/导入；HGT-018/042/070 |
| 7702 delegation | `applyMessage` / authorization list / VM | `HGT-006-R13`：有执行路径，未锁当前组合与 8151/8298 生效时机 | auth nonce/chain/delegation/fault proof；HGT-019/035/036 |
| 7708 ETH logs / 7976 data costs / 3607 / 3529 | processor、VM、receipts | `HGT-006-R14`：现有功能不等于与 latest Frames 的组合符合性 | transfer log、approval/new account、rollback、logs root；HGT-011/012/014 |
| native Frame 字段转换 | `TransactionToMessage` → `applyMessage` 重建旧 FrameTx | `HGT-006-R15`：高位 keyed nonce、chain ID/新字段不能被当前旧 message 完整保持 | typed→message→VM→receipt 字段完整性；HGT-002/010/013/017 |

新增 `HGT-006-R16`：实际 native FCU 没有接通 canonical head/state 切换，依据固定 EIP-3675；详见运行路径报告。新增 pool/reorg 测试验证的是已应用的链回退及真实 listener，没有把它计作 FCU head 切换通过。

这些修复项是 M0 审计的明确输出，状态均为 `open`，不以 A 组完成而标实现完成。它们有 owner、规范 SHA、正反向向量/任务依赖和 release blocker，见 [conformance.json](../../../specs/hegota/conformance.json)。主线若消费相应规则，必须先关闭对应 R 项，再关闭 HGT-013/015；修复成本计入 M0 重估，原来的 18–28 周范围不能直接维持。

A 组新增的激活门、别名迁移、标准 BLS、缺验证器/unsupported 路径和差分工具各自有可执行回归。BLS 的 8 个旧“需要 real blst”跳过用例恢复实际执行；其余全仓库 skip 的逐项名称与原因保存在证据清单。发布门仍要求目标网络全部必需向量可用且零 skip，当前小语料不能替代后续 EEST/EELS 生成语料和独立 CL 联调。
