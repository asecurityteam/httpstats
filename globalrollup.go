package stridestats

import (
	"fmt"
	"time"

	"github.com/rs/xstats"
)

const globalName = "global"

type rollupStatWrapper struct {
	xstats.Sender
	tagMap  map[string]string
	globals []string
	rollups [][]string
}

func (s *rollupStatWrapper) computeTags() {
	var m = make(map[string]string, len(s.tagMap))
	for k, v := range s.tagMap {
		m[k] = v
	}
	for _, global := range s.globals {
		m[global] = globalName
	}
	s.rollups = append(s.rollups, s.computeTagSlice(m))
	for _, global := range s.globals {
		var original, found = s.tagMap[global]
		if !found {
			continue
		}
		m[global] = original
		s.rollups = append(s.rollups, s.computeTagSlice(m))
	}
}

func (s *rollupStatWrapper) computeTagSlice(values map[string]string) []string {
	var result = make([]string, 0, len(values))
	for k, v := range values {
		result = append(result, fmt.Sprintf("%s:%s", k, v))
	}
	return result
}

func (s *rollupStatWrapper) Gauge(stat string, value float64, tags ...string) {}
func (s *rollupStatWrapper) Count(stat string, value float64, tags ...string) {}
func (s *rollupStatWrapper) Histogram(stat string, value float64, tags ...string) {
	for _, rollup := range s.rollups {
		s.Sender.Histogram(stat, value, append(tags[:], rollup...)...)
	}
}
func (s *rollupStatWrapper) Timing(stat string, value time.Duration, tags ...string) {
	for _, rollup := range s.rollups {
		s.Sender.Timing(stat, value, append(tags[:], rollup...)...)
	}
}
