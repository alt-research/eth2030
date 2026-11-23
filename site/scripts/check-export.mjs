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
const current = readRecord('docs/reviews/remaining-20261123/CURRENT_STATE.json');
const bounded = readRecord('docs/reviews/diagnostic-threshold-20261123/COMPLETED_RUN.json');
const stopped = readRecord('docs/reviews/diagnostic-threshold-20261123/STOPPED_EXECUTION_BINDING.json');
assert.deepEqual(current.bounded_run, bounded, 'Current snapshot changed retained duration/BLS evidence');
assert.deepEqual(current.stopped_binding, stopped, 'Current snapshot changed stopped endpoint evidence');
assert.equal(current.selected_scope.task_ids_complete, tasks.tasks.length);
assert.equal(current.selected_scope.whole_public_fork_readiness, false);
assert.equal(current.original_fault_resolved, false);
assert.equal(current.actual48h_passed, false);
assert.equal(bounded.actual48h_passed, false);
assert.equal(bounded.original_fault_resolved, false);
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
assert.match(progress(home), /Frozen development scope delivered; readiness gates open/);
assert.ok(progress(home).includes(current.review_date));
const duration = Math.floor(bounded.runtime_seconds);
const durationLabel = `${Math.floor(duration / 3600)}h${Math.floor(duration % 3600 / 60)}m${String(duration % 60).padStart(2, '0')}s`;
for (const fact of [durationLabel, `${bounded.post_5_5h_samples} healthy samples`, `${bounded.post_5_5h_full_stack_captures} full stacks`, `${bounded.bls_windows_freshly_replayed} BLS windows`, `${bounded.bls_observed_through - bounded.bls_observed_first + 1} observed slots`, `${bounded.bls_missing_slots.length} early missing slots`, `slot ${stopped.stopped_cl_slot}`, `${stopped.extra_stopped_slots} slots beyond the last health sample`, `${bounded.sdk_live_independent_proofs} live + 2 stopped account proofs`]) {
  assert.ok(progress(home).includes(fact), `Missing current retained fact: ${fact}`);
}
assert.equal(current.integrated_ci.gate, 'closed');
assert.equal(current.integrated_ci.conclusion, 'success');
assert.ok(current.integrated_ci.jobs.length >= 26);
assert.ok(current.integrated_ci.jobs.every(job => job.conclusion === 'success'));
assert.equal(current.local_testnet.conclusion, 'success');
const localSkipped = current.local_testnet.jobs.filter(job => job.conclusion === 'skipped');
assert.equal(localSkipped.length, 1);
assert.ok(localSkipped.every(job => job.name.startsWith('checkpoint-sync-test-')));
assert.ok(current.local_testnet.jobs.filter(job => job.conclusion !== 'skipped').every(job => job.conclusion === 'success'));
const campaign = readRecord('docs/reviews/remaining-20261123/CURRENT_TUPLE.json');
assert.equal(campaign.operating_passed, true);
assert.equal(campaign.bls_passed, true);
assert.equal(campaign.actual_stopped_binding_passed, true);
assert.equal(campaign.actual48h_passed, false);
assert.equal(campaign.original_fault_resolved, false);
assert.equal(campaign.startup_through_stopped_bls.passed, true);
assert.equal(campaign.startup_through_stopped_bls.ws_checkpoint_root_agreed, true);
assert.equal(campaign.startup_through_stopped_bls.window.slots, 1012);
assert.equal(campaign.bls.signature_participants, 512);
assert.equal(campaign.live_account_proofs, 14);
assert.equal(campaign.stopped.account_proofs, 2);
assert.ok(campaign.ready_observed_seconds >= 1800);
assert.ok(Math.abs(campaign.ready_observed_seconds - (Date.parse(campaign.last_observed_at) - Date.parse(campaign.first_observed_at)) / 1000) < 0.002);
assert.equal(campaign.stopped.stopped_cl_slot - campaign.observed_through_slot, campaign.stopped.extra_stopped_slots);
assert.ok(progress(home).includes(`${campaign.observed_slots} observed slots`));
assert.ok(progress(home).includes(`slot ${campaign.stopped.stopped_cl_slot}`));
assert.ok(progress(home).includes('5547 passed, 9 skipped'));
const upstream = readRecord('docs/reviews/remaining-20261123/UPSTREAM_HOSTED_GATES.json');
assert.ok(upstream.prs.every(pr => pr.merged === false && pr.runs.some(run => run.conclusion === 'action_required')));
const scale = readRecord('docs/reviews/remaining-20261123/CAPACITY_RESULTS.json');
assert.equal(scale.results.length, 8);
assert.ok(scale.results.every(result => result.passed && result.budget_passed));
const repaired = readRecord('docs/reviews/remaining-20261123/POST_REPAIR_RECOVERY.json');
assert.equal(repaired.operating_passed, true);
assert.equal(repaired.bls_passed, true);
assert.equal(repaired.actual_stopped_binding_passed, true);
assert.equal(repaired.actual48h_passed, false);
assert.equal(repaired.original_fault_resolved, false);
assert.ok(repaired.ready_observed_seconds >= 300);
assert.ok(repaired.native_rpc_startup_seconds <= 300);
assert.ok(repaired.catchup_seconds <= 900);
assert.equal(repaired.full_snapshot_values_verified, 2707772);
assert.equal(repaired.metrics_gauge_passed, true);
assert.ok(repaired.metrics_observations > 0);
assert.equal(repaired.initial_300s_catchup_passed, false);
assert.equal(repaired.runtime_source, current.cumulative_runtime_source);
assert.ok(progress(home).includes('PR146'));
const elReviews = readRecord('docs/reviews/remaining-20261123/FINAL_EL_PR_STATUS.json');
assert.equal(elReviews.all_current_code_heads_ci_passed, true);
assert.equal(elReviews.prs.length, 5);
assert.ok(elReviews.prs.every(pr => pr.base === 'master' && pr.draft === false && pr.ci_conclusion === 'success'));
const mergeAudit = readRecord('docs/reviews/remaining-20261123/FINAL_LAYOUT_PR_MERGE_AUDIT.json');
assert.equal(mergeAudit.all_clean, true);
assert.equal(mergeAudit.subset_extensions_checked, 80);
assert.equal(mergeAudit.current_package_equal_to_measured_runtime, true);
for (const pr of elReviews.prs) assert.equal(mergeAudit.heads[pr.number], pr.head);
assert.equal(mergeAudit.pkg_tree, repaired.runtime_pkg_tree);
const failedJobs = current.prior_integrated_ci.jobs.filter(job => job.conclusion === 'failure');
for (const name of ['check-code', 'http-api-tests', 'beacon-chain-tests', 'basic-simulator-ubuntu']) {
  assert.ok(failedJobs.some(job => job.name === name), `Missing actual failing job ${name}`);
}
for (const [id, state] of [['G-01', current.integrated_ci.gate], ['G-02', 'unconfirmed'], ['G-03', 'open'], ['G-04', 'open'], ['G-05', 'reserved'], ['G-06', 'conditional']]) {
  assert.ok(progress(home).includes(`data-gate="${id}" data-state="${state}"`), `Missing actual acceptance gate ${id}/${state}`);
}
assert.ok(progress(home).includes(current.integrated_ci.url));
assert.match(progress(home), /original 3s failure/);
assert.match(progress(home), /repaired baseline completed 27 hosted jobs/);
assert.match(progress(home), /Rounds 03\/04 remain failed/);
assert.match(progress(home), /exact holder and physical I\/O cause remain unconfirmed/);
assert.match(progress(home), /not a 48-hour pass/);
assert.match(progress(home), /whole-public-fork or production readiness/);
assert.match(progress(home), /research features retain separate acceptance gates/);
assert.doesNotMatch(progress(home), /48-hour (?:passed|complete)|original fault (?:resolved|fixed)|all (?:hosted )?CI (?:passes|green)/i);
for (const number of [135, 138, 139, 140]) assert.ok(progress(home).includes(`/pull/${number}`));
assert.ok(progress(home).includes(`/pull/${current.merged_el.through_pr}`));
assert.ok(progress(home).includes(`/commit/${current.merged_cl.revision}`));
assert.ok(progress(home).includes(`/blob/${current.merged_el.revision}/docs/reviews/diagnostic-threshold-20261123/COMPLETED_RUN.json`));
assert.ok(progress(home).includes(`/blob/${merged.merged_master}/${recordPath}`), 'Missing pinned earlier verification record');
for (const path of ['docs/reviews/remaining-20261123/CURRENT_STATE.json', 'docs/plans/glamsterdam/NEXT_STEPS_20261123.md']) {
  assert.ok(progress(home).includes(`/blob/1317a6624e6246a8e0937f56d2758b300ccbcfcb/${path}`), `Missing pinned current evidence/plan ${path}`);
}
const homepageRoadmap = home.split('id="roadmap"')[1]?.split('</section>')[0];
assert.ok(homepageRoadmap, 'Homepage roadmap section is missing');
assert.match(home, /November 2, 2026/);
assert.match(home, /Signed Gloas route qualified/);
assert.match(home, /PR 127 retires unsupported legacy Builder Engine extensions/);
assert.match(home, /source-specific evidence/);
assert.doesNotMatch(home, /Review source[^<]*awaiting merge/);
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
  assert.ok(page.includes(`/blob/${number === 7732 ? current.historical_el.revision : merged.merged_master}/docs/reviews/glamsterdam-postmerge-20261121/STATUS.md`));
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
assert.match(builderPage, /legacy EL bid-hash\/registry\/header extensions retired in PR127/);
assert.ok(builderPage.includes(`/blob/${current.historical_el.revision}/docs/reviews/builder-authority/STATUS.md`));
assert.doesNotMatch(builderPage, /Legacy EL builder extensions accept unsigned registrations/);
assert.match(builderPage, /passes all 27 test-suite jobs and local-testnet/);
assert.ok(builderPage.includes(`/blob/${current.historical_el.revision}/docs/reviews/diagnostic-threshold-20261123/STOPPED_EXECUTION_BINDING.json`));
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
const siteWorkflow = readFileSync(new URL('../../.github/workflows/site-check.yml', import.meta.url), 'utf8');
for (const input of ['docs/reviews/remaining-*/**', 'docs/reviews/progress-*/**', 'docs/reviews/diagnostic-threshold-*/**', 'docs/plans/glamsterdam/**']) {
  assert.ok(siteWorkflow.includes(`'${input}'`), `Site CI omits new evidence input ${input}`);
}
const pagesWorkflow = readFileSync(new URL('../../.github/workflows/deploy-pages.yml', import.meta.url), 'utf8');
assert.match(pagesWorkflow, /echo 'eth2030\.altllm\.ai' > out\/CNAME/);
console.log('Export claims, retained milestones, shared progress, evidence paths and tracker links verified.');
