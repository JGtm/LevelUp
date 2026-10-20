package media

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"sync"
	"testing"
	"time"
)

// stubFFmpegSleeping remplace le lanceur par un processus d'aide qui dort ms millisecondes,
// quels que soient les arguments (le remux ajoute les siens). Rend le nombre de processus
// démarrés.
func stubFFmpegSleeping(t *testing.T, ms string) *int32Counter {
	t.Helper()
	c := &int32Counter{}
	prev := ffmpegCommand
	t.Cleanup(func() { ffmpegCommand = prev })
	ffmpegCommand = func(ctx context.Context, _ []string) *exec.Cmd {
		c.add()
		cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=TestFFmpegHelperProcess", "--", "sleep", ms)
		cmd.Env = append(os.Environ(), "FFMPEG_HELPER_PROCESS=1")
		return cmd
	}
	return c
}

type int32Counter struct {
	mu sync.Mutex
	n  int
}

func (c *int32Counter) add() { c.mu.Lock(); c.n++; c.mu.Unlock() }
func (c *int32Counter) get() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.n
}

func waitCount(t *testing.T, c *int32Counter, want int) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for c.get() != want {
		if time.Now().After(deadline) {
			t.Fatalf("processus démarrés = %d, attendu %d", c.get(), want)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// Deux lectures remuxées simultanées sont servies ensemble : la seconde n'attend pas la fin du
// flux de la première.
func TestStreamRemuxWebMPlan_TwoViewersAreServedTogether(t *testing.T) {
	started := stubFFmpegSleeping(t, "600")
	prev := remuxSlotWait
	remuxSlotWait = 100 * time.Millisecond
	t.Cleanup(func() { remuxSlotWait = prev })

	errs := make(chan error, 2)
	for range 2 {
		go func() {
			errs <- StreamRemuxWebMPlan(context.Background(), "clip.mkv", RemuxPlan{AudioMap: "0:a"}, &discardWriter{})
		}()
	}
	waitCount(t, started, 2) // les deux flux tournent en même temps, bien avant 600 ms
	for range 2 {
		if err := <-errs; err != nil {
			t.Fatalf("lecture remuxée : %v", err)
		}
	}
}

type discardWriter struct{}

func (*discardWriter) Write(p []byte) (int, error) { return len(p), nil }

// Un transcodage tient l'unique place des transcodages : une lecture remuxée ne l'attend pas.
func TestStreamRemuxWebMPlan_DoesNotWaitForATranscode(t *testing.T) {
	started := stubFFmpegSleeping(t, "1")
	prev := remuxSlotWait
	remuxSlotWait = 50 * time.Millisecond
	t.Cleanup(func() { remuxSlotWait = prev })

	release, err := ffmpegSlots.Acquire(context.Background()) // un transcodage en cours
	if err != nil {
		t.Fatal(err)
	}
	defer release()

	err = StreamRemuxWebMPlan(context.Background(), "clip.mkv", RemuxPlan{AudioMap: "0:a"}, &discardWriter{})
	if err != nil {
		t.Fatalf("la lecture remuxée a attendu le transcodage : %v", err)
	}
	if started.get() != 1 {
		t.Fatalf("processus démarrés = %d, attendu 1", started.get())
	}
}

// Les deux places de remux prises, un transcodage démarre sans attendre.
func TestRunFFmpeg_TranscodeDoesNotWaitForRemuxes(t *testing.T) {
	started := stubFFmpegSleeping(t, "1")
	var releases []func()
	for range 2 {
		release, err := remuxSlots.Acquire(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		releases = append(releases, release)
	}
	defer func() {
		for _, r := range releases {
			r()
		}
	}()

	err := RunFFmpeg(context.Background(), FFmpegJob{Args: []string{"x"}, MaxWait: 50 * time.Millisecond})
	if err != nil {
		t.Fatalf("le transcodage a attendu les lectures remuxées : %v", err)
	}
	if started.get() != 1 {
		t.Fatalf("processus démarrés = %d, attendu 1", started.get())
	}
}

// Le pool des remux a deux places : un troisième flux, qui n'attend que MaxWait, répond
// ErrFFmpegBusy sans démarrer de processus.
func TestRunFFmpeg_RemuxPoolHasTwoPlaces(t *testing.T) {
	started := stubFFmpegSleeping(t, "400")
	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := RunFFmpeg(context.Background(), FFmpegJob{Args: []string{"x"}, Pool: PoolRemux}); err != nil {
				t.Errorf("remux : %v", err)
			}
		}()
	}
	waitCount(t, started, 2)

	err := RunFFmpeg(context.Background(), FFmpegJob{Args: []string{"x"}, Pool: PoolRemux, MaxWait: 30 * time.Millisecond})
	if !errors.Is(err, ErrFFmpegBusy) {
		t.Fatalf("3e remux : err = %v, attendu ErrFFmpegBusy", err)
	}
	if started.get() != 2 {
		t.Fatalf("processus démarrés = %d, attendu 2", started.get())
	}
	wg.Wait()
	if remuxSlots.InUse() != 0 {
		t.Fatal("les places de remux n'ont pas été rendues")
	}
}

// Un pool inconnu ne lance rien : seuls les deux pools nommés peuvent tenir un ffmpeg.
func TestRunFFmpeg_UnknownPoolLaunchesNothing(t *testing.T) {
	started := stubFFmpegSleeping(t, "1")
	err := RunFFmpeg(context.Background(), FFmpegJob{Args: []string{"x"}, Pool: FFmpegPool(99)})
	if err == nil {
		t.Fatal("un pool inconnu doit être refusé")
	}
	if started.get() != 0 {
		t.Fatalf("processus démarrés = %d, attendu 0", started.get())
	}
}

// Le délai maximal de 45 min vaut pour les deux pools.
func TestRunFFmpeg_RemuxProcessIsBounded(t *testing.T) {
	_, lastCtx := stubFFmpeg(t)
	start := time.Now()
	if err := RunFFmpeg(context.Background(), FFmpegJob{Args: []string{"sleep", "1"}, Pool: PoolRemux}); err != nil {
		t.Fatal(err)
	}
	deadline, ok := lastCtx.Load().(context.Context).Deadline()
	if !ok {
		t.Fatal("le remux n'a pas de délai maximal")
	}
	if d := deadline.Sub(start); d < ffmpegMaxDuration-time.Minute || d > ffmpegMaxDuration+time.Minute {
		t.Fatalf("délai = %v, attendu ~%v", d, ffmpegMaxDuration)
	}
}
