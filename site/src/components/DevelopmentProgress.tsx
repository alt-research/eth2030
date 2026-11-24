import { developmentSnapshot } from '@/lib/data';
import { GITHUB_URL } from '@/lib/constants';

export default function DevelopmentProgress() {
  const snapshot = developmentSnapshot;
  return <aside aria-label="Verified development progress" className="mb-8 rounded-xl border border-eth-teal/30 bg-eth-surface/80 p-5 md:p-6">
    <div className="flex flex-wrap items-center justify-between gap-3">
      <div>
        <p className="text-xs uppercase tracking-wider text-eth-teal">Merged development progress · {snapshot.date}</p>
        <h3 className="mt-2 text-lg font-bold text-eth-text">{snapshot.title}</h3>
      </div>
      <a href={`${GITHUB_URL}/pull/${snapshot.pullRequest}`} className="text-sm text-eth-teal underline">Merged PR #{snapshot.pullRequest} ↗</a>
    </div>
    <div className="mt-5 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
      {snapshot.milestones.map(item => <div key={item.title}>
        <p className="text-xs text-eth-dim">{item.title}</p>
        <p className="mt-1 text-base font-bold text-eth-teal">{item.value}</p>
        <p className="mt-2 text-xs leading-6 text-eth-text/80">{item.detail}</p>
      </div>)}
    </div>
    <p className="mt-5 text-xs leading-6 text-eth-dim">{snapshot.boundary}</p>
    <p className="mt-3 text-xs leading-6 text-eth-text/80">{snapshot.currentCampaign.detail} <a className="text-eth-teal underline" href={`${GITHUB_URL}/blob/${snapshot.reviewRevision}/${snapshot.currentCampaign.record}`}>Current tuple evidence ↗</a></p>
    <div className="mt-3 flex flex-wrap gap-4 text-xs text-eth-teal">{snapshot.currentCampaign.reviews.map(number => <a key={number} className="underline" href={`${GITHUB_URL}/pull/${number}`}>Merged EL PR #{number} ↗</a>)}</div>
    <div className="mt-4 border-t border-eth-teal/20 pt-4">
      <h4 className="text-sm font-bold text-eth-text">Development checks</h4>
      <p data-development-complete="true" className="mt-2 text-xs leading-6 text-eth-text/80">{snapshot.development.detail}</p>
      <p data-active-development-todos={snapshot.development.activeTodos} className="text-xs leading-6 text-eth-dim">Active development TODOs: {snapshot.development.activeTodos}</p>
      <ul className="mt-3 grid gap-4 md:grid-cols-2">
        {snapshot.gates.map(gate => <li key={gate.id} data-gate={gate.id} data-state={gate.state} className="text-xs leading-6">
          <p className="font-bold text-eth-text">{gate.title} <span className="ml-1 font-normal capitalize text-eth-teal">· {gate.state}</span></p>
          <p className="text-eth-dim">{gate.detail}</p>
          {'url' in gate && <a className="text-eth-teal underline" href={gate.url}>Current evidence ↗</a>}
        </li>)}
      </ul>
    </div>
    <div role="region" aria-label="Deferred acceptance outside development progress" className="mt-4 border-t border-eth-teal/20 pt-4 text-xs leading-6">
      <h4 className="font-bold text-eth-text">Deferred acceptance · excluded from development progress</h4>
      <ul className="mt-3 grid gap-4 md:grid-cols-2">{snapshot.deferredAcceptance.map(item => <li key={item.id} data-deferred={item.id} data-counted-in-progress="false"><p className="font-bold text-eth-text">{item.title}</p><p className="text-eth-dim">{item.detail}</p>{'url' in item && <a className="text-eth-teal underline" href={item.url}>Upstream record ↗</a>}</li>)}</ul>
    </div>
    <div role="region" aria-label="Separate public-network rollout decision" data-rollout-counted-in-progress="false" className="mt-4 border-t border-eth-teal/20 pt-4 text-xs leading-6"><h4 className="font-bold text-eth-text">{snapshot.rollout.title}</h4><p className="text-eth-dim">{snapshot.rollout.detail}</p></div>
    <div className="mt-4 border-t border-eth-teal/20 pt-4 text-xs leading-6">
      <p className="font-bold text-eth-teal">{snapshot.followUp.title}</p>
      <p className="mt-1 text-eth-dim">{snapshot.followUp.detail}</p>
      <div className="mt-2 flex flex-wrap gap-4 text-eth-teal">{snapshot.followUp.reviews.map(number => <a key={number} className="underline" href={`${GITHUB_URL}/pull/${number}`}>PR #{number} ↗</a>)}<a className="underline" href={`${GITHUB_URL}/commit/${snapshot.followUp.revision}`}>Reviewed source {snapshot.followUp.revision.slice(0, 8)} ↗</a></div>
    </div>
    <div data-extension-progress="reviewed" className="mt-4 border-t border-eth-teal/20 pt-4 text-xs leading-6">
      <h4 className="font-bold text-eth-text">FCR and real-proof extension reviews</h4>
      <p className="mt-2 text-eth-dim">Signed CL FCR feeds native Engine safeBlockHash; durable announcements and mixed-root vote isolation are implemented. Real RV32IM receipts bind the complete execution statement and reach native asynchronous prove/verify RPC. EL #155/#156 await master integration; Lighthouse #17 is merged into eth2030 after all 27 suite checks and local-testnet.</p>
      <p className="mt-2 text-eth-dim">Glamsterdam: 16/17 mapped entries complete for their stated development scope. The remaining canonical quest requires an executable Ethereum STF guest; the real program proof and fixed-output STF adapter do not establish that guest.</p>
      <div className="mt-2 flex flex-wrap gap-4 text-eth-teal"><a className="underline" href={`${GITHUB_URL}/pull/155`}>FCR EL #155 ↗</a><a className="underline" href={`${GITHUB_URL}/pull/156`}>Real proof EL #156 ↗</a><a className="underline" href="https://github.com/jiayaoqijia/lighthouse/pull/17">Durable CL FCR #17 ↗</a></div>
    </div>
    <p className="mt-4 text-xs leading-6 text-eth-dim">Earlier source-specific controls: {snapshot.historicalMilestones.join(' · ')}. These retain their original runtime records.</p>
    <div className="mt-3 flex flex-wrap gap-4 text-xs text-eth-teal">
      <a className="underline" href={`${GITHUB_URL}/blob/${snapshot.reviewRevision}/${snapshot.stateRecord}`}>Reviewed state ↗</a>
      <a className="underline" href={`${GITHUB_URL}/blob/${snapshot.revision}/${snapshot.latestRecord}`}>Bounded run record ↗</a>
      <a className="underline" href={snapshot.companion.url}>Merged CL {snapshot.companion.revision.slice(0, 8)} ↗</a>
      <a className="underline" href={`${GITHUB_URL}/blob/${snapshot.recordRevision}/${snapshot.record}`}>Earlier verification record ↗</a>
      <a className="underline" href={`${GITHUB_URL}/blob/${snapshot.reviewRevision}/${snapshot.plan}`}>Next steps and source TODOs ↗</a>
    </div>
  </aside>;
}
