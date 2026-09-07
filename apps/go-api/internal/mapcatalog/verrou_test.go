package mapcatalog

// verrou_test.go — LE VERROU D'ECRITURE SUR UN DOSSIER QUI N'EXISTE PAS ENCORE (constat C3,
// revue A-R1).
//
// # Le defaut que ces tests ferment
//
// `prendreVerrou` posait `<overlay>.lock` en `O_CREATE|O_EXCL` SANS avoir cree le dossier. Sur
// un titre dont `data/titles/{slug}/reference/generated/` n'existe pas encore — l'etat NOMINAL
// de tout checkout neuf et de toute instance fraichement deployee, le dossier etant ignore par
// git — l'ouverture echouait ENOENT. La boucle ne distinguait pas ENOENT d'un verrou tenu :
// elle attendait DEUX SECONDES, journalisait un « verrou d'ecriture tenu trop longtemps » qui
// MENTAIT sur la cause, puis passait SANS exclusion mutuelle.
//
// Mesure du verdict : 8 ecrivains concurrents sur un dossier absent ne conservaient qu'UNE
// carte sur 8, plus quatre `rename` en echec dur. C'est precisement le trou que
// `TestAddOverlayEntryConcurrentNePerdPasDEntree` pretend fermer — sauf qu'il travaille sur un
// dossier deja cree.

import (
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"

	"levelup/go-api/internal/analysis/replay"
)

// TestPrendreVerrouCreeLeDossierEtNAttendPas — le premier rattrapage d'un titre ne doit ni
// attendre ni mentir : le dossier se cree, le verrou se pose VRAIMENT, et le tout se joue en
// quelques millisecondes.
func TestPrendreVerrouCreeLeDossierEtNAttendPas(t *testing.T) {
	overlay := filepath.Join(t.TempDir(), "generated", "map_weapon_pads.json")

	debut := time.Now()
	retirer := prendreVerrou(overlay)
	ecoule := time.Since(debut)

	// LE VERROU EXISTE : c'est la difference entre « pose » et « passage force ». Un passage
	// force rend une fonction de retrait vide et ne cree aucun fichier.
	if _, err := os.Stat(overlay + ".lock"); err != nil {
		t.Fatalf("aucun verrou pose sur un dossier absent (%v) — l'ecriture se ferait SANS "+
			"exclusion mutuelle, et 7 cartes sur 8 se perdraient en concurrence", err)
	}
	if ecoule > dureeAttenteVerrou/4 {
		t.Errorf("verrou pris en %v — l'attente bornee (%v) a ete consommee pour rien : "+
			"ENOENT n'est pas un verrou tenu", ecoule, dureeAttenteVerrou)
	}
	retirer()
	if _, err := os.Stat(overlay + ".lock"); err == nil {
		t.Error("le verrou n'a pas ete retire")
	}
}

// TestAddOverlayEntryConcurrentDossierAbsentNePerdRien — LA mesure du verdict, rejouee : huit
// rattrapages simultanes sur un titre dont le dossier `generated/` n'existe pas encore.
func TestAddOverlayEntryConcurrentDossierAbsentNePerdRien(t *testing.T) {
	overlay := filepath.Join(t.TempDir(), "generated", "map_weapon_pads.json")
	const n = 8
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()
			id := "carte" + strconv.Itoa(i)
			if err := AddOverlayEntry(overlay, "halo_infinite", id, mcEntree(id)); err != nil {
				t.Errorf("AddOverlayEntry(%s) = %v", id, err)
			}
		}(i)
	}
	wg.Wait()

	cat, err := replay.LoadMapWeaponPads(overlay)
	if err != nil {
		t.Fatalf("overlay illisible apres ecritures concurrentes : %v", err)
	}
	if len(cat.Maps) != n {
		t.Errorf("%d carte(s) conservee(s) sur %d — le premier rattrapage d'un titre ecrivait "+
			"sans verrou (constat C3)", len(cat.Maps), n)
	}
	for i := 0; i < n; i++ {
		if _, ok := cat.Maps["carte"+strconv.Itoa(i)]; !ok {
			t.Errorf("carte%d perdue", i)
		}
	}
	if _, err := os.Stat(overlay + ".lock"); err == nil {
		t.Error("le verrou n'a pas ete retire")
	}
}
