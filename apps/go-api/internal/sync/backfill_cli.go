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
	fs.BoolVar(&scope.EngagementCoefficients, "engagement-coefficients", false, "Recalcule UNIQUEMENT les coefficients d'engagement (mediane glissante des paces, ~5ms par joueur)")
	fs.BoolVar(&scope.ForceEngagementCoefficients, "force-engagement-coefficients", false, "Force le recalcul des coefficients (idempotent en pratique)")
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

	// ── PVE / Firefight (v5.2) ──
	fs.BoolVar(&scope.PVEStats, "pve-stats", false, "Backfill stats PVE (Firefight)")
	fs.BoolVar(&scope.ForcePVEStats, "force-pve-stats", false, "Force backfill PVE")

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

	// ── Assists model — OLS per-mode expected_assists ──
	fs.BoolVar(&scope.AssistsModel, "assists-model", false, "Calculer le modèle OLS expected_assists par mode (stats.duckdb)")
	fs.BoolVar(&scope.ForceAssistsModel, "force-assists-model", false, "Recalculer le modèle même si déjà présent")

	return fs, cli, scope
}
