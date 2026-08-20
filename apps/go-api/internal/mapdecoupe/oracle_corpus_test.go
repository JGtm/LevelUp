package mapdecoupe

// oracle_corpus_test.go — CE QUE L'ORACLE LIT.
//
// L'instrument ne fabrique aucune donnée : il lit le catalogue versionné, les fonds
// publiés, le dump du POC et les documents de rejeu déjà cuits. Rien n'ouvre le jeu, rien
// ne touche une base, rien ne va sur le réseau — et si l'une de ces pièces manque, le test
// se DÉCLARE absent (t.Skip) au lieu de passer au vert sur du vide.

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/testutil"
)

// corpus rassemble les pièces de l'oracle.
type corpus struct {
	racine string
	res    *title.PathResolver
	cat    *replay.MapCalloutsCatalog
	dump   map[int]zonePOC
	// modules : les cartes du catalogue dont le fond est publié, triées.
	modules []string
	// calage : la grille de chaque carte, SANS décoder son image (2,6 M de pixels qu'on ne
	// lit que quand on découpe pour de bon).
	calage map[string]*Masque
	// niveau : l'altitude du sol joué publiée par le sidecar — la signature qui sépare les
	// cartes à la reconnaissance.
	niveau map[string]float64
}

// zonePOC est une zone du dump découpé du POC (l'oracle faible du découpage).
type zonePOC struct {
	VolumeIndex int    `json:"volumeIndex"`
	LibelleEN   string `json:"libelle_en"`
	Utiliser    string `json:"polygone_a_utiliser"`
	Brut        struct {
		Polygone [][2]float64 `json:"polygone"`
	} `json:"brut"`
	Decoupe struct {
		Parties []struct {
			Contour [][2]float64   `json:"contour"`
			Trous   [][][2]float64 `json:"trous"`
		} `json:"parties"`
	} `json:"decoupe"`
}

// anneaux rend la figure SERVIE par le POC pour cette zone (découpée ou repliée sur le brut).
func (z zonePOC) anneaux() [][][2]float64 {
	if z.Utiliser != "decoupe" || len(z.Decoupe.Parties) == 0 {
		return [][][2]float64{z.Brut.Polygone}
	}
	var out [][][2]float64
	for _, p := range z.Decoupe.Parties {
		out = append(out, p.Contour)
		out = append(out, p.Trous...)
	}
	return out
}

// moduleRidgeline : la carte que le POC a découpée, donc la seule où l'oracle IoU existe.
const moduleRidgeline = "ridgeline"

func ouvreCorpus(t *testing.T) corpus {
	t.Helper()
	racine, err := testutil.RepoRoot()
	if err != nil {
		t.Fatalf("racine du dépôt introuvable : %v", err)
	}
	res := title.NewPathResolver(racine)
	cat, err := replay.LoadMapCallouts(res.MapCalloutsPath(title.DefaultSlug))
	if err != nil {
		t.Skip("catalogue de callouts absent :", err)
	}
	c := corpus{racine: racine, res: res, cat: cat,
		calage: map[string]*Masque{}, niveau: map[string]float64{}}
	c.dump = chargeDumpPOC(t, filepath.Join(racine, ".ai", "V7.5", "dumps",
		"callout_zones_ridgeline_clipped.json"))
	for module := range cat.Maps {
		meta := res.MapBackgroundMetaPath(title.DefaultSlug, module)
		if _, err := os.Stat(meta); err != nil {
			continue // pas de fond publié : la carte reste brute, elle n'est pas mesurable
		}
		b, err := replay.LoadMapBackground(meta)
		if err != nil {
			t.Fatalf("calage de %s illisible : %v", module, err)
		}
		c.modules = append(c.modules, module)
		c.calage[module] = &Masque{Module: module, Calage: b.Calibration,
			NX: b.Calibration.WidthPx, NY: b.Calibration.HeightPx}
		c.niveau[module] = b.Stats.PlayLevelZ
	}
	sort.Strings(c.modules)
	return c
}

