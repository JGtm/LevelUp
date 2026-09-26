package replay

// geometry_csv.go — LA LECTURE DES CSV DE GEOMETRIE, ligne par ligne et NOMMEE (lot J2.10,
// constat RA1-7, 2026-09-26). Extrait de geometry.go : la lecture et son erreur typee vivent
// ensemble, le chargeur n y garde que la regle « quoi dessiner ».

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
)

// LigneDeGeometrieError nomme une valeur ILLISIBLE d'un CSV de géométrie : fichier, ligne (1 =
// l'en-tête), colonne et valeur. Elle remplace le saut silencieux d'une ligne et le zéro rendu
// pour une coordonnée illisible — un prop disparu ou dessiné à l'origine sans une ligne de
// journal. L'appelant journalise, puis dégrade.
type LigneDeGeometrieError struct {
	Fichier string
	Ligne   int
	Colonne string
	Valeur  string
	Err     error
}

func (e *LigneDeGeometrieError) Error() string {
	return fmt.Sprintf("geometrie de carte : %s ligne %d, colonne %q : valeur %q illisible : %v",
		e.Fichier, e.Ligne, e.Colonne, e.Valeur, e.Err)
}

func (e *LigneDeGeometrieError) Unwrap() error { return e.Err }

// geomMeasured est la valeur de la colonne `geom` marquant une emprise réellement mesurée
// sur le modèle ; les autres valeurs (`modele_vide`) portent une emprise factice de
// 0,001 m — un filtre « dx > 0 » ne suffirait donc PAS à les écarter.
const geomMeasured = "ok"

// ligneCSV est une ligne de données d'un CSV à en-tête, avec son NUMÉRO dans le fichier (1 =
// l'en-tête) : c'est ce qui permet de nommer une valeur illisible.
type ligneCSV struct {
	fichier string
	numero  int
	rec     []string
	cols    map[string]int
}

// champ renvoie la valeur de la colonne nommée (vide si absente).
func (l ligneCSV) champ(col string) string {
	i, ok := l.cols[col]
	if !ok || i >= len(l.rec) {
		return ""
	}
	return l.rec[i]
}

// entier lit un entier de la colonne `col`, ou rend l'erreur qui la nomme.
func (l ligneCSV) entier(col string) (int64, error) {
	v := l.champ(col)
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return 0, l.illisible(col, v, err)
	}
	return n, nil
}

// flottant lit un float32 de la colonne `col`, ou rend l'erreur qui la nomme. UNE VALEUR VIDE
// EST ILLISIBLE : la lire 0 dessinerait le prop à l'origine.
func (l ligneCSV) flottant(col string) (float32, error) {
	v := l.champ(col)
	f, err := strconv.ParseFloat(v, 32)
	if err != nil {
		return 0, l.illisible(col, v, err)
	}
	return float32(f), nil
}

func (l ligneCSV) illisible(col, valeur string, err error) error {
	return &LigneDeGeometrieError{Fichier: l.fichier, Ligne: l.numero, Colonne: col, Valeur: valeur, Err: err}
}

// position lit la position et le lacet d'un prop, arrondis comme le document les publie.
func (l ligneCSV) position(o *MapObject) error {
	for _, c := range []struct {
		col string
		dst *float32
	}{{"x", &o.X}, {"y", &o.Y}, {"z", &o.Z}, {"yaw_deg", &o.Yaw}} {
		v, err := l.flottant(c.col)
		if err != nil {
			return err
		}
		*c.dst = round2(v)
	}
	return nil
}

// loadTypeExtents indexe l'emprise (dx, dy) par type_id ; les types sans emprise mesurée
// (volumes invisibles : points d'apparition, zones de blocage) sont absents de la table. Une
// ligne MESURÉE dont l'identifiant ou l'emprise ne se lit pas est une erreur nommée ; une ligne
// non mesurée est écartée sans lire son emprise, qui est factice.
func loadTypeExtents(path string) (map[int64][2]float32, error) {
	lignes, err := readCSV(path)
	if err != nil {
		return nil, err
	}
	out := make(map[int64][2]float32, len(lignes))
	for _, l := range lignes {
		if l.champ("geom") != geomMeasured {
			continue
		}
		typeID, err := l.entier("type_id")
		if err != nil {
			return nil, err
		}
		dx, err := l.flottant("dx")
		if err != nil {
			return nil, err
		}
		dy, err := l.flottant("dy")
		if err != nil {
			return nil, err
		}
		if dx <= 0 || dy <= 0 {
			continue
		}
		out[typeID] = [2]float32{dx, dy}
	}
	return out, nil
}

// readCSV lit un CSV à en-tête et renvoie ses lignes de données, numérotées.
func readCSV(path string) ([]ligneCSV, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	rd := csv.NewReader(f)
	rd.FieldsPerRecord = -1
	head, err := rd.Read()
	if err != nil {
		return nil, fmt.Errorf("en-tête %s: %w", path, err)
	}
	cols := make(map[string]int, len(head))
	for i, h := range head {
		cols[h] = i
	}
	var lignes []ligneCSV
	for {
		rec, err := rd.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("lecture %s: %w", path, err)
		}
		numero, _ := rd.FieldPos(0)
		lignes = append(lignes, ligneCSV{fichier: path, numero: numero, rec: rec, cols: cols})
	}
	return lignes, nil
}
