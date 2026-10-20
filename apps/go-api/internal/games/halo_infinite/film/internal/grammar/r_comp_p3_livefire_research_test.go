//go:build research && campagne_overlay

package grammar

// r_comp_p3_livefire_research_test.go — CHANTIER « comp », R-P3, coupable G4 (`DELTA ti=41
// dernier composant i2 object-forward-and-up`, 158 des 161 paquets sur Live Fire `0797ce72`).
//
// MESURES_BIS_3 §3.4 a mesure G4 INCHANGE sous la lecture par index de plage. Mais cette lecture
// ne s applique qu aux lecteurs qui passent par les tables de position ([lireE524Sur]) : la
// position d objet du monde (`ti=41 i0 object-position-component`, `FUN_14076e29c`) est une
// EXCEPTION DATEE du portage unique ([consumeObjectPositionMonde]) qui lit l index puis les
// largeurs de LA plage de la carte quel que soit l index lu. Ce test joue, sous le contexte de la
// cuisson (catalogue de cartes), la position d objet du monde LUE COMME LE JEU (site
// `world-object-i0` de la surcouche) avec et sans la lecture par index.
//
//	CAMPAGNE_CATALOGUE=<map_quant_bounds.json> CAMPAGNE_CARTES="0797ce72=Live Fire;60ae07c4=Live Fire - Ranked"
//	CAMPAGNE_BORNES_<module>=<idx:minx,miny,minz,maxx,maxy,maxz;...> \
//	  go test -tags=research,campagne_overlay -overlay=<json> -run '^TestRCompP3LiveFire$' ...

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
)

// TestRCompP3LiveFire ecrit `r_comp_p3_livefire.tsv` et `r_comp_p3_livefire_tables.tsv`.
func TestRCompP3LiveFire(t *testing.T) {
	racine, sortie, films := b2Env(t)
	cat, err := profile.LoadMapQuantCatalog(os.Getenv("CAMPAGNE_CATALOGUE"))
	if err != nil {
		t.Fatalf("catalogue : %v", err)
	}
	cartes := map[string]string{}
	for _, x := range strings.Split(os.Getenv("CAMPAGNE_CARTES"), ";") {
		if id, carte, ok := strings.Cut(x, "="); ok {
			cartes[strings.TrimSpace(id)] = strings.TrimSpace(carte)
		}
	}
	utiles := cmUtiles(t)
	tete := "film\tbuild\tvariante\tpaquets\tfermes\tfermes_sains\tutiles_fermes_sains\thors_cadre\tgagnes\tperdus\t" +
		"gagnes_contredits\tperdus_sains"
	for _, g := range rp3Gates {
		tete += "\t" + g.nom
	}
	lignes := []string{tete}
	tabs := []string{"film\tbuild\tvariante\ttable\tcle\tn\tpaquets\thors_cadre\tfermes\ten_jeu"}
	defer func() { bis2C3 = nil; b2pVariante{}.poser() }()
	type variante struct {
		nom             string
		monde, parIndex bool
	}
	vs := []variante{{"production", false, false}, {"production + lecture par index", false, true},
		{"production + world-object au jeu", true, false}, {"production + world-object au jeu + lecture par index", true, true}}
	for _, id := range films {
		entry, err := cat.Lookup(cartes[id])
		if err != nil {
			t.Logf("%s : carte %q hors catalogue (%v)", id, cartes[id], err)
			continue
		}
		f, ok := b2pOuvrirProduction(t, racine, id, entry, utiles)
		if !ok {
			continue
		}
		b := cmLireBlocs(f)
		bornes := b2pBornes(entry.Module)
		var jref *cmJuge
		var ref map[[2]int]bool
		for _, v := range vs {
			pv := b2pVariante{nom: v.nom}
			pv.sites[bis2SiteWorldObject] = v.monde
			pv.poser()
			bis2C3 = &bis2EtatC3{parIndex: v.parIndex, bornes: bornes}
			d := b3NouveauDiag(f, b, false)
			diag := &rp3Diag{f: f, b: b, t: cmTables{}, oracle: map[[2]int][]cmLiaison{}, imp: map[[2]int][]uint32{}}
			var j *cmJuge
			if jref == nil {
				j = cmNouveauJuge(f, b, nil)
			} else {
				j = cmNouveauJuge(f, b, ref)
				j.contreRef, j.utilesRef = jref.contreRef, jref.utilesRef
			}
			st := &cmComparateur{ref: map[[2]int]bool{}}
			r, _, _ := cmMarcher(f, cmVariante{}, cmMux{d, diag, j, b3Statut{st}})
			bis2C3 = nil
			b2pVariante{}.poser()
			if jref == nil {
				jref, ref = j, st.ref
			}
			ps, us, perdusSains := rl3Sains(j)
			l := fmt.Sprintf("%s\t%s\t%s\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d", id, f.build, v.nom, r.Paquets, r.PaquetsFermes,
				ps, us, r.Bloquants[CauseHorsCadre].Paquets, j.gagnes, j.perdus, j.gagnesContredits, perdusSains)
			for _, g := range rp3Gates {
				n := 0
				if x := d.t["dernier_composant_x_classe"][g.cle]; x != nil {
					n = x.horsCadre
				}
				l += fmt.Sprintf("\t%d", n)
			}
			lignes = append(lignes, l)
			for nom, m := range diag.t {
				for _, cle := range cmCles(m) {
					x := m[cle]
					tabs = append(tabs, fmt.Sprintf("%s\t%s\t%s\t%s\t%s\t%d\t%d\t%d\t%d\t%d", id, f.build, v.nom, nom, cle,
						x.n, x.paquets, x.horsCadre, x.fermes, x.enJeu))
				}
			}
		}
	}
	b2Ecrire(t, sortie, "r_comp_p3_livefire.tsv", lignes)
	b2Ecrire(t, sortie, "r_comp_p3_livefire_tables.tsv", tabs)
}