func chargeDumpPOC(t *testing.T, path string) map[int]zonePOC {
	t.Helper()
	// Fichier VERSIONNÉ (git ls-files confirme .ai/V7.5/dumps/callout_zones_ridgeline_clipped.json)
	// : son absence sur un checkout propre est une anomalie, pas un cas nominal — t.Fatalf, pas
	// t.Skip (un skip aurait rendu cet oracle muet en CI, cf. l'en-tête du fichier).
	blob, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("dump découpé du POC illisible (fichier versionné) : %v", err)
	}
	var d struct {
		Zones []zonePOC `json:"zones"`
	}
	if err := json.Unmarshal(blob, &d); err != nil {
		t.Fatalf("dump découpé illisible : %v", err)
	}
	out := make(map[int]zonePOC, len(d.Zones))
	for _, z := range d.Zones {
		out[z.VolumeIndex] = z
	}
	return out
}

// masque charge le fond d'une carte, fermeture de production appliquée. nil = pas de fond
// publié (cas nominal : la carte reste brute).
func (c corpus) masque(t *testing.T, module string, tolerance float64) *Masque {
	t.Helper()
	png := c.res.MapBackgroundPath(title.DefaultSlug, module)
	if _, err := os.Stat(png); err != nil {
		return nil
	}
	m, err := ChargeMasque(png, c.res.MapBackgroundMetaPath(title.DefaultSlug, module))
	if err != nil {
		t.Fatalf("fond de %s illisible : %v", module, err)
	}
	return m.Comble(tolerance)
}

// brutDe rend le PAVÉ DU DESIGNER d'une zone, quel que soit l'état du catalogue.
//
// Trois cas, et c'est le prix d'un catalogue qui peut être déjà découpé quand on le mesure :
// Ridgeline vient du dump du POC, une carte déjà passée par la chaîne universelle vient de
// `cat.Brut`, une carte encore brute sert son propre polygone.
func (c corpus) brutDe(module string, z replay.CalloutZone) [][2]float64 {
	if module == moduleRidgeline {
		if d, ok := c.dump[z.VolumeIndex]; ok && len(d.Brut.Polygone) >= 3 {
			return d.Brut.Polygone
		}
	}
	for _, b := range c.cat.Brut[module] {
		if b.VolumeIndex == z.VolumeIndex {
			return b.Polygon
		}
	}
	return z.Polygon
}

// figureBrute rend les pavés du designer d'une carte, une figure par zone à forme propre.
func (c corpus) figureBrute(module string) [][][][2]float64 {
	var out [][][][2]float64
	for _, z := range c.cat.Maps[module].Zones {
		if p := c.brutDe(module, z); len(p) >= 3 {
			out = append(out, [][][2]float64{p})
		}
	}
	return out
}

// Réglages de la reconnaissance de carte.
const (
	// pasEchantillonID : une position sur N suffit à reconnaître une carte, et le balayage
	// reste rapide sur 27 documents x 19 cartes.
	pasEchantillonID = 50
	// margeZID : de combien un joueur peut sortir de la tranche verticale de sa zone (saut,
	// chute, plateforme mince). Serré, parce que c'est LUI qui sépare deux cartes dont les
	// pavés se recouvrent vus du dessus.
	margeZID = 2.0
	// scoreMinID / ecartMinID : le meilleur candidat doit contenir 90 % des positions ET
	// devancer le suivant de 15 points. Sinon le film est ÉCARTÉ — aucune carte ne lui est
	// attribuée, plutôt qu'une carte plausible.
	//
	// UN ÉCART ABSOLU, PAS UN RAPPORT : les pavés du designer couvrent toute l'emprise d'une
	// carte, donc plusieurs cartes contiennent 100 % des positions d'un petit match. Un
	// rapport (« double le suivant ») n'y départage jamais rien ; l'écart, si.
	//
	// UNE ERREUR D'ATTRIBUTION NE PEUT PAS FAIRE PASSER LE GATE, elle ne peut que le faire
	// échouer : un film rattaché à la mauvaise carte garde un score brut élevé (les pavés
	// couvrent tout) mais s'effondre au découpé, ce que le seuil de perte refuse. Le sens de
	// l'erreur est donc sûr.
	scoreMinID = 0.80
	ecartMinID = 0.15
	// ecartNiveauMax : écart toléré, en mètres, entre l'altitude MÉDIANE d'un film et le sol
	// joué publié par le fond de la carte. Deux étages de carte — assez pour une carte à
	// fort dénivelé, trop peu pour confondre une arène native avec un canevas Forge.
	ecartNiveauMax = 20.0
)

