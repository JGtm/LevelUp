//go:build integration

// weapon_resolver_test.go — vérifie le PASSAGE PRINCIPAL P4 (resolveWeaponMeta) :
//   - SOURCE UNIQUE (V72-06) : le nom vient de weapon_name_labels, keyé par weapon_key
//     (weapon_id → weapon_ids → weapon_key → noms par langue). BR75 = « BR75 » (parité).
//   - DIMENSIONS : role/family/faction/weapon_key viennent du registre (plus de nom).
//   - sentinel grenade (0) : nom weapon_label_names (pas de weapon_key), role vide.
//   - id inconnu : label vide (filtré en aval).
//   - fallback silencieux quand le registre/la source de noms sont absents.
//   - UNE SEULE RÈGLE DE REPLI (plan i18n 8.7) : le SQL rend les noms par langue, le nom curé prime
//     langue par langue, la langue demandée se choisit par Text.Pick (langue, puis `en`) ; plus de
//     repli croisé fr/en.

package duckdb

import (
	"context"
	"strconv"
	"testing"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/games/weapons"
	"levelup/go-api/internal/i18n"
	"levelup/go-api/internal/port"
)

func resolverTestMeta(t *testing.T, withRegistry bool) *DB {
	t.Helper()
	meta := openMemDB(t)
	ctx := context.Background()
	if _, err := meta.Exec(ctx, `CREATE TABLE weapon_labels (weapon_id UBIGINT PRIMARY KEY)`); err != nil {
		t.Fatalf("create weapon_labels: %v", err)
	}
	createWeaponNameTables(t, meta)
	brID := uint64(0x2b1824d542c9679f) // BR75 filmshell
	if _, err := meta.Exec(ctx, "INSERT INTO weapon_labels VALUES ("+strconv.FormatUint(brID, 10)+"), (0)"); err != nil {
		t.Fatalf("seed weapon_labels: %v", err)
	}
	putWeaponLabelNames(t, meta, brID, frEn("BR75", "BR75"))
	putWeaponLabelNames(t, meta, 0, frEn("Grenade", "Grenade"))
	if withRegistry {
		if err := weapons.ApplyRegistry(meta.SQLDb()); err != nil {
			t.Fatalf("ApplyRegistry: %v", err)
		}
		// V72-06 : SOURCE UNIQUE des noms keyée par weapon_key. Le resolver lit
		// weapon_name_labels (weapon_id → weapon_ids → weapon_key → noms par langue). On seede
		// les 3 clés exercées par les tests ; les autres ids retombent sur weapon_label_names
		// (sentinelles) ou "" (inconnus).
		putCuratedWeaponNames(t, meta, "halo_infinite", "hinf_br75", frEn("BR75", "BR75"))
		putCuratedWeaponNames(t, meta, "halo_5", "h5_light_rifle", frEn("Fusil léger", "Light Rifle"))
		putCuratedWeaponNames(t, meta, "halo_5", "h5_frag_grenade", frEn("Grenade à fragmentation", "Frag Grenade"))
	}
	return meta
}

func TestResolveWeaponMeta_ParityAndDims(t *testing.T) {
	meta := resolverTestMeta(t, true)
	brID := int64(0x2b1824d542c9679f)
	res := resolveWeaponMeta(context.Background(), meta, "halo_infinite", []int64{brID, 0, 999999})

	br := res[brID]
	if br.label[i18n.LocaleFR] != "BR75" { // PARITÉ : nom curé, surtout PAS "Fusil de combat"
		t.Errorf("BR75 label = %q, want \"BR75\" (parité)", br.label[i18n.LocaleFR])
	}
	if br.role != "precision" || br.family != "battle_rifle" || br.faction != "human" || br.weaponKey != "hinf_br75" {
		t.Errorf("BR75 dims = %+v, want precision/battle_rifle/human/hinf_br75", br)
	}
	if s := res[0]; s.label[i18n.LocaleFR] != "Grenade" || s.role != "" {
		t.Errorf("sentinel 0 = %+v, want label=Grenade role=''", s)
	}
	if u := res[999999]; !u.label.Empty() {
		t.Errorf("id inconnu label = %v, want vide", u.label)
	}
}

