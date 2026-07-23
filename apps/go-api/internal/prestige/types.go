package prestige

import "time"

// ---------- Challenge ----------

// Challenge représente un défi individuel — libre ou piloté, dans un arc ou non.
//
// Persistance : table `challenge` dans stats.duckdb (par joueur).
// Le palier (Tier) est calculé à la création/édition via palier.go et figé
// pour les modes pilote, recalculé à chaque édition pour les modes libres.
type Challenge struct {
	ID         string `json:"id"`
	UserID     string `json:"user_id"`
	TitleSlug  string `json:"title_slug"`
	ArcID      string `json:"arc_id,omitempty"`      // vide si défi standalone
	Position   int    `json:"position,omitempty"`    // ordre dans l'arc (>=1) ; 0 si standalone
	TemplateID string `json:"template_id,omitempty"` // vide si défi 100 % libre

	Metric          string  `json:"metric"`                      // FieldKey canonique
	Target          float64 `json:"target"`                      // valeur cible exposée au joueur
	TargetPerMember float64 `json:"target_per_member,omitempty"` // pour défis collectifs

	WindowType  WindowType `json:"window_type"`
	WindowValue string     `json:"window_value,omitempty"` // "1", "2", "3" sessions ; "7", "14", "30" jours ; ISO date pour deadline
	Cadence     Cadence    `json:"cadence"`
	EvalType    EvalType   `json:"eval_type"`

	Mode     ChallengeMode `json:"mode"`
	Tier     Tier          `json:"tier,omitempty"`
	DataTier DataTier      `json:"data_tier"`

	Label  string          `json:"label,omitempty"`
	Status ChallengeStatus `json:"status"`

	// CurrentValue est la valeur courante mesurée pour la métrique du défi.
	// Renseignée uniquement par les endpoints qui invoquent l'évaluateur
	// (ListActiveChallenges enrichi). Pas persistée — recalculée à la demande.
	CurrentValue float64 `json:"current_value,omitempty"`

	// PPReward = PP crédités à la complétion (PPForCompletion par Tier/DataTier ;
	// isSquad=false, comme creditCompletion). Enrichi par ListActiveChallenges,
	// non persisté. 0 (omis du JSON) pour DataTier=tracking. Permet au front
	// d'afficher la récompense PP de chaque objectif.
	PPReward int `json:"pp_reward,omitempty"`

	// ExpiresAt est le timestamp d'expiration calculé à la création selon le tier et le mode.
	// Nil pour le mode libre (pas de timer). Consulté par l'évaluateur pour toute WindowType.
	ExpiresAt *time.Time `json:"expires_at,omitempty"`

	CreatedAt             time.Time  `json:"created_at"`
	CommittedAt           *time.Time `json:"committed_at,omitempty"`
	CompletedAt           *time.Time `json:"completed_at,omitempty"`
	ExpiredAt             *time.Time `json:"expired_at,omitempty"`
	AbandonedAt           *time.Time `json:"abandoned_at,omitempty"`
	LastPalierRecomputeAt *time.Time `json:"last_palier_recompute_at,omitempty"`

	IsPrivate bool `json:"is_private"`

	// Source trace l'origine du défi (ChallengeSource*: "user" | "pilot_mode" |
	// "coach"), renseignée à la création depuis CreateChallengeRequest.Source et
	// figée (jamais mutée). Recopiée sur chaque événement prestige_telemetry pour
	// le calage coach (ADR 0020). Vide pour les défis créés avant le plumbing.
	Source string `json:"source,omitempty"`
}

// ---------- Arc ----------

// Arc représente une séquence ordonnée de défis avec un fil narratif.
//
// Persistance : table `arc` dans stats.duckdb (par joueur).
// Si IsPreset=true, PresetID référence un `preset_arc` dans metadata.duckdb.
type Arc struct {
	ID          string     `json:"id"`
	UserID      string     `json:"user_id"`
	TitleSlug   string     `json:"title_slug"`
	Title       string     `json:"title"`
	Description string     `json:"description,omitempty"`
	IsPreset    bool       `json:"is_preset"`
	PresetID    string     `json:"preset_id,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`

	// ObjectivesPP est la somme des PP des objectifs de l'arc (chacun via
	// PPForCompletion). Enrichi en lecture par ListArcs/GetArc, non persisté.
	// Permet au front d'afficher la récompense cumulée des étapes.
	ObjectivesPP int `json:"objectives_pp,omitempty"`

	// CompletionBonusPP est le bonus PP crédité à la complétion de l'arc
	// (PPForArcCompletion sur ObjectivesPP) — distinct des PP des objectifs,
	// versé une fois toutes les étapes terminées. Enrichi en lecture, non
	// persisté.
	CompletionBonusPP int `json:"completion_bonus_pp,omitempty"`
}

