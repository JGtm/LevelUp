//go:build research && campagne_overlay

package grammar

// campagne_bis2_positions_research_test.go — MESURES BIS 2 DE LA CAMPAGNE DE GRAMMAIRE
// (2026-10-01), POINT 25 DE LA CRITIQUE DE COMPLETUDE : LES A/B DE POSITION DE LA NOTE T4.
//
// CE FICHIER EXIGE LA SURCOUCHE DE RECHERCHE (tag `campagne_overlay`). Les treize sites
// d exception, la queue de waypointstate et la lecture par index de plage vivent dans trois
// fichiers de production (`capture.go`, `lecteur_position.go`, `lecteur_position_exceptions.go`)
// qu aucune copie de la marche ne peut atteindre : le dispatch des composants les appelle. Leurs
// COPIES DE RECHERCHE sont dans
// `.ai/V7.5/film_re/campagne_grammaire_2026-10-01/mesures_bis2_overlay/` et `go test -overlay`
// les substitue aux fichiers du depot POUR LA SEULE COMPILATION DU TEST : aucun fichier de
// production n est modifie sur disque, `grammar.Rev` ne bouge pas. Sans bascule, chaque copie lit
// exactement comme la production (controle : la variante `reference` rend la carte v2).
//
// La marche est [cmMarcher] (copie de recherche de [FrameClosureDetaillee]). Deux contextes :
//
//	instruments  : [cmOuvrir] (le contexte de la carte v2, decoupage d i0 auto-detecte, region 0)
//	production   : `NewFilmContextForMap` + l entree du catalogue de la carte et sa pose de
//	               largeurs, comme `replay.installWorldObjectPrecision` (sans le profil que
//	               `killsource` calibre) — la region JOUEE est celle du catalogue (Live Fire : 1)
//
// Rejouable (un film a la fois, plafond 4 Gio) :
//
//	go test -tags=research,campagne_overlay -overlay=<json> -count=1 -timeout 180m \
//	  -run '^TestCampagneBis2Positions$' ./internal/games/halo_infinite/film/internal/grammar/

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/filmproc"
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// b2pVariante : une configuration de la surcouche.
type b2pVariante struct {
	nom      string
	sites    [bis2NombreDeSites]bool
	waypoint bool // R(1) de queue de waypointstate
}

// b2pVariantes rend la reference, un site a la fois, les deux moities de waypointstate et
// « tous les sites ».
func b2pVariantes() []b2pVariante {
	out := []b2pVariante{{nom: "reference"}}
	for s := 0; s < bis2NombreDeSites; s++ {
		v := b2pVariante{nom: "jeu:" + bis2NomsDeSites[s]}
		v.sites[s] = true
		v.waypoint = s == bis2SiteWaypoint
		out = append(out, v)
	}
	wp := b2pVariante{nom: "jeu:waypoint-position-seule"}
	wp.sites[bis2SiteWaypoint] = true
	out = append(out, wp, b2pVariante{nom: "jeu:waypoint-queue-seule", waypoint: true})
	tous := b2pVariante{nom: "jeu:tous", waypoint: true}
	for s := range tous.sites {
		tous.sites[s] = true
	}
	return append(out, tous)
}

// poser installe la variante dans la surcouche.
func (v b2pVariante) poser() {
	bis2SitesJeu = v.sites
	bis2WaypointQueue = v.waypoint
	bis2QueuesSansCondition = false
}

// b2pComparateur compare une marche a la reference, paquet par paquet.
type b2pComparateur struct {
	ref                        map[[2]int]int // paquet ferme -> utiles fermes
	gagnes, perdus             int
	utilesGagnes, utilesPerdus int
	statut                     map[[2]int]int
	c3                         *b2pReleveC3
}

func (c *b2pComparateur) debutDeChunk(int, []byte, []FilmPacket, *World) {
	if bis2C3 != nil {
		bis2C3.evenements = bis2C3.evenements[:0]
	}
}
func (c *b2pComparateur) finDeFilm() {}

func (c *b2pComparateur) paquet(_ int, p *cmPaquet, _ *World) {
	cle := [2]int{p.d.Chunk, p.d.Index}
	if p.d.Fermee {
		c.statut[cle] = p.utilesFermes
	}
	if c.c3 != nil && bis2C3 != nil {
		c.c3.noter(p, bis2C3.evenements)
		bis2C3.evenements = bis2C3.evenements[:0]
	}
	if c.ref == nil {
		return
	}
	u, avant := c.ref[cle]
	switch {
	case p.d.Fermee && !avant:
		c.gagnes++
		c.utilesGagnes += p.utilesFermes
	case !p.d.Fermee && avant:
		c.perdus++
		c.utilesPerdus += u
	}
}

// b2pReleveC3 : les index de plage lus, par composant et par index, ventiles par paquet ferme ou
// non ; et les paquets FERMES qui portent un index de plage impossible (hors de la plage jouee sur
// une carte a une plage).
type b2pReleveC3 struct {
	lus                                 map[string][3]int // cle composant|idx -> lus, dans fermes, paquets fermes
	fermesAvecImpossible, impossiblesNF int
	paquetsAvecIndexHorsRegion, plages  int
}

