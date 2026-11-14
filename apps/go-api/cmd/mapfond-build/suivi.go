package main

// Le suivi d'une recuisson : ce que l'outil a décidé et mesuré carte par carte, et le rapport
// TSV qui le range en colonnes (`-rapport-tsv`), à comparer au sidecar d'avant (`-avant-dir`).

import (
	"context"
	"fmt"
	"image"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"levelup/go-api/internal/mapfondsidecar"

	"github.com/JGtm/halo-infinite-map-images/himap"
)

// Actions de la recuisson, telles que le rapport les écrit.
const (
	actionRecuite  = "recuite"
	actionPlanSeul = "plan seul"
	// actionPublieRogne : plan seul, et le fond publié rogné à la silhouette du maillage.
	actionPublieRogne = "publie rogne"
	actionSautee      = "sautee"
	// actionIgnoree : l'avis complémentaire demande de laisser la carte entièrement telle qu'elle est.
	actionIgnoree = "ignoree (existant garde)"
)

// suiviFond est ce que la recuisson a décidé et mesuré pour une carte.
type suiviFond struct {
	Decision decisionFond
	Action   string
	// PartRognee : part de la matière que le rognage au maillage a retirée ; nil sans rognage.
	PartRognee *float64
	// Plan : le plan écrit au sidecar ; nil quand la carte n'en a pas.
	Plan *planFond
	// Rognage : ce que le rognage du fond publié a retiré ; nil sans rognage.
	Rognage *rognagePublie
}

// planDeCible calcule le plan d'une carte qu'on vient de cuire. Un plan impossible est journalisé
// et rend nil : le fond reste publié sans plan. La sélection des composantes est rejouée sur le
// fond publié ; un écart avec celle de la cuisson est journalisé (la matière du fond rogné n'est
// pas tout à fait celle d'avant le rognage).
func (e *environnement) planDeCible(ctx context.Context, c cible, b himap.BilanCuisson, img *image.RGBA, cal himap.CalageFond) *planFond {
	pf, err := calculePlan(c.sourcePlan, c.maillage, c.ancres, cal, matiereDeImage(img, cal))
	if err != nil {
		slog.ErrorContext(ctx, "mapfond: plan impossible, fond publie sans plan", "err", err, "carte", c.cle)
		return nil
	}
	if pf == nil {
		return nil
	}
	nav := b.Navmesh
	if nav.Gardees+nav.Ecartees > 0 && (nav.Gardees != pf.Retenues.Gardees || nav.Ecartees != pf.Retenues.Ecartees) {
		slog.WarnContext(ctx, "mapfond: selection des composantes du plan differente de celle de la cuisson",
			"carte", c.cle, "cuisson", fmt.Sprintf("%d/%d", nav.Gardees, nav.Ecartees),
			"plan", fmt.Sprintf("%d/%d", pf.Retenues.Gardees, pf.Retenues.Ecartees))
	}
	return pf
}

// partRognee rend la part de matière retirée par le rognage au maillage : cellules effacées sur
// cellules effacées plus matière publiée. Nil quand la cuisson n'a pas rogné.
func partRognee(b himap.BilanCuisson, img *image.RGBA) *float64 {
	if b.Navmesh.Mode == "" {
		return nil
	}
	reste := compteMatiere(img)
	total := b.CellulesHorsNavmesh + reste
	if total == 0 {
		return nil
	}
	p := float64(b.CellulesHorsNavmesh) / float64(total)
	return &p
}

// laissee dit si la carte n'a été ni cuite ni réécrite : l'avis la laisse, ou l'ignore.
func (b bilanAsset) laissee() bool {
	return b.Suivi.Action == actionSautee || b.Suivi.Action == actionIgnoree
}

// bilanSaute est le bilan d'une carte que l'avis laisse intacte.
func bilanSaute(cle string, noms []string, d decisionFond) bilanAsset {
	action := actionSautee
	if d.Ignore {
		action = actionIgnoree
	}
	return bilanAsset{Cle: cle, Noms: noms, Suivi: suiviFond{Decision: d, Action: action}}
}

// ancresSidecar est ce que le rapport lit des statistiques d'un sidecar.
type ancresSidecar struct {
	AvecSol, DansLeCadre int
	Aretes               int
	Noms                 []string
	Lu                   bool
}

