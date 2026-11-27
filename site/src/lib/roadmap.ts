import officialScope from '@/data/official-glamsterdam.json';
import { EIPS, type EipEntry, type ForkId, type Layer, type Status } from '@/data/eips';

export const hegotaGroupE = {
  reviewed: 'November 28, 2026',
  revision: '8076e37b7b80ca9c29d67120f4dbd853022bafd5',
  plan: 'docs/plans/hegota-e/PLAN.md',
  rows: [
    { id: 'HGT-040–042', title: 'Canonical Heze and Bogotá', status: 'Native implementation', detail: 'Merged EIP-7805 rebase, pinned Heze SSZ/domain and Bogotá versions. Authenticated branch storage, timely/late/equivocation views, requests and failure-atomic ingress remain qualified. Execution validity stays separate from inclusion satisfaction.' },
    { id: 'HGT-043–045', title: 'Builder and native node', status: 'Local qualification', detail: 'Builders execute transactions and enforce EIP-7934 RLP limits. Omission checks use transaction-end preconditions and independent gas reservations, ignore type 3 and never run the candidate. Version-3 caches reevaluate older views; the unmerged byte-budget exemption remains disabled.' },
    { id: 'HGT-046', title: 'AA-VOPS / Profile 2', status: 'Specification gate', detail: 'Informational EIP-8369 reference predicates are tested. Consensus activation stays disabled pending a Standards Track claims, state-diff, replay and ePBS specification.' },
    { id: 'HGT-047', title: 'Gossip to fork choice', status: 'Independent CL control passed', detail: 'Two pinned Lighthouse/native-EL pairs verified signed inclusion and common finality, real VALID/false omission, CL fork choice and restart. Malformed, nonce and actual full-capacity exemptions passed. Exhaustive request protocols and mixed-client release coverage remain HGT-070/080 work.' },
    { id: 'HGT-048', title: 'Post-quantum IL resources', status: 'Bounded network measurements', detail: 'Real verification and local SSZ/BLS checks are joined by 64 matched signed network samples: 8,183 transaction bytes per IL, within the unchanged 8 KiB limit. Docker-host carriage does not qualify PQ validity/replay. Omitted witness transport and public network tails remain unqualified.' },
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
