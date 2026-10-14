//go:build research

package grammar

// ri27d1_m2_ancres_research_test.go — 2.7.d1 (`.ai/PLAN_RI_ETAT_COMPLET_IMAGES_CLES_2026-10-08.md`,
// commandes I-ancres et I-equipes).
//
// L ELECTION DES ANCRES ET LES PREUVES D IMAGE-CLE, dans le contexte de la cuisson ([ri27cContexte],
// carte posee, MPP declare). Par film, la phase des images-cles ([FilmContext.ImagesCles]) : chaque
// ancre (chunk, paquet, slot, archetype, premier bit, elue ou non), la preuve de son etat complet
// (fermee sur l ancre suivante ou non), et la PREUVE de l election ([PreuveDImageCle.prouve], profil
// invariant du film) jouee sur chaque ancre et sur chaque candidat ecarte. Plus les decisions de la
// marche ([KeyframeWalkStats]). Un seul mode : la marche de production a la tete (etiquette `tete`).
//
//	RI27C_FILMS=<id,...> RI27C_RACINE=<film_chunks> RI27C_OUT=<dossier> [RI27C_CARTES=<id=Carte;...>] \
//	  go test -tags=research -count=1 -run '^TestRI27d1M2Ancres$' ./...grammar/
//
// Sorties dans RI27C_OUT : m2_ancres_tete.tsv (une ligne par ancre ou ecarte), m2_stats_tete.tsv (une
// ligne par film) ; TestRI27d1M2Equipes : m2_equipes.tsv.

import (
	"fmt"
	"path/filepath"
	"runtime"
	"sort"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"
)

func TestRI27d1M2Ancres(t *testing.T) {
	films, racine, sortie := ri27cEnv(t)
	var lignes, stats []string
	for _, court := range films {
		fc := ri27cContexte(t, filepath.Join(racine, court), court, true)
		l, s := ri27d1M2UnFilm(t, fc, court)
		lignes = append(lignes, l...)
		stats = append(stats, s)
		runtime.GC()
	}
	ri27cEcrire(t, filepath.Join(sortie, "m2_ancres_tete.tsv"), lignes)
	ri27cEcrire(t, filepath.Join(sortie, "m2_stats_tete.tsv"), stats)
}

// ri27d1M2UnFilm marche les images-cles d un film et rend ses lignes et sa ligne de decisions.
func ri27d1M2UnFilm(t *testing.T, fc *FilmContext, court string) ([]string, string) {
	t.Helper()
	marche, preuve := fc.MarcheDImageCle(), fc.PreuveDImageCle()
	var st KeyframeWalkStats
	var lignes []string
	ancres, elues, fermes, prouves, ecartes, ecartesProuves := 0, 0, 0, 0, 0, 0
	for p, err := range fc.ImagesCles() {
		if err != nil {
			t.Fatalf("%s : %v", court, err)
		}
		mp := marche.Marcher(p.Payload)
		st.Ajouter(mp.Stats)
		for _, r := range p.Records {
			el := r.Liaison == lecture.LiaisonImageCleElue
			fe := r.Preuve == lecture.PreuveFerme
			pr := preuve.prouve(p.Payload, int(r.Debut), int(r.Vie.Slot))
			ancres++
			if el {
				elues++
			}
			if fe {
				fermes++
			}
			if pr {
				prouves++
			}
			lignes = append(lignes, fmt.Sprintf("A\t%s\t%d\t%d\t%d\t%d\t%d\t%d\t%v\t%v\t%v\t%d\t%d",
				court, p.Chunk, p.Index, r.TI, r.Vie.Slot, r.Vie.Gen, r.Debut, el, fe, pr, r.Desync, r.Bits))
		}
		for _, e := range mp.Ecartes {
			pr := preuve.prouve(p.Payload, e.Bit, e.Slot)
			ecartes++
			if pr {
				ecartesProuves++
			}
			lignes = append(lignes, fmt.Sprintf("E\t%s\t%d\t%d\t%d\t%d\t%d\t%d\tfalse\tfalse\t%v\t-\t-",
				court, p.Chunk, p.Index, e.TI, e.Slot, e.Gen, e.Bit, pr))
		}
	}
	s := fmt.Sprintf("%s\tpayloads=%d\tancres=%d\tbipedes=%d\tvoisins=%d\tsauts=%d\trecalages=%d\telections=%d\t"+
		"refutations=%d\tpreuves_contradictoires=%d\tglissements=%d\telues=%d\tfermes=%d\tprouves=%d\tecartes=%d\tecartes_prouves=%d",
		court, st.Payloads, st.Records, st.Bipedes, st.Voisins, st.Sauts, st.Recalages, st.Elections,
		st.Refutations, st.PreuvesContradictoires, st.Glissements, elues, fermes, prouves, ecartes, ecartesProuves)
	t.Logf("%s", s)
	return lignes, s
}

// TestRI27d1M2Equipes : [ScanPlayerTeams] (equipes ti=9, entites du joueur gere) dans le contexte de la
// cuisson, une ligne par film dans RI27C_OUT/m2_equipes.tsv (etiquette `tete`).
func TestRI27d1M2Equipes(t *testing.T) {
	films, racine, sortie := ri27cEnv(t)
	var lignes []string
	for _, court := range films {
		fc := ri27cContexte(t, filepath.Join(racine, court), court, true)
		teams, rep, ents := ScanPlayerTeams(fc)
		cles := make([]int, 0, len(teams))
		for k := range teams {
			cles = append(cles, k)
		}
		sort.Ints(cles)
		eq := ""
		for _, k := range cles {
			eq += fmt.Sprintf("%d:%d,", k, teams[k])
		}
		lignes = append(lignes, fmt.Sprintf("tete\t%s\tpaquets=%d\trecords=%d\tlus=%d\tnon_atteints=%d\tentites=%d\tdoutes=%d\tequipes=%s\trep=%+v",
			court, rep.Packets, rep.Records, rep.Read, rep.Unreached, len(ents.Entities), len(ents.Doutes), eq, rep))
		runtime.GC()
	}
	ri27cEcrire(t, filepath.Join(sortie, "m2_equipes.tsv"), lignes)
}
