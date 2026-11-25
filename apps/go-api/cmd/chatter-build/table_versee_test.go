package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// dossierVerseDuDepot : `static/sounds/halo_infinite/repliques/` du dépôt, vu depuis ce paquet.
func dossierVerseDuDepot() string {
	return filepath.Join("..", "..", "..", "..", filepath.FromSlash(dossierVerse))
}

// plafondVoixVersees : le poids des voix versées ne dépasse pas 10 Mo. Au-delà, le choix « une variante par
// son » doit être revu avec le propriétaire (le dépôt est public et garde son historique).
const plafondVoixVersees = 10 * 1000 * 1000

func tableVersee(t *testing.T) (Table, []byte) {
	t.Helper()
	brut, err := os.ReadFile(filepath.Join(dossierVerseDuDepot(), nomTable))
	if err != nil {
		t.Fatalf("table versée : %v", err)
	}
	var tab Table
	if err := json.Unmarshal(brut, &tab); err != nil {
		t.Fatalf("table versée illisible : %v", err)
	}
	return tab, brut
}

// TestTableVerseeRespecteSesInvariants : le fichier versionné se relit hors ligne, sans le jeu. Clés uniques,
// un texte dans chaque langue requise de `config/locales.toml`, des voix annoncées qui existent et des voix
// versées qui sont annoncées.
func TestTableVerseeRespecteSesInvariants(t *testing.T) {
	langues := languesDeTest(t)
	tab, brut := tableVersee(t)
	if err := validerTable(tab, langues); err != nil {
		t.Fatal(err)
	}
	if d, err := clesEnDouble(brut); err != nil || len(d) != 0 {
		t.Errorf("clés en double : %v (%v)", d, err)
	}
	if err := verifierVoix(tab, langues, dossierVerseDuDepot()); err != nil {
		t.Error(err)
	}
}

// opusCanaux48 lit l'en-tête OpusHead d'un fichier Ogg : canaux et cadence d'origine.
func opusCanaux48(t *testing.T, chemin string) (canaux byte, cadence uint32) {
	t.Helper()
	b, err := os.ReadFile(chemin)
	if err != nil {
		t.Fatal(err)
	}
	i := bytes.Index(b, []byte("OpusHead"))
	if !bytes.HasPrefix(b, []byte("OggS")) || i < 0 || i+16 > len(b) {
		t.Fatalf("%s : pas un Ogg Opus", chemin)
	}
	return b[i+9], binary.LittleEndian.Uint32(b[i+12:])
}

// TestVoixVerseesSontDeLOpusStereoSousLePlafond : même format que les autres sons du rejeu (Opus en conteneur
// Ogg, deux canaux, 48 kHz d'origine) et poids total sous le plafond. Sans voix versée, le test ne regarde rien.
func TestVoixVerseesSontDeLOpusStereoSousLePlafond(t *testing.T) {
	var total int64
	for _, l := range languesDeTest(t) {
		ents, err := os.ReadDir(filepath.Join(dossierVerseDuDepot(), l.Code))
		if err != nil {
			continue // une langue sans voix versée n'a pas de dossier
		}
		for _, e := range ents {
			chemin := filepath.Join(dossierVerseDuDepot(), l.Code, e.Name())
			info, err := e.Info()
			if err != nil {
				t.Fatal(err)
			}
			total += info.Size()
			if canaux, cadence := opusCanaux48(t, chemin); canaux != 2 || cadence != 48000 {
				t.Errorf("%s : %d canaux, cadence %d (stéréo 48 kHz attendus)", e.Name(), canaux, cadence)
			}
		}
	}
	if total > plafondVoixVersees {
		t.Errorf("voix versées : %d octets (plafond %d)", total, plafondVoixVersees)
	}
}
