package replaybuild

// artifact_store_test.go — LE RANGEMENT NE RÉTROGRADE JAMAIS.
//
// Le scénario que ces tests protègent est daté et concret : à la bascule vers l'ouvrier, tout
// job DÉJÀ en file porte un payload d'AVANT le transport des faits. Son ouvrier construira donc
// un artefact appauvri, en toute bonne foi, et le déposera par-dessus un artefact complet. Le
// match ne repassera jamais par la sélection post-sync (elle ne voit que les matchs INSÉRÉS
// d'un cycle) : sans ce garde, la perte serait DÉFINITIVE et silencieuse.

import (
	"encoding/json"
	"expvar"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/domain/title"
)

// docJSON fabrique un artefact sérialisé au schéma courant, avec ou sans compteurs de joueur.
func docJSON(t *testing.T, matchID string, avecCompteurs bool) []byte {
	t.Helper()
	doc := replay.ReplayDocument{
		SchemaVersion: replay.SchemaVersion,
		MatchID:       matchID,
		TitleSlug:     title.DefaultSlug,
		// Une trajectoire au moins : validateArtifact refuse un document sans piste.
		Tracks: []replay.Track{{XUID: "2533274819954312"}},
	}
	if avecCompteurs {
		doc.ScoreTimeline = &replay.ScoreTimeline{
			Players: []replay.PlayerScore{{XUID: "2533274819954312"}},
		}
	}
	blob, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return blob
}

// compteurRefus lit le compteur de refus publié par expvar.
func compteurRefus(t *testing.T) int64 {
	t.Helper()
	m, ok := expvar.Get("levelup").(*expvar.Map)
	if !ok || m == nil {
		t.Skip("map expvar « levelup » indisponible")
	}
	v := m.Get("replay_artifact_downgrade_refused_total")
	if v == nil {
		return 0
	}
	n, err := strconv.ParseInt(v.String(), 10, 64)
	if err != nil {
		t.Fatalf("compteur illisible (%q) : %v", v.String(), err)
	}
	return n
}

// TestWriteArtifact_NEcrasePasUnArtefactRiche — LE CHEMIN DÉCOUVERT EN RONDE 2.
//
// `StoreArtifact` (la porte de l'ouvrier) n'est qu'UN des quatre écrivains de ce fichier. Les
// trois autres — le fil de l'eau post-sync (`buildAll`), le CLI de rattrapage
// (`levelup backfill-replay`, y compris `--only-existing`) et l'action admin
// (`RunReplayBuild`) — passent par `BuildMatch` -> `writeArtifact`. Avec des faits VIDES, ils
// produisent un document sans compteurs de joueur : sans garde AU POINT D'ÉCRITURE, ils
// écrasaient silencieusement un artefact riche, sans réparation possible.
//
// Les occasions sont réelles et connues : un job enfilé avant le transport des faits, et
// `chargerFaitsReplay` qui dégrade à vide pour TOUTE une passe si son unique ouverture de base
// échoue.
func TestWriteArtifact_NEcrasePasUnArtefactRiche(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "000d5950.json")
	riche := docJSON(t, "000d5950", true)
	if err := os.WriteFile(path, riche, 0o644); err != nil {
		t.Fatalf("pose de l'artefact riche: %v", err)
	}

	avant := compteurRefus(t)
	// Ce que fait BuildMatch avec des faits vides : un document sans compteurs de joueur.
	appauvri := replay.ReplayDocument{
		SchemaVersion: replay.SchemaVersion,
		MatchID:       "000d5950",
		TitleSlug:     title.DefaultSlug,
		Tracks:        []replay.Track{{XUID: "2533274819954312"}},
	}
	taille, err := writeArtifact(path, title.DefaultSlug, "000d5950", appauvri)
	if err != nil {
		t.Fatalf("writeArtifact: %v", err)
	}
	surDisque, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("relecture: %v", err)
	}
	if string(surDisque) != string(riche) {
		t.Fatal("un artefact RICHE a été écrasé par un appauvri via le chemin BuildMatch — " +
			"la perte serait définitive et silencieuse")
	}
	// La taille rendue doit être celle du disque : annoncer celle du candidat ferait croire à
	// une écriture qui n'a pas eu lieu.
	if taille != len(riche) {
		t.Errorf("taille rendue = %d, attendu %d (celle de l'artefact réellement en place)",
			taille, len(riche))
	}
	if apres := compteurRefus(t); apres != avant+1 {
		t.Errorf("compteur de refus = %d, attendu %d : un refus doit être compté, jamais muet",
			apres, avant+1)
	}
}

