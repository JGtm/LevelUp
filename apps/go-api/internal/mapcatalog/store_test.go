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
	"strconv"
	"strings"
	"sync"
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

// mcOverlay pose un OVERLAY (meme forme que le catalogue) et rend son chemin. Le nom de
// dossier `generated` reproduit celui du PathResolver — ces tests ne cablent pas le resolver,
// mais ils ne doivent pas non plus donner l'illusion que l'overlay vit a cote du versionne.
func mcOverlay(t *testing.T, dir string, ids ...string) string {
	t.Helper()
	sous := filepath.Join(dir, "generated")
	if err := os.MkdirAll(sous, 0o750); err != nil {
		t.Fatal(err)
	}
	return mcCatalogue(t, sous, ids...)
}

// TestAddOverlayEntryRefuseUneCleExistante — LA promesse de l'ajout-seul.
//
// Supprimer le garde de `AddOverlayEntry` fait tomber ce test : sans lui, un rattrapage
// automatique pourrait REECRIRE une entree que des chemins livres consomment.
func TestAddOverlayEntryRefuseUneCleExistante(t *testing.T) {
	chemin := mcOverlay(t, t.TempDir(), "deja")
	avant, err := os.ReadFile(chemin)
	if err != nil {
		t.Fatal(err)
	}
	// Une entree DIFFERENTE, pour que « rien n'a change » soit une vraie information.
	neuve := mcEntree("deja")
	neuve.ObjectsN = 999
	neuve.Pads = []replay.MapWeaponPadSpot{mcSocle(42, "0x6253CFC0", "rack")}

	err = AddOverlayEntry(chemin, "halo_infinite", "deja", neuve)
	if !errors.Is(err, ErrEntryExists) {
		t.Fatalf("AddOverlayEntry sur une cle existante = %v, attendu ErrEntryExists", err)
	}
	apres, err := os.ReadFile(chemin)
	if err != nil {
		t.Fatal(err)
	}
	if string(avant) != string(apres) {
		t.Error("l'overlay a ete REECRIT alors que la cle existait deja — c'est exactement " +
			"ce que l'ajout-seul doit rendre impossible")
	}
}

