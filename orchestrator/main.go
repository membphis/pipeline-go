package main

import (
	"flag"
	"os"

	"orchestrator/internal/log"
	"orchestrator/internal/pipeline"
)

func main() {
	specPath := flag.String("spec", "project.yaml", "Path to project.yaml")
	specShort := flag.String("s", "project.yaml", "Path to project.yaml (shorthand)")
	root := flag.String("root", ".", "Project root directory")
	branch := flag.String("branch", "", "Use existing branch instead of creating milestone branches")
	flag.Parse()

	log.Setup()
	logger := log.Get("main")

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
		_ = logger
		finalSpec = *specShort
	}

	cfg := pipeline.Config{
		SpecPath:   finalSpec,
		Root:       *root,
		ExtraSpecs: extraSpecs,
		Branch:     *branch,
	}

	p := pipeline.New(cfg)
	os.Exit(p.Run())
}
