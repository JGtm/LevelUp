package replay

// objectifs_phase0_verdict_test.go — L'ASSEMBLAGE DE L'ITEM 0.1 ET SON VERDICT.
//
// SEUILS, ECRITS AVANT LA MESURE (plan §Phase 0, item 0.1) et JAMAIS rebaisses :
//
//	un motif de 32 bits, LE MEME sur les trois films CTF ;
//	present dans >= 90 % des fenetres de portage qui contiennent au moins une image-cle ;
//	temoin (un slot NON porteur, memes instants, meme code) <= 5 %.
//
// Les denominateurs sont publies avec chaque taux : un pourcentage sans son denominateur
// n'est pas une mesure.

import (
	"sort"
	"testing"

	"levelup/go-api/internal/analysis/objectiveevents"
	"levelup/go-api/internal/analysis/weaponv3"
)

// objSeuilFenetres / objSeuilTemoin — les deux seuils du plan.
const (
	objSeuilFenetres = 0.90
	objSeuilTemoin   = 0.05
)

// objCtx porte tout ce qu'un film a produit, pour que la seconde passe (l'intersection) ne
// re-balaye rien.
type objCtx struct {
	ID    string
	Pont  objBridge
	Recs  []objRecord
	Wins  []objWindow
	Tab   objTable
	Cands []objCandidat
}

// TestObjectifsPhase0FamilleDrapeau — ITEM 0.1, sur les trois films CTF.
func TestObjectifsPhase0FamilleDrapeau(t *testing.T) {
	root := objRequireRoot(t)
	var ctxs []objCtx
	for _, id := range objCTFFilms {
		src, ok := objOpenFilm(t, root, id)
		if !ok {
			t.Logf("film %s absent du cache — non confronte", id)
			continue
		}
		ctxs = append(ctxs, objMesureFilm(t, root, id, src))
	}
	if len(ctxs) == 0 {
		t.Skipf("aucun film CTF dans le cache (%s=%q)", objFilmEnv, root)
	}
	objVerdict01(t, ctxs)
}

// objMesureFilm joue la confrontation d'un film et publie ses chiffres.
func objMesureFilm(t *testing.T, root, id string, src *objDiskFilm) objCtx {
	t.Helper()
	f := objCorpus[id]
	b := objBridgeOf(t, root, id)
	identity := objIdentites(src, b.Deaths)
	evs := objectiveevents.IdentifyNamedEvents(objectiveevents.NamedEvents(src, f.Mode), identity)
	recs, images := objRecordsOf(t, root, id)
	wins, fusions := objPortageWindows(evs, b.Deaths, objFinMatch(evs, b.Deaths))
	tab := objConfronte(recs, b, wins)
	cands := objCandidats(tab)
	avec, total, moy := objControlePositif(recs)
	t.Logf("%s : %d images-cles, %d records bipede (%d sans pont), %d slots statborg apparies, "+
		"%d fenetres de portage (%d prises fusionnees) ; records ETIQUETES portage=%d hors=%d ; "+
		"valeurs distinctes=%d ; candidates=%d",
		id, images, len(recs), tab.SlotsInconnus, len(identity), len(wins), fusions,
		tab.Portage, tab.Hors, len(tab.Par), len(cands))
	t.Logf("%s : CONTROLE POSITIF — %d/%d records (%.1f %%) portent au moins une famille d'arme "+
		"CONNUE, %.2f familles connues par record en moyenne",
		id, avec, total, 100*float64(avec)/float64(max(total, 1)), moy)
	objLogFamillesConnues(t, id, tab)
	objLogMotifs(t, id, tab, cands)
	return objCtx{ID: id, Pont: b, Recs: recs, Wins: wins, Tab: tab, Cands: cands}
}

// objLogFamillesConnues publie le comportement des familles d'arme CONNUES de part et
// d'autre de la frontiere de portage — c'est ce qui separe « le drapeau » de « le porteur a
// change d'arme ».
func objLogFamillesConnues(t *testing.T, id string, tab objTable) {
	t.Helper()
	connues := objFamillesConnuesParCamp(tab)
	for i, c := range connues {
		if i >= 6 {
			t.Logf("%s : ... %d autres familles connues", id, len(connues)-6)
			break
		}
		t.Logf("%s : famille CONNUE 0x%08X (%s) — portage %d/%d = %.1f %% ; hors %d/%d = %.1f %%",
			id, c.Val, weaponv3.WeaponName(c.Val), c.Portage, tab.Portage, 100*c.TauxP,
			c.Hors, tab.Hors, 100*c.TauxH)
	}
}

