// Package transfer porte les données d'un joueur d'une instance hébergée vers l'application de bureau (lot L22.b,
// ADR 0045).
//
// UNE ARCHIVE (zip, entrées non compressées : le Parquet l'est déjà en zstd) :
//
//	manifest.json                    titre, xuid, gamertag, version, tables, lignes, sommes SHA-256
//	shared/<table>.parquet           les matchs du joueur, COMPLETS, dans les tables de match du titre
//	player/<table>.parquet           les tables de sa base joueur
//
// LE PÉRIMÈTRE EST UNE LISTE FERMÉE ([sharedTables], [playerTables]) : ni jeton, ni compte, ni session, ni
// table de référentiel (réimportable telle quelle depuis les sources du jeu), ni donnée PROPRE d'un autre
// profil suivi (ses enrichissements, sa base joueur).
//
// UN MATCH EST EXPORTÉ ENTIER : pour chaque match où le xuid du joueur figure parmi les participants, toutes
// les lignes de toutes les tables de match (tous les participants, tous les événements, toutes les passes de
// lecture). Un match importé à moitié resterait incomplet pour toujours : la synchronisation le range parmi
// les connus (knownset) et ne le redemande plus. Les tables propres à un film (vies, usages, positions de
// joueurs...) ne voyagent pas : elles se recalculent depuis le film, que l'application de bureau charge
// elle-même. weapon_kills n'y est pas : le schéma de Halo Infinite ne la porte plus (armes par match :
// weapon_accuracy) et l'export de Halo 5 n'existe pas.
//
// Côté bureau, l'archive est vérifiée (manifeste, sommes, joueur attendu) puis écrite par
// persist.ImportPersister : INSERT seulement, idempotent, une transaction par base ; seules les lignes des
// matchs du joueur (par les participants de l'archive) et les lignes de son xuid entrent.
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

// tableSpec : une table du périmètre. Key : la clé d'unicité de l'import (une ligne dont la clé existe déjà
// localement n'est pas réécrite) ; une table sans clé naturelle est clée par match_id seul : un match déjà
// présent dans la table n'y est jamais complété ligne à ligne, il l'est ENTIER ou pas du tout.
type tableSpec struct {
	Name string
	Key  []string
}

// Colonnes communes aux tables du périmètre.
const (
	colMatchID = "match_id"
	colXuid    = "xuid"
)

// matchScopeFilter : la clause WHERE de l'export d'une table de match (%s = le xuid, entre apostrophes) :
// les lignes des matchs où le joueur figure parmi les participants.
const matchScopeFilter = "match_id IN (SELECT match_id FROM match_participants WHERE CAST(xuid AS VARCHAR) = %s)"

// sharedTables : les tables de match du joueur, écrites ensemble par la synchronisation (persist.SharedPersister,
// résumés du journal des morts compris). Une table que le titre ne porte pas est ignorée à l'export.
var sharedTables = []tableSpec{
	{Name: "match_registry", Key: []string{colMatchID}},
	{Name: "match_participants", Key: []string{colMatchID, colXuid}},
	{Name: "medals_earned", Key: []string{colMatchID, colXuid, "medal_name_id"}},
	{Name: "weapon_accuracy", Key: []string{colMatchID, colXuid, "weapon_id"}},
	{Name: "highlight_events", Key: []string{colMatchID}},
	{Name: "killer_victim_pairs", Key: []string{colMatchID}},
	{Name: "match_kill_events", Key: []string{colMatchID}},
	{Name: "match_kill_heads", Key: []string{colMatchID}},
	{Name: "match_kill_pairs", Key: []string{colMatchID}},
	{Name: "match_assist_pairs", Key: []string{colMatchID}},
	{Name: "kill_positions", Key: []string{colMatchID}},
	{Name: "match_csrs", Key: []string{colMatchID}},
	{Name: "match_commendations", Key: []string{colMatchID}},
	{Name: "match_objective_stats", Key: []string{colMatchID}},
}

// participantsTable : la table dont les lignes disent quels matchs sont ceux du joueur.
const participantsTable = "match_participants"

// playerTables : les tables de la base joueur (tout ce qu'elles contiennent est le joueur). À l'import, une
// colonne xuid éventuelle ne laisse entrer que le xuid attendu.
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
