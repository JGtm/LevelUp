package media

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"time"

	"levelup/go-api/internal/platform/slots"
)

// ffmpegMaxDuration borne un travail ffmpeg, à partir de l'obtention de la place : un ffmpeg
// bloqué ne retient pas la file indéfiniment.
const ffmpegMaxDuration = 45 * time.Minute

// ErrFFmpegBusy : la place ffmpeg n'a pas été libérée dans le délai d'attente demandé
// (FFmpegJob.MaxWait). Aucun processus n'a été démarré : l'appelant peut répondre « occupé »
// sans rien avoir écrit.
var ErrFFmpegBusy = errors.New("ffmpeg: aucune place libre dans le délai d'attente")

// ffmpegSlots est le sémaphore global de ffmpeg : un seul processus à la fois pour tout le
// process (chacun est gourmand en processeur et en mémoire ; une salve d'envois ou de lectures
// ne doit pas en lancer des dizaines).
var ffmpegSlots = slots.New(1)

// slotHeldKey marque un contexte dont l'appelant tient déjà la place ffmpeg : un lancement
// sous ce contexte ne la redemande pas (la place n'est pas réentrante, un second Acquire
// attendrait indéfiniment son propre titulaire).
type slotHeldKey struct{}

// withFFmpegSlot exécute fn en tenant l'unique place ffmpeg, avec un délai maximal
// ffmpegMaxDuration. Le contexte de l'appelant borne l'attente de la place ; la place est
// rendue au retour de fn. Sous un contexte qui tient déjà la place, fn s'exécute directement.
func withFFmpegSlot(ctx context.Context, fn func(ctx context.Context) error) error {
	return withFFmpegSlotWithin(ctx, 0, fn)
}

// withFFmpegSlotWithin est withFFmpegSlot avec une attente de la place bornée à maxWait
// (0 : bornée par le seul contexte). Une attente épuisée rend une erreur qui enveloppe
// ErrFFmpegBusy ; l'annulation du contexte de l'appelant rend celle du contexte.
func withFFmpegSlotWithin(ctx context.Context, maxWait time.Duration, fn func(ctx context.Context) error) error {
	if ctx.Value(slotHeldKey{}) != nil {
		return fn(ctx)
	}
	waitCtx, cancelWait := ctx, context.CancelFunc(func() {})
	if maxWait > 0 {
		waitCtx, cancelWait = context.WithTimeout(ctx, maxWait)
	}
	release, err := ffmpegSlots.Acquire(waitCtx)
	cancelWait()
	if err != nil {
		if maxWait > 0 && ctx.Err() == nil {
			return fmt.Errorf("attente de la place ffmpeg: %w: %w", ErrFFmpegBusy, err)
		}
		return fmt.Errorf("attente de la place ffmpeg: %w", err)
	}
	defer release()
	runCtx, cancel := context.WithTimeout(context.WithValue(ctx, slotHeldKey{}, true), ffmpegMaxDuration)
	defer cancel()
	return fn(runCtx)
}

// FFmpegJob décrit un lancement de ffmpeg.
type FFmpegJob struct {
	Args   []string
	Stdout io.Writer // nil : sortie standard écartée
	Stderr io.Writer // nil : sortie d'erreur écartée
	// MaxWait borne l'attente de la place (0 : seul le contexte la borne). Un appelant qui
	// répond à une requête HTTP la borne pour répondre « occupé » plutôt que d'attendre un
	// transcodage de longue durée.
	MaxWait time.Duration
}

// ffmpegCommand construit la commande ffmpeg ; seul point du module qui lance ce binaire
// (garde-rail archlint/ffmpeg_single_launcher_test.go). Les tests le substituent.
var ffmpegCommand = func(ctx context.Context, args []string) *exec.Cmd {
	return exec.CommandContext(ctx, "ffmpeg", args...)
}

// RunFFmpeg lance ffmpeg et attend sa fin en tenant l'unique place ffmpeg du process, avec le
// délai maximal ffmpegMaxDuration : tout lancement de ffmpeg du module passe par ici. Le
// processus est tué à l'échéance ou à l'annulation du contexte.
func RunFFmpeg(ctx context.Context, job FFmpegJob) error {
	return withFFmpegSlotWithin(ctx, job.MaxWait, func(ctx context.Context) error {
		cmd := ffmpegCommand(ctx, job.Args)
		cmd.Stdout = job.Stdout
		cmd.Stderr = job.Stderr
		return cmd.Run()
	})
}
