# 固定规范与组合

HGT-001。所有上游版本与 SHA-256 在 [source-lock.json](../../../specs/hegota/source-lock.json)。`refs/` 正文未修改；尚未合入的 8334/8341/8359/8367/8375/8483/8504 以研究时的 PR head 固定副本保存，不冒充已合并 EIP。本文以 2026-11-25 的证据为准，没有使用次日会议的可能结论。

| 来源 | 固定版本 | 用途与相容性 |
| --- | --- | --- |
| EIPs | `ade9e9cab7e148bba4cd293f916a4a189cd7a8aa` | 当前正文和 Meta；草案不是既成激活规则 |
| consensus-specs | `d2021d2719235f0df4726a1d330502cb15aa8ea3` | Gloas/Heze 研究；7805 正文链接的旧 feature 分支不覆盖此版本 |
| EELS + 当前测试源码 | `b7e5b4bda2e1e62004b491dd74f119fe6622f0b3`，`forks/bogota` | 含 Frames 与测试；chain_id/nonce 类型与当前 EIP 存在差异，见 D09 |
| 旧 EEST 仓库 | `10eaa63d5da2f50b63d4359968f36542212f9f50` | 已归档；不是当前测试源码的维护入口，不伪造 fixture release |
| execution-apis | `1ca0818569467cff85b7f41372f22e728d5daa14` | Bogotá `newPayloadV6`/`FCUV5`/`PayloadStatusV2`；与当前客户端旧 wire 不相容 |
| geth 参考执行 | v1.17.1，`16783c167c4be5e6675fd8de0d1b762c88d6232f` | 只作选定历史语料的独立参考，不支持当前 Hegotá codec |
| legacytests | `1f581b8ccdc4c63acf5f2c5c1b155c690c32a8eb` | 两个原始官方 add11 文件，12 个 fork 用例；未编辑 oracle |
| Lighthouse stable | `57c2abc13078bf949e4d19df6178a644b064c32c` | 稳定参考，不等于 Heze 客户端 |
| Lighthouse Bogotá descriptor | `8e0aeef872c0bf8921c835344c68e749e15b2da7` | descriptor 使用的 unstable revision；不是已验证的联合运行组合 |
| Lighthouse Heze descriptor | `f66bd5ac1051d443f11b5c701410203f4e3cf4b3` | `heze-devnet-2` 固定源码；本地兄弟仓库不存在，未声称构建 |
| ethereum-package / spamoor | `90d0f4e795cd4c3298753b428bb588ed7999d4da` / `97fe0b9f36864ec02a73ac4a3da0b6a89ee83365` | 工具来源锁；不把 latest/master tag 当可复现镜像 |
| Bogotá / Heze devnet descriptors | `24ef46c39158297e9993d80f734534e89c2e4af0` / `7dd667c6ce3c702e002bf26a1d37b869f2440170` | config、images、README 的固定哈希；不能合成一个网络 |

| profile | 可以验证 | 不能推导 |
| --- | --- | --- |
| `native-history-v1` | 固定 12 个 GST 的 roots/logs/gas；native processor vs upstream geth | Prague/Glamsterdam/Hegotá 全规则、BAL、receipt root、传播和 CL 互通 |
| `legacy-frame-prototype-v0` | 历史代码的显式 opt-in + Hogota/Bogotá 时间门；结构/VM helper 测试 | 当前 8141 实现、公共网络或已发布实验 devnet 的兼容性 |
| `8141-base-research-v1` | 单独 8141 的字段顺序、签名长度承诺和语义 ADR；编码 goldens | 8250/8272/8369/8504 的合并编码或当前 native 执行符合性 |
| `bogota-devnet-2-descriptor` | 固定网络配置；GLOAS epoch 0、HEZE 禁用 | 当前 FOCIL 联调或 mutable image 的可复现构建 |
| `heze-devnet-2-descriptor` | 固定网络配置；HEZE epoch 5 | 当前 master Heze 参数（例如 IL deadline 不同）及 Frames/PQ 组合 |
| `hegota-release` | 当前状态为 `blocked_by_spec_and_implementation` | 尚无经过验证的联合发布组合；不得选一个“最新 SHA 集合”就称相容 |

D01–D08 沿用主计划编号，完整解除条件保存在 [profiles.json](../../../specs/hegota/profiles.json)。本次新增 D09：EIP-8141 要求 `chain_id < 2**256`、`nonce < 2**64-1`，所锁 EELS dataclass 使用 `chain_id: U64`、`nonce: U256`。选定 oracle 时必须有这些边界的独立向量，不能直接把 EELS 的可解码范围认定为 EIP 有效范围。

Engine API 的另一个明确差异：当前固定 API 将 inclusion satisfaction 放在 `PayloadStatusV2.inclusionListSatisfied`，`status` 仍是四种执行状态；仓库旧 helper 的 `INCLUSION_LIST_UNSATISFIED` 不可宣称实现了它。`getInclusionListV1` 当前返回原始交易数组，仓库旧对象也需在 HGT-042 修改。A 组保留这些功能缺口并对未知新方法/过期旧方法拒绝，不能自行拼接返回结构。

镜像 descriptor 使用 mutable tags，未取得 digest 的条目只用于比较配置；没有加入可运行 profile。HGT-070/091 必须以所锁源码构建镜像或解析并保存 digest，固定目标 EIP、genesis、Engine 版本和 CL 后再做互通。源码 SHA 锁定与镜像运行证明是两个验收条件。
