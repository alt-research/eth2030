# A 组交付：规范、基线与运行路径

日期：2026-11-25（Asia/Shanghai）。实施分支：`codex/hegota-a-baseline`；起点 `8ac0ec4`。A 组包含 HGT-001、004、005、006、007、008、009 七项。本目录与 `specs/hegota/`、`evidence/hegota-a/` 构成一份可检查的交付，不改变 B–J 组尚未实现的事实。

| 任务 | 交付与验收入口 |
| --- | --- |
| HGT-001 | [固定版本与组合](specification-profiles.md)、[来源锁](../../../specs/hegota/source-lock.json)、[决策与 profile](../../../specs/hegota/profiles.json)；14 个仓库版本、106 个文件校验值，未决组合隔离 |
| HGT-004 | [实际运行路径](runtime-and-activation.md)、`TestNativeRuntimePathAndUnsupportedBogotaAPI`、二进制身份与源文件校验；native、wrapper、容器分别登记 |
| HGT-005 | [激活矩阵](runtime-and-activation.md)、配置别名冲突检查、processor/VM/pool/Engine 边界与 reorg 测试；canonical fork 不隐式启用旧 Frames |
| HGT-006 | [前置与历史基线](baseline-and-gaps.md)、[逐规则登记](../../../specs/hegota/conformance.json)、严格差分 CLI；12 个官方历史用例，native 与上游 geth 分开执行 |
| HGT-007 | [组合 ADR](semantic-adrs.md)、[schema](../../../specs/hegota/frame-schema.json)、7 个独立 Python RLP/Keccak golden，并由上游 Go RLP/Keccak 交叉检查；不声称旧 codec 符合新草案 |
| HGT-008 | [校验调用审计](validation-audit.md)、强制拒绝/unsupported、规范 BLS backend 和反向测试；没有把未验证输入作为生产成功 |
| HGT-009 | [72 项准入登记](../../../specs/hegota/admission.json)；3 实施主线、29 条件、3 观察、37 排除，每项有版本、讨论、条件、受影响任务和差异历史 |

A 组完成的是规范边界、可信基线和失败路径的建立。它没有关闭 D01–D09 的全部上游问题，也没有完成最新 FrameTx、AA-VOPS、原生非空 IL 存储、Heze CL 接线或官方 devnet 互通。它们分别有明确后续任务和阻断条件；不能据此发布“支持 Hegotá”的公共网络版本。

重新验证：

```sh
cd /Users/fy/Projects/eth2030
python3 tools/hegota/check.py
python3 tools/hegota/codec_goldens.py
python3 tools/hegota/semantic_goldens.py
cd pkg
env -u GOROOT go run ./cmd/eth2030-baseline
env -u GOROOT go build ./...
env -u GOROOT go test ./... -timeout=15m -count=1
env -u GOROOT go test -race ./core/config ./core/eftest ./core/execution ./core/vm ./crypto/bls ./txpool ./node/... ./engine/... ./epbs/... ./focil
env -u GOROOT go vet ./...
```

运行结果与实际执行/跳过数见[证据摘要](../../../evidence/hegota-a/verification.json)。必需 baseline 缺失、checksum 改动、零 oracle、异常通配、未知执行字段或执行数不符都返回非零；全仓库旧测试的 skip 单独登记，绝不加入这 12 项的通过数。日后更新来源使用 `tools/hegota/check.py --compare-lock NEW_LOCK --compare-admission NEW_ADMISSION` 审查差异；保存新锁、准入历史和影响任务后，重跑受影响用例。
