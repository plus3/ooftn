package main

import (
	"fmt"
	"io"
	"runtime"
	"text/template"
	"time"
)

type Report struct {
	// Configuration
	Duration   time.Duration
	Entities   int
	Components int
	Systems    int

	// Results
	TotalUpdates   int64
	TotalTime      time.Duration
	UpdateTime     Stats
	GCPauseMetrics bool
	MemStatsStart  runtime.MemStats
	MemStatsEnd    runtime.MemStats
}

type Stats struct {
	Min     time.Duration
	Max     time.Duration
	Avg     time.Duration
	Samples []time.Duration
}

func (s *Stats) Finalize() {
	if len(s.Samples) == 0 {
		return
	}

	var total time.Duration
	s.Min = s.Samples[0]
	s.Max = s.Samples[0]

	for _, sample := range s.Samples {
		if sample < s.Min {
			s.Min = sample
		}
		if sample > s.Max {
			s.Max = sample
		}
		total += sample
	}
	s.Avg = total / time.Duration(len(s.Samples))
}

func (r *Report) Generate(w io.Writer) error {
	const reportTemplate = `
# ECS Stress Test Report

## Test Configuration
- **Run Duration:** {{.Duration}}
- **Initial Entities:** {{.Entities}}
- **Generated Components:** {{.Components}}
- **Generated Systems:** {{.Systems}}

## Performance Results
- **Total Updates:** {{.TotalUpdates}}
- **Total Test Time:** {{.TotalTime}}
- **Update Time (Frame):**
  - **Avg:** {{.UpdateTime.Avg}}
  - **Min:** {{.UpdateTime.Min}}
  - **Max:** {{.UpdateTime.Max}}

## Memory Usage (Raw Bytes)
- Heap Alloc:     {{.MemStatsStart.HeapAlloc}} (start) -> {{.MemStatsEnd.HeapAlloc}} (end) -> delta: {{bsub .MemStatsEnd.HeapAlloc .MemStatsStart.HeapAlloc}}
- Total Alloc:    {{.MemStatsStart.TotalAlloc}} (start) -> {{.MemStatsEnd.TotalAlloc}} (end) -> delta: {{bsub .MemStatsEnd.TotalAlloc .MemStatsStart.TotalAlloc}}
- Sys Memory:     {{.MemStatsStart.Sys}} (start) -> {{.MemStatsEnd.Sys}} (end) -> delta: {{bsub .MemStatsEnd.Sys .MemStatsStart.Sys}}
- Num GC:         {{.MemStatsStart.NumGC}} (start) -> {{.MemStatsEnd.NumGC}} (end) -> delta: {{usub .MemStatsEnd.NumGC .MemStatsStart.NumGC}}

{{if .GCPauseMetrics}}
## GC Pause Durations
- **Total GC Pause:** {{.MemStatsEnd.PauseTotalNs | ns}}
- **Num GC Cycles:** {{ usub .MemStatsEnd.NumGC .MemStatsStart.NumGC }}
{{end}}
`

	fm := template.FuncMap{
		"mb": func(v any) string {
			switch val := v.(type) {
			case uint64:
				return fmt.Sprintf("%.2f", float64(val)/1024/1024)
			case int64:
				return fmt.Sprintf("%.2f", float64(val)/1024/1024)
			default:
				return "N/A"
			}
		},
		"bsub": func(a, b uint64) int64 {
			return int64(a) - int64(b)
		},
		"usub": func(a, b uint32) uint32 {
			return a - b
		},
		"ns": func(ns uint64) string {
			return time.Duration(ns).String()
		},
	}

	tmpl, err := template.New("report").Funcs(fm).Parse(reportTemplate)
	if err != nil {
		return err
	}

	return tmpl.Execute(w, r)
}
