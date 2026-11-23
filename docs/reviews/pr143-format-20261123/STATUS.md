# PR143 CI format follow-up

The hosted Build/Vet/Lint/Test job stopped at gofmt, reporting engine/server.go. Its other8 jobs passed. This patch applies gofmt to that file; no Engine behavior or validation changes. Paris forkchoice rules4-6 remain unchanged. The independent cumulative PR142/143/144 control passed219 Go packages,3 affected race packages and native build before this whitespace-only repair. Exact new-head hosted verdict remains separate.

Affected Engine package tests, full package gofmt scan and native Go1.26.11 build pass after the repair.
