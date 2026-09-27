package halo_infinite

import (
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/halo_infinite/film/damagetag"
)

// TestKillSourceCategory_ParLaClasse : chaque tag `OBJET_EXPLOSIF` de la table est un objet
// explosif — cle de registre ou non (c'est tout le point : la plupart n'en ont pas) —,
// chaque `DEGAT_GLOBAL` est la chute / l'environnement, et rien d'autre n'a de categorie
// (« Outils de destruction » de l'Escouade, decision D8 du 2026-09-26).
func TestKillSourceCategory_ParLaClasse(t *testing.T) {
	r := NewKillSourceRegistry()
	objets, objetsSansCle, globaux := 0, 0, 0
	for _, l := range damagetag.Labels() {
		got, ok := r.KillSourceCategory(l.Tag)
		switch l.Class {
		case damagetag.ClassObjet:
			objets++
			if !ok || got != domain.KillSourceCategoryExplosiveObject {
				t.Errorf("tag %08x (OBJET_EXPLOSIF) -> (%q, %v), want explosive_object", l.Tag, got, ok)
			}
			if _, keyed := r.KillSourceRegistryKey(l.Tag); !keyed {
				objetsSansCle++
			}
		case damagetag.ClassGlobal:
			globaux++
			if !ok || got != domain.KillSourceCategoryEnvironment {
				t.Errorf("tag %08x (DEGAT_GLOBAL) -> (%q, %v), want environment", l.Tag, got, ok)
			}
		default:
			if ok {
				t.Errorf("tag %08x (%s) -> categorie %q, want aucune", l.Tag, l.Class, got)
			}
		}
	}
	if objets == 0 || globaux == 0 {
		t.Fatalf("table sans objet explosif (%d) ou sans degat global (%d)", objets, globaux)
	}
	if objetsSansCle == 0 {
		t.Error("tous les objets explosifs ont une cle : la categorie n'apporterait rien, " +
			"verifier la table killicon")
	}
	if got, ok := r.KillSourceCategory(0xffffffff); ok {
		t.Errorf("tag inconnu -> (%q, true), want (\"\", false)", got)
	}
	// L'adapter d'assets repond comme le registre (le cablage ne connait que lui).
	var a AssetURLAdapter
	if got, ok := a.KillSourceCategory(0x00000024); !ok || got != domain.KillSourceCategoryEnvironment {
		t.Errorf("AssetURLAdapter.KillSourceCategory(0x24) = (%q, %v), want environment", got, ok)
	}
}
