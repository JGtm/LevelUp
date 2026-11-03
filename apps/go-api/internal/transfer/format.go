// Package transfer porte les données d'un joueur d'une instance hébergée vers l'application de bureau (lot L22.b,
// ADR 0045).
//
// UNE ARCHIVE (zip, entrées non compressées : le Parquet l'est déjà en zstd) :
//
//	manifest.json                    titre, xuid, gamertag, version, tables, lignes, sommes SHA-256
//	shared/<table>.parquet           les lignes du joueur dans les tables partagées du titre
//	player/<table>.parquet           les tables de sa base joueur
//
// LE PÉRIMÈTRE EST UNE LISTE FERMÉE ([sharedTables], [playerTables]) : ni jeton, ni compte, ni autre joueur,
// ni table de référentiel (réimportable telle quelle depuis les sources du jeu). Les lignes de participants
// des autres joueurs d'un match ne sont pas exportées : l'application les complète à sa propre synchronisation.
//
// Côté bureau, l'archive est vérifiée (manifeste, sommes, joueur attendu) puis écrite par
// persist.ImportPersister : INSERT seulement, idempotent, une transaction par base.
package transfer

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Noms et bornes du format.
const (
	ManifestFile = "manifest.json"
	// FormatVersion : version du format de l'archive.
	FormatVersion = 1
	// ScopeShared et ScopePlayer : la base d'origine d'une table.
	ScopeShared = "shared"
	ScopePlayer = "player"
)

// Erreurs de l'import ; leur identité fixe le code d'API rendu.
var (
	ErrBadArchive     = errors.New("transfer: archive illisible ou altérée")
	ErrXuidMismatch   = errors.New("transfer: l'archive est celle d'un autre joueur")
	ErrTooLarge       = errors.New("transfer: archive trop volumineuse")
	ErrUnknownTable   = errors.New("transfer: table inconnue de l'application")
	ErrInvalidXuid    = errors.New("transfer: identifiant de joueur invalide")
	ErrFormatTooNewer = errors.New("transfer: format d'archive inconnu")
)

// tableSpec : une table du périmètre. Filter est la clause WHERE de l'export (%s = le xuid, entre
// apostrophes) ; vide = toute la table. Key : la clé d'unicité de l'import.
type tableSpec struct {
	Name   string
	Filter string
	Key    []string
}

// sharedTables : les tables partagées du joueur. Une ligne est celle du joueur par son xuid, ou par un
// match auquel il a participé (le registre).
// Colonnes de clé communes aux tables du périmètre.
const (
	colMatchID = "match_id"
	colXuid    = "xuid"
)

var sharedTables = []tableSpec{
	{Name: "match_registry", Filter: "match_id IN (SELECT match_id FROM match_participants WHERE xuid = %s)", Key: []string{colMatchID}},
	{Name: "match_participants", Filter: "xuid = %s", Key: []string{colMatchID, colXuid}},
	{Name: "medals_earned", Filter: "xuid = %s", Key: []string{colMatchID, colXuid}},
	{Name: "highlight_events", Filter: "xuid = %s", Key: []string{colMatchID, colXuid}},
}

// playerTables : les tables de la base joueur (tout ce qu'elles contiennent est le joueur).
var playerTables = []tableSpec{
	{Name: "player_match_enrichment", Key: []string{colMatchID}},
}

// specFor rend la spécification d'une table du périmètre, ou ErrUnknownTable.
func specFor(scope, name string) (tableSpec, error) {
	list := sharedTables
	if scope == ScopePlayer {
		list = playerTables
	} else if scope != ScopeShared {
		return tableSpec{}, fmt.Errorf("%w : %s/%s", ErrUnknownTable, scope, name)
	}
	for _, s := range list {
		if s.Name == name {
			return s, nil
		}
	}
	return tableSpec{}, fmt.Errorf("%w : %s/%s", ErrUnknownTable, scope, name)
}

// Table : une table de l'archive.
type Table struct {
	Scope  string `json:"scope"`
	Name   string `json:"name"`
	File   string `json:"file"`
	Rows   int64  `json:"rows"`
	Bytes  int64  `json:"bytes"`
	SHA256 string `json:"sha256"`
}

// Manifest décrit une archive d'export.
type Manifest struct {
	Format     int       `json:"format"`
	Title      string    `json:"title"`
	Xuid       string    `json:"xuid"`
	Gamertag   string    `json:"gamertag"`
	CreatedAt  time.Time `json:"created_at"`
	AppVersion string    `json:"app_version"`
	Tables     []Table   `json:"tables"`
}

var xuidPattern = regexp.MustCompile(`^\d{5,20}$`)

// NormalizeXuid rend le xuid en chiffres seuls (« xuid(2533…) » accepté), ou ErrInvalidXuid : il entre dans
// du SQL comme littéral, seuls des chiffres passent.
func NormalizeXuid(raw string) (string, error) {
	x := strings.TrimSpace(raw)
	x = strings.TrimSuffix(strings.TrimPrefix(x, "xuid("), ")")
	if !xuidPattern.MatchString(x) {
		return "", ErrInvalidXuid
	}
	return x, nil
}

// entryName : un nom d'entrée valide de l'archive (jamais un chemin libre : pas de ../, pas de séparateur
// hors du préfixe de portée).
var entryName = regexp.MustCompile(`^(shared|player)/[a-z0-9_]{1,64}\.parquet$`)
