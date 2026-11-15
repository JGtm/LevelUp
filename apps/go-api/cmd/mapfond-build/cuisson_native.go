package main

// La cuisson d'une carte native : chargement du maillage de navigation, options, publication.

import (
	"context"
	"errors"
	"log/slog"

	"github.com/JGtm/halo-infinite-map-images/himap"

	"github.com/JGtm/halo-infinite-files-reader/hinavmesh"
)

func (e *environnement) cuitNatives(ctx context.Context) []bilanAsset {
	var out []bilanAsset
	for _, c := range e.ciblesNatives() {
		if d := e.decisionFond(c.cle); d.Saute {
			slog.InfoContext(ctx, "mapfond: carte laissee telle que publiee par l'avis", "carte", c.cle, "etat", d.Etat)
			out = append(out, bilanSaute(c.cle, c.noms, d))
			continue
		}
		out = append(out, e.cuitNative(ctx, c))
	}
	return out
}

// cuitNative cuit une carte native : son maillage de navigation d'abord (rognage et plan des
// murs), puis le rendu. Une carte dont le module n'a pas de maillage est cuite comme avant et
// le porte en dégradation.
func (e *environnement) cuitNative(ctx context.Context, c cible) bilanAsset {
	maillage, degradation, err := e.maillageNatif(ctx, c.cle)
	if err != nil {
		slog.ErrorContext(ctx, "maillage de navigation", "err", err, "carte", c.cle)
		return bilanAsset{Cle: c.cle, Noms: c.noms, Err: err}
	}
	rendu, bilan, err := himap.CuitCarteNative(ctx, e.optionsNative(c, maillage))
	if err != nil {
		if errors.Is(err, himap.ErrAucunTagSbsp) {
			slog.WarnContext(ctx, "carte non cuisinable", "err", err, "carte", c.cle)
			return bilanAsset{Cle: c.cle, Noms: c.noms, Err: err, NonCuisinable: true}
		}
		slog.ErrorContext(ctx, "cuisson", "err", err, "carte", c.cle)
		return bilanAsset{Cle: c.cle, Noms: c.noms, Err: err}
	}
	if degradation != "" {
		bilan.Degradations = append(bilan.Degradations, degradation)
	}
	c.maillage, c.sourcePlan = maillage, sourcePlanNatif
	return e.publie(ctx, c, rendu, bilan)
}

// optionsNative assemble les options de cuisson d'une carte native.
func (e *environnement) optionsNative(c cible, maillage *hinavmesh.Maillage) himap.OptionsCuisson {
	return himap.OptionsCuisson{
		RacineDeploy:           e.racineJeu,
		CheminModule:           e.moduleDeCuisson(c),
		Ancres:                 c.ancres,
		Echelle:                e.echelleDe(c.cle),
		CibleCadrePx:           himap.CibleCadrePx,
		EcreteToits:            e.ecreteToitsDe(c.cle),
		PlafondArene:           e.plafondAreneDe(c.cle),
		SansEau:                e.sansEauDe(c.cle),
		SubstitutionSansPortee: e.substitutionSansPorteeDe(c.cle),
		CombleTrous:            e.combleTrousDe(c.cle),
		CombleZonesEntieres:    e.combleZonesEntieresDe(c.cle),
		PlancherTranche:        e.plancherTrancheDe(c.cle),
		PlafondTranche:         e.plafondTrancheDe(c.cle),
		SeuilArete:             e.seuilAreteDe(c.cle),
		ZonesNommees:           e.zonesNommeesDe(c.cle),
		RogneAuxZones:          e.rogneAuxZonesDe(c.cle),
		MargeZones:             e.margeZonesDe(c.cle),
		BoiteUtile:             e.boiteUtileDe(c.cle),
		// Le MEME reglage que sur le chemin Forge, et c est tout le point : il est declare
		// pour toutes les cartes et n etait cable que la, donc sans effet et en silence sur
		// une carte native.
		RogneAuxComposantesAncrees: e.rogneAuxComposantesAncreesDe(c.cle),
		Navmesh:                    e.optionsNavmeshNatif(c.cle, maillage),
	}
}
