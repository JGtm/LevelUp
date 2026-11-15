package replaybuild

// artifact_cadence_garde_test.go — LE PAS DANS LE GARDE ANTI-RÉGRESSION DU PUITS.
//
// Le pas est une marque de fraîcheur au même titre que le schéma ([CadenceUpToDate]). Le garde
// d'écriture ([wouldDowngrade]) lui doit donc deux conduites, chacune prouvée par sa mutation :
//
//   - un artefact au pas courant n'est pas écrasé par un candidat à un autre pas (mutation : retirer le
//     premier cas du garde laisse passer le retour à la grille périmée) ;
//   - un artefact à un pas périmé se laisse remplacer par un candidat au pas courant, même quand le garde de
//     compteurs refuserait (mutation : retirer le second cas refuse la régénération).

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/replay"
)

// poserArtefactAuPas écrit un artefact au schéma courant, aux couches courantes, au pas donné, portant ou non
// des compteurs de joueur ; rend son chemin.
func poserArtefactAuPas(t *testing.T, pas int, avecJoueurs bool) string {
	t.Helper()
	out := filepath.Join(t.TempDir(), "artefact.json")
	if err := os.WriteFile(out, candidatAuPas(t, pas, avecJoueurs), 0o600); err != nil {
		t.Fatalf("pose : %v", err)
	}
	return out
}

// candidatAuPas sérialise un document candidat au schéma courant et aux couches courantes.
func candidatAuPas(t *testing.T, pas int, avecJoueurs bool) []byte {
	t.Helper()
	doc := map[string]any{
		"schemaVersion":   replay.SchemaVersion,
		"frameIntervalMs": pas,
		"matchId":         "m",
		"tracks":          []map[string]any{{"xuid": "1"}},
		"layers":          couchesCourantes(),
	}
	if avecJoueurs {
		doc["scoreTimeline"] = map[string]any{"players": []map[string]any{{"xuid": "1"}}}
	}
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("serialisation : %v", err)
	}
	return raw
}

func TestGardeDuPuits_UnArtefactAuPasCourantNEstPasEcrase(t *testing.T) {
	enPlace := poserArtefactAuPas(t, CookFrameIntervalMS(), true)
	if _, oui := wouldDowngrade(enPlace, candidatAuPas(t, HistoricalFrameIntervalMS, true)); !oui {
		t.Error("un candidat à un autre pas écrase un artefact au pas courant : le garde ne lit pas le pas")
	}
	// RETOUR : le même candidat au pas courant passe.
	if _, oui := wouldDowngrade(enPlace, candidatAuPas(t, CookFrameIntervalMS(), true)); oui {
		t.Error("un candidat au pas courant est refusé : le garde mord trop large")
	}
}

func TestGardeDuPuits_UnArtefactAUnPasPerimeSeLaisseRemplacer(t *testing.T) {
	enPlace := poserArtefactAuPas(t, HistoricalFrameIntervalMS, true)
	// Le candidat régénéré n'a pas de compteurs : le garde de compteurs le refuserait à pas égal.
	if _, oui := wouldDowngrade(enPlace, candidatAuPas(t, CookFrameIntervalMS(), false)); oui {
		t.Error("un artefact à un pas périmé ne peut pas être remplacé par sa régénération au pas courant")
	}
	// RETOUR : à pas égal (périmé des deux côtés), le garde de compteurs reste celui d'avant.
	if _, oui := wouldDowngrade(enPlace, candidatAuPas(t, HistoricalFrameIntervalMS, false)); !oui {
		t.Error("à pas égal, la perte des compteurs de joueur n'est plus refusée : le garde d'avant est cassé")
	}
}
