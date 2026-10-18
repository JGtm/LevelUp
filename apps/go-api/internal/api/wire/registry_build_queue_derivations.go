// Package api — registry_build_queue_derivations.go : les DÉRIVATIONS d'un artefact déposé par
// un ouvrier (constat A1) — le provider shared du titre, ses segments de lecture et d'écriture
// bornés pour un handler HTTP, et l'appel au point d'entrée unique replayartifacts.Deriver.
package wire

import (
	"context"
	"database/sql"
	"time"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/platform/duckdb/sharedprovider"
	syncpkg "levelup/go-api/internal/sync"
	"levelup/go-api/internal/sync/replayartifacts"
)

// acquireWriterDepot borne l'attente du writer shared SUR LE CHEMIN DU DÉPÔT D'OUVRIER.
//
// ELLE EST PLUS COURTE QUE `acquireWriterTimeout` (60 s), ET C'EST LE CONSTAT C7 DE LA REVUE
// A-R1 : les actions admin sont des commandes d'opérateur qui peuvent attendre un cycle de sync
// ; le dépôt, lui, est un HANDLER HTTP, et le serveur ferme l'écriture à 30 s
// (`cmd/server/main.go`, `WriteTimeout`). Une attente qui dépasse ce budget fait couper la
// connexion et enregistrer à l'ouvrier un ÉCHEC DE DÉPÔT pour un artefact pourtant rangé —
// qu'il rejouera, produisant une passe de plus dans chaque table append-only.
//
// 8 s = de quoi laisser passer un burst writer du post-sync (bornés à cinq matchs) tout en
// gardant, avec les écritures elles-mêmes, une marge confortable sous les 30 s. Au-delà, la
// dérivation est ABANDONNÉE et journalisée : le rattrapage (derivations_backlog.go) la
// rejouera au prochain cycle, et la marque n'est pas posée (constat C1) — c'est précisément ce
// qui rend cet abandon sûr. Le garde-rail `build_queue_writer_budget_test.go` refuse que cette
// valeur repasse au-dessus du `WriteTimeout` du serveur.
const acquireWriterDepot = 8 * time.Second

// deriverArtefactRange applique les dérivations post-rangement à UN artefact déposé par un
// ouvrier, par LE MÊME point d'entrée que la cuisson locale (`replayartifacts.Deriver`).
//
// POURQUOI SYNCHRONE, ET POURQUOI C'EST TENABLE. L'ouvrier dépose ses artefacts un par un ; la
// dérivation lit un document déjà sur disque et ouvre UN SEUL segment writer court, relâché
// aussitôt — une acquisition pour les quatre familles depuis la correction du constat C7
// (`replayartifacts.writerUnique`), bornée par [acquireWriterDepot]. Le temps de réponse du
// dépôt est donc majoré par cette borne plus le coût des écritures d'UN match, très en dessous
// du `WriteTimeout` du serveur.
//
// POURQUOI PAS ASYNCHRONE APRÈS RÉPONSE (choix documenté). Une goroutine détachée demanderait
// un `context.WithoutCancel` (le contexte de la requête meurt avec elle), une borne sur le
// nombre de dérivations en vol, et un arrêt propre au shutdown — trois états à surveiller pour
// gagner quelques secondes sur un chemin qui n'est pas celui d'un utilisateur. Le synchrone
// borné donne la même garantie de réponse sans aucun de ces états.
//
// BEST-EFFORT STRICT : aucune erreur ne remonte. L'artefact est RANGÉ — c'est le produit ; les
// dérivations ne sont que sa projection, et le rattrapage (derivations_backlog.go) les
// rejouera. Un writer indisponible se journalise dans les projections elles-mêmes, et la marque
// de dérivation n'est PAS posée (constat C1) : le match reste candidat au rattrapage.
func (r *ServiceRegistry) deriverArtefactRange(ctx context.Context, titleSlug, matchID, path string) {
	ranges := []replayartifacts.ArtefactRange{{MatchID: matchID, Path: path}}
	if r.replayDerivationsFn != nil {
		r.replayDerivationsFn(ctx, titleSlug, ranges)
		return
	}
	provider := r.sharedProviderForTitle(titleSlug)
	replayartifacts.Deriver(ctx, replayartifacts.DerivationsDeps{
		RepoRoot:  r.cfg.RepoRoot,
		TitleSlug: titleSlug,
		// Gamertag VIDE, et c'est exact : un dépôt d'ouvrier ne travaille pour aucun joueur
		// en particulier. Le champ ne sert qu'au journal.
		WithRead:      sharedReadDepot(provider, titleSlug),
		AcquireWriter: r.sharedWriterForTitle(provider, titleSlug),
	}, ranges)
}

