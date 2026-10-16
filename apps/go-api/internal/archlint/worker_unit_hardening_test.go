// Package archlint — worker_unit_hardening_test.go : garde-rail de l'unité systemd de l'ouvrier
// de rejeu et de son script de déploiement (registre d'audit sécurité 2026-10-15, constat
// C9-4, lot L10, item L10-4).
//
// DEUX CLASSES DE DÉFAUT :
//
//  1. Une unité sans durcissement : packaging/systemd/levelup-worker.service déclare
//     NoNewPrivileges=true, ProtectSystem (full ou strict), ProtectHome (read-only ou true)
//     et des ReadWritePaths qui couvrent le dossier de travail passé à `--work`.
//  2. Une unité installée par lien vers le clone du dépôt, modifiable par le compte
//     `deploy` : scripts/deploy-worker.sh refuse un lien (`[[ -L "$UNIT_DST" ]]` suivi d'un
//     `exit 1`) et n'installe jamais l'unité lui-même (ni `systemctl link`, ni `ln -s`).
//
// PARSING : textuel, lignes `Clé=valeur` de la section [Service]. L'analyse est pure et testée
// sur des textes synthétiques (TestWorkerHardeningAnalysisDetectsDefects).
package archlint

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var workerWorkFlagRe = regexp.MustCompile(`--work\s+(\S+)`)

// TestWorkerUnitIsHardened — l'unité de l'ouvrier est durcie (classe 1).
func TestWorkerUnitIsHardened(t *testing.T) {
	root := repoRootForWorkflows(t)
	unit := mustReadRepoFile(t, filepath.Join(root, "packaging", "systemd", "levelup-worker.service"))
	for _, v := range workerUnitViolations(unit) {
		t.Errorf("packaging/systemd/levelup-worker.service : %s", v)
	}
}

// TestWorkerDeployScriptRefusesLinkedUnit — le script de déploiement refuse un lien et
// n'installe pas l'unité (classe 2).
func TestWorkerDeployScriptRefusesLinkedUnit(t *testing.T) {
	root := repoRootForWorkflows(t)
	script := mustReadRepoFile(t, filepath.Join(root, "scripts", "deploy-worker.sh"))
	for _, v := range workerScriptViolations(script) {
		t.Errorf("scripts/deploy-worker.sh : %s", v)
	}
}

// TestWorkerHardeningAnalysisDetectsDefects — l'analyse sait échouer.
func TestWorkerHardeningAnalysisDetectsDefects(t *testing.T) {
	healthyUnit := `[Service]
ExecStart=/opt/levelup/bin/replay-worker --repo /opt/levelup --work /var/lib/levelup-worker/work
StateDirectory=levelup-worker
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=full
ProtectHome=read-only
ReadWritePaths=/var/lib/levelup-worker -/opt/levelup/data/cache
`
	if got := workerUnitViolations(healthyUnit); len(got) != 0 {
		t.Fatalf("unité saine signalée : %v", got)
	}
	unitDefects := map[string]string{
		"NoNewPrivileges absent":       strings.Replace(healthyUnit, "NoNewPrivileges=true\n", "", 1),
		"NoNewPrivileges à false":      strings.Replace(healthyUnit, "NoNewPrivileges=true", "NoNewPrivileges=false", 1),
		"ProtectSystem absent":         strings.Replace(healthyUnit, "ProtectSystem=full\n", "", 1),
		"ProtectSystem=no":             strings.Replace(healthyUnit, "ProtectSystem=full", "ProtectSystem=no", 1),
		"ProtectHome absent":           strings.Replace(healthyUnit, "ProtectHome=read-only\n", "", 1),
		"ProtectHome=no":               strings.Replace(healthyUnit, "ProtectHome=read-only", "ProtectHome=no", 1),
		"ReadWritePaths absent":        strings.Replace(healthyUnit, "ReadWritePaths=/var/lib/levelup-worker -/opt/levelup/data/cache\n", "", 1),
		"--work hors des chemins":      strings.Replace(healthyUnit, "--work /var/lib/levelup-worker/work", "--work /srv/travail", 1),
		"section [Service] absente":    "ExecStart=/x\n",
		"directive en commentaire":     strings.Replace(healthyUnit, "ProtectSystem=full", "# ProtectSystem=full", 1),
		"racine inscriptible":          strings.Replace(healthyUnit, "ReadWritePaths=/var/lib/levelup-worker", "ReadWritePaths=/ /var/lib/levelup-worker", 1),
		"homes inscriptibles déclarés": strings.Replace(healthyUnit, "ReadWritePaths=/var/lib/levelup-worker", "ReadWritePaths=/home/deploy /var/lib/levelup-worker", 1),
	}
	for name, doc := range unitDefects {
		if doc == healthyUnit {
			t.Fatalf("%s : la mutation n'a rien changé (harnais)", name)
		}
		if got := workerUnitViolations(doc); len(got) == 0 {
			t.Errorf("unité, %s : défaut non signalé", name)
		}
	}

	healthyScript := `UNIT_DST=/etc/systemd/system/x.service
if [[ -L "$UNIT_DST" ]]; then
    echo "ERREUR: lien"
    exit 1
elif [[ ! -e "$UNIT_DST" ]]; then
    echo "# sudo systemctl link n'est pas utilise"
fi
`
	if got := workerScriptViolations(healthyScript); len(got) != 0 {
		t.Fatalf("script sain signalé : %v", got)
	}
	scriptDefects := map[string]string{
		"lien toléré":              strings.Replace(healthyScript, "    exit 1\n", "    echo toléré\n", 1),
		"test de lien absent":      "if [[ ! -e \"$UNIT_DST\" ]]; then\n    exit 1\nfi\n",
		"installation par lien":    healthyScript + "sudo systemctl link /opt/levelup/x.service\n",
		"installation par ln -s":   healthyScript + "ln -sf /opt/levelup/x.service /etc/systemd/system/x.service\n",
		"copie par le script (cp)": healthyScript + "sudo cp \"$UNIT_SRC\" \"$UNIT_DST\"\n",
	}
	for name, doc := range scriptDefects {
		if got := workerScriptViolations(doc); len(got) == 0 {
			t.Errorf("script, %s : défaut non signalé", name)
		}
	}
}

