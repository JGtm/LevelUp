package main

// La sélection des cartes, la cuisson et l'écriture des assets.
//
// SÉLECTION — pourquoi le regroupement. Le catalogue d'objectifs porte 37 entrées pour 33
// modules, et plusieurs entrées désignent le MÊME dossier de l'installation : `aquarius_map` et
// `aquarius_-_ranked_map` sont la même carte, jouée en deux playlists. Publier un asset par
// entrée écraserait deux fois le même fichier avec deux jeux d'ancres différents. On regroupe
// donc par dossier installé, et on prend l'UNION des ancres : plus d'ancres, c'est un cadre
// plus juste et un niveau de jeu mieux estimé.

import (
	"context"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"levelup/go-api/internal/mapfondsidecar"

	"github.com/JGtm/halo-infinite-film-decoder/replay"

	"github.com/JGtm/halo-infinite-map-images/himap"

	"github.com/JGtm/halo-infinite-files-reader/hinavmesh"
)

// sourceCuisson documente la chaîne dans chaque sidecar.
const sourceCuisson = "cmd/mapfond-build (internal/himap, triangles du maillage de rendu)"

// cible est une carte à cuire : un dossier de module installé et les entrées de catalogue qui
// s'y rattachent.
type cible struct {
	cle         string
	chemin      string
	sources     []string
	noms        []string
	ancres      [][3]float64
	ancreConnue map[[3]float64]bool
	// maillage et sourcePlan : le maillage de navigation dont le plan des murs est tiré à la
	// publication (nil sans maillage), et son origine pour `plan.source`.
	maillage   *hinavmesh.Maillage
	sourcePlan string
}

// ajouteNom ajoute un nom affiche s'il est non vide et pas deja present.
func (c *cible) ajouteNom(nom string) {
	nom = strings.TrimSpace(nom)
	if nom == "" {
		return
	}
	for _, n := range c.noms {
		if strings.EqualFold(n, nom) {
			return
		}
	}
	c.noms = append(c.noms, nom)
}

// bilanAsset est le résultat publié pour une carte.
type bilanAsset struct {
	Cle       string
	Noms      []string
	Bilan     himap.BilanCuisson
	LargeurPx int
	HauteurPx int
	OctetsPNG int64
	Err       error
	// NonCuisinable : le module ne porte aucun tag sbsp, donc aucune géométrie à projeter. Ce
	// n'est pas un échec : le compter comme tel rendrait le code de sortie rouge à chaque
	// exécution, donc illisible.
	//
	// `live_fire` (sgh_interlock) en était l'exemple de référence — il ne l'est PLUS depuis le
	// 2026-08-27 : sa géométrie vit dans `common-rtx-new.module` et son réglage
	// `moduleGeometrie` l'y envoie. Un module sans sbsp qui atterrit ici est donc désormais une
	// carte dont on n'a pas encore trouvé où la géométrie est rangée, pas une fatalité.
	NonCuisinable bool
	// Suivi : ce que la cuisson a décidé et mesuré pour le rapport de recuisson.
	Suivi suiviFond
}

