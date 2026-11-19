package payload

import (
	"github.com/eth2030/eth2030/core/types"
	"testing"
)

func TestValidateAttributesAcceptsSlotZero(t *testing.T) {
	attrs := &PayloadAttributesV4{PayloadAttributesV3: PayloadAttributesV3{PayloadAttributesV2: PayloadAttributesV2{PayloadAttributesV1: PayloadAttributesV1{Timestamp: 12}}, ParentBeaconBlockRoot: types.Hash{1}}}
	if err := ValidateAttributesV4(attrs, 0, 12); err != nil {
		t.Fatalf("explicit slot zero was rejected: %v", err)
	}
}
