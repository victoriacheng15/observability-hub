package main

import (
	"os"

	"observability-hub/internal/idp"
)

func main() {
	os.Exit(idp.Run(os.Args[1:], os.Stdout, os.Stderr))
}
