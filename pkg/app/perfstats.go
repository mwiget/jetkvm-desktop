package app

import (
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/lkarlslund/jetkvm-desktop/pkg/logging"
)

const perfReportInterval = 5 * time.Second

// appPerfStats aggregates per-tick timings and logs them at debug level, to
// diagnose input latency on slower platforms. Update and Draw run on the same
// goroutine, so no locking is needed.
type appPerfStats struct {
	windowStart time.Time

	updates, draws, uploads, keyEvents  int
	updateTotal, drawTotal, uploadTotal time.Duration
	updateMax, drawMax, uploadMax       time.Duration
	keySendTotal, keySendMax            time.Duration
}

func trackMax(maxSeen *time.Duration, d time.Duration) {
	if d > *maxSeen {
		*maxSeen = d
	}
}

func (p *appPerfStats) trackUpdate(start time.Time) {
	d := time.Since(start)
	p.updates++
	p.updateTotal += d
	trackMax(&p.updateMax, d)
	p.maybeReport()
}

func (p *appPerfStats) trackDraw(start time.Time) {
	d := time.Since(start)
	p.draws++
	p.drawTotal += d
	trackMax(&p.drawMax, d)
}

func (p *appPerfStats) trackUpload(start time.Time) {
	d := time.Since(start)
	p.uploads++
	p.uploadTotal += d
	trackMax(&p.uploadMax, d)
}

func (p *appPerfStats) trackKeySend(start time.Time) {
	d := time.Since(start)
	p.keyEvents++
	p.keySendTotal += d
	trackMax(&p.keySendMax, d)
}

func average(total time.Duration, count int) time.Duration {
	if count == 0 {
		return 0
	}
	return total / time.Duration(count)
}

func (p *appPerfStats) maybeReport() {
	now := time.Now()
	if p.windowStart.IsZero() {
		p.windowStart = now
		return
	}
	if now.Sub(p.windowStart) < perfReportInterval {
		return
	}
	log := logging.Subsystem("perf")
	log.Debug().
		Float64("tps", ebiten.ActualTPS()).
		Float64("fps", ebiten.ActualFPS()).
		Dur("update_avg", average(p.updateTotal, p.updates)).
		Dur("update_max", p.updateMax).
		Dur("draw_avg", average(p.drawTotal, p.draws)).
		Dur("draw_max", p.drawMax).
		Int("uploads", p.uploads).
		Dur("upload_avg", average(p.uploadTotal, p.uploads)).
		Dur("upload_max", p.uploadMax).
		Int("key_events", p.keyEvents).
		Dur("key_send_avg", average(p.keySendTotal, p.keyEvents)).
		Dur("key_send_max", p.keySendMax).
		Msg("app performance")
	*p = appPerfStats{windowStart: now}
}
