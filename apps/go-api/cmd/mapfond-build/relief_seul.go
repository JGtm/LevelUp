package main

// Le mode -relief-seul : ajoute le relief aux fonds déjà publiés, sans toucher à leur image, à leur plan
// ni à leur calage.
//
// La carte est cuite en mémoire (mêmes réglages que la cuisson complète, sauf l'échelle : celle du fond
// publié), et le relief se projette dans le calage du fond PUBLIÉ, lu dans son sidecar : une recuisson peut sortir un cadre de taille ou de
// position différentes (Recharge : 35 lignes de plus en haut du fond publié), et le relief doit se
// superposer au fond que le site sert.

import (
	"context"
	"fmt"
	"path/filepath"

	"levelup/go-api/internal/mapfondsidecar"

	"github.com/JGtm/halo-infinite-map-images/himap"
)

// ajouteReliefs calcule, pour chaque carte retenue dont le sidecar et le fond sont publiés, le relief
// dans le calage publié, écrit `<cle>.relief.png` et `<cle>.relief.json.gz` et pose `reliefImage` et
// `reliefData` dans le sidecar. Les autres champs du sidecar, l'image du fond, l'image plan et le
// plan des murs restent ceux de la cuisson qui les a produits. Une carte sans sidecar ou sans fond est
// laissée intacte ; une carte sans maillage reçoit un relief dégradé (`sans-maillage`) ; un maillage introuvable pour un fond dont le sidecar
// publié porte un plan (plan_publie_sans_maillage.go), met la carte en échec et n'écrit rien.
func (e *environnement) ajouteReliefs(ctx context.Context) bilanSeul {
	return e.parcourtLesCartes(ctx, e.ajouteReliefForge, e.ajouteReliefNatif)
}

// ajouteReliefForge refait le relief d'une carte Forge.
func (e *environnement) ajouteReliefForge(ctx context.Context, carte himap.CarteForge, b *bilanSeul) {
	noms := []string{carte.Nom}
	err := e.refaitLeRelief(ctx, carte.MapID, noms, b, func(echelle float64) (carteCuite, error) { return e.renduForge(ctx, carte, echelle) })
	e.noteResultat(ctx, b, carte.MapID, noms, err)
}

// ajouteReliefNatif refait le relief d'une carte native.
func (e *environnement) ajouteReliefNatif(ctx context.Context, c cible, b *bilanSeul) {
	err := e.refaitLeRelief(ctx, c.cle, c.noms, b, func(echelle float64) (carteCuite, error) { return e.renduNatif(ctx, c, echelle) })
	e.noteResultat(ctx, b, c.cle, c.noms, err)
}

// refaitLeRelief lit le sidecar et le fond publiés de la carte, la cuit en mémoire avec `cuire` à
// l'échelle du fond publié (`metersPerPixel` de son calage : la cellule du rendu doit l'égaler, quelle
// que soit l'échelle que la cuisson complète choisirait aujourd'hui), calcule son relief dans le calage
// publié et réécrit le sidecar avec ses deux champs de plus. Rend une erreur qui satisfait os.ErrNotExist
// quand le sidecar ou le fond publié n'existe pas ; la cuisson n'est alors pas lancée.
//
// Le cadre de la grille recuite peut différer d'une fraction de pixel du cadre publié : la projection de
// la brique lit le plus proche voisin au centre du pixel publié (erreur d'au plus un demi-pixel, lue au
// bilan du relief `decalage_x` / `decalage_y`) et ne refuse que l'échelle.
func (e *environnement) refaitLeRelief(ctx context.Context, cle string, noms []string, b *bilanSeul,
	cuire func(echelle float64) (carteCuite, error)) error {
	s, fond, cal, err := lisFondEtCalage(e.sortieDir, cle)
	if err != nil {
		return err
	}
	cc, err := cuire(cal.MetresParPixel)
	if err != nil {
		// Aplati : une variante ou un maillage absents de l'outillage ne sont pas un sidecar absent.
		return fmt.Errorf("cuisson en memoire de %s : %s", cle, err)
	}
	if err := verifieLeMaillageDuPlanPublie(cle, s, cc.c.maillage); err != nil {
		return err
	}
	garde := gardeDuMaillage(cc.c.maillage, cc.c.ancres, matiereDeImage(fond, cal))
	rf, err := calculeRelief(ctx, entreesRelief{rendu: cc.rendu, maillage: cc.c.maillage, garde: garde, cal: cal, fond: fond})
	if err != nil {
		return fmt.Errorf("relief de %s : %w", cle, err)
	}
	if err := ecritLeRelief(ctx, e.sortieDir, cle, rf, s); err != nil {
		return err
	}
	if err := mapfondsidecar.Write(filepath.Join(e.sortieDir, cle+".json"), *s); err != nil {
		return err
	}
	b.Bilans = append(b.Bilans, bilanAsset{Cle: cle, Noms: noms,
		Suivi: suiviFond{Decision: e.decisionFond(cle), Action: actionReliefSeul}})
	return nil
}

// executeReliefSeul lance ajouteReliefs, journalise le bilan, écrit les rapports demandés et rend le
// code de sortie.
func executeReliefSeul(ctx context.Context, env *environnement, rapports rapportsDemandes) int {
	return termineUnGesteSeul(ctx, env, rapports, "relief seul termine", env.ajouteReliefs(ctx))
}
