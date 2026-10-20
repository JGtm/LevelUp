package signaux

import (
	"bytes"
	"slices"

	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// rafales_de_capture.go — les RAFALES DE CAPTURE de drapeau : une FRAME qui re-transmet la table
// de score-contribution CTF complete (les six tiers de l'echelle) accompagne une capture. C'est le
// seul signal de mode CTF qui vive DANS le film (RESEARCH_THEATER_RE.md §M-ter).

// Les tetes des 6 records de l'echelle de score-contribution CTF (la valeur gauche double ;
// prefixes constants inter-match). Une FRAME re-transmettant les 6 tiers = un burst de capture.
// Adapte de ctfcap. Des constantes et non une table de paquet : une table de grammaire est du
// code, pas un etat (`archlint/filmdec_package_vars_test.go`).
const (
	tierEchelle1 = "\xa4\x00\x00\x00"
	tierEchelle2 = "\x03\x48\x00\x00\x01"
	tierEchelle3 = "\x06\x90\x00\x00\x02"
	tierEchelle4 = "\x0d\x20\x00\x00\x05"
	tierEchelle5 = "\x1a\x40\x00\x00\x0a"
	tierEchelle6 = "\x34\x80\x00\x00\x15"
)

// captureMinTiers = nombre de tiers distincts requis pour qualifier un burst de
// capture CTF. tiers==6 = rock-solid (0 manque / 0 faux positif sur 4 matchs de
// ground-truth) ; tiers<6 sur-compte (RESEARCH_THEATER_RE.md §M-ter).
const captureMinTiers = 6

// CaptureBurstTimes rend les instants (ms, horloge du match) des BURSTS DE CAPTURE de drapeau du
// film, dans l'ordre du temps — une tranche vide, jamais nil, quand le film n'en porte aucun.
//
// POURQUOI LE BURST SUFFIT A DIRE LE MODE. L'artefact de rejeu 2D est construit HORS LIGNE, a
// partir des seuls chunks : il ne connait ni la carte ni le `game_variant_name`. Or publier le
// portage du drapeau exige de savoir qu'on est en CTF : la table d'emplacements de statistiques du
// drapeau lue sur un film d'un AUTRE mode rendrait des « prises » qui n'en sont pas. Le burst
// repond a la question sans base : c'est l'evenement de score qui accompagne une capture de
// drapeau, detecte a 6 tiers distincts — 0 manque et 0 faux positif sur les matchs de verite
// terrain (cf. [scanCaptureBursts]).
//
// CE QU'IL NE DIT PAS : une partie CTF ou personne ne capture n'en produit aucun. Le rejeu
// publie alors un calque de drapeau VIDE, et sa couverture le dit.
func CaptureBurstTimes(film *source.Film) []int {
	out := []int{}
	for _, c := range manifestChunks(film) {
		if c.meta.ChunkType != chunkTypeJeu {
			continue
		}
		out = append(out, scanCaptureBursts(framesOf(film, c.pos), c.meta.StartMS)...)
	}
	slices.Sort(out) // seul champ : des ex aequo sont indiscernables
	return out
}

// scanCaptureBursts detecte les bursts de capture CTF dans les FRAME d'un chunk gameplay
// (type 2) et rend leurs instants. startMS = start_ms du chunk (manifeste) ; l'us de chaque FRAME
// se convertit en ms match par le premier FRAME comme ancre (la premiere frame du chunk = etat
// complet, ignoree). Adapte de ctfcap detect.
func scanCaptureBursts(frames []types.Packet, startMS int) []int {
	if len(frames) == 0 {
		return nil
	}
	first := frames[0].TS
	var out []int
	for i, f := range frames {
		if i == 0 {
			continue // frame d'etat complet de debut de chunk
		}
		if countDistinctTiers(f.Payload) >= captureMinTiers {
			out = append(out, startMS+int(int64(f.TS-first)/1000))
		}
	}
	return out
}

// countDistinctTiers renvoie le nombre de tiers distincts presents dans payload.
func countDistinctTiers(pay []byte) int {
	distinct := 0
	for _, t := range [...]string{tierEchelle1, tierEchelle2, tierEchelle3, tierEchelle4,
		tierEchelle5, tierEchelle6} {
		if bytes.Contains(pay, []byte(t)) {
			distinct++
		}
	}
	return distinct
}
