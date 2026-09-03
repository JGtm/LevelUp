package main

// lecture.go — LES DEUX SEULES SOURCES DE L'INVENTAIRE, et pourquoi ce sont celles-la.
//
// `match_registry` dit ce qui a ete JOUE : la carte (map_id + libelle), la playlist, le drapeau
// Firefight, le nombre de joueurs. C'est la seule table qui relie une carte a un MODE, et c'est
// exactement ce qui manquait pour trancher « cette carte sans fond est-elle du BTB ou du
// Firefight ? » — l'inventaire UGC ne porte que `forge`/`native`, et les vignettes de
// `static/maps` existent pour toutes les cartes quel que soit le mode.
//
// `asset_translations` (metadata) donne le nom canonique d'un asset meme quand `map_name` porte
// un UUID brut. C'est le PREMIER nom candidat de `ReplayMapRepo.MapKeysForMatch` : l'inventaire
// doit lui donner les memes candidats que la production, sinon il repond sur une autre question
// que celle qui se pose a l'ecran.
//
// LES DEUX BASES SONT OUVERTES EN LECTURE SEULE via `duckdb.OpenReadForQuery` (CLAUDE.md, regle
// des ecritures DuckDB n°4) : le serveur de dev tient couramment ces fichiers en RW, et forcer
// un `OpenReadOnly` sur un fichier tenu par un autre process est precisement ce que ce helper
// evite.

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"

	"levelup/go-api/internal/platform/duckdb"
)

// carteJouee agrege tout ce que le registre sait d'une identite de carte jouee.
type carteJouee struct {
	MapID      string
	NomBrut    string // `match_registry.map_name`, tel quel
	NomAsset   string // `asset_translations`, preference EN — le premier candidat de la prod
	Matchs     int
	Firefight  int
	JoueursMax int
	Dernier    time.Time
	Playlists  map[string]int
}

// Cle identifie la carte dans l'inventaire : le map_id quand il existe, sinon le libelle.
func (c carteJouee) Cle() string {
	if c.MapID != "" {
		return c.MapID + "|" + c.NomBrut
	}
	return "|" + c.NomBrut
}

// Libelle rend le nom a afficher : celui de l'asset s'il existe, sinon le libelle du registre.
func (c carteJouee) Libelle() string {
	if c.NomAsset != "" {
		return c.NomAsset
	}
	if c.NomBrut != "" {
		return c.NomBrut
	}
	return c.MapID
}

// Candidats rend les noms a essayer dans l'index des fonds, DANS L'ORDRE DE LA PRODUCTION :
// nom d'asset resolu d'abord, libelle brut du registre ensuite (cf. MapKeysForMatch).
func (c carteJouee) Candidats() []string {
	var out []string
	ajoute := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" {
			return
		}
		for _, deja := range out {
			if strings.EqualFold(deja, s) {
				return
			}
		}
		out = append(out, s)
	}
	ajoute(c.NomAsset)
	ajoute(c.NomBrut)
	return out
}

// PlaylistsTriees rend les playlists de la carte, la plus jouee d'abord.
func (c carteJouee) PlaylistsTriees() []string {
	noms := make([]string, 0, len(c.Playlists))
	for nom := range c.Playlists {
		noms = append(noms, nom)
	}
	sort.Slice(noms, func(i, j int) bool {
		if c.Playlists[noms[i]] != c.Playlists[noms[j]] {
			return c.Playlists[noms[i]] > c.Playlists[noms[j]]
		}
		return noms[i] < noms[j]
	})
	return noms
}

// requeteCartes : une ligne par (carte, playlist). Le tri temporel passe par le fragment
// canonique `COALESCE(start_time_utc, start_time AT TIME ZONE 'UTC')` (CLAUDE.md, regle 8) —
// `start_time` brut n'est pas comparable d'une ligne a l'autre.
const requeteCartes = `
	SELECT
		COALESCE(map_id, '')                                          AS map_id,
		COALESCE(map_name, '')                                        AS map_name,
		COALESCE(NULLIF(TRIM(playlist_name), ''), '(sans playlist)')  AS playlist,
		COUNT(*)                                                      AS matchs,
		SUM(CASE WHEN is_firefight THEN 1 ELSE 0 END)                 AS firefight,
		MAX(COALESCE(player_count, 0))                                AS joueurs_max,
		MAX(COALESCE(start_time_utc, start_time AT TIME ZONE 'UTC'))  AS dernier
	FROM match_registry
	GROUP BY 1, 2, 3
`