// identifie reconnaît la carte de chaque film : celle dont les PRISMES bruts contiennent la
// plus grande part de ses positions, altitude comprise.
//
// POURQUOI PAS LA BASE. Le document de rejeu ne nomme pas sa carte — en production c'est le
// registre des matchs qui la nomme. Un instrument de mesure n'a pas à ouvrir une base DuckDB
// pour ça.
//
// POURQUOI L'ALTITUDE EST INDISPENSABLE ICI, mesuré le 2026-08-16 : sans elle, un match de
// FORGE (canevas posé vers z = +61) se faisait attribuer `btb_exiled` à 87 % — les pavés du
// designer, vus du dessus, se recouvrent d'une carte à l'autre. Deux filtres verticaux la
// séparent : le SOL JOUÉ publié par le fond (indépendant des zones ET du masque, donc pas
// tautologique) écarte les cartes du mauvais étage, puis la tranche du prisme tranche entre
// les survivantes. Le rejet explicite garde l'échantillon propre : une carte Forge n'a
// AUCUNE zone nommée, elle ne doit ressembler à personne.
func identifie(t *testing.T, c corpus, films []film) {
	t.Helper()
	for i := range films {
		zmed := medianeAltitude(films[i].pts)
		meilleur, second, nom, nom2 := 0.0, 0.0, "", ""
		for _, module := range c.modules {
			if math.Abs(zmed-c.niveau[module]) > ecartNiveauMax {
				continue
			}
			s := c.scoreCarte(module, films[i].pts)
			if s > meilleur {
				meilleur, second, nom, nom2 = s, meilleur, module, nom
				continue
			}
			if s > second {
				second, nom2 = s, module
			}
		}
		if meilleur >= scoreMinID && meilleur-second >= ecartMinID {
			films[i].module = nom
			continue
		}
		films[i].ecart = fmt.Sprintf("z=%.1f ; %s %.0f%% contre %s %.0f%%",
			zmed, vide(nom), 100*meilleur, vide(nom2), 100*second)
	}
}

func vide(s string) string {
	if s == "" {
		return "(aucune)"
	}
	return s
}

// medianeAltitude rend l'altitude médiane d'un film — robuste aux quelques positions en
// vol (chute, véhicule, grappin) que la moyenne ou les bornes déplaceraient.
func medianeAltitude(pts [][3]float64) float64 {
	if len(pts) == 0 {
		return 0
	}
	zs := make([]float64, 0, len(pts)/pasEchantillonID+1)
	for k := 0; k < len(pts); k += pasEchantillonID {
		zs = append(zs, pts[k][2])
	}
	sort.Float64s(zs)
	return zs[len(zs)/2]
}

// scoreCarte rend la part des positions échantillonnées qui tombent DANS LE CADRE publié de
// la carte ET dans un de ses prismes.
//
// Le cadre est le second critère indépendant : il vient du fond (voisinage des ancres), pas
// des zones. Sans lui, un match de Catalyst dont l'action descend à x = -231 se faisait
// attribuer `btb_exiled` — dont les pavés, prolongés, contiennent ce point alors que sa carte
// s'arrête à x = -77.
// Le score est le MINIMUM de deux couvertures, et c'est ce qui le rend décisif :
//   - la carte contient-elle le match (part des positions dans un prisme) ;
//   - le match a-t-il visité la carte (part des GRANDES zones touchées).
//
// La première seule ne sépare rien : une grande carte contient une petite arène tout entière
// et marque 100 %. La seconde la disqualifie — un match d'arène ne visite qu'une poignée des
// zones d'une carte BTB.
func (c corpus) scoreCarte(module string, pts [][3]float64) float64 {
	zones := c.cat.Maps[module].Zones
	cadre := c.calage[module]
	vues := make([]bool, len(zones))
	n, dedans := 0, 0
	for k := 0; k < len(pts); k += pasEchantillonID {
		n++
		p := pts[k]
		if _, _, ok := cadre.Calage.MondeVersPixel(p[0], p[1]); !ok {
			continue
		}
		touche := false
		for i, z := range zones {
			if p[2] < z.ZBottom-margeZID || p[2] > z.ZTop+margeZID {
				continue
			}
			poly := c.brutDe(module, z)
			if len(poly) >= 3 && dansPolygone(p[0], p[1], poly) {
				vues[i], touche = true, true
			}
		}
		if touche {
			dedans++
		}
	}
	if n == 0 {
		return 0
	}
	grandes, visitees := 0, 0
	for i, z := range zones {
		if !z.Big {
			continue
		}
		grandes++
		if vues[i] {
			visitees++
		}
	}
	part := float64(dedans) / float64(n)
	if grandes == 0 {
		return part
	}
	return math.Min(part, float64(visitees)/float64(grandes))
}

