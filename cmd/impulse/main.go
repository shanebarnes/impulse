package main

import (
	"fmt"
	"os"

	"github.com/shanebarnes/impulse/internal/version"
)

func main() {
	fmt.Fprintf(os.Stdout, "impulse version %s\n", version.String())
}
