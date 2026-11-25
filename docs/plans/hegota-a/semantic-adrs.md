# 组合 schema 与语义 ADR

HGT-007。所有正文精确到 [source-lock.json](../../../specs/hegota/source-lock.json)；ADR 是后续实现的约束和未决边界，不是对旧代码的符合性背书。`8141-base-research-v1` 只采用单独 8141；8250、8272、8369、8504 不会因列在同一目录而自动合入。

## ADR-A01：编码、哈希与签名承诺

采用固定 8141 的七字段 payload：`[chain_id,nonce,sender,frames,signatures,fees,blob_versioned_hashes]`；frame 为六字段，`limits=[execution,state]`，fees 为 `[max_fee,max_priority_fee_per_gas]`。交易哈希是 `keccak(0x06 || rlp(payload))`；canonical signature hash 将所有 empty-msg 签名的原始字节替换成“最小 RLP 整数的字节长度”，explicit-msg 签名不替换。不能复用旧 `ComputeFrameSigHash`。

[frame-schema.json](../../../specs/hegota/frame-schema.json) 固定结构与静态范围；签名密码学、flags 组合、batch 相邻性和 state validity 仍由正文规则约束。`chain_id < 2**256`、`nonce < 2**64-1`、1–64 frames、仅 mode 0/1/2，flags 保留位拒绝，blob hash 长度/版本检查。D09 的 EELS 类型差异未消失，goldens 选择两者共同可表达的 chain ID，不据此关闭宽度争议。

[7 个编码 goldens](../../../specs/hegota/codec-goldens.json) 由独立 Python 最小 RLP + Keccak 产生（校验 empty/abc 已知 Keccak），Go 测试再用 upstream geth RLP/Keccak 验证 bytes、tx hash、signature hash。覆盖同长度签名变更、长度变化、explicit-msg 变更、最大 base nonce、64 frames 长 RLP。它们是 encoding-only；ARBITRARY bytes 不证明任何账户授权，不能作为有效执行交易提交。原生新 codec 的正/反向共识向量由 HGT-002/019 消费。

8250 文字仍给三项 fees，8141 已是总 fee budget，两者合成 schema 保持 `blocked_by_spec(D01)`；不自行选择三项中的某个映射，也不让旧 192+64 nonce 的实现成为新 keyed-nonce 编码。

## ADR-A02：approval、身份与验证边界

采用当前 scope bitmask：payment=1、execution=2、combined=3；scope 非零且必须是 frame.flags 的子集，`ADDRESS == resolved_target`。execution 只能由 sender target 批准且不能重复；payment 需要 sender 已批准、payer 未设且资金充足。combined 同时满足两者，不能重复。payment/combined 在必要的 account-creation state charge 成功后递增 sender nonce、设 payer、收取 max_cost；execution-only 不消费 nonce。APPROVE 按 RETURN 的 memory expansion 收费，无额外 base gas。

base 8141 支持空代码默认账户和已有 delegation，不可用“VERIFY target 有代码”作为通用有效性判断。VERIFY 为 static，例外只限 APPROVE 的规范效果；VERIFY 失败使整笔交易无效。公开 mempool admission、block validity、IL eligibility 是不同判断，FOCIL 缺完整 prefix 规则时返回 unavailable。8369 是 Informational，D04 关闭前不能拿简化 sender balance/code size 代替它。

语义 oracle 场景：`scope=0`/reserved bits 拒绝；`scope=execution,target!=sender` 拒绝；payment-before-execution 拒绝；重复 payer 拒绝；执行批准后 payment 仅消费一次 nonce；default account 有有效 canonical SECP256K1 signature 可验证而不要求部署代码。实现验收任务 HGT-003/019/020，当前 prototype tests 不计这些场景已符合。

## ADR-A03：gas、fee 与 block capacity

base 8141 使用 execution/state 两个独立 frame pool；禁止由 execution pool 支付 state charge，也不能把 reservoir 模型套进 Frames。intrinsic=12000+475/帧+数据/签名验证与价值成本；签名 empty-msg 的字节按长度统一计费。`max_gas` 由 standard limit 与 calldata floor + state budget 决定；`required_max_fee=max_gas*base_fee + blob_gas*blob_base_fee`，预算不足在执行前拒绝。

