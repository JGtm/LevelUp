package replayartifacts

// buildone.go — LA CONSTRUCTION D'UN ARTEFACT, DELEGUEE HORS DU PROCESSUS DU SERVEUR.
//
// # POURQUOI CE POINT D'INJECTION EXISTE (lot BUILDALL, 2026-08-26)
//
// `buildAll` enchainait jusqu'a `maxPerCycle` films a travers `replaybuild.BuildMatch`, DANS le
// processus du serveur et SANS plafond memoire. C'est la forme exacte du sinistre du
// 2026-08-20 : quatre petits films cuits, effondrement sur le CINQUIEME. Le decodage d'un film
// est un AMPLIFICATEUR — mesure du 2026-08-24 : 7,9 Go en 2,6 s sur `51101d1d`.
//
// # LE DECOUPAGE DES RESPONSABILITES, ET IL N'EST PAS NEGOCIABLE
//
//	le PARENT (serveur)  toutes les E/S BASE (les faits du match — c'est leger) ET l'ECRITURE
//	                     FINALE de l'artefact, via `replaybuild.StoreArtifact`.
//	l'ENFANT             AUCUNE base. Il lit les chunks du film au cache DISQUE, decode,
//	                     construit, et rend les OCTETS. C'est la partie qui explose.
//
// DEUX RAISONS, ET CHACUNE SUFFIRAIT.
//
// (1) LE MODELE MONO-PROCESSUS DUCKDB. Un enfant lecteur pendant que le parent tient la base en
// ecriture est precisement ce que la doctrine interdit (ADR 0013/0016). L'enfant ne doit donc
// RIEN savoir de la base — d'ou les faits qui lui sont TRANSMIS, deja lus par le parent.
//
// (2) LA NOTIFICATION RESTE OU ELLE EST. Le puits d'evenements « artefact range »
// (`SetArtifactStoredSink`) n'est cable qu'au boot du SERVEUR, et `replaybuild/artifact_events.go`
// en tire la consequence noir sur blanc : un process qui ne le cable pas « ne produit AUCUN
// message ». Si l'ENFANT ecrivait l'artefact, la notification Discord des rejeux post-sync
// disparaitrait — sans rien casser de visible, le pire des defauts. En gardant l'ecriture chez le
// parent, le garde anti-regression ET la publication restent exactement ou ils etaient.
//
// C'EST LE PATRON DU DEPOT DE L'OUVRIER, ET CE N'EST PAS UNE COINCIDENCE : un processus tiers
// produit les octets, le serveur les range par `StoreArtifact`. Le post-sync devient
// structurellement identique, avec un enfant local a la place d'un ouvrier distant.

import (
	"context"
	"errors"

	"levelup/go-api/internal/port"
	"levelup/go-api/internal/replaybuild"
	"levelup/go-api/internal/replaychild"
)

// BuildOneRequest : tout ce qu'il faut pour construire UN artefact, et rien qui vienne d'une
// base — l'enfant n'en ouvre aucune.
type BuildOneRequest struct {
	// MatchID est l'identite CANONIQUE du match (celle du registre), pas celle que le document
	// revendiquera : c'est elle qui sert de cle a l'ecriture et a la notification.
	MatchID string
	// TitleSlug et RepoRoot voyagent AVEC la requete : l enfant monte son propre constructeur
	// et n herite d aucun etat du parent.
	TitleSlug string
	RepoRoot  string
	// MapNames sont les identites de carte candidates, dans l'ordre de preference.
	MapNames []string
	// FilmDir est le repertoire des chunks DEJA persistes au cache disque.
	FilmDir string
	// Facts sont les faits du match, LUS PAR LE PARENT. Ils sont serialisables par
	// construction (le type porte ses etiquettes JSON) et minuscules — au plus les lignes des
	// joueurs, deux scores et deux chaines.
	Facts port.MatchFacts
}

// BuildOneFunc construit l'artefact d'UN film et rend ses OCTETS SERIALISES, sans les ecrire.
//
// L'IMPLEMENTATION DE PRODUCTION DECODE HORS DU PROCESSUS (un enfant borne : plafond memoire
// dur, priorite CPU basse, cf. `internal/filmproc`). Ce paquet ne connait pas les processus, et
// c'est voulu : `internal/sync` orchestre une synchronisation, il n'a pas a savoir lancer des
// binaires. Les tests injectent une fonction pure.
//
// Une erreur est un echec de CE film, jamais du cycle : l'appelant journalise et passe au
// suivant.
type BuildOneFunc func(ctx context.Context, req BuildOneRequest) ([]byte, error)

// ErrNoBuilder : aucune stratégie de construction n'est câblée. Le pont disque continue, la
// cuisson est sautée — jamais remplacée par un décodage in-process.
var ErrNoBuilder = errors.New("aucune construction hors processus câblée")

// buildAndStoreOne délègue la construction d'UN film puis RANGE ses octets.
//
// LES DEUX MOITIÉS SONT ICI, ET DANS CET ORDRE, parce que c'est le découpage qui porte tout le
// lot : décoder ailleurs, écrire ici. Séparer les deux appels chez l'appelant laisserait croire
// qu'on peut ranger sans avoir délégué, ou déléguer sans ranger.
func buildAndStoreOne(ctx context.Context, d Deps, w buildWork, filmDir string,
) (replaybuild.StoredArtifact, error) {
	if d.BuildOne == nil {
		return replaybuild.StoredArtifact{}, ErrNoBuilder
	}
	blob, err := d.BuildOne(ctx, BuildOneRequest{
		MatchID: w.matchID, TitleSlug: d.TitleSlug, RepoRoot: d.RepoRoot,
		MapNames: w.mapNames, FilmDir: filmDir, Facts: w.facts,
	})
	if err != nil {
		return replaybuild.StoredArtifact{}, err
	}
	// L'ÉCRITURE RESTE CHEZ LE PARENT : `StoreArtifact` valide, applique le garde
	// anti-régression et publie l'événement « artefact rangé » — les trois au même endroit
	// qu'avant ce lot.
	return replaybuild.StoreArtifact(d.RepoRoot, d.TitleSlug, w.matchID, blob)
}

// SpawnBuildOne est la strategie de PRODUCTION : elle delegue a un enfant borne (plafond
// memoire dur, priorite CPU basse) qui rend les octets sans rien ecrire.
//
// ELLE VIT ICI ET NON DANS LE PAQUET DU LANCEUR pour que la dependance reste a SENS UNIQUE :
// ce paquet-ci definit le contrat et connait le lanceur ; le lanceur, lui, ne connait rien de
// la synchronisation. L'inverse ferait un cycle d'import.
func SpawnBuildOne(ctx context.Context, req BuildOneRequest) ([]byte, error) {
	return replaychild.Spawn(ctx, replaychild.Request{
		MatchID:   req.MatchID,
		TitleSlug: req.TitleSlug,
		RepoRoot:  req.RepoRoot,
		MapNames:  req.MapNames,
		FilmDir:   req.FilmDir,
		Facts:     req.Facts,
	})
}
