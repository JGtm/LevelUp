package filmprofile

// empreinte_bornes.go — L EMPREINTE DU CATALOGUE DE BORNES, SANS EN RECOPIER UNE LIGNE.
//
// La part DERIVEE du profil (ce qui se deduit des fichiers du jeu) existe deja ailleurs :
// `data/titles/{slug}/reference/map_quant_bounds.json`, produit par `cmd/mapquant-build`. La
// dupliquer dans le profil la ferait diverger au premier build ajoute (regle 6 du depot). Le
// profil en porte donc l EMPREINTE, et c est elle qui repond a la seule question utile :
// « ce profil est-il solidaire du catalogue de bornes qui est dans l arbre ? »
//
// DEUX CHOIX A JUSTIFIER :
//
//	L EMPREINTE PORTE SUR `maps` SEUL, pas sur le fichier entier. Le champ `source` du
//	catalogue de bornes cite le chemin d installation de la machine qui l a produit
//	(`D:\Jeux\...`) : c est une trace de fabrication, pas une donnee du jeu, et la faire
//	entrer dans l empreinte rendrait le gate rouge d un poste a l autre sans qu aucune borne
//	n ait bouge.
//
//	ELLE SE CALCULE SUR LA STRUCTURE, pas sur le texte. Chaque entree est REMISE A PLAT
//	(`json.Compact`) avant d etre hachee, et les clefs sont triees : une reindentation du
//	fichier ne change pas l empreinte, un chiffre si. Ce paquet ne connait donc PAS le type
//	`filmdec.MapQuantEntry` — il n a pas a le connaitre, et ne doit pas l importer.

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
)

// BornesResumees est ce que l on retient d un catalogue de bornes de carte.
type BornesResumees struct {
	// SchemaVersion est la version de schema declaree par le catalogue de bornes.
	SchemaVersion int
	// Cartes est le nombre d entrees de carte.
	Cartes int
	// Empreinte est le sha256 hexadecimal de ces entrees, hors champs de machine.
	Empreinte string
}

// bornesBrutes : la vue MINIMALE du catalogue de bornes. Les entrees restent du JSON brut —
// ce paquet n a pas a savoir ce qu il y a dedans, seulement a le hacher a l identique.
type bornesBrutes struct {
	SchemaVersion int                        `json:"schemaVersion"`
	Maps          map[string]json.RawMessage `json:"maps"`
}

// ResumerBornesDeCarte calcule l empreinte structurelle d un catalogue de bornes de carte.
func ResumerBornesDeCarte(blob []byte) (BornesResumees, error) {
	var brut bornesBrutes
	if err := json.Unmarshal(blob, &brut); err != nil {
		return BornesResumees{}, fmt.Errorf("catalogue de bornes illisible : %w", err)
	}
	if len(brut.Maps) == 0 {
		return BornesResumees{}, fmt.Errorf("catalogue de bornes sans aucune carte")
	}
	noms := make([]string, 0, len(brut.Maps))
	for nom := range brut.Maps {
		noms = append(noms, nom)
	}
	sort.Strings(noms)

	h := sha256.New()
	var plat bytes.Buffer
	for _, nom := range noms {
		plat.Reset()
		if err := json.Compact(&plat, brut.Maps[nom]); err != nil {
			return BornesResumees{}, fmt.Errorf("carte %q illisible : %w", nom, err)
		}
		h.Write([]byte(nom))
		h.Write([]byte{0})
		h.Write(plat.Bytes())
		h.Write([]byte{0})
	}
	return BornesResumees{
		SchemaVersion: brut.SchemaVersion,
		Cartes:        len(brut.Maps),
		Empreinte:     hex.EncodeToString(h.Sum(nil)),
	}, nil
}

// CorrespondAuxBornes dit si le bloc derive du catalogue decrit bien ces bornes-la.
func (d Derive) CorrespondAuxBornes(r BornesResumees) error {
	var ecarts []string
	if d.SchemaVersion != r.SchemaVersion {
		ecarts = append(ecarts, fmt.Sprintf("schemaVersion %d contre %d", d.SchemaVersion, r.SchemaVersion))
	}
	if d.Cartes != r.Cartes {
		ecarts = append(ecarts, fmt.Sprintf("maps %d contre %d", d.Cartes, r.Cartes))
	}
	if d.EmpreinteCartes != r.Empreinte {
		ecarts = append(ecarts, fmt.Sprintf("mapsSha256 %s contre %s", d.EmpreinteCartes, r.Empreinte))
	}
	if len(ecarts) == 0 {
		return nil
	}
	return fmt.Errorf("le bloc derive ne decrit pas le catalogue de bornes lu : %v — "+
		"rejouer `go run ./cmd/film-profiles-build` (cf. docs/RUNBOOK_FILM_PROFILES.md)", ecarts)
}
