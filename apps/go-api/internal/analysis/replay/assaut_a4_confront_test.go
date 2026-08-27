package replay

// assaut_a4_confront_test.go — LOT A, PHASE A4 : CONFRONTATION DU BALAYAGE STATBORG.
//
// LE PROTOCOLE EST ECRIT ET COMMITE AVANT CE FICHIER (`registre_film/A_PROTOCOLE.md`, §5).
// AUCUN film n'est decode ici : l'entree est le TSV du balayage `cmd/statnames-sweep`
// (`A4_statborg_sweep.tsv`) et le releve participants fige (`A_oracle_participants.tsv`).
// Le mode `-confront` du CLI ne s'applique pas : ses colonnes oracle sont celles
// d'Oddball (API par joueur), et l'Assaut n'a AUCUN oracle API par joueur — les
// confrontations du protocole §5 sont d'une autre forme (somme d'equipe et controle de
// lecture), d'ou cet instrument dedie, sous garde d'environnement comme les campagnes D.
//
// LES DEUX CONFRONTATIONS, seuils recopies du protocole §5 :
//
//	(i)  COMPTEUR DE MODE — un emplacement (comp, cote) est CANDIDAT si la SOMME de ses
//	     valeurs finales sur TOUS les slots joueurs vaut exactement les explosions
//	     RETENUES du film (les valeurs du §2, tronquees a la manche 0 sur One Bomb par
//	     RealRounds) sur 4/4 films de RECHERCHE ; verdict REPLIQUE si 4/4 aussi en
//	     VERIFICATION. Garde anti-zero automatique : attendu >= 1 partout.
//	(ii) CONTROLE POSITIF — sur les films sans manche refusee, la valeur finale
//	     comp 2 B des slots NOMMES vaut les morts du releve participants pour >= 90 %
//	     des paires nommees. Un controle de LECTURE, pas un verdict du lot.
//
// REGIME : gardes `ASSAUT_SWEEP` (TSV du balayage) + `ASSAUT_PARTICIPANTS` (releve fige),
// aucune base, aucun film.
//
//	$env:ASSAUT_SWEEP=".ai/V7.5/replay2d/registre_film/A4_statborg_sweep.tsv"
//	$env:ASSAUT_PARTICIPANTS=".ai/V7.5/replay2d/registre_film/A_oracle_participants.tsv"
//	go test ./internal/analysis/replay/ -run AssautA4Confront -v

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// Les moities disjointes et les attendus du protocole §5, RECOPIES SANS MODIFICATION.
var (
	a4Recherche    = []string{"1c01e34f", "35b75a31", "69b16f5d", "c75f33b8"}
	a4Verification = []string{"34bb3bc8", "3d58eb37", "9f57c612", "df8fcbef"}
	// a4Explosions : explosions RETENUES par film (§2 du protocole — manches reelles
	// seulement, la troncature One Bomb est un fait mesure du releve A0.3).
	a4Explosions = map[string]int64{
		"1c01e34f": 4, "35b75a31": 3, "69b16f5d": 3, "c75f33b8": 1,
		"34bb3bc8": 1, "3d58eb37": 3, "9f57c612": 1, "df8fcbef": 1,
	}
	// a4FilmsControle : les films SANS manche refusee (§5) — le controle (ii) ne se joue
	// que la ou les totaux du TSV couvrent tout le match.
	a4FilmsControle = []string{"35b75a31", "69b16f5d", "3d58eb37", "34bb3bc8", "1c01e34f"}
)

// a4AccordControleMin : le seuil du controle (ii), recopie du protocole §5.
const a4AccordControleMin = 0.90

// a4Cle adresse un emplacement du statborg.
type a4Cle struct {
	comp int
	side string
}

func (k a4Cle) String() string { return fmt.Sprintf("comp %d %s", k.comp, k.side) }

// a4Sweep porte le TSV parse : valeurs finales par film -> emplacement -> slot, et le
// nommage des slots par film.
type a4Sweep struct {
	finals map[string]map[a4Cle]map[int]int64
	xuids  map[string]map[int]string
}

