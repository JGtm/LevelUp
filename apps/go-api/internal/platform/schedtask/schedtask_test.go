package schedtask

import (
	"context"
	"encoding/xml"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
	"unicode/utf16"
)

// call : un appel de schtasks enregistré.
type call struct {
	args []string
	// xmlBody : le contenu du fichier passé à /XML, lu PENDANT l'appel (le fichier est éphémère).
	xmlBody []byte
}

// fakeRunner simule schtasks : une liste de tâches (CSV de /Query) et des échecs réglables.
type fakeRunner struct {
	calls   []call
	listing string
	failOn  string // premier argument d'une commande à faire échouer (/Create, /Delete, /Query)
}

func (f *fakeRunner) Run(_ context.Context, args ...string) ([]byte, error) {
	c := call{args: append([]string(nil), args...)}
	for i, a := range args {
		if a == "/XML" && i+1 < len(args) {
			c.xmlBody, _ = os.ReadFile(args[i+1])
		}
	}
	f.calls = append(f.calls, c)
	if f.failOn != "" && args[0] == f.failOn {
		return nil, errors.New("accès refusé")
	}
	if args[0] == "/Query" {
		return []byte(f.listing), nil
	}
	return nil, nil
}

func (f *fakeRunner) commands() []string {
	var out []string
	for _, c := range f.calls {
		out = append(out, c.args[0])
	}
	return out
}

// testRoot : un dossier absolu sur l'OS courant (la validation du paquet est celle de l'OS : `C:\...` n'est
// pas absolu sous Linux), avec un espace dans le nom comme un profil Windows.
var testRoot = filepath.Join(os.TempDir(), "Test User")

var testSpec = Spec{
	ExePath:       filepath.Join(testRoot, "Programs", "LevelUp", "levelup-server.exe"),
	DataRoot:      filepath.Join(testRoot, "AppData", "Local", "LevelUp"),
	ResourcesRoot: filepath.Join(testRoot, "Programs", "LevelUp"),
	LogsRoot:      filepath.Join(testRoot, "AppData", "Local", "LevelUp", "logs"),
}

// testTaskName : le nom de la tâche du gestionnaire de test (empreinte de testSpec.DataRoot).
var testTaskName = TaskNameFor(testSpec.DataRoot)

var fixedNow = func() time.Time { return time.Date(2026, 11, 3, 8, 30, 0, 0, time.Local) }

func newManager(r Runner) *Manager {
	return &Manager{Runner: r, Spec: testSpec, Now: fixedNow, TempDir: os.TempDir()}
}

