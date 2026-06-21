package agent

import (
	"testing"
)

func TestParseTeams(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []uint16
		wantErr bool
	}{
		{"single", "5", []uint16{5}, false},
		{"zero", "0", []uint16{0}, false},
		{"multiple", "1,3,7", []uint16{1, 3, 7}, false},
		{"range", "1-5", []uint16{1, 2, 3, 4, 5}, false},
		{"range single", "3-3", []uint16{3}, false},
		{"mixed", "1,3,5-7,10", []uint16{1, 3, 5, 6, 7, 10}, false},
		{"dedup individual", "1,2,1", []uint16{1, 2}, false},
		{"dedup range overlap", "1-3,2-4", []uint16{1, 2, 3, 4}, false},
		{"spaces around comma", " 1 , 3 ", []uint16{1, 3}, false},
		{"empty string", "", nil, true},
		{"only commas", ",,,", nil, true},
		{"reversed range", "5-3", nil, true},
		{"invalid value", "abc", nil, true},
		{"invalid range start", "a-5", nil, true},
		{"invalid range end", "1-z", nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseTeams(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseTeams(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if len(got) != len(tt.want) {
				t.Fatalf("ParseTeams(%q) = %v, want %v", tt.input, got, tt.want)
			}
			for i, v := range got {
				if v != tt.want[i] {
					t.Errorf("ParseTeams(%q)[%d] = %v, want %v", tt.input, i, v, tt.want[i])
				}
			}
		})
	}
}
