// Package archlint — compact_passes_nuit_script_test.go : le script hôte de compactage de nuit
// (scripts/compact-passes-nuit.sh) exécuté HORS VPS, avec de faux `docker`, `curl`, `df`,
// `systemctl` et `flock` placés en tête du PATH. Chaque faux binaire note son appel ; le test
// vérifie ce que le script a fait (ou refusé de faire), dans quel ordre, et son code de sortie.
//
// Cas couverts : jauge sous le seuil (aucun arrêt), au-dessus (arrêt, compaction, redémarrage
// dans cet ordre), compaction en échec (serveur redémarré quand même, code non nul), /health muet
// après redémarrage, disque insuffisant, verrou tenu, sauvegarde restic en cours, jauge absente
// ou périmée, ménage des sauvegardes (motif seul, après un passage réussi seulement).
//
// Comme ci_go_shards_test.go : hors Linux, bash de Git for Windows, sinon le test est sauté (la CI
// Linux le joue).
package archlint

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// fauxBinaires : chaque faux note `nom args` dans $FAKE_CALLS ; le comportement se règle par env.
var fauxBinaires = map[string]string{
	"docker": `echo "docker $*" >>"$FAKE_CALLS"
case "$*" in
  "compose run"*) exit "${FAKE_RUN_RC:-0}" ;;
  "compose stop"*) exit "${FAKE_STOP_RC:-0}" ;;
esac
exit 0`,
	"curl":      `echo "curl $*" >>"$FAKE_CALLS"; exit "${FAKE_HEALTH_RC:-0}"`,
	"df":        `printf 'Filesystem 1024-blocks Used Available Capacity Mounted\nfake 100 0 %s 0%% /\n' "${FAKE_DF_LIBRE_KO:-999999999}"`,
	"systemctl": `echo "systemctl $*" >>"$FAKE_CALLS"; [ "${FAKE_RESTIC_ACTIF:-0}" = 1 ] && exit 0; exit 3`,
	"flock":     `exit "${FAKE_FLOCK_RC:-0}"`,
}

// bancNuit : un LEVELUP_DIR jetable, les faux binaires, et la jauge.
type bancNuit struct {
	t        *testing.T
	dir      string
	fakeBin  string
	env      []string
	calls    string
	etat     string
	sauvDir  string
	logPath  string
	exitCode int
}

