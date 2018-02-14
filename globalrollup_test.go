package stridestats

import (
	"fmt"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
)

func TestGlobalRollupTagComputation(t *testing.T) {
	var tagMap = map[string]string{
		"service": "test",
		"region":  "west1",
		"az":      "west1-a",
		"host":    "i-abcdefg",
	}
	var globals = []string{"region", "az", "host", "container"}
	var rollup = &rollupStatWrapper{tagMap: tagMap, globals: globals}
	rollup.computeTags()
	for offset, global := range globals {
		var found bool
		for _, tag := range rollup.rollups[offset] {
			if tag == fmt.Sprintf("%s:%s", global, globalName) {
				found = true
			}
		}
		if !found {
			t.Fatalf("invalid rollup: %v", rollup.rollups[offset])
		}
	}
}

func TestGlobalRollupMetricDuplication(t *testing.T) {
	var ctrl = gomock.NewController(t)
	defer ctrl.Finish()

	var stat = NewMockXStater(ctrl)
	var tagMap = map[string]string{
		"service": "test",
		"region":  "west1",
		"az":      "west1-a",
		"host":    "i-abcdefg",
	}
	var globals = []string{"region", "az", "host", "container"}
	var rollup = &rollupStatWrapper{tagMap: tagMap, globals: globals, Sender: stat}
	rollup.computeTags()
	stat.EXPECT().Timing("stat", time.Duration(1), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(len(globals))
	stat.EXPECT().Histogram("stat", 1.0, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(len(globals))
	rollup.Timing("stat", 1)
	rollup.Histogram("stat", 1.0)
}
