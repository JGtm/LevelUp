// Package playerdirectory — l'annuaire des joueurs (ADR 0035 D2/D7).
//
// Quatre registres décrivent un joueur, chacun avec son cycle de vie : le compte
// (`data/auth/users.json`), les credentials (`data/auth/watcher_tokens/{xuid}.json`),
// le profil de suivi (`db_profiles.json`) et le suivi live (le daemon watcher).
// Un cinquième témoin, le disque, dit ce qui a déjà été écrit pour un joueur.
// Chaque consommateur du code lisait jusqu'ici SON registre : un compte sans
// profil n'apparaissait donc nulle part — c'est ce qui a laissé passer
// l'incident du 2026-07-23 (ADR 0035 §Context).
//
// Ce paquet les lit ENSEMBLE, par xuid, et rend une ligne par identité avec ses
// anomalies typées. Il n'écrit rien, ne touche JAMAIS à l'entrepôt partagé et
// n'importe aucun paquet DuckDB : les cinq sources entrent par de petites
// interfaces de lecture, ce qui rend la composition testable sans fichier ni
// base (`directory_test.go`).
package playerdirectory

import (
	"context"
	"sort"
	"strings"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/port"
)

// Compteurs d'en-tête de AdminIdentitiesResponse.Counts, à côté d'un compteur
// par code d'anomalie.
const (
	countIdentities = "identities"
	countWarnings   = "warnings"
	countInfos      = "infos"
)

// ProfilesReader lit les profils de suivi (`db_profiles.json`). Implémenté par
// *config.AppConfig. La définition de « suivi » (domain.SyncablePlayers via
// config.AppConfig.HasTrackedProfile) est celle que lisent les portes de l'ADR
// 0035 D3 ; l'annuaire ne la redéfinit pas et ne la ré-expose pas (revue du
// 2026-09-16 : une délégation sans appelant était du code mort).
type ProfilesReader interface {
	LoadPlayers(titleFilter ...string) ([]domain.PlayerSummary, error)
}

// AccountsReader lit les comptes de connexion. Implémenté par *userstore.Store.
type AccountsReader interface {
	List() ([]domain.AdminUserSummary, error)
}

// TokensReader lit les credentials persistés (ADR 0023). Implémenté par
// *auth.MultiUserTokenStore. L'annuaire n'en extrait jamais un secret.
type TokensReader interface {
	LoadAll() (map[string]*auth.UserTokens, error)
}

// WatchedReader lit le suivi live. Implémenté par *watcher.Daemon
// (WatchedPlayers). nil = pas de watcher dans ce process (CLI, serveur sans
// watcher) : l'annuaire se lit alors sur les trois registres fichiers.
type WatchedReader interface {
	WatchedPlayers() []domain.WatchedPlayerRef
}

// FS est l'adaptateur disque de l'annuaire : il CONSTATE ce qui a été écrit pour
// un joueur (profil ou pas) et, à la purge seulement, retire un dossier que
// l'annuaire vient de trouver orphelin. Tous les chemins passent par
// PathResolver, et aucune base n'est jamais ouverte (cf. fs.go).
type FS interface {
	PlayerDirExists(titleSlug, key string) bool
	PlayerDBExists(titleSlug, key string) bool
	PlayerDBPath(titleSlug, key string) string
	ListPlayerDirs(titleSlug string) ([]string, error)
	RemovePlayerDir(titleSlug, name string) error
}

// Deps porte les sources de l'annuaire. Profiles est la seule obligatoire pour
// la LECTURE : sans elle il n'y a pas de notion de profil suivi, donc pas
// d'anomalie qui ait un sens. Les autres nil ⇒ leur registre est simplement
// absent de la lecture.
type Deps struct {
	Profiles ProfilesReader
	Accounts AccountsReader
	Tokens   TokensReader
	Watched  WatchedReader
	FS       FS
	// Creator écrit le profil de suivi (ADR 0035 D4). nil ⇒ Onboard refuse :
	// mieux vaut une erreur franche qu'un profil silencieusement non créé.
	Creator ProfileCreator
	// Watcher prend le joueur en charge en live après la création du profil.
	// nil ⇒ pas de watcher dans ce process (CLI, serveur sans watcher) : le
	// profil suffit, `initPlayers` reprendra le joueur au prochain démarrage.
	Watcher WatcherNotifier
	// Purge porte les écritures DESTRUCTRICES (cf. purge.go). Regroupées : ce
	// sont les seules de l'annuaire, et une purge incomplète est pire qu'une
	// purge refusée — les voir ensemble rend l'oubli visible au câblage.
	Purge PurgeDeps
	// Titles : slugs balayés pour le témoin disque. Vide ⇒ tous les titres du
	// registre par défaut (jamais une comparaison de slug, cf. CLAUDE.md
	// « Multi-titre »).
	Titles []string
	// Now : seam d'horloge pour les tests. nil ⇒ time.Now.
	Now func() time.Time
}

// Directory implémente port.PlayerDirectory : la lecture unifiée des registres
// et le chemin unique d'entrée d'une identité (`Onboard`, cf. onboard.go).
type Directory struct {
	profiles ProfilesReader
	accounts AccountsReader
	tokens   TokensReader
	watched  WatchedReader
	fs       FS
	creator  ProfileCreator
	watcher  WatcherNotifier
	purge    PurgeDeps
	titles   []string
	now      func() time.Time
}

