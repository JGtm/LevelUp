//go:build research

package replay

// sieges_document_research_test.go — CE QUE LE DOCUMENT PUBLIE AUJOURD'HUI D'UN SIEGE ET D'UNE
// PRESENCE (mesure AVANT de coder du lot 1.9.14).
//
// L'instrument `filmdec/sieges_remplacants_research_test.go` mesure ce que le FILM ecrit ;
// celui-ci mesure ce que l'ARTEFACT en publie. Les deux repondent a deux questions distinctes
// de l'item :
//
//	1. chaque joueur du roster a-t-il un SIEGE publie (`RosterEntry.FilmIndex`) ?
//	2. ses INTERVALLES DE PRESENCE (les vies du lot 1.9.13) tiennent-ils la promesse : un
//	   partant n'a plus de vie apres son depart, un arrivant n'en a pas avant son arrivee ?
//
// IL NE LIT AUCUN OCTET DE FILM : il rejoue l'assemblage depuis les fixtures d'entrees FIGES
// (`testdata/inputs_*.bin.gz`), qui viennent des films ENTIERS — donc sans la coupe des bobines
// et sans le cache. C'est la seule voie qui donne, ici, les arrivees TARDIVES.
//
//	go test -tags research ./internal/games/halo_infinite/film/replay/ \
//	  -run TestSiegesDuDocument -v

import (
	"fmt"
	"sort"
	"testing"
)

// siegesDocTemoins : les quatre temoins du lot, dans l'ordre des builds.
func siegesDocTemoins() map[string]bool {
	return map[string]bool{"a521164d": true, "11de8353": true, "e5adf7b2": true, "bcb6d393": true}
}

// TestSiegesDuDocument : le roster publie, son siege, et les vies de chaque siege.
func TestSiegesDuDocument(t *testing.T) {
	temoins := siegesDocTemoins()
	for _, b := range goldenBuilds() {
		if !temoins[b.Short8] {
			continue
		}
		t.Run(b.Build+"/"+b.Short8, func(t *testing.T) {
			g, entry := chargerGoldenBuild(t, b)
			doc := assemblerGoldenBuild(t, b, g, entry)
			rapporterSiegesDuDocument(t, b.Short8, doc)
		})
	}
}

// rapporterSiegesDuDocument imprime, siege par siege, ce que le document en dit.
func rapporterSiegesDuDocument(t *testing.T, court string, doc ReplayDocument) {
	t.Helper()
	parCle := map[string][]Track{}
	sansCle := 0
	for _, tr := range doc.Tracks {
		cle := cleDIdentiteDeLaTrace(tr)
		if cle == "" {
			sansCle++
			continue
		}
		parCle[cle] = append(parCle[cle], tr)
	}
	t.Logf("=== %s — %d entree(s) de roster, %d piste(s) dont %d sans identite, %d frame(s)",
		court, len(doc.Roster), len(doc.Tracks), sansCle, doc.FrameCount)
	sieges := map[int]int{}
	for _, e := range doc.Roster {
		sieges[e.Seat]++
		t.Logf("  %s | idx=%2d siege=%2d (%s) | %s | equipe=%s | %s",
			court, e.FilmIndex, e.Seat, e.SeatSource, identiteDeLEntree(e), equipeDeLEntree(e),
			resumeDesVies(parCle[cleDIdentiteDeLEntree(e)]))
	}
	if c := doc.Coverage; c != nil && c.Seats != nil {
		t.Logf("  %s | COUVERTURE : %+v", court, *c.Seats)
	}
	partages := 0
	for _, n := range sieges {
		if n > 1 {
			partages++
		}
	}
	t.Logf("  %s | VERDICT DOCUMENT : %d siege(s) distinct(s) pour %d entree(s) | "+
		"%d siege(s) PARTAGE(S) par plusieurs entrees | fiches simultanees = %s",
		court, len(sieges), len(doc.Roster), partages, occupationDesSieges(doc, parCle))
}

// cleDIdentiteDeLaTrace / cleDIdentiteDeLEntree : LA MEME CLE DES DEUX COTES — le xuid quand il
// existe, sinon le nom du bot. C'est la jointure que le web fait (`rosterEntryKey`), rejouee ici
// pour que la mesure porte sur ce que le web verra, pas sur une jointure de laboratoire.
func cleDIdentiteDeLaTrace(tr Track) string {
	if tr.XUID != "" {
		return tr.XUID
	}
	if tr.Bot != "" {
		return "bot:" + tr.Bot
	}
	return ""
}

func cleDIdentiteDeLEntree(e RosterEntry) string {
	if e.XUID != "" {
		return e.XUID
	}
	if e.Bot && e.Name != "" {
		return "bot:" + e.Name
	}
	return ""
}

// identiteDeLEntree rend une identite lisible d'une entree de roster.
func identiteDeLEntree(e RosterEntry) string {
	nom := e.Name
	if nom == "" {
		nom = "(sans nom)"
	}
	if e.Bot {
		return fmt.Sprintf("BOT %-24s", nom)
	}
	return fmt.Sprintf("%-24s", nom)
}

// equipeDeLEntree rend le designateur publie, ou la marque d'un SILENCE — les deux ne se
// confondent pas (cf. RosterEntry.Team, pointeur a trois etats).
func equipeDeLEntree(e RosterEntry) string {
	if e.Team == nil {
		return "(muet)"
	}
	return fmt.Sprintf("%2d", *e.Team)
}

// resumeDesVies rend le nombre de vies et leur enveloppe de frames — l'intervalle de presence.
func resumeDesVies(vies []Track) string {
	if len(vies) == 0 {
		return "0 vie — AUCUNE PRESENCE PUBLIEE"
	}
	sort.Slice(vies, func(i, j int) bool { return vies[i].StartFrame < vies[j].StartFrame })
	deb, fin := vies[0].StartFrame, vies[0].EndFrame
	for _, v := range vies {
		if v.StartFrame < deb {
			deb = v.StartFrame
		}
		if v.EndFrame > fin {
			fin = v.EndFrame
		}
	}
	return fmt.Sprintf("%2d vie(s), presence [%d..%d]", len(vies), deb, fin)
}

// occupationDesSieges compte, sur un echantillon de frames, combien de sieges ont un occupant
// VIVANT — c'est-a-dire combien de fiches la regle de lecture du lot doit afficher, contre le
// nombre d'entrees de roster qu'on affiche aujourd'hui.
func occupationDesSieges(doc ReplayDocument, parCle map[string][]Track) string {
	if doc.FrameCount <= 0 {
		return "(pas de frames)"
	}
	mini, maxi := 1<<30, 0
	for pas := 0; pas < 20; pas++ {
		f := doc.FrameCount * pas / 20
		n := 0
		for _, e := range doc.Roster {
			for _, v := range parCle[cleDIdentiteDeLEntree(e)] {
				if v.StartFrame <= f && f <= v.EndFrame {
					n++
					break
				}
			}
		}
		if n < mini {
			mini = n
		}
		if n > maxi {
			maxi = n
		}
	}
	return fmt.Sprintf("min=%d max=%d (contre %d entrees affichees aujourd'hui)",
		mini, maxi, len(doc.Roster))
}
