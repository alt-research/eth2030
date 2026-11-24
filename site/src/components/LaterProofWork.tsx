import Link from 'next/link';
import { laterProofEntries, roadmapStatusNames } from '@/lib/roadmap';

export default function LaterProofWork() {
  return <section id="later-proof-work" aria-label="Later proof proposals" className="my-6 rounded-xl border border-eth-purple/30 bg-eth-surface/80 p-5">
    <p className="text-xs uppercase tracking-wider text-eth-purple">Later rounds · no official fork assignment</p>
    <h3 className="mt-2 text-lg font-bold text-eth-text">Optional execution proofs and canonical guest research</h3>
    <p className="mt-2 text-sm leading-7 text-eth-dim">EIP-8025 remains a Draft outside the official Glamsterdam inclusion list. This later work is retained as unfinished research and does not reduce Glamsterdam progress.</p>
    <ul className="mt-3 grid gap-3 sm:grid-cols-2">{laterProofEntries.map(entry => <li key={entry.id} data-later-proof={entry.id} className="min-w-0 rounded-lg border border-eth-purple/20 p-3 text-xs leading-6"><Link className="font-bold text-eth-teal underline" href={`/eips/${entry.id}/`}>{entry.number ? `EIP-${entry.number} · ` : ''}{entry.title} ↗</Link><p className="mt-1 text-eth-dim">{roadmapStatusNames[entry.status]} · project implementation label</p><p className="mt-1 text-eth-dim">{entry.number === 8025 ? 'Complete payload-bound proofs, proof-engine lifecycle, signed propagation and recovery under the adopted optional-proof contract.' : 'Execute actual Ethereum block transitions inside the proved guest, using authenticated state witnesses and independent root checks.'}</p></li>)}</ul>
  </section>;
}
