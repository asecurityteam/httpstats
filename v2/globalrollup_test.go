package httpstats

import (
	"testing"
	"time"

	"github.com/golang/mock/gomock"
)

func TestGlobalRollupTagComputation(t *testing.T) {
	var ctrl = gomock.NewController(t)
	defer ctrl.Finish()

	var stat = NewMockXStater(ctrl)
	var globals = []string{"region", "az", "host", "container"}
	var statTags = []string{"region:test", "az:test", "host:test", "extra:value"}
	var expectedTagSets = [][]string{
		[]string{"region:test", "az:test", "host:test", "extra:value", "container:global"},
		[]string{"region:test", "az:test", "host:global", "extra:value", "container:global"},
		[]string{"region:test", "az:global", "host:global", "extra:value", "container:global"},
		[]string{"region:global", "az:global", "host:global", "extra:value", "container:global"},
	}
	var rollupClient = &rollupStatWrapper{globals: globals, Sender: stat}

	for _, expectedTagSet := range expectedTagSets {
		stat.EXPECT().Timing("stat", time.Duration(1), expectedTagSet[0], expectedTagSet[1], expectedTagSet[2], expectedTagSet[3], expectedTagSet[4])
		stat.EXPECT().Histogram("stat", 1.0, expectedTagSet[0], expectedTagSet[1], expectedTagSet[2], expectedTagSet[3], expectedTagSet[4])
	}
	rollupClient.Timing("stat", time.Duration(1), statTags...)
	rollupClient.Histogram("stat", 1.0, statTags...)
}
