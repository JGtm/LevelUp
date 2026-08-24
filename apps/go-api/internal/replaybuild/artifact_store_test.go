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
	"os"
	"path/filepath"
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
