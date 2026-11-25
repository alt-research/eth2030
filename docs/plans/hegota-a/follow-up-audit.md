# A 组提交前复核

日期：2026-11-25；复核起点：`f490fe9`。按主计划 A 组全部 7 项验收条件重新检查，发现遗漏后补齐代码、测试和证据；未将发现的后续功能缺口改成已完成。

| 任务 | 复核结果与补齐 |
| --- | --- |
| HGT-001 | 原 14 个仓库 pin 保持；增补固定 EIP-3675 的规范文件与 dated file-impact record，来源锁共 108 文件；未启用联合发布 profile |
| HGT-004 | 原生 P2P 导入和重组监听器此前只刷新 state，未刷新 fork 上下文。现在二者与 Engine 导入一样，携带实际 canonical head 的 number/time。新增真实 NewBlock 消息导入、链回退与已安装监听器测试。另登记 native FCU 不应用 canonical head/state 的 R16，未把 listener 测试混算为 FCU 验收 |
| HGT-005 | 补齐 London block prerequisite；VM 先选择 base fork 再叠加旧 Frame 指令，避免 I+ 吞掉 Frame gate、Frame 吞掉 Verkle gas。config compatibility 补上 IPlus 以及 remote 侧已生效 schedule 检查。新增 exact activation/after-fork 的成功 processor 向量，明确仅验证旧原型 |
| HGT-006 | 严格 runner 此前未检查 manifest 第一份 JSON 后的尾随数据；现在多份 JSON/垃圾数据都拒绝，破损证据用例从 9 类增至 11 类。固定官方 corpus 保持 12 项、原始 oracle 未改。新发现 FCU/恢复接线问题单独登记 HGT-006-R16 |
| HGT-007 | 重核 schema/ADR、7 个独立编码 golden、2 个 gas oracle、1 个 root-domain oracle 与 9 个未执行 trace seeds 的声明；没有将 prepared seeds 计入 native 当前规范通过数 |
| HGT-008 | 重查零签名、缺 verifier、公开 PQ digest、无实现 state writer 与 backend 路由；保持明确拒绝/disabled。修复的激活条件在 processor、RPC VM 与 pool 使用相同 block/time 前置条件，无测试白名单 |
| HGT-009 | 原比较器能漏掉 lock 改变但 admission 未变、相同字节新 revision、重写历史与非 EIP repo 更新。现在要求 lock/admission 一致，append-only dated 影响决策、repository/file impact records，拒绝 immutable URL checksum 改写及重复 repo/ref；12 个检查器回归用例覆盖 |

`TestNativeP2PAndReorgRefreshFrameActivation` 使用真实节点、编码后的 NewBlock 消息和 native blockchain。重组部分先由 `Blockchain.SetHead` 应用状态回退，再通过真实 forkchoice manager 触发注册监听器。它验证 pool 的 activation context 跟随实际 head，没有声称 `EngineBackend.ForkchoiceUpdated` 已自动应用该 head。

新增 `TestProcessorOptedInLegacyFrameExecutesAtFork` 检查明确 opt-in 的旧 Frame 在边界和边界后一秒产生成功收据、消耗 gas、递增 nonce。其旧 scope=2 与当前 8141 scope 规则不同，仍严格隔离在 prototype profile；它不能作为最新规范的有效执行向量。

开放功能缺口仍阻断发布：HGT-006-R01…R16、组合决策 D01–D09、完整最新 FrameTx/FOCIL/CL 联调。R16 还明确要求启动恢复时实际使用配置兼容性校验，而不是只存在一个 helper。legacy builder 默认 gas-limit 调整的精确边界问题也记入 HGT-013 的构块/导入一致性向量要求；本轮 P2P 测试显式固定 target gas limit。

最终构建、测试、race、baseline、来源与证据 hash 结果见 [verification.json](../../../evidence/hegota-a/verification.json)。提交与 push 后再检查远端分支 SHA 与本地提交相同；未执行 Docker/Kurtosis 或外部发布。


全量测试另暴露已有 `TestStreamSessionUniqueIDs` 间歇失败，单独 20 次复现出现多次相同 ID。根因是 local session key 只使用 blob index、size 和 `UnixNano`，相同 clock tick 可以碰撞。改为加入进程内 atomic sequence，并用冻结时钟下 256 个并发 session 的测试确定性覆盖。它只影响内部接收跟踪 key，不改变 PeerDAS cell/blob/KZG 或共识编码；规范背景 EIP-7594 也纳入来源锁。失败和修复后的重跑分别保留在验证记录中，没有将第一次失败丢弃后直接宣称通过。