var _ port.PlayerDirectory = (*Directory)(nil)

// New construit l'annuaire. Les dépendances absentes sont tolérées (cf. Deps).
func New(d Deps) *Directory {
	titles := d.Titles
	if len(titles) == 0 {
		for _, desc := range title.DefaultRegistry().All() {
			titles = append(titles, desc.Slug)
		}
	}
	now := d.Now
	if now == nil {
		now = time.Now
	}
	return &Directory{
		profiles: d.Profiles,
		accounts: d.Accounts,
		tokens:   d.Tokens,
		watched:  d.Watched,
		fs:       d.FS,
		creator:  d.Creator,
		watcher:  d.Watcher,
		purge:    d.Purge,
		titles:   titles,
		now:      now,
	}
}

// List rend une ligne par identité connue d'au moins un registre, ses anomalies
// et les compteurs d'en-tête.
//
// Ordre rendu : les identités porteuses d'au moins une anomalie `warning`
// d'abord (c'est ce qu'un administrateur ouvre la page pour voir), puis par
// gamertag, puis par xuid — un ordre TOTAL et stable, sans quoi l'itération de
// map ferait danser le tableau à chaque rafraîchissement.
func (d *Directory) List(ctx context.Context) (domain.AdminIdentitiesResponse, error) {
	records, err := d.collect(ctx)
	if err != nil {
		return domain.AdminIdentitiesResponse{}, err
	}
	for i := range records {
		records[i].Anomalies = computeAnomalies(records[i])
	}
	sortRecords(records)
	return domain.AdminIdentitiesResponse{
		GeneratedAt: d.now().UTC().Format(time.RFC3339),
		Identities:  records,
		Counts:      countAnomalies(records),
	}, nil
}

// Get rend l'identité d'un xuid, ou port.ErrIdentityNotFound. La recomposition
// complète est assumée : les registres se comptent en dizaines d'entrées, et
// c'est le prix d'une seule définition de « ce que l'on sait d'un joueur ».
func (d *Directory) Get(ctx context.Context, xuid string) (domain.IdentityRecord, error) {
	if xuid == "" {
		return domain.IdentityRecord{}, port.ErrIdentityNotFound
	}
	resp, err := d.List(ctx)
	if err != nil {
		return domain.IdentityRecord{}, err
	}
	for _, rec := range resp.Identities {
		if rec.XUID == xuid {
			return rec, nil
		}
	}
	// Repli : une identité SANS xuid (dossier joueur orphelin, profil legacy) se
	// désigne par son gamertag — sinon `identity purge` ne saurait pas retirer le
	// résidu que `identity list` signale (revue adversariale du 2026-09-16). Une
	// identité qui a un xuid ne se désigne que par lui : pas d'ambiguïté possible.
	for _, rec := range resp.Identities {
		if rec.XUID == "" && strings.EqualFold(rec.Gamertag, xuid) {
			return rec, nil
		}
	}
	return domain.IdentityRecord{}, port.ErrIdentityNotFound
}

// sortRecords impose l'ordre total documenté sur List.
func sortRecords(records []domain.IdentityRecord) {
	sort.SliceStable(records, func(i, j int) bool {
		wi, wj := hasWarning(records[i]), hasWarning(records[j])
		if wi != wj {
			return wi
		}
		gi, gj := strings.ToLower(records[i].Gamertag), strings.ToLower(records[j].Gamertag)
		if gi != gj {
			return gi < gj
		}
		if records[i].XUID != records[j].XUID {
			return records[i].XUID < records[j].XUID
		}
		// Comptes sans identité Xbox (ni gamertag ni xuid) : l'ordre de
		// userstore.List est celui d'une map, donc aléatoire — sans ce dernier
		// critère, ces lignes permutaient à chaque rafraîchissement (revue
		// adversariale du 2026-09-16).
		return accountName(records[i]) < accountName(records[j])
	})
}

// accountName rend le nom du compte principal en minuscules, "" sans compte.
func accountName(rec domain.IdentityRecord) string {
	if rec.Account == nil {
		return ""
	}
	return strings.ToLower(rec.Account.Username)
}

func hasWarning(rec domain.IdentityRecord) bool {
	for _, a := range rec.Anomalies {
		if a.Severity == domain.AnomalySeverityWarning {
			return true
		}
	}
	return false
}

// countAnomalies construit les compteurs d'en-tête : le total d'identités, les
// deux totaux par sévérité, et un compteur par code d'anomalie PRÉSENT (un code
// absent vaut zéro, il n'encombre pas la réponse).
func countAnomalies(records []domain.IdentityRecord) map[string]int {
	counts := map[string]int{
		countIdentities: len(records),
		countWarnings:   0,
		countInfos:      0,
	}
	for _, rec := range records {
		for _, a := range rec.Anomalies {
			counts[a.Code]++
			if a.Severity == domain.AnomalySeverityWarning {
				counts[countWarnings]++
				continue
			}
			counts[countInfos]++
		}
	}
	return counts
}
