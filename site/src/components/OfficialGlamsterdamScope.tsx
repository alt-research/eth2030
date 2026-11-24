import Link from 'next/link';
import scope from '@/data/official-glamsterdam.json';
import { EIPS } from '@/data/eips';
import { roadmapProgress } from '@/lib/roadmap';

export default function OfficialGlamsterdamScope() {
  const { entries, counts } = roadmapProgress('glamsterdam');
  const pending = entries.filter(entry => entry.status !== 'complete');
  return <section data-official-glamsterdam="true" data-official-complete={counts.complete} data-official-total={entries.length} aria-label="Official Glamsterdam scope" className="mb-6 rounded-xl border border-eth-teal/30 bg-eth-surface/80 p-5 text-sm leading-7">
    <p className="text-xs uppercase tracking-wider text-eth-teal">Official upgrade scope · {scope.reviewed}</p>
    <h3 className="mt-2 text-lg font-bold text-eth-text">{`Glamsterdam · ${counts.complete}/${entries.length} scheduled core EIPs labelled Complete`}</h3>
    <p className="mt-2 text-eth-dim">Scope: <a className="text-eth-teal underline" href={scope.source}>EIP-7773 ↗</a> · official meta status: {scope.officialStatus}. Completion counts ETH2030 implementation labels, not Ethereum network activation or a new conformance measurement.</p>
    <p className="mt-2 text-eth-dim">Open core label: {pending.map(entry => <Link key={entry.id} className="text-eth-teal underline" href={`/eips/${entry.id}/`}>EIP-{entry.number} · {entry.title} ({entry.status}) ↗</Link>)}</p>
    <div className="mt-3 border-t border-eth-teal/20 pt-3">
      <p className="font-bold text-eth-text">Networking references · separate from the core denominator</p>
      <div className="mt-1 flex flex-wrap gap-x-4 gap-y-1">{scope.networking.map(number => { const entry = EIPS.find(item => item.number === number); return <Link key={number} data-official-network-eip={number} className="text-xs text-eth-teal underline" href={`/eips/eip-${number}/`}>EIP-{number} · {entry?.title} ↗</Link>; })}</div>
      <p className="mt-3 font-bold text-eth-text">Informational references · not implementation requirements</p>
      <div className="mt-1 flex flex-wrap gap-x-4 gap-y-1">{scope.informational.map(item => <a key={item.number} data-official-info-eip={item.number} className="text-xs text-eth-teal underline" href={`https://eips.ethereum.org/EIPS/eip-${item.number}`}>EIP-{item.number} · {item.title} ↗</a>)}</div>
    </div>
    <p className="mt-3 text-xs text-eth-dim">EIP-8025 and canonical Ethereum guest research are later proposals, excluded from Glamsterdam progress. Other fork cards retain project planning labels. <a className="text-eth-teal underline" href={scope.sourceSnapshot}>Pinned official source ↗</a></p>
  </section>;
}
