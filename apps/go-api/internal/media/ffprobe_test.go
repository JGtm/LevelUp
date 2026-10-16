package media

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"testing"
	"time"
)

// stubFFprobe remplace le lanceur de ffprobe par le processus d'aide de TestFFmpegHelperProcess
// (sleep <ms> | out <texte>) ; rend le contexte du dernier démarrage.
func stubFFprobe(t *testing.T, helperArgs ...string) (lastCtx *context.Context) {
	t.Helper()
	lastCtx = new(context.Context)
	prev := ffprobeCommand
	t.Cleanup(func() { ffprobeCommand = prev })
	ffprobeCommand = func(ctx context.Context, _ []string) *exec.Cmd {
		*lastCtx = ctx
		cmd := exec.CommandContext(ctx, os.Args[0], append([]string{"-test.run=TestFFmpegHelperProcess", "--"}, helperArgs...)...)
		cmd.Env = append(os.Environ(), "FFMPEG_HELPER_PROCESS=1")
		return cmd
	}
	return lastCtx
}

// Une sonde qui ne répond pas est interrompue au délai maximal : la requête qui l'attend ne reste
// pas suspendue à un fichier qui bloque ffprobe.
func TestRunFFprobe_StopsAtTheMaxDuration(t *testing.T) {
	stubFFprobe(t, "sleep", "20000")
	prev := ffprobeMaxDuration
	ffprobeMaxDuration = 100 * time.Millisecond
	t.Cleanup(func() { ffprobeMaxDuration = prev })

	start := time.Now()
	_, err := RunFFprobe(context.Background(), "-v", "error", "clip.mkv")
	if err == nil {
		t.Fatal("une sonde qui dépasse le délai doit échouer")
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Fatalf("la sonde n'a pas été interrompue au délai : %v", elapsed)
	}
}

// Le délai est posé sur le contexte du processus, sans prise de place.
func TestRunFFprobe_BoundsTheContextAndTakesNoSlot(t *testing.T) {
	lastCtx := stubFFprobe(t, "out", "{}")
	release, err := ffmpegSlots.Acquire(context.Background()) // transcodage en cours
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	start := time.Now()
	out, err := RunFFprobe(context.Background(), "-v", "error", "clip.mkv")
	if err != nil {
		t.Fatalf("la sonde a attendu la place ffmpeg : %v", err)
	}
	if string(out) != "{}" {
		t.Fatalf("sortie = %q", out)
	}
	deadline, ok := (*lastCtx).Deadline()
	if !ok {
		t.Fatal("la sonde n'a pas de délai maximal")
	}
	if d := deadline.Sub(start); d < ffprobeMaxDuration-5*time.Second || d > ffprobeMaxDuration+5*time.Second {
		t.Fatalf("délai = %v, attendu ~%v", d, ffprobeMaxDuration)
	}
}

// L'annulation du contexte de l'appelant interrompt la sonde, comme avant.
func TestRunFFprobe_HonoursCallerCancellation(t *testing.T) {
	stubFFprobe(t, "sleep", "20000")
	ctx, cancel := context.WithTimeout(context.Background(), 80*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err := RunFFprobe(ctx, "clip.mkv")
	if err == nil {
		t.Fatal("la sonde annulée doit échouer")
	}
	if errors.Is(err, ErrFFmpegBusy) {
		t.Fatal("une sonde annulée n'est pas une file pleine")
	}
	if time.Since(start) > 5*time.Second {
		t.Fatal("la sonde n'a pas suivi l'annulation du contexte")
	}
}
