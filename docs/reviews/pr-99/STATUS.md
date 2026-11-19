# Sparse relay completion observation qualified

PR 90 CI observed a legitimate intermediate state: executable pool admission publishes a reconstructed blob before the same response worker returns and removes its sparse staging entry. The test previously waited only for pool publication and then assumed staged cleanup had already completed.

The actual RLPx two-provider test now waits, under the same bounded deadline, for both full pool admission and staged cleanup. All signed/KZG validation, custody/noise/expansion counts, Engine cell projection and exact reconstructed blob assertions remain. No protocol rule, production ordering or timeout is changed, and cleanup that never occurs still fails the test.

Five consecutive full peer controls and two race repetitions pass, along with affected eth/sparse suites, scoped vet and the composed build. This review is stacked on PR 96 so the separately repaired simulation-timeout classification does not obscure its CI. It does not claim the remaining CL partial-column/version/loss/reconnect campaign complete.
