package test

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"github.com/pplmx/aurora/internal/models"
	"time"
)

func testBaseBlockchain() {
	var chain models.Blockchain

	genesisBlock := models.Block{Timestamp: time.Now().String(), Data: "Genesis Block"}
	genesisBlock.Hash = models.CalculateHash(genesisBlock)
	chain = append(chain, genesisBlock)

	fmt.Println("genesisBlock created")
	fmt.Println("Index:", genesisBlock.Index)
	fmt.Println("Timestamp:", genesisBlock.Timestamp)
	fmt.Println("Data:", genesisBlock.Data)
	fmt.Println("Hash:", genesisBlock.Hash)
	fmt.Println()

	firstBlock := models.CreateBlock(genesisBlock, "First Block")
	if models.IsBlockValid(firstBlock, genesisBlock) {
		chain = append(chain, firstBlock)
		fmt.Println("firstBlock created")
		fmt.Println("index:", firstBlock.Index)
		fmt.Println("timestamp:", firstBlock.Timestamp)
		fmt.Println("data:", firstBlock.Data)
		fmt.Println("hash:", firstBlock.Hash)
		fmt.Println()
	}

	secondBlock := models.CreateBlock(firstBlock, "Second Block")
	if models.IsBlockValid(secondBlock, firstBlock) {
		chain = append(chain, secondBlock)
		fmt.Println("secondBlocK created")
		fmt.Println("index:", secondBlock.Index)
		fmt.Println("timestamp:", secondBlock.Timestamp)
		fmt.Println("data:", secondBlock.Data)
		fmt.Println("hash:", secondBlock.Hash)
		fmt.Println()
	}
}

func testDigitalVotingSystem() {
	// Create some candidates with different information
	candidate1 := &models.Candidate{Name: "Alice", Party: "Green", Program: "Promote environmental protection and renewable energy.", Image: "alice.jpg"}
	candidate2 := &models.Candidate{Name: "Bob", Party: "Blue", Program: "Support free trade and international cooperation.", Image: "bob.jpg"}
	candidate3 := &models.Candidate{Name: "Charlie", Party: "Red", Program: "Advocate social justice and welfare reform.", Image: "charlie.jpg"}

	// Create some voters with different tokens
	voter1 := &models.Vote{Token: string(ed25519.PublicKey{}), Candidate: candidate1.GetName()}
	voter2 := &models.Vote{Token: string(ed25519.PublicKey{}), Candidate: candidate2.GetName()}
	voter3 := &models.Vote{Token: string(ed25519.PublicKey{}), Candidate: candidate3.GetName()}

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
