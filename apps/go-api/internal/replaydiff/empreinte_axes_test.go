package replaydiff

// empreinte_axes_test.go — LA SOMME D'UN CALQUE A DEUX NIVEAUX, PAS LE DERNIER GROUPE.
//
// Bug mesure le 2026-09-06 (cmd/replay-corpus-gate, premiere execution sur corpus temoin) :
// `flagCarries.spans/n` rendait 1 sur `084a804d` alors que le journal de cuisson disait
// « portages=15 fermes=15 » — `mesurerTableau` posait `prefixe+"/n"` par un SET (`e.num`), pas
// une SOMME, et un calque a deux niveaux (flagCarries[].spans[], vehicles[].rides[]...) appelle
// cette fonction UNE FOIS PAR GROUPE DE PREMIER NIVEAU (une equipe, un vehicule...) : la mesure
// finale n'etait que celle du DERNIER groupe itere. Ces tests verrouillent la somme.

import "testing"

// TestSpansDeCalqueImbriqueEstLaSomme — deux FlagCarry (deux equipes) de tailles DIFFERENTES :
// `flagCarries.spans/n` doit sommer les deux, pas ne retenir que le dernier itere.
func TestSpansDeCalqueImbriqueEstLaSomme(t *testing.T) {
	texte := `{"schemaVersion":43,"matchId":"a","flagCarries":[
		{"team":0,"spans":[{"t0":0,"t1":9,"xuid":"1"},{"t0":20,"t1":29,"xuid":"1"}]},
		{"team":1,"spans":[{"t0":5,"t1":14,"xuid":"2"}]}]}`
	e := Empreindre(doc(t, texte))
	m, ok := e.Mesures["ports/flagCarries.spans/n"]
	if !ok || m.Num != 3 {
		t.Fatalf("flagCarries.spans/n = %v (present=%v), attendu 3 (2 + 1, la SOMME des deux equipes)",
			m.Num, ok)
	}
}

// TestVehiclesRidesDeCalqueImbriqueEstLaSomme — meme defaut, sur un AUTRE calque a deux
// niveaux (vehicles[].rides[]) : la correction n'est pas specifique a flagCarries.
func TestVehiclesRidesDeCalqueImbriqueEstLaSomme(t *testing.T) {
	texte := `{"schemaVersion":43,"matchId":"a","vehicles":[
		{"slot":1,"t0":0,"t1":999,"rides":[{"t0":0,"t1":9,"slot":10},{"t0":50,"t1":59,"slot":11}]},
		{"slot":2,"t0":0,"t1":999,"rides":[{"t0":0,"t1":9,"slot":12}]},
		{"slot":3,"t0":0,"t1":999,"rides":[]}]}`
	e := Empreindre(doc(t, texte))
	m, ok := e.Mesures["vehicules/vehicles.rides/n"]
	if !ok || m.Num != 3 {
		t.Fatalf("vehicles.rides/n = %v (present=%v), attendu 3 (2 + 1 + 0, la SOMME des trois vehicules)",
			m.Num, ok)
	}
}

// TestCalqueRacineNAPasBesoinDeSomme — non-regression : au niveau RACINE (flagCarries lui-meme,
// pas ses spans), la mesure `/n` reste le compte du tableau de premier niveau — un seul appel,
// la correction (SET -> SOMME) n'y change rien.
func TestCalqueRacineNAPasBesoinDeSomme(t *testing.T) {
	texte := `{"schemaVersion":43,"matchId":"a","flagCarries":[
		{"team":0,"spans":[]}, {"team":1,"spans":[]}]}`
	e := Empreindre(doc(t, texte))
	m, ok := e.Mesures["ports/flagCarries/n"]
	if !ok || m.Num != 2 {
		t.Fatalf("flagCarries/n = %v (present=%v), attendu 2 (deux equipes)", m.Num, ok)
	}
}
