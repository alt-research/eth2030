# Canonical fork schedule and Engine boundaries

Branch `feat/glamsterdam-fork-schema`, independently based on `38ea9259`.
This qualifies canonical startup schedule validation and the production Engine
HTTP boundaries for GL-012. Program/release completion remains separate.

## Normative input and reproduced failures

Read the complete [common](https://github.com/ethereum/execution-apis/blob/68d76a5b1c09dff023782d97eec6ce87a6f6505e/src/engine/common.md),
[Cancun](https://github.com/ethereum/execution-apis/blob/68d76a5b1c09dff023782d97eec6ce87a6f6505e/src/engine/cancun.md),
[Prague](https://github.com/ethereum/execution-apis/blob/68d76a5b1c09dff023782d97eec6ce87a6f6505e/src/engine/prague.md),
[Osaka](https://github.com/ethereum/execution-apis/blob/68d76a5b1c09dff023782d97eec6ce87a6f6505e/src/engine/osaka.md)
and [Amsterdam](https://github.com/ethereum/execution-apis/blob/68d76a5b1c09dff023782d97eec6ce87a6f6505e/src/engine/amsterdam.md)
Engine specifications, pinned to `68d76a5b1c09dff023782d97eec6ce87a6f6505e`.
The Amsterdam/Osaka/Prague/Cancun file blobs were rechecked unchanged at that
revision: `a49c6c4c`, `b8a3e477`, `dba1f2ec`, `73395318` respectively.

Baseline tests reproduce acceptance of missing canonical predecessor forks,
Osaka after Amsterdam and asymmetric activated-fork configuration comparisons.
The real production dispatcher also aliases getPayloadV5 to V6. Existing tests
explicitly asserted that getPayload methods never check fork boundaries, and
several supposed BAL fixtures actually used base64 or JSON objects.

## Implementation

- Validate the canonical Shanghai/Cancun/Prague/Osaka/Amsterdam prefix, allowing
  simultaneous activation and earlier stopping points. Include Osaka in fork
  enumeration, configuration identity and active-fork selection. Test/dev
  schedules explicitly include inherited Osaka rules; the EIP-1559 heavy-block
  fixture now uses two below-cap transactions instead of one invalid 25M tx.
- Apply configuration validation on actual supplied-genesis loading after CLI
  overrides, default-network startup and genesis setup. Invalid schedules are
  rejected before genesis adoption. Configuration comparison checks activation
  on either side, so reversing its arguments cannot hide an already active
  difference. This comparison utility is not claimed to be a persisted startup
  configuration guard; that production consumer remains follow-up work.
- Enforce newPayloadV3 before Prague, newPayloadV4 before Amsterdam, and
  getPayloadV3/V4/V5/V6 in their respective Cancun/Prague/Osaka/Amsterdam windows.
  FCU V3 build attributes stop at Amsterdam; null-attribute forkchoice updates
  remain valid. Existing private successor methods remain separate.
- Canonical getPayloadV5 returns ExecutionPayloadV3 and the cell-proof bundle;
  it no longer leaks Amsterdam BAL/slot fields or an inner request list. The
  historical standalone helper's response is explicitly named
  `LegacyGlamsterdamPayloadResponse`. Production HTTP dispatch uses the canonical
  response. Both actual node and memory backends expose Osaka activation.
- Validate required payload/attribute/forkchoice fields, exact DATA widths,
  canonical bounded QUANTITY values and nested withdrawal/transaction/hash
  arrays before permissive internal codecs can pad or truncate them. Null is
  absent; a present zero beacon root or slot is valid. Invalid attributes report
  -38003 after valid forkchoice adoption, while an unknown head returns SYNCING.
  Selected snake-case Lighthouse attributes retain the same validation.
- Validate canonical getPayload IDs as eight-byte DATA and stop advertising
  engine_exchangeCapabilities in its own response, as required by common.md.
  Attribute encoding always supplies targetGasLimit and empty withdrawals [].

No new runtime option or environment flag is introduced. This is wire validation
at the actual canonical HTTP entry points, not a replacement for all historical
internal JSON decoders or a claim to qualify every experimental helper.

## Actual consumer evidence

The real node-backed HTTP test builds/imports both an Osaka payload at time 12
and an Amsterdam payload at time 24. It checks all wrong getPayload versions,
rejects old newPayload methods after activation, and compares actual response
field sets and empty blob arrays. Both zero root and zero slot are exercised.
This branch qualifies acceptance/presence; independent zero-slot header
commitment is provided separately by PR 74 and is composed in integration.

Every one of 19 required payload fields and seven required attribute fields is
independently missing and null in otherwise valid inputs. Negative cases cover
malformed/short DATA, odd transaction bytes, null arrays, invalid withdrawals,
decimal/negative/noncanonical quantities, 65-bit truncation aliases of a valid
timestamp/gas limit and an oversized base fee. Malformed BAL RLP yields INVALID
with a null ancestor, distinct from malformed JSON DATA yielding -32602.
Missing attributes still adopt valid safe/finalized pointers; unknown heads
take precedence. Actual Node construction rejects each missing predecessor.

All 210 tested Go packages, 27 tagged Core/EIP packages, targeted race, affected
vet and the local build pass. Subsequent affected/race tests include the added
startup/HTTP cases. Logs are `/tmp/glamsterdam-fork-schema-*.log`.

Behavioral mutations that remove the 64-bit wire bound, allow old newPayloadV4
after activation, restore the V5/V6 alias or reject attributes before applying
forkchoice each fail the real HTTP regression. Sources are restored and tests
pass again. The truncation mutation uses values aliasing an otherwise valid
timestamp/gas limit, so another parser's unrelated rejection cannot mask it.

Remaining GL-012/program work includes persisted configuration compatibility,
complete optional/private fork registries, successor/experimental schemas,
custody-set adoption (GL-042), official activation inputs, complete asynchronous
and independent-client matrices, the final gas schedule and release validation.
