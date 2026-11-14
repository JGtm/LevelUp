package trends

import (
	"testing"
	"time"

	"levelup/go-api/internal/domain"
)

var paceKeys = []string{
	domain.TrendsKeyKillsPerMinute,
	domain.TrendsKeyDeathsPerMinute,
	domain.TrendsKeyAssistsPerMinute,
	domain.TrendsKeyDamageDealtPerMinute,
	domain.TrendsKeyDamageTakenPerMinute,
}

// withPace pose les comptes et les secondes d'un match de test.
func withPace(kills, deaths, assists int, seconds float64, dealt, taken *float64) func(*Match) {
	return func(m *Match) {
		m.Kills, m.Deaths, m.Assists, m.Seconds = kills, deaths, assists, seconds
		m.DamageDealt, m.DamageTaken = dealt, taken
	}
}

// horizonValue : valeur de l'horizon de 365 jours d'une ligne, absente = nil.
func horizonValue(t *testing.T, resp domain.TrendsPageResponse, key string) *float64 {
	t.Helper()
	return horizon(t, indicatorOf(t, resp, key, ""), 365).Value
}

// Le rapport est la somme des comptes sur la somme des minutes, pas la moyenne des rapports.
func TestPace_SommeSurSommeEtNonMoyenneDesRapports(t *testing.T) {
	loc := paris(t)
	ms := []Match{
		// 10 frags en 5 minutes (2,0 par minute) puis 2 frags en 15 minutes (0,133 par minute).
		mk(loc, ago(2, 0), withPace(10, 6, 3, 300, f64(1500), f64(900))),
		mk(loc, ago(1, 0), withPace(2, 3, 0, 900, f64(600), f64(1200))),
	}
	resp := BuildSolo(ms, opts(loc))
	want := map[string]float64{
		domain.TrendsKeyKillsPerMinute:       12.0 / 20.0,
		domain.TrendsKeyDeathsPerMinute:      9.0 / 20.0,
		domain.TrendsKeyAssistsPerMinute:     3.0 / 20.0,
		domain.TrendsKeyDamageDealtPerMinute: 2100.0 / 20.0,
		domain.TrendsKeyDamageTakenPerMinute: 2100.0 / 20.0,
	}
	for key, w := range want {
		got := horizonValue(t, resp, key)
		if got == nil || !near(*got, w) {
			t.Errorf("%s = %v, attendu %v", key, got, w)
		}
	}
	meanOfRatios := (10.0/5 + 2.0/15) / 2
	if got := horizonValue(t, resp, domain.TrendsKeyKillsPerMinute); near(*got, meanOfRatios) {
		t.Errorf("frags par minute = moyenne des rapports (%v) : attendu somme sur somme", meanOfRatios)
	}
}

// Un match sans seconde jouée ne compte ni au numérateur ni au dénominateur.
func TestPace_MatchSansSecondesExclu(t *testing.T) {
	loc := paris(t)
	base := []Match{
		mk(loc, ago(2, 0), withPace(10, 5, 4, 600, f64(3000), f64(2000))),
		mk(loc, ago(1, 0), withPace(5, 5, 2, 600, f64(1000), f64(2000))),
	}
	sansDuree := mk(loc, ago(3, 0), withPace(100, 100, 100, 0, f64(99999), f64(99999)))
	avec := BuildSolo(append([]Match{sansDuree}, base...), opts(loc))
	sans := BuildSolo(base, opts(loc))
	for _, key := range paceKeys {
		a, b := horizonValue(t, avec, key), horizonValue(t, sans, key)
		if a == nil || b == nil || !near(*a, *b) {
			t.Errorf("%s : %v avec le match sans durée, %v sans lui", key, a, b)
		}
	}
	if got := horizonValue(t, sans, domain.TrendsKeyKillsPerMinute); !near(*got, 15.0/20.0) {
		t.Errorf("frags par minute = %v, attendu 0,75", *got)
	}
	// Série par match : le match sans durée ne donne aucun point.
	ind := indicatorOf(t, avec, domain.TrendsKeyKillsPerMinute, "")
	if len(ind.Series.Match) != 2 {
		t.Errorf("série par match = %d points, attendu 2 (match sans durée exclu)", len(ind.Series.Match))
	}
}

// Un ensemble dont aucun match n'a de seconde jouée n'a pas de ligne.
func TestPace_AucuneSecondeAucuneLigne(t *testing.T) {
	loc := paris(t)
	ms := many(loc, 6, ago(2, 0), withPace(5, 5, 1, 0, f64(100), f64(100)))
	resp := BuildSolo(ms, opts(loc))
	for _, key := range paceKeys {
		if hasIndicator(resp, key) {
			t.Errorf("%s présent sans aucune seconde jouée", key)
		}
	}
}

