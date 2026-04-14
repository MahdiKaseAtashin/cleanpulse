package devcleanup

import "testing"

func TestParseRiskLevel(t *testing.T) {
	if got := ParseRiskLevel("safe"); got != RiskSafe {
		t.Fatalf("expected safe risk, got %v", got)
	}
	if got := ParseRiskLevel("moderate"); got != RiskModerate {
		t.Fatalf("expected moderate risk, got %v", got)
	}
	if got := ParseRiskLevel("aggressive"); got != RiskAggressive {
		t.Fatalf("expected aggressive risk, got %v", got)
	}
	if got := ParseRiskLevel("unknown"); got != RiskSafe {
		t.Fatalf("expected unknown fallback to safe, got %v", got)
	}
}

func TestIsSafeCleanupPath(t *testing.T) {
	cases := []struct {
		path string
		safe bool
	}{
		{path: ".", safe: false},
		{path: `C:\`, safe: false},
		{path: `/`, safe: false},
		{path: `/tmp/dev-cache`, safe: true},
		{path: `C:\Users\dev\AppData\Local\Temp`, safe: true},
	}
	for _, tc := range cases {
		if got := isSafeCleanupPath(tc.path); got != tc.safe {
			t.Fatalf("path %q expected safe=%t got=%t", tc.path, tc.safe, got)
		}
	}
}
