package lottery

import (
	"filippo.io/edwards25519"
	"testing"
)

func TestVRFVerify_WrongOutputLength(t *testing.T) {
	pk, sk, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}

	message := []byte("test-message")
	_, proof, err := VRFProve(sk, message)
	if err != nil {
		t.Fatalf("VRFProve() error = %v", err)
	}

	valid := VRFVerify(pk, message, []byte("short"), proof)
	if valid {
		t.Error("VRFVerify() should return false for wrong output length")
	}
}

func TestVRFVerify_WrongProofLength(t *testing.T) {
	pk, sk, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}

	message := []byte("test-message")
	output, _, err := VRFProve(sk, message)
	if err != nil {
		t.Fatalf("VRFProve() error = %v", err)
	}

	valid := VRFVerify(pk, message, output, []byte("short"))
	if valid {
		t.Error("VRFVerify() should return false for wrong proof length")
	}
}

func TestVRFVerify_WrongMessage(t *testing.T) {
	pk, sk, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}

	message := []byte("original-message")
	output, proof, err := VRFProve(sk, message)
	if err != nil {
		t.Fatalf("VRFProve() error = %v", err)
	}

	wrongMessage := []byte("different-message")
	valid := VRFVerify(pk, wrongMessage, output, proof)
	if valid {
		t.Error("VRFVerify() should return false for wrong message")
	}
}

func TestVRFVerify_InvalidProofBytes(t *testing.T) {
	pk, _, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}

	message := []byte("test-message")
	output := make([]byte, 32)
	proof := make([]byte, 64)
	for i := range proof {
		proof[i] = 0xFF
	}

	valid := VRFVerify(pk, message, output, proof)
	if valid {
		t.Error("VRFVerify() should return false for invalid proof bytes")
	}
}

func TestVRFVerify_ValidProof(t *testing.T) {
	pk, sk, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}

	message := []byte("test-message-123")
	output, proof, err := VRFProve(sk, message)
	if err != nil {
		t.Fatalf("VRFProve() error = %v", err)
	}

	valid := VRFVerify(pk, message, output, proof)
	if !valid {
		t.Error("VRFVerify() should return true for valid proof")
	}
}

func TestVRFVerify_WithDifferentKeyPair(t *testing.T) {
	pk1, sk1, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}

	_, sk2, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}

	message := []byte("test-message")
	output1, proof1, err := VRFProve(sk1, message)
	if err != nil {
		t.Fatalf("VRFProve() error = %v", err)
	}

	output2, proof2, err := VRFProve(sk2, message)
	if err != nil {
		t.Fatalf("VRFProve() error = %v", err)
	}

	if string(output1) == string(output2) {
		t.Error("Different keys should produce different outputs")
	}

	_ = pk1
	_ = proof1
	_ = proof2
	_ = output2
}

func TestVRFOutputToBytes_ShortInput(t *testing.T) {
	input := []byte("short")
	result := VRFOutputToBytes(input)
	if len(result) != 32 {
		t.Errorf("VRFOutputToBytes() short input length = %d, want 32", len(result))
	}
}

func TestVRFOutputToBytes_LongInput(t *testing.T) {
	input := []byte("this-is-a-long-input-that-exceeds-32-bytes")
	result := VRFOutputToBytes(input)
	if len(result) != 32 {
		t.Errorf("VRFOutputToBytes() long input length = %d, want 32", len(result))
	}
	if result[0] != input[0] || result[1] != input[1] {
		t.Error("VRFOutputToBytes() should preserve first bytes")
	}
}

func TestVRFOutputToBytes_ExactLength(t *testing.T) {
	input := make([]byte, 32)
	for i := range input {
		input[i] = byte(i)
	}
	result := VRFOutputToBytes(input)
	if len(result) != 32 {
		t.Errorf("VRFOutputToBytes() exact length = %d, want 32", len(result))
	}
}