// TestResolveWeaponMeta_WeaponKeyNameSourceH5 fige la SOURCE UNIQUE keyée par weapon_key
// (V72-06, le nom curé prime l'autre source dans une même langue) :
//
//	(a) parité Infinite : BR75 résout le MÊME nom qu'avant (« BR75 ») via
//	    weapon_name_labels ;
//	(b) H5, aucune ligne weapon_label_names : h5_light_rifle (id 2511447508) récupère le nom
//	    depuis weapon_name_labels (weapon_key h5_light_rifle) au lieu de rester vide ;
//	(c) H5, nom EN-ONLY POURRI : une arme dont weapon_label_names porte un nom anglais brut
//	    (« FRAG GRENADE », casse API) et AUCUN nom français doit résoudre le nom de
//	    weapon_name_labels (« Grenade à fragmentation »), PAS le nom brut. C'est le bug d'origine :
//	    l'EN brut fuyait sur le Match view H5. La source keyée par weapon_key le prime.
func TestResolveWeaponMeta_WeaponKeyNameSourceH5(t *testing.T) {
	meta := resolverTestMeta(t, true)
	ctx := context.Background()

	// (a) Parité Infinite : BR75 (weapon_name_labels hinf_br75) → byte-inchangé.
	brID := int64(0x2b1824d542c9679f)
	if br := resolveWeaponMeta(ctx, meta, "halo_infinite", []int64{brID})[brID]; br.label[i18n.LocaleFR] != "BR75" {
		t.Errorf("BR75 label = %q, want \"BR75\" (parité, source keyée weapon_key)", br.label)
	}

	// (b) H5, aucune ligne weapon_label_names : h5_light_rifle (id 2511447508) → nom résolu
	// via weapon_ids → weapon_key h5_light_rifle → weapon_name_labels (« Fusil léger »).
	const lightRifleID = int64(2511447508)
	if lr := resolveWeaponMeta(ctx, meta, "halo_5", []int64{lightRifleID})[lightRifleID]; lr.label[i18n.LocaleFR] != "Fusil léger" {
		t.Errorf("h5 light rifle label = %q, want \"Fusil léger\" (source weapon_key)", lr.label)
	}

	// (c) H5, nom EN-only POURRI : on seede weapon_labels + un nom anglais brut seul (« FRAG
	// GRENADE », comme en prod H5) pour l'id frag grenade (stock_id 4106030681 →
	// h5_frag_grenade). La source weapon_name_labels (« Grenade à fragmentation ») doit primer.
	const fragGrenadeID = int64(4106030681)
	if _, err := meta.Exec(ctx, "INSERT INTO weapon_labels VALUES ("+strconv.FormatInt(fragGrenadeID, 10)+")"); err != nil {
		t.Fatalf("seed h5 frag grenade: %v", err)
	}
	putWeaponLabelNames(t, meta, uint64(fragGrenadeID), i18n.Text{i18n.LocaleEN: "FRAG GRENADE"})
	fg := resolveWeaponMeta(ctx, meta, "halo_5", []int64{fragGrenadeID})[fragGrenadeID]
	if fg.label[i18n.LocaleFR] != "Grenade à fragmentation" {
		t.Errorf("h5 frag grenade label = %q, want \"Grenade à fragmentation\" (weapon_name_labels prime l'EN brut)", fg.label)
	}
	if fg.label[i18n.LocaleEN] != "Frag Grenade" {
		t.Errorf("h5 frag grenade en = %q, want \"Frag Grenade\" (le nom curé prime aussi en anglais)", fg.label[i18n.LocaleEN])
	}
	if fg.canonicalName != "FRAG GRENADE" {
		t.Errorf("canonicalName = %q, want le nom anglais de weapon_label_names (clé d'image)", fg.canonicalName)
	}
}

// TestResolveWeaponMeta_NomCurePrimeLangueParLangue : le nom curé l'emporte sur le nom par
// identifiant DANS LA MÊME LANGUE seulement ; une langue que seule l'autre source porte reste
// servie, sans repli croisé.
func TestResolveWeaponMeta_NomCurePrimeLangueParLangue(t *testing.T) {
	meta := resolverTestMeta(t, true)
	ctx := context.Background()
	const ghostID = int64(3010146366)
	if _, err := meta.Exec(ctx, "INSERT INTO weapon_labels VALUES (3010146366)"); err != nil {
		t.Fatal(err)
	}
	putCuratedWeaponNames(t, meta, "halo_5", "h5_vehicle_ghost", i18n.Text{i18n.LocaleEN: "Ghost"})
	putWeaponLabelNames(t, meta, uint64(ghostID), frEn("Fantôme (API)", "ghost (API)"))
	res := resolveWeaponMeta(ctx, meta, "halo_5", []int64{ghostID})[ghostID]
	if res.label[i18n.LocaleEN] != "Ghost" {
		t.Errorf("en = %q, want \"Ghost\" (nom curé)", res.label[i18n.LocaleEN])
	}
	if res.label[i18n.LocaleFR] != "Fantôme (API)" {
		t.Errorf("fr = %q, want \"Fantôme (API)\" (langue que seul le nom par identifiant porte)", res.label[i18n.LocaleFR])
	}
}

