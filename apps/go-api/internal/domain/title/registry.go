// Package title — registre des titres supportés et résolution de chemins.
//
// Sprint 44 WP0 : colonne vertébrale du runtime multi-titres.
// Le TitleRegistry est la source de vérité centrale pour les titres supportés.
// Le PathResolver est le point unique de résolution des chemins title-aware.
package title

import (
	"fmt"
	"path/filepath"
	"sync"
)

// ---------------------------------------------------------------------------
// TitleDescriptor — description d'un titre supporté
// ---------------------------------------------------------------------------

// Status représente l'état d'un titre dans le registre.
type Status string

const (
	StatusActive     Status = "active"
	StatusComingSoon Status = "coming_soon"
	StatusArchived   Status = "archived"
)

// Capability décrit une fonctionnalité supportée par un titre.
type Capability string

const (
	CapMatchmaking Capability = "matchmaking"
	CapFirefight   Capability = "firefight"
	CapForge       Capability = "forge"
	CapMedia       Capability = "media"
	CapRanked      Capability = "ranked"
	CapCareer      Capability = "career"
	// CapSeasonPass — progression de pass saisonnier / Battlepass (Halo Infinite).
	// Absente ⇒ la page career/season-pass dégrade en FeatureUnavailable + l'onglet
	// « Pass saisonnier » est masqué. Halo 5 ne l'expose PAS (pas de Battlepass ;
	// l'inventaire REQ personnel n'est pas servi — sonde 404 — donc aucune surface
	// de remplacement câblée).
	CapSeasonPass   Capability = "season_pass"
	CapAssetImages  Capability = "asset.images" // Asset Drawer — thumbnails maps & armes
	CapAchievements Capability = "achievements" // Xbox achievements bilingues (page Achievements)
	CapEngagement   Capability = "engagement"   // Score d'engagement intra-match + coefficients personnels
	CapLUSR         Capability = "lusr"         // Rating interne LevelUp (LUSR v2 TrueSkill2) — calcul post-sync + Stratégie C

	// CapWorldLeaderboard — classements mondiaux (CSR scrapé Waypoint + stats
	// agrégées par titre). Absente ⇒ page leaderboard dégrade en vide+200 (PMT-7).
	CapWorldLeaderboard Capability = "world.leaderboard"

	// CapNativeKillMechanics — le titre fournit NATIVEMENT le détail des kills par
	// mécanique (assassinats + compétences spartiate : ground pound, shoulder bash),
	// agrégés par-joueur dans le carnage. Halo 5 only (Infinite ne les expose pas).
	// Absente ⇒ le front masque les sections « assassinats / compétences spartiate ».
	CapNativeKillMechanics Capability = "native_kill_mechanics"

	// CapWeaponKills — le titre produit le détail des kills PAR ARME (pipeline
	// film-decoder côté ingestion). Halo Infinite : oui (films) ; Halo 5 : non
	// (pas de film-decoder weapon-per-kill). Sert le prédicat de complétude du
	// snapshot (Phase 2) : un titre SANS cette cap n'EXIGE pas les weapon-kills
	// pour qu'un match soit « complet ». À NE PAS confondre avec
	// CapNativeKillMechanics (assassinats/ground-pound natifs, Halo-5-only).
	CapWeaponKills Capability = "weapon_kills"

	// CapTeamMMR — le titre fournit NATIVEMENT un MMR d'équipe (et adverse) par
	// match via son API. Forme title-level (capability dans le descripteur) du
	// signal scalaire games.ProvidesTeamMMR(slug) : un titre déclare cette cap
	// SSI ProvidesTeamMMR(slug)==true. Halo Infinite : oui ; Halo 5 : non (le
	// carnage cryptum ne porte pas de MMR équipe/adverse → no_team_mmr=true).
	// Absente ⇒ les surfaces (colonne MMR du tableau Escouade / Explorer)
	// neutralisent la valeur plutôt que d'afficher un « 0 » trompeur. Le flag
	// scalaire ProvidesTeamMMR reste la source projetée dans le DTO bootstrap
	// (availableTitles[].providesTeamMMR) ; cette cap est le pendant déclaratif
	// dans availableTitles[].capabilities.
	CapTeamMMR Capability = "team_mmr"

	// CapDamageTaken — le titre fournit NATIVEMENT le dégât subi par joueur par
	// match via son API. Forme title-level (capability dans le descripteur) du
	// signal scalaire games.ProvidesDamageTaken(slug) : un titre déclare cette
	// cap SSI ProvidesDamageTaken(slug)==true. Halo Infinite : oui ; Halo 5 :
	// non (l'API carnage h5 ne sert pas damage_taken → no_damage_taken=true).
	// Absente ⇒ les surfaces dérivées (defensive resistance, OCDR) masquent ou
	// neutralisent la valeur. Le flag scalaire ProvidesDamageTaken reste la
	// source projetée dans le DTO bootstrap (availableTitles[].providesDamageTaken) ;
	// cette cap est le pendant déclaratif dans availableTitles[].capabilities.
	CapDamageTaken Capability = "damage_taken"

	// CapWeaponAccuracy — le titre fournit la PRÉCISION PAR ARME (tirs au but /
	// tirs tirés par arme). Halo 5 : oui (ShotsFired/ShotsLanded natifs des events
	// weapon_drop → table weapon_accuracy) ; Halo Infinite : non (pas d'events
	// drop, table non peuplée). Absente ⇒ le front masque le graphe « Précision
	// par arme » de la page Synthèse. Pendant title-level de la capability
	// data-level games.CapWeaponAccuracy ("match.weapon.accuracy"). À NE PAS
	// confondre avec CapWeaponKills (kills par arme, complétude snapshot).
	CapWeaponAccuracy Capability = "weapon_accuracy"

	// CapSpartanCustomizer — le titre fournit des masques d'emblèmes/nameplates
	// recolorisables côté client (personnalisation Spartan : choix d'emblème + couleurs
	// primaire/secondaire/tertiaire, rendu live). Halo 5 only (masques extraits du jeu ;
	// Halo Infinite n'expose pas ces masques recolorisables). Absente ⇒ le front ne
	// propose pas la modale de personnalisation depuis la bannière d'identité.
	CapSpartanCustomizer Capability = "spartan_customizer"

	// CapExpectedStats — le titre fournit NATIVEMENT les stats attendues par match
	// (kills_expected / deaths_expected via l'API skill), permettant l'écart au FDA
	// attendu (réel − attendu) sur Timeseries / Sessions / Escouade. Halo Infinite :
	// oui ; Halo 5 : non (0 kills_expected en shared, aucun modèle d'assists). Absente
	// ⇒ le front masque les 3 charts « Écart au FDA attendu » (dégradation par
	// construction côté Go : champs additifs nil, aucun ErrCapabilityNotSupported).
	CapExpectedStats Capability = "expected_stats"

	// CapWaypointMatchURL - le titre expose une page de detail de match publique
	// sur le site officiel Halo Waypoint (URL construite cote front, helper canonique
	// buildWaypointMatchUrl dans apps/web), permettant un lien direct "Ouvrir sur
	// Halo Waypoint" depuis les tableaux de matchs. DECLAREE PAR LES DEUX TITRES
	// depuis le 2026-07-24 (c5dfb9bfd) : Halo Infinite ici meme, Halo 5 dans
	// config/titles/halo_5/title.toml. Les chemins d'URL DIFFERENT par titre et
	// sont resolus cote front par buildWaypointMatchUrl (segment de titre + bucket
	// "arena/" pour Halo 5) : la difference de format n'est donc pas un motif
	// d'exclusion. Absente => le front masque la colonne/lien correspondant (pas
	// de lien mort).
	CapWaypointMatchURL Capability = "waypoint_match_url"

	// CapObjectiveStats — le titre expose des STATS OBJECTIFS par joueur/match des
	// modes à objectif (CTF/Zones/Oddball) : captures/retours/vols de drapeau, zones
	// capturées/sécurisées, temps porteur, etc. Halo Infinite : oui (extraites du
	// payload GetMatchStats → shared.match_objective_stats). Halo 5 : non déclarée
	// (le carnage n'agrège pas ces objectifs). Absente ⇒ le front masque la section
	// « Objectifs » du scoreboard + les KPI objectifs (Synthèse / Escouade) — pas de
	// section vide. Pendant title-level de la capability serveur `match.objective.stats`
	// (games/adapter.go) qui gouverne le chemin de données ; celle-ci gouverne
	// l'AFFICHAGE (useCapability). Cf. PLAN_V72_OBJECTIVE_STATS.md.
	CapObjectiveStats Capability = "objective_stats"
)

