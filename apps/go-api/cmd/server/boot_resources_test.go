package main

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/platform/netguard"
	"levelup/go-api/internal/resourcepack"
)

// TestNewResourcesDownloaderPasseParLaGarde : le téléchargeur du paquet sort par netguard, sous sa surface nommée.
func TestNewResourcesDownloaderPasseParLaGarde(t *testing.T) {
	guarded, ok := newResourcesDownloader().Doer.(netguard.Doer)
	if !ok {
		t.Fatalf("Doer du téléchargeur = %T, attendu netguard.Doer", newResourcesDownloader().Doer)
	}
	if guarded.Surface != resourcesSurface {
		t.Errorf("surface = %q, attendu %q", guarded.Surface, resourcesSurface)
	}
}

// TestResourceActionFor : ce que chaque profil fait de chaque verdict (ADR 0045, amendement L24).
func TestResourceActionFor(t *testing.T) {
	hosted := config.DeploymentCapabilities{ResourcePackFatal: true}
	desktop := config.DeploymentCapabilities{UpdateApp: true}
	dev := config.DeploymentCapabilities{}
	cases := []struct {
		profil string
		caps   config.DeploymentCapabilities
		state  resourcepack.State
		want   resourceAction
	}{
		{"hébergé conforme", hosted, resourcepack.StateMatch, resourceOK},
		{"hébergé différent", hosted, resourcepack.StateMismatch, resourceRefuse},
		{"hébergé absent", hosted, resourcepack.StateAbsent, resourceRefuse},
		{"hébergé illisible", hosted, resourcepack.StateInvalid, resourceRefuse},
		{"bureau conforme", desktop, resourcepack.StateMatch, resourceOK},
		{"bureau différent", desktop, resourcepack.StateMismatch, resourceCatchUp},
		{"bureau illisible", desktop, resourcepack.StateInvalid, resourceCatchUp},
		{"bureau absent", desktop, resourcepack.StateAbsent, resourceDownloadNow},
		{"développement différent", dev, resourcepack.StateMismatch, resourceWarn},
		{"développement absent", dev, resourcepack.StateAbsent, resourceWarn},
		{"sans empreinte embarquée", hosted, resourcepack.StateUnchecked, resourceOK},
	}
	for _, c := range cases {
		if got := resourceActionFor(c.caps, resourcepack.Verdict{State: c.state}); got != c.want {
			t.Errorf("%s : %v, attendu %v", c.profil, got, c.want)
		}
	}
}

func TestVerdictErrorNommeLaCause(t *testing.T) {
	for state, want := range map[resourcepack.State]string{
		resourcepack.StateAbsent:   resourcepack.ManifestFileName,
		resourcepack.StateMismatch: "n'est pas celui du moteur",
	} {
		err := verdictError(resourcepack.Verdict{State: state, Expected: "E", Active: "A"})
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s : %v", state, err)
		}
	}
	if err := verdictError(resourcepack.Verdict{State: resourcepack.StateInvalid, Err: context.Canceled}); err == nil ||
		!strings.Contains(err.Error(), "illisible") {
		t.Errorf("invalide : %v", err)
	}
}

// TestEvenementResources : l'événement tient en UNE ligne JSON, à plat, avec les champs que la coquille lit.
func TestEvenementResources(t *testing.T) {
	var buf bytes.Buffer
	newDesktopEvents(&buf).Resources(resourcepack.Status{
		Phase: resourcepack.PhaseReady, Fingerprint: "f", Expected: "e", Active: "a", Percent: 100, RestartNeeded: true})
	line := strings.TrimSuffix(buf.String(), "\n")
	if strings.Contains(line, "\n") {
		t.Fatalf("plus d'une ligne : %q", buf.String())
	}
	var got map[string]any
	if err := json.Unmarshal([]byte(line), &got); err != nil {
		t.Fatal(err)
	}
	want := map[string]any{"event": "resources", "phase": "ready", "fingerprint": "f", "expected": "e", "active": "a",
		"percent": float64(100), "restart_needed": true}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s = %v, attendu %v (ligne %s)", k, got[k], v, line)
		}
	}
}

// TestStdinCommandeAvecArgument : `resources <json>` atteint son gestionnaire avec le reste de la ligne ;
// l'argument ne figure jamais dans le journal des commandes inconnues.
func TestStdinCommandeAvecArgument(t *testing.T) {
	var cmds stdinCommands
	var got []string
	cmds.HandleArg("resources", func(arg string) { got = append(got, arg) })
	long := `{"fingerprint":"` + strings.Repeat("a", 64) + `","signature":"` + strings.Repeat("S", 600) + `"}`
	in := "resources " + long + "\nresources\nautre chose\n"
	if err := drainStdin(context.Background(), strings.NewReader(in), &cmds); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != long {
		t.Fatalf("arguments reçus : %v", got)
	}
}

func TestStartResourcesFromCommandRefuseUnDescripteurInvalide(t *testing.T) {
	m := resourcepack.NewManager(resourcepack.Config{Root: t.TempDir(), Expected: "e"})
	startResourcesFromCommand(context.Background(), m, "pas du json")
	startResourcesFromCommand(context.Background(), m, `{"fingerprint":"court"}`)
	if st := m.Status(); st.Phase != resourcepack.PhaseIdle {
		t.Fatalf("un descripteur invalide ne doit rien lancer : %+v", st)
	}
}
