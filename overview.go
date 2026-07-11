// overview.go: catalog overview commands (families, stats).
package main

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

type familyStat struct {
	Family string `json:"family"`
	Count  int    `json:"count"`
}

// familyCounts tallies records per Nerd Font family (derived via glyphFamily).
func familyCounts(records []Record) map[string]int {
	counts := make(map[string]int)
	for _, rec := range records {
		counts[glyphFamily(rec.Name)]++
	}
	return counts
}

// familyList returns family stats sorted by count desc, then family asc.
func familyList(records []Record) []familyStat {
	counts := familyCounts(records)
	out := make([]familyStat, 0, len(counts))
	for fam, n := range counts {
		out = append(out, familyStat{Family: fam, Count: n})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Family < out[j].Family
	})
	return out
}

type catalogStats struct {
	Total    int `json:"total"`
	Families int `json:"families"`
	Labeled  int `json:"labeled"`
	Aliased  int `json:"aliased"`
}

// computeStats summarizes record totals, family count, and labeled/aliased counts.
func computeStats(records []Record) catalogStats {
	var s catalogStats
	s.Total = len(records)
	fams := map[string]struct{}{}
	for _, rec := range records {
		if rec.Label != "" {
			s.Labeled++
		}
		if len(rec.Aliases) > 0 {
			s.Aliased++
		}
		fams[glyphFamily(rec.Name)] = struct{}{}
	}
	s.Families = len(fams)
	return s
}

// parseFamiliesArgs parses clyph families flags. limit < 0 means "all" (default).
func parseFamiliesArgs(args []string) (limit int, jsonOut bool, err error) {
	limit = -1
	for i := 0; i < len(args); i++ {
		arg := args[i]
		name, inline, hasInline := strings.Cut(arg, "=")
		switch name {
		case "--json":
			jsonOut = true
		case "--limit":
			val, verr := flagValue(args, &i, "--limit", inline, hasInline)
			if verr != nil {
				return 0, false, verr
			}
			limit, err = strconv.Atoi(val)
			if err != nil {
				return 0, false, cliError{"invalid --limit"}
			}
			if limit < 0 {
				return 0, false, cliError{"--limit must be non-negative"}
			}
		default:
			if strings.HasPrefix(arg, "--") {
				return 0, false, cliError{"unknown flag: " + arg}
			}
			return 0, false, cliError{"too many arguments"}
		}
	}
	return limit, jsonOut, nil
}

func cmdFamilies(args []string) int {
	limit, jsonOut, err := parseFamiliesArgs(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	records, lerr := loadRecords(catalogPath())
	if lerr != nil {
		printError(jsonOut, lerr.Error())
		return 1
	}
	fams := familyList(records)
	if limit >= 0 && limit < len(fams) {
		fams = fams[:limit]
	}
	if jsonOut {
		if perr := printJSON(fams); perr != nil {
			printError(true, perr.Error())
			return 1
		}
		return 0
	}
	for _, f := range fams {
		fmt.Printf("%s\t%d\n", f.Family, f.Count)
	}
	return 0
}

// parseStatsArgs parses clyph stats flags (json only).
func parseStatsArgs(args []string) (jsonOut bool, err error) {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		name, _, _ := strings.Cut(arg, "=")
		switch name {
		case "--json":
			jsonOut = true
		default:
			if strings.HasPrefix(arg, "--") {
				return false, cliError{"unknown flag: " + arg}
			}
			return false, cliError{"too many arguments"}
		}
	}
	return jsonOut, nil
}

func cmdStats(args []string) int {
	jsonOut, err := parseStatsArgs(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	records, lerr := loadRecords(catalogPath())
	if lerr != nil {
		printError(jsonOut, lerr.Error())
		return 1
	}
	s := computeStats(records)
	if jsonOut {
		if perr := printJSON(s); perr != nil {
			printError(true, perr.Error())
			return 1
		}
		return 0
	}
	fmt.Printf("total\t%d\n", s.Total)
	fmt.Printf("families\t%d\n", s.Families)
	fmt.Printf("labeled\t%d\n", s.Labeled)
	fmt.Printf("aliased\t%d\n", s.Aliased)
	return 0
}