// ---------- MomentCard ----------

// MomentCard est l'artefact visuel généré à la validation d'un défi.
//
// Persistance : table `moment_card` dans stats.duckdb. Le blob lui-même
// (PNG/JPG) est stocké hors-DB ; BlobPath y donne accès.
type MomentCard struct {
	ID          string    `json:"id"`
	ChallengeID string    `json:"challenge_id"`
	BlobPath    string    `json:"blob_path,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// ---------- PrestigeEvent ----------

// PrestigeEvent est une émission de PP attribuable à une source identifiable
// (match, défi, arc, streak, médaille…).
//
// Persistance : table `prestige_events` dans shared_social.duckdb.
type PrestigeEvent struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	TitleSlug  string    `json:"title_slug"`
	SourceType string    `json:"source_type"` // "match" | "challenge" | "arc" | "streak" | "medal"
	SourceID   string    `json:"source_id,omitempty"`
	PPAmount   int       `json:"pp_amount"`
	Tier       Tier      `json:"tier,omitempty"` // pour défis et arcs
	CreatedAt  time.Time `json:"created_at"`
}

// ---------- UserPrestige ----------

// UserPrestige est l'agrégat courant de PP et niveau pour un joueur sur un titre.
//
// Persistance : table `user_prestige` dans shared_social.duckdb (PK composite).
type UserPrestige struct {
	UserID       string    `json:"user_id"`
	TitleSlug    string    `json:"title_slug"`
	TotalPP      int       `json:"total_pp"`
	CurrentLevel int       `json:"current_level"`
	UpdatedAt    time.Time `json:"updated_at"`
	// Level est l'agrégat des seuils du niveau courant (nom, prochain seuil,
	// ratio de progression). Vide si non enrichi (lecture brute repository).
	Level *Level `json:"level,omitempty"`
}

// ---------- Template ----------

// Template décrit un modèle de défi pré-calibré, source de l'auto-attribution
// et du mode hybride. Chargé depuis `config/titles/{slug}/challenges/templates.toml`
// dans la table `challenge_template` de metadata.duckdb.
type Template struct {
	ID          string     `json:"id"`
	TitleSlug   string     `json:"title_slug"`
	Metric      string     `json:"metric"`
	WindowType  WindowType `json:"window_type"`
	WindowValue string     `json:"window_value,omitempty"`
	Cadence     Cadence    `json:"cadence"`
	EvalType    EvalType   `json:"eval_type"`
	ModeFilter  string     `json:"mode_filter"` // "universal" | "pvp" | "ranked" | "pve"

	LabelEN       string `json:"label_en"`
	LabelFR       string `json:"label_fr"`
	DescriptionEN string `json:"description_en,omitempty"`
	DescriptionFR string `json:"description_fr,omitempty"`

	NormalTarget    float64 `json:"normal_target"`
	HeroicTarget    float64 `json:"heroic_target"`
	LegendaryTarget float64 `json:"legendary_target"`
	MythicTarget    float64 `json:"mythic_target"`

	// Tagging V1 PlayerProfile (cf. PLAN_PLAYER_PROFILE_ASCENSION.md §5.1).
	// Permet le matching profil → suggestions de défis (Section C).

	// LUSRComponents : composantes LUSR ciblées par le template
	// (ex: ["kills_vs_expected"], ["accuracy_delta", "kills_vs_expected"]).
	// Référentiel : sync.CompositeWeights (skill_config.go).
	LUSRComponents []string `json:"lusr_components,omitempty"`

	// RadarAxes : axes narrative 6 ciblés (optionnel, redondant avec
	// LUSRComponents pour certains templates). Valeurs : combat / survival /
	// support / score / objective / impact.
	RadarAxes []string `json:"radar_axes,omitempty"`

	// IsLongTerm : true si le template encourage la progression durable
	// (window_type=rolling_days OU last_n_matches threshold). Utilisé par
	// PlayerProfile pour favoriser ces templates dans les suggestions de
	// campagne.
	IsLongTerm bool `json:"is_long_term"`

	// Source distingue les templates seedés depuis TOML ("catalog") de ceux
	// synthétisés dynamiquement par le coach_advisor ("coach_synthesized",
	// cf. ADR 0028). Vide à la lecture = "catalog" (rétrocompat).
	//
	// Les templates synthétisés stockent dans normal_target/heroic_target/
	// legendary_target/mythic_target les **stretch ratios** standards
	// (1.08, 1.25, 1.50, 2.00) — la cible absolue est matérialisée par
	// CalculatePalier(baseline) au moment du CreateChallenge (cf. invariants
	// I1/I2 de l'ADR 0020).
	Source string `json:"source,omitempty"`

	SchemaVersion int       `json:"schema_version"`
	UpdatedAt     time.Time `json:"updated_at"`

	// CooldownEndsAt : instant de fin du cooldown anti-farming sur la métrique
	// du template pour le joueur courant. Non persisté — enrichi à la demande
	// par SuggestTemplates (comme Challenge.CurrentValue). Nil = pas de cooldown
	// actif → le template est sélectionnable. Permet au front d'afficher un
	// badge « disponible dans Xh » et de désactiver le choix.
	CooldownEndsAt *time.Time `json:"cooldown_ends_at,omitempty"`
}

// ---------- PresetArc + PresetArcStep ----------

// PresetArc décrit un arc preset chargé depuis `config/titles/{slug}/arcs/presets.toml`.
type PresetArc struct {
	ID            string          `json:"id"`
	TitleSlug     string          `json:"title_slug"`
	TitleEN       string          `json:"title_en"`
	TitleFR       string          `json:"title_fr"`
	DescriptionEN string          `json:"description_en,omitempty"`
	DescriptionFR string          `json:"description_fr,omitempty"`
	SchemaVersion int             `json:"schema_version"`
	UpdatedAt     time.Time       `json:"updated_at"`
	Steps         []PresetArcStep `json:"steps,omitempty"` // hydraté à la demande
}

// PresetArcStep est une étape d'un PresetArc, référençant un Template avec un palier cible.
type PresetArcStep struct {
	PresetArcID string `json:"preset_arc_id"`
	Position    int    `json:"position"` // 1, 2, 3…
	TemplateID  string `json:"template_id"`
	TargetTier  Tier   `json:"target_tier"`
}

// ---------- Squad / SquadMember ----------

// Squad représente un groupe Prestige (différent de l'escouade gameplay existante,
// même si peut être construit à partir d'elle).
type Squad struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedBy string    `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}

