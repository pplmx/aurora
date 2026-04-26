package blockchain

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"log"
	"math"
	"math/big"
)

const Difficulty = 4

type ProofOfWork struct {
	Block  *Block
	Target *big.Int
}

func NewProofOfWork(b *Block) *ProofOfWork {
	target := big.NewInt(1)
	target.Lsh(target, uint(256-Difficulty))

	pow := &ProofOfWork{b, target}

	return pow
}

func ToHex(num int64) []byte {
	buff := new(bytes.Buffer)
	err := binary.Write(buff, binary.BigEndian, num)
	if err != nil {
		log.Panic(err)
	}
	return buff.Bytes()
}

func (pow *ProofOfWork) InitNonce(nonce int) []byte {
	data := bytes.Join(
		[][]byte{
			pow.Block.PrevHash,
			pow.Block.Data,
			ToHex(int64(nonce)),
			ToHex(int64(Difficulty)),
		},
		[]byte{},
	)
	return data
}

func (pow *ProofOfWork) Run() (int, []byte) {
	var intHash big.Int
	var hash [32]byte

	nonce := 0
	for nonce < math.MaxInt64 {
		data := pow.InitNonce(nonce)
		hash = sha256.Sum256(data)

		intHash.SetBytes(hash[:])

		if intHash.Cmp(pow.Target) == -1 {
			break
		} else {
			nonce++
		}

	}

	return nonce, hash[:]
}

func MineBlockWithContext(ctx context.Context, pow *ProofOfWork) (int, []byte, error) {
	var intHash big.Int
	var hash [32]byte

	nonce := 0
	for nonce < math.MaxInt64 {
		select {
		case <-ctx.Done():
			return nonce, hash[:], ctx.Err()
		default:
		}

		data := pow.InitNonce(nonce)
		hash = sha256.Sum256(data)

		intHash.SetBytes(hash[:])

		if intHash.Cmp(pow.Target) == -1 {
			return nonce, hash[:], nil
		}

		nonce++
	}

	return nonce, hash[:], errors.New("max iterations exceeded")
}

func ValidateProof(data []byte, hash []byte, target *big.Int) bool {
	var intHash big.Int
	h := sha256.Sum256(data)
	intHash.SetBytes(h[:])
	return intHash.Cmp(target) == -1
}

func (pow *ProofOfWork) Validate() bool {
	var intHash big.Int

	data := pow.InitNonce(int(pow.Block.Nonce))

	hash := sha256.Sum256(data)
	intHash.SetBytes(hash[:])

	return intHash.Cmp(pow.Target) == -1
}
