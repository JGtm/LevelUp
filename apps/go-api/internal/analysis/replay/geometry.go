package replay

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
)

// Fichiers du fond de carte, produits par le RE de la variante Forge (.mvar) et par la
// résolution des tags de modèle (cf. cmd/tmp_forgedim). Ils vivent sous
// `PathResolver.MapGeometryDir(titleSlug)` — donnée de référence versionnée du titre,
// et non plus dans le répertoire de notes du chantier (lot 3.1).
const (
	MapObjectsFile  = "map_objects.csv"
	ObjectTypesFile = "forge_object_types.csv"
)

// LoadGeometry lit les props Forge (map_objects.csv) et leurs emprises par type
// (forge_object_types.csv) depuis dir, et renvoie les objets DESSINABLES.
//
// Les types sans bounding box mesurée (modèles vides : points d'apparition, volumes de
// blocage) sont écartés — ils n'ont rien à afficher. Le second retour donne le nombre
// d'objets ainsi écartés, pour le journal d'assemblage.
func LoadGeometry(dir string) ([]MapObject, int, error) {
	sizes, err := loadTypeExtents(filepath.Join(dir, ObjectTypesFile))
	if err != nil {
		return nil, 0, err
	}
	rows, cols, err := readCSV(filepath.Join(dir, MapObjectsFile))
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
