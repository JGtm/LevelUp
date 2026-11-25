package main

// rejeux.go — LA LISTE DES SONS QUE LES FILMS ONT JOUES.

import (
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// Pourquoi une liste : le jeu contient 1 950 sons de voix de Spartan (73 Mo) dont moins du tiers est joué
// dans un match ; la table ne porte que ceux que les films jouent. Elle se lit dans les documents de rejeu
// stockés (champ `chatter[].sound`), sans les décoder en entier.
//
// Les documents sont des JSON, compressés ou non. Un document illisible est compté, journalisé et écarté : un
// parc dont quelques fichiers sont abîmés donne encore une liste utile.

// documentRepliques : la seule part d'un document de rejeu que cette étape lit.
type documentRepliques struct {
	Chatter []struct {
		Sound string `json:"sound"`
	} `json:"chatter"`
}

// lireSonsJoues parcourt un dossier de documents de rejeu (`*.json`, `*.json.gz`) et compte les sons.
func lireSonsJoues(dossier string, log *slog.Logger) (SonsJoues, error) {
	res := SonsJoues{Sons: map[string]int{}}
	illisibles := 0
	err := filepath.WalkDir(dossier, func(chemin string, d fs.DirEntry, errParcours error) error {
		if errParcours != nil {
			return errParcours
		}
		nom := strings.ToLower(d.Name())
		if d.IsDir() || !(strings.HasSuffix(nom, ".json") || strings.HasSuffix(nom, ".json.gz")) {
			return nil
		}
		doc, err := lireDocumentRepliques(chemin)
		if err != nil {
			illisibles++
			log.Error("chatter-build: document de rejeu illisible", "fichier", filepath.Base(chemin), "err", err)
			return nil
		}
		res.Films++
		for _, c := range doc.Chatter {
			if c.Sound != "" {
				res.Sons[c.Sound]++
			}
		}
		return nil
	})
	if err != nil {
		return res, fmt.Errorf("parcours de %s : %w", dossier, err)
	}
	if res.Films == 0 {
		return res, fmt.Errorf("aucun document de rejeu lisible dans %s (%d illisibles)", dossier, illisibles)
	}
	return res, nil
}

// lireDocumentRepliques lit un document, décompressé à la volée s'il porte l'extension `.gz`.
func lireDocumentRepliques(chemin string) (documentRepliques, error) {
	var doc documentRepliques
	f, err := os.Open(chemin)
	if err != nil {
		return doc, err
	}
	defer func() { _ = f.Close() }()
	var r io.Reader = f
	if strings.HasSuffix(strings.ToLower(chemin), ".gz") {
		z, err := gzip.NewReader(f)
		if err != nil {
			return doc, err
		}
		defer func() { _ = z.Close() }()
		r = z
	}
	if err := json.NewDecoder(r).Decode(&doc); err != nil && !errors.Is(err, io.EOF) {
		return doc, err
	}
	return doc, nil
}
