package mapvar

import (
	"math"
	"os"
	"path/filepath"
	"testing"
)

// fixtureDir : les .mvar de référence vivent sous .ai/V7.5/dumps/mapvar/ (artefacts de
// rétro-ingénierie déjà présents en dépôt). On ne les duplique pas dans testdata/ :
// ce sont des actifs propriétaires 343/Microsoft.
func fixture(t *testing.T, name string) []byte {
	t.Helper()
	path := filepath.Join("..", "..", "..", "..", "..", "..", ".ai", "V7.5", "dumps", "mapvar", name)
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

// TestShapeAnchorCliffhangerStronghold — ANCRE EXTERNE de la conversion.
// L'objet 178 de cliffhanger_map.mvar porte les valeurs brutes citées dans
// l'état de l'art (§ contrat de données) : famille 3, s5=445644, s6=393216,
// s7=262144, s8=0. Sous la lecture retenue (tailles pleines), elles donnent
// une boîte de 3,400 × 3,000 m de demi-extents, 4 m au-dessus du centre.
// Si quelqu'un rebranche la division par 2 au mauvais endroit, ce test tombe.
func TestShapeAnchorCliffhangerStronghold(t *testing.T) {
	v, err := Parse(fixture(t, "cliffhanger_map.mvar"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(v.Objects) <= 178 {
		t.Fatalf("objets = %d, l'ancre attend au moins 179", len(v.Objects))
	}
	s := v.Objects[178].Shape()
	if s == nil {
		t.Fatal("objet 178 sans forme, attendu une boîte")
	}
	if s.Family != ShapeBox {
		t.Errorf("famille = %q, attendu %q", s.Family, ShapeBox)
	}
	if s.Raw.S5 != 445644 || s.Raw.S6 != 393216 || s.Raw.S7 != 262144 || s.Raw.S8 != 0 {
		t.Fatalf("brut = %+v, attendu s5=445644 s6=393216 s7=262144 s8=0", s.Raw)
	}
	near(t, "half_x", *s.HalfX, 3.400, 1e-3)
	near(t, "half_y", *s.HalfY, 3.000, 1e-3)
	near(t, "up_z", s.UpZ, 4.000, 1e-3)
	near(t, "down_z", s.DownZ, 0.000, 1e-3)
	if s.Radius != nil {
		t.Error("une boîte ne publie pas de rayon")
	}
	// L'orientation est obligatoire et non nulle : une zone tournée mal orientée
	// déclare « dedans » 31 % de positions qui sont dehors.
	near(t, "forward.x", s.Forward.X, 0.99999, 1e-4)
	near(t, "forward.y", s.Forward.Y, -0.00397, 1e-4)
}

// TestShapeFullSizeReadingBeatsHalfExtent — LE TÉMOIN qui a tranché la lecture.
// Sur cliffhanger_map.mvar, l'objet 185 est un cylindre (s5=334233) et l'objet
// 188 une boîte (s5=668441), même carte et même rôle. Sous la lecture retenue
// (« s5 est une taille pleine pour la boîte, un rayon pour le cylindre »), la
// demi-largeur de la boîte et le rayon du cylindre coïncident au dixième de
// millimètre. Sous la lecture rejetée (demi-extents partout), l'écart serait
// d'un facteur 2. Ce test encode donc la mesure, pas la conclusion.
func TestShapeFullSizeReadingBeatsHalfExtent(t *testing.T) {
	v, err := Parse(fixture(t, "cliffhanger_map.mvar"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(v.Objects) <= 188 {
		t.Fatalf("objets = %d, le témoin attend au moins 189", len(v.Objects))
	}
	cyl := v.Objects[185].Shape()
	box := v.Objects[188].Shape()
	if cyl == nil || box == nil {
		t.Fatal("le couple témoin (185, 188) doit porter deux formes")
	}
	if cyl.Family != ShapeCylinder || box.Family != ShapeBox {
		t.Fatalf("familles = %q / %q, attendu cylinder / box", cyl.Family, box.Family)
	}
	if cyl.HalfX != nil || cyl.HalfY != nil {
		t.Error("un cylindre ne publie pas de demi-extents cartésiens")
	}
	gap := *box.HalfX - *cyl.Radius
	if gap < 0 {
		gap = -gap
	}
	if gap > 0.005 {
		t.Errorf("demi-largeur de boîte %.5f vs rayon %.5f : écart %.5f m > 5 mm — "+
			"la lecture « tailles pleines » ne tient plus", *box.HalfX, *cyl.Radius, gap)
	}
}

// TestShapePresenceFollowsSurfaceRule — la règle de couverture, sur pièces.
// 100 % des objectifs SURFACIQUES portent une forme, 0 % des PONCTUELS. Ce
// n'est pas un trou de couverture, c'est la structure : un point d'apparition
// de drapeau EST un point. Un consommateur ne doit jamais inventer un rayon.
func TestShapePresenceFollowsSurfaceRule(t *testing.T) {
	surfacic := map[Role]bool{
		RoleStrongholdZone: true, RoleExtractionZone: true, RoleStockpileNavpoint: true,
	}
	punctual := map[Role]bool{
		RoleFlagSpawn: true, RoleStockpileSocket: true,
	}
	for _, name := range []string{"cliffhanger_map.mvar", "cliffhanger_ridgeline.mvar"} {
		v, err := Parse(fixture(t, name))
		if err != nil {
			t.Fatalf("Parse %s: %v", name, err)
		}
		seen := 0
		for _, ob := range v.Objectives() {
			switch {
			case surfacic[ob.Role]:
				seen++
				if ob.Shape == nil {
					t.Errorf("%s objet %d rôle %s : objectif surfacique SANS forme",
						name, ob.ObjectIdx, ob.Role)
				}
			case punctual[ob.Role]:
				seen++
				if ob.Shape != nil {
					t.Errorf("%s objet %d rôle %s : objectif ponctuel AVEC une forme",
						name, ob.ObjectIdx, ob.Role)
				}
			}
		}
		if seen == 0 {
			t.Errorf("%s : aucun objectif classé, la règle n'a rien testé", name)
		}
	}
}

// TestShapeDerivesFromRaw — le brut et le dérivé ne peuvent pas diverger.
// Le contrat conserve les deux ; si l'un cessait de décrire l'autre, un
// consommateur qui recalcule depuis le brut obtiendrait autre chose que ce que
// le champ dérivé annonce.
func TestShapeDerivesFromRaw(t *testing.T) {
	v, err := Parse(fixture(t, "cliffhanger_map.mvar"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	n := 0
	for _, o := range v.Objects {
		s := o.Shape()
		if s == nil {
			continue
		}
		n++
		near(t, "up_z", s.UpZ, float64(s.Raw.S7)/65536.0, 1e-9)
		near(t, "down_z", s.DownZ, float64(s.Raw.S8)/65536.0, 1e-9)
		switch s.Family {
		case ShapeBox:
			near(t, "half_x", *s.HalfX, float64(s.Raw.S5)/65536.0/2, 1e-9)
			near(t, "half_y", *s.HalfY, float64(s.Raw.S6)/65536.0/2, 1e-9)
		case ShapeCylinder:
			near(t, "radius", *s.Radius, float64(s.Raw.S5)/65536.0, 1e-9)
		}
	}
	if n == 0 {
		t.Fatal("aucune forme lue sur cliffhanger_map.mvar")
	}
}

func near(t *testing.T, what string, got, want, tol float64) {
	t.Helper()
	if math.Abs(got-want) > tol {
		t.Errorf("%s = %.6f, attendu %.6f (tolérance %g)", what, got, want, tol)
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

// TestHillRoleParLeHashDeRole — la colline de KOTH se pose par le hash de ROLE, sauf quand
// un AUTRE mode revendique l'objet par son filtre d'activation. Les quatre cas mesures le
// 2026-08-20 sont ici, et ils corrigent la regle « paire obligatoire » du 2026-08-19 :
//
//	role seul, sans aucun filtre  -> colline  (Forbidden 87c03bfd, Illusion 9e821f5e : 5
//	                                volumes type 1818458590, formes toutes differentes,
//	                                aux positions des 5 marqueurs de KOTH)
//	role + filtre d'un autre mode -> PAS une colline (le decoy documente : 2 cylindres
//	                                `minigame_include` par carte de developpeur)
//	role + filtre etranger + filtre KOTH -> colline (Empyrean : ses volumes sont declares
//	                                pour Bastion ET pour KOTH)
//	filtre KOTH seul              -> PAS une colline (le marqueur ponctuel)
//
// Mutation qui doit le faire rougir : rendre `role && kothInclude` (l'ancienne regle) —
// Forbidden et Illusion reperdent leurs 5 collines.
func TestHillRoleParLeHashDeRole(t *testing.T) {
	cas := []struct {
		nom    string
		labels []int32
		want   []Role
	}{
		{"paire complete", []int32{LabelHashHillInclude, LabelHashHillRole}, []Role{RoleHill}},
		{"paire dans l'autre ordre", []int32{LabelHashHillRole, LabelHashHillInclude}, []Role{RoleHill}},
		{"role seul sans filtre (Forbidden/Illusion)", []int32{LabelHashHillRole}, []Role{RoleHill}},
		{"role + filtre etranger (decoy minigame)", []int32{LabelHash("minigame_include"), LabelHashHillRole}, nil},
		{"role + filtre etranger + filtre KOTH (Empyrean)",
			[]int32{LabelHash("elimination_include"), LabelHashHillInclude, LabelHashHillRole}, []Role{RoleHill}},
		{"filtre seul (marqueur)", []int32{LabelHashHillInclude, LabelHashHillMarker}, nil},
		{"rien", nil, nil},
		// Un volume declare pour Bastion ET pour KOTH porte LES DEUX roles : c'est le cas
		// d'Empyrean, et l'ancienne regle « premier role nomme gagnant » perdait la colline.
		{"volume multi-mode (Bastion + KOTH)",
			[]int32{LabelHash("strongholds_zone"), LabelHashHillInclude, LabelHashHillRole},
			[]Role{RoleStrongholdZone, RoleHill}},
	}
	for _, c := range cas {
		roles, _, unknown := classify(c.labels)
		if len(roles) != len(c.want) {
			t.Errorf("%s : roles %v, attendu %v", c.nom, roles, c.want)
			continue
		}
		for i := range roles {
			if roles[i] != c.want[i] {
				t.Errorf("%s : roles %v, attendu %v", c.nom, roles, c.want)
				break
			}
		}
		// Les hashs de KOTH restent NON RESOLUS : ils n'ont pas de nom, on ne l'invente pas.
		for _, h := range c.labels {
			if h == LabelHashHillRole || h == LabelHashHillInclude {
				trouve := false
				for _, u := range unknown {
					trouve = trouve || u == h
				}
				if !trouve {
					t.Errorf("%s : le hash %d devrait rester compte parmi les non resolus", c.nom, h)
				}
			}
		}
	}
	if LabelName(LabelHashHillRole) != "" || LabelName(LabelHashHillInclude) != "" {
		t.Error("les hashs de KOTH ne doivent pas avoir de nom dans labelNames tant qu'il n'est pas retrouve")
	}
}

// TestObjectifMultiRolePublieChaqueRole — un objet declare pour deux modes produit DEUX
// objectifs, un par role, qui partagent position, forme et object_index. C'est la forme
// mesuree sur Empyrean : ses trois zones de Bastion sont AUSSI ses trois premieres collines
// (le catalogue livre n'en publiait qu'une seule, la quatrieme).
//
// Mutation qui doit le faire rougir : ne publier que le premier role.
func TestObjectifMultiRolePublieChaqueRole(t *testing.T) {
	v := &Variant{Objects: []Object{{
		Index: 42, TypeID: -1476457415, TeamIndex: TeamUnset,
		Pos:    Vec3{X: 0.13, Y: -30.37, Z: 139.05},
		Labels: []int32{LabelHash("strongholds_include"), LabelHash("strongholds_zone"), LabelHashHillInclude, LabelHashHillRole},
	}}}
	objs := v.Objectives()
	if len(objs) != 2 {
		t.Fatalf("objectifs = %d, attendu 2 (zone de Bastion + colline)", len(objs))
	}
	if objs[0].Role != RoleStrongholdZone || objs[1].Role != RoleHill {
		t.Fatalf("roles = %q puis %q, attendu %q puis %q",
			objs[0].Role, objs[1].Role, RoleStrongholdZone, RoleHill)
	}
	for _, o := range objs {
		if o.ObjectIdx != 42 || o.Pos != (Vec3{X: 0.13, Y: -30.37, Z: 139.05}) {
			t.Errorf("role %q : l'objectif a perdu l'identite de son objet (idx %d, pos %v)",
				o.Role, o.ObjectIdx, o.Pos)
		}
	}
	// Un meme role declare deux fois dans le fichier ne se publie qu'UNE fois.
	dbl := &Variant{Objects: []Object{{
		Index:  7,
		Labels: []int32{LabelHash("extraction_zone"), LabelHash("extraction_zone")},
	}}}
	if n := len(dbl.Objectives()); n != 1 {
		t.Errorf("label repete : %d objectifs, attendu 1", n)
	}
}

// TestCollinesDeCliffhanger — la fixture versionnee porte 5 collines de KOTH (relevees le
// 2026-08-19 sur cliffhanger_map.mvar : 2 cylindres r 4,0 / 3,7 et 3 boites), toutes NEUTRES
// et toutes avec forme ; les 5 marqueurs ponctuels qui les accompagnent n'ont pas de role.
func TestCollinesDeCliffhanger(t *testing.T) {
	v, err := Parse(fixture(t, "cliffhanger_map.mvar"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	var hills []Objective
	for _, o := range v.Objectives() {
		if o.Role == RoleHill {
			hills = append(hills, o)
		}
	}
	if len(hills) != 5 {
		t.Fatalf("collines = %d, attendu 5", len(hills))
	}
	cyl := 0
	for _, h := range hills {
		if h.TeamIndex != TeamUnset {
			t.Errorf("colline objet %d : team_index %d, attendu neutre", h.ObjectIdx, h.TeamIndex)
		}
		if h.Shape == nil {
			t.Errorf("colline objet %d : sans forme", h.ObjectIdx)
			continue
		}
		if h.Shape.Family == ShapeCylinder {
			cyl++
		}
		if len(h.Labels) != 0 || len(h.Unresolved) != 2 {
			t.Errorf("colline objet %d : labels %v, non resolus %v — attendu 0 nom et 2 hashs",
				h.ObjectIdx, h.Labels, h.Unresolved)
		}
	}
	if cyl != 2 {
		t.Errorf("cylindres = %d, attendu 2 (r 4,0 et r 3,7)", cyl)
	}
}
