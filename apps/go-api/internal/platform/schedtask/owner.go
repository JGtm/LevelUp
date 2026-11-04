package schedtask

import (
	"context"
	"errors"
	"fmt"
	"html"
	"strings"
	"unicode/utf16"

	"levelup/go-api/internal/platform/startupentry"
)

// RemoveOutcome : ce que [Manager.RemoveIfTargets] a fait.
type RemoveOutcome string

// Les issues possibles, parallèles à celles de l'entrée de démarrage (startupentry.Outcome).
const (
	// RemoveAbsent : la tâche n'existe pas, rien à retirer.
	RemoveAbsent RemoveOutcome = "absent"
	// RemoveRemoved : la tâche a été supprimée.
	RemoveRemoved RemoveOutcome = "removed"
	// RemoveForeign : la tâche existe mais lance le serveur d'un AUTRE dossier (deux installations du même
	// utilisateur partagent la racine de données par défaut, donc le nom de la tâche) ; laissée.
	RemoveForeign RemoveOutcome = "foreign"
)

// ErrActionUnreadable : la définition de la tâche ne porte pas la commande que [Manager.Ensure] écrit ; rien n'est
// supprimé à l'aveugle.
var ErrActionUnreadable = errors.New("schedtask: action de la tâche illisible")

// RemoveIfTargets supprime la tâche seulement si son action lance le serveur situé sous installDir : même règle que
// l'entrée de démarrage (startupentry.CommandTargets). Désinstaller l'installation A ne retire pas la tâche que
// l'installation B a posée sur la même racine de données. Tâche absente = [RemoveAbsent] sans autre appel que la liste.
func (m *Manager) RemoveIfTargets(ctx context.Context, installDir string) (RemoveOutcome, error) {
	state, err := m.Query(ctx)
	if err != nil {
		return "", err
	}
	if state == StateAbsent {
		return RemoveAbsent, nil
	}
	out, err := m.Runner.Run(ctx, "/Query", "/TN", m.name(), "/XML")
	if err != nil {
		return "", fmt.Errorf("schtasks /Query /XML : %w", err)
	}
	command, err := serverCommand(out)
	if err != nil {
		return "", err
	}
	if !startupentry.CommandTargets(command, installDir) {
		return RemoveForeign, nil
	}
	if _, err := m.Runner.Run(ctx, "/Delete", "/TN", m.name(), "/F"); err != nil {
		return "", fmt.Errorf("schtasks /Delete : %w", err)
	}
	return RemoveRemoved, nil
}

// refreshMode : la fin de la commande que la tâche lance (commandArguments).
const refreshMode = "-mode refresh-only"

// serverCommand rend `"<serveur>" -mode refresh-only`, la commande que la définition XML de la tâche (sortie de
// `schtasks /Query /XML`) lance après les trois `set`. La sortie est en UTF-16 avec BOM ou en page de code, et les
// guillemets des arguments sont des entités XML.
func serverCommand(raw []byte) (string, error) {
	text := html.UnescapeString(decodeSchtasksOutput(raw))
	_, rest, ok := strings.Cut(text, "<Arguments>")
	if !ok {
		return "", fmt.Errorf("%w : pas d'élément Arguments", ErrActionUnreadable)
	}
	args, _, _ := strings.Cut(rest, "</Arguments>")
	end := strings.LastIndex(args, refreshMode)
	if end < 0 {
		return "", fmt.Errorf("%w : pas de %q", ErrActionUnreadable, refreshMode)
	}
	start := strings.LastIndex(args[:end], `&& "`)
	if start < 0 {
		return "", fmt.Errorf("%w : serveur introuvable avant %q", ErrActionUnreadable, refreshMode)
	}
	return args[start+len("&& ") : end+len(refreshMode)], nil
}

// decodeSchtasksOutput rend le texte de la sortie : UTF-16 petit-boutiste (BOM, ou octets nuls) décodé, sinon tel quel.
func decodeSchtasksOutput(raw []byte) string {
	bom := len(raw) >= 2 && raw[0] == 0xFF && raw[1] == 0xFE
	nul := len(raw) >= 2 && (raw[1] == 0 || raw[0] == 0)
	if !bom && !nul {
		return string(raw)
	}
	if bom {
		raw = raw[2:]
	}
	units := make([]uint16, 0, len(raw)/2)
	for i := 0; i+1 < len(raw); i += 2 {
		units = append(units, uint16(raw[i])|uint16(raw[i+1])<<8)
	}
	return string(utf16.Decode(units))
}
