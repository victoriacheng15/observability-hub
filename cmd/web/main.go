package main

import (
	"fmt"
	"log"

	"observability-hub/internal/web/generator"
)

func main() {
	if err := generator.Build("../../internal/web", "dist"); err != nil {
		log.Fatalf("Site generation failed: %v", err)
	}
	fmt.Println("Site generated successfully in dist/")
}
