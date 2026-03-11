package main

import (
	"fmt"
	"os"
	"sort"
)

type checkFunc func(policy) ([]string, error)

func main() {
	p := defaultPolicy()

	// Auto-discover allowed value symbols from facade packages.
	symbols, err := discoverFacadeValueSymbols(p)
	if err != nil {
		fmt.Printf("archtest facade discovery failed: %v\n", err)
		os.Exit(1)
	}
	p.allowedCrossSymbols = symbols

	checks := []struct {
		name string
		fn   checkFunc
	}{
		{"import boundaries", checkImportBoundaries},
		{"type boundaries", checkTypeBoundaries},
		{"domain purity", checkDomainPurity},
		{"facade interfaces", checkNoExportedFacadeInterfaces},
		{"vertical subpackages", checkVerticalSubpackages},
		{"unknown verticals", checkNoUnknownVerticals},
		{"event store immutability", checkEventStoreImmutability},
		{"time.Now()", checkNoDirectTimeNow},
		{"event sourcing", checkEventSourcingRequired},
		{"facade write-path purity", checkFacadeWritePathPurity},
	}

	var violations []string
	for _, c := range checks {
		vs, err := c.fn(p)
		if err != nil {
			fmt.Printf("archtest %s check failed: %v\n", c.name, err)
			os.Exit(1)
		}
		violations = append(violations, vs...)
	}

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
