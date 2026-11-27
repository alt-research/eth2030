package config

import "testing"

func TestFrameDOptionalActivationAndCompatibility(t *testing.T) {
	for _, name := range []string{"EIP8298", "EIP7906", "EIP8504"} {
		t.Run(name, func(t *testing.T) {
			before := *TestConfigHogota
			after := before
			at := uint64(100)
			var active func(uint64) bool
			switch name {
			case "EIP8298":
				after.EIP8298Time = &at
				active = after.IsEIP8298
			case "EIP7906":
				after.EIP7906Time = &at
				active = after.IsEIP7906
			case "EIP8504":
				after.EIP8504Time = &at
				active = after.IsEIP8504
			}
			if before.IsEIP8298(100) || before.IsEIP7906(100) || before.IsEIP8504(100) {
				t.Fatal("Group D activated by default")
			}
			if active(99) || !active(100) || !active(101) {
				t.Fatal("incorrect optional activation boundary")
			}
			if err := after.Validate(); err != nil {
				t.Fatal(err)
			}
			if err := CheckConfigCompatible(&before, &after, 1, 99); err != nil {
				t.Fatal(err)
			}
			if err := CheckConfigCompatible(&before, &after, 1, 100); err == nil || err.ForkName != name {
				t.Fatal("retroactive activation accepted", err)
			}
			if err := CheckConfigCompatible(&after, &before, 1, 100); err == nil {
				t.Fatal("retroactive removal accepted")
			}
			if name == "EIP8298" {
				after.AmsterdamTime = nil
			} else {
				after.HogotaTime = nil
			}
			if err := after.Validate(); err == nil {
				t.Fatal("missing prerequisite accepted")
			}
		})
	}
}
