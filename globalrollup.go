package httpstats

import (
	"fmt"
	"strings"
	"time"

	"github.com/rs/xstats"
)

const globalName = "global"

type rollupStatWrapper struct {
	xstats.Sender
	globals []string
}

// computeTags is intended to return a slice of tag sets that represent all
// of the forms of a metric that should be emitted for a proper rollup. The
// order in which the global rollup tags are defined is significant. The
// expected behaviour is that this function will produce a tag set in which all
// possible global keys are set to the global token and subsequent tag sets
// in which the global mask is removed in favour of the original value
// iteratively, from "left to right" according to the global tag list. If
// there is a global tag defined that does not exist in the input then the
// global tag value will be inserted for that key in all tag sets.
//
// For example:
// GIVEN: globals = []string{"region", "az", "host", "container"}
// GIVEN: inputTags = []string{"region:us-west2", "az:a", "host:1234", "myTag:myValue"}
// EXPECTED: results = [][]string{
//												[]string{"region:global", "az:global", "host:global", "container:global", "myTag:myValue"},
//												[]string{"region:us-west2", "az:global", "host:global", "container:global", "myTag:myValue"},
//												[]string{"region:us-west2", "az:a", "host:global", "container:global", "myTag:myValue"},
//												[]string{"region:us-west2", "az:a", "host:1234", "container:global", "myTag:myValue"},
//										 }
func (s *rollupStatWrapper) computeTags(inputTags []string) [][]string {
	var output = make([][]string, 0, len(s.globals))
	// Populate any missing global values.
	for _, global := range s.globals {
		var found = false
		for _, tag := range inputTags {
			if strings.HasPrefix(tag, global+":") {
				found = true
				break
			}
		}
		if !found {
			inputTags = append(inputTags, fmt.Sprintf("%s:%s", global, globalName))
		}
	}

	for x := len(s.globals) - 1; x >= 0; x = x - 1 {
		var tagSet = make([]string, len(inputTags))
		copy(tagSet, inputTags)
		for offset, tag := range tagSet {
			for y := x; y < len(s.globals); y = y + 1 {
				if strings.HasPrefix(tag, s.globals[y]+":") {
					tagSet[offset] = fmt.Sprintf("%s:%s", s.globals[y], globalName)
				}
			}
		}
		output = append(output, tagSet)
	}

	return output
}

func (s *rollupStatWrapper) Gauge(stat string, value float64, tags ...string) {}
func (s *rollupStatWrapper) Count(stat string, value float64, tags ...string) {}
func (s *rollupStatWrapper) Histogram(stat string, value float64, tags ...string) {
	for _, rollup := range s.computeTags(tags) {
		s.Sender.Histogram(stat, value, rollup...)
	}
}
func (s *rollupStatWrapper) Timing(stat string, value time.Duration, tags ...string) {
	for _, rollup := range s.computeTags(tags) {
		s.Sender.Timing(stat, value, rollup...)
	}
}
