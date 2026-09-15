// Package domain — identity.go : types de l'annuaire des joueurs (ADR 0035).
//
// L'identité d'un joueur est éparpillée sur quatre registres de cycles de vie
// différents — le compte (`data/auth/users.json`), les credentials
// (`data/auth/watcher_tokens/{xuid}.json`), le profil de suivi
// (`db_profiles.json`) et le suivi live (le daemon watcher) — et la SEULE clé
// qui les relie est le xuid (ADR 0035 D1). Le gamertag et le slug sont des
// valeurs d'affichage et une composante de chemin, jamais une clé de jointure.
package domain

import "context"

// ProfileGate répond « ce couple (titre, xuid) est-il un profil SUIVI ? ».
//
// Suivi = déclaré dans `db_profiles.json` pour ce titre, non `auth_only`, et
// `sync_enabled != false` — exactement le filtre de SyncablePlayers. C'est la
// porte que franchissent le coordinateur de sync (`sync.Coordinator.Submit`),
// le daemon watcher (`watcher.Daemon.AddPlayer`) et le SSO Xbox avant de
// notifier le watcher (ADR 0035 D3).
//
// POURQUOI (incident du 2026-07-23) : un compte Xbox inconnu s'est connecté par
// SSO sur une instance non verrouillée. Le compte et ses credentials ont été
// créés, le watcher l'a pris en charge, et deux heures plus tard un sync a créé
// une player DB et écrit 25 matchs dans l'entrepôt partagé — alors qu'aucun
// profil n'existait. Tout ce qui vient ensuite (notifications post-sync,
// Prestige, ownership, scheduler, pages admin) résout un joueur par
// `db_profiles.json` : ce joueur n'existait pour personne.
//
// Un compte sans profil est un état VALIDE et INERTE. Sa seule sortie est le
// wizard de mise en place (ou une invitation, cf. le plan frère des invitations).
//
// La recherche se fait par XUID, jamais par gamertag : un gamertag se renomme,
// un xuid non.
type ProfileGate func(ctx context.Context, titleSlug, xuid string) bool

// ProfileRef est le profil de suivi d'une identité pour UN titre, tel que
// `db_profiles.json` le déclare, augmenté de ce que le disque en dit.
//
// Key est la clé du profil dans le fichier (le gamertag tel qu'il y est écrit) :
// c'est aussi la composante de chemin des dossiers joueur
// (`PathResolver.PlayerDir`). DirExists/DBExists disent si ce dossier et sa
// player DB existent — un profil déclaré sans dossier n'a jamais été synchronisé,
// un dossier sans profil est un orphelin (cf. OrphanDirRef).
type ProfileRef struct {
	TitleSlug   string `json:"title_slug"`
	Key         string `json:"key"`
	SyncEnabled bool   `json:"sync_enabled"`
	AuthOnly    bool   `json:"auth_only"`
	DirExists   bool   `json:"dir_exists"`
	DBExists    bool   `json:"db_exists"`
}

// AccountRef est le compte de connexion associé à une identité
// (`data/auth/users.json`) : qui peut ouvrir une session, avec quel rôle.
type AccountRef struct {
	Username    string   `json:"username"`
	Role        UserRole `json:"role"`
	CreatedAt   string   `json:"created_at,omitempty"`
	LastLoginAt string   `json:"last_login_at,omitempty"`
}

// TokenRef est l'état des credentials d'une identité (ADR 0023), SANS jamais
// porter de secret : seulement la présence d'un refresh token, l'état de
// ré-authentification et la dernière erreur d'auth persistée.
type TokenRef struct {
	HasRefreshToken bool   `json:"has_refresh_token"`
	ReauthRequired  bool   `json:"reauth_required"`
	LastAuthError   string `json:"last_auth_error,omitempty"`
	UpdatedAt       string `json:"updated_at,omitempty"`
}

// OrphanDirRef est un dossier joueur présent sur disque qu'aucun profil ne
// déclare. C'est la trace que laisse un sync qui a tourné pour un joueur sans
// profil (incident du 2026-07-23) — ou un profil supprimé sans son dossier.
type OrphanDirRef struct {
	TitleSlug string `json:"title_slug"`
	Name      string `json:"name"`
}

