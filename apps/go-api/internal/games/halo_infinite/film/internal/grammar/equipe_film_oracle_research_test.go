package grammar

// equipe_film_oracle_research_test.go — PHASE 3, LA CONFRONTATION A L'ORACLE EXTERNE.
//
// D'OU VIENT CHAQUE MOITIE DE LA COMPARAISON, et pourquoi elles sont independantes :
//
//	L'ORDRE vient du FILM, et il est deja prouve : `s3sChaine` lit la table des slots de
//	`chunk_00` (phase 2), dont le rang EST le `player_index` de production
//	(`filmIndex - rang` constant sur 76/76 films). Rang -> XUID, donc.
//	L'EQUIPE vient de la BASE : `match_participants.team_id`, passe par la garde
//	CHUNK00_XUID_EQUIPES sous la forme `<prefixe>=<xuid>:<team>,<xuid>:<team>,...` — un
//	ENSEMBLE, pas une suite : l'instrument ne recoit AUCUN ordre de l'oracle, il l'apparie par
//	XUID. Une permutation de la garde ne peut donc pas fabriquer un accord.
//	LA VALEUR vient de la TRAME : le champ de 4 bits du composant
//	`managed-player-team-designator-component` (ti=9 i0), au decalage mesure.
//
// Si la lecture de la trame est fausse, l'accord tombe ; si l'ordre de `chunk_00` est faux,
// l'accord tombe ; si l'oracle est faux, l'accord tombe. Les trois sont produits par des
// chaines sans etape commune.
//
// LA PREDICTION, ecrite avant la mesure et issue du seul desassemblage (`DEC R9B` dans
// `FUN_1407ef804`) : `brut = team_id + 1`. Pas « correle a », pas « a une permutation pres » :
// EGAL, terme a terme.
//
// LE CONTROLE NEGATIF, ecrit avant la mesure : la meme comparaison aux 32 decalages de bit
// voisins (-16..+16, hors 0). Le plancher de bruit est le nombre de ces decalages qui rendent
// l'accord exact — MESURE, pas calcule.
//
// Gardes CHUNK00_FILMS et CHUNK00_XUID_EQUIPES. Lecture seule, aucun code de production touche.

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// equipeVoisinage est la demi-largeur du controle negatif, en bits.
const equipeVoisinage = 16

// equipeOracleXuid lit CHUNK00_XUID_EQUIPES : `<prefixe>=<xuid>:<team>,...`, blocs separes
// par `;`. Rend, par prefixe, la table xuid -> equipe. Aucun ordre n'est lu.
func equipeOracleXuid(t *testing.T) map[string]map[uint64]int {
	t.Helper()
	out := map[string]map[uint64]int{}
	for _, bloc := range strings.Split(os.Getenv("CHUNK00_XUID_EQUIPES"), ";") {
		if bloc = strings.TrimSpace(bloc); bloc == "" {
			continue
		}
		pref, liste, ok := strings.Cut(bloc, "=")
		if !ok {
			continue
		}
		m := map[uint64]int{}
		for _, couple := range strings.Split(liste, ",") {
			xs, ts, ok := strings.Cut(strings.TrimSpace(couple), ":")
			if !ok {
				continue
			}
			x, err1 := strconv.ParseUint(strings.TrimSpace(xs), 10, 64)
			eq, err2 := strconv.Atoi(strings.TrimSpace(ts))
			if err1 != nil || err2 != nil {
				continue
			}
			m[x] = eq
		}
		out[strings.TrimSpace(pref)] = m
	}
	return out
}

// equipeRangsXuid rend les XUID de la table des slots de `chunk_00`, dans l'ordre du rang
// (= le `player_index` de production, phase 2 C.2).
//
// DEUX LECTEURS, ET LA REGLE DE BASCULE EST ECRITE D'AVANCE. Le lecteur canonique de la
// phase 2 (`s3sChaine`, qui avance du pas PREDIT) casse sur les gros rosters : il s'arrete au
// premier enregistrement dont le nom n'est pas imprimable, defaut connu et consigne
// (phase 2, section G.3). Quand il rend MOINS d'enregistrements que le balayage brut, on prend
// le balayage brut filtre : nom imprimable, XUID present dans l'oracle, dedoublonne par XUID,
// TRIE PAR POSITION DE BIT — donc toujours l'ordre du flux, jamais un ordre choisi. Le filtre
// « XUID dans l'oracle » ne peut pas fabriquer d'ordre : il retire des lignes, il n'en
// reordonne aucune.
func equipeRangsXuid(t *testing.T, dir string, orc map[uint64]int) (xuids []uint64, brut bool) {
	t.Helper()
	_, d := readChunk00(t, dir)
	for _, e := range s3sChaine(d) {
		xuids = append(xuids, e.xuid)
	}
	es := s3sEnrs(d)
	sort.Slice(es, func(i, j int) bool { return es[i].debut < es[j].debut })
	vus := map[uint64]bool{}
	var repli []uint64
	for _, e := range es {
		if !s3sImprimable(e.gamertag) || vus[e.xuid] {
			continue
		}
		if _, ok := orc[e.xuid]; !ok {
			continue
		}
		vus[e.xuid] = true
		repli = append(repli, e.xuid)
	}
	if len(repli) > len(xuids) {
		return repli, true
	}
	return xuids, false
}

// equipeAttendu construit le vecteur PREDIT (brut = equipe + 1) dans l'ordre des rangs, en ne
// gardant que les rangs dont le XUID est dans l'oracle. Rend aussi les rangs retenus, pour que
// l'appelant sache a quelles entites ti=9 les comparer.
func equipeAttendu(rangs []uint64, oracle map[uint64]int) (pred equipeVecteur, rgs []int) {
	for i, x := range rangs {
		if eq, ok := oracle[x]; ok {
			pred = append(pred, eq+1)
			rgs = append(rgs, i)
		}
	}
	return pred, rgs
}