// TitleDescriptor décrit un titre supporté avec ses métadonnées.
type TitleDescriptor struct {
	Slug         string       `json:"slug"`
	Name         string       `json:"name"`
	Provider     string       `json:"provider"` // ex: "halo_infinite", "halo_mcc"
	IconURL      string       `json:"icon_url"`
	Status       Status       `json:"status"`
	Capabilities []Capability `json:"capabilities"`
	IsDefault    bool         `json:"is_default"` // halo_infinite = true

	// IsInternal marque un titre INTERNE/TEST (fixture multi-titre comme
	// synthetic_title_b) : découvert + enregistré + inspectable côté admin, mais
	// JAMAIS exposé dans le switcher utilisateur (cf. PublicTitles). Évite qu'une
	// fixture de test « fuite » dans l'UI prod. Défaut false (vrai titre).
	IsInternal bool `json:"is_internal"`

	// Identifiants plateforme pour le matching de présence (watcher)
	XboxTitleID string `json:"xbox_title_id"` // ex: "1144039928" pour Halo Infinite
	SteamAppID  string `json:"steam_app_id"`  // ex: "1336960" pour Halo Infinite (Steam)

	// PlacementMatches est le nombre de matchs de placement des modes classés du
	// titre (Halo Infinite = 5 depuis Season 3 ; Halo 5 = 10). Externalisé du code
	// (le « 5 »/« 10 » était hardcodé). 0 = non déclaré → le consommateur applique
	// son défaut. NB : Halo Infinite a aussi une table de seuils PAR SAISON (10→5)
	// qui prime ; cette valeur est le défaut du titre.
	PlacementMatches int `json:"placement_matches"`

	// CSRSeasonID est l'identifiant de saison CSR FIXE du titre (overlay descripteur,
	// PMT-4). Pour Halo 5, le service record arena est un agrégat LIFETIME sans
	// saison → "h5-lifetime" (déclaré dans title.toml). Consommé par
	// CSRSeasonIDForTitle (précédence : env > overlay fichier > CE champ > global) :
	// aligne la LECTURE (GetCSRSnapshots WHERE season_id) sur l'ÉCRITURE (sync h5).
	// "" = non déclaré → fallback global (CurrentCSRSeasonID), comportement Infinite.
	CSRSeasonID string `json:"csr_season_id"`

	// MediaFilenamePrefixes déclare les préfixes de nom de fichier de capture qui
	// APPARTIENNENT à ce titre (ex. Halo 5 Game Bar : "Halo_5_Guardians-"). Le dossier
	// captures d'un joueur est PARTAGÉ entre titres : sans routage, chaque titre
	// indexait tous les clips (les clips H5 restaient « Sans match » sous Infinite à
	// perpétuité — DEC-8, plan résidus H5). L'indexeur d'un titre SAUTE les fichiers
	// qui matchent le préfixe d'un AUTRE titre (Registry.ForeignMediaFilenamePrefixes).
	// Vide = le titre ne revendique aucun motif (ses fichiers « génériques » restent
	// indexés par tous, comportement historique).
	MediaFilenamePrefixes []string `json:"media_filename_prefixes"`
}

