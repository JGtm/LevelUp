//go:build research

package killsource

// ri27c_calibration_research_test.go — LES MESURES DE LA CALIBRATION, DES KILL-EVENTS ET DU DECOUPAGE
// MPP DECLARE QUI OUVRENT 2.7.c (plan de l etape 2 de la representation intermediaire, item 2.7.c0),
// suite de `ri27c_research_test.go` : lignes C, CK, E et variante A4 de `TestRI27cKillsource`.

import (
	"fmt"
	"strconv"
	"strings"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// ri27cCandidats rend les candidats de la calibration de killsource, dans l ordre de la grammaire :
// les 21 largeurs d axe uniformes (mot de poignee a l invariant), puis les trois mots de poignee au
// triplet de la carte.
func ri27cCandidats(base grammar.ProfilDeBalayage) ([]string, []grammar.FrameConfig) {
	lues, invariant := base.Mouvement.WorldObject, grammar.ProfilDeBalayageParDefaut().Mouvement.Traversal
	var noms []string
	var cfgs []grammar.FrameConfig
	for aw := axisWMin; aw <= axisWMax; aw++ {
		cfg := grammar.DefaultFrameConfig()
		cfg.Profil = base
		cfg.Profil.Mouvement.WorldObject = lues
		cfg.Profil.Mouvement.WorldObject.AxisW = [3]uint{aw, aw, aw}
		cfg.Profil.Mouvement.Traversal = invariant
		noms, cfgs = append(noms, fmt.Sprintf("axe=%d", aw)), append(cfgs, cfg)
	}
	for iw := indexWMin; iw <= indexWMax; iw++ {
		cfg := grammar.DefaultFrameConfig()
		cfg.Profil = base
		cfg.Profil.Mouvement.WorldObject = lues
		cfg.Profil.Mouvement.Traversal.IndexW = iw
		cfg.Profil.Mouvement.Traversal.AxisW = invariant.AxisW
		noms, cfgs = append(noms, fmt.Sprintf("poignee=%d", iw)), append(cfgs, cfg)
	}
	return noms, cfgs
}

// ri27cCalibration rend les lignes C : les scores de chaque candidat sous la timeline de killsource
// et sous le monde des preliminaires de la marche (rendus par la grammaire), et la calibration
// retenue.
func ri27cCalibration(court string, c *decodeCtx, tl *timeline, g ri27cGrammaire) []string {
	base := c.calib.Profil
	base.Mouvement.Traversal = grammar.ProfilDeBalayageParDefaut().Mouvement.Traversal
	noms, cfgs := ri27cCandidats(base)
	ech := calibSample(c.film, calibSampleSize)
	var lignes []string
	for k := range cfgs {
		tl.rewind()
		lignes = append(lignes, fmt.Sprintf("C\t%s\t%s\t%d\t%d", court, noms[k],
			countBipedRecords(ech, tl, cfgs[k], c.opts.Views), g.calib[noms[k]]))
	}
	return append(lignes, fmt.Sprintf("CK\t%s\t%d\t%s\t%s\t%s", court, len(ech), g.calibVus, g.calibTaille,
		c.calib.String()))
}

// ri27cKillEvents rend la ligne E : les kill-events de killsource contre les messages de genre 85 de
// la vue A unique.
func ri27cKillEvents(court string, c *decodeCtx, g ri27cGrammaire) []string {
	tsDe := map[[2]int]uint64{}
	for i := range c.film.t0 {
		p := &c.film.t0[i]
		tsDe[[2]int{p.chunk, p.idx}] = p.ts
	}
	var egaux, differents, arreteeAvant, ksAutre int
	vus := map[[2]uint64]bool{}
	for _, r := range c.killEvents.recs {
		ts := tsDe[[2]int{r.chunk, r.pidx}]
		cle := [2]uint64{ts, uint64(r.bit)} //nolint:gosec // position dans un payload
		k, ok := g.kills[cle]
		if !ok {
			if p, lu := g.paquetsVA[ts]; lu && len(p) >= 3 && ri27cI(p[2]) <= r.bit {
				arreteeAvant++
			} else {
				ksAutre++
			}
			continue
		}
		vus[cle] = true
		f := r.fields
		attendu := []string{strconv.Itoa(f.victim), strconv.Itoa(f.killer), strconv.FormatUint(uint64(f.killerPct), 10),
			strconv.Itoa(f.flag), strconv.Itoa(f.assist), strconv.FormatUint(uint64(f.assistPct), 10)}
		if strings.Join(attendu, ",") == strings.Join(k[:6], ",") {
			egaux++
		} else {
			differents++
		}
	}
	var grAcceptes, grRefuses, grSeulsAcceptes, grSeulsRefuses int
	for cle, k := range g.kills {
		acc := len(k) > 6 && k[6] == "true"
		if acc {
			grAcceptes++
		} else {
			grRefuses++
		}
		if !vus[cle] {
			if acc {
				grSeulsAcceptes++
			} else {
				grSeulsRefuses++
			}
		}
	}
	portes := 0
	for _, p := range g.paquetsVA {
		if len(p) > 0 && p[0] == "true" {
			portes++
		}
	}
	return []string{fmt.Sprintf("E\t%s\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%v\t%s", court,
		len(c.killEvents.recs), egaux, differents, arreteeAvant, ksAutre, grAcceptes, grRefuses, grSeulsAcceptes,
		grSeulsRefuses, len(g.paquetsVA), portes, c.killEvents.gate15, strings.Join(g.variante, "/"))}
}

// ri27cDecoupageDeclare rend le decoupage MPP que le film declare, et s il en declare un.
func ri27cDecoupageDeclare(src *source.Film) (grammar.ProfilDeBalayage, bool) {
	fc := grammar.NewFilmContext(src)
	res := fc.ResolutionMPP()
	var p grammar.ProfilDeBalayage
	if !res.Decide() {
		return p, false
	}
	p.MPP = res.Widths
	return p, true
}

// ri27cSousDecoupageDeclare rejoue la calibration puis la marche de killsource sous le decoupage
// declare (variante A4) ; nil quand le film ne declare rien.
func ri27cSousDecoupageDeclare(c *decodeCtx, src *source.Film, tl *timeline) (*walkResult, *calibration) {
	d, ok := ri27cDecoupageDeclare(src)
	if !ok {
		return nil, nil
	}
	profil, carteLue := ProfilDeDepartPourCarte(c.opts.Carte)
	profil, corrLu := grammar.GrammaireSousFilm(profil, src)
	profil.MPP = d.MPP
	abs := profil.LargeursObjetDuMonde()
	cal := calibration{Profil: profil, CarteLue: carteLue, ControleDeCorruptionLu: corrLu, LueAxisW: abs.AxisW,
		LueIndexW: abs.IndexW, VueA: grammar.VueADuFilmSousCarte(src, c.opts.Carte)}
	tl.rewind()
	infererLargeurs(c.film, tl, c.opts.Views, &cal)
	return runWalk(c.film, tl, c.roster, c.opts.Views, &cal), &cal
}