// ciblesNatives regroupe les entrées du catalogue par dossier de module installé.
func (e *environnement) ciblesNatives() []cible {
	ids := make([]string, 0, len(e.catalogue.Maps))
	for id := range e.catalogue.Maps {
		ids = append(ids, id)
	}
	sort.Strings(ids) // l'ordre d'une map Go n'est pas un ordre : sans ça le rapport bouge
	par := map[string]*cible{}
	var ordre []string
	for _, id := range ids {
		entree := e.catalogue.Maps[id]
		chemin, ok := e.resoutModule(entree.Module)
		if !ok {
			// Signalé, jamais supposé absent : plusieurs cartes du catalogue ne sont pas
			// installées localement, et les confondre avec une chaîne cassée ferait passer
			// une absence pour un échec.
			e.noteNonInstalle(entree.Module)
			continue
		}
		cle := filepath.Base(filepath.Dir(chemin))
		if !e.retenue(cle) {
			continue
		}
		// Une carte FORGE se resout bien vers un dossier — celui de son CANEVAS — mais ce
		// dossier ne porte aucune geometrie : elle appartient à la passe Forge, pas à celle-ci.
		if himap.EstCanevasForge(cle) {
			continue
		}
		c, deja := par[cle]
		if !deja {
			c = &cible{cle: cle, chemin: chemin, ancreConnue: map[[3]float64]bool{}}
			par[cle], ordre = c, append(ordre, cle)
		}
		c.sources = append(c.sources, entree.Module)
		c.ajouteNom(entree.PublicName)
		// LE NOM AFFICHE, sans lequel l'asset n'est pas identifiable. `public_name` est VIDE
		// pour 35 des 37 entrees du catalogue d'objectifs — l'utilisateur a signale au gate
		// visuel qu'il ne reconnaissait pas `sgh_blueprint` ni `fo08_wetland`. Deux sources
		// DECLAREES le donnent, et on ne l'invente pas :
		//   1. le nom de module du catalogue prefixe souvent le module installe
		//      (`recharge_sgh_blueprint` -> Recharge, `prism_sgh_crystalcaves` -> Prism) ;
		//   2. `map_quant_bounds.json` porte le lien nom affiche -> module (aquarius, bazaar...).
		if prefixe := strings.TrimSuffix(entree.Module, "_"+cle); prefixe != entree.Module {
			c.ajouteNom(prefixe)
		}
		for _, n := range e.nomsAffiches[cle] {
			c.ajouteNom(n)
		}
		// Ancres DÉDUPLIQUÉES par position : deux entrées de la même carte (classée et non
		// classée) partagent la plupart de leurs objectifs. Les compter deux fois pèserait sur
		// la médiane qui donne le niveau de jeu — un doublon n'est pas une mesure de plus.
		for _, o := range entree.Objectives {
			a := [3]float64{o.Pos.X, o.Pos.Y, o.Pos.Z}
			if !c.ancreConnue[a] {
				c.ancreConnue[a] = true
				c.ancres = append(c.ancres, a)
			}
		}
	}
	out := make([]cible, 0, len(ordre))
	for _, cle := range ordre {
		out = append(out, *par[cle])
	}
	return out
}

// publie écrit le PNG et son sidecar de calage.
func (e *environnement) publie(ctx context.Context, c cible, rendu *himap.Rendu, b himap.BilanCuisson) bilanAsset {
	ba := bilanAsset{Cle: c.cle, Noms: c.noms, Bilan: b}
	img := himap.FondPNG(rendu, rendu.NX, rendu.NY, nil, e.styleDe(c.cle))
	// LE CADRE PUBLIÉ EST CELUI DE LA MATIÈRE, pas celui de la grille : la coquille de mort a
	// effacé hors frontière et personne ne recalculait le cadre après elle (cadre_utile.go).
	rogne, ok := himap.CadreUtile(img, rendu.Cell)
	if !ok {
		slog.WarnContext(ctx, "fond entierement vide — publie sans rognage", "carte", c.cle)
	} else if rogne != img.Bounds() {
		slog.InfoContext(ctx, "mapfond: cadre rogne sur la matiere", "carte", c.cle,
			"avant", fmt.Sprintf("%dx%d", img.Bounds().Dx(), img.Bounds().Dy()),
			"apres", fmt.Sprintf("%dx%d", rogne.Dx(), rogne.Dy()))
	}
	img = extraitSousImage(img, rogne)
	cal := calageDuFond(rendu, rogne)
	ba.LargeurPx, ba.HauteurPx = img.Bounds().Dx(), img.Bounds().Dy()
	cheminPNG := filepath.Join(e.sortieDir, c.cle+".png")
	octets, err := ecritPNG(cheminPNG, img)
	if err != nil {
		ba.Err = err
		slog.ErrorContext(ctx, "écriture PNG", "err", err, "path", cheminPNG)
		return ba
	}
	ba.OctetsPNG = octets
	pf := e.planDeCible(ctx, c, b, img, cal)
	ba.Suivi = suiviFond{Decision: e.decisionFond(c.cle), Action: actionRecuite, Plan: pf, PartRognee: partRognee(b, img)}
	if err := e.ecritSidecar(ctx, c, b, cal, pf); err != nil {
		ba.Err = err
		slog.ErrorContext(ctx, "écriture sidecar", "err", err, "carte", c.cle)
		return ba
	}
	slog.InfoContext(ctx, "fond de carte figé", "carte", c.cle, "noms", strings.Join(c.noms, ", "),
		"px", fmt.Sprintf("%dx%d", ba.LargeurPx, ba.HauteurPx), "grille", fmt.Sprintf("%dx%d", rendu.NX, rendu.NY),
		"octets", octets,
		"ancres", fmt.Sprintf("%d/%d", b.AncresAvecSol, b.AncresDansLeCadre),
		"path", cheminPNG)
	return ba
}

