// cmd/mapobj-build — refresh_test.go : recette de la régénération HORS LIGNE.
//
// Ce qui est gardé ici, et pourquoi (constat D-P1a de la revue adversariale du
// 2026-08-02) : `refreshOffline` réécrit le catalogue en `schema_version` COURANT même
// quand certaines cartes n'ont pas pu être re-parsées (leur `.mvar` manque). Ces cartes
// restent au schéma d'avant. En v2, une zone non migrée n'a pas de `shape` — exactement
// comme un objectif PONCTUEL, qui n'en a pas non plus. Un consommateur ne pouvait pas
// distinguer les deux et affichait un point dans les deux cas : une absence de migration
// lue comme une mesure de ponctualité.
//
// Les deux premiers tests ne dépendent d'AUCUNE fixture : ils tournent partout, CI
// comprise. Le troisième a besoin d'un `.mvar` réel (actif propriétaire versionné sous
// .ai/V7.5/dumps/) et s'annonce absent plutôt que de passer en silence.
package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"levelup/go-api/internal/testutil"
)

// ecrireCatalogue pose un catalogue de départ à la version demandée, avec des cartes
// dont le `.mvar` sera introuvable — c'est le chemin de report qu'on veut observer.
func ecrireCatalogue(t *testing.T, schema int, entrees map[string]*mapEntry) string {
	t.Helper()
	out := filepath.Join(t.TempDir(), "map_objectives.json")
	cat := &catalog{
		SchemaVersion: schema,
		TitleSlug:     "halo_infinite",
		GeneratedAt:   time.Now().UTC(),
		Maps:          entrees,
		Coverage:      map[string]coverStats{},
	}
	for id, e := range entrees {
		cat.Coverage[id] = coverStats{Objectives: len(e.Objectives)}
	}
	buf, err := json.MarshalIndent(cat, "", "  ")
	if err != nil {
		t.Fatalf("sérialisation du catalogue de départ: %v", err)
	}
	if err := os.WriteFile(out, buf, 0o644); err != nil {
		t.Fatalf("écriture du catalogue de départ: %v", err)
	}
	return out
}

// relire rend le catalogue écrit par refreshOffline.
func relire(t *testing.T, path string) *catalog {
	t.Helper()
	buf, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("relecture du catalogue: %v", err)
	}
	var got catalog
	if err := json.Unmarshal(buf, &got); err != nil {
		t.Fatalf("catalogue illisible: %v", err)
	}
	return &got
}

// TestRefreshMarqueLesCartesReporteesDUnSchemaAnterieur — LE test du constat D-P1a.
//
// Mutation qui doit le faire rougir : retirer le marquage de `carryOver` (revenir à
// `cat.Maps[id] = prev.Maps[id]`).
func TestRefreshMarqueLesCartesReporteesDUnSchemaAnterieur(t *testing.T) {
	out := ecrireCatalogue(t, 1, map[string]*mapEntry{
		"carte-sans-fichier": {MapID: "carte-sans-fichier", MvarFile: "absent.mvar"},
		"carte-sans-nom":     {MapID: "carte-sans-nom", MvarFile: ""},
	})
	vide := t.TempDir()

	if err := refreshOffline(context.Background(), vide, out, "halo_infinite", false); err != nil {
		t.Fatalf("refreshOffline: %v", err)
	}

	got := relire(t, out)
	if got.SchemaVersion != catalogSchemaVersion {
		t.Fatalf("schema_version = %d, attendu %d", got.SchemaVersion, catalogSchemaVersion)
	}
	if len(got.Maps) != 2 {
		t.Fatalf("cartes reportées = %d, attendu 2 (un objectif périmé vaut mieux qu'effacé)", len(got.Maps))
	}
	for id, e := range got.Maps {
		if e.CarriedFromSchema != 1 {
			t.Errorf("carte %s : carried_from_schema = %d, attendu 1 — une carte non migrée "+
				"doit se distinguer d'une carte produite par le schéma courant", id, e.CarriedFromSchema)
		}
	}
}

// TestRefreshNeMarquePasUnReportDepuisLeSchemaCourant — le marqueur ne doit pas devenir
// du bruit : reporter une carte déjà au schéma courant ne cache aucune migration
// manquante, donc rien à signaler.
//
// Mutation qui doit le faire rougir : marquer inconditionnellement dans `carryOver`.
func TestRefreshNeMarquePasUnReportDepuisLeSchemaCourant(t *testing.T) {
	out := ecrireCatalogue(t, catalogSchemaVersion, map[string]*mapEntry{
		"carte-a-jour": {MapID: "carte-a-jour", MvarFile: "absent.mvar"},
	})

	if err := refreshOffline(context.Background(), t.TempDir(), out, "halo_infinite", false); err != nil {
		t.Fatalf("refreshOffline: %v", err)
	}

	got := relire(t, out)
	if e := got.Maps["carte-a-jour"]; e == nil || e.CarriedFromSchema != 0 {
		t.Fatalf("carried_from_schema = %v, attendu absent (0) sur un report depuis le schéma courant",
			e.CarriedFromSchema)
	}
}

