package zinetic

import "testing"

func TestPendingBackendContractOperationsAreClassified(t *testing.T) {
	if len(PendingBackendContractOperations) == 0 {
		t.Fatal("expected pending backend contract operations")
	}
	seen := map[string]bool{}
	for _, op := range PendingBackendContractOperations {
		if op.Service == "" || op.Operation == "" || op.Reason == "" {
			t.Fatalf("operation has empty field: %+v", op)
		}
		key := op.Service + "." + op.Operation
		if seen[key] {
			t.Fatalf("duplicate unsupported operation %s", key)
		}
		seen[key] = true
	}
}
