package replayartifacts

// journal.go — CE QUE L'ÉTAPE 1.58 DIT D'ELLE-MÊME.
//
// # POURQUOI CE FICHIER EXISTE
//
// L'étape était MUETTE. Sur les 222 matchs des 90 derniers jours, UN SEUL portait un
// artefact, et le journal de synchronisation ne contenait pas une ligne « rejeu 2D » —
// mesure du 2026-09-01. Le cycle du 2026-08-27 à 23h02 a inséré 7 matchs, l'étape 1.55 les
// a traités la minute même avec le film disponible (`no_film: 0`), et l'étape 1.58 n'a rien
// dit du tout.
//
// La raison est structurelle : `Run` avait SEPT sorties sans trace au niveau INFO (placement
// éteint, pas de segment de lecture, client sans chunks, sélection vide, titre sans
// catalogue, et le résumé lui-même conditionné à `built > 0 || filmsSaved > 0`). Aucune de
// ces sorties ne se distinguait de « l'étape n'a jamais tourné » — c'est exactement
// l'ambiguïté qui a coûté cinq mois à l'étape 1.57 (cf. killcollector/postsync.go).
//
// # LA RÈGLE, LA MÊME QUE POUR 1.57
//
// Les compteurs portent le « ça a tourné », les journaux portent les incidents. Un compteur
// publié MÊME À ZÉRO se distingue d'une clé absente de /debug/vars ; une ligne de journal
// par CYCLE (jamais par match) dit ce que le cycle a fait et pourquoi il n'a rien fait.

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/replaybuild"
)

// Compteurs expvar de l'étape (ADR 0009 : entiers, snake_case, aucun ratio). Ils sont
// publiés PAR TITRE (`observability.*T`), comme le reste des compteurs de ce paquet.
const (
	// CompteurCycles : cycles où l'étape a effectivement travaillé (placement armé,
	// segment de lecture disponible). Zéro alors que la synchronisation tourne = l'étape
	// est éteinte ou désarmée, et c'est la première chose à regarder.
	CompteurCycles = "postsync_replay_cycles_total"
	// CompteurSelectionnes : matchs retenus par la sélection, avant plafond de cycle.
	CompteurSelectionnes = "postsync_replay_selectionnes_total"
	// CompteurConstruits : artefacts effectivement écrits.
	CompteurConstruits = "postsync_replay_artifacts_built_total"
	// CompteurFilmsPersistes : films archivés au cache disque. Ils EXPIRENT côté serveur
	// Halo (~29 % du corpus déjà perdu) : cette valeur-là est irremplaçable.
	CompteurFilmsPersistes = "postsync_replay_films_persisted_total"
	// CompteurDejaAJour : matchs sautés parce que leur artefact est déjà courant ET complet.
	CompteurDejaAJour = "postsync_replay_deja_a_jour_total"
	// CompteurEnfiles : matchs mis dans la file durable (placement « worker »).
	CompteurEnfiles = "postsync_replay_jobs_enqueued_total"
	// CompteurAppauvrisReEnfiles : artefacts au bon schéma mais sans compteurs de joueur,
	// remis en file.
	CompteurAppauvrisReEnfiles = "postsync_replay_artifacts_factless_requeued_total"
	// CompteurRetard : JAUGE du retard restant après le travail du cycle. Publiée même à
	// zéro — une clé absente ne se distingue pas d'une étape qui ne tourne pas.
	CompteurRetard = "postsync_replay_backlog_restant"
	// CompteurClientSansChunks : le client injecté ne porte pas GetFilmChunks alors que le
	// placement exige une construction locale. C'EST UN DÉFAUT DE CÂBLAGE, PAS UN ÉTAT
	// NORMAL (pendant de killcollector.CompteurPostSyncClientSansFilm).
	CompteurClientSansChunks = "postsync_replay_client_sans_chunks"
	// CompteurClientSansMvar : cycles ou le client ne portait pas FetchMvarForMap alors que le
	// rattrapage du catalogue de cartes aurait pu travailler. MEME NATURE que
	// CompteurClientSansChunks : un DEFAUT DE CABLAGE, pas un etat normal.
	CompteurClientSansMvar = "postsync_mvar_client_sans_capacite"
	// Bilan du rattrapage des cartes absentes. JAUGES, publiees MEME A ZERO : une cle absente
	// de /debug/vars ne se distingue pas d'une etape qui ne tourne pas, et c'est exactement
	// l'ambiguite qu'un compteur de rattrapage doit fermer.
	JaugeMvarAjoutees      = "postsync_mvar_cartes_ajoutees"
	JaugeMvarDejaLa        = "postsync_mvar_cartes_deja_presentes"
	JaugeMvarSansMapID     = "postsync_mvar_matchs_sans_map_id"
	JaugeMvarHorsObjectifs = "postsync_mvar_cartes_hors_catalogue_objectifs"
	JaugeMvarEchecs        = "postsync_mvar_echecs"
	// Report du coup d'envoi mesure dans le film vers `match_registry` (cf. t0film.go).
	// CompteurT0FilmReportes : lignes de registre corrigees ; CompteurT0FilmDejaLa : matchs
	// deja marques `film_movement` a la meme valeur (la garde a mordu) ; CompteurT0FilmEchecs :
	// writer indisponible, absent du cablage, ou UPDATE refuse — un defaut, jamais un etat
	// normal.
	CompteurT0FilmReportes = "postsync_replay_t0_film_reportes_total"
	CompteurT0FilmDejaLa   = "postsync_replay_t0_film_deja_a_jour_total"
	CompteurT0FilmEchecs   = "postsync_replay_t0_film_echecs_total"
	// Resume d'usage equipement/socles derive des artefacts cuits dans le cycle, ecrit dans
	// `match_usage_films` + `match_usage_players` (cf. usage.go). CompteurUsageEcrits :
	// passes persistees ; CompteurUsageEchecs : artefact illisible, writer indisponible,
	// capabilities illisibles ou INSERT refuse — un defaut, jamais un etat normal (un titre
	// SANS la capability film.usage_summary ne compte rien : silence propre, en DEBUG).
	CompteurUsageEcrits = "postsync_replay_usage_ecrits_total"
	CompteurUsageEchecs = "postsync_replay_usage_echecs_total"
)