// TestRefreshNEcritRienEnDryRun — le mode d'essai ne doit pas toucher au fichier, sinon
// il n'y a plus de moyen d'observer une migration avant de la subir.
func TestRefreshNEcritRienEnDryRun(t *testing.T) {
	out := ecrireCatalogue(t, 1, map[string]*mapEntry{
		"carte": {MapID: "carte", MvarFile: "absent.mvar"},
	})
	avant, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("lecture avant: %v", err)
	}

	if err := refreshOffline(context.Background(), t.TempDir(), out, "halo_infinite", true); err != nil {
		t.Fatalf("refreshOffline dry-run: %v", err)
	}

	apres, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("lecture après: %v", err)
	}
	if string(avant) != string(apres) {
		t.Fatal("le dry-run a réécrit le catalogue")
	}
}

// TestRefreshRefuseDEcrireSurUnParseEnEchec — un `.mvar` illisible n'est pas un `.mvar`
// absent : le premier est une corruption qu'il faut voir, le second une carte qu'on
// reporte. Le refresh doit échouer SANS avoir écrit.
func TestRefreshRefuseDEcrireSurUnParseEnEchec(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "casse.mvar"), []byte("ceci n'est pas un mvar"), 0o644); err != nil {
		t.Fatalf("écriture de la fixture corrompue: %v", err)
	}
	out := ecrireCatalogue(t, 1, map[string]*mapEntry{
		"carte-cassee": {MapID: "carte-cassee", MvarFile: "casse.mvar"},
	})
	avant, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("lecture avant: %v", err)
	}

	if err := refreshOffline(context.Background(), dir, out, "halo_infinite", false); err == nil {
		t.Fatal("refreshOffline a réussi sur un .mvar corrompu, attendu une erreur")
	}

	apres, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("lecture après: %v", err)
	}
	if string(avant) != string(apres) {
		t.Fatal("le catalogue a été réécrit malgré l'échec de parse")
	}
}

// TestRefreshRegenereEtDemarqueUneCarteReparsee — le pendant positif : une carte dont le
// `.mvar` est là est RE-PARSÉE, donc elle porte les champs du schéma courant et ne doit
// plus être marquée. C'est ce qui garantit que le marqueur tombe quand la migration a
// réellement eu lieu, au lieu de rester collé.
func TestRefreshRegenereEtDemarqueUneCarteReparsee(t *testing.T) {
	src := filepath.Join("..", "..", "..", "..", ".ai", "V7.5", "dumps", "mapvar", "cliffhanger_map.mvar")
	buf, err := os.ReadFile(src)
	if err != nil {
		t.Skipf("fixture %s absente (%v) — recette du re-parse ignorée", src, err)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "cliffhanger_map.mvar"), buf, 0o644); err != nil {
		t.Fatalf("copie de la fixture: %v", err)
	}
	out := ecrireCatalogue(t, 1, map[string]*mapEntry{
		"cliffhanger": {
			MapID: "cliffhanger", MvarFile: "cliffhanger_map.mvar",
			PublicName: "Cliffhanger", VersionID: "v-fige",
		},
	})

	if err := refreshOffline(context.Background(), dir, out, "halo_infinite", false); err != nil {
		t.Fatalf("refreshOffline: %v", err)
	}

	e := relire(t, out).Maps["cliffhanger"]
	if e == nil {
		t.Fatal("carte absente du catalogue régénéré")
	}
	if e.CarriedFromSchema != 0 {
		t.Errorf("carried_from_schema = %d sur une carte RE-PARSÉE, attendu absent", e.CarriedFromSchema)
	}
	if len(e.Objectives) == 0 {
		t.Fatal("aucun objectif re-parsé — la fixture ou le parse a changé")
	}
	if e.PublicName != "Cliffhanger" || e.VersionID != "v-fige" {
		t.Errorf("métadonnées réseau perdues au refresh: public_name=%q version_id=%q",
			e.PublicName, e.VersionID)
	}
	formes := 0
	for _, ob := range e.Objectives {
		if ob.Shape != nil {
			formes++
		}
	}
	if formes == 0 {
		t.Error("aucune forme de zone après re-parse — c'est précisément ce que le schéma 2 apporte")
	}
}