// Codes d'anomalie de l'annuaire (ADR 0035 D2). Ce sont des codes MACHINE : le
// libellé affiché est construit côté web (table FR/EN), jamais ici.
const (
	// AnomalyAccountWithoutProfile : un compte peut se connecter, mais aucun
	// profil ne le suit — rien ne sera jamais synchronisé pour lui. C'est l'état
	// exact du compte du 2026-07-23.
	AnomalyAccountWithoutProfile = "account_without_profile"
	// AnomalyTokenOrphan : des credentials sans compte NI profil — plus personne
	// ne les utilise ni ne peut les renouveler.
	AnomalyTokenOrphan = "token_orphan"
	// AnomalyPlayerDirOrphan : un dossier joueur sur disque qu'aucun profil ne
	// déclare (données écrites pour un joueur que l'app ne connaît pas).
	AnomalyPlayerDirOrphan = "player_dir_orphan"
	// AnomalyWatchedWithoutProfile : le daemon watcher suit en live un couple
	// (joueur, titre) sans profil suivi. Depuis les portes de l'ADR 0035 D3 c'est
	// devenu impossible à créer ; l'anomalie reste la preuve que la porte tient.
	AnomalyWatchedWithoutProfile = "watched_without_profile"
	// AnomalyProfileWithoutAccount : un profil sans compte de connexion. SITUATION
	// NORMALE — c'est le cas de tous les amis suivis par l'administrateur.
	AnomalyProfileWithoutAccount = "profile_without_account"
	// AnomalyProfileWithoutToken : un profil sans credentials propres. SITUATION
	// NORMALE — le pool d'auth prête les credentials d'un autre joueur.
	AnomalyProfileWithoutToken = "profile_without_token"
)

// Sévérités d'anomalie. `warning` = à regarder (une incohérence entre registres) ;
// `info` = état normal mais utile à voir (un profil d'ami, un profil servi par le
// pool d'auth). Une instance saine n'a AUCUN `warning`.
const (
	AnomalySeverityWarning = "warning"
	AnomalySeverityInfo    = "info"
)

// IdentityAnomaly est une incohérence (ou une particularité) constatée entre les
// registres pour une identité. Detail porte le CONTEXTE machine (un slug de
// titre, un nom de dossier), jamais une phrase d'interface.
type IdentityAnomaly struct {
	Code     string `json:"code"`
	Severity string `json:"severity" enum:"warning,info"`
	Detail   string `json:"detail,omitempty"`
}

// IdentityRecord est la vue unifiée d'une identité : ce que chacun des registres
// en sait, réuni par le xuid (ADR 0035 D1).
//
// XUID peut être vide dans deux cas, tous deux visibles À DESSEIN : un profil ou
// un compte créé sans identité Xbox résolue, et un dossier joueur orphelin que
// plus aucun registre ne réclame. Les masquer reviendrait à reproduire le trou
// que cet annuaire existe pour fermer.
type IdentityRecord struct {
	XUID       string            `json:"xuid,omitempty"`
	Gamertag   string            `json:"gamertag,omitempty"`
	Profiles   []ProfileRef      `json:"profiles"`
	Account    *AccountRef       `json:"account,omitempty"`
	Token      *TokenRef         `json:"token,omitempty"`
	Watched    []string          `json:"watched"`
	OrphanDirs []OrphanDirRef    `json:"orphan_dirs,omitempty"`
	Anomalies  []IdentityAnomaly `json:"anomalies"`
}

// WatchedPlayerRef est un couple (joueur, titre) actuellement suivi en live par
// le daemon watcher. Type de LECTURE seulement (pas un corps HTTP) : il porte le
// xuid pour que l'annuaire rattache le suivi live à la bonne identité sans
// jamais joindre par gamertag.
type WatchedPlayerRef struct {
	XUID      string
	Gamertag  string
	TitleSlug string
}

// AdminIdentitiesResponse est le corps de `GET /admin/identities`.
//
// Counts porte les compteurs de l'en-tête : `identities`, `warnings`, `infos`,
// puis un compteur par code d'anomalie présent (clé = le code). Une clé absente
// vaut zéro.
type AdminIdentitiesResponse struct {
	GeneratedAt string           `json:"generated_at"`
	Identities  []IdentityRecord `json:"identities"`
	Counts      map[string]int   `json:"counts"`
}
