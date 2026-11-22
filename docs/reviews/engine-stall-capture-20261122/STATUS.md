# Original multi-hour import stall: evidence acquisition

As of2026-11-22T14:04Z the unchanged old-source diagnostic had run1h42m without a captured stall. The new source reached readiness at13:59:27Z; its30min clock had not finished. No full operating/BLS acceptance or original-root resolution is claimed.

The private early collector combines select(fd) with TextIOWrapper.readline. A single pipe read can prefetch both the last accepted-block line and a timeout line. If the EL then stops writing, select checks only the empty kernel pipe while the timeout remains in Python's user-space buffer. Test this exact quiet-after-burst condition before trusting a zero-capture report. The primary diagnostic still polls node health independently; keep it unchanged.

Native8s payload/FCU timeouts are the Engine Paris contract; this work captures evidence and does not relax validation or deadlines. Optional debugger attachment occurs only after an Engine timeout on an owned diagnostic and must be recorded as interference, never counted as clean operating time.
