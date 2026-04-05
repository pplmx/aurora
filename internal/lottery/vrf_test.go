package lottery

import (
	"testing"
)

func TestGenerateKeyPair(t *testing.T) {
	pk, sk, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	if pk == nil {
		t.Error("Public key should not be nil")
	}
	if sk == nil {
		t.Error("Secret key should not be nil")
	}
}

func TestVRFProveVerify(t *testing.T) {
	seed := "test-seed-123"
	pk, sk, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	output, proof, err := VRFProve(sk, []byte(seed))
	if err != nil {
		t.Fatalf("VRFProve failed: %v", err)
	}

	if len(output) == 0 {
		t.Error("VRF output should not be empty")
	}

	if len(proof) == 0 {
		t.Error("VRF proof should not be empty")
	}

	valid := VRFVerify(pk, []byte(seed), output, proof)
	if !valid {
		t.Error("VRFVerify should return true for valid proof")
	}
}

func TestVRFUniqueness(t *testing.T) {
	seed := "same-seed"
	_, sk1, _ := GenerateKeyPair()
	_, sk2, _ := GenerateKeyPair()

	output1, _, _ := VRFProve(sk1, []byte(seed))
	output2, _, _ := VRFProve(sk2, []byte(seed))

	if string(output1) == string(output2) {
		t.Error("Different key pairs should produce different outputs")
	}
}

func TestVRFDifferentSeeds(t *testing.T) {
	_, sk, _ := GenerateKeyPair()

	output1, _, _ := VRFProve(sk, []byte("seed-1"))
	output2, _, _ := VRFProve(sk, []byte("seed-2"))

	if string(output1) == string(output2) {
		t.Error("Different seeds should produce different outputs")
	}
}
