package config

import "testing"

func boolPtr(b bool) *bool { return &b }

// La capacité KeepMediaSources prime sur la variable d'environnement, le réglage et la production.
func TestMediaDeleteSource_CapaciteForceFaux(t *testing.T) {
	t.Setenv(EnvMediaDeleteSource, "1")
	cfg := &AppConfig{Environment: "production"}
	cfg.Capabilities.KeepMediaSources = true
	if cfg.MediaDeleteSource(boolPtr(true)) {
		t.Fatal("capacité vraie + variable vraie + réglage vrai : attendu faux")
	}
	if cfg.MediaDeleteSource(nil) {
		t.Fatal("capacité vraie + variable vraie + production : attendu faux")
	}
}

// Capacité à zéro : variable, puis réglage, puis défaut de l'environnement.
func TestMediaDeleteSource_SansCapaciteComportementActuel(t *testing.T) {
	cas := []struct {
		nom   string
		env   string
		store *bool
		prod  bool
		want  bool
	}{
		{"variable gagne sur le réglage", "1", boolPtr(false), false, true},
		{"variable fausse gagne", "0", boolPtr(true), true, false},
		{"réglage sans variable", "", boolPtr(true), false, true},
		{"défaut production", "", nil, true, true},
		{"défaut hors production", "", nil, false, false},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			t.Setenv(EnvMediaDeleteSource, c.env)
			cfg := &AppConfig{}
			if c.prod {
				cfg.Environment = "production"
			}
			if got := cfg.MediaDeleteSource(c.store); got != c.want {
				t.Fatalf("MediaDeleteSource = %v, attendu %v", got, c.want)
			}
		})
	}
}
