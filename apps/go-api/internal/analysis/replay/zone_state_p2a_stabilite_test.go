package replay

// zone_state_p2a_stabilite_test.go — CB.2a.1, seconde clause : LA TABLE slot -> zone EST-ELLE LA
// MEME D'UN MATCH A L'AUTRE, SUR LA MEME CARTE ?
//
// POURQUOI CETTE CLAUSE EST LE COEUR DE CB.2a.1, ET PAS UN SUPPLEMENT. Une carte slot -> zone
// mesuree sur UN match peut n'etre qu'un arrangement de coincidences : rien n'empeche que le
// vote modal tombe juste par hasard sur trois zones. Si la MEME table ressort de DEUX matchs
// joues a des mois d'intervalle, ce n'est plus un arrangement — c'est que le slot ti=13 designe
// bien la zone, et que le jeu attribue ses slots de facon reproductible.
//
// CE TEST NE DECODE AUCUN FILM. Il relit les TSV produits par `TestZoneEtatPhase2a`, donc il
// tourne en une fraction de seconde et se rejoue sans re-payer le balayage. Il se saute
// proprement tant que les deux mesures ne sont pas sur le disque.
//
// USAGE (depuis apps/go-api, apres avoir mesure les deux Strongholds) :
//
//	go test -count=1 -run TestZoneEtatPhase2aStabilite -v ./internal/analysis/replay/

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// p2aPairesStables : les couples de films de la MEME carte dont la table doit coincider.
var p2aPairesStables = [][2]string{{"7344d24f", "696a9d7c"}} // Vagabond, Strongholds

// p2aLigneTable est une ligne `a1_table` du TSV : slot, cle tag 5, rang de zone, votes.
type p2aLigneTable struct {
	slot       uint32
	cle        string
	zone       int
	votes, tot int
}

// TestZoneEtatPhase2aStabilite compare les tables slot -> zone de deux films d'une meme carte.
func TestZoneEtatPhase2aStabilite(t *testing.T) {
	out := p2aOutDir(t)
	for _, paire := range p2aPairesStables {
		a, okA := p2aLitTable(t, out, paire[0])
		b, okB := p2aLitTable(t, out, paire[1])
		if !okA || !okB {
			t.Skipf("tables absentes (%s : %v, %s : %v) — jouer TestZoneEtatPhase2a sur les deux"+
				" films avant cette comparaison", paire[0], okA, paire[1], okB)
		}
		p2aCompareTables(t, paire, a, b)
	}
}

// p2aLitTable relit les lignes `a1_table` du TSV d'un film.
func p2aLitTable(t *testing.T, dir, short string) (map[uint32]p2aLigneTable, bool) {
	t.Helper()
	blob, err := os.ReadFile(filepath.Join(dir, short+"_p2a.tsv"))
	if err != nil {
		return nil, false
	}
	out := map[uint32]p2aLigneTable{}
	for _, line := range strings.Split(string(blob), "\n") {
		f := strings.Split(strings.TrimRight(line, "\r"), "\t")
		if len(f) < 8 || f[0] != "a1_table" {
			continue
		}
		slot, err1 := strconv.ParseUint(f[2], 10, 32)
		zone, err2 := strconv.Atoi(f[4])
		votes, err3 := strconv.Atoi(f[6])
		tot, err4 := strconv.Atoi(f[7])
		if err1 != nil || err2 != nil || err3 != nil || err4 != nil {
			t.Fatalf("%s : ligne a1_table illisible : %q", short, line)
		}
		out[uint32(slot)] = p2aLigneTable{slot: uint32(slot), cle: f[3], zone: zone,
			votes: votes, tot: tot}
	}
	return out, len(out) > 0
}

// p2aCompareTables publie la comparaison et tranche la clause de stabilite.
//
// LA CLAUSE EST JUGEE SUR LES SLOTS COMMUNS AUX DEUX FILMS, et le rapport publie combien de
// slots ne sont pas partages : un slot vu dans un seul match n'infirme rien (il peut n'avoir eu
// qu'une capture), mais le cacher laisserait croire a une concordance plus large qu'elle n'est.
func p2aCompareTables(t *testing.T, paire [2]string, a, b map[uint32]p2aLigneTable) {
	t.Helper()
	var communs, concordants int
	slots := make([]uint32, 0, len(a))
	for s := range a {
		slots = append(slots, s)
	}
	sort.Slice(slots, func(i, j int) bool { return slots[i] < slots[j] })
	t.Logf("STABILITE %s vs %s — %d slots d'un cote, %d de l'autre", paire[0], paire[1],
		len(a), len(b))
	for _, s := range slots {
		la := a[s]
		lb, ok := b[s]
		if !ok {
			t.Logf("  slot %-5d : zone %d (%d/%d votes) — ABSENT de %s", s, la.zone, la.votes,
				la.tot, paire[1])
			continue
		}
		communs++
		verdict := "DIFFERENT"
		if la.zone == lb.zone {
			concordants++
			verdict = "identique"
		}
		t.Logf("  slot %-5d : zone %d (%d/%d) vs zone %d (%d/%d) — %s · cles %s / %s", s,
			la.zone, la.votes, la.tot, lb.zone, lb.votes, lb.tot, verdict, la.cle, lb.cle)
	}
	if communs == 0 {
		t.Fatalf("aucun slot commun entre %s et %s : la clause de stabilite n'est pas jugeable",
			paire[0], paire[1])
	}
	part := p2aRate(concordants, communs)
	v := "NON TENUE"
	if part >= p2aSeuilCoherence {
		v = "TENUE"
	}
	t.Logf("  CLAUSE DE STABILITE : %d/%d slots communs designent la MEME zone = %.1f %%"+
		" (seuil %.0f %%) — %s", concordants, communs, 100*part, 100*p2aSeuilCoherence, v)
}
