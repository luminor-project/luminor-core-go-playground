package main

import (
	"fmt"
	"os"
	"sort"
)

func main() {
	p := defaultPolicy()

	// Auto-discover allowed value symbols from facade packages.
	symbols, err := discoverFacadeValueSymbols(p)
	if err != nil {
		fmt.Printf("archtest facade discovery failed: %v\n", err)
		os.Exit(1)
	}
	p.allowedCrossSymbols = symbols

	importViolations, err := checkImportBoundaries(p)
	if err != nil {
		fmt.Printf("archtest import analysis failed: %v\n", err)
		os.Exit(1)
	}

	typeViolations, err := checkTypeBoundaries(p)
	if err != nil {
		fmt.Printf("archtest type analysis failed: %v\n", err)
		os.Exit(1)
	}

	violations := append(importViolations, typeViolations...)
	sort.Strings(violations)
	reportOnly := os.Getenv("ARCHTEST_REPORT_ONLY") == "1"

	if len(violations) > 0 {
		fmt.Println("Architecture boundary violations:")
		for _, v := range violations {
			fmt.Printf("  VIOLATION: %s\n", v)
		}
		if reportOnly {
			fmt.Println("ARCHTEST_REPORT_ONLY=1 set; reporting only.")
		} else {
			os.Exit(1)
		}
	}

	fmt.Println("Architecture boundary checks passed.")
}
