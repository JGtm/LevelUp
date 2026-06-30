package config

import (
	"encoding/json"
	"fmt"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/title"
	"os"
	"path/filepath"
)

type dbProfilesFile struct {
	Version  string                    `json:"version"`
	Profiles map[string]dbProfileEntry `json:"profiles"`
}

// dbProfilesFileV3 représente le format v3 title-aware de db_profiles.json.
// Structure : { "version": "3.0", "admin": "<gamertag>", "profiles": { "<title_slug>": { "<gamertag>": {...} } } }
type dbProfilesFileV3 struct {
	Version  string                               `json:"version"`
	Admin    string                               `json:"admin"`
	Profiles map[string]map[string]dbProfileEntry `json:"profiles"`
}

// dbProfileEntry représente une entrée dans la map "profiles" de db_profiles.json.
type dbProfileEntry struct {
	DBPath         string `json:"db_path"`
	XUID           string `json:"xuid"`
	WaypointPlayer string `json:"waypoint_player,omitempty"`
	// SyncEnabled : nil ou true = titre actif pour ce joueur ; false = sync en PAUSE
	// (les données restent sur disque, réactivable sans re-sync). Cf. Pass B.
	SyncEnabled *bool `json:"sync_enabled,omitempty"`
	// InitialMaxMatches : nombre de matchs à synchroniser à l'onboarding pour ce
	// (joueur, titre). 0 = défaut (200, borné par le handler de sync initial).
	InitialMaxMatches int `json:"initial_max_matches,omitempty"`
	// AuthOnly marque un profil qui n'existe que pour la gestion des tokens auth
	// (aucun suivi de stats — pas un vrai joueur). Il reste visible côté serveur
	// (pool d'auth, token-capture/import, rotation) mais est exclu des listes
	// front-facing (sélecteur L1, favoris gamertag Escouade/Explorer).
	AuthOnly bool `json:"auth_only,omitempty"`
}

// LoadPlayers charge db_profiles.json et retourne la liste des joueurs.
// Supporte les formats v2.1 (flat) et v3.0 (title-scoped).
// Si titleFilter est non vide, ne retourne que les joueurs de ce titre.
func (c *AppConfig) LoadPlayers(titleFilter ...string) ([]domain.PlayerSummary, error) {
	if c.DemoMode {
		titleSlug := title.DefaultSlug
		if len(titleFilter) > 0 && titleFilter[0] != "" {
			titleSlug = titleFilter[0]
		}
		// Stack démo : DemoPlayer + les 2 coéquipiers (DemoPlayer2/3) dont la
		// player DB a été seedée POUR CE TITRE. Permet à la page Escouade de résoudre
		// un coéquipier vers SA player DB (perf/LUSR) via resolveByGT. Title-aware :
		// un roster peut différer par titre (un titre additionnel a son propre sous-arbre
		// data/demo/titles/{slug}/players/).
		titleDir := demoTitleDir(c.DemoFixturesDir, titleSlug)
		var out []domain.PlayerSummary
		for _, d := range DemoRoster {
			if _, err := os.Stat(filepath.Join(titleDir, "players", d.Dir, "stats.duckdb")); err != nil {
				continue // coéquipier non seedé pour ce titre
			}
			out = append(out, domain.PlayerSummary{
				PlayerSlug:     d.Slug,
				Gamertag:       d.Gamertag,
				XUID:           d.XUID,
				WaypointPlayer: d.Gamertag,
				IsDemo:         true,
				TitleSlug:      titleSlug,
				SyncEnabled:    true,
			})
		}
		if len(out) == 0 && title.IsDefaultSlug(titleSlug) {
			// Fallback (fixtures plates legacy, titre par défaut) : au moins le main.
			out = append(out, domain.PlayerSummary{
				PlayerSlug: "demo-player", Gamertag: "DemoPlayer", XUID: DemoRoster[0].XUID,
				WaypointPlayer: "DemoPlayer", IsDemo: true, TitleSlug: titleSlug, SyncEnabled: true,
			})
		}
		return out, nil
	}

	data, err := os.ReadFile(c.DBProfilesPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []domain.PlayerSummary{}, nil
		}
		return nil, fmt.Errorf("lecture db_profiles.json : %w", err)
	}

	// Détecter la version pour choisir le parser.
	var versionProbe struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(data, &versionProbe); err != nil {
		return nil, fmt.Errorf("parsing db_profiles.json : %w", err)
	}

	if versionProbe.Version == "3.0" {
		return c.loadPlayersV3(data, titleFilter...)
	}
	return c.loadPlayersV2(data, titleFilter...)
}

// AdminPlayer retourne le gamertag désigné comme admin dans db_profiles.json (champ "admin").
// Retourne "" si le fichier est absent, illisible ou si le champ n'est pas défini (format v2).
func (c *AppConfig) AdminPlayer() string {
	data, err := os.ReadFile(c.DBProfilesPath)
	if err != nil {
		return ""
	}
	var f dbProfilesFileV3
	if err := json.Unmarshal(data, &f); err != nil {
		return ""
	}
	return f.Admin
}

// loadPlayersV2 parse le format v2.1 (flat map gamertag → entry).
func (c *AppConfig) loadPlayersV2(data []byte, titleFilter ...string) ([]domain.PlayerSummary, error) {
	var file dbProfilesFile
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("parsing db_profiles.json v2 : %w", err)
	}

	// En v2.1, tous les profils sont implicitement halo_infinite.
	filter := ""
	if len(titleFilter) > 0 {
		filter = titleFilter[0]
	}
	if filter != "" && filter != title.DefaultSlug {
		return []domain.PlayerSummary{}, nil
	}

	players := make([]domain.PlayerSummary, 0, len(file.Profiles))
	for gamertag, p := range file.Profiles {
		wp := p.WaypointPlayer
		if wp == "" {
			wp = gamertag
		}
		players = append(players, domain.PlayerSummary{
			PlayerSlug:     gamertag,
			Gamertag:       gamertag,
			XUID:           p.XUID,
			WaypointPlayer: wp,
			IsDemo:         false,
			TitleSlug:      title.DefaultSlug,
			SyncEnabled:    true, // v2.1 : pas de notion de pause → toujours actif
			AuthOnly:       p.AuthOnly,
		})
	}
	return players, nil
}

// loadPlayersV3 parse le format v3.0 (title_slug → gamertag → entry).
func (c *AppConfig) loadPlayersV3(data []byte, titleFilter ...string) ([]domain.PlayerSummary, error) {
	var file dbProfilesFileV3
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("parsing db_profiles.json v3 : %w", err)
	}

	filter := ""
	if len(titleFilter) > 0 {
		filter = titleFilter[0]
	}

	var players []domain.PlayerSummary
	for titleSlug, titleProfiles := range file.Profiles {
		if filter != "" && titleSlug != filter {
			continue
		}
		for gamertag, p := range titleProfiles {
			wp := p.WaypointPlayer
			if wp == "" {
				wp = gamertag
			}
			// nil/true = actif (rétrocompat : entrées existantes sans le champ).
			syncEnabled := p.SyncEnabled == nil || *p.SyncEnabled
			players = append(players, domain.PlayerSummary{
				PlayerSlug:        gamertag,
				Gamertag:          gamertag,
				XUID:              p.XUID,
				WaypointPlayer:    wp,
				IsDemo:            false,
				TitleSlug:         titleSlug,
				SyncEnabled:       syncEnabled,
				InitialMaxMatches: p.InitialMaxMatches,
				AuthOnly:          p.AuthOnly,
			})
		}
	}
	return players, nil
}

// LoadAppSettings charge app_settings.json. Retourne une map vide si absent.