// Groupe, unité, sens et présence dans la matrice.
func TestPace_DefinitionsGroupeUniteSens(t *testing.T) {
	loc := paris(t)
	ms := many(loc, 6, ago(2, 0), withPace(8, 4, 2, 600, f64(2000), f64(1500)))
	resp := BuildSolo(ms, opts(loc))
	better := map[string]int{
		domain.TrendsKeyKillsPerMinute:       betterHigh,
		domain.TrendsKeyDeathsPerMinute:      betterLow,
		domain.TrendsKeyAssistsPerMinute:     betterHigh,
		domain.TrendsKeyDamageDealtPerMinute: betterHigh,
		domain.TrendsKeyDamageTakenPerMinute: betterLow,
	}
	for _, key := range paceKeys {
		ind := indicatorOf(t, resp, key, "")
		if ind.Group != domain.TrendsGroupPace || ind.Unit != domain.TrendsUnitPerMin || !ind.InMatrix {
			t.Errorf("%s : groupe %q, unité %q, in_matrix %v", key, ind.Group, ind.Unit, ind.InMatrix)
		}
		if ind.Better != better[key] {
			t.Errorf("%s : sens %d, attendu %d", key, ind.Better, better[key])
		}
		if len(ind.Months) != 12 || len(ind.Horizons) != 4 {
			t.Errorf("%s : %d mois et %d horizons", key, len(ind.Months), len(ind.Horizons))
		}
		if len(ind.Series.Match) != 6 {
			t.Errorf("%s : série par match = %d points, attendu 6", key, len(ind.Series.Match))
		}
	}
}

// Sans dégâts connus (titre ou matchs sans cette donnée), les lignes de dégâts sont
// absentes, jamais à zéro ; les frags, morts et assistances restent.
func TestPace_DegatsAbsentsPasZero(t *testing.T) {
	loc := paris(t)
	ms := many(loc, 6, ago(2, 0), withPace(8, 4, 2, 600, nil, nil))
	resp := BuildSolo(ms, opts(loc))
	for _, key := range []string{domain.TrendsKeyDamageDealtPerMinute, domain.TrendsKeyDamageTakenPerMinute} {
		if hasIndicator(resp, key) {
			t.Errorf("%s présent sans dégâts connus", key)
		}
	}
	for _, key := range []string{domain.TrendsKeyKillsPerMinute, domain.TrendsKeyDeathsPerMinute, domain.TrendsKeyAssistsPerMinute} {
		if !hasIndicator(resp, key) {
			t.Errorf("%s absent alors que les comptes sont connus", key)
		}
	}
}

// Seuls les matchs aux dégâts connus entrent dans le numérateur ET le dénominateur des dégâts
// par minute ; les frags par minute gardent tous les matchs durés.
func TestPace_DegatsPartielsMemeEnsembleAuxDeuxTermes(t *testing.T) {
	loc := paris(t)
	ms := []Match{
		mk(loc, ago(3, 0), withPace(10, 5, 2, 600, f64(3000), f64(1000))),
		mk(loc, ago(2, 0), withPace(10, 5, 2, 1800, nil, nil)),
		mk(loc, ago(1, 0), withPace(10, 5, 2, 600, f64(1000), nil)),
	}
	resp := BuildSolo(ms, opts(loc))
	if got := horizonValue(t, resp, domain.TrendsKeyDamageDealtPerMinute); got == nil || !near(*got, 4000.0/20.0) {
		t.Errorf("dégâts infligés par minute = %v, attendu 200", got)
	}
	if got := horizonValue(t, resp, domain.TrendsKeyDamageTakenPerMinute); got == nil || !near(*got, 1000.0/10.0) {
		t.Errorf("dégâts subis par minute = %v, attendu 100", got)
	}
	if got := horizonValue(t, resp, domain.TrendsKeyKillsPerMinute); got == nil || !near(*got, 30.0/50.0) {
		t.Errorf("frags par minute = %v, attendu 0,6", got)
	}
}

// La vue Escouade ne porte pas le groupe « Rythme » : elle ne lit que le FDA et la part de
// frags de chaque membre, et le bilan d'un allié n'a ni secondes jouées ni dégâts.
func TestPace_AbsentDeLaVueEscouade(t *testing.T) {
	me := &MemberStat{Kills: 8, Deaths: 3, Assists: 2}
	al := &MemberStat{Kills: 5, Deaths: 4, Assists: 1}
	var ms []Match
	for i := 0; i < 6; i++ {
		ms = append(ms, squadMatch(ago(2, time.Duration(i)*time.Minute), me, al, 20, withPace(8, 3, 2, 600, f64(2000), f64(1500))))
	}
	resp := BuildSquad(ms, squadOpts(nil, ""))
	for _, ind := range resp.Indicators {
		if ind.Group == domain.TrendsGroupPace {
			t.Errorf("la vue Escouade porte l'indicateur de rythme %s", ind.Key)
		}
	}
}
