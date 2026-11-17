package halo_infinite

import (
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/weapons"

	"github.com/JGtm/halo-infinite-film-decoder/killicon"
)

// tagOfKey rend une source de dégât dont le pont nomme l'arme `key`.
func tagOfKey(t *testing.T, key string) uint32 {
	t.Helper()
	for _, tag := range killicon.ResolvedTags() {
		if ic, ok := killicon.Lookup(tag); ok && ic.WeaponKey == key {
			return tag
		}
	}
	t.Fatalf("aucune source de dégât ne résout vers %q", key)
	return 0
}

// TestKillSourceIcon_ClasseDeLArmeAuRegistre : une source qui nomme une arme sort avec la classe de la
// Synthèse pour cette arme — épaule, poing, lourde (l'épée est une arme lourde, non de la mêlée) et grenade.
func TestKillSourceIcon_ClasseDeLArmeAuRegistre(t *testing.T) {
	a := NewAssetURLAdapter()
	for key, want := range map[string]string{
		"hinf_ma40_ar":        domain.FragClassShoulder,
		"hinf_sidekick":       domain.FragClassSidearm,
		"hinf_energy_sword":   domain.FragClassHeavy,
		"hinf_gravity_hammer": domain.FragClassHeavy,
		"hinf_frag_grenade":   domain.FragClassGrenade,
		"hinf_plasma_grenade": domain.FragClassGrenade,
	} {
		ic, ok := a.KillSourceIcon(tagOfKey(t, key))
		if !ok || ic.WeaponKey != key {
			t.Fatalf("%s : source non résolue (%+v, %v)", key, ic, ok)
		}
		if ic.WeaponClass != want {
			t.Errorf("%s : classe %q, attendu %q", key, ic.WeaponClass, want)
		}
	}
}

// TestKillSourceIcon_MeleeSansArme : la mêlée générique n'a pas de clé d'arme ; sa classe vient de la classe
// que le dictionnaire des sources de dégât déclare pour son tag, jamais de son image.
func TestKillSourceIcon_MeleeSansArme(t *testing.T) {
	a := NewAssetURLAdapter()
	var melee int
	for _, tag := range killicon.ResolvedTags() {
		ic, ok := a.KillSourceIcon(tag)
		if !ok || ic.WeaponKey != "" || ic.WeaponClass != domain.FragClassMelee {
			continue
		}
		melee++
		if ic.ImageURL == "" {
			t.Errorf("source %08x : mêlée sans image", tag)
		}
	}
	if melee == 0 {
		t.Fatal("aucune source de mêlée sans arme classée : la règle `CLASSE MELEE` ne produit plus rien")
	}
}

// TestKillSourceIcon_SansClasseEtablie : une source inconnue, l'écrasement et toute source sans clé hors
// mêlée et grenade n'ont pas de classe.
func TestKillSourceIcon_SansClasseEtablie(t *testing.T) {
	a := NewAssetURLAdapter()
	if got := killSourceClass(0xDEADBEEF, ""); got != "" {
		t.Errorf("source inconnue : classe %q", got)
	}
	if ic, _ := a.SplatterIcon(); ic.WeaponClass != "" {
		t.Errorf("écrasement : classe %q", ic.WeaponClass)
	}
	parClasse := map[string]int{}
	for _, tag := range killicon.ResolvedTags() {
		ic, ok := a.KillSourceIcon(tag)
		if !ok {
			continue
		}
		parClasse[ic.WeaponClass]++
		if ic.WeaponKey != "" {
			continue
		}
		if ic.WeaponClass != "" && ic.WeaponClass != domain.FragClassMelee && ic.WeaponClass != domain.FragClassGrenade {
			t.Errorf("source %08x sans clé : classe %q hors mêlée et grenade", tag, ic.WeaponClass)
		}
	}
	t.Logf("sources résolues par classe servie (\"\" = sans classe) : %v", parClasse)
}

// TestKillSourceIcon_ClasseServieEstUneClasseDeLaSynthese : toute classe posée est une classe de la
// Répartition des frags que le web range (le registre porte aussi `other`, que le lecteur ignore : il ne
// doit pas sortir d'une source dont le pont nomme une arme sans que cela soit voulu).
func TestKillSourceIcon_ClasseServieEstUneClasseDeLaSynthese(t *testing.T) {
	a := NewAssetURLAdapter()
	connues := map[string]bool{
		domain.FragClassShoulder: true, domain.FragClassSidearm: true, domain.FragClassHeavy: true,
		domain.FragClassMelee: true, domain.FragClassGrenade: true, domain.FragClassVehicle: true,
		domain.FragClassTurret: true, domain.FragClassEquipment: true, domain.FragClassEnvironmental: true,
	}
	for _, tag := range killicon.ResolvedTags() {
		ic, ok := a.KillSourceIcon(tag)
		if ok && ic.WeaponClass != "" && !connues[ic.WeaponClass] {
			t.Errorf("source %08x : classe %q inconnue de la Synthèse", tag, ic.WeaponClass)
		}
	}
}

// TestKillSourceIcon_ToutePorteuseDArmeAUneClasse : chaque règle du pont qui nomme une arme nomme une clé
// que le registre classe.
func TestKillSourceIcon_ToutePorteuseDArmeAUneClasse(t *testing.T) {
	classes := weapons.ClassesByKey()
	for _, r := range killicon.Rules() {
		if r.WeaponKey != "" && classes[r.WeaponKey] == "" {
			t.Errorf("règle %s %q : la clé %q n'a aucune classe au registre", r.Genre, r.Key, r.WeaponKey)
		}
	}
}
