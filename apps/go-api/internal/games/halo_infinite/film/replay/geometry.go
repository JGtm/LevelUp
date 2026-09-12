package replay

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
)

// dist3 est LA distance euclidienne 3D du paquet, en mètres — un seul endroit où la formule est
// écrite (correctif de revue du 2026-08-17 : elle en avait quatre, dont deux à six paramètres).
//
// DEUX TRIPLETS ET PAS SIX FLOTTANTS : à six paramètres, une inversion d'argument entre les deux
// points ne se voit ni à la lecture ni au compilateur — et la règle des cinq paramètres du dépôt
// interdisait déjà la signature. Le triplet est la forme que le décodeur rend (`Vec3`, masques de
// position, `[3]float32` des pistes).
//
// UN GARDE-RAIL INTERDIT LA CINQUIÈME COPIE (`TestUneSeuleFormuleDeDistance3D`) : une
// factorisation sans garde-rail re-diverge, et c'est la règle n°6 du dépôt.
func dist3(a, b [3]float32) float64 {
	dx, dy, dz := float64(a[0]-b[0]), float64(a[1]-b[1]), float64(a[2]-b[2])
	return math.Sqrt(dx*dx + dy*dy + dz*dz)
}

// planDist est LA distance EN PLAN (XY) du paquet, en metres — l autre grandeur, a cote de
// `dist3`, et elle a sa raison d etre propre : la hauteur d un occupant et celle du repere d un
// vehicule ne se referent pas au meme point, une distance 3D ecarterait des embarquements reels
// (meme choix, meme raison qu aux socles). Elle est nommee pour que les deux ne se confondent
// pas a la lecture, et pour n avoir qu UNE ecriture comme sa soeur 3D.
func planDist(ax, ay, bx, by float32) float64 {
	return math.Hypot(float64(ax-bx), float64(ay-by))
}

// Fichiers du fond de carte, produits par le RE de la variante Forge (.mvar) et par la
// résolution des tags de modèle (cf. cmd/tmp_forgedim). Ils vivent sous
// `PathResolver.MapGeometryDir(titleSlug, module)` — donnée de référence versionnée, PAR CARTE
// pour les props (le paramètre `module` corrige un défaut : un seul répertoire servait ses props
// à toutes les cartes, cf. registry.go) et par TITRE pour le catalogue des types. Et non plus
// dans le répertoire de notes du chantier (lot 3.1).
const (
	MapObjectsFile  = "map_objects.csv"
	ObjectTypesFile = "forge_object_types.csv"
)

// LoadGeometry lit les props Forge d'UNE carte (map_objects.csv dans mapDir) et leurs emprises
// par type (forge_object_types.csv dans typesDir), et renvoie les objets DESSINABLES.
//
// DEUX RÉPERTOIRES, PARCE QUE LES DEUX FICHIERS N'ONT PAS LA MÊME PORTÉE. Les props
// appartiennent à une carte ; le catalogue des types (quel identifiant a quelle emprise) vaut
// pour tout le titre. Les lire au même endroit obligerait à recopier le catalogue sous chaque
// carte — autant de copies à faire diverger, là où le dépôt en interdit déjà trois.
//
// UNE CARTE SANS FICHIER DE PROPS N'EST PAS UNE ERREUR, c'est le CAS NOMINAL : personne n'a
// extrait les props des 79 cartes du catalogue de bornes. Elle rend zéro prop et aucune erreur,
// pour que le journal de cuisson ne crie pas à chaque match d'une carte non extraite. Le
// CATALOGUE, lui, reste obligatoire : sans lui aucun prop n'a d'emprise, et se taire rendrait
// indiscernables « cette carte n'a pas de props » et « le titre a perdu sa table des emprises ».
//
// Les types sans bounding box mesurée (modèles vides : points d'apparition, volumes de
// blocage) sont écartés — ils n'ont rien à afficher. Le second retour donne le nombre
// d'objets ainsi écartés, pour le journal d'assemblage.
func LoadGeometry(mapDir, typesDir string) ([]MapObject, int, error) {
	sizes, err := loadTypeExtents(filepath.Join(typesDir, ObjectTypesFile))
	if err != nil {
		return nil, 0, err
	}
	rows, cols, err := readCSV(filepath.Join(mapDir, MapObjectsFile))
	if errors.Is(err, os.ErrNotExist) {
		return nil, 0, nil
	}
	if err != nil {
		return nil, 0, err
	}
	var out []MapObject
	skipped := 0
	for _, r := range rows {
		typeID, err := strconv.ParseInt(field(r, cols, "type_id"), 10, 64)
		if err != nil {
			continue
		}
		ext, ok := sizes[typeID]
		if !ok {
			skipped++
			continue
		}
		out = append(out, MapObject{
			TypeID: typeID,
			X:      round2(parseF32(field(r, cols, "x"))),
			Y:      round2(parseF32(field(r, cols, "y"))),
			Z:      round2(parseF32(field(r, cols, "z"))),
			DX:     round2(ext[0]),
			DY:     round2(ext[1]),
			Yaw:    round2(parseF32(field(r, cols, "yaw_deg"))),
		})
	}
	return out, skipped, nil
}

