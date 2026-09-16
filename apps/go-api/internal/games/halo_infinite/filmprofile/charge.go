package filmprofile

// charge.go — LA LECTURE DU CATALOGUE, ET SON REFUS.
//
// Un catalogue de profil a moitie lu vaut moins que pas de catalogue : il donnerait au decodeur
// des largeurs qu il croirait etablies. Toute anomalie est donc une ERREUR rendue, jamais une
// valeur par defaut ; et les champs inconnus sont REFUSES (`DisallowUnknownFields`), parce qu un
// champ mal orthographie serait une donnee saisie que personne ne lit.

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"
)

// formatDeDate : la date de provenance s ecrit `AAAA-MM-JJ`, comme partout dans le depot.
const formatDeDate = "2006-01-02"

// longueurSha256Hex : un sha256 rendu en hexadecimal fait 64 caracteres.
const longueurSha256Hex = 64

// Charger lit et valide le catalogue des profils de film au chemin donne.
//
// Le chemin vient de `title.PathResolver.FilmProfilesPath` — jamais d un `filepath.Join`
// ecrit a la main (D12).
func Charger(chemin string) (*Catalogue, error) {
	blob, err := os.ReadFile(chemin) //nolint:gosec // chemin resolu par le PathResolver
	if err != nil {
		return nil, fmt.Errorf("lecture du catalogue des profils de film (%s) : %w", chemin, err)
	}
	cat, err := Decoder(blob)
	if err != nil {
		return nil, fmt.Errorf("%s : %w", chemin, err)
	}
	return cat, nil
}

// Decoder analyse et valide un catalogue deja en memoire.
func Decoder(blob []byte) (*Catalogue, error) {
	var cat Catalogue
	dec := json.NewDecoder(bytes.NewReader(blob))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&cat); err != nil {
		return nil, fmt.Errorf("catalogue des profils de film illisible : %w", err)
	}
	if err := cat.Valide(); err != nil {
		return nil, err
	}
	return &cat, nil
}

// DecoderSansValider analyse un catalogue SANS le valider.
//
// RESERVE A LA CHAINE DE FABRICATION (`cmd/film-profiles-build`) : elle lit un fichier qu elle
// s apprete a CORRIGER, donc qui peut legitimement etre invalide — un bloc derive perime est
// precisement ce qu elle vient reecrire. Elle valide le resultat avant d ecrire. Tout autre
// lecteur passe par [Charger] ou [Decoder] : un catalogue non valide n a pas a circuler.
func DecoderSansValider(blob []byte) (*Catalogue, error) {
	var cat Catalogue
	dec := json.NewDecoder(bytes.NewReader(blob))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&cat); err != nil {
		return nil, fmt.Errorf("catalogue des profils de film illisible : %w", err)
	}
	return &cat, nil
}

// Valide rend TOUTES les anomalies du catalogue d un coup.
//
// D un coup, et pas a la premiere : celui qui saisit une ligne de profil veut la liste de ce
// qui cloche, pas un aller-retour par anomalie.
func (c *Catalogue) Valide() error {
	var anomalies []error
	if c.SchemaVersion != SchemaVersionCourante {
		anomalies = append(anomalies, fmt.Errorf("schemaVersion %d, attendu %d",
			c.SchemaVersion, SchemaVersionCourante))
	}
	if c.TitleSlug == "" {
		anomalies = append(anomalies, errors.New("titleSlug vide : un catalogue de reference "+
			"appartient a UN titre (isolation par titre, ADR 0008)"))
	}
	if c.APropos == "" {
		anomalies = append(anomalies, errors.New("about vide : le fichier doit dire ce qu il "+
			"est et ou se trouve sa procedure"))
	}
	anomalies = append(anomalies, c.Derive.valide()...)
	if len(c.Entrees) == 0 {
		anomalies = append(anomalies, errors.New("entries vide : un catalogue sans ligne de "+
			"profil ne dit rien"))
	}
	vues := map[string]int{}
	for i, e := range c.Entrees {
		anomalies = append(anomalies, e.valide(i)...)
		paire := e.Cle + "\x00" + e.Champ
		if precedent, deja := vues[paire]; deja {
			anomalies = append(anomalies, fmt.Errorf("entries[%d] : la paire (%q, %q) est deja "+
				"posee en entries[%d] — deux valeurs pour la meme clef et le meme champ, "+
				"aucune ne prime", i, e.Cle, e.Champ, precedent))
			continue
		}
		vues[paire] = i
	}
	return errors.Join(anomalies...)
}

// valide rend les anomalies d UNE entree.
func (e Entree) valide(i int) []error {
	var anomalies []error
	if _, err := ParseCle(e.Cle); err != nil {
		anomalies = append(anomalies, fmt.Errorf("entries[%d] : %w", i, err))
	}
	if e.Champ == "" {
		anomalies = append(anomalies, fmt.Errorf("entries[%d] (%q) : field vide", i, e.Cle))
	}
	if e.Valeur == "" {
		anomalies = append(anomalies, fmt.Errorf("entries[%d] (%q/%q) : value vide", i, e.Cle, e.Champ))
	}
	if !e.Source.EstConnue() {
		anomalies = append(anomalies, fmt.Errorf("entries[%d] (%q/%q) : provenance %q inconnue "+
			"— %q, %q ou %q, et rien d autre", i, e.Cle, e.Champ, e.Source,
			ProvenanceRelue, ProvenanceMesuree, ProvenancePresumee))
	}
	if e.Preuve == "" {
		anomalies = append(anomalies, fmt.Errorf("entries[%d] (%q/%q) : proof vide — une valeur "+
			"sans preuve ecrite n entre pas au catalogue (D3 du plan decodeur)", i, e.Cle, e.Champ))
	}
	if _, err := time.Parse(formatDeDate, e.Date); err != nil {
		anomalies = append(anomalies, fmt.Errorf("entries[%d] (%q/%q) : date %q n est pas une "+
			"date AAAA-MM-JJ", i, e.Cle, e.Champ, e.Date))
	}
	return anomalies
}

// valide rend les anomalies du bloc derive.
func (d Derive) valide() []error {
	var anomalies []error
	if d.Catalogue == "" {
		anomalies = append(anomalies, errors.New("derived.catalogue vide : l empreinte doit "+
			"nommer le fichier derive dont elle est solidaire"))
	}
	if d.SchemaVersion <= 0 {
		anomalies = append(anomalies, fmt.Errorf("derived.schemaVersion %d : attendu > 0", d.SchemaVersion))
	}
	if d.Cartes <= 0 {
		anomalies = append(anomalies, fmt.Errorf("derived.maps %d : un catalogue de bornes sans "+
			"carte n a pas ete produit", d.Cartes))
	}
	if len(d.EmpreinteCartes) != longueurSha256Hex {
		anomalies = append(anomalies, fmt.Errorf("derived.mapsSha256 %q : %d caracteres, attendu "+
			"%d (sha256 hexadecimal)", d.EmpreinteCartes, len(d.EmpreinteCartes), longueurSha256Hex))
	}
	if d.Outil == "" {
		anomalies = append(anomalies, errors.New("derived.tool vide : la part derivee doit nommer "+
			"la commande qui la produit (elle n est jamais saisie a la main)"))
	}
	return anomalies
}