// HasCapability vérifie si le titre supporte une fonctionnalité.
func (t *TitleDescriptor) HasCapability(cap Capability) bool {
	for _, c := range t.Capabilities {
		if c == cap {
			return true
		}
	}
	return false
}

// IsActive indique si le titre est au statut actif. Prédicat pur (zéro IO) —
// MT-22 (PMT-8). Utilisé par le middleware RequireActiveTitle et tout
// consommateur qui n'a pas le registre sous la main.
func (t *TitleDescriptor) IsActive() bool {
	return t.Status == StatusActive
}

// ---------------------------------------------------------------------------
// TitleRegistry — registre centralisé des titres
// ---------------------------------------------------------------------------

// Registry est le registre centralisé des titres supportés.
type Registry struct {
	mu     sync.RWMutex
	titles map[string]*TitleDescriptor
}

// DefaultSlug est le slug du titre par défaut (Halo Infinite).
const DefaultSlug = "halo_infinite"

// IsDefaultSlug indique si `titleID` désigne le titre par défaut (slug vide normalisé
// inclus). Helper canonique pour la résolution de LAYOUT FS / fixtures (structure
// plate legacy du titre par défaut vs sous-arbre title-scopé). NB : ce n'est PAS un
// gating de FEATURE par slug (interdit par ADR 0025 → passer par une capability) ;
// c'est une décision d'agencement disque, centralisée ici plutôt que dispersée en
// comparaisons brutes `slug == DefaultSlug`.
func IsDefaultSlug(titleID string) bool {
	return titleID == "" || titleID == DefaultSlug
}

// defaultRegistry : registre PARTAGÉ par les helpers package-level (XboxTitleIDFor)
// et les call-sites de service qui n'ont pas d'instance sous la main. Lazy par
// défaut (built-in halo_infinite), mais REMPLAÇABLE au boot par un registre piloté
// par config via SetDefaultRegistry (MT-16 / day-one 2e titre). Les descripteurs
// sont la source UNIQUE de vérité.
var (
	defaultRegistryMu   sync.RWMutex
	defaultRegistryInst *Registry
)

func defaultRegistry() *Registry {
	defaultRegistryMu.RLock()
	r := defaultRegistryInst
	defaultRegistryMu.RUnlock()
	if r != nil {
		return r
	}
	defaultRegistryMu.Lock()
	defer defaultRegistryMu.Unlock()
	if defaultRegistryInst == nil {
		defaultRegistryInst = NewRegistry()
	}
	return defaultRegistryInst
}

// SetDefaultRegistry remplace le registre partagé par défaut (appelé UNE fois au
// boot avec le registre piloté par config : built-in halo_infinite + titres
// additionnels découverts sous config/titles/*). Thread-safe. nil est ignoré
// (garde défensive — on ne vide jamais le registre partagé).
//
// Après cet appel, tous les call-sites passant par DefaultRegistry() (front
// switcher BuildAvailableTitles, résolution de titre, XboxTitleIDFor, etc.) voient
// les titres additionnels. Doit être posé AVANT que le serveur n'accepte du trafic.
func SetDefaultRegistry(reg *Registry) {
	if reg == nil {
		return
	}
	defaultRegistryMu.Lock()
	defaultRegistryInst = reg
	defaultRegistryMu.Unlock()
}

// DefaultRegistry expose le registre partagé. Avant SetDefaultRegistry, retombe
// sur le built-in lazy (halo_infinite seul) — donc mono-titre sûr en test/CLI.
func DefaultRegistry() *Registry { return defaultRegistry() }

// XboxTitleIDFor retourne l'identifiant Xbox numérique (en string) pour un slug
// LevelUp, lu depuis le descripteur du titre (registry-driven, PMT-6 — supprime
// le switch qui dupliquait TitleDescriptor.XboxTitleID). Utilisé lors du sync
// achievements pour filtrer l'API Xbox par titre. "" si le slug est inconnu —
// l'appelant doit gérer ce cas. (Halo Infinite standalone = 2043073184, porté par
// son descripteur ; distinct de 1144039928 = MCC.)
func XboxTitleIDFor(slug string) string {
	return defaultRegistry().XboxTitleIDFor(slug)
}

// XboxTitleIDFor lit l'XboxTitleID du descripteur du titre. "" si inconnu.
func (r *Registry) XboxTitleIDFor(slug string) string {
	if d := r.Get(slug); d != nil {
		return d.XboxTitleID
	}
	return ""
}

// NewRegistry crée un registre avec les titres par défaut.
func NewRegistry() *Registry {
	r := &Registry{
		titles: make(map[string]*TitleDescriptor),
	}
	// Halo Infinite est toujours enregistré.
	r.Register(&TitleDescriptor{
		Slug:     DefaultSlug,
		Name:     "Halo Infinite",
		Provider: DefaultSlug,
		Status:   StatusActive,
		Capabilities: []Capability{
			CapMatchmaking, CapFirefight, CapForge,
			CapMedia, CapRanked, CapCareer, CapSeasonPass, CapAssetImages,
			CapAchievements, CapEngagement, CapLUSR, CapWorldLeaderboard,
			CapWeaponKills,
			// Miroir title-level des flags scalaires : Halo Infinite fournit
			// nativement le MMR d'équipe et le dégât subi
			// (ProvidesTeamMMR / ProvidesDamageTaken == true).
			CapTeamMMR, CapDamageTaken,
			// Stats attendues natives (kills_expected/deaths_expected) → écart au FDA attendu.
			CapExpectedStats,
			// Page de détail de match publique sur Halo Waypoint (I19).
			CapWaypointMatchURL,
			// Stats objectifs par match (CTF/Zones/Oddball) — section scoreboard +
			// KPI Synthèse/Escouade (PLAN_V72_OBJECTIVE_STATS).
			CapObjectiveStats,
		},
		IsDefault:        true,
		XboxTitleID:      "2043073184",
		SteamAppID:       "1336960",
		PlacementMatches: 5, // défaut Halo Infinite depuis Season 3 (table par-saison prime)
	})
	return r
}