// TestResolveWeaponMeta_LectureFrEnEtRepli : la langue demandée se choisit par Pick (langue, puis
// `en`) ; le repli croisé fr/en du SQL a disparu (plan i18n 8.7). Changement de sens par rapport
// à l'ancien test « EN vide => nom FR » : une arme sans nom anglais n'a plus de nom en anglais.
func TestResolveWeaponMeta_LectureFrEnEtRepli(t *testing.T) {
	meta := resolverTestMeta(t, true)
	ctx := context.Background()
	const ghostID, warthogID = int64(3010146366), int64(4028516791)
	putCuratedWeaponNames(t, meta, "halo_5", "h5_vehicle_ghost", i18n.Text{i18n.LocaleFR: "Fantôme"})
	putCuratedWeaponNames(t, meta, "halo_5", "h5_vehicle_warthog", i18n.Text{i18n.LocaleEN: "Warthog"})
	res := resolveWeaponMeta(ctx, meta, "halo_5", []int64{ghostID, warthogID})

	ghost, warthog := res[ghostID], res[warthogID]
	if got := ghost.displayLabel("fr"); got != "Fantôme" {
		t.Errorf("ghost fr = %q, want Fantôme", got)
	}
	if got := ghost.displayLabel("en"); got != "" {
		t.Errorf("ghost en = %q, want vide (plus de repli croisé vers le français)", got)
	}
	if got := warthog.displayLabel("fr"); got != "Warthog" {
		t.Errorf("warthog fr = %q, want Warthog (repli en)", got)
	}
	if got := warthog.displayLabel("en"); got != "Warthog" {
		t.Errorf("warthog en = %q, want Warthog", got)
	}
	// Langue absente de la liste : Text.Pick replie sur `en` (displayLabel passe par Parse, qui
	// donne la langue par défaut à une langue inconnue).
	if got := warthog.label.Pick(i18n.Locale("zz")); got != "Warthog" {
		t.Errorf("langue inconnue : Pick = %q, want Warthog (repli en)", got)
	}
}

// TestResolveWeaponMeta_LangueInconnueEnBaseIgnoree : une ligne dont le code n'est pas une langue
// de config/locales.toml n'est pas servie, et le dit dans le journal.
func TestResolveWeaponMeta_LangueInconnueEnBaseIgnoree(t *testing.T) {
	meta := resolverTestMeta(t, true)
	buf := captureSlog(t)
	const ghostID = int64(3010146366)
	if _, err := meta.Exec(context.Background(), "INSERT INTO weapon_labels VALUES (3010146366)"); err != nil {
		t.Fatal(err)
	}
	putWeaponLabelNames(t, meta, uint64(ghostID), i18n.Text{i18n.LocaleEN: "Ghost", i18n.Locale("zz"): "Zzz"})
	res := resolveWeaponMeta(context.Background(), meta, "halo_5", []int64{ghostID})[ghostID]
	if len(res.label) != 1 || res.label[i18n.LocaleEN] != "Ghost" {
		t.Errorf("label = %v, want la seule langue connue", res.label)
	}
	if !hasWarnContaining(t, buf, "weapon_label_names", "langue") {
		t.Errorf("WARN attendu sur la langue inconnue ; logs=%s", buf.String())
	}
}

// TestResolveWeaponMeta_NomsParLangueBothLocales : le resolver fournit les noms de toutes les
// langues dans la MÊME passe (h5_vehicle_warthog porte en et fr dans weapon_name_labels, comme
// weapon_names.toml en pratique) → chaque langue résout au nom natif.
func TestResolveWeaponMeta_NomsParLangueBothLocales(t *testing.T) {
	meta := resolverTestMeta(t, true)
	ctx := context.Background()
	putCuratedWeaponNames(t, meta, "halo_5", "h5_vehicle_warthog", frEn("Chariot de guerre", "Warthog"))
	const warthogID = int64(4028516791)
	res := resolveWeaponMeta(ctx, meta, "halo_5", []int64{warthogID})[warthogID]
	if res.label[i18n.LocaleFR] != "Chariot de guerre" {
		t.Errorf("fr = %q, want \"Chariot de guerre\"", res.label[i18n.LocaleFR])
	}
	if res.label[i18n.LocaleEN] != "Warthog" {
		t.Errorf("en = %q, want \"Warthog\"", res.label[i18n.LocaleEN])
	}
}

