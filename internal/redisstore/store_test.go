package redisstore

import (
	"strings"
	"testing"
)

func TestAdmissionScriptContainsAtomicSteps(t *testing.T) {
	for _, part := range []string{"SISMEMBER", "HINCRBY", "request_id", "SOLD_OUT", "ACCEPTED"} {
		if !strings.Contains(admissionScript, part) {
			t.Fatalf("admission script is missing %q", part)
		}
	}
}
