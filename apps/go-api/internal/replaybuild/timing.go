package replaybuild

// timing.go — LE CHRONOMETRE DES PHASES DE LA CUISSON (PLAN_CUISSON_PERF §3 D5).
//
// # POURQUOI MESURER ICI
//
// Une cuisson dure des minutes et personne ne savait de QUOI. `BuildBytes` enchaine quatre
// travaux de poids tres inegal — le second decodage du statborg (`readFilmStats`), le decodage
// de la source de degat (`decodeKillSource`), les trente et un balayages de `BuildFromFilm`,
// puis la serialisation — et chacun relit le film en entier. Sans ces quatre lignes, un profil
// CPU est le seul moyen de savoir laquelle a coute, ce qui suppose de savoir profiler avant de
// savoir quoi regarder.
//
// # POURQUOI UN HELPER PLUTOT QUE QUATRE `slog.Info` EN LIGNE
//
// Quatre phases, quatre fois les memes trois cles : a la troisieme copie la regle du depot
// impose un point unique (et la cle « duration » y devient impossible a oublier). Le helper
// prend l'instant de DEBUT et non une duree : l'appelant garde `time.Now()` sous les yeux, a
// la ligne qui precede le travail mesure.

import (
	"log/slog"
	"time"
)

// logPhase journalise la duree d'une phase de cuisson, depuis son instant de debut.
//
// slog.Info et non Debug : c'est la granularite qu'un operateur veut voir par defaut quand une
// cuisson traine — le detail par balayage, lui, est en Debug (cf. analysis/replay/observe.go).
func logPhase(phase, matchID string, debut time.Time) {
	slog.Info("cuisson: phase", "phase", phase, "match_id", matchID, "duration", time.Since(debut))
}
