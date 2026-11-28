import { hegotaGroupE } from '@/lib/roadmap';
import { GITHUB_URL } from '@/lib/constants';

export default function HegotaProgress() {
  return <aside id="hegota-group-e" data-hegota-group-e="native-review" aria-label="Hegotá Group E implementation and activation gates" className="my-8 rounded-xl border border-eth-teal/30 bg-eth-surface/80 p-5 md:p-6">
    <p className="text-xs uppercase tracking-wider text-eth-teal">Hegotá · Group E · {hegotaGroupE.reviewed}</p>
    <h3 className="mt-2 text-lg font-bold text-eth-text">FOCIL implementation and activation gates</h3>
    <p className="mt-3 text-sm leading-6 text-eth-dim">Nine tasks, 36 reviewed criteria, 44 required native consumers. Native qualification and an independent two-pair Heze control passed. Unresolved protocol activation and wider release coverage remain explicit. This update does not change the official fork scope or claim public-network readiness.</p>
    <ul className="mt-5 grid gap-5 md:grid-cols-2">{hegotaGroupE.rows.map(row => <li key={row.id} data-hegota-task={row.id}>
      <p className="text-xs text-eth-teal">{row.id} · {row.status}</p>
      <h4 className="mt-1 text-sm font-bold text-eth-text">{row.title}</h4>
      <p className="mt-2 text-xs leading-6 text-eth-dim">{row.detail}</p>
    </li>)}</ul>
    <div className="mt-5 flex flex-wrap gap-4 text-xs text-eth-teal">
      <a className="underline" href={`${GITHUB_URL}/blob/${hegotaGroupE.revision}/${hegotaGroupE.plan}`}>Criterion register ↗</a>
      <a className="underline" href={`${GITHUB_URL}/blob/${hegotaGroupE.revision}/pkg/node/backend/engine_bogota.go`}>Reviewed native source ↗</a>
      <a className="underline" href={`${GITHUB_URL}/blob/${hegotaGroupE.revision}/evidence/hegota-e/external-network.json`}>Independent network evidence ↗</a>
      <a className="underline" href={`${GITHUB_URL}/blob/${hegotaGroupE.revision}/docs/reviews/pr-172/LATEST_2718.md`}>Latest EIP-7805 audit ↗</a>
      <a className="underline" href="https://eips.ethereum.org/EIPS/eip-8369">AA-VOPS specification boundary ↗</a>
    </div>
  </aside>;
}