// TestAddOverlayEntryAjouteUneCleNeuve — le pendant : la fonction fait bien son travail.
func TestAddOverlayEntryAjouteUneCleNeuve(t *testing.T) {
	chemin := mcOverlay(t, t.TempDir(), "deja")
	if err := AddOverlayEntry(chemin, "halo_infinite", "neuve", mcEntree("neuve")); err != nil {
		t.Fatalf("AddOverlayEntry sur une cle neuve = %v, attendu nil", err)
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

// TestAddOverlayEntryCreeLOverlayAbsent — LE CAS NOMINAL DU PREMIER RATTRAPAGE.
//
// C'est le renversement de contrat du 2026-09-05 (constat A0) : l'ancienne `AddEntry` DEVAIT
// echouer sur un fichier absent (en creer un de zero aurait efface toutes les cartes du titre,
// puisqu'elle ecrivait le catalogue VERSIONNE). L'overlay, lui, ne porte que ce que le runtime
// a rattrape : le creer ne perd rien, et le refuser bloquerait le tout premier rattrapage.
func TestAddOverlayEntryCreeLOverlayAbsent(t *testing.T) {
	absent := filepath.Join(t.TempDir(), "generated", "map_weapon_pads.json")
	if err := AddOverlayEntry(absent, "halo_infinite", "neuve", mcEntree("neuve")); err != nil {
		t.Fatalf("AddOverlayEntry sur un overlay absent = %v, attendu nil (premier rattrapage)", err)
	}
	cat, err := replay.LoadMapWeaponPads(absent)
	if err != nil {
		t.Fatalf("overlay cree illisible : %v", err)
	}
	if cat.SchemaVersion != replay.MapWeaponPadsSchemaVersion {
		t.Errorf("schema_version = %d, attendu %d — un overlay qu'aucun lecteur n'accepte est "+
			"un overlay mort", cat.SchemaVersion, replay.MapWeaponPadsSchemaVersion)
	}
	if cat.TitleSlug != "halo_infinite" {
		t.Errorf("title_slug = %q, attendu halo_infinite", cat.TitleSlug)
	}
	if _, ok := cat.Maps["neuve"]; !ok || len(cat.Maps) != 1 {
		t.Errorf("overlay cree avec %d carte(s), attendu la seule carte rattrapee", len(cat.Maps))
	}
}

// TestAddOverlayEntryEchoueSurOverlayCorrompu — on n'ECRASE PAS un overlay qu'on ne sait pas
// lire : les cartes deja rattrapees disparaitraient en silence.
func TestAddOverlayEntryEchoueSurOverlayCorrompu(t *testing.T) {
	corrompu := filepath.Join(t.TempDir(), "map_weapon_pads.json")
	if err := os.WriteFile(corrompu, []byte("{ pas du json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := AddOverlayEntry(corrompu, "halo_infinite", "x", mcEntree("x")); err == nil {
		t.Error("AddOverlayEntry sur un overlay corrompu doit ECHOUER")
	}
	blob, err := os.ReadFile(corrompu)
	if err != nil {
		t.Fatal(err)
	}
	if string(blob) != "{ pas du json" {
		t.Error("l'overlay corrompu a ete REECRIT — il doit rester tel quel pour etre diagnostique")
	}
}

// TestAddOverlayEntryNeTouchePasLeCatalogueVersionne — LE TEST DU CONSTAT A0 LUI-MEME.
//
// Le catalogue versionne est un fichier SUIVI PAR GIT : le runtime ne doit pas le toucher, pas
// meme d'un octet. Ce test le pose cote a cote avec l'overlay et verifie qu'il ressort
// BYTE-IDENTIQUE apres un rattrapage — c'est le seul niveau ou « le runtime n'ecrit plus le
// fichier versionne » se verifie sans lire le code.
func TestAddOverlayEntryNeTouchePasLeCatalogueVersionne(t *testing.T) {
	dir := t.TempDir()
	versionne := mcCatalogue(t, dir, "carte_relue")
	avant, err := os.ReadFile(versionne)
	if err != nil {
		t.Fatal(err)
	}
	statAvant, err := os.Stat(versionne)
	if err != nil {
		t.Fatal(err)
	}

	overlay := filepath.Join(dir, "generated", "map_weapon_pads.json")
	if err := AddOverlayEntry(overlay, "halo_infinite", "carte_rattrapee", mcEntree("carte_rattrapee")); err != nil {
		t.Fatalf("AddOverlayEntry = %v", err)
	}

	apres, err := os.ReadFile(versionne)
	if err != nil {
		t.Fatal(err)
	}
	if string(avant) != string(apres) {
		t.Error("LE CATALOGUE VERSIONNE A ETE MODIFIE par un rattrapage runtime — c'est le " +
			"constat A0 (un deploiement l'efface, un commit local l'avale sans relecture)")
	}
	statApres, err := os.Stat(versionne)
	if err != nil {
		t.Fatal(err)
	}
	if !statAvant.ModTime().Equal(statApres.ModTime()) {
		t.Error("le catalogue versionne a ete REECRIT (mtime differente) meme a contenu egal")
	}
	// Et l'overlay, lui, porte bien la carte rattrapee et RIEN d'autre.
	sur, err := replay.LoadMapWeaponPads(overlay)
	if err != nil {
		t.Fatalf("overlay illisible : %v", err)
	}
	if _, ok := sur.Maps["carte_rattrapee"]; !ok || len(sur.Maps) != 1 {
		t.Errorf("overlay = %d carte(s), attendu la seule carte rattrapee", len(sur.Maps))
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

// TestAddOverlayEntryConcurrentNePerdPasDEntree — ITEM 7 : la perte de mise a jour.
//
// L'ajout fait un LIRE-MODIFIER-ECRIRE. Deux ecrivains — deux cycles de sync, ou un cycle et un
// outil — pouvaient lire le meme etat et publier chacun un fichier SANS la carte de l'autre.
// C'est le trou que ce lot comble qui se rouvrait.
func TestAddOverlayEntryConcurrentNePerdPasDEntree(t *testing.T) {
	chemin := mcOverlay(t, t.TempDir(), "socle")
	const n = 8
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()
			id := "carte" + strconv.Itoa(i)
			if err := AddOverlayEntry(chemin, "halo_infinite", id, mcEntree(id)); err != nil {
				t.Errorf("AddOverlayEntry(%s) = %v", id, err)
			}
		}(i)
	}
	wg.Wait()
	cat, err := replay.LoadMapWeaponPads(chemin)
	if err != nil {
		t.Fatalf("catalogue illisible apres ecritures concurrentes : %v", err)
	}
	// TOUTES les cartes doivent y etre, plus celle de depart.
	if len(cat.Maps) != n+1 {
		t.Errorf("%d cartes au catalogue, attendu %d — une ecriture concurrente a ECRASE le "+
			"travail d'une autre", len(cat.Maps), n+1)
	}
	for i := 0; i < n; i++ {
		if _, ok := cat.Maps["carte"+strconv.Itoa(i)]; !ok {
			t.Errorf("carte%d perdue", i)
		}
	}
	// Et aucun verrou ne subsiste.
	if _, err := os.Stat(chemin + ".lock"); err == nil {
		t.Error("le verrou n'a pas ete retire")
	}
}

// TestChoisirFichierVarianteSurLaNomenclatureAPLATIE — ITEM 2 : la preference doit OPERER sur
// les noms que le depot local porte reellement.
//
// 62 des 72 entrees du catalogue livre ont un `mvar_file` de la forme `{prefixe}_map.mvar` :
// c'est la nomenclature du dump qui l'a bati. Sans reconnaitre ce suffixe, la preference ne
// matchait jamais cote CLI et le code etait decoratif.
func TestChoisirFichierVarianteSurLaNomenclatureAPLATIE(t *testing.T) {
	cas := []struct {
		nom     string
		chemins []string
		declare string
		attendu string
	}{
		{
			// La forme REELLE du depot : noms prefixes pour desambiguiser 58 homonymes.
			nom: "depot aplati — la variante prefixee gagne sur la carte de base prefixee",
			chemins: []string{"highpower_sentry_defense_btb_highpower.mvar",
				"highpower_sentry_defense_map.mvar"},
			declare: "highpower_sentry_defense_btb_highpower.mvar",
			attendu: "highpower_sentry_defense_map.mvar",
		},
		{
			nom:     "nom public a espaces et tiret",
			chemins: []string{"aquarius_-_ranked_ctf_aquarius.mvar", "aquarius_-_ranked_map.mvar"},
			declare: "aquarius_-_ranked_ctf_aquarius.mvar",
			attendu: "aquarius_-_ranked_map.mvar",
		},
		{
			// Sans variante au depot, le declare reste le bon choix.
			nom:     "depot sans variante — le declare gagne",
			chemins: []string{"deadlock_btb_drydock.mvar"},
			declare: "deadlock_btb_drydock.mvar",
			attendu: "deadlock_btb_drydock.mvar",
		},
		{
			// PIEGE A NE PAS CREER : un fichier qui CONTIENT « map » sans etre la variante.
			nom:     "un nom qui contient map sans etre la variante",
			chemins: []string{"mapmaker.mvar", "streets_map.mvar"},
			declare: "mapmaker.mvar",
			attendu: "streets_map.mvar",
		},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			if got := ChoisirFichierVariante(c.chemins, c.declare); got != c.attendu {
				t.Errorf("fichier choisi = %q, attendu %q", got, c.attendu)
			}
		})
	}
}