// Register ajoute un titre au registre.
func (r *Registry) Register(desc *TitleDescriptor) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.titles[desc.Slug] = desc
}

// Get retourne le descripteur d'un titre ou nil si inconnu.
func (r *Registry) Get(slug string) *TitleDescriptor {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.titles[slug]
}

// Exists vérifie si un titre est enregistré.
func (r *Registry) Exists(slug string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.titles[slug]
	return ok
}

// IsActive vérifie qu'un titre est enregistré ET actif (Status == StatusActive).
// MT-22 (PMT-8) : le Status est enfin consulté. Le gating runtime des routes
// title-scoped passe par le middleware RequireActiveTitle (503 title_unavailable),
// pas par le seam TitleExtractor (qui résout n'importe quel titre connu).
func (r *Registry) IsActive(slug string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	td, ok := r.titles[slug]
	return ok && td.IsActive()
}

// Active retourne uniquement les titres au statut actif.
func (r *Registry) Active() []*TitleDescriptor {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*TitleDescriptor, 0, len(r.titles))
	for _, t := range r.titles {
		if t.IsActive() {
			out = append(out, t)
		}
	}
	return out
}

// NonArchived retourne les titres actifs ET coming_soon (exclut uniquement
// archived). MT-22 (PMT-8) : le switcher front liste les titres jouables +
// ceux « bientôt disponibles » (annotés via leur Status), mais jamais les
// titres retirés (archived). Cf. BuildAvailableTitles.
func (r *Registry) NonArchived() []*TitleDescriptor {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*TitleDescriptor, 0, len(r.titles))
	for _, t := range r.titles {
		if t.Status != StatusArchived {
			out = append(out, t)
		}
	}
	return out
}

// PublicTitles retourne les titres exposables dans le SWITCHER UTILISATEUR :
// exclut les archived (titres retirés) ET les internal (fixtures de test comme
// synthetic_title_b). C'est la vue user-facing — par opposition à NonArchived
// (qui inclut les internes) et All (admin). Conserve les coming_soon non-internes
// (badge « bientôt ») et leur Status. Cf. buildAvailableTitlesFrom.
func (r *Registry) PublicTitles() []*TitleDescriptor {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*TitleDescriptor, 0, len(r.titles))
	for _, t := range r.titles {
		if t.Status == StatusArchived || t.IsInternal {
			continue
		}
		out = append(out, t)
	}
	return out
}

// All retourne tous les titres enregistrés.
func (r *Registry) All() []*TitleDescriptor {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*TitleDescriptor, 0, len(r.titles))
	for _, t := range r.titles {
		out = append(out, t)
	}
	return out
}

// Default retourne le titre par défaut (halo_infinite).
func (r *Registry) Default() *TitleDescriptor {
	return r.Get(DefaultSlug)
}

// ForeignMediaFilenamePrefixes retourne les préfixes de nom de fichier de capture
// revendiqués par les AUTRES titres que slug (DEC-8, plan résidus H5). L'indexeur
// média du titre slug saute les fichiers qui matchent un de ces préfixes : un clip
// « Halo_5_Guardians-* » n'est plus indexé sous halo_infinite. Data-driven : les
// motifs viennent des title.toml (MediaFilenamePrefixes) — aucun slug == en dur.
func (r *Registry) ForeignMediaFilenamePrefixes(slug string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []string
	for _, t := range r.titles {
		if t.Slug == slug {
			continue
		}
		out = append(out, t.MediaFilenamePrefixes...)
	}
	return out
}

// ---------------------------------------------------------------------------
// PathResolver — résolution centralisée des chemins title-aware
// ---------------------------------------------------------------------------

// PathResolver résout les chemins physiques en fonction du titre.
// Tous les chemins passent par ce résolveur — aucun filepath.Join(repoRoot, "data", ...)
// n'est autorisé dans les handlers, services, ops ou tests.
type PathResolver struct {
	repoRoot string
	registry *Registry
}

// NewPathResolver crée un PathResolver.
// Si registry n'est pas fourni, un registre par défaut est utilisé.
func NewPathResolver(repoRoot string, registry ...*Registry) *PathResolver {
	var reg *Registry
	if len(registry) > 0 && registry[0] != nil {
		reg = registry[0]
	} else {
		reg = NewRegistry()
	}
	return &PathResolver{repoRoot: repoRoot, registry: reg}
}

// RepoRoot retourne la racine du repo.
func (p *PathResolver) RepoRoot() string {
	return p.repoRoot
}

// --- Chemins title-aware ---

// TitlesRootDir retourne la racine commune des données par titre (le volume
// data monté). Existe pour sonder que la racine de données est bien présente/
// lisible (diagnostic d'inventaire des bases : distinguer « racine data
// introuvable » — RepoRoot mal résolu / volume non monté — de « aucune base »).
// Ex: data/titles/
func (p *PathResolver) TitlesRootDir() string {
	return filepath.Join(p.repoRoot, "data", "titles")
}

// TitleDataDir retourne le répertoire racine des données d'un titre.
// Ex: data/titles/halo_infinite/
func (p *PathResolver) TitleDataDir(titleSlug string) string {
	return filepath.Join(p.repoRoot, "data", "titles", titleSlug)
}

// WarehouseDir retourne le répertoire warehouse d'un titre.
// Ex: data/titles/halo_infinite/warehouse/
func (p *PathResolver) WarehouseDir(titleSlug string) string {
	return filepath.Join(p.TitleDataDir(titleSlug), "warehouse")
}

