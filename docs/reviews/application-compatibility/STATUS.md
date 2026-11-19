# Real application compatibility in progress

Official state conformance is qualified separately. This review executes pinned actual application artifacts and transaction sequences, including their wallet/router gas and failure behavior. The user-selected merged LIFO target remains authoritative; known gasleft attribution limitations are evaluated rather than hidden by adopting open alternatives. Implementation and qualification are in progress.

The first actual sequence identifies a native Osaka transaction-boundary defect: the pair mint's 2,800 refund reaches the following token transfer (native 31,505 vs independent 34,305 gas). A separate ordinary-transaction-context correction owns this defect and transient-storage lifetime. The initial guessed 15,000 forwarded gas does not fund the real Safe child on Amsterdam either; the application gas boundary must be measured rather than inferred from a synthetic SSTORE. Both failed runs are retained and are not qualification results.
