package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/GoogleCloudPlatform/k8s-cloud-provider/e2e"
	_ "k8s.io/klog/v2"
)

func init() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()
		fmt.Fprint(flag.CommandLine.Output(), "\n\n  Example Usage: go run ./cmd/e2e-cleaner/main.go -project my-project -run-id \"\"\n")
	}
}

func main() {
	e2e.ParseFlagsOrDie()
	ctx := context.Background()
	e2e.SetupCloudOrDie(ctx)
	e2e.FallbackCleanup(ctx)
}