// SquadMember représente l'appartenance d'un joueur à une Squad.
//
// Clé universelle = Xuid : tout membre en a un, qu'il soit ou non utilisateur de
// l'app. Cela permet (a) d'inclure des amis hors-app (présents seulement comme
// xuid dans shared.match_participants) et (b) de mesurer la progression des défis
// d'escouade sur match_participants (clé xuid).
//
// UserID (player_slug) est OPTIONNEL : renseigné uniquement quand le membre est
// utilisateur de l'app, ce qui lui ouvre l'accès lecture/écriture aux objectifs
// et arcs de l'escouade (règle « membre-user, sans consentement » —
// PLAN_COACH_V3_GENERATION § Identité d'escouade). Vide pour un ami hors-app.
type SquadMember struct {
	SquadID string `json:"squad_id"`
	Xuid    string `json:"xuid"`
	UserID  string `json:"user_id,omitempty"`
	// Gamertag est un snapshot d'affichage du roster (le libellé choisi à
	// l'ajout). Non utilisé comme clé (la clé reste Xuid) ; sert au front pour
	// afficher les membres et recharger une composition (page Escouade en
	// gamertags). Peut être vide pour des membres legacy (avant la colonne).
	Gamertag string    `json:"gamertag,omitempty"`
	JoinedAt time.Time `json:"joined_at"`
}

// ---------- SquadChallenge / SquadChallengeParticipant ----------

// SquadChallenge représente un défi d'escouade — collectif ou compétitif.
type SquadChallenge struct {
	ID              string     `json:"id"`
	SquadID         string     `json:"squad_id"`
	TemplateID      string     `json:"template_id,omitempty"`
	TitleSlug       string     `json:"title_slug"`
	Mode            SquadMode  `json:"mode"`
	EvalType        EvalType   `json:"eval_type"`
	WindowType      WindowType `json:"window_type"`
	WindowValue     string     `json:"window_value,omitempty"`
	TargetPerMember float64    `json:"target_per_member,omitempty"`
	ExpiresAt       *time.Time `json:"expires_at,omitempty"`
	CreatedBy       string     `json:"created_by"`
	CreatedAt       time.Time  `json:"created_at"`
}

