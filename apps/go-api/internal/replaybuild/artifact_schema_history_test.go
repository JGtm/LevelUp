package replaybuild

// artifact_schema_history_test.go — LES ANCIENS SCHEMAS NE SE RELISENT JAMAIS « A JOUR ».
//
// CE QUE CE FICHIER FERME (architecture §12, garde-rail 5). La reprise de tout backfill se
// fait par `replay.SchemaVersion` : un artefact d'une version anterieure doit se lire « a
// re-cuire ». Le predicat existait, mais sa preuve se limitait a UN cas ecrit a la main
// (`schemaVersion: 1`, cf. artifact_store_test.go) — c'est-a-dire a une version, sur les
// cinquante et une que la chronique declare. Une regle qui ne vaut que pour la version qu'un
// test a nommee n'est pas une regle.
//
// LA LISTE DES VERSIONS N'EST PAS ECRITE ICI, ET C'EST LE POINT. Elle est DERIVEE de
// `document_chronicle.go` (helper `testutil.ReplayChronicleVersions`) : la prochaine montee
// entre donc dans ces tests sans que personne n'y pense, et une version sautee a la
// renumerotation (la 32 l'est) n'y entre pas par erreur.
//
// LA 51 N'ETAIT PAS SAUTEE, contrairement a ce que ce fichier affirmait (corrige le 2026-09-14,
// lot 1.0 revue R2) : `2fb53db4e` la pose, `b6b198baf` la remplace par 52 le lendemain. Elle
// manquait simplement a la chronique, donc a cette liste — une version REELLEMENT cuite que ces
// trois tests ne rejouaient pas. Son entree restauree, elle y entre d'elle-meme.
//
// CE QUE LA MESURE A TROUVE, ET QUI CONTREDIT LE DOCUMENT D'ARCHITECTURE (2026-09-13, lot
// 0.B). §12 point 5 affirme que « le point d'ecriture unique (`writeArtifactBytes`) refuse
// toute retrogradation ». SUR PIECES, IL EN REFUSE UNE SEULE : l'APPAUVRISSEMENT A SCHEMA
// EGAL (`wouldDowngrade`, artifact_store.go:88-98). A schema DIFFERENT il se tait
// DELIBEREMENT — son propre en-tete l'ecrit (artifact_store.go:85-87 : « une montee de schema
// doit TOUJOURS passer »), et `TestWriteArtifact_MonteeDeSchemaToujoursEcrite` le verrouille
// dans ce sens. La retrogradation de VERSION est refusee AILLEURS, en amont : `validateArtifact`
// (artifact_store.go:112) rejette tout document dont le numero n'est pas le courant, et c'est
// la porte par laquelle passe le seul ecrivain qui pourrait porter une autre version (le depot
// d'un ouvrier, `StoreArtifact`). Les deux gardes sont donc prouvees ici PAR VERSION, chacune
// pour ce qu'elle fait — et la difference est consignee au plan (§4), pas corrigee dans ce lot.

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_infinite/film/replay"
	"levelup/go-api/internal/testutil"
)

// versionsAnterieures : les versions que la chronique declare, sous la version courante.
func versionsAnterieures(t *testing.T) []int {
	t.Helper()
	toutes, err := testutil.ReplayChronicleVersions()
	if err != nil {
		t.Fatalf("chronique du rejeu illisible : %v", err)
	}
	var out []int
	for _, v := range toutes {
		if v < replay.SchemaVersion {
			out = append(out, v)
		}
	}
	if len(out) < 10 {
		t.Fatalf("seulement %d version(s) anterieure(s) derivees de la chronique (%v) — "+
			"une preuve par version qui ne couvre presque rien n'en est pas une", len(out), out)
	}
	return out
}

// artefactAuSchema : un artefact serialise A LA VERSION DONNEE, riche ou appauvri.
//
// Le corps est ecrit a la main et non via `replay.ReplayDocument` : le document courant ne
// SAIT PLUS ecrire une version anterieure, et c'est precisement ce qu'on veut simuler — le
// fichier laisse sur disque par un binaire d'hier.
func artefactAuSchema(version int, matchID string, avecCompteurs bool) []byte {
	corps := map[string]any{
		"schemaVersion": version,
		"matchId":       matchID,
		"titleSlug":     title.DefaultSlug,
		"tracks":        []map[string]any{{"xuid": "2533274819954312"}},
	}
	if avecCompteurs {
		corps["scoreTimeline"] = map[string]any{
			"players": []map[string]any{{"xuid": "2533274819954312"}},
		}
	}
	blob, err := json.Marshal(corps)
	if err != nil {
		panic(fmt.Sprintf("artefactAuSchema: %v", err))
	}
	return blob
}

