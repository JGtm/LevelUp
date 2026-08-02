// Package domain contient les types métier purs de LevelUp (0 import externe, 0 IO).
package domain

// FeatureFlags représente les flags de fonctionnalités actifs sur ce déploiement.
type FeatureFlags struct {
	V7Enabled         bool `json:"v7_enabled"`
	MediaEnabled      bool `json:"media_enabled"`
	DemoMode          bool `json:"demo_mode"`
	DiscordConfigured bool `json:"discord_configured"`
	TailscaleEnabled  bool `json:"tailscale_enabled"`
}

// SettingsExcerpt représente l'extrait des préférences utilisateur nécessaires au shell.
type SettingsExcerpt struct {
	Lang                string `json:"lang"`
	UserTimezone        string `json:"user_timezone"`
	ShowRecords         bool   `json:"show_records"`
	NormalizeModeLabels bool   `json:"normalize_mode_labels"`
}

// CapabilityMap représente les capacités actives selon la configuration serveur.
type CapabilityMap struct {
	CanReadLocalData    bool `json:"can_read_local_data"`
	CanRunSync          bool `json:"can_run_sync"`
	CanUseLiveHalo      bool `json:"can_use_live_halo"`
	CanManageSettings   bool `json:"can_manage_settings"`
	CanResetMediaIndex  bool `json:"can_reset_media_index"`
	CanViewMedia        bool `json:"can_view_media"`
	CanSelfProvision    bool `json:"can_self_provision"`
	CanStartInitialSync bool `json:"can_start_initial_sync"`
	CanManageInstance   bool `json:"can_manage_instance"`
}

// PlayerSummary représente le résumé d'un profil joueur (couple joueur × titre).
type PlayerSummary struct {
	PlayerSlug     string `json:"player_slug"`
	Gamertag       string `json:"gamertag"`
	XUID           string `json:"xuid"`
	WaypointPlayer string `json:"waypoint_player"`
	IsDemo         bool   `json:"is_demo"`
	SteamID        string `json:"steam_id,omitempty"`   // Steam ID pour le poller de présence
	TitleSlug      string `json:"title_slug,omitempty"` // Sprint 44 : titre associé
	// SyncEnabled : false = sync en PAUSE pour ce (joueur, titre) — les données
	// restent sur disque mais ne sont plus rafraîchies. Résolu nil→true au chargement.
	SyncEnabled bool `json:"sync_enabled"`
	// InitialMaxMatches : nombre de matchs demandés à l'onboarding (0 = défaut).
	InitialMaxMatches int `json:"initial_max_matches,omitempty"`
	// AuthOnly : profil existant uniquement pour la gestion des tokens auth (pas
	// un vrai joueur suivi). Exclu des listes front-facing (sélecteur L1, favoris
	// gamertag) par le BootstrapService ; conservé pour les usages serveur.
	AuthOnly bool `json:"auth_only,omitempty"`
}

// SyncablePlayers retourne les couples (joueur, titre) dont le sync est ACTIF
// (sync_enabled != false) ET qui sont de VRAIS joueurs suivis (pas AuthOnly).
// C'est le filtre CANONIQUE des CHEMINS SYNC (scheduler auto_sync, watcher daemon,
// fan-out, recompute amis) : un titre « en pause » ne doit plus être rafraîchi
// (données conservées sur disque), et un profil AuthOnly n'a PAS de DB joueur
// (db_path vide) — le synchroniser échoue systématiquement (duckdb.OpenReadOnly
// sur chemin inexistant) et pollue les cycles de sync à chaque tick sans jamais
// pouvoir aboutir. Les profils AuthOnly existent uniquement pour fournir des
// refresh tokens au pool ; ils restent visibles des chemins AUTH (pool discovery,
// worldenrich) mais jamais des chemins SYNC.
//
// NE PAS l'appliquer aux chemins UI/LECTURE (bootstrap, liste /players,
// résolution d'un joueur pour servir une page) : un titre en pause doit y rester
// VISIBLE, sinon réactivation impossible (404 ErrPlayerNotFound) et disparition
// des réglages. Les profils AuthOnly y ont leur propre filtre dédié
// (excludeAuthOnly côté BootstrapService), à ne pas confondre avec celui-ci.
func SyncablePlayers(players []PlayerSummary) []PlayerSummary {
	out := make([]PlayerSummary, 0, len(players))
	for _, p := range players {
		if p.SyncEnabled && !p.AuthOnly {
			out = append(out, p)
		}
	}
	return out
}

