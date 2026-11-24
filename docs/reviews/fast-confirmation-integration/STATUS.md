# Status

Local implementation and affected HTTP/race checks pass. Three executable negatives reproduced mixed-root quorum inflation, out-of-range validator support and caller-owned configuration mutation. The utility now rejects/isolates those inputs; rejected foreign observations do not consume a valid candidate vote. Full counter-package race and the native HTTP safe consumer test pass.

The actual production route uses Lighthouse FCR and standard Engine safeBlockHash, not the standalone Go counter. Native HTTP acceptance covers same-head confidence updates, invalid-checkpoint rejection and restart/reannouncement. Companion PR https://github.com/jiayaoqijia/lighthouse/pull/17 repairs newly reproduced crash durability and qualifies both Fulu/Gloas 22-case signed restart matrices. See CONTRACT.md.

Fresh process/BLS and joint real-proof qualification tools are included. Their actual final run receipts are recorded by the final extension handoff; they are not inferred from controlled HTTP or unit results. The real-proof mode requires the separately reviewed canonical-node bundle. This initial feature snapshot retains process/hosted CI as pending, without claiming an actual 48h/scale pass or original-04 resolution.
