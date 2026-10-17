package domain

import "time"

// AnalysisProbe : ce que le registre et la base du joueur savent de l'avancement de
// l'analyse du film d'UN match. Décodé par le dépôt (jamais de bitmask ici) pour que la
// règle `analysis.IsAnalysisPending` reste une fonction de booléens.
type AnalysisProbe struct {
	// FirstSyncAt : `match_registry.first_sync_at`, instant UTC de la première synchro du
	// match. Nil = inconnu (la règle rend alors faux : pas d'indicateur sans borne d'âge).
	FirstSyncAt *time.Time
	// KillSourceDone : la source du kill est traitée — `MBitWeaponKills` posé, OU film
	// définitivement absent (`MBitFilmAbsent`), OU film lu sans kill-feed
	// (`killsource_sans_killfeed_rev` renseignée : l'analyse a eu lieu, elle n'a rien produit).
	KillSourceDone bool
	// FilmAbsent : `MBitFilmAbsent` — le film est définitivement perdu, aucun rejeu ne viendra.
	FilmAbsent bool
	// HasCitationRows : au moins une ligne dans `match_citations_latest` pour le match
	// (delta, ou sentinelle `_processed` d'un match à zéro citation).
	HasCitationRows bool
}

// Causes d'une analyse non finie, telles que servies dans `analysis_pending_parts`. Les trois
// chaînes sont le contrat avec le web (une section regarde SA cause) : ne pas les renommer
// sans le miroir `lib/analysis-pending/analysisPending.logic.ts`.
const (
	// AnalysisPartKillSource : la source du kill n'est pas traitée (arme par frag, frags assistés).
	AnalysisPartKillSource = "kill_source"
	// AnalysisPartCitations : les citations ne sont pas calculées.
	AnalysisPartCitations = "citations"
	// AnalysisPartReplay : le rejeu 2D est absent alors que le film n'est pas déclaré absent.
	AnalysisPartReplay = "replay"
)

// Causes pour lesquelles une section tirée du film n'a rien à montrer sur le périmètre d'une page, telles
// que servies à côté du bloc manquant (le web choisit entre l'indicateur d'analyse et l'état vide).
const (
	// FilmGapPending : l'analyse du film d'au moins un match du périmètre est EN COURS.
	FilmGapPending = "pending"
	// FilmGapMissing : le film du périmètre n'est ni lu ni en cours d'analyse.
	FilmGapMissing = "missing"
)