// ecritPNG encode l'image et rend le poids du fichier écrit.
func ecritPNG(chemin string, img image.Image) (int64, error) {
	f, err := os.Create(chemin) //nolint:gosec // asset versionné, chemin résolu par PathResolver
	if err != nil {
		return 0, err
	}
	if err := png.Encode(f, img); err != nil {
		_ = f.Close()
		return 0, err
	}
	if err := f.Close(); err != nil {
		return 0, err
	}
	st, err := os.Stat(chemin)
	if err != nil {
		return 0, err
	}
	return st.Size(), nil
}

// ecritSidecar écrit le calage et les stats à côté du PNG, plan des murs et image plan compris.
// `cal` est le calage du PNG publié, rogné à la matière : l'origine suit le rognage, l'échelle non
// (cadre_utile.go).
func (e *environnement) ecritSidecar(ctx context.Context, c cible, b himap.BilanCuisson, cal himap.CalageFond, pf *planFond) error {
	meta := mapfondsidecar.MapBackground{MapBackground: replay.MapBackground{
		SchemaVersion: replay.MapBackgroundSchemaVersion,
		Module:        c.cle,
		MapNames:      append(append([]string{}, c.noms...), c.sources...),
		Image:         c.cle + ".png",
		Source:        sourceCuisson,
		GeneratedAt:   time.Now().UTC(),
		Style:         string(e.styleDe(c.cle)),
		Calibration: replay.MapBackgroundCalibration{
			MetersPerPixel: cal.MetresParPixel, OriginX: cal.OrigineX, OriginY: cal.OrigineY,
			WidthPx: cal.Largeur, HeightPx: cal.Hauteur, Convention: himap.ConventionCalage,
		},
		Stats:        statsDeBilan(b),
		Degradations: b.Degradations,
	}}
	poseLePlan(ctx, e.sortieDir, c.cle, pf, &meta)
	return mapfondsidecar.Write(filepath.Join(e.sortieDir, c.cle+".json"), meta)
}

func statsDeBilan(b himap.BilanCuisson) replay.MapBackgroundStats {
	s := replay.MapBackgroundStats{
		Anchors: b.Ancres, AnchorsInFrame: b.AncresDansLeCadre, AnchorsWithGround: b.AncresAvecSol,
		InstancesDrawn: b.Dessinees, InstancesScenery: b.EcarteesDecor,
		PlayLevelZ:      b.NiveauDeJeu,
		BoundaryApplied: b.FrontiereAppliquee, BoundaryPlanes: b.PlansFrontiere,
		BoundaryCellsCleared: b.CellulesEffacees,
		WaterVolumes:         b.VolumesEau, WaterCells: b.CellulesEau,
		CoveredShare: b.TauxCouverture, Covered: b.CarteCouverte,
		CellsSubstituted:  b.CellulesSubstituees,
		CellsClipped:      b.CellulesEcretees,
		CellsAssumedFloor: b.CellulesSolSuppose,
		ForgeObjects:      b.ObjetsForge, ForgeObjectsDrawn: b.ObjetsDessines,
		ForgeObjectsWithoutModel: b.ObjetsSansModele, ForgeDeathVolumes: b.VolumesDeMort,
	}
	// L'écart médian est ABSENT quand aucune ancre n'a trouvé de sol : publier un zéro le
	// ferait passer pour une mesure parfaite.
	if b.AncresAvecSol > 0 {
		ecart := b.EcartMedianAncre
		s.AnchorMedianGapM = &ecart
	}
	return s
}