`max_cost` 收取时机在 APPROVE；settlement 给 payer refund、burn base/blob fee、coinbase 得 tip。payer 费用使用 post-refund gas，block execution capacity 使用 pre-refund/floor，state capacity 使用净 state gas；两个维度分别做 worst-case reservation。只看总 gas 不足以验收。

state charge owner 与相关 state modification 同时 journal；跨 frame refill 减少原 owner receipt 的 state gas，但不为新 frame 打开额外预算。oracle 场景：仅 state pool 耗尽、execution 耗尽、跨 frame 删除/重建、refill 后 revert、refund cap/floor、blob budget 精确边界；HGT-011/015 在所锁 8037/7778/7976 下执行。未采用的 CPSB 替代候选 8368/8372 不自动改变参数。

## ADR-A04：rollback 与 receipt

frame revert 恢复 state、approval、warm sets、state-gas ownership、早先 receipt state gas 的改动；VERIFY revert/exception 使交易无效。atomic batch 的所有成员禁止 approval bits，失败回到 batch-entry：日志移除，已执行帧保留执行状态和 execution gas、state gas 为零，后续帧 status=2 skipped。batch 前付款和 nonce 消费不因 body batch failure 回滚。transient storage 每帧清空。

receipt 为 `[cumulative_gas_used,payer,[frame_receipt...]]`，frame receipt 为 `[status,[execution,state],logs]`；没有 transaction-level status。block logs 按 frame 顺序连接。oracle 场景：batch 中第二帧失败、batch 前 nonce/payment 保留、跨 batch state refill 被撤销、后帧不能读前帧 transient storage、receipt/log root 改动；HGT-012/014，当前 scalar receipt 不能充当目标 oracle。

## ADR-A05：nonce/roots 组合边界

单独 base 8141 使用 account nonce；8250 扩展将它替为 `(nonce_keys,nonce_seq)`，key 必须严格递增、不得重复，空 key 集选 legacy nonce；non-empty 集合 + MAX_NONCE_SEQ 为 single-use。keyed nonce 在 approval 成功时消费，原型大整数高位拆分不满足此规则。8250 的 fees 描述未同步，所以只冻结其独立规则与待组合向量，不产生可激活合并 envelope。

8272 不再添加 root envelope：通过 canonical VERIFY frame 承载 root references。source_id/entry_hash/storage_key 使用固定宽度域分离；slot 来自 header/Engine `slotNumber`，不能由 timestamp 计算。当前槽写入只从下一槽可引用，8192 环与 8191 usable window，重组必须重验，普通合约部署不能冒充 fork 系统安装。8369 仍依赖旧 envelope/root/nonce 模型，D04 之前不激活。

oracle 场景：key 集排序/重复/空集/MAX、nonce ownership 与 approval rollback；slot 精确窗口、空槽、环覆盖、source/salt 域分离、reorg 后旧 branch root；HGT-017/018/022/046。D01/D04 未闭合时，这些是条件向量需求，不是虚构的组合有效交易。

## ADR-A06：golden 和来源更新

独立编码 goldens 只允许显式 `codec_goldens.py --write` 更新，验收默认仅比较。来源更新保存日期、old/new hash、影响任务、采用/延期理由，72 项不删除历史条目。baseline oracle 不可由 native run 覆盖；新增 expected exception、receipt/BAL 差分、运行网络与大语料必须另增版本和覆盖声明。当前 schema 与 ADR 不授权对外发布。


另交付 [semantic-goldens.json](../../../specs/hegota/semantic-goldens.json)：2 个手工推导的完整 gas/fee/refund 数值 oracle、1 个 8272 source/entry/storage 域分离 oracle，以及 9 个 approval/rollback/keyed-nonce/recent-root trace seeds。`semantic_goldens.py` 用独立公式/Keccak 复核前 3 个；9 个 trace 明确标为待 B 组真实 EVM 执行，不将声明的期望当成已运行测试。