// litCartes agrege le registre en une carte par identite jouee.
func litCartes(ctx context.Context, cheminShared string) (map[string]*carteJouee, error) {
	db, ferme, err := duckdb.OpenReadForQuery(cheminShared)
	if err != nil {
		return nil, fmt.Errorf("ouverture du registre partage (%s) : %w", cheminShared, err)
	}
	defer ferme()

	rows, err := db.QueryContext(ctx, requeteCartes)
	if err != nil {
		return nil, fmt.Errorf("lecture de match_registry : %w", err)
	}
	defer rows.Close()

	cartes := map[string]*carteJouee{}
	for rows.Next() {
		var (
			mapID, mapName, playlist string
			matchs, firefight, jmax  int
			dernier                  sql.NullTime
		)
		if err := rows.Scan(&mapID, &mapName, &playlist, &matchs, &firefight, &jmax, &dernier); err != nil {
			return nil, fmt.Errorf("lecture d'une ligne de match_registry : %w", err)
		}
		c := carteJouee{MapID: strings.TrimSpace(mapID), NomBrut: strings.TrimSpace(mapName)}
		cle := c.Cle()
		agg, vu := cartes[cle]
		if !vu {
			agg = &carteJouee{MapID: c.MapID, NomBrut: c.NomBrut, Playlists: map[string]int{}}
			cartes[cle] = agg
		}
		agg.Matchs += matchs
		agg.Firefight += firefight
		agg.Playlists[playlist] += matchs
		if jmax > agg.JoueursMax {
			agg.JoueursMax = jmax
		}
		if dernier.Valid && dernier.Time.After(agg.Dernier) {
			agg.Dernier = dernier.Time
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("parcours de match_registry : %w", err)
	}
	return cartes, nil
}

// requeteNomsAssets : toutes les traductions de carte en UNE requete. La cascade de langues est
// ensuite appliquee en memoire, exactement comme `MetadataRepo.ResolveAssetName` — mais sans une
// requete par carte, parce que l'inventaire les demande toutes.
const requeteNomsAssets = `
	SELECT asset_id, lang, name
	FROM asset_translations
	WHERE asset_type = 'map' AND name IS NOT NULL AND TRIM(name) != ''
`

// litNomsAssets rend, par map_id, le nom d'asset selon la preference EN de la production.
//
// Une metadata illisible n'est PAS fatale : l'inventaire retombe alors sur le seul libelle du
// registre, ce que la production fait aussi (`MapKeysForMatch` journalise et continue). Le
// nombre de noms resolus est rendu pour que l'appelant le dise a l'ecran.
func litNomsAssets(ctx context.Context, cheminMetadata string) (map[string]string, error) {
	db, ferme, err := duckdb.OpenReadForQuery(cheminMetadata)
	if err != nil {
		return nil, fmt.Errorf("ouverture de metadata (%s) : %w", cheminMetadata, err)
	}
	defer ferme()

	rows, err := db.QueryContext(ctx, requeteNomsAssets)
	if err != nil {
		return nil, fmt.Errorf("lecture d'asset_translations : %w", err)
	}
	defer rows.Close()

	parAsset := map[string]map[string]string{}
	for rows.Next() {
		var assetID, lang, nom string
		if err := rows.Scan(&assetID, &lang, &nom); err != nil {
			return nil, fmt.Errorf("lecture d'une traduction d'asset : %w", err)
		}
		if parAsset[assetID] == nil {
			parAsset[assetID] = map[string]string{}
		}
		parAsset[assetID][lang] = nom
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("parcours d'asset_translations : %w", err)
	}
	return choisitParLangue(parAsset), nil
}

// choisitParLangue applique la cascade `PreferredLangsForLocale("en")` — la MEME que celle dont
// la production se sert pour nommer la carte d'un match — puis, a defaut, la premiere langue par
// ordre alphabetique (choix deterministe, comme ResolveAssetName).
func choisitParLangue(parAsset map[string]map[string]string) map[string]string {
	prefs := duckdb.PreferredLangsForLocale("en")
	out := make(map[string]string, len(parAsset))
	for assetID, traductions := range parAsset {
		choisi := ""
		for _, pref := range prefs {
			if nom, ok := traductions[pref]; ok {
				choisi = nom
				break
			}
		}
		if choisi == "" {
			langues := make([]string, 0, len(traductions))
			for lang := range traductions {
				langues = append(langues, lang)
			}
			sort.Strings(langues)
			if len(langues) > 0 {
				choisi = traductions[langues[0]]
			}
		}
		if choisi != "" {
			out[assetID] = choisi
		}
	}
	return out
}
