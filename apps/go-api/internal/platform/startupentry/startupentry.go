// Package startupentry retire l'entrée « démarrer avec Windows » de l'utilisateur courant quand le programme est
// désinstallé (mode `-mode uninstall-cleanup` du serveur, crochet NSIS de l'installeur, ADR 0045 D-28).
//
// La coquille pose l'entrée par le plugin d'autostart : une valeur `LevelUp` de
// `HKCU\Software\Microsoft\Windows\CurrentVersion\Run` qui lance l'application avec `--minimized`, et
// Windows note son approbation dans `HKCU\...\Explorer\StartupApproved\Run`. Le désinstalleur NSIS n'a sous la
// main que le serveur livré : le retrait se fait ici, en Go, derrière [Store] (les tests branchent un faux, aucune
// valeur réelle n'est touchée).
//
// Une entrée n'est retirée que si sa commande vise le dossier du programme qu'on désinstalle : l'entrée d'un
// autre dossier (installation de développement, copie portable) n'est jamais supprimée.
package startupentry

import (
	"fmt"
	"strings"
)

// EntryName : le nom de la valeur de démarrage, celui que la coquille pose (autostart.rs, ENTRY_NAME).
const EntryName = "LevelUp"

// Outcome : ce que [RemoveForInstall] a fait.
type Outcome string

// Les issues possibles.
const (
	// OutcomeAbsent : aucune entrée de démarrage, rien à retirer.
	OutcomeAbsent Outcome = "absent"
	// OutcomeRemoved : l'entrée et son approbation ont été retirées.
	OutcomeRemoved Outcome = "removed"
	// OutcomeForeign : une entrée existe mais vise un autre dossier que celui désinstallé ; laissée.
	OutcomeForeign Outcome = "foreign"
)

// Store lit et supprime les valeurs de démarrage de l'utilisateur courant.
type Store interface {
	// RunValue rend la commande de la valeur `Run` nommée name, et false si elle est absente.
	RunValue(name string) (command string, present bool, err error)
	// DeleteRunValue supprime la valeur `Run` ; absente = succès.
	DeleteRunValue(name string) error
	// DeleteApprovedValue supprime la valeur d'approbation de name ; absente = succès.
	DeleteApprovedValue(name string) error
}

// RemoveForInstall retire l'entrée de démarrage `LevelUp` si sa commande vise le dossier installDir. Une entrée
// absente ou étrangère n'écrit rien. L'approbation n'est retirée qu'avec la valeur `Run`.
func RemoveForInstall(s Store, installDir string) (Outcome, error) {
	command, present, err := s.RunValue(EntryName)
	if err != nil {
		return "", fmt.Errorf("startupentry: lecture de la valeur Run : %w", err)
	}
	if !present {
		return OutcomeAbsent, nil
	}
	if !CommandTargets(command, installDir) {
		return OutcomeForeign, nil
	}
	if err := s.DeleteRunValue(EntryName); err != nil {
		return "", fmt.Errorf("startupentry: suppression de la valeur Run : %w", err)
	}
	if err := s.DeleteApprovedValue(EntryName); err != nil {
		return "", fmt.Errorf("startupentry: suppression de l'approbation : %w", err)
	}
	return OutcomeRemoved, nil
}

// CommandTargets dit si command (la ligne de commande d'une valeur `Run`, programme entre guillemets ou non,
// suivi de ses arguments) lance un programme situé sous le dossier installDir. Comparaison à la manière de
// Windows : casse ignorée, `/` et `\` équivalents.
func CommandTargets(command, installDir string) bool {
	dir := normalizePath(installDir)
	if dir == "" {
		return false
	}
	return strings.HasPrefix(normalizePath(strings.TrimLeft(strings.TrimSpace(command), `"`)), dir+`\`)
}

// normalizePath met un chemin sous la forme comparable : minuscules, `\`, sans `\` final.
func normalizePath(p string) string {
	p = strings.ToLower(strings.ReplaceAll(strings.TrimSpace(p), "/", `\`))
	return strings.TrimRight(p, `\`)
}
