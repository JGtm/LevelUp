package infinite

import (
	"strings"
	"testing"
)

// TestEndpoints_Defauts : chaque clé a un hôte par défaut ; les deux hôtes des stats restent
// distincts (avec et sans `:443`).
func TestEndpoints_Defauts(t *testing.T) {
	e := DefaultEndpoints()
	cles := []Cle{CleStats, CleGameCMS, CleEconomy, CleSkill, CleUGCFilm, CleDiscoveryUGC, CleChallenges,
		CleNameplate, CleStatsSansPort, CleWaypointWeb}
	for _, k := range cles {
		if h := e.Hote(k); !strings.HasPrefix(h, "https://") || strings.HasSuffix(h, "/") {
			t.Errorf("hôte par défaut de %q = %q", k, h)
		}
	}
	if !strings.HasSuffix(e.Hote(CleStats), ":443") || strings.HasSuffix(e.Hote(CleStatsSansPort), ":443") {
		t.Errorf("stats = %q, stats sans port = %q", e.Hote(CleStats), e.Hote(CleStatsSansPort))
	}
	if strings.TrimSuffix(e.Hote(CleStats), ":443") != e.Hote(CleStatsSansPort) {
		t.Error("les deux hôtes des stats désignent le même service, au port près")
	}
	if e.PrefixeJeu() != "hi" || e.base(CleSkill) != e.Hote(CleSkill)+"/hi" {
		t.Errorf("préfixe = %q, base = %q", e.PrefixeJeu(), e.base(CleSkill))
	}
	if len(hotesParDefaut()) != len(cles) {
		t.Errorf("%d hôtes par défaut, %d clés testées", len(hotesParDefaut()), len(cles))
	}
}

// TestEndpoints_Avec : copie, barre finale retirée, valeur vide = défaut, préfixe de jeu.
func TestEndpoints_Avec(t *testing.T) {
	e := DefaultEndpoints()
	e2 := e.Avec(CleEconomy, " http://local:1234/ ").AvecPrefixeJeu("h5")
	if e2.Hote(CleEconomy) != "http://local:1234" || e2.base(CleEconomy) != "http://local:1234/h5" {
		t.Errorf("e2 = %q / %q", e2.Hote(CleEconomy), e2.base(CleEconomy))
	}
	if e.Hote(CleEconomy) == e2.Hote(CleEconomy) || e.PrefixeJeu() != "hi" {
		t.Error("l'original a changé")
	}
	if e3 := e2.Avec(CleEconomy, ""); e3.Hote(CleEconomy) != e.Hote(CleEconomy) {
		t.Errorf("valeur vide : %q, veut le défaut", e3.Hote(CleEconomy))
	}
	if (Endpoints{}).Hote(CleSkill) != e.Hote(CleSkill) || (Endpoints{}).PrefixeJeu() != "hi" {
		t.Error("une valeur zéro retombe sur les défauts")
	}
}
