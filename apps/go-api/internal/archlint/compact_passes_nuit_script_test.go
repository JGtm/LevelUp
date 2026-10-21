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
	// docker : FAKE_START_ECHECS = nombre de `compose start` qui échouent avant le premier qui réussit ;
	// FAKE_RUN_SIGTERM=1 : la compaction reçoit le SIGTERM de systemd (le script, son parent, aussi).
	"docker": `echo "docker $*" >>"$FAKE_CALLS"
case "$*" in
  "compose run"*)
    [ "${FAKE_RUN_SIGTERM:-0}" = 1 ] && kill -TERM "$PPID"
    exit "${FAKE_RUN_RC:-0}" ;;
  "compose stop"*) exit "${FAKE_STOP_RC:-0}" ;;
  "compose start"*)
    n=$(($(cat "$FAKE_CALLS.start" 2>/dev/null || echo 0) + 1)); echo "$n" >"$FAKE_CALLS.start"
    [ "$n" -le "${FAKE_START_ECHECS:-0}" ] && exit 1
    exit 0 ;;
esac
exit 0`,
	"curl": `echo "curl $*" >>"$FAKE_CALLS"; exit "${FAKE_HEALTH_RC:-0}"`,
	"df":   `printf 'Filesystem 1024-blocks Used Available Capacity Mounted\nfake 100 0 %s 0%% /\n' "${FAKE_DF_LIBRE_KO:-999999999}"`,
	// systemctl : modèle d'une unité oneshot sans RemainAfterExit — pendant la sauvegarde l'état est
	// `activating` et `is-active` rend 3 (seul `active` rend 0).
	"systemctl": `echo "systemctl $*" >>"$FAKE_CALLS"
etat="${FAKE_RESTIC_ETAT:-inactive}"
case "$1" in
  show) echo "$etat"; exit 0 ;;
  is-active) [ "$etat" = active ] && exit 0; exit 3 ;;
esac
exit 0`,
	"flock": `exit "${FAKE_FLOCK_RC:-0}"`,
}

// tailleBaseTestKo : taille de la base partagée du banc. Assez grosse pour qu'un calcul de place qui
// l'oublierait laisse passer un disque insuffisant (cas « disque : la base compte »).
const tailleBaseTestKo = 8192

// margeDisqueKo : MARGE_DISQUE_KO du script (2 Gio).
const margeDisqueKo = 2 * 1024 * 1024

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
	base := filepath.Join(b.dir, "data", "titles", "halo_infinite", "warehouse", "shared_matches_v2.duckdb")
	mustWrite(t, base, "")
	if err := os.Truncate(base, tailleBaseTestKo*1024); err != nil {
		t.Fatal(err)
	}
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
		"LEVELUP_COMPACT_START_DELAI_S=0",
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
	b.attendre(1, []string{appelStop, appelRun, appelStart, appelStart, appelStart}, b.jouer("FAKE_HEALTH_RC=7"))
	if !strings.Contains(b.log(), "/health ne repond pas") {
		t.Errorf("journal sans l'erreur /health :\n%s", b.log())
	}
}

// TestCompactPassesNuit_StartRateUneFoisPuisReussit : un `docker compose start` en échec ne laisse
// pas le site arrêté — le deuxième essai le relance et le passage réussit.
func TestCompactPassesNuit_StartRateUneFoisPuisReussit(t *testing.T) {
	b := nouveauBancNuit(t)
	b.jauge(1, time.Hour)
	b.attendre(0, []string{appelStop, appelRun, appelStart, appelStart}, b.jouer("FAKE_START_ECHECS=1"))
	if etat, _ := os.ReadFile(b.etat); strings.TrimSpace(string(etat)) != "ok" {
		t.Errorf("état = %q, attendu ok", etat)
	}
}

// TestCompactPassesNuit_StartToujoursEnEchec : trois essais (START_ESSAIS), pas un de plus (le filet
// ne les rejoue pas), code non nul.
func TestCompactPassesNuit_StartToujoursEnEchec(t *testing.T) {
	b := nouveauBancNuit(t)
	b.jauge(1, time.Hour)
	b.attendre(1, []string{appelStop, appelRun, appelStart, appelStart, appelStart}, b.jouer("FAKE_START_ECHECS=99"))
	if !strings.Contains(b.log(), "NON redemarre apres 3 essais") {
		t.Errorf("journal sans l'erreur finale :\n%s", b.log())
	}
}

