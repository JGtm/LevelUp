package main

// La cuisson d'une carte Forge : maillage de navigation, options, publication.

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/JGtm/halo-infinite-map-images/himap"

	"github.com/JGtm/halo-infinite-files-reader/hinavmesh"
	"github.com/JGtm/halo-infinite-files-reader/mapvar"
)

func (e *environnement) cuitForge(ctx context.Context) []bilanAsset {
	var out []bilanAsset
	for _, carte := range himap.CartesForge {
		// La cle de selection ET de publication d'une carte Forge est son map_id : un
		// canevas porte des dizaines de cartes, il ne peut keyer ni l'une ni l'autre.
		if !e.retenue(carte.MapID) {
			continue
		}
		if d := e.decisionFond(carte.MapID); d.Saute {
			slog.InfoContext(ctx, "mapfond: carte laissee telle que publiee par l'avis", "carte", carte.Nom, "map_id", carte.MapID, "etat", d.Etat)
			out = append(out, bilanSaute(carte.MapID, []string{carte.Nom}, d))
			continue
		}
		out = append(out, e.cuitUneForge(ctx, carte))
	}
	return out
}

// cuitUneForge cuit une carte Forge : sa variante, son maillage (surface de reference et plan
// des murs), puis le rendu.
func (e *environnement) cuitUneForge(ctx context.Context, carte himap.CarteForge) bilanAsset {
	c, objets, err := e.cibleForge(carte)
	if err != nil {
		slog.ErrorContext(ctx, "carte Forge", "err", err, "carte", carte.Nom, "map_id", carte.MapID)
		return bilanAsset{Cle: carte.MapID, Noms: []string{carte.Nom}, Err: err}
	}
	if e.sourceNavmeshDe(carte.MapID) {
		rendu, bilan, err := e.cuitDepuisNavmesh(ctx, carte, &c)
		if err != nil {
			slog.ErrorContext(ctx, "cuisson depuis le navmesh", "err", err, "carte", carte.Nom)
			return bilanAsset{Cle: carte.MapID, Noms: c.noms, Err: err}
		}
		return e.publie(ctx, c, rendu, bilan)
	}
	navRef, tout := e.maillageForge(ctx, carte)
	c.maillage, c.sourcePlan = tout, sourcePlanForge
	rendu, bilan, err := himap.CuitCarteForge(ctx, e.optionsForge(carte, c, objets, navRef))
	if err != nil {
		slog.ErrorContext(ctx, "cuisson Forge", "err", err, "carte", carte.Nom, "map_id", carte.MapID)
		return bilanAsset{Cle: carte.MapID, Noms: c.noms, Err: err}
	}
	return e.publie(ctx, c, rendu, bilan)
}

// maillageForge charge le maillage de navigation d'une carte Forge qui en a un dans le dépôt et
// rend, d'une part la surface de référence et de rognage (nil si la carte ne la prend pas sur lui
// et qu'aucun avis ne la rogne), d'autre part le maillage entier dont le plan est tiré (nil sans
// maillage). Le plan se publie pour toute carte qui a un blob, que la cuisson s'en serve ou non.
func (e *environnement) maillageForge(ctx context.Context, carte himap.CarteForge) (arme, tout *hinavmesh.Maillage) {
	reference := e.navmeshReferenceDe(carte.MapID) || e.decisionFond(carte.MapID).Rogne
	m, err := e.chargeNavmesh(carte)
	switch {
	case errors.Is(err, os.ErrNotExist) && !reference:
		slog.DebugContext(ctx, "mapfond: pas de maillage de navigation au depot", "carte", carte.Nom)
		return nil, nil
	case err != nil && reference:
		slog.WarnContext(ctx, "maillage de navigation indisponible, reference laissee aux ancres", "err", err, "carte", carte.Nom)
		return nil, nil
	case err != nil:
		slog.WarnContext(ctx, "maillage de navigation illisible, fond publie sans plan", "err", err, "carte", carte.Nom)
		return nil, nil
	}
	if !reference {
		return nil, m
	}
	return m, m
}

