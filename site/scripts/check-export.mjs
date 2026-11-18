import { readFileSync, existsSync, readdirSync } from 'node:fs';
import { strict as assert } from 'node:assert';

const home = readFileSync(new URL('../out/index.html', import.meta.url), 'utf8');
const pq = readFileSync(new URL('../out/pq-roadmap/index.html', import.meta.url), 'utf8');
const tracker = readFileSync(new URL('../out/eips/index.html', import.meta.url), 'utf8');
const roadmap = readFileSync(new URL('../out/roadmap/index.html', import.meta.url), 'utf8');
// The retained live control is the oracle for published milestone values.
const recordPath = 'docs/verification/frame-8141-20261118-paymaster-history-final.json';
const record = JSON.parse(readFileSync(new URL(`../../${recordPath}`, import.meta.url), 'utf8'));
const progress = html => html.match(/<aside aria-label="Verified development progress"[^>]*>(.*?)<\/aside>/s)?.[1];
assert.ok(progress(home), 'Missing homepage development progress');
assert.equal(progress(home), progress(roadmap), 'Homepage/detail development snapshots diverged');
for (const milestone of [
  `Slot ${record.sponsored.proposal_slot}`,
  `Slot ${record.blob.slot}`,
  `Joined at slot ${record.observer.join_slot}`,
  'Readback passed',
]) {
  assert.ok(progress(home).includes(milestone), `Missing qualified milestone: ${milestone}`);
}
assert.equal(record.sponsored.finalized_on_both_pairs, true);
assert.match(record.outcome.blob, /finalized on both pairs/);
assert.match(record.outcome.restart, /recovered from the restarted receiver/);
assert.match(record.outcome.qualification_boundary, /already-synchronized second EL/);
assert.match(progress(home), /synchronized EL/);
assert.match(progress(home), /Independent EL interoperability, Dora consumer migration and state-sync acceptance remain/);
assert.match(progress(home), /November 18, 2026/);
assert.match(progress(home), /href="https:\/\/github\.com\/alt-research\/eth2030\/pull\/61"/);
assert.ok(progress(home).includes(`/blob/911afb0c42a2e33c9e65881b31ed1b1c446390b8/${recordPath}`), 'Missing pinned verification record');
const homepageRoadmap = home.split('id="roadmap"')[1]?.split('</section>')[0];
assert.ok(homepageRoadmap, 'Homepage roadmap section is missing');
assert.match(home, /November 2, 2026/);
assert.match(home, /VERIFICATION INCOMPLETE/i);
assert.match(home, /not production-ready/i);
assert.match(home, /github\.com\/alt-research\/eth2030/);
assert.match(home, /Go 1\.26\.8\+/);
assert.match(home, /Moving-pivot range reuse/);
const homepageHero = home.split('<section')[1]?.split('</section>')[0];
assert.ok(homepageHero, 'Homepage hero is missing');
for (const action of ['roadmap', 'github', 'docs']) {
  assert.match(homepageHero, new RegExp(`data-hero-action="${action}"`), `Missing hero action ${action}`);
}
assert.match(homepageHero, /href="\/roadmap\/"/);
assert.match(pq, /stub rather than a real signature check/);
assert.doesNotMatch(home + pq, /65\/65|26\/26 gaps|100% EF State Tests|Live on Sepolia &amp; Mainnet/);
assert.match(home, /href="\/eips\/"[^>]*>EIP Progress Tracker/);
assert.match(homepageRoadmap, /href="\/roadmap\/"/);
assert.doesNotMatch(homepageRoadmap, /data-status="complete"/, 'Homepage work list contains a complete entry');
for (const fork of ['glamsterdam', 'hegota', 'i', 'j', 'k', 'l']) {
  assert.ok(homepageRoadmap.includes(`href="/roadmap/#fork-${fork}"`), `Missing homepage link for ${fork}`);
  assert.ok(homepageRoadmap.includes(`data-phase="${fork}"`), `Missing homepage card for ${fork}`);
  assert.ok(roadmap.includes(`id="fork-${fork}"`), `Missing roadmap detail card for ${fork}`);
  assert.ok(tracker.includes(`id="fork-${fork}"`), `Missing tracker section for ${fork}`);
  const homeCard = homepageRoadmap.match(new RegExp(`<a[^>]*data-phase="${fork}"[^>]*>(.*?)</a>`, 's'))?.[1];
  const detailCard = roadmap.match(new RegExp(`<article[^>]*id="fork-${fork}"[^>]*>(.*?)</article>`, 's'))?.[1];
  assert.ok(homeCard && detailCard, `Missing rendered cards for ${fork}`);
  const percentage = html => html.match(/aria-label="(\d+)% of mapped tracker entries (?:are )?labelled complete"/)?.[1] ?? null;
  assert.notEqual(percentage(homeCard), null, `Missing tracker percentage for ${fork}`);
  assert.equal(percentage(homeCard), percentage(detailCard), `Homepage/detail percentage mismatch for ${fork}`);
}
assert.ok(tracker.includes('id="fork-longer"'), 'Missing longer-term tracker section');
for (const [id, expected, obsolete] of [
  ['eip-7928', 'BAL header commitment encoded', 'block_access_list_hash header field not yet added'],
  ['eip-7805', 'Inclusion-list gossip validation', 'CL P2P gossip validation rules for ILs'],
  ['eip-8141', 'exact APPROVE scopes and atomic rollback', 'Payer gas-approval logic'],
  ['eip-7495', 'ssz/stable_container_test.go', 'No tests'],
]) {
  const page = readFileSync(new URL(`../out/eips/${id}/index.html`, import.meta.url), 'utf8');
  assert.ok(page.includes(expected), `Missing current evidence on ${id}`);
  assert.ok(!page.includes(obsolete), `Obsolete claim on ${id}`);
}
const framePage = readFileSync(new URL('../out/eips/eip-8141/index.html', import.meta.url), 'utf8');
for (const claim of ['scalar nonce', 'DEPLOY / VERIFY / SENDER', 'cross-frame refills', 'delayed withdrawals', 'synchronized EL', 'Dora production epoch/receipt']) {
  assert.ok(framePage.includes(claim), `Missing scoped Frame evidence: ${claim}`);
}
assert.doesNotMatch(framePage, /2D nonces|TXPARAMLOAD \/ TXPARAMSIZE \/ TXPARAMCOPY|core\/eips\/frame_execution/);
for (const path of ['pkg/bal/spec_test.go', 'pkg/epbs/validation_test.go', 'pkg/focil/il_satisfaction_test.go', 'pkg/node/frame_8141_integration_test.go', 'pkg/core/execution/frame_8141.go', recordPath, 'pkg/rollup/execute_test.go', 'pkg/crypto/pqc/pubkey_registry.go']) {
  assert.ok(existsSync(new URL(`../../${path}`, import.meta.url)), `Missing evidence: ${path}`);
}
function checkPages(dir) {
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const path = new URL(`${entry.name}${entry.isDirectory() ? '/' : ''}`, dir);
    if (entry.isDirectory()) checkPages(path);
    else if (entry.name.endsWith('.html')) {
      const html = readFileSync(path, 'utf8');
      assert.doesNotMatch(html, /github\.com\/jiayaoqijia\/eth2030|https:\/\/eth2030\.com/, `Outdated link in ${path.pathname}`);
    }
  }
}
checkPages(new URL('../out/', import.meta.url));
const pagesWorkflow = readFileSync(new URL('../../.github/workflows/deploy-pages.yml', import.meta.url), 'utf8');
assert.match(pagesWorkflow, /echo 'eth2030\.altllm\.ai' > out\/CNAME/);
console.log('Export claims, retained milestones, shared progress, evidence paths and tracker links verified.');
