package lottery

import (
	"crypto/rand"
	"crypto/sha256"

	"filippo.io/edwards25519"
)

type VRFKeyPair struct {
	PublicKey *edwards25519.Point
	SecretKey *edwards25519.Scalar
}

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
