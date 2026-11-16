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
	reference := valeurDe(e, carte.MapID, levierNavmeshReference) || e.decisionFond(carte.MapID).Rogne
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

// optionsForge assemble les options de cuisson d'une carte Forge. Ce sont les mêmes leviers que
// ceux de la chaîne native : un réglage déclaré pour une carte Forge est lu ici comme là-bas.
func (e *environnement) optionsForge(carte himap.CarteForge, c cible, objets []mapvar.Object, navRef *hinavmesh.Maillage) himap.OptionsCuissonForge {
	cle := carte.MapID
	rogneAlt, seuilAlt, margeAlt := e.rogneAuxAltitudesProchesDe(cle)
	posJouees, rayonPos := e.positionsJoueesDe(cle)
	return himap.OptionsCuissonForge{
		Navmesh:                    e.optionsNavmeshForge(cle, navRef),
		RacineDeploy:               e.racineJeu,
		Objets:                     objets,
		Ancres:                     c.ancres,
		CheminModuleCanevas:        himap.CheminCanevasForge(carte),
		Cle:                        cle,
		Echelle:                    e.echelleDe(cle),
		CibleCadrePx:               himap.CibleCadrePx,
		EcreteToits:                valeurDe(e, cle, levierEcreteToits),
		PlafondArene:               valeurDe(e, cle, levierPlafondArene),
		SubstitutionSansPortee:     valeurDe(e, cle, levierSubstitutionSansPortee),
		BoiteUtile:                 e.boiteUtileDe(cle),
		RogneAuxVolumesDeMort:      valeurDe(e, cle, levierRogneAuxVolumesDeMort),
		PlancherTranche:            valeurDe(e, cle, levierPlancherTranche),
		RogneAuxZones:              valeurDe(e, cle, levierRogneAuxZones),
		MargeZones:                 valeurDe(e, cle, levierMargeZones),
		CombleTrous:                valeurDe(e, cle, levierCombleTrous),
		CombleAuMaillage:           valeurDe(e, cle, levierCombleAuMaillage),
		RogneAuxComposantesAncrees: valeurDe(e, cle, levierRogneAuxComposantesAncrees),
		CadreAuxAncres:             valeurDe(e, cle, levierCadreAuxAncres),
		MargeAncres:                valeurDe(e, cle, levierMargeAncres),
		SeuilCouverture:            valeurDe(e, cle, levierSeuilCouverture),
		PositionsJouees:            posJouees,
		RayonPositions:             rayonPos,
		SeuilRecollement:           valeurDe(e, cle, levierSeuilRecollement),
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
