# 真实运行路径与激活矩阵

HGT-004 / HGT-005。代码证据对应本次提交，源文件校验见运行证据；表内的“支持”指已检查的路径边界，不代表整个 fork 符合性。

| 入口 | 实际路径 | 验收归属 |
| --- | --- | --- |
| `cmd/eth2030` | `node.New` → `core/chain.Blockchain`、`core/block.BlockBuilder` → `core/execution.StateProcessor` → native VM/state | 主验收二进制；新差分 runner 也明确调用 native `execution.ApplyTransaction`，但不是节点 wire 测试 |
| native Engine | `node.New` 注入 `node/backend.EngineBackend` → `engine.EngineAPI`/`HandleRequest` → backend process channel → `Blockchain.InsertBlock`/processor | `TestNativeRuntimePathAndUnsupportedBogotaAPI` 检查实际注入对象和 wire 拒绝；不把另一个 backend 单测混入 |
| helper Engine | `engine.NewEngineBackend` → 自有 state copy、processor、异步 builder | 独立 helper；FOCIL post-state/prefix 未接线时返回明确 error，不能当 native 通过 |
| RPC | `rpc.Server` / `ExtServer` → `node/backend.RPCBackend` → 实际 pool/chain；`eth_call` 使用 native VM | 与 processor 共用显式 Frame 激活条件；pool 使用 canonical head 时间，导入后更新，reorg 跨 fork 移除旧 Frame |
| `cmd/eth2030-geth` | upstream `node.New` → upstream `eth.New` → upstream `catalyst.Register` | geth wrapper；额外预编译注入不等于 native processor 或 Hegotá codec 已接通 |
| `pkg/Dockerfile` | builder 构建两种二进制，`ENTRYPOINT ["eth2030"]`；`geth` 链接到原生二进制 | 静态容器入口已核实；本任务未运行 Docker，不把宿主构建当镜像执行结果 |
| Kurtosis local | `run-devnet.sh` 选择配置 → `network_params.json`/ethereum-package → image/entrypoint | `full-feature` 是旧实验组合，不是官方 Hegotá profile；HGT-070/080/090 继续负责实际 CL 互通和脚本整改 |
| baseline 参考 | `go-ethereum@v1.17.1/tests.StateTest.RunNoVerify` | 直接 upstream EVM，不经过 ETH2030 geth wrapper，也不注入自定义预编译 |

| 配置/规则 | processor / VM | pool / API | 边界验证 |
| --- | --- | --- | --- |
| 历史 block forks → `Rules(number, merge, timestamp)` | 原有 cumulative rules 保留；TangerineWhistle fixture config 修正为 EIP150 level | 与时间 fork 分开 | 官方 EIP150/EIP158/Homestead/Byzantium/Constantinople 样本差分 |
| Prague / Amsterdam | 按旧配置时间启用已有执行规则，未宣称完整符合性 | Engine 版本使用区间：Cancun V3 在 Prague 前；Prague V4 在 Amsterdam 前；Amsterdam V5 在 Bogotá 前 | `TestEngineVersionForkIntervals` 前一秒/当秒/后一秒；native/helper 均使用相同区间函数 |
| `glamsterdamTime` / `glamsterdanTime` | 两种 JSON 名归一到原有 `GlamsterdanTime`，保留存储兼容；新增正确拼写查询别名 | 同值别名可共存，不同值拒绝 | `TestHegotaGenesisAliases` 的 legacy/canonical/equal/conflicting 用例 |
| `bogotaTime` / `hegotaTime` / `hogotaTime` | 归一到原有 `HogotaTime`；不设置任何官方主网日期 | 别名冲突拒绝，旧 CLI override 保留 | 同上；这里是配置名称适配，不意味着支持 Bogotá 共识 |
| 旧 Frame prototype | 只有 `ExperimentalLegacyFrames && merge && HogotaTime<=time` 才能到达执行；普通 Glamsterdam、I+、Verkle 不装旧 APPROVE/TXPARAM 指令 | genesis 中 `experimentalLegacyFrames` 显式 opt-in；无 chain config、未到 fork、canonical schedule 均拒绝；公开 RPC/pool 无单独绕过开关 | config、processor（状态/gas 不变）、VM、pool exact/reorg 四层测试 |
| 最新 8141 | 当前 native codec/processor 未实现，research profile 不激活 | 不能拿新时间点启用旧类型作为实现 | HGT-002…019/020…029 待实施；新 schema/goldens 单列 |
| AATx / LocalTx / PQ / STARK prototype | PQ 原有 I+ 门；其他旧实验接口纳入审计 | AATx 默认改为关闭；LocalTx/STARK 明确实验 opt-in；不列入 Hegotá admission | 默认配置、交易池负向测试；不是新候选自动激活 |
| Bogotá Engine | 新 `newPayloadV6`、`FCUV5` 未实现 | wire 返回 method-not-found，过期旧版本返回 unsupported-fork；旧 builder bid 提交拒绝 unsupported | 真实 native dispatcher 与 Engine 单测 |

兼容性限制：加载旧 genesis 仍可读取原有字段和 CLI override，不静默重写数据库。旧实验链若依赖缺省开启 Frame/AATx、无签名 ePBS 或无验证状态写入，升级后会被拒绝，必须显式承认原型范围；这类历史原型从未被本任务认定为规范兼容链。不存在通过开关把当前正式网络切到旧 schema 的迁移。