// ─── Analyse ─────────────────────────────────────────────────────────────────────

// workerUnitViolations rend les manquements de la classe 1.
func workerUnitViolations(unit string) []string {
	svc, ok := unitSectionDirectives(unit, "Service")
	if !ok {
		return []string{"section [Service] introuvable (le garde-rail a perdu sa cible)"}
	}
	var out []string
	if last(svc["NoNewPrivileges"]) != "true" {
		out = append(out, "NoNewPrivileges=true absent")
	}
	if v := last(svc["ProtectSystem"]); v != "full" && v != "strict" {
		out = append(out, fmt.Sprintf("ProtectSystem doit valoir full ou strict (lu : %q)", v))
	}
	if v := last(svc["ProtectHome"]); v != "read-only" && v != "true" {
		out = append(out, fmt.Sprintf("ProtectHome doit valoir read-only ou true (lu : %q)", v))
	}
	rw := strings.Fields(strings.Join(svc["ReadWritePaths"], " "))
	if len(rw) == 0 {
		return append(out, "ReadWritePaths absent : les chemins d'écriture de l'ouvrier ne sont pas déclarés")
	}
	for _, p := range rw {
		p = strings.TrimLeft(p, "-+")
		if p == "/" || p == "/home" || strings.HasPrefix(p, "/home/") || p == "/etc" || p == "/usr" {
			out = append(out, fmt.Sprintf("ReadWritePaths %q rouvre ce que ProtectSystem/ProtectHome ferment", p))
		}
	}
	if m := workerWorkFlagRe.FindStringSubmatch(strings.Join(svc["ExecStart"], " ")); m == nil {
		out = append(out, "ExecStart sans --work : le dossier de travail n'est pas explicite")
	} else if !pathCoveredBy(m[1], rw) {
		out = append(out, fmt.Sprintf("--work %s n'est couvert par aucun ReadWritePaths", m[1]))
	}
	return out
}

// workerScriptViolations rend les manquements de la classe 2.
func workerScriptViolations(script string) []string {
	var out []string
	code := scriptCodeLines(script)
	joined := strings.Join(code, "\n")
	refuses := false
	for i, l := range code {
		if strings.Contains(l, `-L "$UNIT_DST"`) {
			for _, next := range code[i+1 : min(i+8, len(code))] {
				if strings.TrimSpace(next) == "exit 1" {
					refuses = true
					break
				}
			}
		}
	}
	if !refuses {
		out = append(out, `aucun refus de lien (`+"`[[ -L \"$UNIT_DST\" ]]`"+` suivi d'un exit 1)`)
	}
	for _, banned := range []string{"systemctl link", "ln -s", "ln -sf", "cp "} {
		if strings.Contains(joined, banned) {
			out = append(out, fmt.Sprintf("le script exécute %q : l'unité s'installe par un geste de root, pas par ce script", banned))
		}
	}
	return out
}

// unitSectionDirectives lit les directives `Clé=valeur` d'une section d'unité (commentaires
// `#` et `;` ignorés), plusieurs occurrences d'une clé conservées dans l'ordre.
func unitSectionDirectives(unit, section string) (map[string][]string, bool) {
	out := map[string][]string{}
	in, found := false, false
	for _, raw := range strings.Split(strings.ReplaceAll(unit, "\r\n", "\n"), "\n") {
		l := strings.TrimSpace(raw)
		if l == "" || strings.HasPrefix(l, "#") || strings.HasPrefix(l, ";") {
			continue
		}
		if strings.HasPrefix(l, "[") && strings.HasSuffix(l, "]") {
			in = l == "["+section+"]"
			found = found || in
			continue
		}
		if in {
			if k, v, ok := strings.Cut(l, "="); ok {
				out[strings.TrimSpace(k)] = append(out[strings.TrimSpace(k)], strings.TrimSpace(v))
			}
		}
	}
	return out, found
}

// scriptCodeLines — lignes d'un script shell hors commentaires purs et lignes `echo`/`#`
// (le texte des messages cite des commandes pour l'opérateur sans les exécuter).
func scriptCodeLines(script string) []string {
	var out []string
	for _, raw := range strings.Split(strings.ReplaceAll(script, "\r\n", "\n"), "\n") {
		l := strings.TrimSpace(raw)
		if l == "" || strings.HasPrefix(l, "#") || strings.HasPrefix(l, "echo ") || strings.HasPrefix(l, `INSTALL_CMD=`) {
			continue
		}
		out = append(out, raw)
	}
	return out
}

func pathCoveredBy(p string, allowed []string) bool {
	for _, a := range allowed {
		a = strings.TrimLeft(a, "-+")
		if p == a || strings.HasPrefix(p, strings.TrimSuffix(a, "/")+"/") {
			return true
		}
	}
	return false
}

func last(vs []string) string {
	if len(vs) == 0 {
		return ""
	}
	return vs[len(vs)-1]
}
