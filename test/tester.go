package test

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"github.com/pplmx/aurora/internal/blockchain"
	"github.com/pplmx/aurora/internal/voting"
	"strconv"
)

func Blockchain() {
	chain := blockchain.InitBlockChain()

	chain.AddBlock("first block after genesis")
	chain.AddBlock("second block after genesis")
	chain.AddBlock("third block after genesis")

	for _, block := range chain.Blocks {

		fmt.Printf("Previous hash: %x\n", block.PrevHash)
		fmt.Printf("data: %s\n", block.Data)
		fmt.Printf("hash: %x\n", block.Hash)

		pow := blockchain.NewProofOfWork(block)
		fmt.Printf("Pow: %s\n", strconv.FormatBool(pow.Validate()))
		fmt.Println()
	}
}

func DigitalVotingSystem() {
	// Create some candidates with different information
	candidate1 := &voting.Candidate{Name: "Alice", Party: "Green", Program: "Promote environmental protection and renewable energy.", Image: "alice.jpg"}
	candidate2 := &voting.Candidate{Name: "Bob", Party: "Blue", Program: "Support free trade and international cooperation.", Image: "bob.jpg"}
	candidate3 := &voting.Candidate{Name: "Charlie", Party: "Red", Program: "Advocate social justice and welfare reform.", Image: "charlie.jpg"}

	// Create some voters with different tokens
	voter1 := &voting.Vote{Token: string(ed25519.PublicKey{}), Candidate: candidate1.GetName()}
	voter2 := &voting.Vote{Token: string(ed25519.PublicKey{}), Candidate: candidate2.GetName()}
	voter3 := &voting.Vote{Token: string(ed25519.PublicKey{}), Candidate: candidate3.GetName()}

	// Generate a pair of ed25519 keys for each voter
	publicKey1, privateKey1, err := ed25519.GenerateKey(nil)
	if err != nil {
		fmt.Println(err)
	}
	publicKey2, privateKey2, err := ed25519.GenerateKey(nil)
	if err != nil {
		fmt.Println(err)
	}
	publicKey3, privateKey3, err := ed25519.GenerateKey(nil)
	if err != nil {
		fmt.Println(err)
	}

	// Assign the public keys to the voters
	voter1.Token = string(publicKey1)
	voter2.Token = string(publicKey2)
	voter3.Token = string(publicKey3)

	// Sign the votes with the private keys
	signature1 := ed25519.Sign(privateKey1, []byte(voter1.Candidate))
	signature2 := ed25519.Sign(privateKey2, []byte(voter2.Candidate))
	signature3 := ed25519.Sign(privateKey3, []byte(voter3.Candidate))

	// Assign the signatures to the votes
	voter1.Signature = base64.StdEncoding.EncodeToString(signature1)
	voter2.Signature = base64.StdEncoding.EncodeToString(signature2)
	voter3.Signature = base64.StdEncoding.EncodeToString(signature3)

	// Print the votes and their signatures
	fmt.Println("Voter 1 voted for:", voter1.Candidate)
	fmt.Println("Voter 1's signature is:", voter1.Signature)
	fmt.Println("Voter 2 voted for:", voter2.Candidate)
	fmt.Println("Voter 2's signature is:", voter2.Signature)
	fmt.Println("Voter 3 voted for:", voter3.Candidate)
	fmt.Println("Voter 3's signature is:", voter3.Signature)
}
