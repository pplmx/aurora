package test

import (
	"fmt"
	"strconv"

	blockchain "github.com/pplmx/aurora/internal/domain/blockchain"
)

func Blockchain() {
	chain := blockchain.InitBlockChain()

	_, _ = chain.AddBlock("first block after genesis")
	_, _ = chain.AddBlock("second block after genesis")
	_, _ = chain.AddBlock("third block after genesis")

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
	fmt.Println("DigitalVotingSystem: Use 'aurora voting' commands instead")
}
