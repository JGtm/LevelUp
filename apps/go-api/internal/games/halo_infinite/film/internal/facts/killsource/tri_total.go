package killsource

// tri_total.go — LES TRIS TOTAUX DE KILLSOURCE (lot J10.1, 2026-09-27, DT-9).
//
// Chacun de ces tris DECIDE D UNE SORTIE (l assistant d une mort, l epinglage d un bot, l ordre
// de la marche et des images-cles) et sa cle n etait pas prouvee unique : `sort.Slice` y tirait le
// rang des ex aequo. Chacun porte une chaine `cmp.Or` qui finit sur une cle unique, ou, quand aucun
// champ ne l est, un tri STABLE sur une entree deja dans l ordre du film — et le dit. Un test
// d ex aequo par tri (`tri_total_test.go`) ; le cliquet `archlint/film_tri_total_test.go` interdit
// tout nouvel appel `sort.Slice*` / `sort.Sort`.

import (
	"cmp"
	"slices"
)

// trierKillEvents range les kill-events dans l ORDRE TOTAL DU FILM (lot J10.1, 2026-09-27, DT-9) :
// instant, chunk, paquet, puis position de bit — unique dans un paquet. Plusieurs kill-events
// partagent un instant (un meme paquet en porte plusieurs) ; `pickAssistHit` retient le PREMIER
// porteur et `killEventsOu` rend les candidats dans cet ordre : l assistant publie en dependait.
func trierKillEvents(recs []killEventRec) {
	slices.SortFunc(recs, func(a, b killEventRec) int {
		return cmp.Or(cmp.Compare(a.ms, b.ms), cmp.Compare(a.chunk, b.chunk),
			cmp.Compare(a.pidx, b.pidx), cmp.Compare(a.bit, b.bit))
	})
}

// trierBotsParSlot range les bots par slot, les bots d un MEME slot dans leur ORDRE DE DECOUVERTE
// (lot J10.1, 2026-09-27, DT-9). C est cet ordre que [grammar.PaquetsBotMetadata] garantit et que
// [roster.pinBots] lit pour decider lequel nomme l indice au kill-feed ; `m.Bots` est bati dans
// l ordre de decouverte, et ce rang — la seule cle qui separe deux remplacants successifs d un
// slot — est conserve par le tri STABLE. Sous `sort.Slice`, il ne l etait que sous treize bots.
func trierBotsParSlot(bots []bot) {
	slices.SortStableFunc(bots, func(a, b bot) int { return cmp.Compare(a.Slot, b.Slot) })
}

// trierPaquetsT0 range les paquets de replication dans l ORDRE TOTAL DU FILM (lot J10.1, 2026-09-27,
// DT-9) : horodatage, puis chunk, puis rang dans le chunk — le couple (chunk, rang) est unique. Tout
// ce qui parcourt `f.t0` (la marche, le balayage, les kill-events) herite de cet ordre.
func trierPaquetsT0(t0 []packet) {
	slices.SortFunc(t0, func(a, b packet) int {
		return cmp.Or(cmp.Compare(a.ts, b.ts), cmp.Compare(a.chunk, b.chunk), cmp.Compare(a.idx, b.idx))
	})
}

// trierMortsDeLaMarche range les dead-states de la marche par instant, chunk et paquet, les ex aequo
// d un meme paquet dans l ORDRE DE LA MARCHE (lot J10.1, 2026-09-27, DT-9). Un record sans position
// enregistree porte `bit = -1` : il n a pas d autre cle unique que ce rang, et `dedup` garde le
// PREMIER de deux candidats de meme contenu — donc la position publiee.
func trierMortsDeLaMarche(deads []deadRecord) {
	slices.SortStableFunc(deads, func(a, b deadRecord) int {
		return cmp.Or(cmp.Compare(a.ms, b.ms), cmp.Compare(a.chunk, b.chunk), cmp.Compare(a.pidx, b.pidx))
	})
}

// trierImagesCles range les images-cles par horodatage, les ex aequo dans l ORDRE DU FILM (lot J10.1,
// 2026-09-27, DT-9) : [timeline.preload] retient la PREMIERE declaration de chaque slot, et deux
// images-cles de meme horodatage n ont pas d autre cle unique que leur rang dans le film.
func trierImagesCles(events []keyframeEvent) {
	slices.SortStableFunc(events, func(a, b keyframeEvent) int { return cmp.Compare(a.ts, b.ts) })
}
