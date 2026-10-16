package media

import (
	"context"
	"slices"
	"testing"
	"time"
)

// Un transcodage tient l'unique place des transcodages : une miniature (envoi d'un média,
// indexation) démarre sans l'attendre.
func TestRunWebPThumbnail_DoesNotWaitForATranscode(t *testing.T) {
	started := stubFFmpegSleeping(t, "1")
	release, err := ffmpegSlots.Acquire(context.Background()) // un transcodage en cours
	if err != nil {
		t.Fatal(err)
	}
	defer release()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := RunWebPThumbnail(ctx, "clip.mp4", "clip.webp", 5, 3, nil); err != nil {
		t.Fatalf("la miniature a attendu le transcodage : %v", err)
	}
	if started.get() != 1 {
		t.Fatalf("processus démarrés = %d, attendu 1", started.get())
	}
}

// Les arguments de la miniature : fenêtre demandée, vidéo source, WebP animé en sortie.
func TestThumbnailArgs(t *testing.T) {
	args := thumbnailArgs("in.mp4", "out.webp", 4.5, 3)
	for _, pair := range [][2]string{{"-ss", "4.50"}, {"-t", "3.00"}, {"-i", "in.mp4"}, {"-c:v", "libwebp"}} {
		i := slices.Index(args, pair[0])
		if i < 0 || i+1 >= len(args) || args[i+1] != pair[1] {
			t.Errorf("%s %s absent de %v", pair[0], pair[1], args)
		}
	}
	if args[len(args)-1] != "out.webp" {
		t.Errorf("sortie = %q, attendu out.webp", args[len(args)-1])
	}
}

// La place des miniatures prise, un transcodage démarre sans attendre.
func TestRunFFmpeg_TranscodeDoesNotWaitForAThumbnail(t *testing.T) {
	started := stubFFmpegSleeping(t, "1")
	release, err := thumbnailSlots.Acquire(context.Background()) // une miniature en cours
	if err != nil {
		t.Fatal(err)
	}
	defer release()

	err = RunFFmpeg(context.Background(), FFmpegJob{Args: []string{"x"}, MaxWait: 50 * time.Millisecond})
	if err != nil {
		t.Fatalf("le transcodage a attendu la miniature : %v", err)
	}
	if started.get() != 1 {
		t.Fatalf("processus démarrés = %d, attendu 1", started.get())
	}
}

// Une miniature est bornée par thumbnailMaxDuration, plus court que le délai d'un transcodage.
func TestRunWebPThumbnail_IsBoundedByItsOwnMaxDuration(t *testing.T) {
	_, lastCtx := stubFFmpeg(t)
	start := time.Now()
	if err := RunWebPThumbnail(context.Background(), "clip.mp4", "clip.webp", 5, 3, nil); err != nil {
		t.Fatal(err)
	}
	deadline, ok := lastCtx.Load().(context.Context).Deadline()
	if !ok {
		t.Fatal("la miniature n'a pas de délai maximal")
	}
	if d := deadline.Sub(start); d < thumbnailMaxDuration-time.Minute || d > thumbnailMaxDuration+time.Minute {
		t.Fatalf("délai = %v, attendu ~%v", d, thumbnailMaxDuration)
	}
	if thumbnailMaxDuration >= ffmpegMaxDuration {
		t.Fatalf("thumbnailMaxDuration (%v) doit rester sous ffmpegMaxDuration (%v)", thumbnailMaxDuration, ffmpegMaxDuration)
	}
}
