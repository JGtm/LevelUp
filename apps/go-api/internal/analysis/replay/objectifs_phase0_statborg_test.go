package replay

// objectifs_phase0_statborg_test.go — LE PONT STATBORG -> JOUEUR, SUR FILMS REELS.
//
// CE FICHIER NE CONTIENT PLUS DE PONT : il APPELLE celui de la production
// (`objectiveevents.SlotIdentityFromDeaths` / `SlotIdentityResolved`, slotidentity_deaths.go),
// porte le 2026-08-18 a l'item 1.0(a) du plan. Une seconde copie aurait diverge au premier
// correctif, et surtout la mesure ne dirait plus rien de ce que l'artefact publie.
//
// CE QUE LE PONT PAR INSTANTS CORRIGE. `objectiveevents.SlotIdentity` apparie un slot statborg
// a une ligne de match par le triplet (frags, morts, assistances). C'est exact quand les
// compteurs du film atteignent leurs valeurs finales — et c'est mesure ici que ce n'est PAS
// toujours le cas : sur `64e8adfa` et `24dbb67d`, l'appariement rend 0 slot sur 8, alors qu'il
// en rend 8 sur 8 sur `530820e5` et `53ce4390`. Un film tronque (le Theater ne rend pas
// toujours la fin) suffit a le mettre en defaut.
//
// LA VOIE DE REMPLACEMENT N'EMPRUNTE RIEN A L'API : les MORTS. Le statborg replique le
// compteur de morts de chaque joueur (`comp 2 B`) et le film porte par ailleurs son FIL DES
// MORTS, qui date chaque mort et NOMME sa victime. Les deux sont sur l'horloge du MATCH.
//
// CE QUE CE TEST CONTROLE : que les deux ponts, TOTALEMENT DISJOINTS, ne se contredisent
// jamais la ou tous deux repondent — c'est ce qui donne le droit d'employer le second la ou le
// premier se tait.

import (
	"sort"
	"strconv"
	"testing"

	"levelup/go-api/internal/analysis/objectiveevents"
)

// Emplacements de base du statborg, pour le seul DIAGNOSTIC des triplets ci-dessous (les
// memes valeurs que `objectiveevents`, non exportees mais etablies : frags en A et morts en B
// de comp 2, assistances en A de comp 3).
const (
	objCompCoreKills   = 2
	objCompCoreAssists = 3
)

// objDeathInstants traduit le fil des morts du rejeu dans la forme qu'attend le pont de
// production.
func objDeathInstants(deaths []Death) []objectiveevents.DeathInstant {
	out := make([]objectiveevents.DeathInstant, 0, len(deaths))
	for _, d := range deaths {
		out = append(out, objectiveevents.DeathInstant{
			XUID: strconv.FormatUint(d.XUID, 10), TimeMS: int(d.TimeMS),
		})
	}
	return out
}

// objIdentites rend le pont slot statborg -> xuid par les INSTANTS DE MORT, tel que la
// PRODUCTION le calcule.
func objIdentites(src objectiveevents.FilmSource, deaths []Death) map[int]string {
	return objectiveevents.SlotIdentityFromDeaths(src, objDeathInstants(deaths))
}

// objTriplets rend, par slot statborg, le triplet final (frags, morts, assistances).
// Diagnostic : c'est lui qui montre POURQUOI l'appariement par totaux echoue sur un film.
func objTriplets(recs []objectiveevents.StatRecord) map[int][3]int64 {
	out := map[int][3]int64{}
	for _, r := range recs {
		if objectiveevents.IsTeamSlot(r.Slot) {
			continue
		}
		tri := out[r.Slot]
		if v, ok := r.Comps[objCompCoreKills]; ok {
			tri[0], tri[1] = objMax64(tri[0], v.A), objMax64(tri[1], v.B)
		}
		if v, ok := r.Comps[objCompCoreAssists]; ok {
			tri[2] = objMax64(tri[2], v.A)
		}
		out[r.Slot] = tri
	}
	return out
}

// objMax64 rend le plus grand des deux entiers (les emissions negatives sont des ancrages
// parasites : elles ne peuvent pas gagner).
func objMax64(a, b int64) int64 {
	if b > a {
		return b
	}
	return a
}

// TestObjectifsPhase0PontStatborg — le CONTROLE du pont de remplacement, et sa confrontation
// au pont par triplets la ou celui-ci fonctionne.
//
// Deux ponts totalement disjoints (des TOTAUX venus de l'API contre des INSTANTS venus du
// film) qui rendent la meme table sur les films ou les deux repondent : c'est ce qui donne
// le droit d'employer le second la ou le premier se tait.
func TestObjectifsPhase0PontStatborg(t *testing.T) {
	root := objRequireRoot(t)
	joues := 0
	for _, id := range append(append([]string{}, objCTFFilms...), objBallFilm) {
		f := objCorpus[id]
		src, ok := objOpenFilm(t, root, id)
		if !ok {
			continue
		}
		joues++
		objCheckTripletsUniques(t, f)
		b := objBridgeOf(t, root, id)
		lines := objPlayerLines(f)
		parInstants := objIdentites(src, b.Deaths)
		parTriplets := objectiveevents.SlotIdentity(src, lines)
		resolu, st := objectiveevents.SlotIdentityResolved(src, lines, objDeathInstants(b.Deaths))
		accords, desaccords := objCompareTables(parInstants, parTriplets)
		t.Logf("%s : pont par INSTANTS %d apparies ; pont par TRIPLETS %d apparies ; "+
			"recoupement %d accords / %d desaccords ; PONT RESOLU (production) %d apparies, "+
			"voie %q, %d desaccords ecartes",
			id, len(parInstants), len(parTriplets), accords, desaccords,
			len(resolu), st.Source, st.Conflicts)
		for slot, tri := range objTriplets(objectiveevents.StatRecords(src)) {
			t.Logf("%s : slot %d — film (frags %d, morts %d, assist %d)", id, slot, tri[0], tri[1], tri[2])
		}
		if desaccords > 0 {
			t.Errorf("%s : %d slots ou les deux ponts se contredisent — aucun des deux n'est employable",
				id, desaccords)
		}
		if len(resolu) < len(parTriplets) {
			t.Errorf("%s : le pont RESOLU nomme %d slots la ou celui par triplets en nomme %d — "+
				"le repli degrade l'existant", id, len(resolu), len(parTriplets))
		}
	}
	if joues == 0 {
		t.Skipf("aucun film du corpus dans le cache (%s=%q)", objFilmEnv, root)
	}
}

// objPlayerLines rend les lignes de match gelees du corpus dans la forme qu'attend le pont
// par totaux.
func objPlayerLines(f objFilm) []objectiveevents.PlayerLine {
	out := make([]objectiveevents.PlayerLine, 0, len(f.Players))
	for _, p := range f.Players {
		out = append(out, objectiveevents.PlayerLine{
			XUID: p.XUID, Kills: p.Kills, Deaths: p.Deaths, Assists: p.Assists,
		})
	}
	return out
}

// objCompareTables compte les slots ou les deux ponts disent la meme chose, et ceux ou ils
// se contredisent (un slot absent d'une table n'est ni l'un ni l'autre).
func objCompareTables(instants, triplets map[int]string) (accords, desaccords int) {
	slots := make([]int, 0, len(instants))
	for slot := range instants {
		slots = append(slots, slot)
	}
	sort.Ints(slots)
	for _, slot := range slots {
		other, ok := triplets[slot]
		if !ok {
			continue
		}
		if other == instants[slot] {
			accords++
		} else {
			desaccords++
		}
	}
	return accords, desaccords
}