// SharedDBPath retourne le chemin de la base partagée d'un titre.
// Ex: data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb
func (p *PathResolver) SharedDBPath(titleSlug string) string {
	return filepath.Join(p.WarehouseDir(titleSlug), "shared_matches_v2.duckdb")
}

// SnapshotsDir retourne le répertoire racine des snapshots Parquet immuables
// d'un titre (durabilité / lecture découplée des écritures, Phase 2).
// Ex: data/titles/halo_infinite/warehouse/snapshots/
func (p *PathResolver) SnapshotsDir(titleSlug string) string {
	return filepath.Join(p.WarehouseDir(titleSlug), "snapshots")
}

// SnapshotVersionDir retourne le répertoire d'une version de snapshot. Le nom
// est zéro-paddé pour que l'ordre lexicographique du système de fichiers
// coïncide avec l'ordre chronologique des versions.
// Ex: data/titles/halo_infinite/warehouse/snapshots/v00000000000000000042/
func (p *PathResolver) SnapshotVersionDir(titleSlug string, version int64) string {
	return filepath.Join(p.SnapshotsDir(titleSlug), fmt.Sprintf("v%020d", version))
}

// SnapshotCurrentManifestPath retourne le chemin du pointeur atomique CURRENT
// désignant la version de snapshot active (flip via os.Rename).
// Ex: data/titles/halo_infinite/warehouse/snapshots/CURRENT.json
func (p *PathResolver) SnapshotCurrentManifestPath(titleSlug string) string {
	return filepath.Join(p.SnapshotsDir(titleSlug), "CURRENT.json")
}

// GlobalXuidAliasesDBPath retourne le chemin de la base globale xbox aliases
// (P5.1, ADR 0008). Le mapping xuid → gamertag est un identifiant Microsoft
// (Xbox Services) qui ne dépend pas du titre — donc DB globale partagée
// par tous les titres.
//
// Ex: data/global/xbox_aliases.duckdb
func (p *PathResolver) GlobalXuidAliasesDBPath() string {
	return filepath.Join(p.repoRoot, "data", "global", "xbox_aliases.duckdb")
}

// GlobalMonitoringDB retourne le chemin de la base monitoring globale
// (persistance du dashboard admin : détections avec cycle de vie, historique
// des crons, runs data-health). Globale et NON per-titre : l'observabilité du
// process et de la machine ne dépend d'aucun titre. Un seul writer = le process
// serveur (aucune CLI concurrente ne l'ouvre en écriture — DC-1 du plan
// monitoring 2026-07).
//
// Ex: data/global/monitoring.duckdb
func (p *PathResolver) GlobalMonitoringDB() string {
	return filepath.Join(p.repoRoot, "data", "global", "monitoring.duckdb")
}

// AdminStateDir retourne le répertoire de l'état runtime PERSISTANT du dashboard
// admin (JSON léger, HORS DuckDB — survit au reboot, invariants anti-ART
// intouchés). Global et NON per-titre : l'état du dashboard (dernier cycle
// post-sync, journal des actions globales) ne dépend d'aucun titre. Un seul
// writer = le process serveur.
//
// Ex: data/global/admin_state/
func (p *PathResolver) AdminStateDir() string {
	return filepath.Join(p.repoRoot, "data", "global", "admin_state")
}

// PostSyncSnapshotPath retourne le chemin du snapshot post-sync persistant
// (timeline + matrice par joueur + horodatage du dernier cycle) — réhydraté au
// boot pour que le dashboard ne soit plus amnésique après un redémarrage (C1).
//
// Ex: data/global/admin_state/post_sync_snapshot.json
func (p *PathResolver) PostSyncSnapshotPath() string {
	return filepath.Join(p.AdminStateDir(), "post_sync_snapshot.json")
}

// ActionJournalPath retourne le chemin du journal des actions globales du
// dashboard admin (dernière exécution / issue / déclencheur par action) — écrit
// par la couche service + le scheduler, survit au reboot (C2).
//
// Ex: data/global/admin_state/action_journal.json
func (p *PathResolver) ActionJournalPath() string {
	return filepath.Join(p.AdminStateDir(), "action_journal.json")
}

// DiskWatchStatePath retourne le chemin de l'état PERSISTANT de la surveillance
// disque (dernier statut confirmé + horodatage de la dernière notification +
// débounce d'amélioration). Persisté hors DuckDB pour survivre au reboot : sans
// cela l'anti-spam de ops.ShouldNotifyDisk repart de zéro à chaque redémarrage
// du conteneur (deploy à chaque push main, crash-loop disque plein) → RAFALE de
// notifications Discord identiques à chaque boot. Un seul writer = le process
// serveur (boucle RunDiskWatchLoop).
//
// Ex: data/global/admin_state/disk_watch_state.json
func (p *PathResolver) DiskWatchStatePath() string {
	return filepath.Join(p.AdminStateDir(), "disk_watch_state.json")
}

// MetadataDBPath retourne le chemin de la base metadata d'un titre.
// Ex: data/titles/halo_infinite/warehouse/metadata.duckdb
func (p *PathResolver) MetadataDBPath(titleSlug string) string {
	return filepath.Join(p.WarehouseDir(titleSlug), "metadata.duckdb")
}

// DemoManifestPath retourne le chemin du manifeste démo FIGÉ d'un couple (gamertag
// source, titre). Committé sous config/ (config curée, ≠ data/demo généré et
// volume-monté), il gèle la sélection de matchs de la démo pour que `seed-demo`
// produise un corpus reproductible (médias associés + sessions représentatives
// préservés à chaque regen).
// Ex: config/demo/JGtm/halo_infinite.json
func (p *PathResolver) DemoManifestPath(gamertag, titleSlug string) string {
	if titleSlug == "" {
		titleSlug = DefaultSlug
	}
	return filepath.Join(p.repoRoot, "config", "demo", gamertag, titleSlug+".json")
}

