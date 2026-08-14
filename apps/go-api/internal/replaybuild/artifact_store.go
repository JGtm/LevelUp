// Package replaybuild — artifact_store.go : RANGER un artefact construit AILLEURS.
//
// C'est le dernier maillon du transport (piste F §1) : l'ouvrier n'a aucun port
// entrant, il POUSSE son artefact vers le web, et le web le VALIDE puis le RANGE.
// Ce fichier porte les deux gestes, hors de la couche HTTP — un handler ne décide
// pas de ce qui mérite d'atterrir sur le disque.
//
// TROIS REFUS AVANT LA MOINDRE ÉCRITURE, et l'ordre compte :
//
//  1. ça se désérialise en ReplayDocument (sinon ce n'est pas un artefact) ;
//  2. la version de schéma est CELLE DE CE DÉPÔT (un artefact construit par un
//     décodeur plus ancien se refuse, il ne se range pas — c'est déjà la clé de
//     reprise des backfills, cf. ArtifactUpToDate) ;
//  3. le match est BIEN celui du job (un artefact rangé sous le nom d'un autre
//     match serait servi comme le rejeu de ce match, sans que rien ne le signale).
//
// L'ÉCRITURE EST ATOMIQUE (temporaire puis renommage, via platform/atomicfile) :
// le service de lecture sert le fichier tel quel, sans verrou ni version — une
// écriture en place lui ferait servir un artefact à moitié écrit à qui consulte
// pendant le dépôt.
package replaybuild

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/atomicfile"
)

// StoredArtifact décrit ce qui a été rangé, tel qu'on le rend à l'ouvrier.
type StoredArtifact struct {
	Path          string
	MatchID       string
	Bytes         int
	Tracks        int
	SchemaVersion int
}

// StoreArtifact valide un artefact reçu d'un ouvrier et l'écrit à la place
// canonique du match (PathResolver.ReplayArtifactPath). matchID est celui du JOB,
// jamais celui que l'expéditeur revendique.
//
// Tout refus rend une erreur qui enveloppe domain.ErrBuildArtifactInvalid et
// n'écrit RIEN.
func StoreArtifact(repoRoot, titleSlug, matchID string, blob []byte) (StoredArtifact, error) {
	doc, err := validateArtifact(titleSlug, matchID, blob)
	if err != nil {
		return StoredArtifact{}, err
	}
	outPath := title.NewPathResolver(repoRoot).ReplayArtifactPath(titleSlug, matchID)
	if err := writeArtifactBytes(outPath, blob); err != nil {
		return StoredArtifact{}, fmt.Errorf("écriture artefact %s: %w", outPath, err)
	}
	return StoredArtifact{
		Path: outPath, MatchID: matchID, Bytes: len(blob),
		Tracks: len(doc.Tracks), SchemaVersion: doc.SchemaVersion,
	}, nil
}

// validateArtifact désérialise et contrôle l'artefact. Rend le document lu.
func validateArtifact(titleSlug, matchID string, blob []byte) (replay.ReplayDocument, error) {
	var doc replay.ReplayDocument
	if len(blob) == 0 {
		return doc, fmt.Errorf("%w (corps vide)", domain.ErrBuildArtifactInvalid)
	}
	if titleSlug == "" || matchID == "" {
		return doc, fmt.Errorf("%w (titre et match du job requis)", domain.ErrBuildArtifactInvalid)
	}
	if err := json.Unmarshal(blob, &doc); err != nil {
		return doc, fmt.Errorf("%w (JSON illisible: %v)", domain.ErrBuildArtifactInvalid, err)
	}
	if doc.SchemaVersion != replay.SchemaVersion {
		return doc, fmt.Errorf("%w (schéma %d, attendu %d)",
			domain.ErrBuildArtifactInvalid, doc.SchemaVersion, replay.SchemaVersion)
	}
	// La comparaison porte sur la forme COURTE : c'est l'identité sous laquelle
	// l'artefact est rangé (ReplayArtifactPath), donc la seule qui décide où le
	// fichier atterrit. Un ouvrier qui rend l'id complet et un job qui porte la
	// forme courte désignent le même match — et le même chemin.
	if title.FilmShortMatchID(doc.MatchID) != title.FilmShortMatchID(matchID) {
		return doc, fmt.Errorf("%w (match %q, attendu %q)",
			domain.ErrBuildArtifactInvalid, doc.MatchID, matchID)
	}
	if doc.TitleSlug != "" && doc.TitleSlug != titleSlug {
		return doc, fmt.Errorf("%w (titre %q, attendu %q)",
			domain.ErrBuildArtifactInvalid, doc.TitleSlug, titleSlug)
	}
	if len(doc.Tracks) == 0 {
		// Même règle que ErrNoTracks à la construction : un document sans
		// trajectoire se servirait comme un rejeu « propre » d'un match vide.
		return doc, fmt.Errorf("%w (aucune trajectoire)", domain.ErrBuildArtifactInvalid)
	}
	return doc, nil
}

// writeArtifactBytes écrit un artefact déjà sérialisé, ATOMIQUEMENT. Source
// unique des deux producteurs (construction locale et dépôt d'un ouvrier) : deux
// façons d'écrire le même fichier finiraient par diverger sur l'atomicité.
func writeArtifactBytes(outPath string, blob []byte) error {
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}
	return atomicfile.WriteFile(outPath, blob, 0o644)
}
