package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

func runCheck(args []string) {
	fs := flag.NewFlagSet("check", flag.ExitOnError)
	session := fs.String("session", "", "path to git repository")
	failOn := fs.String("fail-on", "", "comma-separated list of metrics (complexity,coverage,secrets)")
	maxComplexityDelta := fs.Int("max-complexity-delta", 10, "maximum allowed complexity delta")
	minCoverageTouch := fs.Float64("min-coverage-touch", 0.0, "minimum coverage touch ratio (0.0-1.0)")
	fs.Parse(args)

	if *session == "" || *failOn == "" {
		fmt.Fprintln(os.Stderr, "Usage: ghostlog check -session <dir> -fail-on complexity,coverage,secrets")
		os.Exit(1)
	}

	checkComplexity := strings.Contains(*failOn, "complexity")
	checkCoverage := strings.Contains(*failOn, "coverage")
	checkSecrets := strings.Contains(*failOn, "secrets")

	bursts := extractBursts(*session)
	failed := false

	for _, b := range bursts {
		delta := b.ComplexityAfter - b.ComplexityBefore

		if checkComplexity && delta > *maxComplexityDelta {
			fmt.Fprintf(os.Stderr, "burst %d: complexity delta %d exceeds %d\n", b.ID, delta, *maxComplexityDelta)
			failed = true
		}

		if checkSecrets && len(b.SecretLeaks) > 0 {
			fmt.Fprintf(os.Stderr, "burst %d: detected %d secret leaks:\n", b.ID, len(b.SecretLeaks))
			for _, leak := range b.SecretLeaks {
				fmt.Fprintf(os.Stderr, "  - %s\n", leak)
			}
			failed = true
		}

		if checkCoverage && b.TotalChangedFunctions > 0 {
			tested := b.TotalChangedFunctions - len(b.UntestedFunctions)
			cov := float64(tested) / float64(b.TotalChangedFunctions)
			if cov < *minCoverageTouch {
				fmt.Fprintf(os.Stderr, "burst %d: coverage touch %.2f below %.2f (%d untested)\n", b.ID, cov, *minCoverageTouch, len(b.UntestedFunctions))
				failed = true
			}
		} else if checkCoverage && *minCoverageTouch > 0.0 && len(b.UntestedFunctions) > 0 {
			// edge case: if somehow TotalChangedFunctions didn't catch it but untested is populated
			fmt.Fprintf(os.Stderr, "burst %d: has %d untested functions\n", b.ID, len(b.UntestedFunctions))
			failed = true
		}
	}

	if failed {
		os.Exit(1)
	}
	os.Exit(0)
}
