import officialScope from '@/data/official-glamsterdam.json';
import { EIPS, type EipEntry, type ForkId, type Layer, type Status } from '@/data/eips';

export const hegotaGroupE = {
  reviewed: 'November 27, 2026',
  revision: 'f328d938',
  plan: 'docs/plans/hegota-e/PLAN.md',
  rows: [
    { id: 'HGT-040–042', title: 'Canonical Heze and Bogotá', status: 'Native implementation', detail: 'Pinned SSZ/domain, authenticated branch-keyed storage, timely/late/equivocation views, retained requests and strict Engine versions/capabilities.' },
    { id: 'HGT-043–045', title: 'Builder and native node', status: 'Local qualification', detail: 'Nonce/funding cascades, cancellation and deadlines, exact RLP capacity, Profile 1 account checks, two isolated native nodes and restart persistence.' },
    { id: 'HGT-046', title: 'AA-VOPS / Profile 2', status: 'Specification gate', detail: 'Informational EIP-8369 reference predicates are tested. Consensus activation stays disabled pending a Standards Track claims, state-diff, replay and ePBS specification.' },
    { id: 'HGT-047', title: 'Gossip to fork choice', status: 'Native adapter qualified', detail: 'Authenticated IL ingress reaches Engine verdicts and Heze extension predicates. External beacon Gossipsub/reqresp, fork digests and CL/ePBS devnet qualification remain HGT-070/080 prerequisites.' },
    { id: 'HGT-048', title: 'Post-quantum IL resources', status: 'Bounded measurements', detail: 'Canonical direct-8504 bytes, real uncached verification and local signed SSZ/BLS/store pipeline. Omitted witness transport and network tail latency remain unqualified; limits are unchanged.' },
  ],
};

export const roadmapStages: { id: ForkId; name: string; glyph: string; heading: string; description: string }[] = [
  { id: 'glamsterdam', name: 'Glamsterdam', glyph: 'G*', heading: 'Scale the execution layer', description: 'Scope follows the official EIP-7773 scheduled core list. ETH2030 implementation labels are counted against that list; networking, informational references and later proof proposals are separate.' },
  { id: 'hegota', name: 'Hegotá', glyph: 'H*', heading: 'Open the protocol', description: 'Inclusion lists, quick slots and programmable transactions.' },
  { id: 'i', name: 'I*', glyph: 'I*', heading: 'The next foundations', description: 'Decoupled consensus, binary state and post-quantum research.' },
  { id: 'j', name: 'J*', glyph: 'J*', heading: 'Towards post-quantum', description: 'Attestations, sampling and proofs for a new security model.' },
  { id: 'k', name: 'K*', glyph: 'K*', heading: 'Proofs and state', description: 'Mandatory proofs and a path towards decentralized state.' },
  { id: 'l', name: 'L*', glyph: 'L*', heading: 'The longer horizon', description: 'One-round finality, native rollups and protocol simplification.' },
];

export const roadmapStatuses: Status[] = ['complete', 'substantial', 'functional', 'partial', 'referenced', 'planned'];
export const roadmapStatusNames: Record<Status, string> = { complete: 'Complete', substantial: 'Substantial', functional: 'Functional', partial: 'Partial', referenced: 'Referenced', planned: 'Planned' };

// These positions follow the EF drawing where the older tracker differs.
export function roadmapFork(entry: EipEntry): ForkId {
  if (entry.id === 'feature-leanvm-pq-proofs') return 'i';
  if (entry.id === 'feature-ethp2p-unification') return 'l';
  return entry.fork;
}

export function roadmapLayer(entry: EipEntry): Layer {
  if (['feature-fast-payload-broadcast', 'feature-ethp2p-unification', 'eip-8077'].includes(entry.id)) return 'EL';
  return entry.layer;
}

export function roadmapProgress(fork: ForkId) {
  const entries = fork === 'glamsterdam'
    ? EIPS.filter(entry => entry.number !== null && officialScope.scheduledCore.includes(entry.number))
    : EIPS.filter(entry => roadmapFork(entry) === fork && entry.strawmap);
  const counts = Object.fromEntries(roadmapStatuses.map(status => [status, entries.filter(entry => entry.status === status).length])) as Record<Status, number>;
  const completePercentage = entries.length ? Math.round((counts.complete / entries.length) * 100) : null;
  return { entries, counts, completePercentage };
}

export function groupRoadmapNodes(entries: EipEntry[]) {
  const groups = new Map<string, EipEntry[]>();
  for (const entry of entries) {
    const key = `${roadmapLayer(entry)}:${entry.strawmap?.item ?? entry.id}`;
    groups.set(key, [...(groups.get(key) ?? []), entry]);
  }
  return [...groups.values()];
}

export const laterProofEntries = EIPS.filter(entry => officialScope.laterProofEntries.includes(entry.id));