// ecritRapport écrit le tableau de cuisson, tel qu'il se colle dans un document de chantier.
func ecritRapport(chemin string, bilans []bilanAsset, env *environnement) error {
	var sb strings.Builder
	fmt.Fprintf(&sb, "# Cuisson des fonds de carte — %s\n\n", time.Now().UTC().Format("2006-01-02"))
	// L'habillage et l'échelle annoncés sont ceux de la CUISSON ; les cartes qui portent leur
	// propre réglage sont comptées à côté — sans ça le tableau annoncerait `jeu` pour une carte
	// publiée en `encre`.
	fmt.Fprintf(&sb, "Habillage `%s` · %.4f m/px · %d réglage(s) par carte · sortie `%s`\n\n",
		env.style, env.echelleEffective(), len(env.reglages.Cartes), env.sortieDir)
	fmt.Fprintln(&sb, "`ancres avec sol` est l'ORACLE FAIBLE : une ancre d'objectif sans sol dessine")
	fmt.Fprintln(&sb, "sous elle est un trou de reconstruction. `matiere` compte les instances de bsp")
	fmt.Fprintln(&sb, "dessinees pour une carte native, et les OBJETS poses pour une carte Forge — une")
	fmt.Fprintln(&sb, "carte Forge n'a pas d'instance de bsp dessinee, sa carte EST son rack d'objets.")
	fmt.Fprintln(&sb, "")
	fmt.Fprintln(&sb, "`toits` est la part de matiere qui cache un sol praticable (voie de reference,")
	fmt.Fprintln(&sb, "himap/rendu_reference.go) : au-dela d'un tiers la carte est COUVERTE et montre")
	fmt.Fprintln(&sb, "l'etage de jeu au lieu du plafond, dans la portee des ancres.")
	fmt.Fprintln(&sb, "")
	fmt.Fprintln(&sb, "| carte | noms | px | Ko | ancres avec sol | matiere | frontiere | eau | toits | degradations |")
	fmt.Fprintln(&sb, "|---|---|---|---:|---|---|---|---:|---|---|")
	for _, b := range bilans {
		if b.Suivi.Action == actionSautee {
			fmt.Fprintf(&sb, "| `%s` | %s | SAUTEE (%s) | | | | | | | |\n",
				b.Cle, strings.Join(b.Noms, ", "), b.Suivi.Decision.Etat)
			continue
		}
		if b.Err != nil {
			etat := "ECHEC"
			if b.NonCuisinable {
				etat = "NON CUISINABLE"
			}
			fmt.Fprintf(&sb, "| `%s` | %s | %s | | | | | | | %v |\n",
				b.Cle, strings.Join(b.Noms, ", "), etat, b.Err)
			continue
		}
		frontiere := "non"
		if b.Bilan.FrontiereAppliquee {
			frontiere = fmt.Sprintf("%d plans", b.Bilan.PlansFrontiere)
		}
		matiere := fmt.Sprintf("%d instances", b.Bilan.Dessinees)
		if b.Bilan.ObjetsForge > 0 {
			matiere = fmt.Sprintf("%d/%d objets Forge", b.Bilan.ObjetsDessines, b.Bilan.ObjetsForge)
		}
		toits := fmt.Sprintf("%.0f %%", 100*b.Bilan.TauxCouverture)
		if b.Bilan.CarteCouverte {
			toits = fmt.Sprintf("COUVERTE %.0f %% (%d px)", 100*b.Bilan.TauxCouverture, b.Bilan.CellulesSubstituees)
		}
		fmt.Fprintf(&sb, "| `%s` | %s | %dx%d | %d | %d/%d | %s | %s | %d | %s | %s |\n",
			b.Cle, strings.Join(b.Noms, ", "), b.LargeurPx, b.HauteurPx, b.OctetsPNG/1024,
			b.Bilan.AncresAvecSol, b.Bilan.AncresDansLeCadre, matiere,
			frontiere, b.Bilan.CellulesEau, toits, strings.Join(b.Bilan.Degradations, " · "))
	}
	if len(env.nonInstalle) > 0 {
		fmt.Fprintf(&sb, "\n%d modules du catalogue non installes localement : %s\n",
			len(env.nonInstalle), strings.Join(env.nonInstalle, ", "))
	}
	return os.WriteFile(chemin, []byte(sb.String()), 0o644) //nolint:gosec // rapport de chantier
}

// extraitSousImage rend une image RGBA autonome bornee a `r`, origine ramenee a (0,0).
//
// `SubImage` partagerait le tableau de pixels et garderait des bornes decalees : l'encodeur PNG
// les respecte, mais tout ce qui lit `Bounds().Min` ensuite doit y penser. Une copie coute une
// fois, a la cuisson, et supprime le piege.
func extraitSousImage(src *image.RGBA, r image.Rectangle) *image.RGBA {
	if r == src.Bounds() {
		return src
	}
	dst := image.NewRGBA(image.Rect(0, 0, r.Dx(), r.Dy()))
	draw.Draw(dst, dst.Bounds(), src, r.Min, draw.Src)
	return dst
}

// moduleDeCuisson rend le module d'ou tirer la geometrie de cette carte : le sien, sauf
// reglage explicite (cf. reglageCarte.ModuleGeometrie — le cas de Live Fire).
func (e *environnement) moduleDeCuisson(c cible) string {
	if p := e.moduleGeometrieDe(c.cle); p != "" {
		return p
	}
	return c.chemin
}