func TestCompactPassesNuit_ArretEnEchecTenteLeRedemarrage(t *testing.T) {
	b := nouveauBancNuit(t)
	b.jauge(1, time.Hour)
	b.attendre(1, []string{appelStop, appelStart}, b.jouer("FAKE_STOP_RC=1"))
}

// TestCompactPassesNuit_SigtermPendantLaCompaction : le timeout systemd (SIGTERM) pendant la
// compaction passe par le filet — le serveur est relancé, le code est celui du signal.
func TestCompactPassesNuit_SigtermPendantLaCompaction(t *testing.T) {
	b := nouveauBancNuit(t)
	b.jauge(1, time.Hour)
	b.attendre(143, []string{appelStop, appelRun, appelStart}, b.jouer("FAKE_RUN_SIGTERM=1"))
	if !strings.Contains(b.log(), "filet : redemarrage du serveur") {
		t.Errorf("journal sans le filet :\n%s", b.log())
	}
}

// TestCompactPassesNuit_PlaceDisque : la place exigée est la taille des bases PLUS la marge — un
// disque qui couvre la marge mais pas la base est refusé, un disque qui couvre les deux passe.
func TestCompactPassesNuit_PlaceDisque(t *testing.T) {
	compacte := []string{appelStop, appelRun, appelStart}
	cas := []struct {
		nom    string
		libre  int
		code   int
		docker []string
	}{
		{"marge seule : la base compte", margeDisqueKo + tailleBaseTestKo/2, 1, nil},
		{"base et marge couvertes", margeDisqueKo + tailleBaseTestKo + 64, 0, compacte},
		{"presque rien", 10, 1, nil},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			b := nouveauBancNuit(t)
			b.jauge(1, time.Hour)
			b.attendre(c.code, c.docker, b.jouer(fmt.Sprintf("FAKE_DF_LIBRE_KO=%d", c.libre)))
		})
	}
}

// TestCompactPassesNuit_AgeDeLaJauge : la borne de 48 h, de part et d'autre, et le cas nominal
// d'une jauge de la veille (~23,5 h : le contrôle de santé tourne toutes les 24 h).
func TestCompactPassesNuit_AgeDeLaJauge(t *testing.T) {
	compacte := []string{appelStop, appelRun, appelStart}
	cas := []struct {
		nom    string
		age    time.Duration
		code   int
		docker []string
	}{
		{"nominal 23,5 h", 23*time.Hour + 30*time.Minute, 0, compacte},
		{"juste sous 48 h", 47*time.Hour + 50*time.Minute, 0, compacte},
		{"juste au-dessus de 48 h", 48*time.Hour + 10*time.Minute, 1, nil},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			b := nouveauBancNuit(t)
			b.jauge(1, c.age)
			b.attendre(c.code, c.docker, b.jouer())
		})
	}
}

// TestCompactPassesNuit_JaugeIllisible : un fichier corrompu n'est jamais lu comme « 0 table ».
func TestCompactPassesNuit_JaugeIllisible(t *testing.T) {
	for nom, contenu := range map[string]string{
		"date corrompue":     "mesure_utc=pas-une-date\ntables_au_dessus_du_seuil=0\n",
		"compte corrompu":    fmt.Sprintf("mesure_utc=%s\ntables_au_dessus_du_seuil=beaucoup\n", time.Now().UTC().Format(time.RFC3339)),
		"fichier tronqué":    "mesure_utc=2026-10",
		"octets quelconques": "\x00\x01\x02 binaire \xff\n",
	} {
		t.Run(nom, func(t *testing.T) {
			b := nouveauBancNuit(t)
			mustWrite(t, filepath.Join(b.dir, "data", "global", "admin_state", "passes_par_match.txt"), contenu)
			b.attendre(1, nil, b.jouer())
			if !strings.Contains(b.log(), "jauge illisible") {
				t.Errorf("journal sans « jauge illisible » :\n%s", b.log())
			}
		})
	}
}

func TestCompactPassesNuit_RefusSansRienToucher(t *testing.T) {
	cas := []struct {
		nom   string
		jauge bool
		env   []string
		code  int
	}{
		{"verrou tenu", true, []string{"FAKE_FLOCK_RC=1"}, 75},
		{"restic en cours (oneshot : activating)", true, []string{"FAKE_RESTIC_ETAT=activating"}, 75},
		{"restic en fin (deactivating)", true, []string{"FAKE_RESTIC_ETAT=deactivating"}, 75},
		{"jauge absente", false, nil, 1},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			b := nouveauBancNuit(t)
			if c.jauge {
				b.jauge(1, time.Hour)
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
