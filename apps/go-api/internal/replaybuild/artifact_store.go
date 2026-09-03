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
// ET UN QUATRIÈME CAS, QUI N'EST NI UN REFUS NI UNE ÉCRITURE : l'artefact est ACCEPTÉ mais
// RIEN N'EST ÉCRIT, parce qu'il rétrograderait celui déjà en place (compteurs de joueur
// présents sur le disque, absents dans le candidat, même schéma). L'appelant reçoit un accusé
// décrivant CE QUI RESTE RANGÉ, un WARN est journalisé et un compteur monte.
//
// Ce n'est pas une erreur de protocole : l'expéditeur a bien travaillé, avec ce qu'on lui
// avait donné. Le lui rendre en 400 lui ferait perdre son bail et re-décoder deux fois de plus
// pour le même résultat. Le garde ne vit PAS dans ce fichier-ci mais au point d'écriture
// (`writeArtifactBytes`), que ce dépôt partage avec les trois autres écrivains canoniques.
//
// L'ÉCRITURE EST ATOMIQUE (temporaire puis renommage, via platform/atomicfile) :
// le service de lecture sert le fichier tel quel, sans verrou ni version — une
// écriture en place lui ferait servir un artefact à moitié écrit à qui consulte
// pendant le dépôt.
package replaybuild

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/observability"
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
	if _, err := validateArtifact(titleSlug, matchID, blob); err != nil {
		return StoredArtifact{}, err
	}
	outPath := title.NewPathResolver(repoRoot).ReplayArtifactPath(titleSlug, matchID)
	// Le garde anti-régression n'est PAS ici : il vit au point d'écriture, que ce dépôt
	// partage avec les trois autres écrivains (cf. writeArtifactBytes). L'accusé décrit ce
	// que le disque porte APRÈS l'appel — donc l'artefact conservé quand l'écriture est
	// refusée, et le nouveau sinon.
	surDisque, err := writeArtifactBytes(outPath, titleSlug, matchID, blob)
	if err != nil {
		return StoredArtifact{}, fmt.Errorf("écriture artefact %s: %w", outPath, err)
	}
	return StoredArtifact{
		Path: outPath, MatchID: matchID, Bytes: surDisque.Bytes,
		Tracks: surDisque.Tracks, SchemaVersion: surDisque.SchemaVersion,
	}, nil
}

// wouldDowngrade dit si `blob` RÉTROGRADERAIT l'artefact déjà en place : même version de
// schéma, mais des compteurs de joueur présents sur le disque et absents dans le candidat.
// Rend aussi le digest de ce qui est en place, pour que l'appelant sache quoi rapporter.
//
// LE GARDE NE MONTE PAS DE VERSION : à schéma DIFFÉRENT il se tait. Une montée de schéma doit
// TOUJOURS passer — c'est une reconstruction voulue, et un artefact d'un autre schéma ne se
// compare pas à celui-ci.
func wouldDowngrade(outPath string, blob []byte) (enPlace Digest, oui bool) {
	current, ok := ArtifactDigest(outPath)
	if !ok || current.Players == 0 {
		return Digest{}, false // rien à protéger
	}
	incoming, ok := digestFromBytes(blob)
	if !ok || incoming.Players > 0 || incoming.SchemaVersion != current.SchemaVersion {
		return current, false
	}
	return current, true
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

// writeArtifactBytes écrit un artefact déjà sérialisé, ATOMIQUEMENT — SAUF s'il RÉTROGRADERAIT
// celui déjà en place. Rend le digest de ce que le disque porte APRÈS l'appel : le nouvel
// artefact s'il a été écrit, l'ancien s'il a été conservé.
//
// POURQUOI LE GARDE EST ICI, ET NULLE PART AILLEURS. Quatre écrivains canoniques visent ce même
// fichier : le dépôt d'un ouvrier (`StoreArtifact`), le fil de l'eau post-sync (`buildAll`), le
// CLI de rattrapage (`levelup backfill-replay`, y compris `--only-existing`) et l'action admin
// (`RunReplayBuild`). Les trois derniers passent par `BuildMatch` -> `writeArtifact` ; tous les
// quatre finissent ICI. C'est le seul endroit qui voit À LA FOIS le document candidat et le
// fichier en place — le garder à l'étage au-dessus laisserait trois portes ouvertes, et en
// poser quatre copies violerait la règle des <= 2 exemplaires.
//
// LE SCÉNARIO QU'IL FERME. Des faits VIDES produisent un `scoreTimeline.players` vide, donc un
// artefact appauvri qui porte le bon numéro de schéma. Il écraserait alors un artefact riche,
// SILENCIEUSEMENT et sans réparation possible (la sélection post-sync ne voit que les matchs
// insérés d'un cycle). Les occasions sont réelles et connues : un job enfilé AVANT le transport
// des faits, et `chargerFaitsReplay` qui dégrade à vide pour TOUTE une passe de backfill si son
// unique ouverture de base échoue.
//
// C'EST AUSSI LE FILET DU PRÉDICAT DE FRAÎCHEUR. `ArtifactHasPlayerCounters` ne peut que
// PRÉSUMER l'appauvrissement (trois vacuités légitimes, cf. son en-tête) : avec ce garde, une
// présomption fausse coûte au pire un décodage gâché, jamais un artefact rétrogradé.
//
// C'EST ENFIN LE POINT D'OBSERVATION de « un rejeu vient de devenir disponible »
// (artifact_events.go) : parce que les quatre écrivains finissent ici, un seul appel suffit
// à couvrir la construction locale COMME la livraison d'un ouvrier. La publication est
// placée APRÈS l'écriture réelle, et NULLE PART AILLEURS : le refus anti-régression
// ci-dessus rend un digest sans erreur alors que RIEN n'a été écrit — annoncer un rejeu
// « prêt » sur ce chemin annoncerait un fichier que personne n'a touché.
//
// titleSlug et matchID sont ceux de l'APPELANT (identité du job / du registre), pas ceux du
// document : cf. ArtifactStored.
func writeArtifactBytes(outPath, titleSlug, matchID string, blob []byte) (Digest, error) {
	if enPlace, oui := wouldDowngrade(outPath, blob); oui {
		// Jamais muet : un artefact non écrit doit s'expliquer, sinon l'admin verra une
		// construction « réussie » sans comprendre pourquoi le fichier n'a pas changé.
		slog.Warn("replaybuild: écriture d'artefact REFUSÉE — elle rétrograderait celui en place "+
			"(compteurs de joueur présents sur disque, absents dans le candidat, même schéma)",
			"match_id", enPlace.MatchID, "path", outPath, "joueurs_en_place", enPlace.Players,
			"schema", enPlace.SchemaVersion, "octets_refuses", len(blob))
		observability.IncCounter("replay_artifact_downgrade_refused_total")
		return enPlace, nil
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return Digest{}, err
	}
	if err := atomicfile.WriteFile(outPath, blob, 0o644); err != nil {
		return Digest{}, err
	}
	ecrit, _ := digestFromBytes(blob)
	publishArtifactStored(ArtifactStored{
		TitleSlug: titleSlug, MatchID: matchID, Path: outPath,
		Bytes: ecrit.Bytes, Tracks: ecrit.Tracks, SchemaVersion: ecrit.SchemaVersion,
	})
	return ecrit, nil
}
