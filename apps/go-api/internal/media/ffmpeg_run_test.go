package media

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"sync/atomic"
	"testing"
	"time"
)

// TestFFmpegHelperProcess joue le rôle de ffmpeg dans les tests de RunFFmpeg : il n'est
// exécuté que relancé par stubFFmpeg (variable d'environnement FFMPEG_HELPER_PROCESS).
//
//	sleep <ms>  attend, puis sort 0
//	out <texte> écrit le texte sur la sortie standard
//	err <texte> écrit le texte sur la sortie d'erreur et sort 3
func TestFFmpegHelperProcess(t *testing.T) {
	if os.Getenv("FFMPEG_HELPER_PROCESS") != "1" {
		return
	}
	args := os.Args
	for len(args) > 0 && args[0] != "--" {
		args = args[1:]
	}
	args = args[1:]
	switch args[0] {
	case "sleep":
		ms, _ := strconv.Atoi(args[1])
		time.Sleep(time.Duration(ms) * time.Millisecond)
	case "out":
		fmt.Fprint(os.Stdout, args[1])
	case "err":
		fmt.Fprint(os.Stderr, args[1])
		os.Exit(3)
	}
	os.Exit(0)
}

// stubFFmpeg remplace le lanceur par le processus d'aide ; rend le nombre de processus
// démarrés et le contexte du dernier démarrage.
func stubFFmpeg(t *testing.T) (started *atomic.Int32, lastCtx *atomic.Value) {
	t.Helper()
	started, lastCtx = new(atomic.Int32), new(atomic.Value)
	prev := ffmpegCommand
	t.Cleanup(func() { ffmpegCommand = prev })
	ffmpegCommand = func(ctx context.Context, args []string) *exec.Cmd {
		started.Add(1)
		lastCtx.Store(ctx)
		cmd := exec.CommandContext(ctx, os.Args[0], append([]string{"-test.run=TestFFmpegHelperProcess", "--"}, args...)...)
		cmd.Env = append(os.Environ(), "FFMPEG_HELPER_PROCESS=1")
		return cmd
	}
	return started, lastCtx
}

func waitSlotInUse(t *testing.T, want int) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for ffmpegSlots.InUse() != want {
		if time.Now().After(deadline) {
			t.Fatalf("places ffmpeg tenues = %d, attendu %d", ffmpegSlots.InUse(), want)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// Un lancement tient l'unique place pendant toute la vie du processus ; un second, qui n'attend
// que MaxWait, répond ErrFFmpegBusy sans démarrer de processus.
func TestRunFFmpeg_HoldsTheSlotForTheProcessLifetime(t *testing.T) {
	started, _ := stubFFmpeg(t)
	first := make(chan error, 1)
	go func() {
		first <- RunFFmpeg(context.Background(), FFmpegJob{Args: []string{"sleep", "400"}})
	}()
	waitSlotInUse(t, 1)

	err := RunFFmpeg(context.Background(), FFmpegJob{Args: []string{"sleep", "1"}, MaxWait: 40 * time.Millisecond})
	if !errors.Is(err, ErrFFmpegBusy) {
		t.Fatalf("second lancement : err = %v, attendu ErrFFmpegBusy", err)
	}
	if got := started.Load(); got != 1 {
		t.Fatalf("processus démarrés = %d, attendu 1 (le second ne doit pas démarrer)", got)
	}
	if err := <-first; err != nil {
		t.Fatalf("premier lancement : %v", err)
	}
	if ffmpegSlots.InUse() != 0 {
		t.Fatal("la place n'a pas été rendue à la fin du processus")
	}
}

// Le processus a un délai maximal, compté à partir de l'obtention de la place.
func TestRunFFmpeg_BoundsTheProcessDuration(t *testing.T) {
	_, lastCtx := stubFFmpeg(t)
	start := time.Now()
	if err := RunFFmpeg(context.Background(), FFmpegJob{Args: []string{"sleep", "1"}}); err != nil {
		t.Fatal(err)
	}
	deadline, ok := lastCtx.Load().(context.Context).Deadline()
	if !ok {
		t.Fatal("le processus ffmpeg n'a pas de délai maximal")
	}
	if d := deadline.Sub(start); d < ffmpegMaxDuration-time.Minute || d > ffmpegMaxDuration+time.Minute {
		t.Fatalf("délai = %v, attendu ~%v", d, ffmpegMaxDuration)
	}
}

// Sous une place déjà tenue (BuildHLS sonde, analyse, puis transcode sous UNE place), un lancement
// ne reprend pas de place : pas d'interblocage, une seule place tenue.
func TestRunFFmpeg_UnderAHeldSlotDoesNotTakeASecond(t *testing.T) {
	stubFFmpeg(t)
	err := withFFmpegSlot(context.Background(), func(ctx context.Context) error {
		if err := RunFFmpeg(ctx, FFmpegJob{Args: []string{"sleep", "1"}, MaxWait: 20 * time.Millisecond}); err != nil {
			return err
		}
		if got := ffmpegSlots.InUse(); got != 1 {
			return fmt.Errorf("places tenues = %d, attendu 1", got)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if ffmpegSlots.InUse() != 0 {
		t.Fatal("la place n'a pas été rendue")
	}
}

// Les sorties sont relayées et un code de sortie non nul remonte en erreur.
func TestRunFFmpeg_RelaysOutputsAndExitStatus(t *testing.T) {
	stubFFmpeg(t)
	var out bytes.Buffer
	if err := RunFFmpeg(context.Background(), FFmpegJob{Args: []string{"out", "bonjour"}, Stdout: &out}); err != nil {
		t.Fatal(err)
	}
	if out.String() != "bonjour" {
		t.Fatalf("stdout = %q", out.String())
	}
	var errOut bytes.Buffer
	err := RunFFmpeg(context.Background(), FFmpegJob{Args: []string{"err", "boum"}, Stderr: &errOut})
	if err == nil {
		t.Fatal("un code de sortie 3 doit remonter en erreur")
	}
	if errOut.String() != "boum" {
		t.Fatalf("stderr = %q", errOut.String())
	}
	if errors.Is(err, ErrFFmpegBusy) {
		t.Fatal("un échec du processus n'est pas une file pleine")
	}
}

// Un remux qui n'obtient pas la place dans remuxSlotWait rend ErrFFmpegBusy sans démarrer de
// processus ni écrire un octet : l'appelant HTTP peut encore répondre 503.
func TestStreamRemuxWebMPlan_BusyWritesNothing(t *testing.T) {
	started, _ := stubFFmpeg(t)
	prev := remuxSlotWait
	remuxSlotWait = 30 * time.Millisecond
	t.Cleanup(func() { remuxSlotWait = prev })

	for range remuxPlaces {
		release, err := remuxSlots.Acquire(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		defer release()
	}

	var out bytes.Buffer
	err := StreamRemuxWebMPlan(context.Background(), "clip.mkv", RemuxPlan{AudioMap: "0:a"}, &out)
	if !errors.Is(err, ErrFFmpegBusy) {
		t.Fatalf("err = %v, attendu ErrFFmpegBusy", err)
	}
	if out.Len() != 0 || started.Load() != 0 {
		t.Fatalf("octets écrits = %d, processus démarrés = %d, attendu 0 et 0", out.Len(), started.Load())
	}
}