// geomMeasured est la valeur de la colonne `geom` marquant une emprise réellement mesurée
// sur le modèle ; les autres valeurs (`modele_vide`) portent une emprise factice de
// 0,001 m — un filtre « dx > 0 » ne suffirait donc PAS à les écarter.
const geomMeasured = "ok"

// loadTypeExtents indexe l'emprise (dx, dy) par type_id ; les types sans emprise mesurée
// (volumes invisibles : points d'apparition, zones de blocage) sont absents de la table.
func loadTypeExtents(path string) (map[int64][2]float32, error) {
	rows, cols, err := readCSV(path)
	if err != nil {
		return nil, err
	}
	out := make(map[int64][2]float32, len(rows))
	for _, r := range rows {
		typeID, err := strconv.ParseInt(field(r, cols, "type_id"), 10, 64)
		if err != nil || field(r, cols, "geom") != geomMeasured {
			continue
		}
		dx, dy := parseF32(field(r, cols, "dx")), parseF32(field(r, cols, "dy"))
		if dx <= 0 || dy <= 0 {
			continue
		}
		out[typeID] = [2]float32{dx, dy}
	}
	return out, nil
}

// readCSV lit un CSV à en-tête et renvoie les lignes de données plus l'index colonne->rang.
func readCSV(path string) ([][]string, map[string]int, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = f.Close() }()
	rd := csv.NewReader(f)
	rd.FieldsPerRecord = -1
	head, err := rd.Read()
	if err != nil {
		return nil, nil, fmt.Errorf("en-tête %s: %w", path, err)
	}
	cols := make(map[string]int, len(head))
	for i, h := range head {
		cols[h] = i
	}
	var rows [][]string
	for {
		rec, err := rd.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, nil, fmt.Errorf("lecture %s: %w", path, err)
		}
		rows = append(rows, rec)
	}
	return rows, cols, nil
}

// field renvoie la valeur de la colonne nommée (vide si absente).
func field(rec []string, cols map[string]int, name string) string {
	i, ok := cols[name]
	if !ok || i >= len(rec) {
		return ""
	}
	return rec[i]
}

// parseF32 lit un float32 ; renvoie 0 si la valeur est absente ou invalide.
func parseF32(s string) float32 {
	v, err := strconv.ParseFloat(s, 32)
	if err != nil {
		return 0
	}
	return float32(v)
}

