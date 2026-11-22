package schedtask

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf16"
)

// Les drapeaux de la coquille que la tâche lance (`LevelUp.exe --refresh-only ...`) : la coquille les lit
// (module refresh_only de apps/desktop) et relaie au serveur voisin.
const (
	flagRefreshOnly = "--refresh-only"
	flagDataDir     = "--data-dir"
	flagResourceDir = "--resources-dir"
	flagLogsDir     = "--logs-dir"
)

// validatePath refuse un chemin que la ligne de commande de la tâche ne sait pas porter. La coquille la découpe
// par la règle de CommandLineToArgvW : un chemin vide ou relatif ne désigne rien de stable, un guillemet ou un
// retour à la ligne rompt l'argument, et une barre oblique inverse finale échapperait le guillemet fermant. Les
// autres caractères (% & ^ ! < > |) passent tels quels : aucun interpréteur de commandes ne lit cette ligne.
func validatePath(label, p string) error {
	switch {
	case strings.TrimSpace(p) == "":
		return fmt.Errorf("%w : %s est vide", ErrUnsupportedPath, label)
	case !filepath.IsAbs(p):
		return fmt.Errorf("%w : %s n'est pas absolu", ErrUnsupportedPath, label)
	case strings.ContainsAny(p, "\"\r\n"):
		return fmt.Errorf("%w : %s porte un guillemet ou un retour à la ligne", ErrUnsupportedPath, label)
	case strings.HasSuffix(p, `\`):
		return fmt.Errorf("%w : %s se termine par une barre oblique inverse", ErrUnsupportedPath, label)
	}
	return nil
}

// commandArguments rend les arguments de la coquille : le mode, puis les trois racines (LEVELUP_DATA_DIR,
// LEVELUP_RESOURCES_DIR, LEVELUP_LOGS_DIR du serveur) entre guillemets. Les chemins ont passé [validatePath].
func commandArguments(spec Spec) string {
	return fmt.Sprintf(`%s %s "%s" %s "%s" %s "%s"`,
		flagRefreshOnly, flagDataDir, spec.DataRoot, flagResourceDir, spec.ResourcesRoot, flagLogsDir, spec.LogsRoot)
}

// taskXML rend la définition de la tâche (schéma Task Scheduler 1.2), encodée en UTF-16 petit-boutiste avec
// son BOM, la seule forme que `schtasks /Create /XML` accepte partout. now donne le premier déclenchement.
//
// L'action lance la coquille (LevelUp.exe, sous-système graphique) et non le serveur : un programme console lancé
// par le planificateur ouvre une fenêtre visible au premier plan, la coquille n'en ouvre aucune et lance le serveur
// sans fenêtre.
//
// Ce que la définition pose, et pourquoi : une répétition toutes les 6 heures ; un compte interactif sans
// élévation (LeastPrivilege) ; `StartWhenAvailable` (une exécution manquée parce que le poste dormait part au
// réveil) ; aucune condition de batterie (un portable sur batterie rafraîchit aussi) ; `IgnoreNew` (jamais deux
// exécutions) ; `Hidden` ; une limite de 15 minutes (le mode refresh-only se borne déjà à 10).
func taskXML(spec Spec, now time.Time) ([]byte, error) {
	if spec.ShellExe == "" {
		return nil, ErrShellUnknown
	}
	for _, check := range []struct{ label, path string }{
		{"la coquille", spec.ShellExe}, {"la racine des données", spec.DataRoot},
		{"la racine des ressources", spec.ResourcesRoot}, {"la racine des journaux", spec.LogsRoot},
	} {
		if err := validatePath(check.label, check.path); err != nil {
			return nil, err
		}
	}
	var command, args, workdir bytes.Buffer
	for _, field := range []struct {
		out   *bytes.Buffer
		label string
		text  string
	}{
		{&command, "commande", spec.ShellExe},
		{&args, "arguments", commandArguments(spec)},
		{&workdir, "dossier de travail", filepath.Dir(spec.ShellExe)},
	} {
		if err := xml.EscapeText(field.out, []byte(field.text)); err != nil {
			return nil, fmt.Errorf("schedtask: %s : %w", field.label, err)
		}
	}
	doc := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-16"?>
<Task version="1.2" xmlns="http://schemas.microsoft.com/windows/2004/02/mit/task">
  <RegistrationInfo>
    <Description>LevelUp: keeps the sign-in tokens fresh while the application is closed (refresh-only mode).</Description>
  </RegistrationInfo>
  <Triggers>
    <TimeTrigger>
      <Repetition>
        <Interval>PT%dH</Interval>
        <StopAtDurationEnd>false</StopAtDurationEnd>
      </Repetition>
      <StartBoundary>%s</StartBoundary>
      <Enabled>true</Enabled>
    </TimeTrigger>
  </Triggers>
  <Principals>
    <Principal id="Author">
      <LogonType>InteractiveToken</LogonType>
      <RunLevel>LeastPrivilege</RunLevel>
    </Principal>
  </Principals>
  <Settings>
    <MultipleInstancesPolicy>IgnoreNew</MultipleInstancesPolicy>
    <DisallowStartIfOnBatteries>false</DisallowStartIfOnBatteries>
    <StopIfGoingOnBatteries>false</StopIfGoingOnBatteries>
    <AllowHardTerminate>true</AllowHardTerminate>
    <StartWhenAvailable>true</StartWhenAvailable>
    <RunOnlyIfNetworkAvailable>false</RunOnlyIfNetworkAvailable>
    <AllowStartOnDemand>true</AllowStartOnDemand>
    <Enabled>true</Enabled>
    <Hidden>true</Hidden>
    <ExecutionTimeLimit>PT15M</ExecutionTimeLimit>
  </Settings>
  <Actions Context="Author">
    <Exec>
      <Command>%s</Command>
      <Arguments>%s</Arguments>
      <WorkingDirectory>%s</WorkingDirectory>
    </Exec>
  </Actions>
</Task>
`, int(Interval/time.Hour), now.Format("2006-01-02T15:04:05"), command.String(), args.String(), workdir.String())
	return encodeUTF16LE(doc), nil
}

// encodeUTF16LE encode s en UTF-16 petit-boutiste, BOM en tête.
func encodeUTF16LE(s string) []byte {
	units := utf16.Encode([]rune(s))
	out := make([]byte, 0, 2+2*len(units))
	out = append(out, 0xFF, 0xFE)
	for _, u := range units {
		out = append(out, byte(u), byte(u>>8))
	}
	return out
}