// TestAssautA4Confront — la confrontation, sur pieces figees.
func TestAssautA4Confront(t *testing.T) {
	sweepPath := os.Getenv("ASSAUT_SWEEP")
	partPath := os.Getenv("ASSAUT_PARTICIPANTS")
	if sweepPath == "" || partPath == "" {
		t.Skip("mesure non demandee : ASSAUT_SWEEP et ASSAUT_PARTICIPANTS requis")
	}
	sw := a4LireSweep(t, sweepPath)
	morts := a4LireMorts(t, partPath)
	a4Compteur(t, sw)
	a4Controle(t, sw, morts)
}

// a4Compteur joue la confrontation (i) : somme des slots joueurs contre les explosions.
func a4Compteur(t *testing.T, sw a4Sweep) {
	t.Helper()
	cles := a4Cles(sw)
	t.Logf("CONFRONTATION (i) : %d emplacement(s) balayes ; recherche %v ; verification %v",
		len(cles), a4Recherche, a4Verification)
	candidats := 0
	for _, k := range cles {
		if !a4SommeExacte(sw, k, a4Recherche) {
			continue
		}
		candidats++
		verdict := "NE REPLIQUE PAS (verification en echec)"
		if a4SommeExacte(sw, k, a4Verification) {
			verdict = "REPLIQUE"
		}
		t.Logf("CANDIDAT %s : sommes de recherche exactes 4/4 — %s ; sommes publiees : %s",
			k, verdict, a4Sommes(sw, k))
	}
	if candidats == 0 {
		// Le meilleur emplacement se publie quand meme : un negatif sans denominateur ne
		// se juge pas (regle du chantier).
		best, hits := a4Meilleur(sw, cles)
		t.Logf("NEGATIF (i) : aucun emplacement dont la somme vaut les explosions retenues "+
			"sur 4/4 films de recherche — meilleur : %s avec %d/4 films exacts ; sommes : %s",
			best, hits, a4Sommes(sw, best))
	}
}

// a4SommeExacte dit si la somme des valeurs finales des slots joueurs vaut l'attendu sur
// TOUS les films donnes.
func a4SommeExacte(sw a4Sweep, k a4Cle, films []string) bool {
	for _, f := range films {
		var somme int64
		for _, v := range sw.finals[f][k] {
			somme += v
		}
		if somme != a4Explosions[f] {
			return false
		}
	}
	return true
}

// a4Meilleur rend l'emplacement au plus grand nombre de films de recherche exacts.
func a4Meilleur(sw a4Sweep, cles []a4Cle) (a4Cle, int) {
	var best a4Cle
	bestHits := -1
	for _, k := range cles {
		hits := 0
		for _, f := range a4Recherche {
			var somme int64
			for _, v := range sw.finals[f][k] {
				somme += v
			}
			if somme == a4Explosions[f] {
				hits++
			}
		}
		if hits > bestHits {
			best, bestHits = k, hits
		}
	}
	return best, bestHits
}

// a4Sommes publie les sommes par film d'un emplacement, dans l'ordre du protocole.
func a4Sommes(sw a4Sweep, k a4Cle) string {
	var parts []string
	for _, f := range append(append([]string{}, a4Recherche...), a4Verification...) {
		var somme int64
		for _, v := range sw.finals[f][k] {
			somme += v
		}
		parts = append(parts, fmt.Sprintf("%s=%d/%d", f, somme, a4Explosions[f]))
	}
	return strings.Join(parts, " ")
}

// a4Controle joue la confrontation (ii) : comp 2 B des slots nommes contre les morts.
func a4Controle(t *testing.T, sw a4Sweep, morts map[string]map[string]int64) {
	t.Helper()
	k := a4Cle{comp: 2, side: "B"}
	paires, matches := 0, 0
	for _, f := range a4FilmsControle {
		for slot, xuid := range sw.xuids[f] {
			want, ok := morts[f][xuid]
			if !ok {
				continue
			}
			paires++
			got := sw.finals[f][k][slot]
			if got == want {
				matches++
			} else {
				t.Logf("CONTROLE (ii) %s : slot %d (%s) comp 2 B = %d, morts releve = %d — ECART",
					f, slot, xuid, got, want)
			}
		}
	}
	taux := 0.0
	if paires > 0 {
		taux = float64(matches) / float64(paires)
	}
	verdict := "LECTURE SAINE"
	if taux < a4AccordControleMin {
		verdict = "LECTURE DOUTEUSE — les verdicts (i) de ces films ne se lisent qu'avec cette reserve"
	}
	t.Logf("CONTROLE (ii) : comp 2 B contre morts du releve — %d/%d = %.1f %% (seuil %.0f %%) : %s",
		matches, paires, 100*taux, 100*a4AccordControleMin, verdict)
}

