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

	domainViolations, err := checkDomainPurity(p)
	if err != nil {
		fmt.Printf("archtest domain purity check failed: %v\n", err)
		os.Exit(1)
	}

	facadeIfaceViolations, err := checkNoExportedFacadeInterfaces(p)
	if err != nil {
		fmt.Printf("archtest facade interface check failed: %v\n", err)
		os.Exit(1)
	}

	subpkgViolations, err := checkVerticalSubpackages(p)
	if err != nil {
		fmt.Printf("archtest vertical subpackage check failed: %v\n", err)
		os.Exit(1)
	}

	unknownViolations, err := checkNoUnknownVerticals(p)
	if err != nil {
		fmt.Printf("archtest unknown verticals check failed: %v\n", err)
		os.Exit(1)
	}

	eventstoreViolations, err := checkEventStoreImmutability(p)
	if err != nil {
		fmt.Printf("archtest event store immutability check failed: %v\n", err)
		os.Exit(1)
	}

	timeNowViolations, err := checkNoDirectTimeNow(p)
	if err != nil {
		fmt.Printf("archtest time.Now() check failed: %v\n", err)
		os.Exit(1)
	}

	violations := append(importViolations, typeViolations...)
	violations = append(violations, domainViolations...)
	violations = append(violations, facadeIfaceViolations...)
	violations = append(violations, subpkgViolations...)
	violations = append(violations, unknownViolations...)
	violations = append(violations, eventstoreViolations...)
	violations = append(violations, timeNowViolations...)
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