// mvarVersionne rend le chemin d'un `.mvar` SUIVI EN DÉPÔT (témoins de test de
// `.ai/V7.5/dumps/mapvar/`). Il est versionné : son absence est une installation cassée,
// pas un cas à sauter — la racine vient de testutil.RepoRoot(), sans variable
// d'environnement (revue ronde 1, R1-1).
func mvarVersionne(t *testing.T, nom string) string {
	t.Helper()
	root, err := testutil.RepoRoot()
	if err != nil {
		t.Fatalf("racine du dépôt introuvable : %v", err)
	}
	p := filepath.Join(root, ".ai", "V7.5", "dumps", "mapvar", nom)
	if _, statErr := os.Stat(p); statErr != nil {
		t.Fatalf("témoin versionné absent : %v", statErr)
	}
	return p
}

// TestIngestLocalPreserveLesMetadonneesReseau — LE GARDE-FOU du chemin `--from-file` :
// une carte DÉJÀ au catalogue garde ses métadonnées RÉSEAU quand on la re-parse hors ligne.
//
// Pourquoi c'est une règle et pas un détail : `--from-file` ne parle à personne. S'il
// écrasait `version_id`, `public_name` et `fetched_at`, un catalogue gelé serait daté du
// jour de sa RELECTURE et perdrait le nom public et la version de l'asset — sans un mot,
// et sans moyen de savoir ensuite à quel état de l'UGC il correspond. C'est la même règle
// que `--refresh-from` (refresh.go), qui, lui, était déjà gardé. La revue adversariale
// ronde 1 (R1-4) a relevé que ce bloc de `ingestLocal` n'avait aucun test.
//
// Mutation qui doit le faire rougir : retirer le `if prev, ok := cat.Maps[mapID]` de
// ingestLocal (cmd/mapobj-build/main.go).
func TestIngestLocalPreserveLesMetadonneesReseau(t *testing.T) {
	src := mvarVersionne(t, "cliffhanger_map.mvar")
	gele := time.Date(2026, 7, 26, 0, 13, 14, 0, time.UTC)
	cat := newCatalog("halo_infinite")
	cat.Maps["carte-a"] = &mapEntry{
		MapID: "carte-a", VersionID: "v-reseau", PublicName: "Cliffhanger",
		MvarFile: "map.mvar", Module: "map", FetchedAt: gele,
	}

	if err := ingestLocal(context.Background(), cat, "carte-a", src, ""); err != nil {
		t.Fatalf("ingestLocal: %v", err)
	}
	e := cat.Maps["carte-a"]
	if e == nil {
		t.Fatal("la carte a disparu du catalogue")
	}
	if e.VersionID != "v-reseau" {
		t.Errorf("version_id = %q, attendu « v-reseau » — le re-parse local a écrasé une "+
			"métadonnée qu'il ne connaît pas", e.VersionID)
	}
	if e.PublicName != "Cliffhanger" {
		t.Errorf("public_name = %q, attendu « Cliffhanger »", e.PublicName)
	}
	if !e.FetchedAt.Equal(gele) {
		t.Errorf("fetched_at = %s, attendu %s — le catalogue serait daté du jour de sa relecture",
			e.FetchedAt.Format(time.RFC3339), gele.Format(time.RFC3339))
	}
	// Le nom de fichier, LUI, doit suivre le parse : c'est la vérité de ce que le
	// catalogue porte désormais.
	if e.MvarFile != "cliffhanger_map.mvar" || e.Module != "cliffhanger_map" {
		t.Errorf("mvar_file/module = %q/%q, attendu ceux du fichier parsé", e.MvarFile, e.Module)
	}
	if len(e.Objectives) == 0 {
		t.Error("aucun objectif ingéré — le témoin ne dit rien")
	}

	// Une carte NEUVE n'hérite de rien et se date du jour : la préservation ne doit pas
	// se transformer en refus d'écrire.
	if err := ingestLocal(context.Background(), cat, "carte-neuve", src, ""); err != nil {
		t.Fatalf("ingestLocal (carte neuve): %v", err)
	}
	n := cat.Maps["carte-neuve"]
	if n.VersionID != "" || n.PublicName != "" {
		t.Errorf("carte neuve : version_id=%q public_name=%q, attendus vides", n.VersionID, n.PublicName)
	}
	if !n.FetchedAt.After(gele) {
		t.Errorf("carte neuve : fetched_at = %s, attendu la date du parse",
			n.FetchedAt.Format(time.RFC3339))
	}
}