func TestResolveWeaponMeta_FallbackWhenNoRegistry(t *testing.T) {
	meta := resolverTestMeta(t, false) // registre absent
	brID := int64(0x2b1824d542c9679f)
	res := resolveWeaponMeta(context.Background(), meta, "halo_infinite", []int64{brID, 0})

	if br := res[brID]; br.label[i18n.LocaleFR] != "BR75" || br.role != "" {
		t.Errorf("BR75 sans registre = %+v, want label=BR75 role='' (parité, dims vides)", br)
	}
	// Les deux langues sont aussi servies par le fallback weapon_label_names seul (résolveur
	// cohérent quel que soit le chemin interne emprunté).
	if br := res[brID]; br.label[i18n.LocaleEN] != "BR75" {
		t.Errorf("BR75 sans registre : en = %q, want \"BR75\"", br.label[i18n.LocaleEN])
	}
	if s := res[0]; s.label[i18n.LocaleFR] != "Grenade" {
		t.Errorf("sentinel sans registre = %+v, want label=Grenade", s)
	}
	if s := res[0]; s.label[i18n.LocaleEN] != "Grenade" {
		t.Errorf("sentinel sans registre : en = %q, want \"Grenade\"", s.label[i18n.LocaleEN])
	}
}

// TestResolveWeaponMeta_H5VehiclesDistinctPerEngine fige le SOCLE de la ventilation par
// ENGIN (V73-3.2) et documente pourquoi elle ne peut pas passer par le rôle :
//
//	(a) les stock_ids RÉELS des véhicules H5 (relevés en base le 2026-08-02 :
//	    Warthog 4028516791, Ghost 3010146366) résolvent chacun un weapon_key DISTINCT —
//	    ils ne s'écrasent PAS sur le sentinel véhicule (id 2), qui ne porte aucun kill H5 ;
//	(b) leur class vaut « vehicle » pour les deux, tout comme role et family : ventiler le
//	    niveau 2 par rôle donnerait donc un arc unique sans information. Seul weapon_key
//	    distingue un Warthog d'un Ghost — d'où domain.IsPerWeaponFragClass.
func TestResolveWeaponMeta_H5VehiclesDistinctPerEngine(t *testing.T) {
	meta := resolverTestMeta(t, true)
	ctx := context.Background()
	putCuratedWeaponNames(t, meta, "halo_5", "h5_vehicle_warthog", frEn("Warthog", "Warthog"))
	putCuratedWeaponNames(t, meta, "halo_5", "h5_vehicle_ghost", frEn("Ghost", "Ghost"))
	const warthogID, ghostID = int64(4028516791), int64(3010146366)
	res := resolveWeaponMeta(ctx, meta, "halo_5", []int64{warthogID, ghostID})

	warthog, ghost := res[warthogID], res[ghostID]
	if warthog.weaponKey != "h5_vehicle_warthog" || ghost.weaponKey != "h5_vehicle_ghost" {
		t.Fatalf("weapon_key par engin = %q / %q, want h5_vehicle_warthog / h5_vehicle_ghost",
			warthog.weaponKey, ghost.weaponKey)
	}
	if warthog.weaponKey == ghost.weaponKey {
		t.Error("deux engins distincts partagent une clé : le sous-niveau ne pourrait pas les séparer")
	}
	if warthog.label[i18n.LocaleFR] != "Warthog" || ghost.label[i18n.LocaleFR] != "Ghost" {
		t.Errorf("libellés registre = %q / %q, want Warthog / Ghost (source weapon_names.toml)",
			warthog.label[i18n.LocaleFR], ghost.label[i18n.LocaleFR])
	}
	// (b) class/role/family confondus → le rôle ne distingue aucun engin.
	for name, m := range map[string]weaponResolved{"warthog": warthog, "ghost": ghost} {
		if m.class != "vehicle" {
			t.Errorf("%s class = %q, want vehicle", name, m.class)
		}
		if m.role != m.class {
			t.Errorf("%s : role=%q class=%q — le test suppose role == class (socle de la ventilation par engin)",
				name, m.role, m.class)
		}
	}
}

