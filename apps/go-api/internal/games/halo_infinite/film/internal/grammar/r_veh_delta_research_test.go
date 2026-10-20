//go:build research && campagne_overlay

package grammar

// r_veh_delta_research_test.go — CAMPAGNE DE GRAMMAIRE, RECHERCHE R-L4 (2026-10-02) : l effet sur
// la CARTE DE FERMETURE v2 (paquets delta) des trois constats de l image-cle `ti=40`. Mesure
// seulement ; exige la surcouche `r_veh_overlay/` (cinq copies de recherche, dont `traverse.go`
// et `default_state_ti40.go`).
//
// Variantes (chaque film, contexte des instruments, juge des invariants de BIS_1 sur chaque
// marche, reference = la premiere) :
//
//	reference                      la carte v2
//	ti40-composants                les 16 composants `ti=40` lus par le crochet (T6 §4), porte
//	                               `+0x818` posee (en delta, l annonce de i33/i34 la prouve,
//	                               FUN_142f09c74)
//	portee-neuf                    tout record NEW lu sous DAT_144e61ea0 = 1 (FUN_142e309b4 et
//	                               FUN_142e31a0c la posent pour tout le record) et chemin absolu
//	                               d i0 de l ecrivain — TOUS les archetypes
//	ti40-composants+portee-neuf
//	mpp8/3                         decoupage MPP 8/3 (formats <= 25 ; temoin negatif sur 27)
//	mpp8/3+ti40-composants+portee-neuf
//
//	go test -tags=research,campagne_overlay -overlay=<json> -count=1 -timeout 180m \
//	  -run '^TestRVehDeltaTi40$' ./internal/games/halo_infinite/film/internal/grammar/

import (
	"fmt"
	"os"
	"testing"
	"time"

	"levelup/go-api/internal/filmproc"
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
)

// rvdVariante : une marche de la carte.
type rvdVariante struct {
	nom                     string
	ti40, neuf, mp83, feuil bool
}

// rvdVariantes : de la reference au cumul.
func rvdVariantes() []rvdVariante {
	return []rvdVariante{
		{nom: "reference"},
		{nom: "ti40-composants", ti40: true},
		{nom: "portee-neuf", neuf: true},
		{nom: "ti40-composants+portee-neuf", ti40: true, neuf: true},
		{nom: "ti40-composants+feuille4-brute", ti40: true, feuil: true},
		{nom: "mpp8/3", mp83: true},
		{nom: "mpp8/3+ti40-composants", mp83: true, ti40: true},
		{nom: "mpp8/3+ti40-composants+portee-neuf", mp83: true, ti40: true, neuf: true},
		{nom: "mpp8/3+ti40-composants+feuille4-brute", mp83: true, ti40: true, feuil: true},
	}
}

// TestRVehDeltaTi40 : la carte v2 sous chaque variante, gagnes / perdus et juge des invariants.
func TestRVehDeltaTi40(t *testing.T) {
	racine, sortie, films := b2Env(t)
	utiles := cmUtiles(t)
	lignes := []string{"film\tbuild\tvariante\tpaquets\tfermes\tutiles_fermes\tutiles_lus\thors_cadre\t" +
		"paquets_gagnes\tpaquets_perdus\tutiles_gagnes\tfermes_contredits\tgagnes_contredits\tcause_ti40_paquets"}
	defer func() { bis2Intercepteur, b2vPorte, rvehPorteeNeuf = nil, false, false }()
	for _, id := range films {
		garde := filmproc.Arm("campagne/r-veh-delta", 4, func(pic uint64) {
			fmt.Fprintf(os.Stderr, "%s : plafond memoire franchi (%d octets)\n", id, pic)
			os.Exit(3)
		})
		debut := time.Now()
		f, ok := cmOuvrir(t, racine, id, utiles)
		if !ok {
			garde.Disarm()
			continue
		}
		blocs := cmLireBlocs(f)
		var ref map[[2]int]bool
		for _, v := range rvdVariantes() {
			fv := *f
			if v.mp83 {
				fv.cfg.Profil.MPP = profile.MPPWidths{Lead: 8, Index: 3}
			}
			bis2Intercepteur, b2vPorte, rvehPorteeNeuf, rvehPorteeEtat = nil, false, v.neuf, v.feuil
			if v.ti40 {
				bis2Intercepteur, b2vPorte = b2vCrochet(true, false), true
			}
			e := &b2vFenetre{variante: v.nom, film: id, statut: map[[2]int]bool{}, causes: map[string]int{},
				pertes: &b2vPertes{ref: ref}}
			juge := cmNouveauJuge(&fv, blocs, ref)
			rep, _, _ := cmMarcher(&fv, cmVariante{}, cmMux{e, juge})
			bis2Intercepteur, b2vPorte, rvehPorteeNeuf, rvehPorteeEtat = nil, false, false, false
			if ref == nil {
				ref = e.statut
			}
			n40 := 0
			for cause, k := range e.causes {
				if len(cause) >= 5 && cause[:5] == "ti=40" {
					n40 += k
				}
			}
			lignes = append(lignes, fmt.Sprintf("%s\t%s\t%s\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d", id, f.build, v.nom,
				rep.Paquets, rep.PaquetsFermes, rep.Utiles.RecordsFermes, rep.Utiles.Records,
				rep.Bloquants[CauseHorsCadre].Paquets, e.pertes.gagnes, e.pertes.perdus, e.pertes.utilesGagnes,
				juge.fermesContre, juge.gagnesContredits, n40))
			t.Logf("%s %-38s fermes %6d / %6d ; +%d / -%d ; contredits %d (gagnes %d)", id, v.nom, rep.PaquetsFermes,
				rep.Paquets, e.pertes.gagnes, e.pertes.perdus, juge.fermesContre, juge.gagnesContredits)
		}
		t.Logf("%s %s : pic %d Mio, %s", id, f.build, garde.Peak()>>20, time.Since(debut).Round(time.Second))
		garde.Disarm()
	}
	b2Ecrire(t, sortie, "r_veh_delta.tsv", lignes)
}
