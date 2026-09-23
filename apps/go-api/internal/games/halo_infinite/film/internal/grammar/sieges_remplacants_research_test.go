//go:build research

package grammar

// sieges_remplacants_research_test.go — LE FILM DONNE-T-IL LE SIEGE D'UN REMPLACANT ?
// (mesure AVANT de coder du lot 1.9.14, rejeu de « D-remplacants (1.7) » du plan.)
//
// CE QUE L'INSTRUMENT MESURE, ET RIEN D'AUTRE. Chaque entite `ti=9` (« managed-player ») du
// film porte, dans son etat par defaut, l'INDEX DE JOUEUR de son occupant, et dans son
// composant i0 le DESIGNATEUR D'EQUIPE (lot 1.7, cf. player_teams.go). En relevant, pour
// chaque entite, le PREMIER et le DERNIER paquet d'image-cle qui la porte, on obtient la
// fenetre de presence de cette entite, et donc :
//
//	ARRIVEE       une entite dont le premier paquet n'est pas le premier paquet du film ;
//	PARTANT       une entite dont le dernier paquet n'est pas le dernier du film ;
//	REUTILISATION un index tenu par DEUX entites dont les fenetres ne se recouvrent pas —
//	              c'est-a-dire un remplacant qui reprend LE SIEGE du partant.
//
// LA QUESTION A LAQUELLE IL REPOND : le film donne-t-il le siege DIRECTEMENT (le remplacant
// reprend l'index du partant) ou faut-il apparier ? La reponse decide si l'appariement
// ordinal par equipe du modele des sieges du 2026-09-02 reste un repli NOMME ou devient la
// lecture.
//
// # CE QU'IL NE PEUT PAS MESURER ICI, ET POURQUOI C'EST ECRIT
//
// Il tourne sur les BOBINES versionnees (`replay/testdata/minifilm_*`), pas sur le cache de
// films : les bobines sont COUPEES au budget de 1 Mio et ne portent qu'une partie des paquets
// d'image-cle de leur film (cf. leur PROVENANCE.txt). Une arrivee posterieure au dernier
// paquet retenu est donc INVISIBLE a cet instrument — ce n'est pas un negatif du film, c'est
// une borne de la bobine. L'instrument PUBLIE cette borne (paquets retenus, et l'avertissement
// qu'une entite vivante au dernier paquet n'est pas « partie »), de sorte qu'aucune ligne ne
// se lise comme un fait du film alors qu'elle est un fait de la coupe.
//
// INSTRUMENT LOURD (rebalaye les bobines) : `//go:build research`, joue a la demande —
//
//	go test -tags research ./internal/games/halo_infinite/film/filmdec/ \
//	  -run TestSiegesDesRemplacants -v

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// siegesTemoins : les quatre temoins que le lot 1.9.14 nomme, plus leur build — pour que la
// ligne se lise sans rouvrir la table des bobines.
func siegesTemoins() []struct{ Court, Build string } {
	return []struct{ Court, Build string }{
		{"a521164d", "HI_1_4_1"},
		{"11de8353", "HI_1_9_0"},
		{"e5adf7b2", "HI_1_11_0"},
		{"bcb6d393", "HI_1_12_0"},
	}
}

// siegeEntite : ce qu'on retient d'UNE entite ti=9.
type siegeEntite struct {
	Slot                 int
	Index                int
	Designateurs         map[int]int
	PremierPk, DernierPk int
	PremierUS, DernierUS uint64
	Paquets              int
	IndexInstables       map[int]int
}

// TestSiegesDesRemplacants : le tableau des entites ti=9 par temoin, et le verdict de siege.
func TestSiegesDesRemplacants(t *testing.T) {
	// SIEGES_FILM (sonde P4, 2026-09-23) : un film COMPLET du cache au lieu des bobines coupees —
	// la borne de la bobine ne s applique plus, un depart avant le dernier paquet est un fait du film.
	if dir := os.Getenv("SIEGES_FILM"); dir != "" {
		film, err := source.LoadDir(dir, nil)
		if err != nil {
			t.Fatalf("%s : chargement du film : %v", dir, err)
		}
		ents, pk, err := balayerEntitesTi9(NewFilmContext(film))
		if err != nil {
			t.Fatalf("%s : %v", dir, err)
		}
		rapporterSieges(t, filepath.Base(dir), "cache", ents, pk)
		return
	}
	for _, tem := range siegesTemoins() {
		dir := filepath.Join("..", "..", "replay", "testdata", "minifilm_"+tem.Court)
		film, err := source.LoadDir(dir, nil)
		if err != nil {
			t.Fatalf("%s : chargement de la bobine : %v", tem.Court, err)
		}
		ents, pk, err := balayerEntitesTi9(NewFilmContext(film))
		if err != nil {
			t.Fatalf("%s : %v", tem.Court, err)
		}
		rapporterSieges(t, tem.Court, tem.Build, ents, pk)
	}
}

