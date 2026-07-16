package media

// ffmpeg_args.go — préfixe commun des invocations ffmpeg du package.

// ffmpegQuietArgs retourne le préfixe commun des invocations ffmpeg du package
// (bannière masquée, logs limités aux erreurs), suivi des arguments fournis.
// Source unique du motif "-hide_banner -loglevel error" (règle ≤ 2 copies ;
// goconst est le garde-rail). Retourne un slice fraîchement alloué : les
// callers peuvent l'étendre par append sans risque d'aliasing. Ne concerne pas
// les probes ffprobe, qui utilisent un motif distinct ("-v error").
func ffmpegQuietArgs(extra ...string) []string {
	return append([]string{"-hide_banner", "-loglevel", "error"}, extra...)
}
