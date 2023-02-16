package main

import (
	"fmt"
	"github.com/pplmx/aurora/internal/models"
	"time"
)

// main function to demonstrate blockchain creation and validation
func main() {
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