// TestChaqueSchemaAnterieurSeLitARecuire : pour CHAQUE version de la chronique inferieure a la
// courante, le digest la classe perimee — en memoire comme sur disque.
//
// LES DEUX FORMES SONT VERIFIEES parce que ce sont deux appelants distincts : les gardes de
// fraicheur lisent le digest une fois (`ArtifactDigest(...).UpToDate()`), les CLI de backfill
// appellent la vue `ArtifactUpToDate(path)`. Une des deux qui divergerait relancerait, ou
// omettrait, toute une passe de recuisson.
func TestChaqueSchemaAnterieurSeLitARecuire(t *testing.T) {
	dir := t.TempDir()
	for _, v := range versionsAnterieures(t) {
		t.Run(fmt.Sprintf("schema_%d", v), func(t *testing.T) {
			if (Digest{SchemaVersion: v}).UpToDate() {
				t.Fatalf("un artefact au schema %d se lit « a jour » alors que le producteur "+
					"ecrit %d — aucune recuisson ne le rattraperait", v, replay.SchemaVersion)
			}
			path := filepath.Join(dir, fmt.Sprintf("%d.json", v))
			if err := os.WriteFile(path, artefactAuSchema(v, "000d5950", true), 0o600); err != nil {
				t.Fatalf("pose : %v", err)
			}
			if ArtifactUpToDate(path) {
				t.Errorf("ArtifactUpToDate dit « a jour » sur un artefact au schema %d", v)
			}
			d, ok := ArtifactDigest(path)
			if !ok {
				t.Fatalf("digest illisible sur un artefact au schema %d", v)
			}
			if d.SchemaVersion != v {
				t.Errorf("le digest lit le schema %d, le fichier porte %d", d.SchemaVersion, v)
			}
		})
	}
}

// TestLeDepotRefuseChaqueSchemaAnterieur : le depot d'un ouvrier portant une version
// ANTERIEURE est refuse, pour chaque version de la chronique — et RIEN n'est ecrit.
//
// C'EST LA VRAIE PORTE DE LA RETROGRADATION DE VERSION (cf. en-tete) : c'est le seul ecrivain
// qui puisse presenter un artefact d'un autre binaire. Le controle porte aussi sur le disque :
// un refus qui aurait quand meme ecrit serait pire qu'une absence de refus, puisqu'il se
// raconterait comme un succes de protection.
func TestLeDepotRefuseChaqueSchemaAnterieur(t *testing.T) {
	const matchID = "000d5950"
	for _, v := range versionsAnterieures(t) {
		t.Run(fmt.Sprintf("schema_%d", v), func(t *testing.T) {
			repoRoot := t.TempDir()
			path := title.NewPathResolver(repoRoot).ReplayArtifactPath(title.DefaultSlug, matchID)
			_, err := StoreArtifact(repoRoot, title.DefaultSlug, matchID,
				artefactAuSchema(v, matchID, true))
			if !errors.Is(err, domain.ErrBuildArtifactInvalid) {
				t.Fatalf("un depot au schema %d rend %v, attendu ErrBuildArtifactInvalid", v, err)
			}
			if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
				t.Errorf("un depot refuse au schema %d a tout de meme ecrit %s", v, path)
			}
		})
	}
}

// TestLePointDEcritureRefuseLAppauvrissementAChaqueSchema : a schema EGAL, un candidat sans
// compteurs de joueur n'ecrase jamais un artefact qui en porte — quelle que soit la version.
//
// POURQUOI LE PROUVER SUR LES ANCIENNES VERSIONS AUSSI, alors que la production n'ecrit que la
// courante : le parc, lui, porte toutes les versions de la chronique, et un backfill partiel
// re-cuit a la version d'hier sur un poste dont le binaire est en retard. Le garde ne doit pas
// dependre du numero.
func TestLePointDEcritureRefuseLAppauvrissementAChaqueSchema(t *testing.T) {
	const matchID = "000d5950"
	for _, v := range versionsAnterieures(t) {
		t.Run(fmt.Sprintf("schema_%d", v), func(t *testing.T) {
			path := filepath.Join(t.TempDir(), matchID+".json")
			riche := artefactAuSchema(v, matchID, true)
			if err := os.WriteFile(path, riche, 0o600); err != nil {
				t.Fatalf("pose : %v", err)
			}
			enPlace, err := writeArtifactBytes(path, title.DefaultSlug, matchID,
				artefactAuSchema(v, matchID, false))
			if err != nil {
				t.Fatalf("writeArtifactBytes : %v", err)
			}
			if enPlace.Players == 0 {
				t.Errorf("l'accuse decrit un artefact sans compteurs alors que celui en place "+
					"en porte (schema %d)", v)
			}
			surDisque, err := os.ReadFile(path) //nolint:gosec // chemin de repertoire temporaire de test
			if err != nil {
				t.Fatalf("relecture : %v", err)
			}
			if string(surDisque) != string(riche) {
				t.Errorf("l'artefact riche au schema %d a ete ecrase par un appauvri", v)
			}
		})
	}
}