// balayerEntitesTi9 releve chaque entite ti=9 des paquets d'image-cle, dans l'ordre du film.
//
// LE RANG DE PAQUET EST GLOBAL au film (tous chunks confondus) : c'est la meme numerotation
// que le rapport « D-remplacants (1.7) » du plan, pour que les deux tableaux se comparent.
func balayerEntitesTi9(fc *FilmContext) (map[int]*siegeEntite, int, error) {
	reg, err := fc.Registry()
	if err != nil {
		return nil, 0, fmt.Errorf("registre du film : %w", err)
	}
	arch, ok := reg.Archetype(managedPlayerTypeIndex)
	if !ok || len(arch.Components) == 0 || arch.Components[0] != teamDesignatorComponent {
		return nil, 0, fmt.Errorf("ti=9 absent ou i0 inattendu (%q)", rapporterComposant(arch))
	}
	ents := map[int]*siegeEntite{}
	rang := -1
	for _, c := range fc.ChunkNumbers() {
		raw, paquets, ok := fc.ChunkAt(c)
		if !ok {
			continue
		}
		for _, pk := range paquets {
			if pk.Type != PacketTypeKeyframe {
				continue
			}
			rang++
			noterEntitesDuPaquet(pk.Payload(raw), reg, rang, pk.TimestampUS, ents)
		}
	}
	return ents, rang + 1, nil
}

// rapporterComposant rend le nom d'i0, ou une marque d'absence — jamais un panic d'index.
func rapporterComposant(a Archetype) string {
	if len(a.Components) == 0 {
		return "(aucun composant)"
	}
	return a.Components[0]
}

// noterEntitesDuPaquet lit les records ti=9 d'UN payload et met a jour les entites.
func noterEntitesDuPaquet(pay []byte, reg *Registry, rang int, ts uint64, ents map[int]*siegeEntite) {
	for _, b := range keyframeBornesToutes(pay) {
		if b.TI != managedPlayerTypeIndex {
			continue
		}
		idx, brut, ok := lireEquipeDuRecord(pay, b.Bit, reg, ContexteParDefaut())
		if !ok || idx < 0 || idx >= playerTableSlots || brut < 0 || brut > teamDesignatorRawMax {
			continue
		}
		e := ents[b.Slot]
		if e == nil {
			e = &siegeEntite{
				Slot: b.Slot, Index: idx, Designateurs: map[int]int{},
				IndexInstables: map[int]int{}, PremierPk: rang, PremierUS: ts,
			}
			ents[b.Slot] = e
		}
		e.IndexInstables[idx]++
		e.Designateurs[brut-1]++
		e.DernierPk, e.DernierUS, e.Paquets = rang, ts, e.Paquets+1
	}
}

// rapporterSieges imprime le tableau d'un temoin et son verdict.
func rapporterSieges(t *testing.T, court, build string, ents map[int]*siegeEntite, paquets int) {
	t.Helper()
	tri := make([]*siegeEntite, 0, len(ents))
	for _, e := range ents {
		tri = append(tri, e)
	}
	sort.Slice(tri, func(i, j int) bool {
		if tri[i].PremierPk != tri[j].PremierPk {
			return tri[i].PremierPk < tri[j].PremierPk
		}
		return tri[i].Index < tri[j].Index
	})
	t.Logf("=== %s (%s) — %d paquet(s) d'image-cle retenu(s) par la bobine, %d entite(s) ti=9",
		court, build, paquets, len(tri))
	for _, e := range tri {
		t.Logf("  %s | idx=%2d | slot=%5d | pk[%2d..%2d] | t[%d..%d] us | des=%s | paquets=%d%s",
			court, e.Index, e.Slot, e.PremierPk, e.DernierPk, e.PremierUS, e.DernierUS,
			histogrammeDesSieges(e.Designateurs), e.Paquets, marqueInstable(e.IndexInstables))
	}
	verdictDesSieges(t, court, tri, paquets)
}