// HaloIdentitySummary représente l'identité Halo résolue côté backend.
type HaloIdentitySummary struct {
	Gamertag string `json:"gamertag"`
	XUID     string `json:"xuid"`
}

// TitleSummary résume un titre supporté pour le frontend.
type TitleSummary struct {
	Slug         string   `json:"slug"`
	Name         string   `json:"name"`
	IconURL      string   `json:"icon_url,omitempty"`
	Status       string   `json:"status" enum:"active,coming_soon,archived"`
	Capabilities []string `json:"capabilities"`
	IsDefault    bool     `json:"is_default"`
	// EffectiveHpToKill : PV effectifs pour tuer un joueur (baseline rendement/
	// résistance), title-spécifique (225 Infinite, 115 Halo 5). Permet au front de
	// rendre le copy d'aide combat title-aware sans dupliquer la constante.
	EffectiveHpToKill float64 `json:"effective_hp_to_kill"`
	// OffensiveConversionP80 : frontière élite (80e percentile) du rendement OC du
	// titre, repère de normalisation des barres/radars de rendement (0.90 Infinite,
	// 1.264 Halo 5). Permet au front de normaliser les barres OC sur la bonne échelle.
	OffensiveConversionP80 float64 `json:"offensive_conversion_p80"`
	// DefensiveResistanceP80 : frontière élite (80e percentile) de la résistance DR du
	// titre (dégâts_subis / (PV × morts)), pendant défensif d'OffensiveConversionP80
	// (1.65 Infinite). Permet au front de tracer l'écart à la frontière élite sans
	// dupliquer la constante. Un titre sans damage_taken sert le défaut, non consommé
	// (ProvidesDamageTaken=false → les surfaces de résistance sont neutralisées).
	DefensiveResistanceP80 float64 `json:"defensive_resistance_p80"`
	// ProvidesDamageTaken : false si l'API du titre ne fournit PAS damage_taken
	// (Halo 5 — carnage cryptum sans dégâts subis). Le front NEUTRALISE alors la
	// Résistance défensive (N/A) au lieu d'afficher 0 (trompeur : « résistance
	// nulle »). Défaut true (Infinite, byte-identique).
	ProvidesDamageTaken bool `json:"provides_damage_taken" doc:"false si l'API du titre ne fournit pas damage_taken (Halo 5) : le front neutralise la Résistance défensive (N/A) au lieu d'afficher 0."`
	// ProvidesTeamMMR : false si l'API du titre ne fournit PAS de MMR d'équipe/
	// adverse par match (Halo 5). Le front MASQUE alors la colonne MMR du tableau
	// Escouade (et d'Explorer) au lieu d'afficher 0 (trompeur). Défaut true (Infinite).
	ProvidesTeamMMR bool `json:"provides_team_mmr" doc:"false si l'API du titre ne fournit pas de MMR d'équipe/adverse par match (Halo 5) : le front masque la colonne MMR au lieu d'afficher 0."`
	// ProvidesMaxKillingSpree : true si le titre SUPPORTE la « folie meurtrière max »
	// par match — soit via sa valeur native (Infinite), soit en la CALCULANT depuis ses
	// events kill/death horodatés (Halo 5, dérivé de la capability events-timeline). La
	// série n'est MASQUÉE que pour un titre sans events horodatés (false). Défaut true.
	ProvidesMaxKillingSpree bool `json:"provides_max_killing_spree" doc:"true si le titre fournit la folie meurtrière max par match (native ou calculée via events horodatés). Front masque la série si false."`
}

