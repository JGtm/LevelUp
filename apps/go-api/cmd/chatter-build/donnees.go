package main

// donnees.go — LES FICHIERS QUE LES ETAPES SE PASSENT.
//
// La chaîne tient en plusieurs processus parce que les modules du jeu pèsent plusieurs gigaoctets (celui des
// tags `snd!` 0,6 Go, celui des banques 7 Go) et ne se chargent jamais ensemble. Chaque étape lit le fichier
// de la précédente et écrit le sien, en JSON : `rejeux` -> `textes` -> `banques` -> `choix` -> `medias` ->
// `voix` -> (sounds-encode) -> `livrer`.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// SonsJoues : les sons de réplique que les films ont joués, avec le nombre de fois. Identifiant en
// hexadécimal sur huit chiffres, comme `chatter[].sound` du document de rejeu.
type SonsJoues struct {
	Films int            `json:"films"`
	Sons  map[string]int `json:"sons"`
}

// VarianteTexte : une variante d'un son, c'est-à-dire un évènement Wwise et la clé de texte que le tag `snd!`
// lui associe, avec le texte dans chaque langue de l'application.
type VarianteTexte struct {
	Evenement string            `json:"evenement"`
	Cle       string            `json:"cle"`
	Textes    map[string]string `json:"textes"`
}

// SonTextes : un tag `snd!` lu dans le module des tags.
type SonTextes struct {
	Son       string          `json:"son"`
	Banque    string          `json:"banque"`
	Liste     string          `json:"liste"`
	Variantes []VarianteTexte `json:"variantes"`
}

// MediaVoix : un média de voix d'une langue. Paquet est le fichier `.pck` qui le contient ; Embarque dit qu'il
// vit dans la banque elle-même (médias courts, identiques dans toutes les langues).
type MediaVoix struct {
	ID       string `json:"id"`
	Paquet   string `json:"paquet,omitempty"`
	Embarque bool   `json:"embarque,omitempty"`
}

// EvenementBanque : les médias qu'un évènement peut jouer, par langue de l'application.
type EvenementBanque struct {
	Evenement string                 `json:"evenement"`
	Langues   map[string][]MediaVoix `json:"langues"`
}

// SonBanque : ce que la banque d'un son dit de ses évènements.
type SonBanque struct {
	Son        string            `json:"son"`
	Absente    bool              `json:"absente,omitempty"`
	Evenements []EvenementBanque `json:"evenements"`
}

// Choix : la variante retenue pour un son parlé. Voix donne, par langue, le média à extraire.
type Choix struct {
	Son       string               `json:"son"`
	Evenement string               `json:"evenement"`
	Cle       string               `json:"cle"`
	Textes    map[string]string    `json:"textes"`
	Voix      map[string]MediaVoix `json:"voix,omitempty"`
	Banque    string               `json:"banque,omitempty"`
}

// ecrireJSON écrit v en JSON indenté, dossier parent créé.
func ecrireJSON(chemin string, v any) error {
	b, err := json.MarshalIndent(v, "", " ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(chemin), 0o755); err != nil {
		return err
	}
	return os.WriteFile(chemin, append(b, '\n'), 0o644)
}

// lireJSON lit un fichier JSON dans v.
func lireJSON(chemin string, v any) error {
	b, err := os.ReadFile(chemin)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(b, v); err != nil {
		return fmt.Errorf("%s : %w", chemin, err)
	}
	return nil
}