// SharedPVEDBPath retourne le chemin de la base PvE partagée d'un titre.
// Ex: data/titles/halo_infinite/warehouse/shared_pve.duckdb
func (p *PathResolver) SharedPVEDBPath(titleSlug string) string {
	return filepath.Join(p.WarehouseDir(titleSlug), "shared_pve.duckdb")
}

// SharedSocialDBPath retourne le chemin de la base sociale partagée d'un titre.
// Ex: data/titles/halo_infinite/warehouse/shared_social.duckdb
func (p *PathResolver) SharedSocialDBPath(titleSlug string) string {
	return filepath.Join(p.WarehouseDir(titleSlug), "shared_social.duckdb")
}

// PlayersRootDir retourne le répertoire racine des joueurs d'un titre (source
// unique du sous-chemin "players" — ne pas reconstruire à la main, cf. garde-rail
// archlint/no_players_root_join_test.go).
// Ex: data/titles/halo_infinite/players/
func (p *PathResolver) PlayersRootDir(titleSlug string) string {
	return filepath.Join(p.TitleDataDir(titleSlug), "players")
}

// PlayerDir retourne le répertoire d'un joueur pour un titre.
// Ex: data/titles/halo_infinite/players/Chocoboflor/
func (p *PathResolver) PlayerDir(titleSlug, gamertag string) string {
	return filepath.Join(p.PlayersRootDir(titleSlug), gamertag)
}

// PlayerDBPath retourne le chemin de la DB stats d'un joueur.
// Ex: data/titles/halo_infinite/players/Chocoboflor/stats.duckdb
func (p *PathResolver) PlayerDBPath(titleSlug, gamertag string) string {
	return filepath.Join(p.PlayerDir(titleSlug, gamertag), "stats.duckdb")
}

// PlayerArchiveDir retourne le répertoire d'archive d'un joueur.
// Ex: data/titles/halo_infinite/players/Chocoboflor/archive/
func (p *PathResolver) PlayerArchiveDir(titleSlug, gamertag string) string {
	return filepath.Join(p.PlayerDir(titleSlug, gamertag), "archive")
}

// PlayerCapturesDir retourne le répertoire captures d'un joueur.
// Ex: data/titles/halo_infinite/players/Chocoboflor/captures/
func (p *PathResolver) PlayerCapturesDir(titleSlug, gamertag string) string {
	return filepath.Join(p.PlayerDir(titleSlug, gamertag), "captures")
}

// PlayerMediaAudioConfigPath retourne le chemin du sidecar de réglage audio média
// d'un joueur (rôle voix/jeu/autres des pistes source, mode auto/manuel). Placé à
// côté du dossier captures dans le répertoire joueur du titre.
// Ex: data/titles/halo_infinite/players/Chocoboflor/media_audio_config.json
func (p *PathResolver) PlayerMediaAudioConfigPath(titleSlug, gamertag string) string {
	return filepath.Join(p.PlayerDir(titleSlug, gamertag), "media_audio_config.json")
}

// ResolveCapturesDir centralise la résolution du dossier captures d'un joueur.
//
// Tous les call-sites manipulant un dossier de médias (upload, scan, reindex,
// URL serving, CLI) DOIVENT passer par cette fonction pour garantir un
// comportement uniforme et éviter qu'un fallback hardcodé crée un dossier
// fantôme jamais relu.
//
// Règle :
//   - baseDir non vide (settings.MediaCapturesBaseDir) → {baseDir}/{gamertag}
//   - baseDir vide → fallback interne PlayerCapturesDir = .../{gamertag}/captures
func (p *PathResolver) ResolveCapturesDir(titleSlug, gamertag, baseDir string) string {
	if baseDir != "" {
		return filepath.Join(baseDir, gamertag)
	}
	return p.PlayerCapturesDir(titleSlug, gamertag)
}

// MediaDataDir retourne la base média interne canonique {repoRoot}/data/media.
//
// Layout title-agnostic et PLAT : data/media/{gamertag}/{thumbs,hls,...} — distinct
// du PlayerCapturesDir per-titre (data/titles/.../{gamertag}/captures). C'est la
// localisation réelle des médias en production (montée sur /app/data/media dans le
// conteneur). Sert de fallback de résolution quand le media_captures_base_dir
// configuré est absent ou invalide (garde-fou incident prod 2026-06-13 : un chemin
// Windows déployé sur le VPS Linux rendait TOUS les médias illisibles).
func (p *PathResolver) MediaDataDir() string {
	return filepath.Join(p.repoRoot, "data", "media")
}

// BackupDir retourne le répertoire de backup d'un joueur pour un titre.
// Ex: data/titles/halo_infinite/backups/Chocoboflor/
func (p *PathResolver) BackupDir(titleSlug, gamertag string) string {
	return filepath.Join(p.TitleDataDir(titleSlug), "backups", gamertag)
}

// DemoFixturesDir retourne le répertoire de fixtures démo pour un titre.
// Ex: tests/fixtures/titles/halo_infinite/ref_player/
func (p *PathResolver) DemoFixturesDir(titleSlug string) string {
	return filepath.Join(p.repoRoot, "tests", "fixtures", "titles", titleSlug, "ref_player")
}

// --- Chemins globaux (non namespacés par titre) ---

// SessionDir retourne le répertoire des sessions.
// Ex: data/sessions/
func (p *PathResolver) SessionDir() string {
	return filepath.Join(p.repoRoot, "data", "sessions")
}

// CacheRootDir retourne le répertoire racine du cache applicatif (jobs, cache
// BP/challenges, release notes...). Source unique du sous-chemin "data/cache".
// Ex: data/cache/
func (p *PathResolver) CacheRootDir() string {
	return filepath.Join(p.repoRoot, "data", "cache")
}

// JobsCachePath retourne le chemin du cache des jobs.
// Ex: data/cache/jobs.json
func (p *PathResolver) JobsCachePath() string {
	return filepath.Join(p.CacheRootDir(), "jobs.json")
}

