package mapvar

import (
	"math"
	"os"
	"path/filepath"
	"testing"
)

// fixtureDir : les .mvar de référence vivent sous .ai/re_dump/mapvar/ (artefacts de
// rétro-ingénierie déjà présents en dépôt). On ne les duplique pas dans testdata/ :
// ce sont des actifs propriétaires 343/Microsoft.
func fixture(t *testing.T, name string) []byte {
	t.Helper()
	path := filepath.Join("..", "..", "..", "..", "..", "..", ".ai", "re_dump", "mapvar", name)
	buf, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("fixture %s absente (%v) — test de parse ignoré", name, err)
	}
	return buf
}

// TestLabelTableIsSelfConsistent — GARDE-FOU principal, ne dépend d'aucune fixture.
// Chaque nom de la table doit re-hasher exactement sur sa clé. Une entrée devinée
// « à peu près » est donc impossible : elle ferait échouer ce test.
func TestLabelTableIsSelfConsistent(t *testing.T) {
	for hash, name := range labelNames {
		if got := LabelHash(name); got != hash {
			t.Errorf("labelNames[%d] = %q mais LabelHash(%q) = %d", hash, name, name, got)
		}
	}
}

// TestLabelHashKnownValue — ancre externe : la valeur observée dans ctf_breaker.mvar.
func TestLabelHashKnownValue(t *testing.T) {
	if got := LabelHash("stockpile_socket"); got != 2110778921 {
		t.Fatalf("LabelHash(\"stockpile_socket\") = %d, attendu 2110778921", got)
	}
}

func TestParseBreaker(t *testing.T) {
	v, err := Parse(fixture(t, "breaker_ctf_breaker.mvar"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(v.Objects) != 439 {
		t.Errorf("objets = %d, attendu 439", len(v.Objects))
	}
	types := map[int32]bool{}
	for _, o := range v.Objects {
		types[o.TypeID] = true
	}
	if len(types) != 26 {
		t.Errorf("type_id distincts = %d, attendu 26", len(types))
	}

	// Les 10 objets porteurs du label stockpile_socket doivent être exactement
	// ceux que la table de noms du fichier appelle stockpile_{blue,red}_socket_NN.
	var sockets []Object
	for _, o := range v.Objects {
		for _, l := range o.Labels {
			if labelNames[l] == "stockpile_socket" {
				sockets = append(sockets, o)
			}
		}
	}
	if len(sockets) != 10 {
		t.Fatalf("objets stockpile_socket = %d, attendu 10", len(sockets))
	}

	// TÉMOIN : la séquence d'équipes des sockets doit reproduire la séquence
	// bleu/rouge des noms d'instance. Sous hypothèse nulle (ordre arbitraire),
	// la probabilité de coïncidence est 1/C(10,5) = 1/252.
	wantBlue := []bool{true, false, true, true, true, true, false, false, false, false}
	for i, o := range sockets {
		isTeam1 := o.TeamIndex == 1
		if isTeam1 != wantBlue[i] {
			t.Errorf("socket %d: TeamIndex=%d, séquence bleu/rouge attendue %v", i, o.TeamIndex, wantBlue[i])
		}
	}
	if len(v.Names) != 20 {
		t.Errorf("noms = %d, attendu 20", len(v.Names))
	}
	if v.Names[0] != "stockpile_blue_socket_01" {
		t.Errorf("Names[0] = %q, attendu stockpile_blue_socket_01", v.Names[0])
	}
}

func TestParseCliffhanger(t *testing.T) {
	v, err := Parse(fixture(t, "cliffhanger_map.mvar"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(v.Objects) != 453 {
		t.Errorf("objets = %d, attendu 453", len(v.Objects))
	}
}

// TestFlagSpawnAndDeliveryColocated — TÉMOIN le plus fort disponible.
// Deux labels indépendants (flag_spawn, flag_delivery) portés par deux types
// d'objets DIFFÉRENTS doivent tomber au même point pour une même équipe : en CTF
// classique, on rapporte le drapeau adverse sur son propre socle.
func TestFlagSpawnAndDeliveryColocated(t *testing.T) {
	v, err := Parse(fixture(t, "cliffhanger_ridgeline.mvar"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	spawn := map[int]Vec3{}
	deliver := map[int]Vec3{}
	for _, o := range v.Objectives() {
		switch o.Role {
		case RoleFlagSpawn:
			spawn[o.TeamIndex] = o.Pos
		case RoleFlagDelivery:
			deliver[o.TeamIndex] = o.Pos
		}
	}
	if len(deliver) == 0 {
		t.Fatal("aucun point de livraison de drapeau trouvé")
	}
	for team, d := range deliver {
		s, ok := spawn[team]
		if !ok {
			t.Errorf("équipe %d : livraison sans apparition de drapeau", team)
			continue
		}
		if gap := distance(s, d); gap > 0.05 {
			t.Errorf("équipe %d : écart apparition/livraison = %.4f m, attendu < 0.05 m", team, gap)
		}
		if s.X != d.X && math.Abs(s.X-d.X) > 0.05 {
			t.Errorf("équipe %d : X divergent", team)
		}
	}
}

func distance(a, b Vec3) float64 {
	dx, dy, dz := a.X-b.X, a.Y-b.Y, a.Z-b.Z
	return math.Sqrt(dx*dx + dy*dy + dz*dz)
}

// TestDecodeRootRejectsTruncated — le lecteur doit échouer BRUYAMMENT, jamais
// tolérer un fichier tronqué (un décalage d'octet produirait des positions
// plausibles mais fausses).
func TestDecodeRootRejectsTruncated(t *testing.T) {
	buf := fixture(t, "breaker_ctf_breaker.mvar")
	if _, err := DecodeRoot(buf[:len(buf)-1]); err == nil {
		t.Error("DecodeRoot a accepté un fichier tronqué")
	}
	if _, err := DecodeRoot(append(append([]byte{}, buf...), 0x00)); err == nil {
		t.Error("DecodeRoot a accepté un octet résiduel")
	}
}
