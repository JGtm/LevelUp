package main

// livrer.go — LE FICHIER VERSE : `repliques.json` et les voix encodées à côté.
//
// Disposition (sous `static/sounds/halo_infinite/repliques/`, servie comme les sons d'armes) :
//
//	repliques.json            la table : identifiant du son -> textes par langue, voix disponibles
//	<langue>/<son>.ogg        la voix de la variante retenue, Opus en conteneur Ogg, une par langue
//
// La voix d'une langue n'est annoncée dans `audio` que si son fichier est versé : le client n'affiche une icône
// de voix que pour ce qui existe. Recevoir les voix d'une langue, c'est déposer ses `.ogg` et relancer `livrer`.

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// versionTable : la forme du fichier versé. Elle monte quand un lecteur doit changer.
const versionTable = 1

// nomTable : le nom du fichier de la table dans le dossier des répliques.
const nomTable = "repliques.json"

// Table est le fichier versé.
type Table struct {
	Version int                    `json:"version"`
	Sons    map[string]EntreeTable `json:"sons"`
}

// EntreeTable : une réplique parlée. Texte par code de langue de l'application ; Audio, les codes de langue dont
// la voix est versée (ordre de `config/locales.toml`).
type EntreeTable struct {
	Texte map[string]string `json:"texte"`
	Audio []string          `json:"audio"`
}

// sourcesVoix : langue -> dossier des `.ogg` encodés, nommés `<identifiant de média>.ogg`.
type sourcesVoix map[string]string

// livrer écrit la table et copie les voix retenues. Les `.ogg` du dossier de livraison qui ne servent plus
// sont retirés : un fichier qu'aucune entrée n'annonce serait du poids mort.
func livrer(choix []Choix, voix sourcesVoix, langues []langueApp, dossier string, log *slog.Logger) (Table, error) {
	t := Table{Version: versionTable, Sons: map[string]EntreeTable{}}
	gardes := map[string]map[string]bool{}
	for _, l := range langues {
		gardes[l.Code] = map[string]bool{}
	}
	for _, c := range choix {
		e := EntreeTable{Texte: c.Textes, Audio: []string{}}
		for _, l := range langues {
			src, ok := voix[l.Code]
			m, aUneVoix := c.Voix[l.Code]
			if !ok || !aUneVoix {
				continue
			}
			dst := filepath.Join(dossier, l.Code, c.Son+".ogg")
			if err := copierFichier(filepath.Join(src, m.ID+".ogg"), dst); err != nil {
				if errors.Is(err, fs.ErrNotExist) {
					log.Warn("chatter-build: voix absente, la réplique reste sans voix dans cette langue",
						"son", c.Son, "langue", l.Code, "media", m.ID)
					continue
				}
				return t, err
			}
			gardes[l.Code][c.Son+".ogg"] = true
			e.Audio = append(e.Audio, l.Code)
		}
		t.Sons[c.Son] = e
	}
	if err := validerTable(t, langues); err != nil {
		return t, err
	}
	for _, l := range langues {
		if err := retirerOrphelins(filepath.Join(dossier, l.Code), gardes[l.Code], log); err != nil {
			return t, err
		}
	}
	return t, ecrireTable(filepath.Join(dossier, nomTable), t)
}

// copierFichier copie src vers dst, dossier créé. Un src absent rend une erreur `fs.ErrNotExist`.
func copierFichier(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

// retirerOrphelins supprime d'un dossier de langue les `.ogg` que la livraison n'annonce pas.
func retirerOrphelins(dossier string, gardes map[string]bool, log *slog.Logger) error {
	ents, err := os.ReadDir(dossier)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, e := range ents {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".ogg") || gardes[e.Name()] {
			continue
		}
		log.Info("chatter-build: voix retirée, plus annoncée", "fichier", e.Name())
		if err := os.Remove(filepath.Join(dossier, e.Name())); err != nil {
			return err
		}
	}
	return nil
}