// ---------------------------------------------------------------------------
// LES ETENDUES du document — deplacees de `build.go` au correctif de revue du 2026-08-17,
// pour la meme raison que `document_labels.go` : le lot des socles avait pousse `build.go`
// de 621 a 640 lignes, au-dessus d un seuil deja gele par la baseline. C est un
// deplacement, aucune ligne de calcul n a change (le golden d assemblage fige les bornes).
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// LE REJET DES ECHANTILLONS ABERRANTS (2026-09-08)
//
// LE DEFAUT QU IL CORRIGE. `boundsOf` etait un min/max BRUT sur tous les points publies :
// UN seul echantillon faux suffisait a definir plusieurs bornes. Mesure sur l artefact
// `81c02726` (Isolement, Bases) : le point (slot 523, image 645) sort a
// x=-78.6 y=+46.38 z=-325.4 alors que le sol joue est a 117.8 en mediane — quatre cent
// quarante metres SOUS le terrain, sur une carte qui n a pas de vide. Il est SEUL : un point
// sur 16 064. Et il fixait a lui tout seul MinX, MaxY et MinZ.
//
// CE QUE CA CASSAIT EN AVAL, ET C EST DOUBLE. Le client ecarte le fond de carte quand l image
// ne contient pas les bornes (`coversPlayedArea`) : sur ces matchs le fond disparaissait alors
// que l image couvre 99 % des positions. Et les MEMES bornes cadrent la scene (`sceneBounds`) :
// la carte etait dessinee plus petite qu elle ne devait, pour loger un point fantome.
//
// LE SEUIL EST MESURE, PAS CHOISI. Balayage des 64 artefacts du parc, par axe : l ecart du
// point le plus lointain, exprime en unites d ETENDUE CENTRALE (p1..p99). La population se
// separe en deux, avec un trou franc :
//
//	artefacts de decodage : 105.3  71.9  65.5  49.3  47.7  46.8  45.9  38.5  17.7
//	--- trou ---
//	jeu legitime          :   9.5   8.2   8.1   7.0   5.9   5.5   5.2  ...  (mediane 0.12)
//
// `boundsRejectSpreads = 12` tombe dans ce trou : 1.5x sous le plus petit artefact, 1.3x
// au-dessus du plus grand ecart legitime. Un tir depuis un perchoir ou une chute dans un vide
// REEL (Launch Site) reste tres en deca — c est ce que dit la colonne de droite.
//
// POURQUOI L ETENDUE CENTRALE ET NON L ECART-TYPE : un seul point a -325 m tire la moyenne ET
// l ecart-type, donc se masque lui-meme. Les centiles, non.
// ---------------------------------------------------------------------------

// boundsRejectSpreads : au-dela de combien d'ETENDUES CENTRALES un echantillon est tenu pour
// un artefact de decodage. Calibre sur le parc (cf. le bloc ci-dessus).
const boundsRejectSpreads = 12

// boundsMinSamples : en deca, aucun rejet. Les centiles n'ont pas de sens sur une poignee de
// points, et un match si court ne vaut pas le risque d'ecarter une position vraie.
const boundsMinSamples = 200

// boundsMinSpread : plancher de l'etendue centrale, en metres. Sans lui, une partie ou tout le
// monde reste sur un plan (spread 0) rejetterait le moindre deplacement.
const boundsMinSpread = 0.5

// boundsOf calcule l'étendue XY (et Z) des points publiés, en écartant les échantillons
// ABERRANTS (cf. le bloc ci-dessus). Le second retour est le nombre de points écartés — il est
// journalisé par l'appelant, jamais avalé.
func boundsOf(tracks []Track) (Bounds, int) {
	xs, ys, zs := axisValues(tracks)
	if len(xs) < boundsMinSamples {
		return rawBounds(tracks, nil), 0
	}
	garde := [3]axisGuard{guardOf(xs), guardOf(ys), guardOf(zs)}
	aberrant := func(p Point) bool {
		return garde[0].rejects(p.X) || garde[1].rejects(p.Y) || garde[2].rejects(p.Z)
	}
	b := rawBounds(tracks, aberrant)
	// GARDE DE DERNIER RESSORT : si le filtre a tout ecarte, il s est trompe sur la forme de la
	// donnee, pas la donnee sur elle-meme. On rend les bornes brutes plutot qu'une etendue vide.
	if b.MinX > b.MaxX {
		return rawBounds(tracks, nil), 0
	}
	return b, countRejected(tracks, aberrant)
}