// ReplayArtifactPath retourne le chemin de l'artefact de rejeu 2D d'un match (produit
// hors ligne par cmd/replay-build, servi tel quel par l'API).
// Ex: data/cache/replays/halo_infinite/000d5950.json
// Sous data/cache/* (non versionné, regénérable depuis capture + film).
//
// La clé est la forme COURTE (cf. FilmShortMatchID) : l'artefact est un dérivé du film,
// et tout ce qui dérive du film est rangé sous cette forme. Passer le match_id complet ou
// sa forme courte donne donc le MÊME chemin — c'est ce qui rend l'artefact atteignable
// depuis une route de l'application.
func (p *PathResolver) ReplayArtifactPath(titleSlug, matchID string) string {
	return filepath.Join(p.ReplayArtifactsDir(titleSlug), FilmShortMatchID(matchID)+".json")
}

// ReplayArtifactsDir retourne le dossier des artefacts de rejeu 2D d'un titre (un
// fichier {short8}.json par match). Sert la purge récurrente (replay_purge_cron) — qui
// supprime des ARTEFACTS, jamais des films.
func (p *PathResolver) ReplayArtifactsDir(titleSlug string) string {
	return filepath.Join(p.repoRoot, "data", "cache", "replays", titleSlug)
}

// MapQuantBoundsPath retourne le chemin du catalogue des bornes de quantification par
// carte (AABB des BSP extraites des modules du jeu, cf. filmdec.MapQuantCatalog). Donnée
// de RÉFÉRENCE versionnée — pas un cache : elle ne se régénère qu'avec les fichiers du
// jeu installé (cmd/mapquant-build).
// Ex: data/titles/halo_infinite/reference/map_quant_bounds.json
func (p *PathResolver) MapQuantBoundsPath(titleSlug string) string {
	return filepath.Join(p.TitleDataDir(titleSlug), "reference", "map_quant_bounds.json")
}

// MapObjectivesPath retourne le chemin du catalogue des objets d'objectif par carte
// (socles de drapeau, points de livraison, sockets Stockpile, zones...), extrait des
// variantes de carte UGC (.mvar) par cmd/mapobj-build. Donnée de RÉFÉRENCE versionnée
// — pas un cache : elle fige un appel réseau par carte pour que le rejeu reste
// 100 % hors ligne à l'exécution.
// Ex: data/titles/halo_infinite/reference/map_objectives.json
func (p *PathResolver) MapObjectivesPath(titleSlug string) string {
	return filepath.Join(p.TitleDataDir(titleSlug), "reference", "map_objectives.json")
}

// MapWeaponPadsPath retourne le chemin du catalogue des EMPLACEMENTS DE SOCLE par carte
// (socles d'arme et de power-up), extrait des mêmes variantes de carte UGC (.mvar) par
// cmd/mapopads-build. Donnée de RÉFÉRENCE versionnée — pas un cache.
//
// IL NE SE SERT JAMAIS SEUL : le fichier de carte POSE les socles, le mode les ALLUME
// (Cliffhanger : 17 posés, 10 en CTF, 0 en Super Fiesta). Le calque servi au rejeu ne
// publie donc que les emplacements qu'un socle du MATCH confirme (replay/map_weapon_pads.go).
// Ex: data/titles/halo_infinite/reference/map_weapon_pads.json
func (p *PathResolver) MapWeaponPadsPath(titleSlug string) string {
	return filepath.Join(p.TitleDataDir(titleSlug), "reference", "map_weapon_pads.json")
}

// MapCalloutsPath retourne le chemin du catalogue des CALLOUTS (zones nommées
// officielles) par carte : polygones monde, tranche verticale et libellés FR/EN, extraits
// du tag levl des modules du jeu par cmd/mapcallouts-build. Donnée de RÉFÉRENCE
// versionnée — pas un cache : elle ne se régénère qu'avec les fichiers du jeu installé.
// La clé d'entrée est le nom de MODULE, celui de map_quant_bounds.json.
// Ex: data/titles/halo_infinite/reference/map_callouts.json
func (p *PathResolver) MapCalloutsPath(titleSlug string) string {
	return filepath.Join(p.TitleDataDir(titleSlug), "reference", "map_callouts.json")
}

// MapGeometryDir retourne le répertoire des PROPS de carte (géométrie Forge : socles,
// caisses, rampes posées dans la variante), lus par cmd/replay-build pour poser des
// repères contextuels sur le rejeu 2D — à distinguer de la STRUCTURE, qui est le sol.
// Donnée de RÉFÉRENCE versionnée (produite par le RE de la variante .mvar), au même
// titre que map_structure/ et map_quant_bounds.json : ce n'est pas un cache.
// Ex: data/titles/halo_infinite/reference/map_geometry/
func (p *PathResolver) MapGeometryDir(titleSlug string) string {
	return filepath.Join(p.TitleDataDir(titleSlug), "reference", "map_geometry")
}

// MapStructurePath retourne le chemin du fichier de STRUCTURE d'une carte : les emprises
// (AABB monde) des instances de géométrie instanciée de son BSP — sols, plateformes,
// rampes, murs. Un fichier PAR MODULE (et non un catalogue unique) : une carte pèse 10 000
// à 26 000 emprises, les agréger produirait un blob de plusieurs Mo rechargé en entier
// pour un seul match. La clé est le nom de MODULE, celui déjà porté par
// map_quant_bounds.json — le lien nom affiché -> module reste déclaré à un seul endroit.
// Donnée de RÉFÉRENCE versionnée (cmd/mapstruct-build, exige les fichiers du jeu).
// Ex: data/titles/halo_infinite/reference/map_structure/ridgeline.json
func (p *PathResolver) MapStructurePath(titleSlug, module string) string {
	return filepath.Join(p.TitleDataDir(titleSlug), "reference", "map_structure", module+".json")
}