func (r *b2pReleveC3) noter(p *cmPaquet, ev []bis2EvenementIndex) {
	horsRegion := false
	vu := map[string]bool{}
	for _, e := range ev {
		cle := fmt.Sprintf("ti=%d %s|%d", e.ti, e.composant, e.idx)
		x := r.lus[cle]
		x[0]++
		if p.d.Fermee {
			x[1]++
			if !vu[cle] {
				x[2]++
			}
		}
		vu[cle] = true
		r.lus[cle] = x
		if e.idx >= 0 && uint32(e.idx) != e.region {
			horsRegion = true
		}
	}
	if horsRegion {
		r.paquetsAvecIndexHorsRegion++
		if r.plages <= 1 {
			if p.d.Fermee {
				r.fermesAvecImpossible++
			} else {
				r.impossiblesNF++
			}
		}
	}
}

// b2pMesure marche un film sous une variante et rend la ligne de sortie.
func b2pMesure(f *cmFilm, v b2pVariante, ref map[[2]int]int, c3 *b2pReleveC3, j *cmJuge) (string, map[[2]int]int) {
	v.poser()
	defer b2pVariante{}.poser()
	cmp := &b2pComparateur{ref: ref, statut: map[[2]int]int{}, c3: c3}
	var ec cmEcouteur = cmp
	if j != nil {
		ec = cmMux{cmp, j}
	}
	rep, _, _ := cmMarcher(f, cmVariante{}, ec)
	kf, err := KeyframeClosure(f.fc)
	kfF, kfT, k38, k42 := 0, 0, [2]int{}, [2]int{}
	if err == nil {
		for ti, s := range kf {
			kfF += s.Closed
			kfT += s.Total
			switch ti {
			case 38:
				k38 = [2]int{s.Closed, s.Total}
			case 42:
				k42 = [2]int{s.Closed, s.Total}
			}
		}
	}
	return fmt.Sprintf("%s\t%s\t%s\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d",
		f.id, f.build, v.nom, rep.Paquets, rep.PaquetsFermes, rep.Utiles.RecordsFermes, rep.Utiles.Records,
		cmp.gagnes, cmp.perdus, cmp.utilesGagnes, cmp.utilesPerdus, kfF, kfT, k38[0], k38[1], k42[0], k42[1]), cmp.statut
}

const b2pEntete = "film\tbuild\tvariante\tpaquets\tfermes\tutiles_fermes\tutiles_lus\tpaquets_gagnes\t" +
	"paquets_perdus\tutiles_gagnes\tutiles_perdus\timage_cle_fermes\timage_cle_bornes\tti38_fermes\t" +
	"ti38_bornes\tti42_fermes\tti42_bornes"

// TestCampagneBis2Positions : A/B par site d exception, contexte des instruments.
func TestCampagneBis2Positions(t *testing.T) {
	racine, sortie, films := b2Env(t)
	utiles := cmUtiles(t)
	lignes := []string{b2pEntete}
	for _, id := range films {
		garde := filmproc.Arm("campagne/bis2-positions", 4, func(pic uint64) {
			fmt.Fprintf(os.Stderr, "%s : plafond memoire franchi (%d octets)\n", id, pic)
			os.Exit(3)
		})
		debut := time.Now()
		f, ok := cmOuvrir(t, racine, id, utiles)
		if !ok {
			garde.Disarm()
			continue
		}
		var ref map[[2]int]int
		for _, v := range b2pVariantes() {
			l, statut := b2pMesure(f, v, ref, nil, nil)
			if ref == nil {
				ref = statut
			}
			lignes = append(lignes, l)
		}
		t.Logf("%s %s : %d variantes ; pic %d Mio, %s", id, f.build, len(b2pVariantes()),
			garde.Peak()>>20, time.Since(debut).Round(time.Second))
		garde.Disarm()
	}
	b2Ecrire(t, sortie, "mb2_positions.tsv", lignes)
}

// b2pOuvrirProduction ouvre un film sous le contexte de la cuisson pour sa carte.
func b2pOuvrirProduction(t *testing.T, racine, id string, entry profile.MapQuantEntry,
	utiles UsagesProduit) (*cmFilm, bool) {
	t.Helper()
	film, err := source.LoadDir(racine+"/"+id, nil)
	if err != nil {
		t.Errorf("%s : %v", id, err)
		return nil, false
	}
	fc := NewFilmContextForMap(film, &entry, nil)
	bal := fc.ProfilDeBalayage()
	fc.NoterReplis(bal.PoserLargeursObjetDuMondeDepuisDecoupage(entry.Layout()))
	fc.PoserProfilDeBalayage(bal)
	reg, err := fc.Registry()
	if err != nil {
		t.Errorf("%s : registre : %v", id, err)
		return nil, false
	}
	cfg := fc.CadreDeBalayage()
	return &cmFilm{id: id, build: cmBuild(fc), fc: fc, reg: reg, cfg: cfg, utiles: utiles}, true
}

