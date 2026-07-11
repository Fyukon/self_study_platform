package app

import "testing"

func TestValidateNodeProgress(t *testing.T) {
	if err := validateNode("HTTP", "concept", "understood", 5, 0); err != nil {
		t.Fatalf("valid progress rejected: %v", err)
	}
	for _, test := range []struct {
		name       string
		status     string
		confidence int
	}{
		{name: "status", status: "done", confidence: 3},
		{name: "confidence low", status: "learning", confidence: 0},
		{name: "confidence high", status: "learning", confidence: 6},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := validateNode("HTTP", "concept", test.status, test.confidence, 0); err == nil {
				t.Fatal("invalid progress accepted")
			}
		})
	}
}
