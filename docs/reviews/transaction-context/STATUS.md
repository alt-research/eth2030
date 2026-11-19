# Transaction context isolation in progress

An actual published Uniswap V2 mint followed by a token transfer diverges from the independent Osaka oracle by 2,800 gas. The preceding pair lock generated a refund that the ordinary transaction path carried into the next transaction. The same path also omitted the EIP-1153 transaction transient-storage reset; Amsterdam and simulation had local resets. Consecutive signed controls and a common-boundary correction are being qualified independently of the application artifact review.
