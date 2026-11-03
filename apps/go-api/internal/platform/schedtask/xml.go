package schedtask

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf16"

	"levelup/go-api/internal/config"
)

// forbiddenPathChars : ce qu'un chemin ne peut pas porter pour que la ligne de commande de la tâche reste
// exacte sous cmd.exe (guillemet, expansion %, enchaînements, échappement, redirections, retours à la ligne).
const forbiddenPathChars = "\"%&^!<>|\r\n"

// validatePath refuse un chemin vide, relatif ou portant un caractère que cmd.exe interpréterait.
func validatePath(label, p string) error {
	switch {
	case strings.TrimSpace(p) == "":
		return fmt.Errorf("%w : %s est vide", ErrUnsupportedPath, label)
	case !filepath.IsAbs(p):
		return fmt.Errorf("%w : %s n'est pas absolu", ErrUnsupportedPath, label)
	case strings.ContainsAny(p, forbiddenPathChars):
		return fmt.Errorf("%w : %s porte un caractère réservé", ErrUnsupportedPath, label)
	}
	return nil
}

// commandArguments rend les arguments de cmd.exe : les trois racines dans l'environnement, puis le serveur en
// mode refresh-only. `/s` retire le premier et le dernier guillemet de la commande, ce qui laisse les guillemets
// intérieurs intacts.
func commandArguments(spec Spec) string {
	return fmt.Sprintf(`/d /s /c "set "%s=%s" && set "%s=%s" && set "%s=%s" && "%s" -mode refresh-only"`,
		config.EnvDataDir, spec.DataRoot, config.EnvResourcesDir, spec.ResourcesRoot, config.EnvLogsDir, spec.LogsRoot,
		spec.ExePath)
}

// taskXML rend la définition de la tâche (schéma Task Scheduler 1.2), encodée en UTF-16 petit-boutiste avec
// son BOM, la seule forme que `schtasks /Create /XML` accepte partout. now donne le premier déclenchement.
//
// Ce que la définition pose, et pourquoi : une répétition toutes les 6 heures ; un compte interactif sans
// élévation (LeastPrivilege) ; `StartWhenAvailable` (une exécution manquée parce que le poste dormait part au
// réveil) ; aucune condition de batterie (un portable sur batterie rafraîchit aussi) ; `IgnoreNew` (jamais deux
// exécutions) ; `Hidden` ; une limite de 15 minutes (le mode refresh-only se borne déjà à 10).
func taskXML(spec Spec, now time.Time) ([]byte, error) {
	for _, check := range []struct{ label, path string }{
		{"l'exécutable", spec.ExePath}, {"la racine des données", spec.DataRoot},
		{"la racine des ressources", spec.ResourcesRoot}, {"la racine des journaux", spec.LogsRoot},
	} {
		if err := validatePath(check.label, check.path); err != nil {
			return nil, err
		}
	}
	var args, workdir bytes.Buffer
	if err := xml.EscapeText(&args, []byte(commandArguments(spec))); err != nil {
		return nil, fmt.Errorf("schedtask: arguments : %w", err)
	}
	if err := xml.EscapeText(&workdir, []byte(spec.ResourcesRoot)); err != nil {
		return nil, fmt.Errorf("schedtask: dossier de travail : %w", err)
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
      <Command>cmd.exe</Command>
      <Arguments>%s</Arguments>
      <WorkingDirectory>%s</WorkingDirectory>
    </Exec>
  </Actions>
</Task>
`, int(Interval/time.Hour), now.Format("2006-01-02T15:04:05"), args.String(), workdir.String())
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