// optionsForge assemble les options de cuisson d'une carte Forge.
func (e *environnement) optionsForge(carte himap.CarteForge, c cible, objets []mapvar.Object, navRef *hinavmesh.Maillage) himap.OptionsCuissonForge {
	rogneAlt, seuilAlt, margeAlt := e.rogneAuxAltitudesProchesDe(carte.MapID)
	posJouees, rayonPos := e.positionsJoueesDe(carte.MapID)
	return himap.OptionsCuissonForge{
		Navmesh:             e.optionsNavmeshForge(carte.MapID, navRef),
		RacineDeploy:        e.racineJeu,
		Objets:              objets,
		Ancres:              c.ancres,
		CheminModuleCanevas: himap.CheminCanevasForge(carte),
		Cle:                 carte.MapID,
		Echelle:             e.echelleDe(carte.MapID),
		CibleCadrePx:        himap.CibleCadrePx,
		// Les memes leviers que la chaine native : jusqu au 2026-08-27 une carte Forge
		// n en recevait aucun, et un reglage declare pour elle etait silencieusement
		// ignore — trois cuissons d Isolation a trois plafonds ont rendu le MEME octet.
		EcreteToits:                e.ecreteToitsDe(carte.MapID),
		PlafondArene:               e.plafondAreneDe(carte.MapID),
		SubstitutionSansPortee:     e.substitutionSansPorteeDe(carte.MapID),
		BoiteUtile:                 e.boiteUtileDe(carte.MapID),
		RogneAuxVolumesDeMort:      e.rogneAuxVolumesDeMortDe(carte.MapID),
		TypesExclus:                e.typesExclusDe(carte.MapID),
		MinceurMin:                 e.minceurMinDe(carte.MapID),
		SolVuDuDessous:             e.solVuDuDessousDe(carte.MapID),
		PlafondObjets:              e.plafondObjetsDe(carte.MapID),
		DrapeauxExclus:             e.drapeauxExclusDe(carte.MapID),
		PlancherTranche:            e.plancherTrancheDe(carte.MapID),
		PlafondTranche:             e.plafondTrancheDe(carte.MapID),
		DessineCanevas:             e.dessineCanevasDe(carte.MapID),
		SeuilArete:                 e.seuilAreteDe(carte.MapID),
		RogneAuxZones:              e.rogneAuxZonesDe(carte.MapID),
		MargeZones:                 e.margeZonesDe(carte.MapID),
		CombleTrous:                e.combleTrousDe(carte.MapID),
		CombleAuMaillage:           e.combleAuMaillageDe(carte.MapID),
		RogneAuxComposantesAncrees: e.rogneAuxComposantesAncreesDe(carte.MapID),
		CadreAuxZones:              e.cadreAuxZonesDe(carte.MapID),
		CadreAuxAncres:             e.cadreAuxAncresDe(carte.MapID),
		MargeAncres:                e.margeAncresDe(carte.MapID),
		SansSubstitution:           e.sansSubstitutionDe(carte.MapID),
		SeuilSubstitution:          e.seuilSubstitutionDe(carte.MapID),
		SeuilCouverture:            e.seuilCouvertureDe(carte.MapID),
		MargeSolBas:                e.margeSolBasDe(carte.MapID),
		PositionsJouees:            posJouees,
		RayonPositions:             rayonPos,
		SeuilRecollement:           e.seuilRecollementDe(carte.MapID),
		RogneAuxAltitudesProches:   rogneAlt,
		SeuilAltitude:              seuilAlt,
		MargeAltitude:              margeAlt,
	}
}