// MapBackgroundDir retourne le répertoire des FONDS DE CARTE : l'image vue du dessus de
// chaque carte, reconstruite depuis les triangles du maillage de rendu de son .module.
// À distinguer de map_structure/, qui ne publie que des boîtes englobantes (AABB) et ne
// dessine aucune forme. Donnée de RÉFÉRENCE versionnée (cmd/mapfond-build, exige les
// fichiers du jeu). Ex: data/titles/halo_infinite/reference/map_backgrounds/
func (p *PathResolver) MapBackgroundDir(titleSlug string) string {
	return filepath.Join(p.TitleDataDir(titleSlug), "reference", "map_backgrounds")
}

// MapBackgroundPath retourne le chemin de l'IMAGE du fond de carte d'un module (PNG RGBA,
// fond transparent). La clé est le nom de MODULE, celui déjà porté par map_quant_bounds.json
// et map_structure/ — le lien nom affiché -> module reste déclaré à un seul endroit.
// Ex: data/titles/halo_infinite/reference/map_backgrounds/ridgeline.png
func (p *PathResolver) MapBackgroundPath(titleSlug, module string) string {
	return filepath.Join(p.MapBackgroundDir(titleSlug), module+".png")
}

// MapBackgroundMetaPath retourne le chemin du SIDECAR de calage du fond de carte
// (replay.MapBackground) : mètres par pixel, origine monde, convention monde->pixel, stats
// de cuisson. Un sidecar PAR CARTE et non un manifeste unique : une re-cuisson partielle ne
// peut alors jamais abîmer l'entrée d'une autre carte, et chaque PNG porte sa propre vérité
// — le défaut exact de carte_validee_v1.png, dont le calage a dû être retrouvé à la main.
// Ex: data/titles/halo_infinite/reference/map_backgrounds/ridgeline.json
func (p *PathResolver) MapBackgroundMetaPath(titleSlug, module string) string {
	return filepath.Join(p.MapBackgroundDir(titleSlug), module+".json")
}

// SyncCacheDir retourne le répertoire racine du cache fetch intermédiaire
// (Phase 2 refactor Collect→Persist). Chaque cycle de sync crée un
// sous-dossier `{cycle_id}/` à l'intérieur. Ex: data/sync_cache/
func (p *PathResolver) SyncCacheDir() string {
	return filepath.Join(p.repoRoot, "data", "sync_cache")
}

// WALDir retourne le répertoire WAL de la BatchQueue persist (Phase 4.7
// closure). Contient les batches JSON pending entre Submit et ACK Persister.
// Recovery au boot via persist.BatchQueue.RecoverPending. Ex: data/wal/
func (p *PathResolver) WALDir() string {
	return filepath.Join(p.repoRoot, "data", "wal")
}

// TitleMappingsDir retourne le répertoire des mappings VERSIONNÉS d'un titre
// (config/titles/<slug>/mappings/) : fields, assets, outcomes, capabilities,
// weapon_names, replay_labels… C'est du CONFIG versionné en Git, pas de la donnée
// sous data/ — d'où la racine distincte. slug vide → titre par défaut.
func (p *PathResolver) TitleMappingsDir(slug string) string {
	if slug == "" {
		slug = DefaultSlug
	}
	return filepath.Join(p.repoRoot, "config", "titles", slug, "mappings")
}

// DBProfilesPath retourne le chemin de db_profiles.json (global).
func (p *PathResolver) DBProfilesPath() string {
	return filepath.Join(p.repoRoot, "db_profiles.json")
}

// AppSettingsPath retourne le chemin de app_settings.json (global).
func (p *PathResolver) AppSettingsPath() string {
	return filepath.Join(p.repoRoot, "app_settings.json")
}

// TitleSettingsPath retourne le chemin de l'overlay de settings d'un titre
// (data/titles/<slug>/settings.json) — miroir per-titre d'AppSettingsPath
// (global), point d'extension du seam PMT-4 « base globale + overlay par titre ».
// Le fichier est OPTIONNEL : absent ⇒ le titre hérite intégralement du global.
// slug vide → titre par défaut.
func (p *PathResolver) TitleSettingsPath(slug string) string {
	if slug == "" {
		slug = DefaultSlug
	}
	return filepath.Join(p.repoRoot, "data", "titles", slug, "settings.json")
}

// WatcherTokensPath retourne le chemin du fichier de tokens watcher.
// Ex: data/auth/watcher_tokens.json
//
// LEGACY (mono-user) : conservé pour le watcher daemon historique.
// Pour le SSO Xbox multi-user, utiliser WatcherTokensDir() + 1 fichier par XUID.
func (p *PathResolver) WatcherTokensPath() string {
	return filepath.Join(p.repoRoot, "data", "auth", "watcher_tokens.json")
}

// WatcherTokensDir retourne le répertoire des tokens watcher multi-user (SSO Xbox).
// Store GLOBAL title-agnostic : les tokens d'auth (refresh-token / XSTS) sont attachés
// au compte Xbox (xuid), PAS à un jeu — un même compte donne accès à tous les titres.
// Layout : data/auth/watcher_tokens/{xuid}.json (1 fichier par utilisateur).
// Décision D4 (cf. SPRINT_XBOX_SSO §0bis) : source unique, write-to-temp + rename atomique,
// perms 0600 sur fichiers / 0700 sur répertoire.
func (p *PathResolver) WatcherTokensDir() string {
	return filepath.Join(p.repoRoot, "data", "auth", "watcher_tokens")
}

// --- Validation ---

// ValidateTitle vérifie qu'un slug est valide dans le registre.
func (p *PathResolver) ValidateTitle(slug string) error {
	if slug == "" {
		return fmt.Errorf("title_slug vide")
	}
	if !p.registry.Exists(slug) {
		return fmt.Errorf("titre inconnu : %q", slug)
	}
	return nil
}
