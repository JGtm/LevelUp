// Package archlint — compose_hardening_test.go : garde-rail de docker-compose.yml
// (registre d'audit sécurité 2026-10-15, constat C9-4, lot L10 ; couvre aussi L8-9).
//
// DEUX CLASSES DE DÉFAUT, UN TEST CHACUNE :
//
//  1. Un conteneur qui garde plus de privilèges que son entrée n'en demande. Chaque
//     service déclare `security_opt: [no-new-privileges:true]` et `cap_drop: [ALL]` ; son
//     `cap_add` ne contient que des capacités de composeAllowedCaps (celles de l'entrée
//     scripts/docker-entrypoint.sh : chown des volumes puis `gosu`), et il n'est jamais
//     `privileged`. Sans cela, un processus compromis dans le conteneur garde les
//     capacités par défaut de Docker (dont DAC_OVERRIDE, FOWNER, NET_RAW, SETFCAP) et peut
//     en acquérir par un binaire setuid.
//  2. Un service publié derrière nginx sans LEVELUP_TRUST_PROXY_HEADERS : le limiteur de
//     débit et les journaux voient l'IP du proxy (127.0.0.1) pour tous les visiteurs, qui
//     partagent un seul seau (429 en masse). La variable se pose dans `environment` du
//     service ; seule exception, datée, dans composeProxyHeadersFromEnvFile.
//
// PARSING : parseur YAML (gopkg.in/yaml.v3, dépendance directe du module), car la forme de
// `environment` (liste `K=V` ou table) varie. Un garde-rail qui perd sa cible rougit : un
// compose sans service est une violation. L'analyse est pure et testée sur des textes
// synthétiques (TestComposeAnalysisDetectsDefects), indépendamment de l'état du dépôt.
package archlint

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// composeAllowedCaps — capacités qu'un service peut ajouter après `cap_drop: [ALL]`, avec
// la raison tirée de scripts/docker-entrypoint.sh (démarre en root, corrige les droits des
// volumes, puis passe à appuser par gosu). Toute autre capacité est un défaut ; en ajouter
// une exige une ligne datée et justifiée ici.
var composeAllowedCaps = map[string]string{
	// 2026-10-16 : `chown -R appuser:appuser /app/data` et chown des fichiers de config.
	"CHOWN": "chown des volumes liés par l'entrée",
	// 2026-10-16 : `chown -R` doit ouvrir et parcourir des dossiers déjà possédés par
	// appuser en mode 0700/0750 (auth, faits de film) ; root sans capacité n'est plus
	// propriétaire ni dans leur groupe. Plus étroite que DAC_OVERRIDE (lecture seule).
	"DAC_READ_SEARCH": "parcours des dossiers d'appuser par le chown récursif de l'entrée",
	// 2026-10-16 : gosu appelle setgroups/setgid puis setuid avant d'exécuter la commande.
	"SETGID": "gosu : changement de groupe",
	"SETUID": "gosu : changement d'utilisateur",
}

// composeProxyHeadersFromEnvFile — services publiés dont LEVELUP_TRUST_PROXY_HEADERS vient
// de leur `env_file` (hors dépôt, donc non vérifiable ici) au lieu de `environment`.
// 2026-10-16 : `levelup` (production) lit .env.local, où la variable est OBLIGATOIRE
// derrière nginx (commentaire du service dans docker-compose.yml, .env.local.example).
// Une entrée ne vaut que si le service existe encore et porte bien un env_file.
var composeProxyHeadersFromEnvFile = map[string]bool{
	"levelup": true,
}

const (
	composeTrustProxyVar = "LEVELUP_TRUST_PROXY_HEADERS"
	composeNoNewPrivVal  = "no-new-privileges"
)

// composeFile — la part de docker-compose.yml que le garde-rail lit.
type composeFile struct {
	Services map[string]composeService `yaml:"services"`
}

type composeService struct {
	Privileged  bool     `yaml:"privileged"`
	SecurityOpt []string `yaml:"security_opt"`
	CapDrop     []string `yaml:"cap_drop"`
	CapAdd      []string `yaml:"cap_add"`
	Ports       []any    `yaml:"ports"`
	EnvFile     any      `yaml:"env_file"`
	Environment any      `yaml:"environment"`
}