var listingWithTask = `"\Microsoft\Windows\Defrag\ScheduledDefrag","N/A","Ready"
"\` + testTaskName + `","11/3/2026 2:30:00 PM","Ready"
"\Autre","N/A","Ready"
`

// listingWithout : sans la tâche de cette racine, mais avec celle d'une autre racine (l'application) : elle ne compte pas.
var listingWithout = `"\Microsoft\Windows\Defrag\ScheduledDefrag","N/A","Ready"
"\LevelUp\Autre","N/A","Ready"
"\` + TaskNameFor(filepath.Join(testRoot, "autre-racine")) + `","N/A","Ready"
`

// decodeUTF16 rend le texte d'un fichier UTF-16 petit-boutiste avec BOM.
func decodeUTF16(t *testing.T, raw []byte) string {
	t.Helper()
	if len(raw) < 2 || raw[0] != 0xFF || raw[1] != 0xFE {
		t.Fatalf("BOM UTF-16 LE absent : %v", raw[:min(4, len(raw))])
	}
	units := make([]uint16, 0, (len(raw)-2)/2)
	for i := 2; i+1 < len(raw); i += 2 {
		units = append(units, uint16(raw[i])|uint16(raw[i+1])<<8)
	}
	return string(utf16.Decode(units))
}

func TestQuery_ArgumentsExactsEtLecteDeLaListe(t *testing.T) {
	r := &fakeRunner{listing: listingWithTask}
	m := newManager(r)
	state, err := m.Query(context.Background())
	if err != nil || state != StatePresent {
		t.Fatalf("Query = %q, %v ; attendu present", state, err)
	}
	if want := []string{"/Query", "/FO", "CSV", "/NH"}; !reflect.DeepEqual(r.calls[0].args, want) {
		t.Fatalf("arguments = %v, attendu %v", r.calls[0].args, want)
	}
	r.listing = listingWithout
	if state, err := m.Query(context.Background()); err != nil || state != StateAbsent {
		t.Fatalf("Query sans la tâche = %q, %v ; attendu absent (une tâche voisine du même dossier ne compte pas)", state, err)
	}
	r.failOn = "/Query"
	if state, err := m.Query(context.Background()); err == nil || state != StateError {
		t.Fatalf("Query en échec = %q, %v ; attendu error", state, err)
	}
}

func TestEnsure_ArgumentsExactsEtDefinitionDeLaTache(t *testing.T) {
	r := &fakeRunner{}
	if err := newManager(r).Ensure(context.Background()); err != nil {
		t.Fatalf("Ensure : %v", err)
	}
	if len(r.calls) != 1 {
		t.Fatalf("%d appels, attendu 1 (/Create seul)", len(r.calls))
	}
	args := r.calls[0].args
	if len(args) != 6 || args[0] != "/Create" || args[1] != "/TN" || args[2] != testTaskName || args[3] != "/XML" || args[5] != "/F" {
		t.Fatalf("arguments = %v, attendu /Create /TN %s /XML <fichier> /F", args, testTaskName)
	}
	if _, err := os.Stat(args[4]); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("le fichier de définition doit être supprimé après l'appel : %v", err)
	}

	doc := decodeUTF16(t, r.calls[0].xmlBody)
	var task struct {
		Triggers struct {
			Time struct {
				Interval string `xml:"Repetition>Interval"`
				Start    string `xml:"StartBoundary"`
			} `xml:"TimeTrigger"`
		} `xml:"Triggers"`
		Principal struct {
			Logon string `xml:"LogonType"`
			Level string `xml:"RunLevel"`
		} `xml:"Principals>Principal"`
		Settings struct {
			Start     string `xml:"StartWhenAvailable"`
			Battery   string `xml:"DisallowStartIfOnBatteries"`
			Instances string `xml:"MultipleInstancesPolicy"`
			Limit     string `xml:"ExecutionTimeLimit"`
			Hidden    string `xml:"Hidden"`
		} `xml:"Settings"`
		Exec struct {
			Command   string `xml:"Command"`
			Arguments string `xml:"Arguments"`
			Workdir   string `xml:"WorkingDirectory"`
		} `xml:"Actions>Exec"`
	}
	utf8Doc := strings.Replace(doc, `encoding="UTF-16"`, `encoding="UTF-8"`, 1)
	if err := xml.Unmarshal([]byte(utf8Doc), &task); err != nil {
		t.Fatalf("la définition n'est pas du XML valide : %v\n%s", err, doc)
	}
	if task.Triggers.Time.Interval != "PT6H" || task.Triggers.Time.Start != "2026-11-03T08:30:00" {
		t.Errorf("déclencheur = %+v, attendu toutes les 6 h depuis 2026-11-03T08:30:00", task.Triggers.Time)
	}
	if task.Principal.Level != "LeastPrivilege" || task.Principal.Logon != "InteractiveToken" {
		t.Errorf("compte = %+v, attendu sans élévation (LeastPrivilege), session interactive", task.Principal)
	}
	if task.Settings.Start != "true" || task.Settings.Battery != "false" || task.Settings.Instances != "IgnoreNew" ||
		task.Settings.Limit != "PT15M" || task.Settings.Hidden != "true" {
		t.Errorf("réglages = %+v", task.Settings)
	}
	if task.Exec.Command != "cmd.exe" || task.Exec.Workdir != testSpec.ResourcesRoot {
		t.Errorf("action = %+v", task.Exec)
	}
	wantArgs := `/d /s /c "set "LEVELUP_DATA_DIR=` + testSpec.DataRoot + `" && set "LEVELUP_RESOURCES_DIR=` +
		testSpec.ResourcesRoot + `" && set "LEVELUP_LOGS_DIR=` + testSpec.LogsRoot + `" && "` + testSpec.ExePath +
		`" -mode refresh-only"`
	if task.Exec.Arguments != wantArgs {
		t.Errorf("arguments de l'action =\n%s\nattendu\n%s", task.Exec.Arguments, wantArgs)
	}
	if strings.Contains(doc, "HighestAvailable") {
		t.Error("la tâche ne demande jamais l'élévation")
	}
}

// TestEnsure_Idempotent : deux appels laissent la même tâche (/F remplace), avec la même définition.
func TestEnsure_Idempotent(t *testing.T) {
	r := &fakeRunner{}
	m := newManager(r)
	for range 2 {
		if err := m.Ensure(context.Background()); err != nil {
			t.Fatalf("Ensure : %v", err)
		}
	}
	if len(r.calls) != 2 || r.calls[0].args[0] != "/Create" || r.calls[1].args[0] != "/Create" {
		t.Fatalf("appels = %v", r.commands())
	}
	if string(r.calls[0].xmlBody) != string(r.calls[1].xmlBody) {
		t.Error("deux Ensure du même instant doivent écrire la même définition")
	}
	if r.calls[0].args[5] != "/F" {
		t.Error("/F remplace une tâche existante : c'est ce qui rend Ensure idempotent")
	}
}

func TestEnsure_ErreurDeSchtasksRendueEtFichierSupprime(t *testing.T) {
	r := &fakeRunner{failOn: "/Create"}
	err := newManager(r).Ensure(context.Background())
	if err == nil || !strings.Contains(err.Error(), "schtasks /Create") || !strings.Contains(err.Error(), "accès refusé") {
		t.Fatalf("erreur = %v", err)
	}
	if _, serr := os.Stat(r.calls[0].args[4]); !errors.Is(serr, os.ErrNotExist) {
		t.Error("le fichier de définition est supprimé même après un échec")
	}
}

func TestRemove_AbsenteRienNEstSupprime_PresenteUnDeleteExact(t *testing.T) {
	r := &fakeRunner{listing: listingWithout}
	m := newManager(r)
	if err := m.Remove(context.Background()); err != nil {
		t.Fatalf("Remove (absente) : %v", err)
	}
	if got := r.commands(); !reflect.DeepEqual(got, []string{"/Query"}) {
		t.Fatalf("tâche absente : appels = %v, attendu seulement /Query (retirer ce qui n'existe pas est un succès)", got)
	}

	r = &fakeRunner{listing: listingWithTask}
	m = newManager(r)
	if err := m.Remove(context.Background()); err != nil {
		t.Fatalf("Remove (présente) : %v", err)
	}
	if got := r.calls[1].args; !reflect.DeepEqual(got, []string{"/Delete", "/TN", testTaskName, "/F"}) {
		t.Fatalf("suppression = %v", got)
	}

	r = &fakeRunner{listing: listingWithTask, failOn: "/Delete"}
	if err := newManager(r).Remove(context.Background()); err == nil || !strings.Contains(err.Error(), "schtasks /Delete") {
		t.Fatalf("échec de suppression : %v", err)
	}
	r = &fakeRunner{failOn: "/Query"}
	if err := newManager(r).Remove(context.Background()); err == nil {
		t.Fatal("liste illisible : l'erreur est rendue, rien n'est supprimé à l'aveugle")
	}
}

func TestEnsure_CheminsRefusesAvantToutAppel(t *testing.T) {
	// Chaque cas réservé part d'un chemin ABSOLU sur l'OS courant et y ajoute le caractère interdit : le refus
	// vient bien du caractère, pas d'un chemin devenu relatif.
	avec := func(c string) string { return filepath.Join(testRoot, "a"+c+"b") }
	cas := map[string]struct {
		mutate  func(*Spec)
		absolus bool // les quatre chemins restent absolus : seul le caractère réservé peut refuser
	}{
		"exécutable vide":          {func(s *Spec) { s.ExePath = "" }, false},
		"racine relative":          {func(s *Spec) { s.DataRoot = `donnees` }, false},
		"guillemet":                {func(s *Spec) { s.LogsRoot = avec(`"`) }, true},
		"pourcentage":              {func(s *Spec) { s.DataRoot = filepath.Join(testRoot, `%TEMP%`) }, true},
		"esperluette":              {func(s *Spec) { s.ResourcesRoot = avec(`&`) }, true},
		"point d'exclamation":      {func(s *Spec) { s.ExePath = filepath.Join(avec(`!`), "levelup-server.exe") }, true},
		"retour à la ligne":        {func(s *Spec) { s.LogsRoot = avec("\n") }, true},
		"redirection":              {func(s *Spec) { s.DataRoot = avec(`>`) }, true},
		"accent circonflexe":       {func(s *Spec) { s.DataRoot = avec(`^`) }, true},
		"barre verticale":          {func(s *Spec) { s.DataRoot = avec(`|`) }, true},
		"racine des ressources":    {func(s *Spec) { s.ResourcesRoot = "" }, false},
		"racine des journaux vide": {func(s *Spec) { s.LogsRoot = "  " }, false},
	}
	for nom, c := range cas {
		t.Run(nom, func(t *testing.T) {
			spec := testSpec
			c.mutate(&spec)
			if c.absolus {
				for _, p := range []string{spec.ExePath, spec.DataRoot, spec.ResourcesRoot, spec.LogsRoot} {
					if !filepath.IsAbs(p) {
						t.Fatalf("précondition : %q doit être absolu sur cet OS", p)
					}
				}
			}
			r := &fakeRunner{}
			m := &Manager{Runner: r, Spec: spec, Now: fixedNow}
			if err := m.Ensure(context.Background()); !errors.Is(err, ErrUnsupportedPath) {
				t.Fatalf("erreur = %v, attendu ErrUnsupportedPath", err)
			}
			if len(r.calls) != 0 {
				t.Fatalf("aucun appel de schtasks avec un chemin refusé : %v", r.commands())
			}
		})
	}
}