// dansPolygone : test de parité par lancer de rayon horizontal.
func dansPolygone(x, y float64, poly [][2]float64) bool {
	dedans := false
	for i, j := 0, len(poly)-1; i < len(poly); j, i = i, i+1 {
		if (poly[i][1] > y) != (poly[j][1] > y) &&
			x < (poly[j][0]-poly[i][0])*(y-poly[i][1])/(poly[j][1]-poly[i][1])+poly[i][0] {
			dedans = !dedans
		}
	}
	return dedans
}

// film : les positions réellement jouées d'un match, et la carte qu'on leur a reconnue.
//
// Les positions sont en 3D : l'altitude ne sert pas à découper (le masque ne la porte pas)
// mais elle SÉPARE les cartes à la reconnaissance.
type film struct {
	id     string
	pts    [][3]float64
	module string
	// ecart dit POURQUOI un film n a pas ete rattache a une carte.
	ecart string
}

// chargeFilms lit les documents de rejeu déjà cuits. Seules les positions comptent.
func chargeFilms(t *testing.T, c corpus) []film {
	t.Helper()
	dir := c.res.ReplayArtifactsDir(title.DefaultSlug)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Skip("documents de rejeu absents :", err)
	}
	var films []film
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		blob, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatalf("document %s illisible : %v", e.Name(), err)
		}
		var doc struct {
			Tracks []struct {
				Points []struct {
					X float64 `json:"x"`
					Y float64 `json:"y"`
					Z float64 `json:"z"`
				} `json:"points"`
			} `json:"tracks"`
		}
		if err := json.Unmarshal(blob, &doc); err != nil {
			t.Fatalf("document %s invalide : %v", e.Name(), err)
		}
		f := film{id: strings.TrimSuffix(e.Name(), ".json")}
		for _, tr := range doc.Tracks {
			for _, p := range tr.Points {
				f.pts = append(f.pts, [3]float64{p.X, p.Y, p.Z})
			}
		}
		if len(f.pts) > 0 {
			films = append(films, f)
		}
	}
	return films
}

// unionRaster peint toutes les figures d'une carte sur la grille du fond.
func unionRaster(m *Masque, zones [][][][2]float64) []bool {
	g := make([]bool, m.NX*m.NY)
	for _, anneaux := range zones {
		e, ok := m.emprise(anneaux)
		if !ok {
			continue
		}
		r := m.rasterise(anneaux, e)
		for k, v := range r {
			if v {
				g[(e.j0+k/e.nx)*m.NX+e.i0+k%e.nx] = true
			}
		}
	}
	return g
}

// partDansGrille rend la part des positions qui tombent sur une cellule allumée.
func partDansGrille(m *Masque, g []bool, pts [][3]float64) float64 {
	if len(pts) == 0 {
		return 0
	}
	n := 0
	for _, p := range pts {
		px, py, ok := m.Calage.MondeVersPixel(p[0], p[1])
		if ok && g[py*m.NX+px] {
			n++
		}
	}
	return float64(n) / float64(len(pts))
}

// iou rend l'accord de deux figures, mesuré sur la grille du fond.
func iou(m *Masque, a, b [][][2]float64) float64 {
	tout := append(append([][][2]float64{}, a...), b...)
	e, ok := m.emprise(tout)
	if !ok {
		return 0
	}
	ra, rb := m.rasterise(a, e), m.rasterise(b, e)
	inter, union := 0, 0
	for k := range ra {
		switch {
		case ra[k] && rb[k]:
			inter++
			union++
		case ra[k] || rb[k]:
			union++
		}
	}
	if union == 0 {
		return 0
	}
	return float64(inter) / float64(union)
}
