// Package schedtask pose et retire la tâche planifiée Windows qui garde les jetons vivants quand l'application
// de bureau est fermée (réglage `scheduled_refresh`, ADR 0045 D-10 et lot L21.e).
//
// La tâche, par utilisateur, sans élévation, lance `levelup-server.exe -mode refresh-only` toutes les 6 heures
// avec les trois racines de données dans son environnement. Le mode refresh-only prend le verrou d'instance :
// application ouverte, il sort en 0 sans rien faire (un refresh token ne se présente qu'à un processus à la
// fois, ADR 0023).
//
// Tout passe par `schtasks` derrière [Runner] (lancé par childproc, jamais une fenêtre de console) : les tests
// simulent l'exécuteur et vérifient les arguments exacts, aucun appel réel hors du test à étiquette
// e2e_windows. La définition de la tâche est un fichier XML (schtasks /Create /XML) et non des options de la
// ligne de commande : `/TR` est borné à 261 caractères, que trois chemins de racines dépassent aisément, et
// seul le XML porte les réglages de la tâche (reprise après une veille, aucune condition de batterie).
package schedtask

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// TaskName : le nom de la tâche, dans le dossier LevelUp du planificateur.
const TaskName = `LevelUp\Refresh`

// Interval : la période de la tâche.
const Interval = 6 * time.Hour

// State : ce que la dernière observation a trouvé.
type State string

// Les états observables.
const (
	// StatePresent : la tâche existe.
	StatePresent State = "present"
	// StateAbsent : la tâche n'existe pas.
	StateAbsent State = "absent"
	// StateError : le planificateur n'a pas répondu ou a refusé.
	StateError State = "error"
)

// ErrUnsupportedPath : un chemin porte un caractère que la ligne de commande de la tâche ne sait pas
// protéger (guillemet, %, &, ^, !, saut de ligne) ; la tâche n'est pas posée plutôt que mal quotée.
var ErrUnsupportedPath = errors.New("schedtask: chemin non pris en charge par la ligne de commande de la tâche")

// Runner exécute schtasks avec args et rend sa sortie ; une sortie non nulle est une erreur.
type Runner interface {
	Run(ctx context.Context, args ...string) ([]byte, error)
}

// Spec : ce que la tâche lance.
type Spec struct {
	// ExePath : le serveur (levelup-server.exe), chemin absolu.
	ExePath string
	// DataRoot, ResourcesRoot, LogsRoot : les trois racines, posées dans l'environnement de la tâche
	// (LEVELUP_DATA_DIR, LEVELUP_RESOURCES_DIR, LEVELUP_LOGS_DIR) : le serveur de la tâche lit les mêmes
	// données que l'application.
	DataRoot      string
	ResourcesRoot string
	LogsRoot      string
}

// Manager pose, observe et retire la tâche.
type Manager struct {
	// Runner : l'exécuteur de schtasks ; obligatoire.
	Runner Runner
	// Spec : ce que la tâche lance.
	Spec Spec
	// Name : le nom de la tâche ; vide = [TaskName].
	Name string
	// Now : l'horloge du premier déclenchement ; nil = time.Now.
	Now func() time.Time
	// TempDir : le dossier du fichier XML éphémère ; vide = os.TempDir().
	TempDir string
}

func (m *Manager) name() string {
	if m.Name != "" {
		return m.Name
	}
	return TaskName
}

func (m *Manager) now() time.Time {
	if m.Now != nil {
		return m.Now()
	}
	return time.Now()
}

// Query dit si la tâche existe : elle liste les tâches (CSV, sans en-tête) et cherche le nom. Le code de
// sortie de `/Query /TN` ne distingue pas « absente » d'un refus et son message dépend de la langue du
// système : la liste n'a aucune de ces deux ambiguïtés.
func (m *Manager) Query(ctx context.Context) (State, error) {
	out, err := m.Runner.Run(ctx, "/Query", "/FO", "CSV", "/NH")
	if err != nil {
		return StateError, fmt.Errorf("schtasks /Query : %w", err)
	}
	if taskListed(out, m.name()) {
		return StatePresent, nil
	}
	return StateAbsent, nil
}

// taskListed dit si la liste CSV de schtasks porte la tâche `name` (première colonne, `\<nom>`).
func taskListed(csv []byte, name string) bool {
	want := `\` + strings.TrimPrefix(name, `\`)
	for _, line := range strings.Split(string(csv), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, `"`) {
			continue
		}
		first, _, _ := strings.Cut(line[1:], `"`)
		if strings.EqualFold(first, want) {
			return true
		}
	}
	return false
}

// Ensure pose la tâche, ou la remplace (`/F`) : deux appels laissent la même tâche. La définition passe par un
// fichier XML éphémère, supprimé ensuite.
func (m *Manager) Ensure(ctx context.Context) error {
	xmlDoc, err := taskXML(m.Spec, m.now())
	if err != nil {
		return err
	}
	dir := m.TempDir
	if dir == "" {
		dir = os.TempDir()
	}
	f, err := os.CreateTemp(dir, "levelup-refresh-*.xml")
	if err != nil {
		return fmt.Errorf("schedtask: fichier de définition : %w", err)
	}
	path := f.Name()
	defer func() {
		if rerr := os.Remove(path); rerr != nil && !errors.Is(rerr, os.ErrNotExist) {
			slog.WarnContext(ctx, "schedtask: fichier de définition non supprimé", "chemin", filepath.Base(path), "err", rerr)
		}
	}()
	if _, err := f.Write(xmlDoc); err != nil {
		_ = f.Close() // l'erreur d'écriture est celle qui compte
		return fmt.Errorf("schedtask: écriture de la définition : %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("schedtask: fermeture de la définition : %w", err)
	}
	if _, err := m.Runner.Run(ctx, "/Create", "/TN", m.name(), "/XML", path, "/F"); err != nil {
		return fmt.Errorf("schtasks /Create : %w", err)
	}
	return nil
}

// Remove supprime la tâche ; une tâche absente est un succès (rien n'est appelé).
func (m *Manager) Remove(ctx context.Context) error {
	state, err := m.Query(ctx)
	if err != nil {
		return err
	}
	if state == StateAbsent {
		return nil
	}
	if _, err := m.Runner.Run(ctx, "/Delete", "/TN", m.name(), "/F"); err != nil {
		return fmt.Errorf("schtasks /Delete : %w", err)
	}
	return nil
}