// TestWriteArtifact_MonteeDeSchemaToujoursEcrite — LE CAS CONTRÔLE.
//
// Le garde ne doit jamais empêcher une MONTÉE DE SCHÉMA : c'est une reconstruction voulue, et
// un artefact d'un autre schéma ne se compare pas à celui en place. Sans ce contrôle, le garde
// figerait tout le cache au premier incrément de version.
func TestWriteArtifact_MonteeDeSchemaToujoursEcrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "000d5950.json")
	// En place : RICHE, mais à un schéma ANTÉRIEUR.
	ancien := `{"schemaVersion":1,"matchId":"000d5950","scoreTimeline":{"players":[{"xuid":"2533274819954312"}]}}`
	if err := os.WriteFile(path, []byte(ancien), 0o644); err != nil {
		t.Fatalf("pose: %v", err)
	}
	nouveau := replay.ReplayDocument{
		SchemaVersion: replay.SchemaVersion,
		MatchID:       "000d5950",
		TitleSlug:     title.DefaultSlug,
		Tracks:        []replay.Track{{XUID: "2533274819954312"}},
	}
	if _, err := writeArtifact(path, title.DefaultSlug, "000d5950", nouveau); err != nil {
		t.Fatalf("writeArtifact: %v", err)
	}
	surDisque, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("relecture: %v", err)
	}
	if string(surDisque) == ancien {
		t.Fatal("une montée de schéma a été bloquée par le garde — tout le cache resterait figé " +
			"au premier incrément de version")
	}
}

// TestStoreArtifact_RefuseLaRegression : un dépôt SANS compteurs ne remplace pas un artefact
// QUI EN A, à schéma égal. Le fichier sur disque doit rester l'ancien, octet pour octet.
func TestStoreArtifact_RefuseLaRegression(t *testing.T) {
	repoRoot := t.TempDir()
	const matchID = "000d5950"

	complet := docJSON(t, matchID, true)
	if _, err := StoreArtifact(repoRoot, title.DefaultSlug, matchID, complet); err != nil {
		t.Fatalf("dépôt initial: %v", err)
	}
	path := title.NewPathResolver(repoRoot).ReplayArtifactPath(title.DefaultSlug, matchID)

	// Le dépôt appauvri est ACCEPTÉ (pas d'erreur : l'ouvrier a bien travaillé, avec ce qu'on
	// lui avait donné) mais il ne doit RIEN écraser.
	stored, err := StoreArtifact(repoRoot, title.DefaultSlug, matchID, docJSON(t, matchID, false))
	if err != nil {
		t.Fatalf("un dépôt appauvri ne doit pas être une erreur de protocole : %v", err)
	}
	surDisque, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("relecture: %v", err)
	}
	if string(surDisque) != string(complet) {
		t.Fatal("l'artefact COMPLET a été écrasé par un artefact appauvri — la perte serait définitive")
	}
	// L'accusé doit décrire CE QUI EST RANGÉ, pas ce qui a été refusé : un ouvrier qui lirait
	// la taille du blob refusé croirait son travail retenu.
	if stored.Bytes != len(complet) {
		t.Errorf("accusé = %d octets, attendu %d (la taille de l'artefact réellement en place)",
			stored.Bytes, len(complet))
	}
}

// TestStoreArtifact_EnrichissementAccepte : le sens INVERSE doit passer. C'est tout l'objet du
// lot — un artefact complet qui remplace un appauvri est exactement ce qu'on cherche.
func TestStoreArtifact_EnrichissementAccepte(t *testing.T) {
	repoRoot := t.TempDir()
	const matchID = "000d5950"

	if _, err := StoreArtifact(repoRoot, title.DefaultSlug, matchID, docJSON(t, matchID, false)); err != nil {
		t.Fatalf("dépôt initial: %v", err)
	}
	complet := docJSON(t, matchID, true)
	if _, err := StoreArtifact(repoRoot, title.DefaultSlug, matchID, complet); err != nil {
		t.Fatalf("dépôt enrichi: %v", err)
	}
	path := title.NewPathResolver(repoRoot).ReplayArtifactPath(title.DefaultSlug, matchID)
	surDisque, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("relecture: %v", err)
	}
	if string(surDisque) != string(complet) {
		t.Fatal("l'artefact complet n'a pas remplacé l'appauvri — le garde mord dans le mauvais sens")
	}
}

// TestStoreArtifact_PremierDepotToujoursAccepte : sans rien en place, il n'y a rien à protéger.
// Un artefact appauvri vaut mieux que pas de rejeu du tout.
func TestStoreArtifact_PremierDepotToujoursAccepte(t *testing.T) {
	repoRoot := t.TempDir()
	const matchID = "000d5950"
	if _, err := StoreArtifact(repoRoot, title.DefaultSlug, matchID, docJSON(t, matchID, false)); err != nil {
		t.Fatalf("premier dépôt refusé alors qu'aucun artefact n'existait : %v", err)
	}
	if _, err := os.Stat(filepath.Clean(
		title.NewPathResolver(repoRoot).ReplayArtifactPath(title.DefaultSlug, matchID))); err != nil {
		t.Fatalf("aucun artefact rangé: %v", err)
	}
}
