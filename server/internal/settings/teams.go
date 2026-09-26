package settings

import (
	"fmt"
	"strconv"
	"strings"
)

// ParseTeams parses a string like "1,3,5-8,10" into a slice of uint16.
// Individual values and inclusive ranges (a-b) are supported and may be combined.
func ParseTeams(s string) ([]uint16, error) {
	var result []uint16
	seen := make(map[uint16]struct{})

	for part := range strings.SplitSeq(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		if lo, hi, found := strings.Cut(part, "-"); found {
			loN, err := strconv.ParseUint(strings.TrimSpace(lo), 10, 16)
			if err != nil {
				return nil, fmt.Errorf("invalid range start %q: %w", lo, err)
			}

			hiN, err := strconv.ParseUint(strings.TrimSpace(hi), 10, 16)
			if err != nil {
				return nil, fmt.Errorf("invalid range end %q: %w", hi, err)
			}

			if loN > hiN {
				return nil, fmt.Errorf("range %q has start greater than end", part)
			}

			for n := loN; n <= hiN; n++ {
				t := uint16(n) // #nosec G115
				if _, dup := seen[t]; !dup {
					seen[t] = struct{}{}
					result = append(result, t)
				}
			}
		} else {
			n, err := strconv.ParseUint(part, 10, 16)
			if err != nil {
				return nil, fmt.Errorf("invalid team number %q: %w", part, err)
			}

			t := uint16(n) // #nosec G115
			if _, dup := seen[t]; !dup {
				seen[t] = struct{}{}
				result = append(result, t)
			}
		}
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("no teams specified")
	}
	return result, nil
}
