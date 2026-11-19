# Sparse relay CI observation correction in progress

After full blob admission, the cell-response worker removes staged bytes and announces the result. The integration test observes pool publication before that worker returns, then assumes cleanup has already happened. PR 90 CI reproduced this valid intermediate state. The correction waits for both pool admission and staged cleanup while retaining actual signed/KZG and bounded eventual-completion assertions; no protocol or production ordering is changed.