// equipeCompteAccord compte les positions egales entre deux vecteurs de meme longueur.
func equipeCompteAccord(a, b equipeVecteur) int {
	n := 0
	for i := range a {
		if i < len(b) && a[i] == b[i] {
			n++
		}
	}
	return n
}

// TestEquipeFilmOracleXuid est la confrontation : au decalage mesure, le vecteur de la trame
// doit valoir `team_id + 1` terme a terme, l'ordre venant de `chunk_00`.
func TestEquipeFilmOracleXuid(t *testing.T) {
	dirs := chunk00Films(t, "CHUNK00_FILMS")
	oracle := equipeOracleXuid(t)
	if len(oracle) == 0 {
		t.Skip("CHUNK00_XUID_EQUIPES absent : instrument saute")
	}
	var slotsTotal, slotsBons, filmsTotal, filmsBons int
	var voisinsEssais, voisinsTouches int
	for _, dir := range dirs {
		pref := filepath.Base(dir)
		orc, ok := oracle[pref]
		if !ok {
			continue
		}
		v, err := equipePrepare(dir)
		if err != nil || !v.Longueur {
			t.Logf("%s : ECARTE (%v)", pref, err)
			continue
		}
		rangs, repli := equipeRangsXuid(t, dir, orc)
		pred, rgs := equipeAttendu(rangs, orc)
		lu := equipeVecteurA(v.Paquets[0], equipeDecalageMesure)
		if len(rgs) != v.Card || len(pred) == 0 {
			t.Logf("%s : %d rangs apparies a l'oracle contre %d entites ti=%d — "+
				"CARDINAUX DIFFERENTS, film compte a part (rangs %v)",
				pref, len(rgs), v.Card, equipeTI, rgs)
			continue
		}
		bons := equipeCompteAccord(pred, lu)
		filmsTotal++
		slotsTotal += len(pred)
		slotsBons += bons
		if bons == len(pred) {
			filmsBons++
		}
		e, h := equipeControleVoisins(v, pred)
		voisinsEssais += e
		voisinsTouches += h
		t.Logf("%s : %d entites ; predit %v ; lu %v ; accord %d/%d ; voisins %d/%d ; repli=%v",
			pref, v.Card, pred, lu, bons, len(pred), h, e, repli)
	}
	t.Logf("ORACLE EXTERNE : %d/%d films en accord TOTAL, %d/%d slots ; "+
		"CONTROLE NEGATIF : %d touches sur %d decalages voisins",
		filmsBons, filmsTotal, slotsBons, slotsTotal, voisinsTouches, voisinsEssais)
}

// equipeControleVoisins rejoue la comparaison aux 32 decalages de bit voisins (hors 0) et
// compte ceux qui rendent l'accord EXACT. C'est le plancher de bruit mesure.
func equipeControleVoisins(v equipeVue, pred equipeVecteur) (essais, touches int) {
	for dd := -equipeVoisinage; dd <= equipeVoisinage; dd++ {
		if dd == 0 {
			continue
		}
		d := equipeDecalageMesure + dd
		if d < 0 || d+equipeDesignatorBits > v.Long {
			continue
		}
		essais++
		if equipeCompteAccord(pred, equipeVecteurA(v.Paquets[0], d)) == len(pred) {
			touches++
		}
	}
	return essais, touches
}

// TestEquipeFilmOracleBalayage rejoue la confrontation a TOUS les decalages du record, et rend
// ceux qui donnent l'accord exact. C'est la version « sans a priori » : elle verifie que le
// decalage mesure est bien le SEUL (a son echo d'un bit pres) qui satisfasse l'oracle.
func TestEquipeFilmOracleBalayage(t *testing.T) {
	dirs := chunk00Films(t, "CHUNK00_FILMS")
	oracle := equipeOracleXuid(t)
	if len(oracle) == 0 {
		t.Skip("CHUNK00_XUID_EQUIPES absent : instrument saute")
	}
	commun := map[int]int{}
	films := 0
	for _, dir := range dirs {
		pref := filepath.Base(dir)
		orc, ok := oracle[pref]
		if !ok {
			continue
		}
		v, err := equipePrepare(dir)
		if err != nil || !v.Longueur {
			continue
		}
		rangs, _ := equipeRangsXuid(t, dir, orc)
		pred, rgs := equipeAttendu(rangs, orc)
		if len(rgs) != v.Card || len(pred) == 0 {
			continue
		}
		films++
		var accord []int
		for d := 0; d+equipeDesignatorBits <= v.Long; d++ {
			if equipeCompteAccord(pred, equipeVecteurA(v.Paquets[0], d)) == len(pred) {
				accord = append(accord, d)
				commun[d]++
			}
		}
		t.Logf("%s : %d decalages sur %d rendent l'accord EXACT : %v",
			pref, len(accord), v.Long-equipeDesignatorBits+1, accord)
	}
	if films == 0 {
		t.Skip("aucun film apparie")
	}
	var tous []equipePaire
	for d, n := range commun {
		tous = append(tous, equipePaire{d, n})
	}
	sort.Slice(tous, func(i, j int) bool {
		if tous[i].n != tous[j].n {
			return tous[i].n > tous[j].n
		}
		return tous[i].d < tous[j].d
	})
	t.Logf("BALAYAGE COMPLET sur %d films : %s", films, equipePairesTexte(tous, 12))
}
