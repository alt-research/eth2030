package backend

import "github.com/eth2030/eth2030/zkvm/canonicalproof"

// CanonicalProofService returns the node-owned service without widening every
// legacy backend/mock interface. Request contexts flow through the RPC API.
func (b *RPCBackend) CanonicalProofService() *canonicalproof.Service {
	provider, ok := b.node.(interface {
		CanonicalProofService() *canonicalproof.Service
	})
	if !ok {
		return nil
	}
	return provider.CanonicalProofService()
}