// BootstrapResponse est la réponse de GET /api/v1/bootstrap.
// Contient tout ce dont le shell React a besoin pour s'initialiser.
type BootstrapResponse struct {
	SetupRequired       bool                 `json:"setup_required"`
	AuthState           string               `json:"auth_state" enum:"missing,partial,ready"`
	SetupState          string               `json:"setup_state" enum:"no_halo_link,halo_linked_no_profile,profile_ready_no_sync,ready"`
	CurrentPlayer       *PlayerSummary       `json:"current_player"`
	AvailablePlayers    []PlayerSummary      `json:"available_players"`
	CurrentTitleSlug    string               `json:"current_title_slug" doc:"Sprint 44 : slug du titre courant de la session."`
	AvailableTitles     []TitleSummary       `json:"available_titles" doc:"Sprint 44 : titres disponibles (title switcher) : active + coming_soon."`
	Locale              string               `json:"locale" default:"fr"`
	HintsVisibleDefault bool                 `json:"hints_visible_default" default:"true"`
	FeatureFlags        FeatureFlags         `json:"feature_flags"`
	Capabilities        CapabilityMap        `json:"capabilities"`
	SettingsExcerpt     SettingsExcerpt      `json:"settings_excerpt"`
	LinkedHaloIdentity  *HaloIdentitySummary `json:"linked_halo_identity,omitempty"`
	ActiveSyncJobID     *string              `json:"active_sync_job_id,omitempty"`
	// Sprint 54 B : privacy du compte courant (chargée en parallèle).
	Privacy *MatchPrivacyInfo `json:"privacy,omitempty" doc:"Sprint 54-B : informations de confidentialité des matchs du joueur actif."`
	// Auth locale
	AuthMode         string `json:"auth_mode" enum:"none,password,xbox"`
	RegistrationMode string `json:"registration_mode" enum:"invite,open,closed"`
	// InstanceLocked : instance fermée (aucune nouvelle identité/BDD). Le frontend
	// l'utilise pour afficher un bandeau « inscriptions fermées » et désactiver les
	// CTA de création. Verrou effectif = env (LEVELUP_INSTANCE_LOCKED) OU app_settings.
	InstanceLocked bool `json:"instance_locked"`
	// ReauthRequired : true si le refresh_token Microsoft du joueur courant est mort
	// (refresh silencieux définitivement KO). Le front affiche une bannière
	// « reconnecte ton compte Xbox ». Remis à false après une ré-auth réussie.
	ReauthRequired bool `json:"reauth_required"`
	// HasPassword : l'utilisateur connecté a défini un mot de passe (opt-in PR-C).
	// Le front l'utilise pour proposer « définir » vs « changer » et masquer la
	// proposition en onboarding une fois faite.
	HasPassword     bool    `json:"has_password"`
	IsAdmin         bool    `json:"is_admin"`
	CurrentUsername *string `json:"current_username"`
	FirstLaunch     bool    `json:"first_launch"`
	// PR 4 — Authorization Code Flow disponible (true si cfg.OAuthRedirectURI configuré).
	// Le frontend affiche un bouton "SSO redirect" en plus du Device Code si true.
	OAuthCodeFlowEnabled bool `json:"oauth_code_flow_enabled"`
	// DemoMode : instance démo publique. Le frontend l'utilise pour figer les
	// settings (read-only, sauf langue/accessibilité) et basculer le changement de
	// langue en client-side (le PATCH /settings est refusé en démo).
	DemoMode bool `json:"demo_mode"`
}

// PlayersListResponse est la réponse de GET /api/v1/players.
type PlayersListResponse struct {
	Items             []PlayerSummary `json:"items"`
	DefaultPlayerSlug *string         `json:"default_player_slug"`
}