// TestComposeServicesDropPrivileges — chaque service du compose réduit ses privilèges.
func TestComposeServicesDropPrivileges(t *testing.T) {
	src := readComposeFile(t)
	for _, v := range composePrivilegeViolations(src) {
		t.Errorf(`docker-compose.yml : %s

  CORRECTION : sur chaque service,
      security_opt:
        - no-new-privileges:true
      cap_drop:
        - ALL
      cap_add:            # seulement ce que l'entrée exige (composeAllowedCaps)
        - CHOWN ...`, v)
	}
}

// TestComposePublishedServicesTrustProxyHeaders — chaque service publié derrière nginx
// pose LEVELUP_TRUST_PROXY_HEADERS (couvre L8-9 : la démo n'a pas d'env_file).
func TestComposePublishedServicesTrustProxyHeaders(t *testing.T) {
	src := readComposeFile(t)
	for _, v := range composeProxyHeaderViolations(src) {
		t.Errorf(`docker-compose.yml : %s

  CORRECTION : ajouter dans l'environment du service
      - LEVELUP_TRUST_PROXY_HEADERS=true
  (nginx écrase X-Real-IP à chaque requête : packaging/nginx/*.conf, nginx_conf_test.go).`, v)
	}
}

// TestComposeAnalysisDetectsDefects — l'analyse sait échouer : chaque défaut de la classe,
// injecté dans un compose sain, est signalé.
func TestComposeAnalysisDetectsDefects(t *testing.T) {
	healthy := `services:
  web:
    ports: ["127.0.0.1:8000:8000"]
    environment:
      - LEVELUP_TRUST_PROXY_HEADERS=true
    security_opt:
      - no-new-privileges:true
    cap_drop:
      - ALL
    cap_add:
      - CHOWN
      - SETUID
  prod:
    ports: ["127.0.0.1:8001:8000"]
    env_file: [.env.local]
    security_opt: ["no-new-privileges:true"]
    cap_drop: [ALL]
`
	saved := composeProxyHeadersFromEnvFile
	composeProxyHeadersFromEnvFile = map[string]bool{"prod": true}
	defer func() { composeProxyHeadersFromEnvFile = saved }()

	if got := composeAllViolations(healthy); len(got) != 0 {
		t.Fatalf("compose sain signalé : %v", got)
	}
	defects := map[string]string{
		"no-new-privileges absent":   strings.Replace(healthy, "    security_opt:\n      - no-new-privileges:true\n", "", 1),
		"no-new-privileges à false":  strings.Replace(healthy, "no-new-privileges:true\n    cap_drop:\n      - ALL\n    cap_add", "no-new-privileges:false\n    cap_drop:\n      - ALL\n    cap_add", 1),
		"cap_drop absent":            strings.Replace(healthy, "    cap_drop:\n      - ALL\n    cap_add:\n      - CHOWN\n      - SETUID\n", "", 1),
		"cap_drop sans ALL":          strings.Replace(healthy, "    cap_drop:\n      - ALL\n", "    cap_drop:\n      - NET_RAW\n", 1),
		"capacité hors liste":        strings.Replace(healthy, "      - SETUID\n", "      - SETUID\n      - SYS_ADMIN\n", 1),
		"ALL ajouté en retour":       strings.Replace(healthy, "      - SETUID\n", "      - SETUID\n      - ALL\n", 1),
		"privileged":                 strings.Replace(healthy, "  web:\n", "  web:\n    privileged: true\n", 1),
		"proxy absent":               strings.Replace(healthy, "      - LEVELUP_TRUST_PROXY_HEADERS=true\n", "      - OTHER=1\n", 1),
		"proxy à false":              strings.Replace(healthy, "LEVELUP_TRUST_PROXY_HEADERS=true", "LEVELUP_TRUST_PROXY_HEADERS=false", 1),
		"proxy sur service sans env": strings.Replace(healthy, "    env_file: [.env.local]\n", "", 1),
		"aucun service (vacuité)":    "services: {}\n",
		"YAML illisible":             "services: [",
	}
	for name, doc := range defects {
		if doc == healthy {
			t.Fatalf("%s : la mutation n'a rien changé (harnais)", name)
		}
		if got := composeAllViolations(doc); len(got) == 0 {
			t.Errorf("%s : défaut non signalé", name)
		}
	}

	// Table (map) et liste d'environment sont toutes deux lues.
	asMap := strings.Replace(healthy, "    environment:\n      - LEVELUP_TRUST_PROXY_HEADERS=true\n",
		"    environment:\n      LEVELUP_TRUST_PROXY_HEADERS: \"true\"\n", 1)
	if got := composeAllViolations(asMap); len(got) != 0 {
		t.Errorf("environment en table non reconnu : %v", got)
	}

	// Une entrée d'exception périmée (service disparu) est elle-même une violation.
	composeProxyHeadersFromEnvFile = map[string]bool{"prod": true, "fantome": true}
	if got := composeAllViolations(healthy); len(got) == 0 {
		t.Errorf("exception sur un service inexistant non signalée")
	}
}

