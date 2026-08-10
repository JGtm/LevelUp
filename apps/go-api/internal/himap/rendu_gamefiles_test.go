package himap

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRenduCliffhanger produit le rendu du maillage vu du dessus, au calage de la carte
// validee, et l'ecrit cote a cote avec elle. Il ne juge rien : le juge est l'utilisateur.
func TestRenduCliffhanger(t *testing.T) {
	racine, err := DeployRoot()
	if err != nil {
		t.Skip(err)
	}
	ref, err := cheminDepuisDepot(".ai/V7.5/dumps/carte_validee_v1.png")
	if err != nil {
		t.Skip(err)
	}
	f, err := os.Open(ref)
	if err != nil {
		t.Skip(err)
	}
	defer func() { _ = f.Close() }()
	validee, err := png.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	bo := validee.Bounds()
	larg, haut := bo.Dx(), bo.Dy()

	lo := [2]float64{gateX0, gateY1 - float64(haut)*gateEchelle}
	hi := [2]float64{gateX0 + float64(larg)*gateEchelle, gateY1}
	rendu := NewRendu(lo, hi, gateEchelle)
	// LA TRANCHE DE JEU — configuration validee par l'utilisateur le 2026-08-09.
	//
	// L'ancien defaut (plafond « deduit des ancres » a +6,95 m, aucun plancher) est FAUX des
	// deux cotes : il coupait 118 000 pixels justes entre +10 et +35 m, et laissait entrer
	// 807 000 pixels de decor sous -10 m. RENDU_TRANCHE le rejoue pour la comparaison.
	plancher, plafond := TrancheDeJeuMin, TrancheDeJeuMax
	if v := os.Getenv("RENDU_TRANCHE"); v != "" {
		fmt.Sscanf(v, "%f,%f", &plancher, &plafond)
	}
	if v := os.Getenv("RENDU_PLAFOND"); v == "non" {
		plafond = math.Inf(1)
		t.Log("TEMOIN : AUCUN plafond d'altitude")
	} else if v != "" {
		fmt.Sscanf(v, "%f", &plafond)
	}
	rendu.Tranche(plancher, plafond)
	t.Logf("tranche d'altitude : [%+.2f ; %+.2f] m", plancher, plafond)

	modCarte := moduleDuJeu(t, "pc", "ridgeline")
	chemins, _ := GeometrySearchPath(racine, modCarte)
	idx, err := NewModuleIndex(chemins...)
	if err != nil {
		t.Fatal(err)
	}
	bsps, _ := ReadModuleInstances(modCarte)
	// Le bsp DESIGNE PAR LES ANCRES, comme partout ailleurs dans ce package. Prendre « tous
	// les bsp » ramenait l'horizon de 6 619 x 10 471 m, dont la dalle de fond de scene couvre
	// 100 % du cadre — porte deja fermee au §9.2 du handoff.
	bsp := choisitBSP(bsps, ancresCliffhanger(t))
	// Le niveau JOUE, deduit des ancres : c'est contre lui que les habillages `jeu` et `encre`
	// mesurent l'altitude, au lieu de la mesurer contre elle-meme.
	rendu.NiveauDeJeu(medianeZ(ancresCliffhanger(t)) - AncrageDecalageSol)
	// Temoin A/B : neutraliser le champ `scale` reproduit la lecture d'avant le 2026-08-09.
	// N'existe que pour departager, jamais pour produire une carte.
	sansEchelle := os.Getenv("RENDU_SANS_ECHELLE") != ""
	if sansEchelle {
		t.Log("TEMOIN : echelle d'instance NEUTRALISEE (lecture concurrente)")
	}
	gardeOmbres := os.Getenv("RENDU_GARDE_OMBRES") != ""
	if gardeOmbres {
		t.Log("TEMOIN : projecteurs d'ombre CONSERVES (ce que faisait la chaine avant lecture de Reclaimer)")
	}
	unModule := os.Getenv("RENDU_MODULE")
	// TOUS LES MODULES par defaut — configuration validee le 2026-08-09.
	//
	// Le filtre « module de la carte seul » retirait la MOITIE de la carte : mesure pixel a
	// pixel contre la carte validee, 52,1 % de sa matiere manquante, dont le second pont et la
	// zone haut-gauche que l'utilisateur reclamait. Aucun module n'est « le decor » : le seul
	// module de la carte ne couvre que 50,3 % de la reference, `common` 60,2 %, `multiplayer`
	// 47,5 % — ils se completent.
	carteSeule := os.Getenv("RENDU_CARTE_SEULE") != ""
	if carteSeule {
		t.Log("TEMOIN : module de la carte SEUL (l'ancien defaut, qui perdait la moitie de la carte)")
	}
	// Le bornage coupe la matiere qui sort de la boite declaree de l'instance — EN BAS comme
	// en haut (`triangleBorne` : `z < lo[2] || z > hi[2]`). Sa justification etait un
	// debordement median de 0,231 de la diagonale ; c'est EXACTEMENT la mesure de l'echelle
	// non appliquee. Echelle corrigee, le debordement tombe a 0,0665 — le bornage se met donc
	// a couper de la vraie geometrie au lieu d'ecarter une aberration.
	// Emprise bornee au VOISINAGE DES ANCRES (`PorteeAncre`), la regle deja retenue par
	// `reference.go` : sur Cliffhanger elle ramene 113 x 114 m declares a 51 x 55 m joues.
	// Elle est testee ici comme discriminant du decor lateral, que la tranche d'altitude ne
	// separe pas (l'exces se repartit sur toute la tranche).
	empriseAncres := os.Getenv("RENDU_EMPRISE_ANCRES") != ""
	var aLo, aHi [2]float64
	if empriseAncres {
		aLo = [2]float64{math.Inf(1), math.Inf(1)}
		aHi = [2]float64{math.Inf(-1), math.Inf(-1)}
		for _, a := range ancresCliffhanger(t) {
			for k := 0; k < 2; k++ {
				aLo[k] = math.Min(aLo[k], a[k]-PorteeAncre)
				aHi[k] = math.Max(aHi[k], a[k]+PorteeAncre)
			}
		}
		t.Logf("TEMOIN : emprise bornee aux ancres [%.1f ; %.1f] x [%.1f ; %.1f] = %.0f x %.0f m",
			aLo[0], aHi[0], aLo[1], aHi[1], aHi[0]-aLo[0], aHi[1]-aLo[1])
	}

	borne := 0.5
	if v := os.Getenv("RENDU_BORNAGE"); v == "non" {
		borne = -1
		t.Log("TEMOIN : AUCUN bornage a la boite de l'instance")
	} else if v != "" {
		fmt.Sscanf(v, "%f", &borne)
	}
	// TRI DE LA ZONE JOUABLE par la finesse du maillage — defaut valide par l'utilisateur le
	// 2026-08-09 (cf. `zone_jouable.go`). RENDU_AIRE_MAX=0 le desactive pour la comparaison.
	aireMax := AireMaxTriangleJouable
	if v := os.Getenv("RENDU_AIRE_MAX"); v != "" {
		fmt.Sscanf(v, "%f", &aireMax)
	}
	t.Logf("tri de la zone jouable : aire mediane de triangle <= %.4f m2 (0 = desactive)", aireMax)
	var aires []float64

	assets := map[uint32]*RuntimeGeoAsset{}
	n, globaux, nonResolues, maillageNil, supprimees, ombres := 0, 0, 0, 0, 0, 0
	coupeesParPlafond, horsEmprise, tropGrossieres := 0, 0, 0
	var inconnues []Instance
	for _, in := range bsp.Instances {
		if in.QuickDeleted() {
			supprimees++
			continue
		}
		// Reclaimer ECARTE ces instances (`MeshIsCustomShadowCaster`) : leur maillage est un
		// volume de projection d'ombre, pas de la geometrie visible.
		if in.ProjecteurOmbre() {
			ombres++
			if !gardeOmbres {
				continue
			}
		}
		if empriseAncres && (in.AABBMax[0] < aLo[0] || in.AABBMin[0] > aHi[0] ||
			in.AABBMax[1] < aLo[1] || in.AABBMin[1] > aHi[1]) {
			horsEmprise++
			continue
		}
		if sansEchelle {
			in.Scale = [3]float64{1, 1, 1}
		}
		id := in.RuntimeGeoID()
		g, mod, ok := idx.Lookup(id)
		if !ok || g != "rtgo" {
			nonResolues++
			inconnues = append(inconnues, in)
			continue
		}
		// Les modules globaux sont RENDUS (defaut bascule le 2026-08-09) : ils portent la
		// moitie de la carte. RENDU_CARTE_SEULE rejoue l'ancien defaut.
		if filepath.Base(mod) != filepath.Base(modCarte) {
			globaux++
			if carteSeule {
				continue
			}
		}
		// RENDU_MODULE isole la contribution d'UN module. Sert a repondre a une question que
		// l'utilisateur a tranchee a l'oeil le 2026-08-09 : la zone jouable est le GRIS + ROUGE
		// de la carte des ecarts, le BLEU n'est pas jouable. Reste a savoir QUI produit le bleu.
		if unModule != "" && !strings.Contains(filepath.Base(mod), unModule) {
			continue
		}
		a, deja := assets[id]
		if !deja {
			tag, blob, err := idx.ExtractWithResources(id)
			if err != nil {
				continue
			}
			if a, err = NewRuntimeGeoAsset(tag, blob); err != nil {
				continue
			}
			assets[id] = a
		}
		if m := a.Mesh(in.MeshIndex); m != nil {
			if aireMax > 0 {
				aires = append(aires, AireMedianeProjetee(m, in))
			}
			if EstDecorGrossier(m, in, aireMax) {
				tropGrossieres++
				continue
			}
			n++
			// Combien d'instances le plafond supprime-t-il ENTIEREMENT ? Une carte a laquelle
			// il manque un etage doit se voir ici, pas seulement a l'oeil.
			if !math.IsNaN(plafond) && in.AABBMin[2] > plafond {
				coupeesParPlafond++
			}
			if borne < 0 {
				rendu.AddMesh(m, in)
			} else {
				rendu.AddMeshBorne(m, in, borne)
			}
		} else {
			maillageNil++
		}
	}
	t.Logf("bsp %d instances · %d rendues · %d globales · %d non resolues · %d maillage nil · %d supprimees · %d projecteurs d'ombre",
		len(bsp.Instances), n, globaux, nonResolues, maillageNil, supprimees, ombres)
	t.Logf("%d bsp dans le module", len(bsps))
	t.Logf("%d instances rendues sur une grille %d x %d a %.4f m/px", n, rendu.NX, rendu.NY, rendu.Cell)
	t.Logf("%d instances entierement au-dessus du plafond · %d hors emprise des ancres", coupeesParPlafond, horsEmprise)
	if len(aires) > 0 {
		t.Logf("aire mediane de triangle, par instance : p10 %.4f · p50 %.4f · p90 %.4f · p99 %.4f m2 (%d instances ecartees)",
			centile(aires, 0.10), centile(aires, 0.50), centile(aires, 0.90), centile(aires, 0.99), tropGrossieres)
	}

	// Trois vignettes : reference | reconstruction | carte des ecarts (cf. comparePixels).
	out := image.NewRGBA(image.Rect(0, 0, 3*larg+16, haut))
	couverts := 0
	for py := 0; py < haut; py++ {
		for px := 0; px < larg; px++ {
			out.Set(px, py, validee.At(bo.Min.X+px, bo.Min.Y+py))
		}
		for px := 0; px < 8; px++ {
			out.Set(larg+px, py, color.RGBA{40, 40, 40, 255})
		}
		j := haut - 1 - py
		for px := 0; px < larg; px++ {
			c := color.RGBA{0, 0, 0, 255}
			if e, ok := rendu.Eclairement(px, j); ok {
				couverts++
				g := uint8(255 * e)
				c = color.RGBA{g, g, g, 255}
			}
			out.Set(larg+8+px, py, c)
		}
	}
	// Empreinte des instances NON RESOLUES, en rouge : ou manque-t-il de la geometrie ?
	for _, in := range inconnues {
		for y := in.AABBMin[1]; y <= in.AABBMax[1]; y += gateEchelle {
			for x := in.AABBMin[0]; x <= in.AABBMax[0]; x += gateEchelle {
				px := int((x - gateX0) / gateEchelle)
				py := int((gateY1 - y) / gateEchelle)
				if px >= 0 && px < larg && py >= 0 && py < haut {
					out.Set(larg+8+px, py, color.RGBA{210, 40, 40, 255})
				}
			}
		}
	}
	t.Logf("%d px couverts sur %d (%.1f %%)", couverts, larg*haut, 100*float64(couverts)/float64(larg*haut))
	comparePixels(t, validee, bo, rendu, out, larg, haut)

	sortie := os.Getenv("RENDU_PNG")
	if sortie == "" {
		sortie = filepath.Join(os.TempDir(), "rendu.png")
	}
	ecritPNG(t, sortie, out)
	fmt.Println("planche de revue ecrite:", sortie)

	// L'EAU — volumes du tag sddt (variante any/), posee APRES comparePixels : le banc juge le
	// terrain, l'eau est un habillage du livrable.
	poseEauDepuisSddt(t, rendu, modCarte)

	// LA CARTE SEULE — le livrable, par opposition a la planche de revue ci-dessus.
	//
	// Fond TRANSPARENT, pas noir : le noir est une couleur, et il se retrouverait tel quel
	// dans les trous au milieu de la zone jouee. Un canal alpha laisse la couche d'affichage
	// poser son propre fond, ce que l'utilisateur a demande au gate du 2026-08-09.
	if seule := os.Getenv("RENDU_PNG_CARTE"); seule != "" {
		var masque []bool
		if os.Getenv("RENDU_MASQUE_REFERENCE") != "" {
			masque, _ = matiereReference(validee, bo, larg, haut)
			t.Log("REPLI : carte masquee par la silhouette de la reference (Cliffhanger UNIQUEMENT)")
		}
		// `combine` est le defaut depuis sa validation par l'utilisateur le 2026-08-09.
		style := styleRendu(os.Getenv("RENDU_STYLE"))
		if style == "" {
			style = StyleCarteParDefaut
		}
		bas, haute, _ := rendu.BornesAltitudeRobustes()
		t.Logf("style %q · bornes d'altitude robustes (centiles 2/98) [%+.2f ; %+.2f] m",
			style, bas, haute)
		ecritPNG(t, seule, carteSeulePNG(rendu, larg, haut, masque, style))
		fmt.Println("carte seule ecrite:", seule)
	}
}

