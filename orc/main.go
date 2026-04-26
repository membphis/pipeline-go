package main

import (
	"flag"
	"fmt"
	"os"

	"orc/internal/log"
	"orc/internal/pipeline"
)

func main() {
	specPath := flag.String("spec", "project.yaml", "Path to project.yaml")
	specShort := flag.String("s", "project.yaml", "Path to project.yaml (shorthand)")
	root := flag.String("root", "", "Project root directory (required)")
	flag.Parse()

	if *root == "" {
		fmt.Fprintln(os.Stderr, "ERROR: --root is required")
		flag.Usage()
		os.Exit(1)
	}

	log.Setup()

	var extraSpecs []string
	args := flag.Args()
	for i := 0; i < len(args); i++ {
		if args[i] == "--extra-spec" && i+1 < len(args) {
			extraSpecs = append(extraSpecs, args[i+1])
			i++
		}
	}

	finalSpec := *specPath
	if *specShort != "project.yaml" {
		finalSpec = *specShort
	}

	cfg := pipeline.Config{
		SpecPath:   finalSpec,
		Root:       *root,
		ExtraSpecs: extraSpecs,
	}

	p := pipeline.New(cfg)
	os.Exit(p.Run())
}
