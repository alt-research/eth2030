import { readFileSync, existsSync, readdirSync } from 'node:fs';
import { strict as assert } from 'node:assert';

const home = readFileSync(new URL('../out/index.html', import.meta.url), 'utf8');
const pq = readFileSync(new URL('../out/pq-roadmap/index.html', import.meta.url), 'utf8');
const tracker = readFileSync(new URL('../out/eips/index.html', import.meta.url), 'utf8');
const roadmap = readFileSync(new URL('../out/roadmap/index.html', import.meta.url), 'utf8');
// Published measurements are checked against retained source-specific inputs.
const recordPath = 'docs/reviews/glamsterdam-postmerge-20261121/RESULTS.json';
const readRecord = path => JSON.parse(readFileSync(new URL(`../../${path}`, import.meta.url), 'utf8'));
const record = readRecord(recordPath);
const tasks = readRecord('docs/reviews/glamsterdam-final-campaign/TASK_RECONCILIATION.json');
const recovery = readRecord('docs/reviews/glamsterdam-postmerge-20261121/RECOVERY.json');
const reproduced = readRecord('docs/reviews/glamsterdam-postmerge-20261121/PUBLIC_REPRODUCTION.json');
const merged = readRecord('docs/audits/merged-glamsterdam-20261122/MERGED_EVIDENCE.json');
const progress = html => html.match(/<aside aria-label="Verified development progress"[^>]*>(.*?)<\/aside>/s)?.[1];
assert.ok(progress(home), 'Missing homepage development progress');
assert.equal(progress(home), progress(roadmap), 'Homepage/detail development snapshots diverged');
assert.equal(tasks.tasks.length, 33);
assert.equal(tasks.remaining_engineering_campaign.length, 0);
assert.ok(tasks.tasks.every(task => task.engineering_status === 'complete for frozen development target'));
assert.equal(record.qualified_source, merged.qualified_composition);
assert.equal(merged.package_and_code_ci_tree_equal, true);
assert.equal(Object.keys(merged.merged_prs).length, 3);
assert.equal(merged.upstream_lodestar.merged, false);
assert.equal(recovery.passed, true);
assert.equal(Object.keys(recovery.roles).length, 4);
assert.ok(Object.values(recovery.roles).every(role => role.accepted_elapsed_from_container_start_seconds < 300));
const steady = record.operating['steady-reviewed-02'];
const pressure = record.operating['pressure-reviewed-02'];
for (const [phase, slots] of [[steady, 400], [pressure, 60]]) {
  assert.equal(phase.latency.latency_passed, true);
  assert.equal(phase.finality.passed, true);
  assert.equal(phase.offline.passed, true);
  assert.equal(phase.finality.beacon_window.slots, slots);
  assert.equal(phase.finality.beacon_window.missing_fraction, 0);
}
assert.equal(record.blob.cell_proofs, 544);
assert.equal(record.blob.finality.passed, true);
assert.equal(record.blob.finality_offline.passed, true);
assert.equal(reproduced.passed, true);
assert.equal(reproduced.complete_manifest_files_verified, 245);
assert.equal(reproduced.checks.length, 11);
assert.ok(reproduced.checks.every(check => check.passed));
for (const milestone of [`${tasks.tasks.length}/33 task IDs`, '4 roles under 300s', `${steady.finality.beacon_window.slots}/400 + ${pressure.finality.beacon_window.slots}/60 slots`, `${record.blob.cell_proofs} KZG cell proofs`]) {
  assert.ok(progress(home).includes(milestone), `Missing qualified milestone: ${milestone}`);
}
assert.match(progress(home), /Glamsterdam development target merged and qualified/);
assert.match(progress(home), /November 22, 2026/);
assert.match(progress(home), /upstream Lodestar integration and production-scale operation remain separate/);
assert.match(progress(home), /Frame\/FOCIL and service migration retain their own later-fork acceptance gates/);
assert.ok(progress(home).includes('https://github.com/alt-research/eth2030/pull/125'));
assert.ok(progress(home).includes(`/blob/${merged.merged_master}/${recordPath}`), 'Missing pinned verification record');
assert.doesNotMatch(progress(home), /Native Frame execution and CL history qualified|Independent EL interoperability, Dora consumer migration and state-sync acceptance remain next steps/);
const homepageRoadmap = home.split('id="roadmap"')[1]?.split('</section>')[0];
assert.ok(homepageRoadmap, 'Homepage roadmap section is missing');
assert.match(home, /November 2, 2026/);
assert.match(home, /Signed Gloas route qualified/);
assert.match(home, /Legacy EL builder-registration\/bid extensions still lack signature authentication/);
assert.match(home, /not production-ready/i);
assert.match(home, /github\.com\/alt-research\/eth2030/);
assert.match(home, /Go 1\.26\.8\+/);
assert.match(home, /Independent Geth full\/snap late join/);
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

for (const number of [2780, 7688, 7708, 7732, 7778, 7843, 7928, 7954, 7976, 7981, 7997, 8024, 8037, 8038, 8045, 8061, 8246, 8282, 7975, 8159, 8070, 8189, 8136]) {
  const page = readFileSync(new URL(`../out/eips/eip-${number}/index.html`, import.meta.url), 'utf8');
  assert.match(page, /Qualification evidence/);
  assert.ok(page.includes(`/blob/${merged.merged_master}/docs/reviews/glamsterdam-postmerge-20261121/STATUS.md`));
}
for (const number of [7688, 8045, 8061, 8136]) {
  const page = readFileSync(new URL(`../out/eips/eip-${number}/index.html`, import.meta.url), 'utf8');
  assert.match(page, /Implemented in the companion consensus-client fork/);
  assert.doesNotMatch(page, /No implementation yet/);
}
const refundPage = readFileSync(new URL('../out/eips/eip-7778/index.html', import.meta.url), 'utf8');
assert.match(refundPage, /Block Gas Accounting without Refunds/);
assert.match(refundPage, /User refunds remain applied/);
assert.doesNotMatch(refundPage, /Remove SSTORE Gas Refunds/);
const builderPage = readFileSync(new URL('../out/eips/eip-7732/index.html', import.meta.url), 'utf8');
assert.match(builderPage, /Legacy EL builder extensions accept unsigned registrations/);
const authPage = readFileSync(new URL('../out/eips/eip-7702/index.html', import.meta.url), 'utf8');
assert.doesNotMatch(authPage, /auth.chain_id validation|s ≤ secp256k1n/);

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
