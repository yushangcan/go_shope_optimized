package redisstore

import "testing"

func TestAdmissionScriptContainsAtomicSteps(t *testing.T) {
	for _, part := range []string{"SISMEMBER", "HINCRBY", "XADD", "request_id", "SOLD_OUT"} {
		if !stringsContains(admissionScript, part) {
			t.Fatalf("admission script is missing %q", part)
		}
	}
}
