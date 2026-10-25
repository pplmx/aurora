package lottery

import (
	"testing"

	"filippo.io/edwards25519"
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
	proof := make([]byte, vrfProofLen)
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
	if len(proof) != vrfProofLen {
		t.Errorf("VRFProve() proof length = %d, want %d", len(proof), vrfProofLen)
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

	_, err = new(edwards25519.Point).SetBytes(proof[:32])
	if err != nil {
		t.Error("Proof should contain a valid commitment point R1 at offset 0")
	}
	_, err = new(edwards25519.Point).SetBytes(proof[32:64])
	if err != nil {
		t.Error("Proof should contain a valid commitment point R2 at offset 32")
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
	if len(proof) != vrfProofLen {
		t.Errorf("VRFProve() proof should be %d bytes, got %d", vrfProofLen, len(proof))
	}

	point := new(edwards25519.Point)
	_, err = point.SetBytes(output)
	if err != nil {
		t.Errorf("VRFProve() output should be valid point: %v", err)
	}
}

// TestVRFVerify_RejectsForgedOutputNoSecret is the regression test for the
// integrity flaw where VRFVerify ignored the public key entirely (ISS-096).
// The old no-op verifier accepted any 32-byte output with
// proof = output ‖ hashToPoint(message) — no secret key required — so an
// attacker could record attacker-chosen winners as "verified".
func TestVRFVerify_RejectsForgedOutputNoSecret(t *testing.T) {
	pk, _, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}

	message := []byte("election-2026")
	// A 32-byte output chosen so that SelectWinners would pick the attacker's
	// preferred participant — under the old no-op verifier any such output was
	// accepted with proof = output ‖ hashToPoint(message), no secret needed.
	forgedOutput := []byte("attacker-chosen-output-32-bytes!")
	legacyForgedProof := make([]byte, 0, 64)
	legacyForgedProof = append(legacyForgedProof, forgedOutput...)
	legacyForgedProof = append(legacyForgedProof, hashToPoint(message).Bytes()...)

	// Reject the legacy 64-byte forgery outright.
	if valid := VRFVerify(pk, message, forgedOutput, legacyForgedProof); valid {
		t.Fatal("VRFVerify() must reject an output+proof forged without the secret key (old no-op verifier accepted it)")
	}

	// Reject a padded 96-byte forgery via the binding equations, not the length
	// check: output ‖ H(m) ‖ zeroed response cannot satisfy R₁/R₂ = s·G/c·pk.
	paddedForgedProof := make([]byte, vrfProofLen)
	copy(paddedForgedProof, forgedOutput)
	copy(paddedForgedProof[32:], hashToPoint(message).Bytes())
	if valid := VRFVerify(pk, message, forgedOutput, paddedForgedProof); valid {
		t.Fatal("VRFVerify() must reject a 96-byte forged proof that does not satisfy the Schnorr equations")
	}
}

// TestVRFVerify_WrongPublicKey proves the output is bound to the key: a valid
// proof for key A must fail verification against an unrelated key B.
func TestVRFVerify_WrongPublicKey(t *testing.T) {
	_, skA, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair(A) error = %v", err)
	}
	pkB, _, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair(B) error = %v", err)
	}

	message := []byte("test-message")
	output, proof, err := VRFProve(skA, message)
	if err != nil {
		t.Fatalf("VRFProve() error = %v", err)
	}

	if valid := VRFVerify(pkB, message, output, proof); valid {
		t.Fatal("VRFVerify() must reject a proof verified against a different public key")
	}
}

// TestVRFVerify_TamperedOutput ensures mutating the output breaks the proof.
func TestVRFVerify_TamperedOutput(t *testing.T) {
	pk, sk, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}

	message := []byte("test-message")
	output, proof, err := VRFProve(sk, message)
	if err != nil {
		t.Fatalf("VRFProve() error = %v", err)
	}

	tampered := append([]byte(nil), output...)
	tampered[0] ^= 0x01
	if valid := VRFVerify(pk, message, tampered, proof); valid {
		t.Fatal("VRFVerify() must reject a tampered output")
	}
}

// TestVRFVerify_TamperedProof ensures mutating any proof byte breaks it.
func TestVRFVerify_TamperedProof(t *testing.T) {
	pk, sk, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}

	message := []byte("test-message")
	output, proof, err := VRFProve(sk, message)
	if err != nil {
		t.Fatalf("VRFProve() error = %v", err)
	}

	for _, i := range []int{0, 31, 32, 63, 64, 95} {
		tampered := append([]byte(nil), proof...)
		tampered[i] ^= 0x01
		if valid := VRFVerify(pk, message, output, tampered); valid {
			t.Fatalf("VRFVerify() must reject a proof tampered at byte %d", i)
		}
	}
}

// The tests above derive their inputs from a randomly generated key pair, so
// which validation branch a malformed input takes depends on the draw: a
// made-up proof either fails to decode, decodes to a point that is not the
// group identity, or decodes to the identity — all "false", but through
// different code. The package's coverage therefore wandered between runs
// (measured: 89.8%, 90.3% and 90.8% on consecutive runs).
//
// These cases name the bytes instead, so every rejection reason is exercised
// every time.

// identityNonCanonical is the 32-byte identity point. SetBytes accepts it, and
// isIdentity must then reject it: a proof built on the point at infinity is
// forgeable.
var identityNonCanonical = []byte{
	0x01, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
}

