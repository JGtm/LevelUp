// Package duckdb — sentinelle_base_tenue_test.go : LE CONTRAT DE `ErrBaseTenueEnEcriture`
// (lot 2.10.4, découverte D3 (2.8)).
//
// Il remplace `TestMarqueurBaseTenueExisteChezLevelup` de `cmd/replay-corpus-gate`, qui lisait
// un fichier `package main` voisin pour y exiger un littéral. Ce que ce garde-fou protégeait —
// « le réessai du gate ne se désarme pas en silence » — est désormais tenu en deux points :
// ici, l'erreur est TYPÉE et se reconnaît par `errors.Is` ; et
// `archlint.TestMarqueurDuGateEgaleLaSentinelleBaseTenue` tient le miroir de texte dont le gate,
// qui vit dans un AUTRE processus, ne peut pas se passer.
package duckdb

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

// TestSentinelleBaseTenueEnveloppeUnVerrou — une erreur de verrou inter-processus devient
// reconnaissable par `errors.Is`, SANS perdre le message d'origine (c'est lui qui nomme le
// détenteur : `File is already open in <exe> (PID N)`).
func TestSentinelleBaseTenueEnveloppeUnVerrou(t *testing.T) {
	brut := errors.New("IO Error: File is already open in server.exe (PID 4242)")
	err := marqueBaseTenue(brut)
	if !errors.Is(err, ErrBaseTenueEnEcriture) {
		t.Fatalf("errors.Is(%v, ErrBaseTenueEnEcriture) = false — la sentinelle ne porte pas", err)
	}
	if !errors.Is(err, brut) {
		t.Errorf("le message d'origine a été perdu : %v", err)
	}
	if !strings.Contains(err.Error(), "PID 4242") {
		t.Errorf("le détenteur n'est plus nommé dans %q", err.Error())
	}
}

// TestSentinelleBaseTenueNeMarquePasCeQuiNEstPasUnVerrou — LE DÉFAUT QUE LE LOT CORRIGE : avant,
// l'indication « serveur en ecriture ? » était collée à la main à TOUTE erreur d'ouverture, y
// compris à un fichier absent — que le gate réessayait alors trois fois pour rien.
func TestSentinelleBaseTenueNeMarquePasCeQuiNEstPasUnVerrou(t *testing.T) {
	cas := map[string]error{
		"nil":                 nil,
		"fichier absent":      errors.New(`IO Error: Cannot open file "x.duckdb": No such file or directory`),
		"erreur de catalogue": errors.New("Catalog Error: Table with name foo does not exist"),
	}
	for nom, brut := range cas {
		t.Run(nom, func(t *testing.T) {
			err := marqueBaseTenue(brut)
			if errors.Is(err, ErrBaseTenueEnEcriture) {
				t.Errorf("%q a été marqué « base tenue » alors que ce n'est pas un verrou", nom)
			}
			if err != brut { //nolint:errorlint // on exige l'identité, pas l'équivalence
				t.Errorf("l'erreur a été enveloppée sans raison : %v", err)
			}
		})
	}
}

// TestSentinelleBaseTenueNEnveloppePasDeuxFois — un appelant qui repasse une erreur déjà marquée
// (retry, couche intermédiaire) ne l'empile pas.
func TestSentinelleBaseTenueNEnveloppePasDeuxFois(t *testing.T) {
	deja := fmt.Errorf("%w : %w", ErrBaseTenueEnEcriture,
		errors.New("IO Error: File is already open in server.exe (PID 7)"))
	if got := marqueBaseTenue(deja); got != deja { //nolint:errorlint // identité voulue
		t.Errorf("la sentinelle a été empilée : %v", got)
	}
}
