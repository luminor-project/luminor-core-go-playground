package main

import (
	"fmt"
	"os"
	"sort"
)

func main() {
	policy := defaultPolicy()
	importViolations, err := checkImportBoundaries(policy)
	if err != nil {
		fmt.Printf("archtest import analysis failed: %v\n", err)
		os.Exit(1)
	}

	typeViolations, err := checkTypeBoundaries(policy)
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
