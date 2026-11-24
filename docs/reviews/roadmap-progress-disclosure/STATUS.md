# Status

Moved the roadmap's Merged development progress panel to the final content position immediately before the footer. A native details/summary disclosure starts closed and supports mouse and keyboard toggling without additional client state. The shared homepage panel remains visible.

Validation passed: the 157-page production build, 149-entry / 398-path evidence verifier and export-claim checks. Four Playwright route/viewport checks cover desktop (1440px) and mobile (375px), initial/reload collapse, bottom placement, click/Enter/Space toggling, visible expanded content, no horizontal overflow or page errors, and unchanged homepage visibility. Both collapsed layouts were visually inspected. See BROWSER_CHECKS.json.

Based on merged master f39af0ba. The change is submitted independently for master review; hosted CI and deployment state remain authoritative on the PR. No protocol behavior or specification accounting changes.