// SignalerClientSansChunks journalise et compte l'échec de l'assertion `ChunksFetcher` faite
// par le câblage (`sync.postSyncFilmSteps.runReplayArtifacts`).
//
// ELLE EXISTE PARCE QUE L'ASSERTION JETAIT SON RÉSULTAT. `fetcher, _ := s.client.(...)` :
// un client sans la capacité donnait un fetcher nil, l'étape sortait, et RIEN ne le disait.
// C'est le défaut qui a fait tourner l'étape 1.57 nulle part pendant cinq mois, reproduit
// dans le câblage de l'étape 1.58.
//
// LE NIVEAU DÉPEND DU PLACEMENT, ET CE N'EST PAS UNE COMMODITÉ. Mettre en file ne télécharge
// aucun film (c'est l'ouvrier qui le fera) : sur le chemin « worker » — celui de la
// production — l'absence de la capacité est SANS CONSÉQUENCE, et un avertissement par cycle
// y serait du bruit qui finirait par masquer les vrais. Sur le chemin « local », en
// revanche, elle désarme complètement l'étape : WARN et compteur.
func SignalerClientSansChunks(ctx context.Context, placement replaybuild.Placement, gamertag, typeClient string) {
	if placement != replaybuild.PlacementLocal {
		slog.DebugContext(ctx, "post-sync: rejeu 2D — le client ne porte pas GetFilmChunks (sans effet sur ce placement)",
			"gamertag", gamertag, "client", typeClient, "placement", string(placement))
		return
	}
	observability.IncCounterT(ctxkeys.TitleSlug(ctx), CompteurClientSansChunks)
	slog.WarnContext(ctx, "post-sync: rejeu 2D désarmée — le client ne porte pas GetFilmChunks",
		"gamertag", gamertag, "client", typeClient,
		"consequence", "aucun film archivé ni artefact construit sur ce cycle")
}

// SignalerClientSansMvar dit qu'un client ne porte pas FetchMvarForMap.
//
// IL EXISTE PARCE QUE LE SILENCE A ETE REINTRODUIT UNE FOIS DE PLUS. Le premier jet de ce
// chantier ecrivait `mvarFetcher, _ := s.client.(...)` — une ligne SOUS le commentaire qui
// explique pourquoi cette forme est interdite. L'assertion echouait sur les deux wrappers de
// production, et le rattrapage sortait sans un mot : indistinguable d'un lot non deploye.
func SignalerClientSansMvar(ctx context.Context, gamertag, typeClient string) {
	observability.IncCounterT(ctxkeys.TitleSlug(ctx), CompteurClientSansMvar)
	slog.WarnContext(ctx, "post-sync: rattrapage des cartes DESARME — le client ne porte pas "+
		"FetchMvarForMap",
		"gamertag", gamertag, "client", typeClient,
		"consequence", "aucune carte absente n'entrera au catalogue ; les rejeux de ces cartes "+
			"resteront sans origine `spawner`")
}
