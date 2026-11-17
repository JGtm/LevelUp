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

// optionsNative assemble les options de cuisson d'une carte native. Les leviers sont les mêmes que
// sur la chaîne Forge (`optionsForge`).
func (e *environnement) optionsNative(c cible, maillage *hinavmesh.Maillage) himap.OptionsCuisson {
	cle := c.cle
	return himap.OptionsCuisson{
		RacineDeploy:               e.racineJeu,
		CheminModule:               e.moduleDeCuisson(c),
		Ancres:                     c.ancres,
		Echelle:                    e.echelleDe(cle),
		CibleCadrePx:               himap.CibleCadrePx,
		EcreteToits:                valeurDe(e, cle, levierEcreteToits),
		PlafondArene:               valeurDe(e, cle, levierPlafondArene),
		SansEau:                    valeurDe(e, cle, levierSansEau),
		SubstitutionSansPortee:     valeurDe(e, cle, levierSubstitutionSansPortee),
		CombleTrous:                valeurDe(e, cle, levierCombleTrous),
		CombleZonesEntieres:        valeurDe(e, cle, levierCombleZonesEntieres),
		PlancherTranche:            valeurDe(e, cle, levierPlancherTranche),
		ZonesNommees:               e.zonesNommeesDe(cle),
		RogneAuxZones:              valeurDe(e, cle, levierRogneAuxZones),
		MargeZones:                 valeurDe(e, cle, levierMargeZones),
		BoiteUtile:                 e.boiteUtileDe(cle),
		RogneAuxComposantesAncrees: valeurDe(e, cle, levierRogneAuxComposantesAncrees),
		Navmesh:                    e.optionsNavmeshNatif(cle, maillage),
	}
}