func nouveauBancNuit(t *testing.T) *bancNuit {
	t.Helper()
	root := t.TempDir()
	b := &bancNuit{t: t, dir: filepath.Join(root, "levelup"), fakeBin: filepath.Join(root, "bin")}
	b.calls = filepath.Join(root, "calls.txt")
	b.etat = filepath.Join(b.dir, "data", "logs", "compact-passes-nuit.etat")
	b.sauvDir = filepath.Join(b.dir, "data", "compaction-backups")
	b.logPath = filepath.Join(b.dir, "data", "logs", "compact-passes-nuit.log")
	for _, d := range []string{b.fakeBin, filepath.Join(b.dir, "data", "global", "admin_state"),
		filepath.Join(b.dir, "data", "titles", "halo_infinite", "warehouse")} {
		mustMkdir(t, d)
	}
	mustWrite(t, filepath.Join(b.dir, "data", "titles", "halo_infinite", "warehouse", "shared_matches_v2.duckdb"), strings.Repeat("x", 4096))
	for nom, corps := range fauxBinaires {
		mustWrite(t, filepath.Join(b.fakeBin, nom), "#!/usr/bin/env bash\n"+corps+"\n")
		if err := os.Chmod(filepath.Join(b.fakeBin, nom), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	b.env = []string{
		"LEVELUP_DIR=" + filepath.ToSlash(b.dir),
		"LEVELUP_COMPACT_VERROU=" + filepath.ToSlash(filepath.Join(root, "verrou")),
		"LEVELUP_COMPACT_HEALTH_ATTENTE_S=0",
		"FAKE_CALLS=" + filepath.ToSlash(b.calls),
	}
	return b
}

func mustMkdir(t *testing.T, d string) {
	t.Helper()
	if err := os.MkdirAll(d, 0o755); err != nil {
		t.Fatal(err)
	}
}

func mustWrite(t *testing.T, p, s string) {
	t.Helper()
	if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
		t.Fatal(err)
	}
}

// jauge écrit le fichier de jauge, mesuré `age` avant maintenant.
func (b *bancNuit) jauge(auDessus int, age time.Duration) {
	mustWrite(b.t, filepath.Join(b.dir, "data", "global", "admin_state", "passes_par_match.txt"), fmt.Sprintf(
		"# en-tete\nmesure_utc=%s\nseuil=1.50\ntables_au_dessus_du_seuil=%d\ntable=halo_infinite match_kill_events brutes=10 gardees=2 rapport=5.00\n",
		time.Now().UTC().Add(-age).Format(time.RFC3339), auDessus))
}

// jouer exécute le script avec les réglages `env` (FAKE_*) et rend les appels notés.
func (b *bancNuit) jouer(env ...string) []string {
	b.t.Helper()
	root := repoRootForWorkflows(b.t)
	// Le PATH des faux est posé DANS bash : le lanceur de Git for Windows (bin/bash.exe) place ses
	// propres dossiers (curl, df…) devant le PATH reçu.
	cmd := exec.Command(bashForScripts(b.t), "-c", lanceurAvecFaux, "_", b.fakeBin, "scripts/compact-passes-nuit.sh")
	cmd.Dir = root
	cmd.Env = append(append(os.Environ(), b.env...), env...)
	err := cmd.Run()
	b.exitCode = 0
	if ee, ok := err.(*exec.ExitError); ok {
		b.exitCode = ee.ExitCode()
	} else if err != nil {
		b.t.Fatalf("exécution du script : %v", err)
	}
	data, _ := os.ReadFile(b.calls)
	var out []string
	for _, l := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if strings.HasPrefix(l, "docker ") {
			out = append(out, l)
		}
	}
	return out
}

func (b *bancNuit) log() string {
	data, _ := os.ReadFile(b.logPath)
	return string(data)
}

func (b *bancNuit) attendre(code int, docker []string, gotDocker []string) {
	b.t.Helper()
	if b.exitCode != code {
		b.t.Errorf("code de sortie %d, attendu %d\njournal :\n%s", b.exitCode, code, b.log())
	}
	if strings.Join(gotDocker, "\n") != strings.Join(docker, "\n") {
		b.t.Errorf("appels docker :\n%s\nattendu :\n%s\njournal :\n%s",
			strings.Join(gotDocker, "\n"), strings.Join(docker, "\n"), b.log())
	}
}

// lanceurAvecFaux : met le dossier des faux binaires ($1, chemin converti sous Git for Windows) en
// tête du PATH puis exécute le script ($2).
const lanceurAvecFaux = `d="$1"; if command -v cygpath >/dev/null 2>&1; then d="$(cygpath -u "$1")"; fi; export PATH="$d:$PATH"; exec bash "$2"`

const (
	appelStop  = "docker compose stop levelup"
	appelRun   = "docker compose run --rm --no-deps -u appuser --entrypoint levelup levelup compact-passes --backup-dir /app/data/compaction-backups"
	appelStart = "docker compose start levelup"
)

func TestCompactPassesNuit_SousLeSeuilNeTouchePasAuServeur(t *testing.T) {
	b := nouveauBancNuit(t)
	b.jauge(0, time.Hour)
	b.attendre(0, nil, b.jouer())
	if !strings.Contains(b.log(), "rien a faire") {
		t.Errorf("journal sans « rien a faire » :\n%s", b.log())
	}
}

func TestCompactPassesNuit_AuDessusArreteCompacteRedemarre(t *testing.T) {
	b := nouveauBancNuit(t)
	b.jauge(1, time.Hour)
	b.attendre(0, []string{appelStop, appelRun, appelStart}, b.jouer())
	if etat, _ := os.ReadFile(b.etat); strings.TrimSpace(string(etat)) != "ok" {
		t.Errorf("état après réussite = %q, attendu ok", etat)
	}
}

