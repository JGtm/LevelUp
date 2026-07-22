package domain

// admin_lusr_gaps.go — DTO du panneau monitoring « Notes LUSR — trous & garde-fou ».
// Modèle : AdminWeaponCoverage (couverture % + top offenders). Alimenté par
// ServiceRegistry.LUSRGapsReport (scan par joueur via skill.ScanLUSRGaps).

// AdminLUSRGaps agrège l'état des trous LUSR d'un titre (tous joueurs suivis).
type AdminLUSRGaps struct {
	TitleSlug   string `json:"title_slug"`
	GeneratedAt string `json:"generated_at"`
	// Agrégats tous joueurs.
	EligibleTotal      int     `json:"eligible_total"`
	RatedTotal         int     `json:"rated_total"`
	InteriorGapsTotal  int     `json:"interior_gaps_total"`  // trous permanents (sous watermark)
	PendingRecentTotal int     `json:"pending_recent_total"` // en attente (au-dessus watermark)
	CoveragePercent    float64 `json:"coverage_percent"`     // rated / eligible (0..100)
	// Par joueur (les plus impactés d'abord).
	Players []LUSRGapPlayer `json:"players"`
	// Santé du garde-fou (compteurs expvar ré-exposés + horodatages).
	Guardrail LUSRGuardrailHealth `json:"guardrail"`
}

// LUSRGapPlayer : état LUSR d'un joueur suivi.
type LUSRGapPlayer struct {
	PlayerSlug    string `json:"player_slug"`
	Gamertag      string `json:"gamertag"`
	XUID          string `json:"xuid"`
	Eligible      int    `json:"eligible"`
	Rated         int    `json:"rated"`
	InteriorGaps  int    `json:"interior_gaps"`
	PendingRecent int    `json:"pending_recent"`
	// TopGaps : échantillon des trous d'intérieur (max 10), plus anciens d'abord.
	TopGaps []LUSRGapItem `json:"top_gaps"`
	// CheckError : non vide si le scan de ce joueur a échoué (DB inaccessible, etc.).
	CheckError string `json:"check_error,omitempty"`
}

// LUSRGapItem : un match sans note LUSR (trou d'intérieur).
type LUSRGapItem struct {
	MatchID   string `json:"match_id"`
	Group     string `json:"group"`      // chaîne LUSR (playlist_group)
	Playlist  string `json:"playlist"`   // libellé playlist source (pair_name)
	StartTime string `json:"start_time"` // RFC3339 UTC
}

// LUSRGuardrailHealth : santé du garde-fou (compteurs LUSR v2 + horodatages).
type LUSRGuardrailHealth struct {
	// InteriorGapsGauge : dernier total publié par le cron data_health (peut différer
	// du scan à la demande ci-dessus si le cron n'a pas encore retourné).
	InteriorGapsGauge int64 `json:"interior_gaps_gauge"`
	// HeldWatermark : cumul depuis le boot des écritures canonical tenues (watermark
	// non avancé, retry) — croissance soutenue = player DB durablement en échec.
	HeldWatermark int64 `json:"held_watermark"`
	// OwnerMissing : cumul depuis le boot des matchs où l'owner était absent des
	// rosters (désync data). Doit rester 0.
	OwnerMissing int64 `json:"owner_missing"`
	// LastAuditAt : dernier cycle data_health ayant scanné les trous (RFC3339, vide
	// si jamais couru).
	LastAuditAt string `json:"last_audit_at,omitempty"`
}

// AdminLUSRRecomputeResponse : accusé d'un replay LUSR déclenché manuellement.
type AdminLUSRRecomputeResponse struct {
	Gamertag string `json:"gamertag"`
	XUID     string `json:"xuid"`
	Updated  int    `json:"updated"` // # de lignes LUSR (re)écrites par le replay
	OK       bool   `json:"ok"`
}
