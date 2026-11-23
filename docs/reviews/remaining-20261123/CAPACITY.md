# Rapid finite capacity qualification

The predeclared profile in CAPACITY_RESULTS.json tests current package bytes with a fresh native filedb-profile binary, separate fixture preparation, two real process opens and full value/digest verification for every key. Fixtures are newly owned under /datas/tmp/eth2030-capacity-20261123 on filesystem device8:1; unrelated nodes and OS caches are not modified. Limits are the existing300s startup and1GiB peak RSS budgets. No data-read throughput budget or multi-day verdict is invented after measurement.

| Fixture | First / second open seconds | Peak RSS KiB | Full-read seconds | Scope |
| --- | --- | --- | --- | --- |
|100k keys,128-byte values |0.166 /0.152 |72568 /72640 |1.355 /1.233 | Actual finite key metadata and per-key value access |
|500k keys,128-byte values |0.671 /0.608 |209272 /209784 |6.865 /6.085 | Same full-value/digest checks |
|1M keys,128-byte values |1.366 /1.189 |360820 /360504 |114.447 /100.681 | Metadata still grows with cardinality; all values actually read |
|10k keys,100 WAL generations |3.911 /0.011 |15076 /14832 |0.176 /0.118 | First open replays the original WAL; second opens persisted recovered data |

All8 process runs meet the declared startup/RSS budgets and preserve fixture contents. Preparation, filesystem/cache conditions and read costs are retained. This is a synthetic KV result, not proof of block/state-root execution, transaction throughput, physical I/O causation, arbitrary database size or48h operation. Original private million-file root/proof recovery and bounded6h45m/14-BLS-window records remain separate.

The current limit is explicit: value/WAL residency is bounded, but key/header/canonical metadata, history and complete authentication still grow. A public operating envelope must set supported state/history/workload and retention/restart budgets. The existing results justify a finite development profile; they do not silently close unrestricted production capacity.