// ─── Analyse ─────────────────────────────────────────────────────────────────────

func composeAllViolations(src string) []string {
	return append(composePrivilegeViolations(src), composeProxyHeaderViolations(src)...)
}

// composePrivilegeViolations rend les manquements de la classe 1.
func composePrivilegeViolations(src string) []string {
	cf, err := parseCompose(src)
	if err != nil {
		return []string{err.Error()}
	}
	var out []string
	for _, name := range sortedServiceNames(cf) {
		svc := cf.Services[name]
		if svc.Privileged {
			out = append(out, fmt.Sprintf("service %q : privileged: true", name))
		}
		if !hasNoNewPrivileges(svc.SecurityOpt) {
			out = append(out, fmt.Sprintf("service %q : security_opt sans no-new-privileges:true", name))
		}
		if !containsFold(svc.CapDrop, "ALL") {
			out = append(out, fmt.Sprintf("service %q : cap_drop sans ALL", name))
		}
		for _, c := range svc.CapAdd {
			if _, ok := composeAllowedCaps[strings.ToUpper(strings.TrimSpace(c))]; !ok {
				out = append(out, fmt.Sprintf("service %q : cap_add %q hors de composeAllowedCaps", name, c))
			}
		}
	}
	return out
}

// composeProxyHeaderViolations rend les manquements de la classe 2.
func composeProxyHeaderViolations(src string) []string {
	cf, err := parseCompose(src)
	if err != nil {
		return []string{err.Error()}
	}
	var out []string
	for _, name := range sortedServiceNames(cf) {
		svc := cf.Services[name]
		if len(svc.Ports) == 0 {
			continue
		}
		env := composeEnvironment(svc.Environment)
		if isTrue(env[composeTrustProxyVar]) {
			continue
		}
		if composeProxyHeadersFromEnvFile[name] && svc.EnvFile != nil {
			continue
		}
		out = append(out, fmt.Sprintf("service publié %q : %s=true absent de environment", name, composeTrustProxyVar))
	}
	for name := range composeProxyHeadersFromEnvFile {
		if svc, ok := cf.Services[name]; !ok || svc.EnvFile == nil {
			out = append(out, fmt.Sprintf("composeProxyHeadersFromEnvFile : %q n'est plus un service avec env_file (exception périmée)", name))
		}
	}
	return out
}

func parseCompose(src string) (composeFile, error) {
	var cf composeFile
	if err := yaml.Unmarshal([]byte(src), &cf); err != nil {
		return cf, fmt.Errorf("YAML illisible : %w", err)
	}
	if len(cf.Services) == 0 {
		return cf, fmt.Errorf("aucun service lu : le garde-rail a perdu sa cible")
	}
	return cf, nil
}

func sortedServiceNames(cf composeFile) []string {
	names := make([]string, 0, len(cf.Services))
	for n := range cf.Services {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// composeEnvironment normalise `environment` (liste "K=V" ou table) en table.
func composeEnvironment(raw any) map[string]string {
	out := map[string]string{}
	switch v := raw.(type) {
	case []any:
		for _, item := range v {
			s := fmt.Sprint(item)
			k, val, _ := strings.Cut(s, "=")
			out[strings.TrimSpace(k)] = strings.TrimSpace(val)
		}
	case map[string]any:
		for k, val := range v {
			out[k] = fmt.Sprint(val)
		}
	}
	return out
}

func hasNoNewPrivileges(opts []string) bool {
	for _, o := range opts {
		name, val, hasVal := strings.Cut(strings.ReplaceAll(strings.TrimSpace(o), "=", ":"), ":")
		if name == composeNoNewPrivVal && (!hasVal || isTrue(val)) {
			return true
		}
	}
	return false
}

func containsFold(list []string, want string) bool {
	for _, s := range list {
		if strings.EqualFold(strings.TrimSpace(s), want) {
			return true
		}
	}
	return false
}

func isTrue(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "true", "1":
		return true
	}
	return false
}

func readComposeFile(t *testing.T) string {
	t.Helper()
	path := filepath.Join(repoRootForWorkflows(t), "docker-compose.yml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("lecture de %s impossible : %v (le garde-rail a perdu sa cible)", path, err)
	}
	return string(data)
}
