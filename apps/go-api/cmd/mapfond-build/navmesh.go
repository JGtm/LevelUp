package main

// Le maillage de navigation dans la cuisson : ce qu'on en charge, ce qu'on en fait, et le plan
// des murs qu'on en publie au sidecar.

import (
	"context"
	"errors"
	"log/slog"

	"github.com/JGtm/halo-infinite-map-images/himap"

	"github.com/JGtm/halo-infinite-files-reader/hinavmesh"
)

const (
	// sourcePlanNatif et sourcePlanForge nomment l'origine du maillage dans `plan.source`.
	sourcePlanNatif = "navmesh module any"
	sourcePlanForge = "navmesh.blob"

	// degradationSansMaillage : texte du bilan d'une carte dont le module `any` ne porte aucun
	// maillage de navigation.
	degradationSansMaillage = "aucun maillage de navigation dans le module any"

	// millimetre : le plan est publié au millimètre (voir aretesAuMillimetre).
	millimetre = 1000.0
)

// sansNavmeshDe dit si cette carte cuit sans s'appuyer sur son maillage de navigation (ni
// référence, ni rognage). Journalisé. Le maillage reste chargé : le plan des murs n'en dépend pas.
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

// optionsNavmeshNatif rend ce qu'une carte native fait de son maillage : rognage à la marge par
// défaut de la brique (0,5 m), sans tolérance ni niveau haut, sauf réglage de la carte
// (`rogneAuNavmesh`, `margeNavmesh`, `toleranceNavmesh`, `maillageNiveauHaut`). `rogneAuNavmesh:
// false` garde le maillage comme référence sans rogner ; `sansNavmesh` rend des options nulles (le
// maillage ne sert plus qu'au plan des murs). Un avis (`-avis`) arme le rognage et fixe sa forme.
func (e *environnement) optionsNavmeshNatif(cle string, m *hinavmesh.Maillage) himap.OptionsNavmesh {
	if e.sansNavmeshDe(cle) {
		return himap.OptionsNavmesh{}
	}
	d := e.decisionFond(cle)
	rogne := d.Rogne || e.rogneAuNavmeshNatifDe(cle)
	return himap.OptionsNavmesh{
		Maillage:   m,
		Rogne:      rogne,
		Mode:       d.Mode,
		Marge:      e.margeNavmeshDe(cle),
		Tolerance:  e.toleranceNavmeshDe(cle),
		NiveauHaut: e.maillageNiveauHautDe(cle),
	}
}

// optionsNavmeshForge rend les réglages navmesh d'une carte Forge : ceux du fichier de réglages,
// que l'avis (`-avis`) complète en armant le rognage et en fixant sa forme. Le maillage est nil
// quand la carte ne prend pas sa référence sur lui et n'est pas rognée par un avis.
func (e *environnement) optionsNavmeshForge(cle string, m *hinavmesh.Maillage) himap.OptionsNavmesh {
	d := e.decisionFond(cle)
	return himap.OptionsNavmesh{
		Maillage:   m,
		Rogne:      d.Rogne || e.rogneAuNavmeshDe(cle),
		Mode:       d.Mode,
		Marge:      e.margeNavmeshDe(cle),
		Tolerance:  e.toleranceNavmeshDe(cle),
		NiveauHaut: e.maillageNiveauHautDe(cle),
	}
}

// maillageNatif charge le maillage d'une carte native, y compris quand elle cuit `sansNavmesh` :
// il sert alors au seul plan des murs. Rend nil, sans erreur, quand la carte n'en a pas ; la
// dégradation à publier est alors rendue à part. Une lecture qui échoue est une erreur : la carte
// serait cuite sans un maillage qu'elle a.
func (e *environnement) maillageNatif(ctx context.Context, cle string) (m *hinavmesh.Maillage, degradation string, err error) {
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
