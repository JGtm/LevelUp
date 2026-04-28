// Package sync — backfill_cli.go : parsing des arguments CLI pour le backfill.
//
// Portage de scripts/backfill/cli.py (create_argument_parser).
// Utilise flag.FlagSet (stdlib) pour rester léger et testable.
//
// Usage :
//
//	fs, scope := sync.NewBackfillFlagSet()
//	fs.Parse(os.Args[1:])
//	scope.Resolve()
package sync

import (
	"flag"
)

// BackfillCLI regroupe les options CLI non-scope du backfill.
type BackfillCLI struct {
	Player string
	All    bool
}

// NewBackfillFlagSet crée un FlagSet configuré avec tous les arguments backfill.
// Retourne le flagset, les options CLI générales, et le SyncScope lié.
//
// L'appelant doit invoquer scope.Resolve() après fs.Parse().
func NewBackfillFlagSet() (*flag.FlagSet, *BackfillCLI, *SyncScope) {
	fs := flag.NewFlagSet("backfill", flag.ContinueOnError)
	cli := &BackfillCLI{}
	scope := &SyncScope{DetectionMode: "or"}

	// ── Sélection du joueur ──
	fs.StringVar(&cli.Player, "player", "", "Gamertag du joueur (ignoré si --all)")
	fs.BoolVar(&cli.All, "all", false, "Traiter tous les joueurs DuckDB")

	// ── Options générales ──
	fs.BoolVar(&scope.DryRun, "dry-run", false, "Mode dry-run (ne fait que lister)")
	fs.IntVar(&scope.MaxMatches, "max-matches", 0, "Nombre maximum de matchs à traiter par joueur")
	fs.IntVar(&scope.RequestsPerSec, "requests-per-second", 5, "Rate limiting API (défaut: 5)")
	fs.StringVar(&scope.DetectionMode, "detection-mode", "or", "Mode détection: 'or' ou 'and'")

	// ── Types de données ──
	fs.BoolVar(&scope.Medals, "medals", false, "Backfill les médailles")
	fs.BoolVar(&scope.Events, "events", false, "Backfill les highlight events")
	fs.BoolVar(&scope.ForceEvents, "force-events", false, "Force rescan events pour TOUS les matchs")
	fs.BoolVar(&scope.Skill, "skill", false, "Backfill les stats skill/MMR")
	fs.BoolVar(&scope.ForceSkill, "force-skill", false, "Force rescan skill/MMR pour TOUS les matchs")
	fs.BoolVar(&scope.PersonalScores, "personal-scores", false, "Backfill personal score awards")
	fs.BoolVar(&scope.ForcePersonalScores, "force-personal-scores", false, "Force rescan personal scores pour TOUS les matchs")
	fs.BoolVar(&scope.PerformanceScores, "performance-scores", false, "Calculer les scores de performance manquants")
	fs.BoolVar(&scope.ForcePerformanceScores, "force-performance-scores", false, "Force le calcul pour TOUS les matchs")
	fs.BoolVar(&scope.EngagementScores, "engagement-scores", false, "Calculer le score d'engagement par match (Phase 3 du plan engagement)")
	fs.BoolVar(&scope.ForceEngagementScores, "force-engagement-scores", false, "Force le recalcul du score d'engagement pour TOUS les matchs")
	fs.BoolVar(&scope.Aliases, "aliases", false, "Mettre à jour les aliases XUID")
	fs.BoolVar(&scope.AllData, "all-data", false, "Backfill toutes les données")

	// ── Accuracy ──
	fs.BoolVar(&scope.Accuracy, "accuracy", false, "Backfill la précision")
	fs.BoolVar(&scope.ForceAccuracy, "force-accuracy", false, "Force accuracy pour TOUS les matchs")

	// ── Shots ──
	fs.BoolVar(&scope.Shots, "shots", false, "Backfill shots_fired/shots_hit")
	fs.BoolVar(&scope.ForceShots, "force-shots", false, "Force shots pour TOUS les matchs")

	// ── Enemy MMR ──
	fs.BoolVar(&scope.EnemyMMR, "enemy-mmr", false, "Backfill enemy_mmr")
	fs.BoolVar(&scope.ForceEnemyMMR, "force-enemy-mmr", false, "Force enemy_mmr pour TOUS les matchs")

	// ── Assets ──
	fs.BoolVar(&scope.Assets, "assets", false, "Récupérer les noms via Discovery UGC")
	fs.BoolVar(&scope.ForceAssets, "force-assets", false, "Force assets pour TOUS les matchs")

	// ── Aliases force ──
	fs.BoolVar(&scope.ForceAliases, "force-aliases", false, "Force ré-extraction des aliases")
	fs.BoolVar(&scope.ForceMedals, "force-medals", false, "Force rescan TOUS les matchs pour les médailles")

	// ── Participants ──
	fs.BoolVar(&scope.Participants, "participants", false, "Backfill les participants de match")
	fs.BoolVar(&scope.ForceParticipants, "force-participants", false, "Force ré-extraction des participants")
	fs.BoolVar(&scope.ParticipantsScores, "participants-scores", false, "Backfill score et rang des participants")
	fs.BoolVar(&scope.ParticipantsKDA, "participants-kda", false, "Backfill kills, deaths, assists des participants")
	fs.BoolVar(&scope.ParticipantsShots, "participants-shots", false, "Backfill shots des participants")
	fs.BoolVar(&scope.ForceParticipantsShots, "force-participants-shots", false, "Force shots participants")
	fs.BoolVar(&scope.ParticipantsDamage, "participants-damage", false, "Backfill damage des participants")
	fs.BoolVar(&scope.ForceParticipantsDamage, "force-participants-damage", false, "Force damage participants")
	fs.BoolVar(&scope.ParticipantsAvgLife, "participants-avg-life", false, "Backfill avg_life_seconds participants")
	fs.BoolVar(&scope.ForceParticipantsAvgLife, "force-participants-avg-life", false, "Force avg_life participants")

	// ── Killer/victim ──
	fs.BoolVar(&scope.KillerVictim, "killer-victim", false, "Backfill paires killer/victim")

	// ── End time ──
	fs.BoolVar(&scope.EndTime, "end-time", false, "Remplir end_time")
	fs.BoolVar(&scope.ForceEndTime, "force-end-time", false, "Recalculer end_time")

	// ── Sessions ──
	fs.BoolVar(&scope.Sessions, "sessions", false, "Backfill session_id + colonnes amis")
	fs.BoolVar(&scope.ForceSessions, "force-sessions", false, "Recalculer sessions même si rempli")

	// ── Teammates signature ──
	fs.BoolVar(&scope.TeammatesSig, "teammates-sig", false, "Reconstruire teammates_signature")
	fs.BoolVar(&scope.ForceTeammatesSig, "force-teammates-sig", false, "Recalculer teammates_signature")

	// ── Citations ──
	fs.BoolVar(&scope.Citations, "citations", false, "Calculer les citations")
	fs.BoolVar(&scope.ForceCitations, "force-citations", false, "Recalculer les citations")

	// ── Participants enrich ──
	fs.BoolVar(&scope.ParticipantsEnrich, "participants-enrich", false, "Backfill colonnes étendues + MMR")
	fs.BoolVar(&scope.ForceParticipantsEnrich, "force-participants-enrich", false, "Force backfill colonnes étendues")

	// ── Granulaire MMR (v5.2) ──
	fs.BoolVar(&scope.TeamMMR, "team-mmr", false, "Backfill team_mmr si NULL")
	fs.BoolVar(&scope.ForceTeamMMR, "force-team-mmr", false, "Force rechargement team_mmr")
	fs.BoolVar(&scope.MMR, "mmr", false, "= --team-mmr + --enemy-mmr")
	fs.BoolVar(&scope.ForceMMR, "force-mmr", false, "= --force-team-mmr + --force-enemy-mmr")

	// ── Granulaire Expected (v5.2) ──
	fs.BoolVar(&scope.KillsExpected, "kills-expected", false, "Backfill kills_expected/stddev")
	fs.BoolVar(&scope.DeathsExpected, "deaths-expected", false, "Backfill deaths_expected/stddev")
	fs.BoolVar(&scope.AssistsExpected, "assists-expected", false, "Backfill assists_expected/stddev")
	fs.BoolVar(&scope.Expected, "expected", false, "= --kills-expected + --deaths-expected + --assists-expected")
	fs.BoolVar(&scope.ForceKillsExpected, "force-kills-expected", false, "Force kills_expected")
	fs.BoolVar(&scope.ForceDeathsExpected, "force-deaths-expected", false, "Force deaths_expected")
	fs.BoolVar(&scope.ForceAssistsExpected, "force-assists-expected", false, "Force assists_expected")
	fs.BoolVar(&scope.ForceExpected, "force-expected", false, "Force toutes les expected values")

	// ── Combat granulaire (v5.2) ──
	fs.BoolVar(&scope.Damage, "damage", false, "Backfill damage_dealt/damage_taken")
	fs.BoolVar(&scope.AvgLife, "avg-life", false, "Backfill avg_life_seconds")
	fs.BoolVar(&scope.Combat, "combat", false, "= --accuracy + --shots + --damage")
	fs.BoolVar(&scope.ForceDamage, "force-damage", false, "Force damage")
	fs.BoolVar(&scope.ForceAvgLife, "force-avg-life", false, "Force avg_life")
	fs.BoolVar(&scope.ForceCombat, "force-combat", false, "= --force-accuracy + --force-shots + --force-damage")

	// ── Kills détaillés (v5.2) ──
	fs.BoolVar(&scope.GrenadeKills, "grenade-kills", false, "Backfill grenade_kills")
	fs.BoolVar(&scope.MeleeKills, "melee-kills", false, "Backfill melee_kills")
	fs.BoolVar(&scope.PowerWeaponKills, "power-weapon-kills", false, "Backfill power_weapon_kills")
	fs.BoolVar(&scope.HeadshotKills, "headshot-kills", false, "Backfill headshot_kills")
	fs.BoolVar(&scope.MaxSpree, "max-spree", false, "Backfill max_killing_spree")
	fs.BoolVar(&scope.KillsDetail, "kills-detail", false, "= grenade + melee + power_weapon + headshot kills")
	fs.BoolVar(&scope.ForceGrenadeKills, "force-grenade-kills", false, "Force grenade_kills")
	fs.BoolVar(&scope.ForceMeleeKills, "force-melee-kills", false, "Force melee_kills")
	fs.BoolVar(&scope.ForcePowerWeaponKills, "force-power-weapon-kills", false, "Force power_weapon_kills")
	fs.BoolVar(&scope.ForceHeadshotKills, "force-headshot-kills", false, "Force headshot_kills")
	fs.BoolVar(&scope.ForceMaxSpree, "force-max-spree", false, "Force max_spree")
	fs.BoolVar(&scope.ForceKillsDetail, "force-kills-detail", false, "Force tous les kills détaillés")

	// ── Divers granulaires (v5.2) ──
	fs.BoolVar(&scope.KDARecalc, "kda-recalc", false, "Recalcule kda si NULL")
	fs.BoolVar(&scope.TimePlayed, "time-played", false, "Backfill time_played_seconds")
	fs.BoolVar(&scope.ForceKDARecalc, "force-kda-recalc", false, "Force kda recalc")
	fs.BoolVar(&scope.ForceTimePlayed, "force-time-played", false, "Force time_played")

	// ── Core Stats (v5.2) ──
	fs.BoolVar(&scope.CoreStats, "core-stats", false, "= accuracy + shots + damage + avg-life + kills-detail + kda + time-played")
	fs.BoolVar(&scope.ForceCoreStats, "force-core-stats", false, "Force tous les core-stats")

	// ── PVE / Firefight (v5.2) ──
	fs.BoolVar(&scope.PVEStats, "pve-stats", false, "Backfill stats PVE (Firefight)")
	fs.BoolVar(&scope.ForcePVEStats, "force-pve-stats", false, "Force backfill PVE")

	// ── Weapon kills (v5.5) ──
	fs.BoolVar(&scope.Weapons, "weapons", false, "Backfill kills par arme")
	fs.BoolVar(&scope.ForceWeapons, "force-weapons", false, "Force backfill armes")
	fs.BoolVar(&scope.ForceNoFilm, "force-no-film", false, "Re-tenter matchs WEAPON_KILLS_NO_FILM")

	// ── LUSR / CSR (v5.3) ──
	fs.BoolVar(&scope.LUSR, "lusr", false, "Calculer le LUSR (TrueSkill 2 local)")
	fs.BoolVar(&scope.ForceLUSR, "force-lusr", false, "Recalculer LUSR depuis zéro")
	fs.BoolVar(&scope.CSR, "csr", false, "Backfill CSR depuis l'API")
	fs.BoolVar(&scope.ForceCSR, "force-csr", false, "Force re-fetch CSR")
	fs.BoolVar(&scope.FetchCSR, "fetch-csr", false, "Snapshot CSR actuel")
	fs.BoolVar(&scope.SkillRank, "skill-rank", false, "= --lusr + --csr")
	fs.BoolVar(&scope.ForceSkillRank, "force-skill-rank", false, "= --force-lusr + --force-csr")

	// ── Comeback badges (v6.2) ──
	fs.BoolVar(&scope.ComebackBadges, "comeback-badges", false, "Calcul Remontada / Débandade")
	fs.BoolVar(&scope.ForceComebackBadges, "force-comeback-badges", false, "Re-traiter les matchs déjà badgés")

	// ── Playable duration (v6.3) ──
	fs.BoolVar(&scope.PlayableDuration, "playable-duration", false, "Backfill playable_duration_seconds")
	fs.BoolVar(&scope.ForcePlayableDuration, "force-playable-duration", false, "Re-traiter même si déjà rempli")

	return fs, cli, scope
}