// TestTaskNameFor : le nom porte l'empreinte de la racine des données, stable pour une même racine (chemin
// nettoyé, casse ignorée) et distinct d'une racine à l'autre ; jamais le nom nu des anciennes versions.
func TestTaskNameFor(t *testing.T) {
	a := TaskNameFor(testSpec.DataRoot)
	if !strings.HasPrefix(a, TaskNamePrefix+"-") || len(a) != len(TaskNamePrefix)+1+8 {
		t.Fatalf("nom = %q, attendu %s-<8 hex>", a, TaskNamePrefix)
	}
	if got := TaskNameFor(strings.ToUpper(testSpec.DataRoot) + string(filepath.Separator)); got != a {
		t.Errorf("même racine (casse, séparateur final) : %q, attendu %q", got, a)
	}
	if b := TaskNameFor(filepath.Join(testRoot, "autre-racine")); b == a {
		t.Errorf("deux racines distinctes portent le même nom %q", a)
	}
}

// TestManager_NeTouchePasLaTacheDUneAutreRacine : la suppression ne vise que le nom de SA racine ; la tâche
// d'une autre racine (l'application de l'utilisateur) listée au même moment n'est ni vue ni supprimée.
func TestManager_NeTouchePasLaTacheDUneAutreRacine(t *testing.T) {
	autre := TaskNameFor(filepath.Join(testRoot, "autre-racine"))
	r := &fakeRunner{listing: `"\` + autre + `","N/A","Ready"` + "\n"}
	m := newManager(r)
	if state, err := m.Query(context.Background()); err != nil || state != StateAbsent {
		t.Fatalf("Query = %q, %v ; la tâche d'une autre racine ne compte pas", state, err)
	}
	if err := m.Remove(context.Background()); err != nil {
		t.Fatalf("Remove : %v", err)
	}
	if got := r.commands(); !reflect.DeepEqual(got, []string{"/Query", "/Query"}) {
		t.Fatalf("appels = %v, attendu aucune suppression", got)
	}
	if m.TaskName() != testTaskName {
		t.Errorf("TaskName = %q, attendu %q", m.TaskName(), testTaskName)
	}
}

func TestTaskListed(t *testing.T) {
	for _, c := range []struct {
		csv  string
		name string
		want bool
	}{
		{listingWithTask, testTaskName, true},
		{listingWithTask, `\` + testTaskName, true},
		{strings.ToLower(listingWithTask), testTaskName, true}, // casse ignorée
		{listingWithout, testTaskName, false},
		{`"\` + testTaskName + `2","N/A","Ready"`, testTaskName, false},
		{"", testTaskName, false},
		{"INFO: aucune tâche", testTaskName, false},
	} {
		if got := taskListed([]byte(c.csv), c.name); got != c.want {
			t.Errorf("taskListed(%q, %q) = %v, attendu %v", c.csv, c.name, got, c.want)
		}
	}
}

func TestObservedEtLastLine(t *testing.T) {
	var o Observed
	if s, c := o.Get(); s != "" || c != "" {
		t.Fatalf("rien observé : %q %q", s, c)
	}
	o.Set(StateError, ErrorCodeSchtasks)
	if s, c := o.Get(); s != StateError || c != ErrorCodeSchtasks {
		t.Fatalf("après Set : %q %q", s, c)
	}
	o.Set(StatePresent, "")
	if s, c := o.Get(); s != StatePresent || c != "" {
		t.Fatalf("un état sans erreur efface le code : %q %q", s, c)
	}
	if got := lastLine("", "ligne 1\nligne 2\n"); got != "ligne 2" {
		t.Errorf("lastLine = %q", got)
	}
	if got := lastLine("", " "); got != "aucun message" {
		t.Errorf("lastLine vide = %q", got)
	}
}
