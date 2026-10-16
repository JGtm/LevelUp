package media

import (
	"context"
	"fmt"
	"time"

	"levelup/go-api/internal/platform/slots"
)

// ffmpegMaxDuration borne un travail de transcodage, à partir de l'obtention de la place :
// un ffmpeg bloqué ne retient pas la file indéfiniment.
const ffmpegMaxDuration = 45 * time.Minute

// ffmpegSlots est le sémaphore global du transcodage : un seul ffmpeg à la fois pour tout
// le process (chacun est gourmand en processeur et en mémoire ; une salve d'envois ne doit
// pas en lancer des dizaines).
var ffmpegSlots = slots.New(1)

// withFFmpegSlot exécute fn en tenant l'unique place ffmpeg, avec un délai maximal
// ffmpegMaxDuration. Le contexte de l'appelant borne l'attente de la place ; la place est
// rendue au retour de fn.
func withFFmpegSlot(ctx context.Context, fn func(ctx context.Context) error) error {
	release, err := ffmpegSlots.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("attente de la place ffmpeg: %w", err)
	}
	defer release()
	runCtx, cancel := context.WithTimeout(ctx, ffmpegMaxDuration)
	defer cancel()
	return fn(runCtx)
}
