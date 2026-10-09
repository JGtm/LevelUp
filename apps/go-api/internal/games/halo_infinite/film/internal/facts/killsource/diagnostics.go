package killsource

import (
	"levelup/go-api/internal/games/halo_infinite/film/internal/constat"
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
)

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
	// DiagEquipesDeBots : des bots n ont pas d equipe lue dans BOT_METADATA (botmeta_equipe.go).
	DiagEquipesDeBots constat.Code = "killsource.equipes_de_bots"
	// DiagEntreesDeBots : des paquets BOT_METADATA ne ferment pas sous la grammaire de l ecrivain, la
	// marche et le lecteur historique ne lisent pas les memes entrees, ou des entrees sont refusees
	// (jumeau discordant, equipe hors domaine) — meme quand chaque bot a son equipe par un autre
	// paquet.
	DiagEntreesDeBots constat.Code = "killsource.entrees_de_bots"
	// DiagDecoupageMPPNonResolu : le decoupage du bloc MPP du film n est pas resolu (format inconnu,
	// declaration absente ou discordante) ; la marche le lit sous le decoupage du profil.
	DiagDecoupageMPPNonResolu constat.Code = "killsource.decoupage_mpp_non_resolu"
)

// signaler note un diagnostic du decodage.
func (c *decodeCtx) signaler(code constat.Code, niveau constat.Niveau, msg string, attrs ...any) {
	c.diag.Signaler(constat.Diagnostic{Code: code, Niveau: niveau, Message: msg, Attrs: attrs})
}

// signalerLesEquipesDesBots dit, en ERREUR, chaque bot dont l equipe n est pas lue : aucun repli ne
// la remplace, et son entree de rejeu n aura d equipe que si une entite `ti=9` la donne. Les
// paquets qui ne ferment pas, les desaccords entre les deux lecteurs du paquet et les entrees refusees
// se disent en AVERTISSEMENT, meme quand chaque bot tient son equipe d un autre paquet.
func (c *decodeCtx) signalerLesEquipesDesBots(e EquipesDesBots, build string) {
	if e.aLire() {
		c.signaler(DiagEquipesDeBots, constat.NiveauError, "killsource: equipe de bot(s) NON LUE dans BOT_METADATA "+
			"— aucun repli", "film", c.name, "build", build, "illisibles", e.Illisibles,
			"contradictoires", e.Contradictoires, "hors_grammaire", e.HorsGrammaire,
			"paquets_non_fermes", e.PaquetsNonFermes, "perso_inconnue", e.PersoInconnue)
	}
	if e.PaquetsNonFermes > 0 || e.EntreesHorsBalayage > 0 || e.JumeauxDiscordants > 0 || e.HorsDomaine > 0 {
		c.signaler(DiagEntreesDeBots, constat.NiveauWarn, "killsource: paquet(s) BOT_METADATA non ferme(s), ou "+
			"entree(s) hors du lecteur historique ou refusee(s)", "film", c.name,
			"paquets_non_fermes", e.PaquetsNonFermes, "hors_balayage", e.EntreesHorsBalayage,
			"jumeaux_discordants", e.JumeauxDiscordants, "hors_domaine", e.HorsDomaine)
	}
}

// signalerLeDecoupageMPP dit, en AVERTISSEMENT, un film dont le decoupage du bloc MPP n est pas
// resolu ([grammar.ResolutionMPP.Decide] faux) : la marche le lit sous le decoupage du profil, qui
// n est celui d aucune declaration du film.
func (c *decodeCtx) signalerLeDecoupageMPP(res grammar.ResolutionMPP) {
	if res.Decide() {
		return
	}
	c.signaler(DiagDecoupageMPPNonResolu, constat.NiveauWarn, "killsource: decoupage MPP du film non resolu "+
		"— decoupage du profil garde", "film", c.name, "format", res.FormatVersion,
		"format_inconnu", res.FormatInconnu, "records", res.Declaration.Records,
		"discordants", res.Declaration.Discordants)
}