// axisGuard : les bornes d'acceptation d'un axe, deduites de son etendue centrale.
type axisGuard struct{ lo, hi float32 }

func (g axisGuard) rejects(v float32) bool { return v < g.lo || v > g.hi }

// guardOf deduit les bornes d'acceptation d'un axe. `vals` est TRIE par l'appelant.
func guardOf(vals []float32) axisGuard {
	n := len(vals)
	p1, p99 := vals[n/100], vals[99*n/100]
	spread := p99 - p1
	if spread < boundsMinSpread {
		spread = boundsMinSpread
	}
	marge := boundsRejectSpreads * spread
	return axisGuard{lo: p1 - marge, hi: p99 + marge}
}

// axisValues rend les trois axes de tous les points, TRIES — la forme qu'attend `guardOf`.
func axisValues(tracks []Track) (xs, ys, zs []float32) {
	n := 0
	for _, tr := range tracks {
		n += len(tr.Points)
	}
	xs, ys, zs = make([]float32, 0, n), make([]float32, 0, n), make([]float32, 0, n)
	for _, tr := range tracks {
		for _, p := range tr.Points {
			xs, ys, zs = append(xs, p.X), append(ys, p.Y), append(zs, p.Z)
		}
	}
	sortFloats(xs)
	sortFloats(ys)
	sortFloats(zs)
	return xs, ys, zs
}

func sortFloats(v []float32) {
	sort.Slice(v, func(i, j int) bool { return v[i] < v[j] })
}

// rawBounds accumule l'étendue des points que `skip` ne rejette pas (`nil` = tous).
func rawBounds(tracks []Track, skip func(Point) bool) Bounds {
	var b Bounds
	first := true
	for _, tr := range tracks {
		for _, p := range tr.Points {
			if skip != nil && skip(p) {
				continue
			}
			if first {
				b = Bounds{MinX: p.X, MinY: p.Y, MaxX: p.X, MaxY: p.Y, MinZ: p.Z, MaxZ: p.Z}
				first = false
				continue
			}
			b.MinX, b.MaxX = minf(b.MinX, p.X), maxf(b.MaxX, p.X)
			b.MinY, b.MaxY = minf(b.MinY, p.Y), maxf(b.MaxY, p.Y)
			b.MinZ, b.MaxZ = minf(b.MinZ, p.Z), maxf(b.MaxZ, p.Z)
		}
	}
	// Aucun point retenu : `first` est reste vrai et `b` est le zero. On le SIGNALE par une
	// etendue inversee, que l'appelant reconnait (cf. la garde de dernier ressort).
	if first {
		return Bounds{MinX: 1, MaxX: -1}
	}
	return b
}

// countRejected compte les échantillons écartés — la mesure que l'appelant journalise.
func countRejected(tracks []Track, skip func(Point) bool) int {
	n := 0
	for _, tr := range tracks {
		for _, p := range tr.Points {
			if skip(p) {
				n++
			}
		}
	}
	return n
}

// geometryBounds calcule l'étendue XY des props (nil si pas de géométrie).
func geometryBounds(objs []MapObject) *Bounds {
	if len(objs) == 0 {
		return nil
	}
	b := Bounds{MinX: objs[0].X, MinY: objs[0].Y, MaxX: objs[0].X, MaxY: objs[0].Y}
	for _, o := range objs[1:] {
		b.MinX, b.MaxX = minf(b.MinX, o.X), maxf(b.MaxX, o.X)
		b.MinY, b.MaxY = minf(b.MinY, o.Y), maxf(b.MaxY, o.Y)
	}
	return &b
}

func minf(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}

func maxf(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}
