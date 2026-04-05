package lottery

import (
	"testing"
)

func TestNameToAddress(t *testing.T) {
	tests := []struct {
		name     string
		wantLen  int
		wantPref string
	}{
		{"张三", 42, "0x"},
		{"李四", 42, "0x"},
		{"Alice", 42, "0x"},
		{"Bob", 42, "0x"},
		{"", 42, "0x"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NameToAddress(tt.name)
			if len(got) != tt.wantLen {
				t.Errorf("NameToAddress(%q) len = %d, want %d", tt.name, len(got), tt.wantLen)
			}
			if got[:2] != tt.wantPref {
				t.Errorf("NameToAddress(%q) = %v, want prefix %v", tt.name, got, tt.wantPref)
			}
		})
	}
}

func TestNameToAddressConsistency(t *testing.T) {
	name := "张三"
	addr1 := NameToAddress(name)
	addr2 := NameToAddress(name)
	if addr1 != addr2 {
		t.Errorf("NameToAddress(%q) not consistent: %q != %q", name, addr1, addr2)
	}
}

func TestNameToAddressDifferent(t *testing.T) {
	addr1 := NameToAddress("张三")
	addr2 := NameToAddress("李四")
	if addr1 == addr2 {
		t.Errorf("Different names should produce different addresses")
	}
}