// objLogMotifs publie les candidates INCONNUES du catalogue, apres repli des vues decalees.
func objLogMotifs(t *testing.T, id string, tab objTable, cands []objCandidat) {
	t.Helper()
	vals := make([]uint32, 0, len(cands))
	taux := make(map[uint32]objCandidat, len(cands))
	for _, c := range cands {
		vals = append(vals, c.Val)
		taux[c.Val] = c
	}
	groupes := objGroupesDecalage(vals)
	racines := make([]uint32, 0, len(groupes))
	for r := range groupes {
		racines = append(racines, r)
	}
	sort.Slice(racines, func(i, j int) bool {
		if taux[racines[i]].TauxP != taux[racines[j]].TauxP {
			return taux[racines[i]].TauxP > taux[racines[j]].TauxP
		}
		return racines[i] < racines[j]
	})
	t.Logf("%s : %d candidates -> %d MOTIFS distincts apres repli des vues decalees",
		id, len(cands), len(groupes))
	for i, r := range racines {
		if i >= 6 {
			t.Logf("%s : ... %d autres motifs", id, len(racines)-6)
			break
		}
		c := taux[r]
		t.Logf("%s : motif racine 0x%08X (%d vues decalees) — portage %d/%d = %.1f %% ; "+
			"hors %d/%d = %.2f %%", id, r, len(groupes[r]), c.Portage, tab.Portage,
			100*c.TauxP, c.Hors, tab.Hors, 100*c.TauxH)
	}
}

// objVerdict01 intersecte les candidates des films, replie les decalages, puis applique les
// deux seuils du plan sur les FENETRES.
func objVerdict01(t *testing.T, ctxs []objCtx) {
	t.Helper()
	compte := map[uint32]int{}
	for _, c := range ctxs {
		for _, cd := range c.Cands {
			compte[cd.Val]++
		}
	}
	var communs []uint32
	for v, n := range compte {
		if n == len(ctxs) {
			communs = append(communs, v)
		}
	}
	sort.Slice(communs, func(i, j int) bool { return communs[i] < communs[j] })
	groupes := objGroupesDecalage(communs)
	t.Logf("ITEM 0.1 — %d films confrontes ; %d valeurs communes aux %d films, soit %d MOTIFS "+
		"distincts : %s", len(ctxs), len(communs), len(ctxs), len(groupes), objHex(communs))
	racines := make([]uint32, 0, len(groupes))
	for r := range groupes {
		racines = append(racines, r)
	}
	sort.Slice(racines, func(i, j int) bool { return racines[i] < racines[j] })
	tenus := 0
	for _, r := range racines {
		if objVerdictMotif(t, ctxs, r) {
			tenus++
		}
	}
	t.Logf("ITEM 0.1 — VERDICT : %d motif(s) sur %d tiennent les deux seuils "+
		"(fenetres >= %.0f %%, temoin <= %.0f %%)",
		tenus, len(racines), 100*objSeuilFenetres, 100*objSeuilTemoin)
}

// objVerdictMotif applique les seuils du plan a UN motif, film par film puis en cumul.
func objVerdictMotif(t *testing.T, ctxs []objCtx, val uint32) bool {
	t.Helper()
	var cum objFenetreStat
	for _, c := range ctxs {
		st := objStatFenetres(c.Recs, c.Pont, c.Wins, val)
		cum.AvecKF += st.AvecKF
		cum.Touchees += st.Touchees
		cum.TemoinKF += st.TemoinKF
		cum.TemoinTouchees += st.TemoinTouchees
		t.Logf("motif 0x%08X sur %s : fenetres avec image-cle %d, portees %d (%.1f %%) ; "+
			"temoin %d/%d (%.1f %%)", val, c.ID, st.AvecKF, st.Touchees,
			100*objPart(st.Touchees, st.AvecKF), st.TemoinTouchees, st.TemoinKF,
			100*objPart(st.TemoinTouchees, st.TemoinKF))
	}
	tf, tt := objPart(cum.Touchees, cum.AvecKF), objPart(cum.TemoinTouchees, cum.TemoinKF)
	ok := cum.AvecKF > 0 && tf >= objSeuilFenetres && tt <= objSeuilTemoin
	t.Logf("motif 0x%08X CUMUL : %d/%d fenetres = %.1f %% (seuil %.0f %%) ; temoin %d/%d = "+
		"%.1f %% (seuil %.0f %%) -> %s", val, cum.Touchees, cum.AvecKF, 100*tf,
		100*objSeuilFenetres, cum.TemoinTouchees, cum.TemoinKF, 100*tt, 100*objSeuilTemoin,
		objTenu(ok))
	return ok
}

// objPart rend a/b, 0 si b vaut 0.
func objPart(a, b int) float64 {
	if b == 0 {
		return 0
	}
	return float64(a) / float64(b)
}

// objTenu met en mots le resultat d'un seuil.
func objTenu(ok bool) string {
	if ok {
		return "TENU"
	}
	return "NON TENU"
}
