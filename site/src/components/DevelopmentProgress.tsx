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
    <div className="mt-4 border-t border-eth-teal/20 pt-4">
      <h4 className="text-sm font-bold text-eth-text">Remaining acceptance gates</h4>
      <ul className="mt-3 grid gap-4 md:grid-cols-2">
        {snapshot.gates.map(gate => <li key={gate.id} data-gate={gate.id} data-state={gate.state} className="text-xs leading-6">
          <p className="font-bold text-eth-text">{gate.title} <span className="ml-1 font-normal capitalize text-eth-teal">· {gate.state}</span></p>
          <p className="text-eth-dim">{gate.detail}</p>
          {'url' in gate && <a className="text-eth-teal underline" href={gate.url}>Current evidence ↗</a>}
        </li>)}
      </ul>
    </div>
    <div className="mt-4 border-t border-eth-teal/20 pt-4 text-xs leading-6">
      <p className="font-bold text-eth-teal">{snapshot.followUp.title}</p>
      <p className="mt-1 text-eth-dim">{snapshot.followUp.detail}</p>
      <div className="mt-2 flex flex-wrap gap-4 text-eth-teal">{snapshot.followUp.reviews.map(number => <a key={number} className="underline" href={`${GITHUB_URL}/pull/${number}`}>PR #{number} ↗</a>)}<a className="underline" href={`${GITHUB_URL}/commit/${snapshot.followUp.revision}`}>Reviewed source {snapshot.followUp.revision.slice(0, 8)} ↗</a></div>
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
