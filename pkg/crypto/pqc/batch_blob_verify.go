package pqc

import "errors"

// Batch verification errors.
var (
	ErrBatchVerifyEmpty    = errors.New("batch_verify: empty batch")
	ErrBatchVerifyMismatch = errors.New("batch_verify: length mismatch")
	ErrBatchVerifyNilEntry = errors.New("batch_verify: nil entry in batch")
)

// BatchVerifyResult contains the result of a batch verification.
type BatchVerifyResult struct {
	// Valid is true if the entire batch verified successfully.
	Valid bool

	// FailedIndices lists the indices of failed commitments (only populated
	// when Valid is false and failure isolation was performed).
	FailedIndices []int
}

// LatticeBatchVerifier performs batch verification of lattice commitments.
type LatticeBatchVerifier struct {
	scheme *LatticeCommitScheme
}

// NewLatticeBatchVerifier creates a batch verifier for the given scheme.
func NewLatticeBatchVerifier(scheme *LatticeCommitScheme) *LatticeBatchVerifier {
	return &LatticeBatchVerifier{scheme: scheme}
}

// BatchVerify verifies each complete opening against its commitment and blob.
func (bv *LatticeBatchVerifier) BatchVerify(
	commitments []*LatticeCommitment,
	openings []*LatticeOpening,
	blobs [][]byte,
) (bool, error) {
	if len(commitments) == 0 {
		return false, ErrBatchVerifyEmpty
	}
	if len(commitments) != len(openings) || len(commitments) != len(blobs) {
		return false, ErrBatchVerifyMismatch
	}
	for i := range commitments {
		if commitments[i] == nil || openings[i] == nil || blobs[i] == nil {
			return false, ErrBatchVerifyNilEntry
		}
	}
	for i := range commitments {
		if !bv.scheme.Verify(commitments[i], openings[i], blobs[i]) {
			return false, nil
		}
	}
	return true, nil
}

// BatchVerifyWithIsolation verifies a batch and, on failure, identifies
// which specific commitments are invalid.
func (bv *LatticeBatchVerifier) BatchVerifyWithIsolation(
	commitments []*LatticeCommitment,
	openings []*LatticeOpening,
	blobs [][]byte,
) (*BatchVerifyResult, error) {
	valid, err := bv.BatchVerify(commitments, openings, blobs)
	if err != nil {
		return nil, err
	}

	result := &BatchVerifyResult{Valid: valid}
	if valid {
		return result, nil
	}

	// Batch failed: isolate individual failures.
	result.FailedIndices = bv.isolateFailures(commitments, openings, blobs)
	return result, nil
}

// isolateFailures individually verifies each commitment to find failures.
func (bv *LatticeBatchVerifier) isolateFailures(
	commitments []*LatticeCommitment,
	openings []*LatticeOpening,
	blobs [][]byte,
) []int {
	var failed []int
	for i := range commitments {
		if commitments[i] == nil || openings[i] == nil || blobs[i] == nil {
			failed = append(failed, i)
			continue
		}
		if !bv.scheme.Verify(commitments[i], openings[i], blobs[i]) {
			failed = append(failed, i)
		}
	}
	return failed
}

// BatchVerifyBlobs is a convenience function that verifies PQBlobCommitments
// against blob data by recomputing commitments and comparing hashes.
func (bv *LatticeBatchVerifier) BatchVerifyBlobs(
	pqCommits []*PQBlobCommitment,
	blobs [][]byte,
) (bool, error) {
	if len(pqCommits) == 0 {
		return false, ErrBatchVerifyEmpty
	}
	if len(pqCommits) != len(blobs) {
		return false, ErrBatchVerifyMismatch
	}

	for i := range pqCommits {
		if pqCommits[i] == nil || blobs[i] == nil {
			return false, ErrBatchVerifyNilEntry
		}
		if !bv.scheme.VerifyBlob(pqCommits[i], blobs[i]) {
			return false, nil
		}
	}
	return true, nil
}

// BatchVerifyBlobsWithIsolation verifies PQBlobCommitments and identifies
// failing entries.
func (bv *LatticeBatchVerifier) BatchVerifyBlobsWithIsolation(
	pqCommits []*PQBlobCommitment,
	blobs [][]byte,
) (*BatchVerifyResult, error) {
	valid, err := bv.BatchVerifyBlobs(pqCommits, blobs)
	if err != nil {
		return nil, err
	}

	result := &BatchVerifyResult{Valid: valid}
	if valid {
		return result, nil
	}

	// Isolate failures.
	var failed []int
	for i := range pqCommits {
		if pqCommits[i] == nil || blobs[i] == nil || len(blobs[i]) == 0 {
			failed = append(failed, i)
			continue
		}
		if !bv.scheme.VerifyBlob(pqCommits[i], blobs[i]) {
			failed = append(failed, i)
		}
	}
	result.FailedIndices = failed
	return result, nil
}
