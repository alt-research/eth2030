import Link from 'next/link';
import remaining from '@/data/remaining-work.json';
import { EIPS } from '@/data/eips';
import { GITHUB_URL } from '@/lib/constants';
import { roadmapStages, roadmapProgress, roadmapStatusNames } from '@/lib/roadmap';

export default function RemainingRoadmapWork() {
  const inventory = roadmapStages.map(stage => ({ stage, pending: roadmapProgress(stage.id).entries.filter(entry => entry.status !== 'complete') }));
  const labelledPending = inventory.reduce((total, phase) => total + phase.pending.length, 0);
  return <section id="remaining-work" aria-labelledby="remaining-work-heading" className="mb-8 rounded-xl border border-eth-teal/30 bg-eth-surface/80 p-5 md:p-6">
    <p className="text-xs uppercase tracking-wider text-eth-teal">Remaining work · {remaining.date}</p>
    <h2 id="remaining-work-heading" className="mt-2 text-xl font-bold text-eth-text">What still needs work</h2>
    <p data-remaining-adopted-todos={remaining.adoptedDevelopmentTodos} className="mt-3 text-sm leading-7 text-eth-text/80">Adopted Glamsterdam development: {remaining.adoptedDevelopmentTodos} confirmed open implementation TODOs. PR #{remaining.mergedProgressPr} is merged into master.</p>
    <p className="mt-2 text-xs leading-6 text-eth-dim">{remaining.policy}</p>
    <div className="mt-5 grid gap-3 md:grid-cols-2" aria-label="Later-scope workstreams">
      {remaining.items.map(item => <details key={item.id} data-remaining-work={item.id} className="min-w-0 rounded-lg border border-eth-teal/20 p-4">
        <summary className="cursor-pointer text-sm font-bold leading-6 text-eth-text"><span className="text-xs font-normal text-eth-teal">{item.kind} · </span>{item.title}<span className="mt-2 block text-xs font-normal leading-6 text-eth-dim">{item.remaining}</span></summary>
        <p className="mt-3 text-xs leading-6 text-eth-text/80"><strong>Current evidence:</strong> {item.current}</p>
        <p className="mt-2 text-xs leading-6 text-eth-text/80"><strong>Done when:</strong> {item.doneWhen}</p>
        <div className="mt-3 flex flex-wrap gap-x-3 gap-y-2 text-xs text-eth-teal">{item.entries.map(id => <Link className="underline" key={id} href={`/eips/${id}/`}>{EIPS.find(entry => entry.id === id)?.title ?? id} ↗</Link>)}</div>
        <ul className="mt-3 text-xs leading-6 text-eth-teal">{item.evidence.map(path => <li key={path}><a className="break-all underline" href={`${GITHUB_URL}/blob/${remaining.reviewedMaster}/${path}`}>{path} ↗</a></li>)}</ul>
      </details>)}
    </div>
    <div className="mt-5 border-t border-eth-teal/20 pt-4">
      <h3 className="text-sm font-bold text-eth-text">Planned-only tracker entries</h3>
      <p className="mt-2 text-xs leading-6 text-eth-dim">These four retained roadmap entries list no implementation or test paths. Their protocol scope must be adopted before implementation; they are outside the completed Glamsterdam task ledger.</p>
      <ul className="mt-2 grid gap-2 text-xs leading-6 text-eth-teal sm:grid-cols-2">{remaining.plannedOnly.map(id => <li key={id} data-planned-work={id}><Link className="underline" href={`/eips/${id}/`}>{EIPS.find(entry => entry.id === id)?.title ?? id} ↗</Link></li>)}</ul>
    </div>
    <div className="mt-5 border-t border-eth-teal/20 pt-4">
      <h3 className="text-sm font-bold text-eth-text">Historical non-complete labels: {labelledPending} entries</h3>
      <p className="mt-2 text-xs leading-6 text-eth-dim">This is an inventory of old labels, not {labelledPending} newly confirmed missing features. In particular, the broad ePBS label does not reopen the delivered signed Gloas route. Expand a phase to inspect every non-complete entry.</p>
      <div className="mt-3 grid gap-3 sm:grid-cols-2 lg:grid-cols-3">{inventory.map(({ stage, pending }) => <details key={stage.id} data-pending-phase={stage.id} data-pending-count={pending.length} className="min-w-0 rounded-lg border border-eth-teal/20 p-3 text-xs leading-6">
        <summary className="cursor-pointer font-bold text-eth-text">{stage.name} · {pending.length} entries</summary>
        <ul className="mt-2 space-y-2">{pending.map(entry => <li key={entry.id} data-pending-entry={entry.id}><Link className="text-eth-teal underline" href={`/eips/${entry.id}/`}>{entry.title}</Link><span className="ml-2 text-eth-dim">{roadmapStatusNames[entry.status]}</span></li>)}</ul>
      </details>)}</div>
    </div>
    <p className="mt-5 text-xs leading-6 text-eth-dim">48h/larger-scale admission and upstream Lodestar approval remain excluded from development progress. Original 04 attribution remains unconfirmed. Public activation is a separate network rollout decision; no activation parameter is changed here.</p>
  </section>;
}
