package score

import (
	"testing"
)

func TestCleanTemplateString(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"{{.TeamNumber}}", ".TeamNumber"},
		{"{{.TeamNumberHex}}.example.com", ".TeamNumberHex.example.com"},
		{"plain", "plain"},
		{"no braces", "no braces"},
		{"{{nested{{double}}", "nesteddouble"},
	}
	for _, tt := range tests {
		got := cleanTemplateString(tt.input)
		if got != tt.want {
			t.Errorf("cleanTemplateString(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestAllowedArgumentOverrides(t *testing.T) {
	t.Run("allows only permitted fields", func(t *testing.T) {
		allowed := []string{"host", "port"}
		attrs := map[string]string{
			"host":   "{{.TeamNumberHex}}.example.com",
			"port":   "8080",
			"secret": "password",
		}

		var result map[string]string
		allowedArgumentOverrides(allowed, attrs, &result)

		if _, ok := result["secret"]; ok {
			t.Error("secret should not be in result")
		}
		if result["host"] != ".TeamNumberHex.example.com" {
			t.Errorf("host = %q, want cleaned template", result["host"])
		}
		if result["port"] != "8080" {
			t.Errorf("port = %q, want 8080", result["port"])
		}
	})

	t.Run("nil attributes produces empty map", func(t *testing.T) {
		var result map[string]string
		allowedArgumentOverrides([]string{"host"}, nil, &result)
		if len(result) != 0 {
			t.Errorf("expected empty map, got %v", result)
		}
	})

	t.Run("empty allowed list produces empty map", func(t *testing.T) {
		var result map[string]string
		allowedArgumentOverrides(nil, map[string]string{"host": "x"}, &result)
		if len(result) != 0 {
			t.Errorf("expected empty map, got %v", result)
		}
	})
}

func TestApplyOverrides(t *testing.T) {
	type target struct {
		Host    string
		Port    int
		Enabled bool
		Score   float64
	}

	t.Run("string field", func(t *testing.T) {
		s := &target{Host: "original"}
		if err := applyOverrides(s, map[string]string{"Host": "new-host"}); err != nil {
			t.Fatal(err)
		}
		if s.Host != "new-host" {
			t.Errorf("Host = %q, want %q", s.Host, "new-host")
		}
	})

	t.Run("int field", func(t *testing.T) {
		s := &target{}
		if err := applyOverrides(s, map[string]string{"Port": "9090"}); err != nil {
			t.Fatal(err)
		}
		if s.Port != 9090 {
			t.Errorf("Port = %d, want 9090", s.Port)
		}
	})

	t.Run("bool field", func(t *testing.T) {
		s := &target{}
		if err := applyOverrides(s, map[string]string{"Enabled": "true"}); err != nil {
			t.Fatal(err)
		}
		if !s.Enabled {
			t.Error("Enabled should be true")
		}
	})

	t.Run("float field", func(t *testing.T) {
		s := &target{}
		if err := applyOverrides(s, map[string]string{"Score": "3.14"}); err != nil {
			t.Fatal(err)
		}
		if s.Score != 3.14 {
			t.Errorf("Score = %f, want 3.14", s.Score)
		}
	})

	t.Run("case insensitive match", func(t *testing.T) {
		s := &target{}
		if err := applyOverrides(s, map[string]string{"host": "lower"}); err != nil {
			t.Fatal(err)
		}
		if s.Host != "lower" {
			t.Errorf("Host = %q, want %q", s.Host, "lower")
		}
	})

	t.Run("empty overrides is no-op", func(t *testing.T) {
		s := &target{Host: "original"}
		if err := applyOverrides(s, map[string]string{}); err != nil {
			t.Fatal(err)
		}
		if s.Host != "original" {
			t.Error("Host should not have changed")
		}
	})

	t.Run("nil pointer returns error", func(t *testing.T) {
		var s *target
		if err := applyOverrides(s, map[string]string{"Host": "x"}); err == nil {
			t.Error("expected error for nil pointer")
		}
	})

	t.Run("non-pointer returns error", func(t *testing.T) {
		s := target{}
		if err := applyOverrides(s, map[string]string{"Host": "x"}); err == nil {
			t.Error("expected error for non-pointer")
		}
	})

	t.Run("invalid int returns error", func(t *testing.T) {
		s := &target{}
		if err := applyOverrides(s, map[string]string{"Port": "not-a-number"}); err == nil {
			t.Error("expected error for invalid int")
		}
	})

	t.Run("unknown field is skipped", func(t *testing.T) {
		s := &target{Host: "original"}
		if err := applyOverrides(s, map[string]string{"NonExistent": "x"}); err != nil {
			t.Fatal(err)
		}
		if s.Host != "original" {
			t.Error("Host should not have changed")
		}
	})
}
