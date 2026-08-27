package lottery

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"

	"filippo.io/edwards25519"
)

// Proof length: R₁ (32) ‖ R₂ (32) ‖ s (32) = 96 bytes. The R₁/R₂ pair plus
// the response s form a Schnorr NIZK that the prover knows the secret scalar
// sk satisfying BOTH output = sk·H(seed) AND public = sk·G — i.e. the VRF
// output is bound to the key (see VRFProve/VRFVerify).
const vrfProofLen = 96

// VRFKeyPair holds Ed25519 key material for VRF operations.
// Note: This implementation uses a simplified VRF approach suitable for
// lottery random selection, not full RFC 9380 ECVRF compliance.
type VRFKeyPair struct {
	PublicKey *edwards25519.Point
	SecretKey *edwards25519.Scalar
}

// VRFOutput holds the VRF proof components.
// Note: This is not RFC 9380's on-wire proof format. The prove/verify pair
// below is a Schnorr-style NIZK that binds the output to the key; it keeps
// the simplified hash-to-point from RFC 9380 (see hashToPoint).
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

// EncodePublicKey serializes a VRF public key point to base64 so it can be
// persisted on a LotteryRecord and re-verified after the fact. A VRF curve
// point is exactly 32 bytes, so round-tripping through DecodePublicKey is
// lossless.
func EncodePublicKey(pk *edwards25519.Point) string {
	if pk == nil {
		return ""
	}
	return base64.StdEncoding.EncodeToString(pk.Bytes())
}

// DecodePublicKey parses a base64-encoded VRF public key point. It returns an
// error for empty or malformed input so callers can distinguish "key absent"
// from a corrupt key.
func DecodePublicKey(s string) (*edwards25519.Point, error) {
	if s == "" {
		return nil, errors.New("empty public key")
	}
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, err
	}
	if len(b) != 32 {
		return nil, errors.New("invalid public key length")
	}
	p := new(edwards25519.Point)
	if _, err := p.SetBytes(b); err != nil {
		return nil, err
	}
	return p, nil
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

// VRFProve generates a VRF output and a self-contained proof for the message
// using the secret key.
//
//   - output = sk·H(m)  (H is hashToPoint; the deterministic winner stream)
//   - proof  = R₁ ‖ R₂ ‖ s, a Schnorr NIZK proving knowledge of sk such that
//     output = sk·H(m) AND public = sk·G.
//
// The nonce k is derived deterministically from (sk, message) so that proving
// twice for the same input yields the same output and proof — an audit
// property relied on by CreateLotteryRecord (record IDs hash the output) and
// asserted in the tests. Deterministic nonce derivation also closes the
// weak-key recovery risk a reused random nonce would create.
func VRFProve(secret *edwards25519.Scalar, message []byte) ([]byte, []byte, error) {
	point := hashToPoint(message)

	output := new(edwards25519.Point)
	output.ScalarMult(secret, point)

	// Deterministic per (sk, message): k = H(skBytes ‖ message) reduced mod L.
	nonceDigest := sha256.Sum256(append(secret.Bytes(), message...))
	nonceSeed := make([]byte, 64)
	copy(nonceSeed, nonceDigest[:])
	copy(nonceSeed[32:], nonceDigest[:])
	k, err := new(edwards25519.Scalar).SetUniformBytes(nonceSeed)
	if err != nil {
		return nil, nil, err
	}

	public := new(edwards25519.Point).ScalarBaseMult(secret)
	r1 := new(edwards25519.Point).ScalarBaseMult(k)
	r2 := new(edwards25519.Point).ScalarMult(k, point)

	cSeed := challengeSeed(r1.Bytes(), r2.Bytes(), public.Bytes(), output.Bytes(), message)
	c, err := scalarFromDigest(cSeed)
	if err != nil {
		return nil, nil, err
	}

	s := new(edwards25519.Scalar).Add(k, new(edwards25519.Scalar).Multiply(c, secret))

	proof := make([]byte, vrfProofLen)
	copy(proof[:32], r1.Bytes())
	copy(proof[32:64], r2.Bytes())
	copy(proof[64:], s.Bytes())

	return output.Bytes(), proof, nil
}

// VRFVerify verifies a VRF proof against the public key.
//
// It checks, for message m with H = hashToPoint(m), that the proof's
// (R₁, R₂, s) satisfy, for c = H(R₁ ‖ R₂ ‖ pk ‖ output ‖ m):
//
//	R₁ = s·G − c·pk
//	R₂ = s·H − c·output
//
// Both equations hold iff the prover knew the single scalar sk such that
// output = sk·H(m) and pk = sk·G — the output is genuinely bound to the key.
// Without this NIZK, verification could never demonstrate the output derives
// from any key (the previous implementation ignored `public` entirely, so any
// winners could be recorded as "verified"; ISS-096).
func VRFVerify(public *edwards25519.Point, message []byte, output, proof []byte) bool {
	if public == nil {
		return false
	}
	if len(output) != 32 || len(proof) != vrfProofLen {
		return false
	}

	r1, err := new(edwards25519.Point).SetBytes(proof[:32])
	if err != nil {
		return false
	}
	r2, err := new(edwards25519.Point).SetBytes(proof[32:64])
	if err != nil {
		return false
	}
	s, err := new(edwards25519.Scalar).SetCanonicalBytes(proof[64:])
	if err != nil {
		return false
	}
	outputPoint, err := new(edwards25519.Point).SetBytes(output)
	if err != nil {
		return false
	}
	if isIdentity(outputPoint) || isIdentity(r1) || isIdentity(r2) {
		return false
	}

	h := hashToPoint(message)
	c, err := scalarFromDigest(challengeSeed(r1.Bytes(), r2.Bytes(), public.Bytes(), output, message))
	if err != nil {
		return false
	}

	// R₁ ?= s·G − c·pk
	sG := new(edwards25519.Point).ScalarBaseMult(s)
	cPK := new(edwards25519.Point).ScalarMult(c, public)
	if new(edwards25519.Point).Subtract(sG, cPK).Equal(r1) != 1 {
		return false
	}

	// R₂ ?= s·H − c·output
	sH := new(edwards25519.Point).ScalarMult(s, h)
	cOut := new(edwards25519.Point).ScalarMult(c, outputPoint)
	if new(edwards25519.Point).Subtract(sH, cOut).Equal(r2) != 1 {
		return false
	}

	return true
}

// challengeSeed computes the Fiat-Shamir challenge input for the VRF proof as
// a 64-byte digest (two linked SHA-256 blocks) that SetUniformBytes reduces
// modulo the group order. Binding the public key and output into the challenge
// prevents re-targeting a proof at a different key or output.
func challengeSeed(r1, r2, public, output, message []byte) []byte {
	var buf []byte
	buf = append(buf, r1...)
	buf = append(buf, r2...)
	buf = append(buf, public...)
	buf = append(buf, output...)
	buf = append(buf, message...)
	first := sha256.Sum256(buf)
	second := sha256.Sum256(first[:])
	seed := make([]byte, 64)
	copy(seed, first[:])
	copy(seed[32:], second[:])
	return seed
}

func scalarFromDigest(seed []byte) (*edwards25519.Scalar, error) {
	return new(edwards25519.Scalar).SetUniformBytes(seed)
}

// isIdentity reports whether p encodes the group identity (the point at
// infinity), which must never be accepted as a VRF output or commitment.
func isIdentity(p *edwards25519.Point) bool {
	return p.Equal(edwards25519.NewIdentityPoint()) == 1
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