// lisSidecarSuivi lit les ancres et le plan d'un sidecar ; `Lu` est faux quand il est absent.
func lisSidecarSuivi(dir, cle string) ancresSidecar {
	s, err := mapfondsidecar.Load(filepath.Join(dir, cle+".json"))
	if err != nil {
		return ancresSidecar{}
	}
	a := ancresSidecar{AvecSol: s.Stats.AnchorsWithGround, DansLeCadre: s.Stats.AnchorsInFrame, Noms: s.MapNames, Lu: true}
	if s.Plan != nil {
		a.Aretes = len(s.Plan.Edges)
	}
	return a
}

// enteteTSV nomme les colonnes du rapport de recuisson.
const enteteTSV = "cle\tnom\tetat_avis\taction\tmode\tmarge_m\tcomposantes_gardees\tcomposantes_ecartees\t" +
	"part_rognee_pct\tancres_sol_avant\tancres_sol_apres\taretes_plan\ta_revoir\tperd_ancres\treprise\teau_retiree\tcellules_finales\tcellules_sauvees_zones"

// ecritRapportTSV écrit une ligne par carte traitée, triées par clé. `avantDir` porte les sidecars
// d'avant la recuisson ; vide, les colonnes « avant » restent vides.
func ecritRapportTSV(chemin, avantDir string, bilans []bilanAsset, env *environnement) error {
	lignes := make([]string, 0, len(bilans))
	sorted := append([]bilanAsset(nil), bilans...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Cle < sorted[j].Cle })
	for _, b := range sorted {
		lignes = append(lignes, ligneTSV(b, avantDir, env))
	}
	contenu := enteteTSV + "\n" + strings.Join(lignes, "\n") + "\n"
	return os.WriteFile(chemin, []byte(contenu), 0o644) //nolint:gosec // rapport de chantier
}

// ligneTSV rend la ligne d'une carte.
func ligneTSV(b bilanAsset, avantDir string, env *environnement) string {
	apres := lisSidecarSuivi(env.sortieDir, b.Cle)
	var avant ancresSidecar
	if avantDir != "" {
		avant = lisSidecarSuivi(avantDir, b.Cle)
	}
	nom := strings.Join(b.Noms, " / ")
	if nom == "" {
		nom = strings.Join(apres.Noms, " / ")
	}
	s := b.Suivi
	action, mode, marge, gardees, ecartees, part := s.Action, "-", "-", "-", "-", "-"
	eau, finales, sauvees := "-", "-", "-"
	switch {
	case b.Err != nil:
		action = "ECHEC : " + strings.ReplaceAll(b.Err.Error(), "\t", " ")
	case s.Action == actionRecuite:
		nav := b.Bilan.Navmesh
		if nav.Mode != "" {
			mode, marge = string(nav.Mode), fmt.Sprintf("%g", nav.Marge)
		}
		gardees, ecartees = fmt.Sprint(nav.Gardees), fmt.Sprint(nav.Ecartees)
		eau, finales, sauvees = fmt.Sprint(nav.EauRetiree), fmt.Sprint(nav.CellulesFinales), fmt.Sprint(b.Bilan.CellulesSauveesParPlan)
	case (s.Action == actionPlanSeul || s.Action == actionPublieRogne) && s.Plan != nil:
		gardees, ecartees = fmt.Sprint(s.Plan.Retenues.Gardees), fmt.Sprint(s.Plan.Retenues.Ecartees)
	}
	if s.Rognage != nil {
		mode, marge = string(s.Decision.Mode), fmt.Sprintf("%g", s.Rognage.Marge)
	}
	if s.PartRognee != nil {
		part = fmt.Sprintf("%.1f", 100**s.PartRognee)
	}
	perd := "non"
	if avant.Lu && apres.Lu && apres.AvecSol < avant.AvecSol {
		perd = fmt.Sprintf("OUI (%d -> %d)", avant.AvecSol, apres.AvecSol)
	}
	return strings.Join([]string{
		b.Cle, nom, s.Decision.Etat, action, mode, marge, gardees, ecartees, part,
		formatAncres(avant), formatAncres(apres), fmt.Sprint(apres.Aretes), ouiNon(s.Decision.ARevoir), perd,
		ouiNon(s.Decision.Reprise), eau, finales, sauvees,
	}, "\t")
}

// formatAncres rend « avec sol / dans le cadre », ou « - » pour un sidecar absent.
func formatAncres(a ancresSidecar) string {
	if !a.Lu {
		return "-"
	}
	return fmt.Sprintf("%d/%d", a.AvecSol, a.DansLeCadre)
}

func ouiNon(v bool) string {
	if v {
		return "oui"
	}
	return "non"
}
