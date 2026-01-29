package main

import (
	"fmt"
	"log"

	"page/generator"
)

func main() {
	if err := generator.Build(".", "dist"); err != nil {
		log.Fatalf("Site generation failed: %v", err)
	}
	fmt.Println("Site generated successfully in dist/")
}