// b2pBornes lit CAMPAGNE_BORNES_<module> : `idx:minx,miny,minz,maxx,maxy,maxz;...`.
func b2pBornes(module string) map[int][3][2]float32 {
	brut := os.Getenv("CAMPAGNE_BORNES_" + module)
	if brut == "" {
		return nil
	}
	out := map[int][3][2]float32{}
	for _, morceau := range strings.Split(brut, ";") {
		idx, reste, ok := strings.Cut(morceau, ":")
		if !ok {
			continue
		}
		i, err := strconv.Atoi(strings.TrimSpace(idx))
		if err != nil {
			continue
		}
		var v [6]float32
		for k, x := range strings.Split(reste, ",") {
			if k < 6 {
				f, _ := strconv.ParseFloat(strings.TrimSpace(x), 32)
				v[k] = float32(f)
			}
		}
		out[i] = [3][2]float32{{v[0], v[3]}, {v[1], v[4]}, {v[2], v[5]}}
	}
	return out
}

// TestCampagneBis2PositionsProduction : reference et « tous les sites » sous le contexte de la
// cuisson, le releve des index de plage (T4-C3) et, quand les bornes des autres plages sont
// fournies, la lecture par index.
func TestCampagneBis2PositionsProduction(t *testing.T) {
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
	lignes := []string{b2pEntete + "\tfermes_contredits\tgagnes_contredits\tregion\tindex_bits\tplages_connues"}
	idx := []string{"film\tbuild\tvariante\tcomposant\tindex\tlus\tlus_dans_fermes\tpaquets_fermes"}
	imp := []string{"film\tbuild\tvariante\tregion\tplages\tpaquets_avec_index_hors_region\t" +
		"fermes_avec_index_impossible\tnon_fermes_avec_index_impossible"}
	defer func() { bis2C3 = nil }()
	for _, id := range films {
		entry, err := cat.Lookup(cartes[id])
		if err != nil {
			t.Logf("%s : carte %q hors catalogue (%v) — film ignore", id, cartes[id], err)
			continue
		}
		garde := filmproc.Arm("campagne/bis2-positions-prod", 4, func(pic uint64) {
			fmt.Fprintf(os.Stderr, "%s : plafond memoire franchi (%d octets)\n", id, pic)
			os.Exit(3)
		})
		debut := time.Now()
		f, ok := b2pOuvrirProduction(t, racine, id, entry, utiles)
		if !ok {
			garde.Disarm()
			continue
		}
		blocs := cmLireBlocs(f)
		bornes := b2pBornes(entry.Module)
		// Le nombre de plages : la plage jouee plus celles dont les bornes sont fournies (les tags
		// sbsp de la carte, cf. `cmd_fermeture/bis2_regions_research_test.go`).
		plages := 1 + len(bornes)
		vs := []b2pVariante{{nom: "reference"}, b2pVariantes()[len(b2pVariantes())-1]}
		var ref map[[2]int]int
		run := func(v b2pVariante, parIndex bool) {
			bis2C3 = &bis2EtatC3{parIndex: parIndex, bornes: bornes}
			rel := &b2pReleveC3{lus: map[string][3]int{}, plages: plages}
			nom := v.nom
			if parIndex {
				nom += "+lecture-par-index"
				v.nom = nom
			}
			var refB map[[2]int]bool
			if ref != nil {
				refB = map[[2]int]bool{}
				for k := range ref {
					refB[k] = true
				}
			}
			juge := cmNouveauJuge(f, blocs, refB)
			l, statut := b2pMesure(f, v, ref, rel, juge)
			l += fmt.Sprintf("\t%d\t%d", juge.fermesContre, juge.gagnesContredits)
			if ref == nil {
				ref = statut
			}
			lignes = append(lignes, fmt.Sprintf("%s\t%d\t%d\t%d", l, entry.Region, entry.EffectiveRegionIndexBits(), len(bornes)))
			for cle, x := range rel.lus {
				comp, i, _ := strings.Cut(cle, "|")
				idx = append(idx, fmt.Sprintf("%s\t%s\t%s\t%s\t%s\t%d\t%d\t%d", id, f.build, nom, comp, i, x[0], x[1], x[2]))
			}
			imp = append(imp, fmt.Sprintf("%s\t%s\t%s\t%d\t%d\t%d\t%d\t%d", id, f.build, nom, entry.Region, plages,
				rel.paquetsAvecIndexHorsRegion, rel.fermesAvecImpossible, rel.impossiblesNF))
		}
		for _, v := range vs {
			run(v, false)
		}
		if len(bornes) > 0 {
			for _, v := range vs {
				run(v, true)
			}
		}
		bis2C3 = nil
		t.Logf("%s %s (%s, region %d) : pic %d Mio, %s", id, f.build, cartes[id], entry.Region,
			garde.Peak()>>20, time.Since(debut).Round(time.Second))
		garde.Disarm()
	}
	b2Ecrire(t, sortie, "mb2_positions_production.tsv", lignes)
	b2Ecrire(t, sortie, "mb2_index_de_plage.tsv", idx)
	b2Ecrire(t, sortie, "mb2_index_impossibles.tsv", imp)
}
