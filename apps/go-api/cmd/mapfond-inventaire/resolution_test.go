package main

// resolution_test.go — L INVENTAIRE DOIT REPONDRE A LA QUESTION QUE LE PRODUIT SE POSE.
//
// La seule raison d etre de cette commande est de dire quelles cartes n ont pas de fond A L
// ECRAN. Elle ne le fait qu a une condition : resoudre EXACTEMENT comme
// `replayService.resolveBackgroundKeyDepuis` (la cascade partagee par les deux enveloppes,
// par match et par carte). Le jour ou la production gagne un chemin de resolution et
// pas l inventaire, celui-ci se met a compter des manques qui n existent pas — c est deja arrive
// deux fois a la main, et les deux fois la conclusion a ete fausse
// (`.ai/V7.5/cartes/HANDOFF_FONDS_CARTE_2026-09-03.md`, sections 2.2 et 7).
//
// DEUX FAMILLES DE TESTS, ET IL FAUT LES DEUX :
//
//  1. le COMPORTEMENT de `resoutFond` sur un repertoire fabrique — les trois voies ;
//  2. un garde-rail de SOURCE sur le fichier de production, qui echoue si un chemin de
//     resolution y est ajoute. Le comportement seul ne verrait rien : un troisieme chemin cote
//     service laisserait ces tests verts et l inventaire faux.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/testutil"
)

// poseSidecar ecrit un sidecar de fond minimal mais VALIDE (le schema est verifie a la lecture).
func poseSidecar(t *testing.T, dir, cle string, noms ...string) {
	t.Helper()
	bg := replay.MapBackground{
		SchemaVersion: replay.MapBackgroundSchemaVersion,
		Module:        cle,
		MapNames:      noms,
		Image:         cle + ".png",
		Source:        "test",
		Style:         "encre",
		Calibration: replay.MapBackgroundCalibration{
			MetersPerPixel: 0.05, OriginX: -10, OriginY: 10, WidthPx: 100, HeightPx: 100,
			Convention: "xMonde = originX + (px + 0.5) * metersPerPixel",
		},
	}
	blob, err := json.Marshal(bg)
	if err != nil {
		t.Fatalf("serialisation du sidecar %s : %v", cle, err)
	}
	if err := os.WriteFile(filepath.Join(dir, cle+".json"), blob, 0o600); err != nil {
		t.Fatalf("ecriture du sidecar %s : %v", cle, err)
	}
}

func indexDe(t *testing.T, dir string) *replay.MapBackgroundIndex {
	t.Helper()
	idx, err := replay.BuildMapBackgroundIndex(dir)
	if err != nil {
		t.Fatalf("index des fonds : %v", err)
	}
	return idx
}

// TestResoutFondLesTroisVoies couvre les trois chemins que la production emprunte, plus l absence.
func TestResoutFondLesTroisVoies(t *testing.T) {
	dir := t.TempDir()
	poseSidecar(t, dir, "aa11-forge")                       // carte Forge, clé map_id
	poseSidecar(t, dir, "sgh_streets", "Streets")           // carte native, clé module
	poseSidecar(t, dir, "empyrean_key", "Empyrean")         // base d'une variante
	poseSidecar(t, dir, "insolence_key", "Insolence")       // base ET variante publiées
	poseSidecar(t, dir, "insolence_h", "Insolence Heavies") // la variante a SON fond
	idx := indexDe(t, dir)

	cas := []struct {
		nom      string
		carte    carteJouee
		veutCle  string
		veutVoie voieResolution
	}{
		{
			nom:   "map_id publie : la clé map_id gagne, et AVANT l index",
			carte: carteJouee{MapID: "aa11-forge", NomBrut: "Streets"},
			// « Streets » resoudrait vers sgh_streets par l index : si le map_id ne primait pas,
			// le test servirait le fond d une AUTRE carte.
			veutCle: "aa11-forge", veutVoie: voieMapID,
		},
		{
			nom:     "nom du registre : l index rattrape une carte native",
			carte:   carteJouee{MapID: "asset-mort", NomBrut: "Streets"},
			veutCle: "sgh_streets", veutVoie: voieIndex,
		},
		{
			nom:     "nom d asset prioritaire sur le libelle brut",
			carte:   carteJouee{NomAsset: "Streets", NomBrut: "libelle-inconnu"},
			veutCle: "sgh_streets", veutVoie: voieIndex,
		},
		{
			nom:     "heritage : une variante sans fond prend celui de sa base",
			carte:   carteJouee{NomBrut: "Empyrean - Ranked"},
			veutCle: "empyrean_key", veutVoie: voieIndex,
		},
		{
			nom: "la variante qui a SON fond garde le sien",
			// L heritage ne doit s essayer qu APRES l identite exacte, sinon Insolence Heavies
			// se ferait servir le fond d Insolence.
			carte:   carteJouee{NomBrut: "Insolence Heavies"},
			veutCle: "insolence_h", veutVoie: voieIndex,
		},
		{
			nom:     "carte inconnue : aucun fond, et surtout pas celui du voisin",
			carte:   carteJouee{MapID: "jamais-vu", NomBrut: "Carte Inconnue"},
			veutCle: "", veutVoie: voieAucune,
		},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			cle, voie := resoutFond(c.carte, dir, idx)
			if cle != c.veutCle || voie != c.veutVoie {
				t.Errorf("resoutFond = (%q, %s), attendu (%q, %s)", cle, voie, c.veutCle, c.veutVoie)
			}
		})
	}
}