// sharedProviderForTitle résout le provider shared DU TITRE. Nil quand aucun n'est câblé, ou
// quand celui du titre est introuvable : on REFUSE alors de lire et d'écrire plutôt que de
// toucher le shared d'un autre titre.
//
// LE PROVIDER EST RÉSOLU PAR TITRE, JAMAIS `cfg.SharedProvider` NU : celui-ci est le provider
// du titre par DÉFAUT (B-swap), et un artefact d'un autre titre écrirait dans le mauvais
// fichier. Le Manager déduplique par chemin — pour le titre par défaut il rend le MÊME
// provider, sans ouvrir la moindre connexion supplémentaire.
func (r *ServiceRegistry) sharedProviderForTitle(titleSlug string) sharedprovider.Provider {
	if r.cfg.SharedManager == nil {
		return r.cfg.SharedProvider
	}
	p, err := r.cfg.SharedManager.For(
		config.SharedDBPath(r.cfg, titleSlug), r.cfg.UserTimezone)
	if err != nil {
		monitoringLog.Warn("build queue: provider shared du titre introuvable — dérivations non persistées",
			"title", titleSlug, "err", err)
		return nil
	}
	return p
}

// sharedWriterForTitle rend l'acquisition d'un writer shared POUR CE TITRE, sous la forme que
// les dérivations attendent. Nil quand rien ne peut écrire (aucun provider) : les projections
// le journalisent alors et ne persistent rien.
//
// LE CHEMIN EST CELUI DE config.SharedDBPath (lot B5.4 du backlog 2026-09-26) : en démo, la
// base de la fixture — la même que les lecteurs et les 4 bases du boot —, jamais le shared
// réel du dépôt.
//
// L'ACQUISITION EST BORNÉE PAR [acquireWriterDepot], et non par `acquireWriterTimeout` : ce
// chemin-ci vit dans un handler HTTP dont le serveur ferme l'écriture à 30 s (constat C7).
func (r *ServiceRegistry) sharedWriterForTitle(
	provider sharedprovider.Provider, titleSlug string,
) func(context.Context) (*sql.DB, func(), error) {
	if provider == nil {
		return nil
	}
	return func(ctx context.Context) (*sql.DB, func(), error) {
		acquireCtx, cancel := context.WithTimeout(ctx, acquireWriterDepot)
		defer cancel()
		return syncpkg.AcquireSharedWriterStandalone(
			ctxkeys.WithDBWriterLabel(acquireCtx, "replay_derivations"),
			provider, config.SharedDBPath(r.cfg, titleSlug))
	}
}

// sharedReadDepot rend le segment de LECTURE shared du titre que les dérivations empruntent
// (identité des matchs des niveaux d'armes, lot L4.1 : son absence vidait la table). Nil sans
// provider — les niveaux ne sont alors ni projetés ni marqués, et c'est journalisé.
//
// L'ATTENTE D'UN SWAP EST BORNÉE PAR [acquireWriterDepot], comme celle du writer, et pour la
// même raison (constat C7) : les deux attentes sont successives — la lecture a lieu AVANT la
// première écriture — et leur somme est ce que borne `build_queue_writer_budget_test.go`.
func sharedReadDepot(provider sharedprovider.Provider, titleSlug string) func(context.Context, string, func(*sql.DB)) {
	if provider == nil {
		return nil
	}
	return func(ctx context.Context, step string, fn func(*sql.DB)) {
		db, release, err := provider.Get(sharedprovider.WithSwapWaitBudget(ctx, acquireWriterDepot))
		if err != nil {
			monitoringLog.WarnContext(ctx, "build queue: lecture shared indisponible — étape de dérivation sautée",
				"title", titleSlug, "step", step, "err", err)
			return
		}
		defer release()
		fn(db)
	}
}
