package canonical

import "testing"

func TestMatchEventType_KnownAndAll(t *testing.T) {
	all := AllMatchEventTypes()
	if len(all) != 9 {
		t.Fatalf("AllMatchEventTypes() = %d, want 9", len(all))
	}
	for _, ty := range all {
		if !IsKnownMatchEventType(ty) {
			t.Errorf("type %q devrait être connu", ty)
		}
	}
	if IsKnownMatchEventType("bogus") {
		t.Error("type inconnu devrait retourner false")
	}
}

func TestKillKind_KnownAndAll(t *testing.T) {
	all := AllKillKinds()
	if len(all) != 5 {
		t.Fatalf("AllKillKinds() = %d, want 5", len(all))
	}
	for _, k := range all {
		if !IsKnownKillKind(k) {
			t.Errorf("kind %q devrait être connu", k)
		}
	}
	// assassination est bien une mécanique canonique (5e valeur, cross-titre).
	if !IsKnownKillKind(KillKindAssassination) {
		t.Error("assassination doit être une KillKind connue")
	}
	// headshot N'EST PAS une mécanique (c'est un modificateur orthogonal MatchEvent.Headshot).
	if IsKnownKillKind("headshot") {
		t.Error("headshot ne doit PAS être une KillKind (modificateur orthogonal)")
	}
}

func TestMatchEventOptions_Wants(t *testing.T) {
	// Types vide = tout.
	empty := MatchEventOptions{}
	if !empty.Wants(MatchEventKill) || !empty.Wants(MatchEventWeaponPickup) {
		t.Error("options vides devraient tout vouloir")
	}
	// Filtre kill-feed = kill + medal seulement.
	killfeed := MatchEventOptions{Types: []MatchEventType{MatchEventKill, MatchEventMedal}}
	if !killfeed.Wants(MatchEventKill) || !killfeed.Wants(MatchEventMedal) {
		t.Error("kill+medal devraient être voulus")
	}
	if killfeed.Wants(MatchEventWeaponPickup) || killfeed.Wants(MatchEventSpawn) {
		t.Error("weapon_pickup/spawn ne devraient PAS être voulus (filtre kill-feed)")
	}
}