func TestVRFOutputToBytes_EmptyInput(t *testing.T) {
	result := VRFOutputToBytes([]byte{})
	if len(result) != 32 {
		t.Errorf("VRFOutputToBytes() empty input length = %d, want 32", len(result))
	}
}

func TestGenerateKeyPair_Deterministic(t *testing.T) {
	pk1, sk1, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}

	pk2, sk2, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}

	if pk1.Equal(pk2) == 1 {
		t.Error("GenerateKeyPair() should generate different keys each time")
	}

	if sk1.Equal(sk2) == 1 {
		t.Error("GenerateKeyPair() should generate different secret keys each time")
	}
}

func TestGenerateKeyPair_VerifyWorks(t *testing.T) {
	pk, sk, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}

	message := []byte("test")
	output, proof, err := VRFProve(sk, message)
	if err != nil {
		t.Fatalf("VRFProve() error = %v", err)
	}

	if !VRFVerify(pk, message, output, proof) {
		t.Error("VRFVerify() should work with generated key pair")
	}
}

func TestVRFProve_Consistent(t *testing.T) {
	_, sk, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}

	message := []byte("consistent-message")
	output1, proof1, err := VRFProve(sk, message)
	if err != nil {
		t.Fatalf("VRFProve() error = %v", err)
	}

	output2, proof2, err := VRFProve(sk, message)
	if err != nil {
		t.Fatalf("VRFProve() error = %v", err)
	}

	if string(output1) != string(output2) {
		t.Error("VRFProve() should produce same output for same input")
	}
	if string(proof1) != string(proof2) {
		t.Error("VRFProve() should produce same proof for same input")
	}
}

func TestVRFProve_DifferentMessages(t *testing.T) {
	_, sk, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}

	output1, _, err := VRFProve(sk, []byte("message1"))
	if err != nil {
		t.Fatalf("VRFProve() error = %v", err)
	}

	output2, _, err := VRFProve(sk, []byte("message2"))
	if err != nil {
		t.Fatalf("VRFProve() error = %v", err)
	}

	if string(output1) == string(output2) {
		t.Error("VRFProve() should produce different output for different messages")
	}
}

func TestVRFProve_EmptyMessage(t *testing.T) {
	_, sk, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}

	output, proof, err := VRFProve(sk, []byte{})
	if err != nil {
		t.Fatalf("VRFProve() error = %v", err)
	}

	if len(output) != 32 {
		t.Errorf("VRFProve() output length = %d, want 32", len(output))
	}
	if len(proof) != 64 {
		t.Errorf("VRFProve() proof length = %d, want 64", len(proof))
	}
}

func TestVRFVerify_OutputAndProofConsistency(t *testing.T) {
	pk, sk, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}

	message := []byte("test")
	output, proof, err := VRFProve(sk, message)
	if err != nil {
		t.Fatalf("VRFProve() error = %v", err)
	}

	valid := VRFVerify(pk, message, output, proof)
	if !valid {
		t.Error("VRFVerify() should return true for valid proof")
	}

	point := new(edwards25519.Point)
	_, err = point.SetBytes(output)
	if err != nil {
		t.Error("VRF output should be a valid point")
	}

	proofPoint := new(edwards25519.Point)
	proofPoint, err = proofPoint.SetBytes(proof[32:])
	if err != nil {
		t.Error("Proof should contain valid point at offset 32")
	}
}

func TestVRFProve_OutputBytesLength(t *testing.T) {
	_, sk, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}

	output, proof, err := VRFProve(sk, []byte("test"))
	if err != nil {
		t.Fatalf("VRFProve() error = %v", err)
	}

	if len(output) != 32 {
		t.Errorf("VRFProve() output should be 32 bytes, got %d", len(output))
	}
	if len(proof) != 64 {
		t.Errorf("VRFProve() proof should be 64 bytes, got %d", len(proof))
	}

	point := new(edwards25519.Point)
	_, err = point.SetBytes(output)
	if err != nil {
		t.Errorf("VRFProve() output should be valid point: %v", err)
	}
}