// styleRendu : les habillages compares au gate de nettete du 2026-08-09.
type styleRendu string

const (
	styleDoux     styleRendu = "doux"     // Lambert continu — l'ancien defaut
	stylePlat     styleRendu = "plat"     // aplats + aretes
	styleAltitude styleRendu = "altitude" // nuancier d'altitude + ombrage
	styleCombine  styleRendu = "combine"  // nuancier + aplats + aretes
	styleJeu      styleRendu = "jeu"      // teinte par l ecart au NIVEAU JOUE + aplats + aretes
	styleEncre    styleRendu = "encre"    // quasi monochrome, la lisibilite vient des aretes

	// StyleCarteParDefaut : choisi par l'utilisateur au gate du 2026-08-09. `doux` seul ne
	// peut PAS separer deux dalles de meme inclinaison a des hauteurs differentes — c'est
	// structurel, pas un reglage (cf. `rendu_couleur.go`).
	StyleCarteParDefaut = styleCombine
)

// carteSeulePNG rend la reconstruction sans reference, sans separateur et sans diagnostic —
// l'empreinte des instances non resolues n'y figure PAS, c'est un outil de revue.
//
// `masque` non nil ne garde que les pixels ou il vaut true. Il n'existe que pour le REPLI
// explicite decrit au §10.11 du handoff : masquer par la silhouette de la carte validee ne
// vaut que pour Cliffhanger, seule carte a en posseder une. Ne jamais s'en servir comme regle.
func carteSeulePNG(rendu *Rendu, larg, haut int, masque []bool, style styleRendu) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, larg, haut))
	bas, haute, bornesOK := rendu.BornesAltitudeRobustes()
	for py := 0; py < haut; py++ {
		j := haut - 1 - py
		for px := 0; px < larg; px++ {
			if masque != nil && !masque[py*larg+px] {
				continue
			}
			// L'eau (volumes sddt, cf. PoseEau) prend le pixel — y compris la ou le rendu n'a
			// pas de matiere : l'eau n'est pas dans les instances de rendu, c'est justement
			// pourquoi elle manquait.
			if rendu.Eau(px, j) {
				img.Set(px, py, TeinteEau(rendu.BordEau(px, j)))
				continue
			}
			e, ok := rendu.Eclairement(px, j)
			if !ok {
				continue // alpha 0 : pas de matiere, pas de pixel
			}
			if style == stylePlat || style == styleCombine || style == styleJeu || style == styleEncre {
				e, _ = rendu.EclairementPlat(px, j)
			}
			c := color.RGBA{uint8(255 * e), uint8(255 * e), uint8(255 * e), 255}
			if (style == styleAltitude || style == styleCombine) && bornesOK {
				z, _ := rendu.Altitude(px, j)
				c = TeinteAltitude((z-bas)/(haute-bas), e)
			}
			// Les deux habillages qui remettent l'ARENE au premier plan : l'altitude y est
			// mesuree contre le niveau JOUE, pas contre elle-meme.
			if style == styleJeu || style == styleEncre {
				dz, okDz := rendu.EcartAuNiveauDeJeu(px, j)
				if !okDz {
					dz = 0
				}
				if style == styleJeu {
					c = TeinteNiveauDeJeu(dz, e)
				} else {
					c = TeinteEncre(dz, e)
				}
			}
			if style != styleDoux && style != styleAltitude && rendu.Arete(px, j) {
				c = color.RGBA{c.R / 3, c.G / 3, c.B / 3, 255}
			}
			img.Set(px, py, c)
		}
	}
	return img
}

func ecritPNG(t *testing.T, chemin string, img image.Image) {
	t.Helper()
	f, err := os.Create(chemin)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
}
