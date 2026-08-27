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
	"encoding/json"
	"errors"
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

	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/analysis/replay/mapvar"
	"levelup/go-api/internal/himap"
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
	// NonCuisinable : la carte ne PEUT PAS être cuite, et on le sait — `live_fire`
	// (sgh_interlock) ne porte aucun tag sbsp, exception instruite au §1 ter du handoff. Ce
	// n'est pas un échec : le compter comme tel rendrait le code de sortie rouge à chaque
	// exécution, donc illisible.
	NonCuisinable bool
}

func (e *environnement) cuitNatives(ctx context.Context) []bilanAsset {
	var out []bilanAsset
	for _, c := range e.ciblesNatives() {
		rendu, bilan, err := himap.CuitCarteNative(ctx, himap.OptionsCuisson{
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
			PlancherTranche:        e.plancherTrancheDe(c.cle),
			PlafondTranche:         e.plafondTrancheDe(c.cle),
			SeuilArete:             e.seuilAreteDe(c.cle),
			ZonesNommees:           e.zonesNommeesDe(c.cle),
			RogneAuxZones:          e.rogneAuxZonesDe(c.cle),
			MargeZones:             e.margeZonesDe(c.cle),
			BoiteUtile:             e.boiteUtileDe(c.cle),
		})
		if err != nil {
			if errors.Is(err, himap.ErrAucunTagSbsp) {
				slog.WarnContext(ctx, "carte non cuisinable", "err", err, "carte", c.cle)
				out = append(out, bilanAsset{Cle: c.cle, Noms: c.noms, Err: err, NonCuisinable: true})
				continue
			}
			slog.ErrorContext(ctx, "cuisson", "err", err, "carte", c.cle)
			out = append(out, bilanAsset{Cle: c.cle, Noms: c.noms, Err: err})
			continue
		}
		out = append(out, e.publie(ctx, c, rendu, bilan))
	}
	return out
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

func (e *environnement) cuitForge(ctx context.Context) []bilanAsset {
	var out []bilanAsset
	for _, carte := range himap.CartesForge {
		// La cle de selection ET de publication d'une carte Forge est son map_id : un
		// canevas porte des dizaines de cartes, il ne peut keyer ni l'une ni l'autre.
		if !e.retenue(carte.MapID) {
			continue
		}
		c, objets, err := e.cibleForge(carte)
		if err != nil {
			slog.ErrorContext(ctx, "carte Forge", "err", err, "carte", carte.Nom, "map_id", carte.MapID)
			out = append(out, bilanAsset{Cle: carte.MapID, Noms: []string{carte.Nom}, Err: err})
			continue
		}
		rendu, bilan, err := himap.CuitCarteForge(ctx, himap.OptionsCuissonForge{
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
			EcreteToits:            e.ecreteToitsDe(carte.MapID),
			PlafondArene:           e.plafondAreneDe(carte.MapID),
			SubstitutionSansPortee: e.substitutionSansPorteeDe(carte.MapID),
			BoiteUtile:             e.boiteUtileDe(carte.MapID),
			RogneAuxVolumesDeMort:  e.rogneAuxVolumesDeMortDe(carte.MapID),
			TypesExclus:            e.typesExclusDe(carte.MapID),
			MinceurMin:             e.minceurMinDe(carte.MapID),
			PlancherTranche:        e.plancherTrancheDe(carte.MapID),
			PlafondTranche:         e.plafondTrancheDe(carte.MapID),
			DessineCanevas:         e.dessineCanevasDe(carte.MapID),
			SeuilArete:             e.seuilAreteDe(carte.MapID),
		})
		if err != nil {
			slog.ErrorContext(ctx, "cuisson Forge", "err", err, "carte", carte.Nom, "map_id", carte.MapID)
			out = append(out, bilanAsset{Cle: carte.MapID, Noms: c.noms, Err: err})
			continue
		}
		out = append(out, e.publie(ctx, c, rendu, bilan))
	}
	return out
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
	ba.LargeurPx, ba.HauteurPx = img.Bounds().Dx(), img.Bounds().Dy()
	cheminPNG := filepath.Join(e.sortieDir, c.cle+".png")
	octets, err := ecritPNG(cheminPNG, img)
	if err != nil {
		ba.Err = err
		slog.ErrorContext(ctx, "écriture PNG", "err", err, "path", cheminPNG)
		return ba
	}
	ba.OctetsPNG = octets
	if err := e.ecritSidecar(c, rendu, b, rogne); err != nil {
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

// ecritSidecar écrit le calage et les stats à côté du PNG.
func (e *environnement) ecritSidecar(c cible, rendu *himap.Rendu, b himap.BilanCuisson, rogne image.Rectangle) error {
	x0, y1, mpp := himap.CalageDuRendu(rendu)
	// Le PNG publie est ROGNE a la matiere : l'origine suit, l'echelle NON (cadre_utile.go).
	x0, y1 = himap.CalageRogne(x0, y1, mpp, rogne)
	meta := replay.MapBackground{
		SchemaVersion: replay.MapBackgroundSchemaVersion,
		Module:        c.cle,
		MapNames:      append(append([]string{}, c.noms...), c.sources...),
		Image:         c.cle + ".png",
		Source:        sourceCuisson,
		GeneratedAt:   time.Now().UTC(),
		Style:         string(e.styleDe(c.cle)),
		Calibration: replay.MapBackgroundCalibration{
			MetersPerPixel: mpp, OriginX: x0, OriginY: y1,
			WidthPx: rogne.Dx(), HeightPx: rogne.Dy(), Convention: himap.ConventionCalage,
		},
		Stats:        statsDeBilan(b),
		Degradations: b.Degradations,
	}
	blob, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(e.sortieDir, c.cle+".json"), append(blob, '\n'), 0o644) //nolint:gosec // asset versionné
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
