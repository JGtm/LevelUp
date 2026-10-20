package title

// player_db_path.go — le gabarit du chemin d'une base joueur, et sa reconnaissance.
//
// PlayerDBPath (et DemoLayout.PlayerDBPath) produisent `<racine>/players/<dossier>/stats.duckdb`.
// IsPlayerDBPath reconnaît cette forme à partir des mêmes constantes : une couche qui ne reçoit
// qu'un chemin nu (l'ouvreur DuckDB, platform/duckdb) sait ainsi qu'elle tient une base joueur
// sans redéfinir le gabarit.

import "path/filepath"

const (
	// playersDirName — répertoire qui regroupe les dossiers des joueurs d'un titre.
	playersDirName = "players"
	// playerDBFileName — fichier DuckDB des enrichissements d'un joueur.
	playerDBFileName = "stats.duckdb"
)

// IsPlayerDBPath dit si path a la forme d'une base joueur : un fichier `stats.duckdb` posé
// dans un dossier de joueur, lui-même enfant d'un répertoire `players`, quels que soient la
// racine et le titre. Séparateurs : ceux de l'OS (filepath). Ne touche pas au disque.
func IsPlayerDBPath(path string) bool {
	clean := filepath.Clean(path)
	if filepath.Base(clean) != playerDBFileName {
		return false
	}
	return filepath.Base(filepath.Dir(filepath.Dir(clean))) == playersDirName
}
