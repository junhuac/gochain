package perfutils

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"sync"
	"text/tabwriter"
	"time"
)

func NewPerfTimer() PerfTimer {
	return &PerfTimerNormal{}
}

// PerfTimer is a way to time pieces of code, in particular ones that happen many times,
// then get the metrics for it.
type PerfTimer interface {
	Start(sectionName string) PerfRun
	Print()
	Fprint(w io.Writer)
}

// PerfTimerNormal great name huh?
type PerfTimerNormal struct {
	// Sections map[string]*PerfSectionNormal
	Sections sync.Map
}

func (pt *PerfTimerNormal) Start(sectionName string) PerfRun {
	// ps := pt.Sections[sectionName]
	var ps *PerfSectionNormal
	psl, _ := pt.Sections.LoadOrStore(sectionName, &PerfSectionNormal{name: sectionName})
	ps = psl.(*PerfSectionNormal)
	return &PerfRunNormal{ps: ps, startTime: time.Now()}
}
func (pt *PerfTimerNormal) Print() {
	pt.Fprint(os.Stdout)
}
func (pt *PerfTimerNormal) Fprint(w io.Writer) {
	tw := tabwriter.NewWriter(w, 0, 8, 1, '\t', 0)
	fmt.Fprintln(tw, "Section\tCount\tTotal Dur\tAvg Dur")
	pt.Sections.Range(func(k, v interface{}) bool {
		ps := v.(*PerfSectionNormal)
		fmt.Fprintf(tw, "%v\t%v\t%v\t%v\n", k.(string), strconv.FormatInt(ps.count, 10), ps.totalDuration.String(), (ps.totalDuration / time.Duration(ps.count)).String())
		return true
	})
	tw.Flush()
}

type PerfSection interface {
	Name() string
	TotalDuration() time.Duration
	Count() int64
}

type PerfSectionNormal struct {
	name          string
	totalDuration time.Duration
	count         int64

	mutex sync.Mutex
}

func (ps *PerfSectionNormal) update(dur time.Duration) {
	ps.mutex.Lock()
	ps.totalDuration += dur
	ps.count++
	ps.mutex.Unlock()
}

func (ps *PerfSectionNormal) Name() string {
	return ps.name
}

func (ps *PerfSectionNormal) Count() int64 {
	return ps.count
}
func (ps *PerfSectionNormal) TotalDuration() time.Duration {
	return ps.totalDuration
}

// PerfRun keep time for each particular run
type PerfRun interface {
	Stop()
}
type PerfRunNormal struct {
	ps        *PerfSectionNormal
	startTime time.Time
}

func (pr *PerfRunNormal) Stop() {
	pr.ps.update(time.Since(pr.startTime))
}

type contextKey string

var (
	contextKeyPerfTimer = contextKey("perf-timer")
	defaultPerfTimer    = &NoopTimer{
		section: &NoopSection{},
	}
)

type NoopTimer struct {
	section *NoopSection
}

type NoopSection struct {
}

func (ps *NoopSection) Name() string {
	return "NONAME"
}

func (ps *NoopSection) Stop() {}

func (ps *NoopSection) Count() int64 {
	return 0
}
func (ps *NoopSection) TotalDuration() time.Duration {
	return 0
}

func (t *NoopTimer) Start(sectionName string) PerfRun {
	return t.section
}
func (t *NoopTimer) Print() {}

func (t *NoopTimer) Fprint(w io.Writer) {}

func GetTimer(ctx context.Context) PerfTimer {
	perfTimer, ok := ctx.Value(contextKeyPerfTimer).(PerfTimer)
	if ok {
		return perfTimer
	}
	return defaultPerfTimer
}

func WithTimer(ctx context.Context) context.Context {
	return context.WithValue(ctx, contextKeyPerfTimer, NewPerfTimer())
}