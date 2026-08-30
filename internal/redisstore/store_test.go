package redisstore

import "testing"

func TestAdmissionScriptContainsAtomicSteps(t *testing.T) {
	for _, part := range []string{"SISMEMBER", "HINCRBY", "request_id", "SOLD_OUT", "ACCEPTED"} {
		if !stringsContains(admissionScript, part) {
			t.Fatalf("admission script is missing %q", part)
		}
	}
}