// TestVRFVerify_RejectsTheIdentityPoint drives the isIdentity guard directly:
// r1 is the identity, so the proof is refused before any group arithmetic.
func TestVRFVerify_RejectsTheIdentityPoint(t *testing.T) {
	pk, _, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}

	proof := make([]byte, vrfProofLen)
	copy(proof[:32], identityNonCanonical)   // r1 = identity
	copy(proof[32:64], identityNonCanonical) // r2 = identity
	for i := 64; i < vrfProofLen; i++ {
		proof[i] = 1 // a valid canonical scalar
	}
	output := make([]byte, 32)
	copy(output, identityNonCanonical) // identity output

	if VRFVerify(pk, []byte("identity"), output, proof) {
		t.Fatal("a proof built on the identity point must be rejected")
	}
}

// invalidPointEncoding is rejected by edwards25519.Point.SetBytes: the y
// coordinate it holds is not a field element. Note that 0xFF repeated — what
// an earlier version of this test used — IS a valid encoding, so it exercised
// the identity guard rather than the decode guard, and which one it reached
// depended on the random key pair. Measured: the package's coverage wandered
// between 89.8% and 90.8% across consecutive runs until these cases named
// their bytes.
var invalidPointEncoding = func() []byte {
	b := make([]byte, 32)
	for i := range b {
		b[i] = 0x7F
	}
	return b
}()

// TestVRFVerify_RejectsAnInvalidPointEncoding proves the decode guard: the
// bytes are not a point at all, so the proof is refused before any group
// arithmetic.
func TestVRFVerify_RejectsAnInvalidPointEncoding(t *testing.T) {
	pk, _, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}

	proof := make([]byte, vrfProofLen)
	copy(proof[:32], invalidPointEncoding)   // r1
	copy(proof[32:64], invalidPointEncoding) // r2
	for i := 64; i < vrfProofLen; i++ {
		proof[i] = 1
	}

	if VRFVerify(pk, []byte("invalid-point"), invalidPointEncoding, proof) {
		t.Fatal("a proof of an invalid point encoding must be rejected")
	}
}

// TestVRFVerify_RejectsInvalidR1Alone isolates r1's decode guard: r2 and the
// scalar are valid, so only the r1 check can reject this proof. The existing
// tampered-proof test hits it only when its random key pair produced a proof
// whose first 32 bytes happen to corrupt into a non-point.
func TestVRFVerify_RejectsInvalidR1Alone(t *testing.T) {
	pk, sk, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}
	output, proof, err := VRFProve(sk, []byte("r1-only"))
	if err != nil {
		t.Fatalf("VRFProve() error = %v", err)
	}

	copy(proof[:32], invalidPointEncoding) // r1 invalid, r2 and s intact

	if VRFVerify(pk, []byte("r1-only"), output, proof) {
		t.Fatal("a proof whose r1 is not a point must be rejected")
	}
}

// TestVRFVerify_RejectsInvalidR2Alone isolates r2's decode guard the same way.
func TestVRFVerify_RejectsInvalidR2Alone(t *testing.T) {
	pk, sk, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}
	output, proof, err := VRFProve(sk, []byte("r2-only"))
	if err != nil {
		t.Fatalf("VRFProve() error = %v", err)
	}

	copy(proof[32:64], invalidPointEncoding) // r2 invalid, r1 and s intact

	if VRFVerify(pk, []byte("r2-only"), output, proof) {
		t.Fatal("a proof whose r2 is not a point must be rejected")
	}
}

// TestVRFVerify_RejectsAnInvalidOutputEncoding covers the same guard on the
// output half: the proof's points decode, but the output does not.
func TestVRFVerify_RejectsAnInvalidOutputEncoding(t *testing.T) {
	pk, sk, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}
	_, proof, err := VRFProve(sk, []byte("invalid-output"))
	if err != nil {
		t.Fatalf("VRFProve() error = %v", err)
	}

	if VRFVerify(pk, []byte("invalid-output"), invalidPointEncoding, proof) {
		t.Fatal("an output that is not a point must be rejected")
	}
}

// TestVRFVerify_RejectsNonCanonicalScalar drives the scalar decode guard: the
// point halves decode, but the 32-byte scalar is above the group order.
func TestVRFVerify_RejectsNonCanonicalScalar(t *testing.T) {
	pk, sk, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}
	output, proof, err := VRFProve(sk, []byte("scalar"))
	if err != nil {
		t.Fatalf("VRFProve() error = %v", err)
	}

	// Keep the two valid points, corrupt only the scalar: it is no longer
	// canonical (all 0xFF exceeds the group order).
	for i := 64; i < vrfProofLen; i++ {
		proof[i] = 0xFF
	}

	if VRFVerify(pk, []byte("scalar"), output, proof) {
		t.Fatal("a non-canonical scalar must be rejected")
	}
}

// TestVRFVerify_RejectsAProofOfADifferentMessage keeps the final equation
// honest: a well-formed proof for another message must not verify.
func TestVRFVerify_RejectsAProofOfADifferentMessage(t *testing.T) {
	pk, sk, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}
	output, proof, err := VRFProve(sk, []byte("message-a"))
	if err != nil {
		t.Fatalf("VRFProve() error = %v", err)
	}

	if VRFVerify(pk, []byte("message-b"), output, proof) {
		t.Fatal("a proof must not verify for a different message")
	}
}

// TestVRFVerify_RejectsNilKey covers the nil guard, which a caller reaches when
// a record carries no stored public key.
func TestVRFVerify_RejectsNilKey(t *testing.T) {
	if VRFVerify(nil, []byte("m"), make([]byte, 32), make([]byte, vrfProofLen)) {
		t.Fatal("a nil public key must be rejected")
	}
}
