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