func TestCompactPassesNuit_CompactionEnEchecRedemarreQuandMeme(t *testing.T) {
	b := nouveauBancNuit(t)
	b.jauge(2, time.Hour)
	b.attendre(1, []string{appelStop, appelRun, appelStart}, b.jouer("FAKE_RUN_RC=3"))
	if etat, _ := os.ReadFile(b.etat); strings.TrimSpace(string(etat)) != "echec" {
		t.Errorf("état après échec = %q, attendu echec", etat)
	}
}

func TestCompactPassesNuit_HealthMuetApresRedemarrage(t *testing.T) {
	b := nouveauBancNuit(t)
	b.jauge(1, time.Hour)
	b.attendre(1, []string{appelStop, appelRun, appelStart}, b.jouer("FAKE_HEALTH_RC=7"))
	if !strings.Contains(b.log(), "/health ne repond pas") {
		t.Errorf("journal sans l'erreur /health :\n%s", b.log())
	}
}

func TestCompactPassesNuit_ArretEnEchecTenteLeRedemarrage(t *testing.T) {
	b := nouveauBancNuit(t)
	b.jauge(1, time.Hour)
	b.attendre(1, []string{appelStop, appelStart}, b.jouer("FAKE_STOP_RC=1"))
}

func TestCompactPassesNuit_RefusSansRienToucher(t *testing.T) {
	cas := []struct {
		nom   string
		age   time.Duration
		jauge bool
		env   []string
		code  int
	}{
		{"disque insuffisant", time.Hour, true, []string{"FAKE_DF_LIBRE_KO=10"}, 1},
		{"verrou tenu", time.Hour, true, []string{"FAKE_FLOCK_RC=1"}, 75},
		{"restic en cours", time.Hour, true, []string{"FAKE_RESTIC_ACTIF=1"}, 75},
		{"jauge périmée", 49 * time.Hour, true, nil, 1},
		{"jauge absente", 0, false, nil, 1},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			b := nouveauBancNuit(t)
			if c.jauge {
				b.jauge(1, c.age)
			}
			b.attendre(c.code, nil, b.jouer(c.env...))
		})
	}
}

// TestCompactPassesNuit_MenageDesSauvegardes : après un passage réussi, seules les sauvegardes
// `*.avant-compaction-*` de plus de 24 h partent ; après un passage en échec, rien ne part.
func TestCompactPassesNuit_MenageDesSauvegardes(t *testing.T) {
	for _, etat := range []string{"ok", "echec"} {
		t.Run(etat, func(t *testing.T) {
			b := nouveauBancNuit(t)
			mustMkdir(t, b.sauvDir)
			mustMkdir(t, filepath.Dir(b.etat))
			vieille := filepath.Join(b.sauvDir, "shared_matches_v2.halo_infinite.avant-compaction-20261019T033000Z.duckdb")
			recente := filepath.Join(b.sauvDir, "shared_matches_v2.halo_infinite.avant-compaction-20261021T033000Z.duckdb")
			autre := filepath.Join(b.sauvDir, "notes.txt")
			for _, p := range []string{vieille, recente, autre} {
				mustWrite(t, p, "x")
			}
			deuxJours := time.Now().Add(-48 * time.Hour)
			for _, p := range []string{vieille, autre} {
				if err := os.Chtimes(p, deuxJours, deuxJours); err != nil {
					t.Fatal(err)
				}
			}
			mustWrite(t, b.etat, etat+"\n")
			b.jauge(0, time.Hour)
			b.attendre(0, nil, b.jouer())
			_, errV := os.Stat(vieille)
			if (etat == "ok") != os.IsNotExist(errV) {
				t.Errorf("état %s : vieille sauvegarde présente=%v", etat, errV == nil)
			}
			for _, p := range []string{recente, autre} {
				if _, err := os.Stat(p); err != nil {
					t.Errorf("état %s : %s supprimé à tort", etat, filepath.Base(p))
				}
			}
		})
	}
}
