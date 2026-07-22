package logging

import "testing"

// Verrouille le mapping package → module (ADR 0029 : les logs du middleware
// d'autorisation doivent atterrir dans logs/http.log, pas general.log).
func TestMapPackageToModule(t *testing.T) {
	cases := map[string]string{
		"middleware":     ModuleHTTP,
		"api":            ModuleHTTP,
		"handlers":       ModuleHandlers,
		"service":        ModuleService,
		"external":       ModuleNotif, // relais Discord coach → logs/notifications.log
		"notify":         ModuleNotif, // client webhook Discord → logs/notifications.log
		"sync":           ModuleSync,
		"duckdb":         ModuleDuckDB,
		"sharedprovider": ModuleProvider,
		"persist":        ModulePersist,
		"session":        ModuleSession, // platform/session → logs/session.log (diag torn-read login loop)
	}
	for pkg, want := range cases {
		full := "levelup/go-api/internal/" + pkg + ".SomeFunc"
		if got := mapPackageToModule(pkg, full); got != want {
			t.Errorf("mapPackageToModule(%q) = %q, want %q", pkg, got, want)
		}
	}
}

// Un package inconnu retombe sur "general".
func TestMapPackageToModule_UnknownFallsBackToGeneral(t *testing.T) {
	if got := mapPackageToModule("wibble", "levelup/go-api/internal/wibble.Fn"); got != ModuleGeneral {
		t.Errorf("paquet inconnu = %q, want %q", got, ModuleGeneral)
	}
}
