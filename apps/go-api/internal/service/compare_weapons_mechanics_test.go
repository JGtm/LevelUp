package service

// compare_weapons_mechanics_test.go — LES MÉCANIQUES DE KILL NATIVES DANS LE PROFIL D'ARMES
// (correctif P0 du lot 3-ter, 2026-09-17 — revue adversariale du diff Go).
//
// # CE QUE CES DEUX TESTS ÉPINGLENT, ET POURQUOI LEUR ABSENCE COÛTAIT CHER
//
// Sur un titre à mécaniques natives, `fragdist.Build` RETRANCHE les frags de mécanique des
// classes d'arme — un assassinat est inscrit dans `weapon_kills` au compte de l'ARME TENUE —
// en supposant que `FragKillTypeCounts.Assassination/GroundPound/ShoulderBash` les rapportent
// par ailleurs. Ne pas les charger ne « perd » donc pas seulement la classe « Capacités
// spartanes » : « Mêlée » est sous-évaluée du nombre d'assassinats, et « Non attribué » gonfle
// d'autant. Le total boucle quand même — l'erreur est arithmétiquement cohérente, donc
// invisible à l'œil sur la page comme dans un test de somme.
//
// Fichier séparé de `compare_weapons_test.go` (534 lignes) : la frontière suit le sujet, et
// aucun des deux ne s'approche du plafond du dépôt.

import (
	"context"
	"math"
	"testing"

	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/port"
)

// fakeCompareWeaponKillsAvecMecaniques : le MÊME repo, plus la capability OPTIONNELLE
// `LoadKillMechanicsAggregated`.
//
// DEUX TYPES PLUTÔT QU'UN DRAPEAU, parce que ce qui est testé est justement une
// TYPE-ASSERTION : un champ « je réponds nil » ne dirait rien de la dégradation réelle, où le
// repo n'implémente tout simplement pas la méthode.
type fakeCompareWeaponKillsAvecMecaniques struct {
	fakeCompareWeaponKills
	mechs []port.KillMechanicsRow
}

func (f *fakeCompareWeaponKillsAvecMecaniques) LoadKillMechanicsAggregated(
	_ context.Context, _ port.WeaponKillFilters,
) ([]port.KillMechanicsRow, error) {
	return f.mechs, nil
}

// cwRowsAvecMecaniques : les mêmes trois armes, mais le BR75 porte 4 frags de MÉCANIQUE.
//
// C'EST LA FORME RÉELLE DE LA DONNÉE sur un titre à mécaniques natives : un assassinat y est
// inscrit au compte de l'arme tenue. `fragdist.Build` les retranche donc de la classe d'arme
// (10 - 4 = 6 frags d'épaule) et compte sur les compteurs natifs pour les servir ailleurs.
func cwRowsAvecMecaniques() []port.WeaponKillRow {
	rows := cwRows()
	rows[0].MechanicKills = 4
	return rows
}

// cwTitreAMecaniques enregistre un titre PORTEUR de la capability `native_kill_mechanics`
// comme registre par défaut, et le restaure à la fin du test.
//
// SANS CE SWAP, LES DEUX TESTS MESURERAIENT L'AUTRE BRANCHE EN CROYANT MESURER CELLE-CI : le
// registre runtime par défaut ne connaît qu'Halo Infinite hors boot (les autres titres
// viennent de leur TOML), et Infinite ne déclare PAS cette capability. Même motif de swap et
// restauration que `skill_v2_shadow_test.go`.
func cwTitreAMecaniques(t *testing.T) string {
	t.Helper()
	const slug = "titre_a_mecaniques"
	old := titlePkg.DefaultRegistry()
	reg := titlePkg.NewRegistry()
	reg.Register(&titlePkg.TitleDescriptor{
		Slug: slug, Name: "Titre de test", Status: titlePkg.StatusActive,
		Capabilities: []titlePkg.Capability{titlePkg.CapNativeKillMechanics},
	})
	titlePkg.SetDefaultRegistry(reg)
	t.Cleanup(func() { titlePkg.SetDefaultRegistry(old) })
	return slug
}

// cwScopesMecaniques : les deux joueurs sur un scope de 30 frags, 6 mêlées, 2 grenades.
func cwScopesMecaniques() *fakeCompareWeaponRepo {
	return &fakeCompareWeaponRepo{scopes: map[string]*domain.CompareWeaponScope{
		"xuid-a": cwScope(3, 30, 12, 6, 2),
		"xuid-b": cwScope(3, 30, 12, 6, 2),
	}}
}

// classeDe rend la classe publiée portant cette clé, ou nil.
func classeDe(side domain.CompareWeaponSide, classe string) *domain.CompareFragClass {
	for i := range side.FragClasses {
		if side.FragClasses[i].Class == classe {
			return &side.FragClasses[i]
		}
	}
	return nil
}

