package ops

import (
	"testing"

	"levelup/go-api/internal/migration"
)

func TestParseBackupKey(t *testing.T) {
	cases := []struct {
		key  string
		want BackupKey
		ok   bool
	}{
		{"halo_infinite:shared_matches_v2", BackupKey{Slug: "halo_infinite", Target: migration.TargetShared}, true},
		{"halo_5:metadata", BackupKey{Slug: "halo_5", Target: migration.TargetMetadata}, true},
		{"halo_infinite:shared_pve", BackupKey{Slug: "halo_infinite", Target: migration.TargetSharedPvE}, true},
		{"halo_infinite:shared_social", BackupKey{Slug: "halo_infinite", Target: migration.TargetSharedSocial}, true},
		{"halo_infinite:player:Chocoboflor", BackupKey{Slug: "halo_infinite", Target: migration.TargetPlayer, Gamertag: "Chocoboflor"}, true},
		{"xbox_aliases", BackupKey{}, false},
		{"halo_infinite:inconnue", BackupKey{Slug: "halo_infinite"}, false},
		{"halo_infinite:player:", BackupKey{}, false},
	}
	for _, c := range cases {
		got, ok := ParseBackupKey(c.key)
		if ok != c.ok || (ok && got != c.want) {
			t.Errorf("ParseBackupKey(%q) = %+v, %v ; want %+v, %v", c.key, got, ok, c.want, c.ok)
		}
	}
	// Aller-retour avec les constructeurs de la découverte.
	if k, ok := ParseBackupKey(playerBackupKey("halo_5", "GT")); !ok || k.Gamertag != "GT" || k.Slug != "halo_5" {
		t.Errorf("aller-retour clé joueur : %+v %v", k, ok)
	}
	if k, ok := ParseBackupKey(warehouseBackupKey("halo_5", backupDBSharedSoc)); !ok || k.Target != migration.TargetSharedSocial {
		t.Errorf("aller-retour clé d'entrepôt : %+v %v", k, ok)
	}
}
