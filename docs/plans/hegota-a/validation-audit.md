# 校验、stub 与实验路径调用审计

> Historical A-branch audit through `6a17479`. Current master integration changes several runtime conclusions; see the [PR #167 integration audit](master-integration-167.md) and current evidence.

HGT-008。审计覆盖本任务的 native node 初始化、Engine wire、processor/VM、pool、ePBS/FOCIL/PQ/BLS/KZG 校验入口和 unwired state backends。它不是整个仓库所有密码算法或 zkVM 的安全证明。实验模型可以保留，但实际入口不能把“没验证”包装为成功。

| 路径 / 调用者 | 原风险 | 处置、反向证据与剩余职责 |
| --- | --- | --- |
| `epbs.ValidateBuilderBid` → auction / full bid validator | 零 pubkey 或零 signature 跳过验证，注释允许 testing；旧 BidHash 非 canonical SSZ domain | nil/零签名/零 key 拒绝；非零也因缺可信 CL authentication context 返回 unavailable。结构检查独立命名，不能用于认证。auction 不存被拒绝 bid；排序单测仅给私有插入函数预置模型数据，不调用公共 admission 假装成功 |
| `engine.SubmitBuilderBidV1` / `RegisterBuilderV1` → 内部 registry | 只检查字段/注册状态，忽略 signed wrapper | 两个外层提交接口明确 unsupported。wire 有效参数也收到错误，不承认测试签名。内部 registry 保留结构/排序模型，未接 canonical CL；HGT-006-R01/070 负责正式集成 |
| 默认 BLS backend / `BLSSign` / `BLSVerify` / aggregate | 自制 hash-to-curve 与 pairing、已知私钥重签 fallback；节点日志称 blst 但实际未选中 | 使用固定 gnark-crypto 的 canonical hash-to-G2、曲线/子群/infinity/长度检查与 pairing；删除私钥 registry/fallback，不能吞掉无效 aggregate 成员。独立 blst 生成固定签名，gnark 验证；错误 msg、zero、infinity、截断 key/sig、空 aggregate 负向测试。pure-go 默认真实实现；显式请求未编译 blst 返回错误 |
| `crypto/pqc.PQKeyRegistry.VerifyRegistration` | 公开可计算的 digest 冒充签名 | 默认一律 false；RegisterKey 只是元数据存储。固定 8483 是 withdrawal/execution ownership + leanXMSS PoP，请求预部署地址仍 TBD；不能用 BLS/ECDSA 临时替代它。旧 digest 不再通过测试；HGT-076 负责规范实现 |
| `focil.CheckILSatisfaction[WithVerifier]` | nil state 跳过状态检查；Frame 的 code-size 或 nil verifier 代替完整 prefix | 新 unavailable 结果；缺 post-state、未恢复 sender 或任何 absent Frame 不能获得 proved satisfied/unsatisfied。存在于 block 或 gas exemption 可独立判定。helper Engine 将 unavailable 回为明确 error；不把它变成执行 INVALID 或已符合最新 PayloadStatusV2 |
| native `ProcessInclusionList` | 非空 IL 没有实际 store/build 路由 | 保留并验证非空 IL unsupported，未将“不报错”作已接线。空 IL 没有存储义务。正式 wire/存储由 HGT-041/042/045；原生节点不使用上行 helper 的 IL state |
| `engine/blobsbundle.AddBlob` | nil KZG verifier 直接保存数据 | 缺 verifier 返回 `ErrVerifierUnavailable` 且 count 不变；commitment/proof 错误不能保存。原有 builder 结构单测显式提供模型 verifier；新的缺 verifier 测试调用真实拒绝路径。真实 native Engine 的 KZG 路径与这个 helper 单独登记 |
| native KZG initialization | trusted setup/backend 初始化失败仅告警并使用零 cell proof | 初始化错误现在使 `node.New` 失败；不构造可运行但伪造 proofs 的节点。现有 backend 正/反向测试和节点构造验证正常路径；KZG 原始异常不被吞掉 |
| snap fallback / account+storage proof / trie node / healer+state-sync writers | 返回 `nil` 冒充完成读证明/持久化 | 所有 unwired error-capable操作明确 unsupported；真实 snapshot trie proof 未接线也报错。带 stub writer 的 healer/scheduler 不再在 node 初始化，避免 `MissingTrieNodes=nil` 被当作完整状态。真实修复接线由 HGT-087 |
| `stubBeamFetcher` / `stubCustodyManager` | 尚未有真实 P2P 数据源 | 原代码已返回明确 unavailable/error；没有 false success，未改成 permissive mock。同步/DAS 端到端仍是后续任务 |
| 旧 Frame / AATx / LocalTx / STARK | helper 通过可能被误写成正式支持 | Frame 必须实验配置与 fork 双门；AATx 默认关闭，LocalTx/STARK 原有 opt-in 保留。默认 canonical 节点不启用旧 Frames。旧原型、PQ/raw witness/聚合、zkVM 不进入 Hegotá conformance 统计 |
| `types.ValidateSignedInclusionList` 等结构 helper | 仅非零 signature 的历史模型函数，不能证明 CL gossip 认证 | 检查调用：没有生产调用者。它只做结构检查，CL registry/domain/gossip 认证必须在 HGT-040/070 接通后以独立向量验证，不将该 helper 成功算签名验证 |

回归包括 nil/zero 输入、缺 verifier、canonical schedule 隔离、未到/正好到 fork、reorg 跨边界、canonical unsigned Engine 入口、unsupported writes、错误 wire 版本和真实标准 BLS 签名。拒绝后不修改 state/gas、pool 或 auction；这一性质有测试，不能通过“测试环境可跳过”恢复旧行为。

行为变化：无签名 ePBS 原型、公开 digest PQ、缺 verifier blobs、默认 AA/Frames、假 state persistence 升级后不能继续得到成功。它们没有作为规范兼容网络被本任务承诺。要恢复相应能力，必须补齐实际校验/持久化并关闭对应修复任务，不能增加测试白名单。