// TestResoutFondSidecarSansPngResoutQuandMeme : la PRESENCE DU SIDECAR décide, comme dans la
// production (`os.Stat` sur `MapBackgroundMetaPath`, jamais sur le PNG). Un inventaire qui
// exigerait l image compterait comme manquantes des cartes que le rejeu sert.
func TestResoutFondSidecarSansPngResoutQuandMeme(t *testing.T) {
	dir := t.TempDir()
	poseSidecar(t, dir, "bb22-forge")
	if cle, voie := resoutFond(carteJouee{MapID: "bb22-forge"}, dir, indexDe(t, dir)); cle != "bb22-forge" || voie != voieMapID {
		t.Fatalf("resoutFond = (%q, %s) sans PNG, attendu (bb22-forge, %s)", cle, voie, voieMapID)
	}
}

// cheminServiceProduction est le fichier dont l inventaire rejoue la logique.
const cheminServiceProduction = "apps/go-api/internal/service/replay_map_background.go"

// cheminsDeResolutionAttendus : le nombre de `return` de `resolveBackgroundKeyDepuis` qui
// rendent une CLÉ (par opposition aux retours d'erreur, qui rendent la chaîne vide). La cascade
// vit dans `resolveBackgroundKeyDepuis`, appelée par les deux enveloppes `resolveBackgroundKey`
// (par match) et `resolveBackgroundKeyForMap` (par carte) : lire l'une des enveloppes ne verrait
// qu'un relais vide. Il en vaut DEUX : la clé map_id, puis la clé rendue par l'index — l'héritage
// variante vers base vit DANS `Lookup`, pas dans un troisième retour.
const cheminsDeResolutionAttendus = 2

// retourAvecCle capte un `return <identifiant>, nil` — donc un retour qui SERT une clé. Les
// retours d'erreur s'écrivent `return "", ...` et ne matchent pas.
var retourAvecCle = regexp.MustCompile(`(?m)^\s*return\s+[a-zA-Z_][\w.]*\s*,\s*nil\s*$`)

// TestInventaireEtProductionResolventPareil — LE GARDE-RAIL QUI COMPTE.
//
// Il ne compare pas deux exécutions (l'inventaire appelle DÉJÀ le même index, une comparaison
// serait circulaire) : il compte les chemins de résolution du fichier de production. Un troisième
// chemin ajouté là-bas fait tomber ce test, et son message dit quoi faire — parce que sans lui
// l'inventaire continuerait de tourner en rendant des manques qui n'existent pas.
func TestInventaireEtProductionResolventPareil(t *testing.T) {
	racine, err := testutil.RepoRoot()
	if err != nil {
		t.Fatalf("racine du depot introuvable : %v", err)
	}
	blob, err := os.ReadFile(filepath.Join(racine, filepath.FromSlash(cheminServiceProduction)))
	if err != nil {
		t.Fatalf("service de fond de carte illisible : %v", err)
	}
	corps, err := corpsDeFonction(string(blob), "func (s *replayService) resolveBackgroundKeyDepuis(")
	if err != nil {
		t.Fatalf("%v — la signature a changé, l'inventaire doit être relu", err)
	}

	if n := len(retourAvecCle.FindAllString(corps, -1)); n != cheminsDeResolutionAttendus {
		t.Errorf("resolveBackgroundKeyDepuis rend une clé par %d chemin(s), attendu %d.\n"+
			"La production a gagné (ou perdu) un chemin de résolution. `cmd/mapfond-inventaire`\n"+
			"le rejoue dans `resoutFond` : le mettre à jour DANS LE MÊME COMMIT, sinon l'inventaire\n"+
			"comptera des cartes « sans fond » qui sont servies à l'écran.", n, cheminsDeResolutionAttendus)
	}
	// L ORDRE EST LE FOND DU SUJET : la clé map_id d abord, l index ensuite. Inversé, une carte
	// Forge encore publiée sous son asset se ferait servir le fond trouvé par son nom.
	iMapID := strings.Index(corps, "MapBackgroundMetaPath")
	iIndex := strings.Index(corps, "MapBackgroundIndexFor")
	if iMapID < 0 || iIndex < 0 {
		t.Fatalf("resolveBackgroundKeyDepuis n'appelle plus MapBackgroundMetaPath (%d) et/ou "+
			"MapBackgroundIndexFor (%d) — `resoutFond` repose sur les deux", iMapID, iIndex)
	}
	if iMapID > iIndex {
		t.Error("l'essai par map_id ne passe plus AVANT l'index des noms — `resoutFond` suppose cet ordre")
	}
}

// corpsDeFonction extrait le corps d une fonction a partir de sa signature, par comptage
// d accolades. Suffisant ici : le fichier vise ne contient ni chaine ni commentaire portant une
// accolade non appariee, et le test echoue bruyamment si la signature bouge.
func corpsDeFonction(src, signature string) (string, error) {
	i := strings.Index(src, signature)
	if i < 0 {
		return "", fmt.Errorf("signature introuvable dans %s : %q", cheminServiceProduction, signature)
	}
	reste := src[i:]
	debut := strings.Index(reste, "{")
	if debut < 0 {
		return "", fmt.Errorf("corps introuvable pour %q", signature)
	}
	profondeur := 0
	for j := debut; j < len(reste); j++ {
		switch reste[j] {
		case '{':
			profondeur++
		case '}':
			profondeur--
			if profondeur == 0 {
				return reste[debut : j+1], nil
			}
		}
	}
	return "", fmt.Errorf("accolades non appariees pour %q", signature)
}
