package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Fichiers du mode refresh-only, sous la racine de donnees : le test pose le code de sortie voulu avant le
// lancement (le relais de la coquille n'a pas d'autre canal : il retire LEVELUP_* et FAKE_* ne traverse pas un
// environnement de test partage), le faux serveur ecrit ce qu'il a recu.
const (
	refreshExitFile = "fake-refresh-exit.txt"
	refreshRanFile  = "fake-refresh-ran.txt"
)

// refreshOnly joue `-mode refresh-only`, lance par le relais `LevelUp.exe --refresh-only` de la coquille : il lit
// le code de sortie dans refreshExitFile (defaut 0), ecrit dans refreshRanFile le repertoire courant, les
// racines recues et la presence du secret (qui ne doit pas etre pose), puis rend le code. ok est faux quand
// args n'est pas ce mode.
func refreshOnly(args []string, getenv func(string) string, cwd string, stderr io.Writer) (code int, ok bool) {
	if len(args) != 2 || args[0] != "-mode" || args[1] != "refresh-only" {
		return 0, false
	}
	dataDir := getenv("LEVELUP_DATA_DIR")
	if dataDir == "" {
		fmt.Fprintln(stderr, "fake: refresh-only sans LEVELUP_DATA_DIR")
		return exitSetup, true
	}
	code = exitOK
	if raw, err := os.ReadFile(filepath.Join(dataDir, refreshExitFile)); err == nil {
		n, convErr := strconv.Atoi(strings.TrimSpace(string(raw)))
		if convErr != nil {
			fmt.Fprintf(stderr, "fake: %s : %v\n", refreshExitFile, convErr)
			return exitSetup, true
		}
		code = n
	}
	ran := fmt.Sprintf("cwd=%s resources=%s data=%s logs=%s secret_set=%t\n", cwd,
		getenv("LEVELUP_RESOURCES_DIR"), dataDir, getenv("LEVELUP_LOGS_DIR"), getenv("LEVELUP_DESKTOP_SECRET") != "")
	if err := os.WriteFile(filepath.Join(dataDir, refreshRanFile), []byte(ran), 0o600); err != nil {
		fmt.Fprintf(stderr, "fake: %s : %v\n", refreshRanFile, err)
		return exitSetup, true
	}
	return code, true
}