// TestCompareFragClasses_MecaniquesNatives_ChargeesEtComptees — LE TÉMOIN DU CORRECTIF.
//
// LA FIXTURE EST CALIBRÉE POUR QUE LE RETRAIT DE L'APPEL SE VOIE SUR TROIS CLASSES À LA FOIS.
// Avec les mécaniques chargées : Mêlée = 10, Capacités spartanes = 2, Non attribué = 2. Sans
// elles : Mêlée = 6, Capacités spartanes ABSENTE, Non attribué = 8. Les parts somment à 100
// dans les deux cas — c'est bien pourquoi une assertion de somme ne suffisait pas.
//
// Le contrat du compare ne publie que des CLASSES (pas de niveau 2) : le rôle `assassination`
// n'y apparaît donc pas en tant que tel. Il se lit dans le total de « Mêlée », qui ne vaut 10
// que si les 4 assassinats ont été chargés.
func TestCompareFragClasses_MecaniquesNatives_ChargeesEtComptees(t *testing.T) {
	slug := cwTitreAMecaniques(t)
	kills := &fakeCompareWeaponKillsAvecMecaniques{
		fakeCompareWeaponKills: fakeCompareWeaponKills{rows: map[string][]port.WeaponKillRow{
			"xuid-a": cwRowsAvecMecaniques(), "xuid-b": cwRowsAvecMecaniques(),
		}},
		mechs: []port.KillMechanicsRow{{XUID: "xuid-a", Assassinations: 4, GroundPound: 2}},
	}
	svc := NewCompareService(cwScopesMecaniques(), &mockStatsProvider{}, "xuid-a", slug).
		WithWeaponProfile(kills, &fakeCompareWeaponRange{}, nil)

	profile := svc.buildWeaponProfile(context.Background(), "xuid-b")
	if profile == nil {
		t.Fatal("profil nil")
	}
	side := profile.PlayerA

	melee := classeDe(side, domain.FragClassMelee)
	if melee == nil || melee.Kills != 10 {
		t.Fatalf("classe Mêlée = %+v, attendu 10 frags (6 mêlées + 4 assassinats) — "+
			"6 signifie que les mécaniques natives n'ont PAS été chargées", melee)
	}
	spartan := classeDe(side, domain.FragClassSpartanAbility)
	if spartan == nil || spartan.Kills != 2 {
		t.Fatalf("classe Capacités spartanes = %+v, attendu 2 frags (coup au sol) — "+
			"absente signifie que les mécaniques natives n'ont PAS été chargées", spartan)
	}
	residu := classeDe(side, domain.FragClassUnattributed)
	if residu == nil || residu.Kills != 2 {
		t.Fatalf("classe « Non attribué » = %+v, attendu 2 frags — 8 signifie que les frags "+
			"retranchés des classes d'arme ne sont servis nulle part", residu)
	}
	if got := sommeDesParts(side); math.Abs(got-100) > epsPart {
		t.Errorf("somme des parts = %v, attendu 100 ± %v", got, epsPart)
	}
}

// TestCompareFragClasses_SansCapabilityDeMecaniques_DegradationPropre : un repo qui n'expose
// PAS `LoadKillMechanicsAggregated` dégrade proprement — Mêlée non splittée, pas de classe
// « Capacités spartanes », et le résidu porte la différence.
//
// LE TITRE DÉCLARE POURTANT LA CAPABILITY : c'est le cas « le titre sait, le repo ne sait
// pas ». Ce témoin garantit que le correctif n'a pas transformé une dégradation en panne — le
// profil reste publié, seulement moins fin.
func TestCompareFragClasses_SansCapabilityDeMecaniques_DegradationPropre(t *testing.T) {
	slug := cwTitreAMecaniques(t)
	kills := &fakeCompareWeaponKills{rows: map[string][]port.WeaponKillRow{
		"xuid-a": cwRowsAvecMecaniques(), "xuid-b": cwRowsAvecMecaniques(),
	}}
	svc := NewCompareService(cwScopesMecaniques(), &mockStatsProvider{}, "xuid-a", slug).
		WithWeaponProfile(kills, &fakeCompareWeaponRange{}, nil)

	profile := svc.buildWeaponProfile(context.Background(), "xuid-b")
	if profile == nil {
		t.Fatal("profil nil : un repo sans la capability ne doit pas faire tomber la section")
	}
	side := profile.PlayerA
	if melee := classeDe(side, domain.FragClassMelee); melee == nil || melee.Kills != 6 {
		t.Fatalf("classe Mêlée = %+v, attendu 6 (le compteur natif seul)", melee)
	}
	if spartan := classeDe(side, domain.FragClassSpartanAbility); spartan != nil {
		t.Errorf("classe Capacités spartanes = %+v, attendue absente sans la capability", spartan)
	}
	if residu := classeDe(side, domain.FragClassUnattributed); residu == nil || residu.Kills != 8 {
		t.Fatalf("classe « Non attribué » = %+v, attendu 8 (les 4 assassinats et les 2 coups "+
			"au sol y retombent, faute d'être servis ailleurs)", residu)
	}
	if got := sommeDesParts(side); math.Abs(got-100) > epsPart {
		t.Errorf("somme des parts = %v, attendu 100 ± %v", got, epsPart)
	}
}