// cibleForge lit le .mvar d'une carte Forge et VÉRIFIE sa jointure au catalogue par le compte
// d'objets — sans quoi on cuirait la carte d'une autre.
func (e *environnement) cibleForge(carte himap.CarteForge) (cible, []mapvar.Object, error) {
	entree, err := e.catalogue.Lookup(carte.MapID)
	if err != nil {
		return cible{}, nil, err
	}
	chemin := filepath.Join(e.repoRoot, himap.DepotVariantesCarte, carte.FichierMvar)
	brut, err := os.ReadFile(chemin) //nolint:gosec // entrée d'outillage hors ligne
	if err != nil {
		return cible{}, nil, fmt.Errorf("variante de carte illisible (%s) : %w", chemin, err)
	}
	v, err := mapvar.Parse(brut)
	if err != nil {
		return cible{}, nil, fmt.Errorf("variante de carte invalide (%s) : %w", chemin, err)
	}
	if entree.ObjectsN != len(v.Objects) {
		return cible{}, nil, fmt.Errorf("jointure catalogue<->mvar cassée : %d objets au catalogue, %d dans %s",
			entree.ObjectsN, len(v.Objects), filepath.Base(chemin))
	}
	c := cible{cle: carte.MapID, sources: []string{entree.Module}}
	c.ajouteNom(entree.PublicName)
	// Le nom vient de la DECLARATION, jamais du catalogue de bornes : ce catalogue est keye
	// par nom -> module de CANEVAS, et un canevas porte les noms de toutes les cartes qui y
	// sont baties — le lire ici attacherait a Vagabond les noms de Dynasty et d'Origin.
	c.ajouteNom(carte.Nom)
	for _, o := range entree.Objectives {
		c.ancres = append(c.ancres, [3]float64{o.Pos.X, o.Pos.Y, o.Pos.Z})
	}
	return c, v.Objects, nil
}

// cuitDepuisNavmesh rend le fond d'une carte Forge a partir de son MAILLAGE DE NAVIGATION.
//
// C'est la seule source qui ne demande rien a soustraire : sur une carte a ciel ferme, le
// maillage vit sous les coques et ne les contient pas. Le blob vit dans le depot hors ligne,
// a cote des variantes.
func (e *environnement) cuitDepuisNavmesh(ctx context.Context, carte himap.CarteForge, c *cible) (*himap.Rendu, himap.BilanCuisson, error) {
	chemin := filepath.Join(e.repoRoot, himap.DepotNavmesh, carte.MapID+".blob")
	blob, err := os.ReadFile(chemin) //nolint:gosec // entrée d'outillage hors ligne
	if err != nil {
		return nil, himap.BilanCuisson{}, fmt.Errorf("maillage de navigation illisible (%s) : %w", chemin, err)
	}
	if m, errDecode := hinavmesh.Decode(blob); errDecode != nil {
		slog.WarnContext(ctx, "mapfond: maillage de navigation illisible, fond publie sans plan", "err", errDecode, "carte", carte.Nom)
	} else {
		c.maillage, c.sourcePlan = m, sourcePlanForge
	}
	return himap.CuitCarteNavmesh(ctx, himap.OptionsCuissonNavmesh{
		Blob:         blob,
		Ancres:       c.ancres,
		Cle:          carte.MapID,
		Echelle:      e.echelleDe(carte.MapID),
		CibleCadrePx: himap.CibleCadrePx,
	})
}

// blobNavmesh lit le navmesh.blob d'une carte dans le depot hors ligne.
func (e *environnement) blobNavmesh(carte himap.CarteForge) ([]byte, error) {
	chemin := filepath.Join(e.repoRoot, himap.DepotNavmesh, carte.MapID+".blob")
	blob, err := os.ReadFile(chemin) //nolint:gosec // entree d'outillage hors ligne
	if err != nil {
		return nil, fmt.Errorf("maillage de navigation illisible (%s) : %w", chemin, err)
	}
	return blob, nil
}

// chargeNavmesh lit et decode le maillage de navigation d'une carte.
func (e *environnement) chargeNavmesh(carte himap.CarteForge) (*hinavmesh.Maillage, error) {
	blob, err := e.blobNavmesh(carte)
	if err != nil {
		return nil, err
	}
	return hinavmesh.Decode(blob)
}