// marqueInstable signale une entite dont l'index n'est pas constant — jamais tu en silence.
func marqueInstable(h map[int]int) string {
	if len(h) <= 1 {
		return ""
	}
	return " | INDEX INSTABLE " + histogrammeDesSieges(h)
}

// histogrammeDesSieges rend `v x n` trie par valeur, pour que deux executions rendent la meme chaine.
func histogrammeDesSieges(h map[int]int) string {
	cles := make([]int, 0, len(h))
	for k := range h {
		cles = append(cles, k)
	}
	sort.Ints(cles)
	out := ""
	for i, k := range cles {
		if i > 0 {
			out += ","
		}
		out += fmt.Sprintf("%d x%d", k, h[k])
	}
	return out
}

// verdictDesSieges nomme les arrivees, les departs, et les index REUTILISES.
//
// UNE ENTITE VIVANTE AU DERNIER PAQUET DE LA BOBINE N'EST PAS « PARTIE » : la coupe de la
// bobine s'arrete la. Le verdict le dit, au lieu de compter un depart qui n'existe pas.
func verdictDesSieges(t *testing.T, court string, tri []*siegeEntite, paquets int) {
	t.Helper()
	parIndex := map[int][]*siegeEntite{}
	arrivees, departs := 0, 0
	for _, e := range tri {
		parIndex[e.Index] = append(parIndex[e.Index], e)
		if e.PremierPk > 0 {
			arrivees++
		}
		if e.DernierPk < paquets-1 {
			departs++
		}
	}
	reutilises := 0
	for _, idx := range triDesCles(parIndex) {
		lot := parIndex[idx]
		if len(lot) < 2 {
			continue
		}
		reutilises++
		for i := 1; i < len(lot); i++ {
			t.Logf("  %s | SIEGE REPRIS idx=%d : partant slot=%d fin pk=%d (t=%d us) "+
				"-> arrivant slot=%d debut pk=%d (t=%d us) | recouvrement=%v",
				court, idx, lot[i-1].Slot, lot[i-1].DernierPk, lot[i-1].DernierUS,
				lot[i].Slot, lot[i].PremierPk, lot[i].PremierUS,
				lot[i].PremierPk <= lot[i-1].DernierPk)
		}
	}
	t.Logf("  %s | VERDICT : %d arrivee(s) | %d depart(s) AVANT le dernier paquet retenu | "+
		"%d index tenu(s) par plusieurs entites | siege ecrit directement = %v",
		court, arrivees, departs, reutilises, reutilises > 0)
	occupationParPaquet(t, court, tri, paquets)
}

// occupationParPaquet compte, paquet par paquet, les entites ti=9 VIVANTES et les index
// DISTINCTS qu'elles occupent.
//
// C'EST LA MESURE QUI DECIDE LA REGLE D'AFFICHAGE. Si le film ecrit plus d'index que de places
// (un 4v4 a 12 index quand quatre joueurs se relaient), alors « un siege = une fiche » applique
// a TOUT le roster donne douze fiches pour huit places — le defaut que l'utilisateur constate.
// Applique aux seuls OCCUPANTS d'un instant, il en donne huit. L'ecart entre les deux comptes
// est exactement ce que la regle de lecture doit retirer de l'ecran.
func occupationParPaquet(t *testing.T, court string, tri []*siegeEntite, paquets int) {
	t.Helper()
	maxi, mini := 0, 1<<30
	ligne := ""
	for pk := 0; pk < paquets; pk++ {
		n := 0
		for _, e := range tri {
			if e.PremierPk <= pk && pk <= e.DernierPk {
				n++
			}
		}
		if n > maxi {
			maxi = n
		}
		if n < mini {
			mini = n
		}
		ligne += fmt.Sprintf(" %d", n)
	}
	t.Logf("  %s | OCCUPANTS par paquet :%s", court, ligne)
	t.Logf("  %s | index ECRITS sur tout le film = %d | occupants SIMULTANES min=%d max=%d | "+
		"fiches en trop si on affiche tout le roster = %d",
		court, len(tri), mini, maxi, len(tri)-maxi)
}

// triDesCles rend les cles d'une table d'index, triees — l'ordre d'une map Go est aleatoire et
// un rapport qui change d'ordre a chaque execution ne se compare pas.
func triDesCles(m map[int][]*siegeEntite) []int {
	out := make([]int, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Ints(out)
	return out
}
