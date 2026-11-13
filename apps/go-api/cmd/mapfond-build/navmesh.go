package main

// Le maillage de navigation dans la cuisson : ce qu'on en charge, ce qu'on en fait, et le plan
// des murs qu'on en publie au sidecar.

import (
	"context"
	"errors"
	"log/slog"
	"math"

	"levelup/go-api/internal/domain/replaydoc"

	"github.com/JGtm/halo-infinite-map-images/himap"

	"github.com/JGtm/halo-infinite-files-reader/hinavmesh"
)

const (
	// margeNavmeshNatifM : dilatation du masque de rognage d'une carte native, en mètres.
	// `himap.MargeNavmesh` (3 m) a été mesurée sur les cartes Forge ; sur une carte native à
	// zones de callout, 1,5 m garde les murs qui bordent le sol sans garder les masses qui le
	// touchent.
	margeNavmeshNatifM = 1.5

	// sourcePlanNatif et sourcePlanForge nomment l'origine du maillage dans `plan.source`.
	sourcePlanNatif = "navmesh module any"
	sourcePlanForge = "navmesh.blob"

	// degradationSansMaillage : texte du bilan d'une carte dont le module `any` ne porte aucun
	// maillage de navigation.
	degradationSansMaillage = "aucun maillage de navigation dans le module any"

	// millimetre : le plan est publié au millimètre, ce qui tient le sidecar à une taille
	// raisonnable sans que le trait se voie à aucune échelle du fond.
	millimetre = 1000.0
)

// sansNavmeshDe dit si cette carte renonce au maillage de navigation. Journalisé : c'est ce qui
// décide si la carte est rognée à son maillage et si elle publie un plan.
func (e *environnement) sansNavmeshDe(cle string) bool {
	if e.reglages == nil {
		return false
	}
	c, ok := e.reglages.Cartes[cle]
	if !ok || !c.SansNavmesh {
		return false
	}
	slog.Info("mapfond: maillage de navigation desarme pour cette carte", "carte", cle, "gateLe", c.GateLe)
	return true
}

// optionsNavmeshNatif rend ce qu'une carte native fait de son maillage : rognage à 1,5 m, sans
// tolérance ni niveau haut, sauf réglage de la carte (`rogneAuNavmesh`, `margeNavmesh`,
// `toleranceNavmesh`, `maillageNiveauHaut`). `rogneAuNavmesh: false` garde le maillage comme
// référence sans rogner ; `sansNavmesh` le retire entièrement.
func (e *environnement) optionsNavmeshNatif(cle string, m *hinavmesh.Maillage) himap.OptionsNavmesh {
	marge := e.margeNavmeshDe(cle)
	if marge <= 0 {
		marge = margeNavmeshNatifM
	}
	return himap.OptionsNavmesh{
		Maillage:   m,
		Rogne:      e.rogneAuNavmeshNatifDe(cle),
		Marge:      marge,
		Tolerance:  e.toleranceNavmeshDe(cle),
		NiveauHaut: e.maillageNiveauHautDe(cle),
	}
}

// optionsNavmeshForge rend les réglages navmesh d'une carte Forge : ceux du fichier de réglages,
// rien d'autre. Le maillage est nil quand la carte ne prend pas sa référence sur lui.
func (e *environnement) optionsNavmeshForge(cle string, m *hinavmesh.Maillage) himap.OptionsNavmesh {
	return himap.OptionsNavmesh{
		Maillage:   m,
		Rogne:      e.rogneAuNavmeshDe(cle),
		Marge:      e.margeNavmeshDe(cle),
		Tolerance:  e.toleranceNavmeshDe(cle),
		NiveauHaut: e.maillageNiveauHautDe(cle),
	}
}

// maillageNatif charge le maillage d'une carte native. Rend nil, sans erreur, quand la carte
// n'en a pas ou y renonce (`sansNavmesh`) ; la dégradation à publier est alors rendue à part.
// Une lecture qui échoue est une erreur : la carte serait cuite sans un maillage qu'elle a.
func (e *environnement) maillageNatif(ctx context.Context, cle string) (m *hinavmesh.Maillage, degradation string, err error) {
	if e.sansNavmeshDe(cle) {
		return nil, "", nil
	}
	m, bilan, err := himap.ChargeMaillageNatif(ctx, e.racineJeu, cle)
	if errors.Is(err, himap.ErrSansMaillage) {
		slog.InfoContext(ctx, "mapfond: carte sans maillage de navigation, cuite comme avant", "carte", cle)
		return nil, degradationSansMaillage, nil
	}
	if err != nil {
		return nil, "", err
	}
	slog.InfoContext(ctx, "mapfond: maillage de navigation charge", "carte", cle,
		"ressource", bilan.Ressource, "octets", bilan.Octets,
		"secondaires", bilan.Secondaires, "illisibles", bilan.Illisibles)
	return m, "", nil
}

// planDeMaillage tire le plan des murs d'un maillage pour le sidecar. Nil quand le maillage est
// absent ou incohérent : l'erreur est journalisée, le fond reste publié sans plan.
func planDeMaillage(ctx context.Context, cle, source string, m *hinavmesh.Maillage) *replaydoc.MapBackgroundPlan {
	if m == nil {
		return nil
	}
	plan, err := himap.PlanDesMurs(m)
	if err != nil {
		slog.ErrorContext(ctx, "mapfond: plan des murs impossible, fond publie sans plan", "err", err, "carte", cle)
		return nil
	}
	aretes := make([][4]float64, len(plan.Aretes))
	for i, a := range plan.Aretes {
		for k, v := range a {
			r := math.Round(v*millimetre) / millimetre
			if r == 0 {
				r = 0 // ecarte le zero negatif, que JSON ecrirait « -0 »
			}
			aretes[i][k] = r
		}
	}
	return &replaydoc.MapBackgroundPlan{Source: source, Vertices: plan.Sommets, Faces: plan.Faces, Edges: aretes}
}
