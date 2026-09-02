package replaybuild

// artifact_digest.go — LA LECTURE D'UN ARTEFACT DEJA RANGE : les gardes, et rien d'autre.
//
// Ce fichier ne CONSTRUIT rien. Il porte les fonctions qui INTERROGENT un artefact sur disque
// (est-il a la version de schema courante ? porte-t-il des compteurs de joueur ?) et l'ecriture
// qui le range. Elles vivaient dans `replaybuild.go` et l'avaient pousse au-dela des 500 lignes
// du depot : ce fichier est une SCISSION PURE, aucun corps n'a change, aucun commentaire n'a ete
// touche — seul l'emplacement bouge.

import (
	"encoding/json"
	"os"

	"levelup/go-api/internal/analysis/replay"
)

// ArtifactUpToDate dit si l'artefact au chemin donné existe ET porte la version de schéma
// courante. C'est LA clé de reprise des backfills (cf. replay.SchemaVersion) : un artefact
// d'une version antérieure se lit « à re-cuire », jamais « à jour ». Un fichier illisible
// est traité comme périmé (il sera réécrit), pas comme une erreur.
func ArtifactUpToDate(path string) bool {
	raw, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	var head struct {
		SchemaVersion int `json:"schemaVersion"`
	}
	if err := json.Unmarshal(raw, &head); err != nil {
		return false
	}
	return head.SchemaVersion == replay.SchemaVersion
}

// ArtifactHasPlayerCounters dit si l'artefact au chemin donné PORTE des compteurs de joueur
// (`scoreTimeline.players` non vide). Faux aussi quand le fichier est absent ou illisible.
//
// CE QU'IL AFFIRME, ET CE QU'IL N'AFFIRME PAS. Il constate une PROPRIÉTÉ DU DOCUMENT, il ne
// devine pas comment il a été cuit. L'implication ne vaut que dans un sens :
//
//	compteurs présents  =>  les lignes de match ont été fournies   (sûr)
//	compteurs absents   =>  les lignes de match manquaient          (FAUX en général)
//
// TROIS FAÇONS LÉGITIMES d'être vide MALGRÉ des faits complets, toutes constatées dans le code :
// (a) le film n'a aucun enregistrement d'entité à lire (cas journalisé, `matchfacts.go:70-73`) ;
// (b) l'appariement slot -> joueur échoue — `SlotIdentityFrom` écarte les triplets ambigus, et
// les `COALESCE(..., 0)` de `replay_facts_repo.go` peuvent rendre plusieurs (0,0,0)
// indistinguables ; (c) aucun compteur ne bouge dans la fenêtre lue (`PlayerScore` vide).
//
// UN APPELANT NE DOIT DONC JAMAIS EN DÉDUIRE « à re-cuire » À LUI SEUL. Le vide est une
// PRÉSOMPTION d'appauvrissement, pas une preuve : c'est pour cela que `replayartifacts.enqueueAll`
// exige EN PLUS de tenir des lignes de match (`len(facts.Players) > 0`), et que le rangement
// d'artefact (`StoreArtifact`) refuse de son côté toute régression. Le pire résidu possible est
// alors UN cycle d'ouvrier gâché — jamais un artefact rétrogradé, jamais une boucle qui converge
// vers rien.
//
// POURQUOI CE SIGNAL PLUTÔT QU'UN AUTRE. Mesuré sur deux témoins le 2026-08-24 : 8 joueurs avec
// faits, 0 sans, sur 7344d24f comme sur 530820e5. Les autres candidats sont pires :
// `coverage.score.teamIdentity` vaut légitimement `unresolved` sur 7 des 34 artefacts du cache
// POURTANT cuits avec faits, et `objectives` est vide de plein droit sur un Slayer.
//
// PAS DE CHAMP DÉDIÉ DANS LE DOCUMENT, ET C'EST DÉLIBÉRÉ : un marqueur `factsApplied` — qui
// LUI porterait l'implication dans les deux sens — forcerait un incrément de
// `replay.SchemaVersion`, donc la re-cuisson de tout le cache, aujourd'hui bloquée par la bombe
// RAM de `NamedEventsFrom` (registre du 2026-08-24). C'est la dette assumée de ce choix.
func ArtifactHasPlayerCounters(path string) bool {
	d, ok := readArtifactDigest(path)
	return ok && d.players > 0
}

// artifactDigest : les seules marques d'un artefact que les gardes ont à lire. Une SEULE forme
// de lecture pour tous — deux structures anonymes concurrentes finiraient par diverger sur le
// nom d'un champ, et le garde deviendrait muet sans que rien ne le signale.
type artifactDigest struct {
	matchID       string
	schemaVersion int
	players       int
	tracks        int
	bytes         int
}

// readArtifactDigest lit un artefact sur disque. ok=false si absent ou illisible : dans le
// doute, l'appelant traite l'artefact comme inexploitable, jamais comme bon.
func readArtifactDigest(path string) (artifactDigest, bool) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return artifactDigest{}, false
	}
	return digestFromBytes(raw)
}

// digestFromBytes lit les marques d'un artefact déjà en mémoire (cas du dépôt d'un ouvrier :
// le blob est là, le relire depuis le disque serait absurde).
func digestFromBytes(raw []byte) (artifactDigest, bool) {
	var head struct {
		MatchID       string            `json:"matchId"`
		SchemaVersion int               `json:"schemaVersion"`
		Tracks        []json.RawMessage `json:"tracks"`
		ScoreTimeline struct {
			Players []json.RawMessage `json:"players"`
		} `json:"scoreTimeline"`
	}
	if err := json.Unmarshal(raw, &head); err != nil {
		return artifactDigest{}, false
	}
	return artifactDigest{
		matchID:       head.MatchID,
		schemaVersion: head.SchemaVersion,
		players:       len(head.ScoreTimeline.Players),
		tracks:        len(head.Tracks),
		bytes:         len(raw),
	}, true
}

// writeArtifact sérialise le document et l'écrit ATOMIQUEMENT (cf.
// writeArtifactBytes, artifact_store.go) ; renvoie la taille en octets. Même
// écriture que le dépôt d'un ouvrier : le service de lecture sert le fichier tel
// quel, il ne doit jamais tomber sur un artefact à moitié écrit.
// La taille rendue est celle de ce qui est FINALEMENT sur le disque, pas celle du document
// qu'on voulait écrire : quand le garde anti-régression conserve l'artefact en place, annoncer
// la taille du candidat ferait croire à une écriture qui n'a pas eu lieu.
func writeArtifact(outPath, titleSlug, matchID string, doc replay.ReplayDocument) (int, error) {
	blob, err := json.Marshal(doc)
	if err != nil {
		return 0, err
	}
	surDisque, err := writeArtifactBytes(outPath, titleSlug, matchID, blob)
	if err != nil {
		return 0, err
	}
	return surDisque.bytes, nil
}
