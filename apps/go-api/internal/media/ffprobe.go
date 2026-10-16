package media

import (
	"context"
	"os/exec"
	"time"
)

// ffprobeMaxDuration borne une sonde : un fichier qui bloque ffprobe ne suspend pas la requête
// (ou le travail) qui l'attend. Variable pour que les tests la raccourcissent.
var ffprobeMaxDuration = 30 * time.Second

// ffprobeKillGrace : délai laissé à un ffprobe tué pour fermer ses sorties avant que Output
// ne rende la main.
const ffprobeKillGrace = 2 * time.Second

// ffprobeCommand construit la commande ffprobe ; seul point du module qui lance ce binaire
// (garde-rail archlint/ffprobe_single_launcher_test.go). Les tests le substituent.
var ffprobeCommand = func(ctx context.Context, args []string) *exec.Cmd {
	return exec.CommandContext(ctx, "ffprobe", args...)
}

// RunFFprobe lance ffprobe et rend sa sortie standard. Le processus est tué à l'annulation du
// contexte de l'appelant ou après ffprobeMaxDuration. Une sonde lit un en-tête et ne transcode
// rien : elle ne prend pas de place ffmpeg (elle peut donc s'exécuter sous une place tenue par
// BuildHLS sans interblocage).
func RunFFprobe(ctx context.Context, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, ffprobeMaxDuration)
	defer cancel()
	cmd := ffprobeCommand(ctx, args)
	cmd.WaitDelay = ffprobeKillGrace
	return cmd.Output()
}
