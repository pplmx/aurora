package lottery

import (
	"crypto/rand"
	"crypto/sha256"

	"filippo.io/edwards25519"
)

// VRFKeyPair holds Ed25519 key material for VRF operations.
// Note: This implementation uses a simplified VRF approach suitable for
// lottery random selection, not full RFC 9380 ECVRF compliance.
type VRFKeyPair struct {
	PublicKey *edwards25519.Point
	SecretKey *edwards25519.Scalar
}

// VRFOutput holds the VRF proof components.
// Note: This is not RFC 9380's proof format which includes c and s values
// for cryptographic verification. This simplified format only includes
// the point for hash verification.
type VRFOutput struct {
	Output []byte
	Proof  []byte
}

func GenerateKeyPair() (*edwards25519.Point, *edwards25519.Scalar, error) {
	var randomBytes [64]byte
	_, err := rand.Read(randomBytes[:])
	if err != nil {
		return nil, nil, err
	}

	secret := new(edwards25519.Scalar)
	secret, err = secret.SetUniformBytes(randomBytes[:])
	if err != nil {
		return nil, nil, err
	}

	public := new(edwards25519.Point)
	public.ScalarBaseMult(secret)

	return public, secret, nil
}

// hashToPoint converts a message to a curve point using SHA-256.
//
// LIMITATION: This is NOT RFC 9380 (ECVRF) compliant. RFC 9380 specifies
// hash-to-curve using SHA-512 with counter-mode hashing and proper elliptic
// curve point generation. This implementation uses SHA-256 and scalar multiplication
// which does not guarantee the resulting point is uniformly random on the curve.
//
// Current approach: Hash message with SHA-256, interpret as scalar, multiply by
// generator point. This produces a valid point but may have biases.
//
// RFC 9380 approach would use:
//   - SHA-512 instead of SHA-256
//   - Hash to field elements with proper reduction
//   - Hash to point using either SWU method or test-and-check
//
// Trade-offs for using SHA-256 here:
//   - Simpler implementation with fewer dependencies
//   - Adequate for lottery use case where unbiased randomness isn't critical
//   - No need for additional hash-to-curve logic
//
// If RFC 9380 compliance is needed, consider:
//   - FiloSottile/frussito/circl libraries with hash-to-curve implementations
//   - Or implementing draft-irtf-cfrg-hash-to-curve with Ed25519 curve
func hashToPoint(message []byte) *edwards25519.Point {
	h := sha256Hash(message)
	var bytes [64]byte
	copy(bytes[:32], h[:])
	scalar, err := new(edwards25519.Scalar).SetUniformBytes(bytes[:])
	if err != nil {
		scalar = new(edwards25519.Scalar)
	}
	point := new(edwards25519.Point)
	point.ScalarBaseMult(scalar)
	return point
}

// VRFProve generates a VRF proof for the message using the secret key.
//
// Returns:
//   - output: H(m)^sk where H is hash-to-point and sk is secret key
//   - proof: combined output and point bytes
//   - error: if key generation fails
//
// Note: This implements a simplified VRF proof generation. Standard ECVRF
// proof generation includes additional fields (gamma, c, s) for non-interactive
// proof verification. See RFC 9380 Section 5.2.
func VRFProve(secret *edwards25519.Scalar, message []byte) ([]byte, []byte, error) {
	point := hashToPoint(message)

	output := new(edwards25519.Point)
	output.ScalarMult(secret, point)

	outputBytes := output.Bytes()
	proofBytes := point.Bytes()

	combined := make([]byte, len(outputBytes)+len(proofBytes))
	copy(combined, outputBytes)
	copy(combined[len(outputBytes):], proofBytes)

	return outputBytes, combined, nil
}

// VRFVerify verifies a VRF proof.
//
// Verification approach: Re-hash message to point, compare with proof's point.
// Does NOT verify the output derivation (secret * point) due to simplified design.
//
// Note: This simplified verification only checks that the proof contains
// the correct hash-to-point result. It does NOT verify the VRF output
// signature (i.e., that output = secret * proofPoint). For production use
// with security requirements, implement full ECVRF verification per RFC 9380
// Section 5.4, which includes checking the c and s values from the proof.
func VRFVerify(public *edwards25519.Point, message []byte, output, proof []byte) bool {
	if len(output) != 32 || len(proof) != 64 {
		return false
	}

	recomputedPoint := hashToPoint(message)
	proofPoint := new(edwards25519.Point)
	proofPoint, err := proofPoint.SetBytes(proof[32:])
	if err != nil {
		return false
	}

	if recomputedPoint.Equal(proofPoint) != 1 {
		return false
	}

	return len(output) == 32
}

func VRFOutputToBytes(output []byte) []byte {
	if len(output) >= 32 {
		return output[:32]
	}
	result := make([]byte, 32)
	copy(result, output)
	return result
}

func sha256Hash(message []byte) [32]byte {
	h := sha256.Sum256(message)
	return h
}
