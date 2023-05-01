package kyc_test

import (
	"github.com/mergestat/kyc"
	"testing"
)

func TestBuildIndex(t *testing.T) {
	var fixture, _ = kyc.Open("../mergestat")

	index, err := fixture.ScanHead()
	if err != nil {
		t.Fatalf("failed to build index: %v", err)
	}

	t.Logf("index: %v", index)
}