// TestMatchViewWeaponLabels_FollowRequestLocale (2026-09-17) : les trois lecteurs d'armes de
// la Match view qui ne publient qu'UN nom par arme (kills par arme du viewer, arme favorite
// du scoreboard, sources de dégât du film) choisissent ce nom dans la LOCALE DE REQUÊTE.
// Avant : label FR-first quelle que soit la locale — « Tourelle LMG du Falcon » et
// « Apparition » sortaient sur la capture README anglaise de la vue de match.
func TestMatchViewWeaponLabels_FollowRequestLocale(t *testing.T) {
	meta := resolverTestMeta(t, true)
	putCuratedWeaponNames(t, meta, "halo_5", "h5_vehicle_warthog", frEn("Chariot de guerre", "Warthog"))
	repo := NewMatchViewRepo(&PlayerDB{Metadata: meta, TitleSlug: "halo_5"}, "xuid-test")
	const warthogID = int64(4028516791)

	cas := []struct {
		locale string
		want   string
	}{
		{"fr", "Chariot de guerre"},
		{"en", "Warthog"},
		{"en-US", "Warthog"},
		{"", "Chariot de guerre"}, // locale absente = défaut ctxkeys ("fr")
	}
	for _, c := range cas {
		ctx := context.Background()
		if c.locale != "" {
			ctx = ctxkeys.WithLocale(ctx, c.locale)
		}
		if got := repo.lookupWeaponLabels(ctx, []int64{warthogID})[warthogID]; got != c.want {
			t.Errorf("locale %q : lookupWeaponLabels = %q, want %q", c.locale, got, c.want)
		}
		if got := repo.lookupWeaponMeta(ctx, []int64{warthogID})[warthogID].label; got != c.want {
			t.Errorf("locale %q : lookupWeaponMeta.label = %q, want %q", c.locale, got, c.want)
		}
		// Lecteur keyé par weapon_key (sources de dégât du film) : même contrat.
		if got := resolveWeaponKeyDimensions(ctx, meta, "halo_5", []string{"h5_vehicle_warthog"})["h5_vehicle_warthog"].displayLabel(ctxkeys.Locale(ctx)); got != c.want {
			t.Errorf("locale %q : weaponKeyResolved.displayLabel = %q, want %q", c.locale, got, c.want)
		}
		// « Précision par arme » (Synthèse / Session / Escouade) : même défaut, même correction.
		rows := []port.WeaponAccuracyRow{{WeaponID: warthogID, ShotsFired: 10, ShotsLanded: 5}}
		NewWeaponAccuracyRepo(repo.pdb).attachWeaponLabels(ctx, "halo_5", rows)
		if rows[0].Label != c.want {
			t.Errorf("locale %q : attachWeaponLabels = %q, want %q", c.locale, rows[0].Label, c.want)
		}
	}
}

// TestResolveWeaponKeyResolvers_NomsParLangueEtRepli : les deux lecteurs keyés par weapon_key
// (resolveWeaponKeyLabelsAny, resolveWeaponKeyDimensions) rendent les noms curés par langue, sans
// repli croisé : la langue demandée, sinon `en` (Text.Pick), et vide quand `en` manque.
func TestResolveWeaponKeyResolvers_NomsParLangueEtRepli(t *testing.T) {
	meta := resolverTestMeta(t, true)
	ctx := context.Background()
	putCuratedWeaponNames(t, meta, "halo_5", "h5_vehicle_warthog", i18n.Text{i18n.LocaleEN: "Warthog"})
	putCuratedWeaponNames(t, meta, "halo_5", "h5_vehicle_ghost", i18n.Text{i18n.LocaleFR: "Fantôme"})
	keys := []string{"h5_vehicle_warthog", "h5_vehicle_ghost"}

	byAny := resolveWeaponKeyLabelsAny(ctx, meta, "halo_5", keys)
	byDims := resolveWeaponKeyDimensions(ctx, meta, "halo_5", keys)
	for _, c := range []struct{ key, fr, en, inconnue string }{
		{"h5_vehicle_warthog", "Warthog", "Warthog", "Warthog"},
		{"h5_vehicle_ghost", "Fantôme", "", ""},
	} {
		for name, text := range map[string]i18n.Text{"any": byAny[c.key].label, "dims": byDims[c.key].label} {
			if got := text.Pick(i18n.LocaleFR); got != c.fr {
				t.Errorf("%s %s fr = %q, want %q", name, c.key, got, c.fr)
			}
			if got := text.Pick(i18n.LocaleEN); got != c.en {
				t.Errorf("%s %s en = %q, want %q", name, c.key, got, c.en)
			}
			if got := text.Pick(i18n.Locale("zz")); got != c.inconnue {
				t.Errorf("%s %s langue inconnue = %q, want %q", name, c.key, got, c.inconnue)
			}
		}
	}
	if byDims["h5_vehicle_warthog"].class != "vehicle" || byDims["h5_vehicle_warthog"].numericID == 0 {
		t.Errorf("dimensions perdues : %+v", byDims["h5_vehicle_warthog"])
	}
}
