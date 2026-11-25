# Status

Moved Remaining work to the bottom of the roadmap, immediately before Merged development progress and the footer. Both panels use the same native disclosure styling and start closed. The existing remaining-work fragment belongs to the outer disclosure so direct links and homepage navigation land on its visible summary while keeping the content collapsed.

Validation passed: production build (157 static pages), the 149-entry / 398-path verifier and export-claim checks. Six desktop/mobile Playwright checks at 1440px and 375px cover default/reload collapse, bottom order, click/Enter/Space toggling, independent merged-panel state, nested workstream expansion, retained inventories, direct fragments and homepage navigation. No horizontal overflow or page errors. The collapsed mobile layout was visually inspected. See BROWSER_CHECKS.json.

Based on merged master 2b94f73d. Submitted independently for master review; hosted CI and deployment state remain authoritative on the PR. No protocol or development-progress accounting changes.
