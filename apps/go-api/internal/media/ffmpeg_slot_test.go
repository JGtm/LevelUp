package media

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

// Un seul travail ffmpeg à la fois : un second attend le premier.
func TestWithFFmpegSlot_OneAtATime(t *testing.T) {
	var inFlight, peak atomic.Int32
	done := make(chan struct{}, 4)
	for range 4 {
		go func() {
			err := withFFmpegSlot(context.Background(), func(context.Context) error {
				n := inFlight.Add(1)
				for {
					p := peak.Load()
					if n <= p || peak.CompareAndSwap(p, n) {
						break
					}
				}
				time.Sleep(10 * time.Millisecond)
				inFlight.Add(-1)
				return nil
			})
			if err != nil {
				t.Errorf("withFFmpegSlot: %v", err)
			}
			done <- struct{}{}
		}()
	}
	for range 4 {
		<-done
	}
	if got := peak.Load(); got != 1 {
		t.Fatalf("travaux ffmpeg simultanés = %d, attendu 1", got)
	}
}

// Un appelant qui attend la place rend la main quand son contexte finit, sans lancer le travail.
func TestWithFFmpegSlot_WaitHonoursContext(t *testing.T) {
	release, err := ffmpegSlots.Acquire(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	ran := false
	err = withFFmpegSlot(ctx, func(context.Context) error { ran = true; return nil })
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v, attendu DeadlineExceeded", err)
	}
	if ran {
		t.Fatal("le travail ffmpeg a tourné sans place")
	}
}

// Le travail a un délai maximal, compté à partir de l'obtention de la place.
func TestWithFFmpegSlot_BoundsTheJobDuration(t *testing.T) {
	var deadline time.Time
	var has bool
	start := time.Now()
	if err := withFFmpegSlot(context.Background(), func(ctx context.Context) error {
		deadline, has = ctx.Deadline()
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if !has {
		t.Fatal("le travail ffmpeg n'a pas de délai maximal")
	}
	if d := deadline.Sub(start); d < ffmpegMaxDuration-time.Minute || d > ffmpegMaxDuration+time.Minute {
		t.Fatalf("délai = %v, attendu ~%v", d, ffmpegMaxDuration)
	}
	if ffmpegSlots.InUse() != 0 {
		t.Fatal("la place n'a pas été rendue")
	}
}
