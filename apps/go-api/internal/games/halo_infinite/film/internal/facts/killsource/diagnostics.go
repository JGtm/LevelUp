package killsource

import "levelup/go-api/internal/games/halo_infinite/film/internal/constat"

// diagnostics.go — CE QUE LE DECODAGE DU KILL-FEED CONSTATE, RENDU A L ORCHESTRATEUR (lot J12.3,
// ADR 0034 D-4).
//
// Ce paquet ne journalise plus : chaque constat devient un [constat.Diagnostic] note dans le
// `decodeCtx`, rendu par [Result.Diagnostics], et journalise par l orchestrateur qui a appele
// [Decode] (`replaybuild`, `sync/killcollector`, les outils) avec SON contexte
// (`replay.JournaliserDiagnostics`).

// Les diagnostics de `killsource`.
const (
	// DiagRechercheSansCarte : instrument de recherche sans carte, largeurs d axe par defaut.
	DiagRechercheSansCarte constat.Code = "killsource.recherche_sans_carte"
	// DiagControleCorruptionAbsent : le film ne declare pas son controle de corruption.
	DiagControleCorruptionAbsent constat.Code = "killsource.controle_corruption_absent"
	// DiagVersionIllisible : version de film illisible, decoupage historique du kill-feed.
	DiagVersionIllisible constat.Code = "killsource.version_illisible"
	// DiagTableNonLue : la table des joueurs du film n a pas ete lue.
	DiagTableNonLue constat.Code = "killsource.table_non_lue"
	// DiagLienParMotif : desaccords, absents ou tueurs ecartes du lien par motif de xuid.
	DiagLienParMotif constat.Code = "killsource.lien_par_motif"
	// DiagBotsNonEpingles : des bots n ont pas pu etre epingles a un siege.
	DiagBotsNonEpingles constat.Code = "killsource.bots_non_epingles"
)

// signaler note un diagnostic du decodage.
func (c *decodeCtx) signaler(code constat.Code, niveau constat.Niveau, msg string, attrs ...any) {
	c.diag.Signaler(constat.Diagnostic{Code: code, Niveau: niveau, Message: msg, Attrs: attrs})
}
