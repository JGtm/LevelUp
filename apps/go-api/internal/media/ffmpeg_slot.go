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

// remuxPlaces : nombre de lectures remuxées simultanées. Un remux copie les flux sans
// transcoder (peu de processeur) mais dure autant que la lecture du clip : il ne partage pas la
// place des transcodages, que deux spectateurs d'un même clip se disputeraient.
const remuxPlaces = 2

// ErrFFmpegBusy : la place ffmpeg n'a pas été libérée dans le délai d'attente demandé
// (FFmpegJob.MaxWait). Aucun processus n'a été démarré : l'appelant peut répondre « occupé »
// sans rien avoir écrit.
var ErrFFmpegBusy = errors.New("ffmpeg: aucune place libre dans le délai d'attente")

// FFmpegPool nomme l'un des deux pools de places de ffmpeg. Tout lancement en tient une place :
// archlint/ffmpeg_single_launcher_test.go interdit tout autre lanceur et tout autre pool.
type FFmpegPool int

const (
	// PoolTranscode (valeur zéro) : transcodages HLS, analyses audio, miniatures, sondes de
	// version. Une place pour tout le process (chacun est gourmand en processeur et en mémoire ;
	// une salve d'envois ne doit pas en lancer des dizaines).
	PoolTranscode FFmpegPool = iota
	// PoolRemux : flux remuxés vers un client, remuxPlaces à la fois.
	PoolRemux
)

var (
	ffmpegSlots = slots.New(1)
	remuxSlots  = slots.New(remuxPlaces)
)

// slotsFor rend le sémaphore d'un pool nommé ; un pool inconnu n'a pas de place.
func slotsFor(pool FFmpegPool) (*slots.Pool, error) {
	switch pool {
	case PoolTranscode:
		return ffmpegSlots, nil
	case PoolRemux:
		return remuxSlots, nil
	}
	return nil, fmt.Errorf("ffmpeg: pool inconnu (%d)", int(pool))
}

// slotHeldKey marque un contexte dont l'appelant tient déjà une place du pool : un lancement
// sous ce contexte, pour le même pool, ne la redemande pas (la place n'est pas réentrante, un
// second Acquire attendrait indéfiniment son propre titulaire).
type slotHeldKey struct{ pool FFmpegPool }

// withFFmpegSlot exécute fn en tenant une place du pool des transcodages, avec un délai
// maximal ffmpegMaxDuration. Le contexte de l'appelant borne l'attente de la place ; la place
// est rendue au retour de fn. Sous un contexte qui tient déjà la place, fn s'exécute
// directement.
func withFFmpegSlot(ctx context.Context, fn func(ctx context.Context) error) error {
	return withFFmpegSlotWithin(ctx, PoolTranscode, 0, fn)
}

// withFFmpegSlotWithin est withFFmpegSlot pour un pool donné, avec une attente de la place
// bornée à maxWait (0 : bornée par le seul contexte). Une attente épuisée rend une erreur qui
// enveloppe ErrFFmpegBusy ; l'annulation du contexte de l'appelant rend celle du contexte.
func withFFmpegSlotWithin(ctx context.Context, pool FFmpegPool, maxWait time.Duration, fn func(ctx context.Context) error) error {
	places, err := slotsFor(pool)
	if err != nil {
		return err
	}
	if ctx.Value(slotHeldKey{pool}) != nil {
		return fn(ctx)
	}
	waitCtx, cancelWait := ctx, context.CancelFunc(func() {})
	if maxWait > 0 {
		waitCtx, cancelWait = context.WithTimeout(ctx, maxWait)
	}
	release, err := places.Acquire(waitCtx)
	cancelWait()
	if err != nil {
		if maxWait > 0 && ctx.Err() == nil {
			return fmt.Errorf("attente de la place ffmpeg: %w: %w", ErrFFmpegBusy, err)
		}
		return fmt.Errorf("attente de la place ffmpeg: %w", err)
	}
	defer release()
	runCtx, cancel := context.WithTimeout(context.WithValue(ctx, slotHeldKey{pool}, true), ffmpegMaxDuration)
	defer cancel()
	return fn(runCtx)
}

// FFmpegJob décrit un lancement de ffmpeg.
type FFmpegJob struct {
	Args   []string
	Stdout io.Writer // nil : sortie standard écartée
	Stderr io.Writer // nil : sortie d'erreur écartée
	// Pool choisit les places à tenir (PoolTranscode par défaut).
	Pool FFmpegPool
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

// RunFFmpeg lance ffmpeg et attend sa fin en tenant une place du pool demandé, avec le délai
// maximal ffmpegMaxDuration : tout lancement de ffmpeg du module passe par ici. Le processus est
// tué à l'échéance ou à l'annulation du contexte. Un pool inconnu est refusé sans lancement.
func RunFFmpeg(ctx context.Context, job FFmpegJob) error {
	return withFFmpegSlotWithin(ctx, job.Pool, job.MaxWait, func(ctx context.Context) error {
		cmd := ffmpegCommand(ctx, job.Args)
		cmd.Stdout = job.Stdout
		cmd.Stderr = job.Stderr
		return cmd.Run()
	})
}
