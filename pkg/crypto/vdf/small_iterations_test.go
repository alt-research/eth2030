package vdf

import "testing"

func TestSmallIterationProofsUseCanonicalVerifier(t *testing.T) {
	for _, iterations := range []uint64{1, 2, 3} {
		input := []byte("low difficulty timer regression")
		result, err := Evaluate(input, iterations)
		if err != nil {
			t.Fatalf("%d: %v", iterations, err)
		}
		solution := result.EncodeSolution()
		if _, ok := VerifySolution(input, iterations, solution); !ok {
			t.Fatalf("%d: canonical proof rejected", iterations)
		}
		if _, ok := VerifySolution(input, iterations+1, solution); ok {
			t.Fatalf("%d: wrong difficulty accepted", iterations)
		}
		solution[len(solution)-1] ^= 1
		if _, ok := VerifySolution(input, iterations, solution); ok {
			t.Fatalf("%d: corrupted proof accepted", iterations)
		}
	}
}
