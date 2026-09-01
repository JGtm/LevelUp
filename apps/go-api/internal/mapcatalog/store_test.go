package mapcatalog

// store_test.go — L'AJOUT-SEUL ET L'ECRITURE ATOMIQUE, testes.
//
// Le contrat du store tient en quatre promesses, et deux n'etaient couvertes par rien :
// le REFUS d'une cle existante (supprimer les trois lignes du garde ne rougissait nulle part)
// et l'echec d'ECRITURE.

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/analysis/replay/mapvar"
)

func mcEntree(id string) replay.MapWeaponPadsEntry {
	sp := []replay.MapSpawnPointSpot{
		{Pos: mapvar.Vec3{X: 1}, TypeID: "0xADEEE6D8", Kind: "grenade", Objects: 1},
	}
	return replay.MapWeaponPadsEntry{
		MapID: id, MvarFile: id + ".mvar", ObjectsN: 400, LevelID: 7,
		Pads:        []replay.MapWeaponPadSpot{mcSocle(1, "0x5F379533", "power")},
		SpawnPoints: &sp,
	}
}

func mcCatalogue(t *testing.T, dir string, ids ...string) string {
	t.Helper()
	cat := &replay.MapWeaponPadsCatalog{
		SchemaVersion: replay.MapWeaponPadsSchemaVersion, TitleSlug: "halo_infinite",
		Maps: map[string]replay.MapWeaponPadsEntry{},
	}
	for _, id := range ids {
		cat.Maps[id] = mcEntree(id)
	}
	chemin := filepath.Join(dir, "map_weapon_pads.json")
	b, err := json.MarshalIndent(cat, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(chemin, b, 0o600); err != nil {
		t.Fatal(err)
	}
	return chemin
}

// TestAddEntryRefuseUneCleExistante — LA promesse de l'ajout-seul.
//
// Supprimer le garde de `AddEntry` fait tomber ce test : sans lui, un rattrapage automatique
// pourrait REECRIRE une entree que des chemins livres consomment.
func TestAddEntryRefuseUneCleExistante(t *testing.T) {
	chemin := mcCatalogue(t, t.TempDir(), "deja")
	avant, err := os.ReadFile(chemin)
	if err != nil {
		t.Fatal(err)
	}
	// Une entree DIFFERENTE, pour que « rien n'a change » soit une vraie information.
	neuve := mcEntree("deja")
	neuve.ObjectsN = 999
	neuve.Pads = []replay.MapWeaponPadSpot{mcSocle(42, "0x6253CFC0", "rack")}

	err = AddEntry(chemin, "deja", neuve)
	if !errors.Is(err, ErrEntryExists) {
		t.Fatalf("AddEntry sur une cle existante = %v, attendu ErrEntryExists", err)
	}
	apres, err := os.ReadFile(chemin)
	if err != nil {
		t.Fatal(err)
	}
	if string(avant) != string(apres) {
		t.Error("le catalogue a ete REECRIT alors que la cle existait deja — c'est exactement " +
			"ce que l'ajout-seul doit rendre impossible")
	}
}

// TestAddEntryAjouteUneCleNeuve — le pendant : la fonction fait bien son travail.
func TestAddEntryAjouteUneCleNeuve(t *testing.T) {
	chemin := mcCatalogue(t, t.TempDir(), "deja")
	if err := AddEntry(chemin, "neuve", mcEntree("neuve")); err != nil {
		t.Fatalf("AddEntry sur une cle neuve = %v, attendu nil", err)
	}
	cat, err := replay.LoadMapWeaponPads(chemin)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := cat.Maps["neuve"]; !ok {
		t.Error("la cle neuve n'a pas ete ajoutee")
	}
	// L'EXISTANTE EST BYTE-IDENTIQUE.
	a, _ := json.Marshal(mcEntree("deja"))
	z, _ := json.Marshal(cat.Maps["deja"])
	if string(a) != string(z) {
		t.Errorf("l'entree existante a change :\navant %s\napres %s", a, z)
	}
}

// TestAddEntryEchoueSurCatalogueIllisible — la 4e forme du contrat, jamais couverte.
func TestAddEntryEchoueSurCatalogueIllisible(t *testing.T) {
	dir := t.TempDir()
	// Catalogue ABSENT : `AddEntry` doit rendre une erreur, pas creer un catalogue de rien.
	absent := filepath.Join(dir, "map_weapon_pads.json")
	if err := AddEntry(absent, "x", mcEntree("x")); err == nil {
		t.Error("AddEntry sur un catalogue absent doit ECHOUER — en creer un de zero effacerait " +
			"silencieusement toutes les cartes du titre")
	}
	if _, err := os.Stat(absent); err == nil {
		t.Error("aucun fichier ne doit etre cree quand le catalogue de depart est illisible")
	}
	// Catalogue CORROMPU.
	corrompu := filepath.Join(dir, "corrompu.json")
	if err := os.WriteFile(corrompu, []byte("{ pas du json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := AddEntry(corrompu, "x", mcEntree("x")); err == nil {
		t.Error("AddEntry sur un catalogue corrompu doit ECHOUER")
	}
}

// TestWriteAtomicEchoueSansLaisserDeTemporaire — P2-2, le nettoyage.
func TestWriteAtomicEchoueSansLaisserDeTemporaire(t *testing.T) {
	dir := t.TempDir()
	// La CIBLE est un REPERTOIRE : le `rename` echouera a coup sur, quel que soit l'OS.
	cible := filepath.Join(dir, "cible")
	if err := os.Mkdir(cible, 0o750); err != nil {
		t.Fatal(err)
	}
	cat := &replay.MapWeaponPadsCatalog{
		SchemaVersion: replay.MapWeaponPadsSchemaVersion, TitleSlug: "halo_infinite",
		Maps: map[string]replay.MapWeaponPadsEntry{"a": mcEntree("a")},
	}
	if err := WriteAtomic(cat, cible); err == nil {
		t.Fatal("WriteAtomic vers un repertoire doit echouer")
	}
	// AUCUN ORPHELIN : sans le nettoyage, chaque echec laisserait un .tmp de plusieurs
	// centaines de kilo-octets a cote du catalogue.
	entrees, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entrees {
		if strings.HasSuffix(e.Name(), ".tmp") {
			t.Errorf("temporaire orphelin laisse apres echec : %s", e.Name())
		}
	}
}

// TestWriteAtomicDonneUnTemporaireUNIQUE — le coeur du correctif P2-2.
//
// Avec un nom FIXE (`<chemin>.tmp`), deux ecrivains concurrents ecrivaient dans le MEME fichier
// et le `rename` du plus rapide publiait un JSON tronque pour TOUS les lecteurs.
func TestWriteAtomicDonneUnTemporaireUNIQUE(t *testing.T) {
	dir := t.TempDir()
	cible := filepath.Join(dir, "map_weapon_pads.json")
	// On observe les noms de temporaires en faisant echouer le rename : la cible est un
	// repertoire, donc chaque appel laisse son temporaire visible le temps du defer.
	repertoire := filepath.Join(dir, "occupe")
	if err := os.Mkdir(repertoire, 0o750); err != nil {
		t.Fatal(err)
	}
	cat := &replay.MapWeaponPadsCatalog{
		SchemaVersion: replay.MapWeaponPadsSchemaVersion, TitleSlug: "halo_infinite",
		Maps: map[string]replay.MapWeaponPadsEntry{"a": mcEntree("a")},
	}
	// Deux ecritures REUSSIES a la suite : chacune doit publier un catalogue VALIDE. Avec un
	// nom fixe et deux ecrivains, l'une des deux publiait un fichier tronque.
	for i := 0; i < 2; i++ {
		if err := WriteAtomic(cat, cible); err != nil {
			t.Fatalf("ecriture %d : %v", i, err)
		}
		if _, err := replay.LoadMapWeaponPads(cible); err != nil {
			t.Fatalf("catalogue invalide apres l'ecriture %d : %v", i, err)
		}
	}
	// Et aucun temporaire ne subsiste.
	entrees, _ := os.ReadDir(dir)
	for _, e := range entrees {
		if strings.HasSuffix(e.Name(), ".tmp") {
			t.Errorf("temporaire subsistant apres succes : %s", e.Name())
		}
	}
}

// TestChoisirFichierVariante — P1-a : la fonction PURE que la production consomme.
//
// L'ancien test rejouait une COPIE de la boucle : inverser la vraie laissait tout vert. Celui-ci
// porte sur la fonction elle-meme.
func TestChoisirFichierVariante(t *testing.T) {
	cas := []struct {
		nom     string
		chemins []string
		declare string
		attendu string
	}{
		{
			// LE CAS QUI A COUTE 80 METRES. Le catalogue d'objectifs declare le nom du NIVEAU ;
			// l'asset porte aussi la variante. C'est la variante qui est jouee.
			nom:     "asset a deux fichiers, le catalogue declare la carte de BASE",
			chemins: []string{"pre/btb_highpower.mvar", "pre/map.mvar"},
			declare: "btb_highpower.mvar",
			attendu: "pre/map.mvar",
		},
		{
			nom:     "la variante est premiere dans la liste",
			chemins: []string{"pre/map.mvar", "pre/btb_highpower.mvar"},
			declare: "btb_highpower.mvar",
			attendu: "pre/map.mvar",
		},
		{
			// Carte NATIVE : pas de `map.mvar`, le fichier declare EST la variante.
			nom:     "asset sans map.mvar — le fichier declare gagne",
			chemins: []string{"pre/autre.mvar", "pre/catalyst.mvar"},
			declare: "catalyst.mvar",
			attendu: "pre/catalyst.mvar",
		},
		{
			nom:     "rien ne correspond — le premier, faute de mieux",
			chemins: []string{"pre/inconnu.mvar"},
			declare: "absent.mvar",
			attendu: "pre/inconnu.mvar",
		},
		{
			// Le canevas Forge ne doit pas l'emporter : il est nomme, la variante aussi.
			nom:     "canevas Forge et variante",
			chemins: []string{"pre/fo11_blank.mvar", "pre/map.mvar"},
			declare: "fo11_blank.mvar",
			attendu: "pre/map.mvar",
		},
		{
			nom:     "liste vide — pas de defaut a inventer",
			chemins: nil,
			declare: "x.mvar",
			attendu: "",
		},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			if got := ChoisirFichierVariante(c.chemins, c.declare); got != c.attendu {
				t.Errorf("fichier choisi = %q, attendu %q — prendre la carte de BASE pour la "+
					"VARIANTE deplace les socles de plusieurs dizaines de metres", got, c.attendu)
			}
		})
	}
}