// SquadChallengeParticipant représente la participation d'un membre à un défi d'escouade.
type SquadChallengeParticipant struct {
	SquadChallengeID string     `json:"squad_challenge_id"`
	UserID           string     `json:"user_id"`
	ChosenTier       Tier       `json:"chosen_tier,omitempty"`
	DataTier         DataTier   `json:"data_tier"`
	CurrentValue     float64    `json:"current_value"`
	CompletedAt      *time.Time `json:"completed_at,omitempty"`
	IsPrivate        bool       `json:"is_private"`
	JoinedAt         time.Time  `json:"joined_at"`
}

// SquadChallengeView enrichit un défi d'escouade pour l'affichage : libellés
// localisés (résolus depuis le template — vides si le défi n'a pas de template ou
// s'il a été retiré du catalogue) et participants courants. Le défi de base est
// embarqué (JSON aplati) : la vue est un sur-ensemble strict de SquadChallenge.
type SquadChallengeView struct {
	SquadChallenge
	LabelFR string `json:"label_fr,omitempty"`
	LabelEN string `json:"label_en,omitempty"`
	// Expired : expires_at dépassé (comparé à l'horloge canonique UTC du service).
	// Le défi reste listé (le membre peut le voir et le supprimer) mais l'UI le
	// signale et empêche de le rejoindre.
	Expired      bool                        `json:"expired"`
	Participants []SquadChallengeParticipant `json:"participants"`
}

// ---------- PrestigeTelemetry ----------

// PrestigeTelemetry est un événement structurel utilisé pour le calage post-alpha.
//
// Consommé par l'endpoint diag GET /api/v1/_diag/prestige/telemetry/{player_slug}
// (agrégation acceptation/complétion par origine, ADR 0020).
//
// Persistance : table `prestige_telemetry` dans stats.duckdb (par joueur),
// append-only INSERT-only (un événement = une ligne distincte).
type PrestigeTelemetry struct {
	ID                     string        `json:"id"`
	UserID                 string        `json:"user_id"`
	ChallengeID            string        `json:"challenge_id,omitempty"`
	EventType              string        `json:"event_type"` // "created" | "rejected" | "completed" | "expired" | "abandoned" | "palier_recomputed"
	Palier                 Tier          `json:"palier,omitempty"`
	StretchRatio           float64       `json:"stretch_ratio,omitempty"`
	BaselineValue          float64       `json:"baseline_value,omitempty"`
	Mode                   ChallengeMode `json:"mode,omitempty"`
	Cadence                Cadence       `json:"cadence,omitempty"`
	EvalType               EvalType      `json:"eval_type,omitempty"`
	TimeSinceCreateSeconds int           `json:"time_since_create_seconds,omitempty"`
	// Source recopie Challenge.Source (origine du défi) sur l'événement — permet
	// d'agréger les taux par origine sans jointure. Vide pour l'historique.
	Source    string    `json:"source,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// ---------- BaselineState ----------

// BaselineState suit la fraîcheur de la baseline pour un (joueur, titre, métrique).
//
// Permet le reset après 60 jours d'inactivité (Axe 2 du plan) et la phase
// de recovery (10 matchs en data_tier=estimated avant retour au plein bonus).
type BaselineState struct {
	UserID                   string     `json:"user_id"`
	TitleSlug                string     `json:"title_slug"`
	Metric                   string     `json:"metric"`
	LastMatchAt              *time.Time `json:"last_match_at,omitempty"`
	IsStale                  bool       `json:"is_stale"`
	RecoveryMatchesRemaining int        `json:"recovery_matches_remaining"`
	UpdatedAt                time.Time  `json:"updated_at"`
}

// ---------- Baseline ----------

// Baseline est la valeur calculée d'une moyenne personnelle sur la fenêtre interne
// (20 matchs glissants par défaut, configurable).
//
// Pas persistée : recalculée à la volée dans baseline.go.
type Baseline struct {
	UserID     string    `json:"user_id"`
	TitleSlug  string    `json:"title_slug"`
	Metric     string    `json:"metric"`
	Value      float64   `json:"value"`
	MatchCount int       `json:"match_count"`
	DataTier   DataTier  `json:"data_tier"`
	ComputedAt time.Time `json:"computed_at"`
}

// ---------- Level ----------

// Level est le niveau Prestige déduit d'un total PP.
type Level struct {
	Index           int     `json:"index"`             // 0=Recrue, 1=Soldat, ...
	Name            string  `json:"name"`              // "Recrue", "Soldat", ...
	ThresholdPP     int     `json:"threshold_pp"`      // PP à partir duquel ce niveau commence
	NextThresholdPP int     `json:"next_threshold_pp"` // -1 si max
	ProgressRatio   float64 `json:"progress_ratio"`    // 0..1 vers le niveau suivant
}
