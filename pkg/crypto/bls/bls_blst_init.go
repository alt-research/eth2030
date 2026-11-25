//go:build blst

// bls_blst_init.go wires BlstRealBackend as the default BLS backend when
// the binary is compiled with -tags blst (GAP-7.2). Without this build tag,
// ConsensusBLSBackend (gnark) is the default without CGO.
package bls

func init() {
	SetBLSBackend(&BlstRealBackend{})
}
