package main

import (
	"fmt"
	"log"

	"observability-hub/internal/web"
)

func main() {
	if err := web.Build("./internal/web", "./dist"); err != nil {
		log.Fatalf("Site generation failed: %v", err)
	}
	fmt.Println("Site generated successfully in dist/")
}