// validerTable : les invariants du fichier versé. Il sert à `livrer` avant d'écrire et au test qui relit le
// fichier versionné.
//
//   - un texte non vide dans chaque langue requise de `config/locales.toml` ;
//   - un code de langue d'`audio` qui est une langue de l'application, sans doublon ;
//   - une version connue.
//
// Les fichiers de voix eux-mêmes se vérifient contre le dossier (`verifierVoix`).
func validerTable(t Table, langues []langueApp) error {
	if t.Version != versionTable {
		return fmt.Errorf("version %d, %d attendue", t.Version, versionTable)
	}
	if len(t.Sons) == 0 {
		return errors.New("table vide")
	}
	connues := map[string]bool{}
	for _, l := range langues {
		connues[l.Code] = true
	}
	ids := make([]string, 0, len(t.Sons))
	for id := range t.Sons {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		if _, ok := parseHex(id); !ok || len(id) != 8 {
			return fmt.Errorf("son %q : identifiant hexadécimal sur huit chiffres attendu", id)
		}
		e := t.Sons[id]
		if !texteComplet(e.Texte, langues) {
			return fmt.Errorf("son %s : texte manquant dans une langue requise", id)
		}
		vus := map[string]bool{}
		for _, a := range e.Audio {
			if !connues[a] || vus[a] {
				return fmt.Errorf("son %s : voix %q inconnue ou en double", id, a)
			}
			vus[a] = true
		}
	}
	return nil
}

// verifierVoix : chaque voix annoncée est un fichier du dossier, et chaque fichier du dossier est annoncé.
func verifierVoix(t Table, langues []langueApp, dossier string) error {
	for _, l := range langues {
		annonces := map[string]bool{}
		for id, e := range t.Sons {
			for _, a := range e.Audio {
				if a == l.Code {
					annonces[id+".ogg"] = true
				}
			}
		}
		for nom := range annonces {
			if _, err := os.Stat(filepath.Join(dossier, l.Code, nom)); err != nil {
				return fmt.Errorf("voix annoncée absente : %s/%s", l.Code, nom)
			}
		}
		ents, err := os.ReadDir(filepath.Join(dossier, l.Code))
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}
		for _, e := range ents {
			if strings.HasSuffix(e.Name(), ".ogg") && !annonces[e.Name()] {
				return fmt.Errorf("voix versée mais non annoncée : %s/%s", l.Code, e.Name())
			}
		}
	}
	return nil
}

// ecrireTable écrit la table, une réplique par ligne (diff lisible d'une extraction à l'autre).
func ecrireTable(chemin string, t Table) error {
	ids := make([]string, 0, len(t.Sons))
	for id := range t.Sons {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	var b bytes.Buffer
	fmt.Fprintf(&b, "{\"version\":%d,\"sons\":{\n", t.Version)
	for i, id := range ids {
		ligne, err := json.Marshal(t.Sons[id])
		if err != nil {
			return err
		}
		sep := ","
		if i == len(ids)-1 {
			sep = ""
		}
		fmt.Fprintf(&b, "%q:%s%s\n", id, ligne, sep)
	}
	b.WriteString("}}\n")
	if err := os.MkdirAll(filepath.Dir(chemin), 0o755); err != nil {
		return err
	}
	return os.WriteFile(chemin, b.Bytes(), 0o644)
}

// clesEnDouble rend les clés répétées de l'objet `sons` d'un fichier de table : un décodage en `map` garde la
// dernière et ne dit rien.
func clesEnDouble(brut []byte) ([]string, error) {
	dec := json.NewDecoder(bytes.NewReader(brut))
	if _, err := dec.Token(); err != nil { // {
		return nil, err
	}
	var doubles []string
	for dec.More() {
		cle, err := dec.Token()
		if err != nil {
			return nil, err
		}
		if cle != "sons" {
			var ignore json.RawMessage
			if err := dec.Decode(&ignore); err != nil {
				return nil, err
			}
			continue
		}
		if _, err := dec.Token(); err != nil { // {
			return nil, err
		}
		vus := map[string]bool{}
		for dec.More() {
			id, err := dec.Token()
			if err != nil {
				return nil, err
			}
			nom, _ := id.(string)
			if vus[nom] {
				doubles = append(doubles, nom)
			}
			vus[nom] = true
			var ignore json.RawMessage
			if err := dec.Decode(&ignore); err != nil {
				return nil, err
			}
		}
		if _, err := dec.Token(); err != nil { // }
			return nil, err
		}
	}
	return doubles, nil
}