// a4Cles rend tous les emplacements vus, ordonnes.
func a4Cles(sw a4Sweep) []a4Cle {
	seen := map[a4Cle]bool{}
	for _, byKey := range sw.finals {
		for k := range byKey {
			seen[k] = true
		}
	}
	out := make([]a4Cle, 0, len(seen))
	for k := range seen {
		out = append(out, k)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].comp != out[j].comp {
			return out[i].comp < out[j].comp
		}
		return out[i].side < out[j].side
	})
	return out
}

// a4LireSweep parse le TSV du balayage (lignes JOUEUR et SWEEP, format de sweep.go).
func a4LireSweep(t *testing.T, path string) a4Sweep {
	t.Helper()
	f, err := os.Open(path) //nolint:gosec // chemin fourni par l'operateur
	if err != nil {
		t.Fatalf("balayage : %v", err)
	}
	defer f.Close()
	sw := a4Sweep{finals: map[string]map[a4Cle]map[int]int64{}, xuids: map[string]map[int]string{}}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		champs := strings.Split(strings.TrimSpace(sc.Text()), "\t")
		switch {
		case champs[0] == "JOUEUR" && len(champs) == 4:
			film := champs[1]
			slot, err := strconv.Atoi(champs[2])
			if err != nil {
				t.Fatalf("ligne JOUEUR : slot %q", champs[2])
			}
			if sw.xuids[film] == nil {
				sw.xuids[film] = map[int]string{}
			}
			sw.xuids[film][slot] = champs[3]
		case champs[0] == "SWEEP" && len(champs) == 7:
			film := champs[1]
			slot, err1 := strconv.Atoi(champs[2])
			comp, err2 := strconv.Atoi(champs[4])
			final, err3 := strconv.ParseInt(champs[6], 10, 64)
			if err1 != nil || err2 != nil || err3 != nil {
				t.Fatalf("ligne SWEEP illisible : %q", champs)
			}
			k := a4Cle{comp: comp, side: champs[5]}
			if sw.finals[film] == nil {
				sw.finals[film] = map[a4Cle]map[int]int64{}
			}
			if sw.finals[film][k] == nil {
				sw.finals[film][k] = map[int]int64{}
			}
			sw.finals[film][k][slot] = final
		}
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("balayage : %v", err)
	}
	return sw
}

// a4LireMorts parse le releve participants fige : film -> xuid -> morts. Les bots
// (`bid(...)`) et les lignes NULL sortent — aucun pont ne peut les nommer.
func a4LireMorts(t *testing.T, path string) map[string]map[string]int64 {
	t.Helper()
	f, err := os.Open(path) //nolint:gosec // chemin fourni par l'operateur
	if err != nil {
		t.Fatalf("participants : %v", err)
	}
	defer f.Close()
	out := map[string]map[string]int64{}
	sc := bufio.NewScanner(f)
	premier := true
	for sc.Scan() {
		if premier {
			premier = false // l'en-tete
			continue
		}
		champs := strings.Split(strings.TrimSpace(sc.Text()), "\t")
		if len(champs) != 6 {
			continue
		}
		film, xuid := champs[0], champs[1]
		morts, err := strconv.ParseInt(champs[4], 10, 64)
		if err != nil {
			continue // bot sans stats ou NULL : hors pont, hors controle
		}
		if _, err := strconv.ParseUint(xuid, 10, 64); err != nil {
			continue // bot bid(...) : pas de xuid
		}
		if out[film] == nil {
			out[film] = map[string]int64{}
		}
		out[film][xuid] = morts
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("participants : %v", err)
	}
	return out
}
