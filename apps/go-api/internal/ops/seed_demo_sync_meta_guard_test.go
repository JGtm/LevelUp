// Package ops — seed_demo_sync_meta_guard_test.go : garde-rails de la politique
// d'inclusion `sync_meta` du seed démo (seed_demo_sync_meta.go).
//
// Ce que ces tests protègent, dans l'ordre d'importance :
//  1. aucune clé credential CONNUE ne traverse (le défaut historique : le seed
//     excluait `msal_token_cache` mais pas `oauth_refresh_token`) ;
//  2. aucune clé INCONNUE ne traverse — donc pas non plus une clé credential
//     FUTURE, celle que personne n'aurait pensé à exclure. C'est l'apport propre de
//     la liste d'inclusion sur la liste d'exclusion qu'elle remplace ;
//  3. la liste d'inclusion elle-même ne peut pas accueillir une clé de forme
//     credential par mégarde ;
//  4. la clause SQL réellement branchée sur l'extraction VIENT de la liste (un
//     retour à une clause écrite à la main, ou à un `NOT IN`, casse ici) ;
//  5. les sentinelles de migration autorisées existent encore côté migrations (une
//     entrée périmée ferait rejouer un rebuild de schéma figé sur la base démo).
//
// Tests unitaires purs (pas de CGO/DuckDB). La preuve end-to-end « le RT n'est pas
// dans la base démo publiée » est dans seed_demo_integration_test.go.
package ops

import (
	"os"
	"strings"
	"testing"
)

// credentialSyncMetaKeys : clés `sync_meta` qui ont porté un credential joueur.
// Elles ne doivent JAMAIS traverser vers la démo, quel que soit l'état du drop
// physique des colonnes (ADR 0023 Phase 5 / ADR 0026).
var credentialSyncMetaKeys = map[string]string{
	"oauth_refresh_token": "refresh token OAuth du joueur source (ADR 0023) — TRAVERSAIT avant le 2026-08-26",
	"msal_token_cache":    "cache MSAL sérialisé (jeton + compte)",
}

func TestDemoSyncMetaCredentialKeysNeverTravel(t *testing.T) {
	for key, why := range credentialSyncMetaKeys {
		if demoSyncMetaKeyAllowed(key) {
			t.Errorf("clé credential %q autorisée vers la démo (%s) — le jeu de données démo est PUBLIC", key, why)
		}
		if strings.Contains(demoSyncMetaWhere(), key) {
			t.Errorf("clé credential %q présente dans la clause d'extraction : %s", key, demoSyncMetaWhere())
		}
	}
}

// TestDemoSyncMetaDefaultDeny : toute clé non listée est refusée. Si l'une d'elles
// devient réellement nécessaire à la démo, l'ajouter à demoSyncMetaAllowedKeys avec
// une justification datée ET la retirer d'ici — c'est le point du test : la décision
// doit être explicite, jamais un effet de bord.
func TestDemoSyncMetaDefaultDeny(t *testing.T) {
	denied := map[string]string{
		// Clés credential PLAUSIBLES demain : aucune n'existe aujourd'hui, et c'est
		// exactement le cas que la liste d'exclusion ne couvrait pas.
		"spartan_token":          "credential futur",
		"xsts_token":             "credential futur",
		"sisu_session":           "credential futur",
		"api_client_secret":      "credential futur",
		"oauth_refresh_token_v3": "credential futur (variante de nom)",
		"discord_webhook_url":    "secret d'intégration",
		// Clé de dé-anonymisation : player_xuid porte le xuid RÉEL du joueur source
		// et n'est PAS réécrite par extractPlayerTables (seule 'xuid' l'est).
		"player_xuid": "xuid réel du joueur source, jamais anonymisé",
		// Clés bénignes mais NON nécessaires : la démo a la synchronisation coupée
		// (app_settings spnkr_refresh_* = false), ces horodatages/compteurs n'y ont
		// aucun lecteur. Hors liste par choix, pas par oubli.
		"last_sync":               "non nécessaire (sync coupée en démo)",
		"last_delta_sync":         "non nécessaire (sync coupée en démo)",
		"last_post_sync_at":       "non nécessaire (diag admin, best-effort si absent)",
		"last_career_sync_at":     "non nécessaire (sync coupée en démo)",
		"current_rank":            "non nécessaire (aucun lecteur runtime)",
		"live_update":             "non nécessaire (legacy sans lecteur)",
		"last_seen_app_version":   "non nécessaire (absent = initialisation silencieuse au boot)",
		"une_cle_inventee_demain": "clé inconnue : le défaut est le refus",
	}
	for key, why := range denied {
		if demoSyncMetaKeyAllowed(key) {
			t.Errorf("clé %q autorisée vers la démo alors qu'elle est hors liste (%s).\n"+
				"Si elle est devenue nécessaire : l'ajouter à demoSyncMetaAllowedKeys avec justification datée, puis la retirer de ce test.",
				key, why)
		}
	}
}

// TestDemoSyncMetaAllowlistHasNoCredentialShapedKey : filet contre le futur — une
// clé de forme credential ajoutée à la liste d'inclusion par mégarde.
func TestDemoSyncMetaAllowlistHasNoCredentialShapedKey(t *testing.T) {
	forbidden := []string{
		"token", "secret", "password", "passwd", "credential", "cookie",
		"cache", "session", "bearer", "refresh", "api_key", "private", "auth",
	}
	for _, key := range demoSyncMetaAllowedKeys {
		lower := strings.ToLower(key)
		for _, frag := range forbidden {
			if strings.Contains(lower, frag) {
				t.Errorf("clé %q autorisée vers la démo mais de forme credential (contient %q) — le jeu de données démo est PUBLIC", key, frag)
			}
		}
		if strings.ContainsAny(key, "'\"\\;") {
			t.Errorf("clé %q porteuse d'un caractère de citation : la clause WHERE générée serait cassée", key)
		}
	}
}

// TestDemoSyncMetaCoversDemoRuntimeNeeds : sentinelle anti-faux-vert. Une liste
// vidée ferait passer tous les tests ci-dessus au vert tout en cassant la démo
// (ResolveXUID sans clé xuid) — et une liste trop maigre ferait rejouer les
// migrations de schéma figé sur la base démo.
func TestDemoSyncMetaCoversDemoRuntimeNeeds(t *testing.T) {
	required := map[string]string{
		"xuid":                             "duckdb.ResolveXUID (Q3ResolveXUID) — sans elle la player DB démo ne résout plus son joueur",
		"career_progression_rebuilt_v1":    "sans la sentinelle, rebuild_career_progression rejoue son DDL FIGÉ sur la base démo",
		"career_xp_total_default_fixed_v1": "sans la sentinelle, fix_career_xp_total_default_zero rejoue sur la base démo",
	}
	for key, why := range required {
		if !demoSyncMetaKeyAllowed(key) {
			t.Errorf("clé %q retirée de demoSyncMetaAllowedKeys : %s", key, why)
		}
	}
}

// TestDemoSyncMetaWhereIsInclusionBuiltFromAllowlist : la clause branchée sur
// l'extraction est bien celle dérivée de la liste, et c'est bien une INCLUSION.
// Réécrire la clause à la main dans playerTablesWhere, ou revenir à un `NOT IN`,
// casse ici.
func TestDemoSyncMetaWhereIsInclusionBuiltFromAllowlist(t *testing.T) {
	var wired string
	found := false
	for _, tbl := range playerTablesWhere {
		if tbl.name == "sync_meta" {
			wired, found = tbl.where, true
			break
		}
	}
	if !found {
		t.Fatal("table sync_meta absente de playerTablesWhere — extraction renommée/supprimée : réviser ce garde-rail")
	}
	if wired != demoSyncMetaWhere() {
		t.Errorf("clause sync_meta branchée = %q, attendu celle dérivée de la liste = %q", wired, demoSyncMetaWhere())
	}
	if strings.Contains(strings.ToUpper(wired), "NOT IN") {
		t.Errorf("clause sync_meta redevenue une liste d'EXCLUSION (%q) : toute clé future, credential compris, traverserait par défaut", wired)
	}
	if !strings.HasPrefix(wired, "key IN (") {
		t.Errorf("clause sync_meta = %q, attendu une inclusion `key IN (...)`", wired)
	}
	if strings.Contains(wired, "%s") {
		t.Errorf("clause sync_meta = %q : un verbe de format y serait interpolé avec les match_ids par extractPlayerTables", wired)
	}
	for _, key := range demoSyncMetaAllowedKeys {
		if !strings.Contains(wired, "'"+key+"'") {
			t.Errorf("clé autorisée %q absente de la clause générée %q", key, wired)
		}
	}
}

// TestDemoSyncMetaMigrationSentinelsStillExist : les sentinelles de migration
// autorisées doivent toujours être les constantes réelles des migrations player.
// Si une migration renomme la sienne, l'entrée devient morte SANS bruit et la base
// démo se remet à rejouer un rebuild de schéma figé au seed suivant.
func TestDemoSyncMetaMigrationSentinelsStillExist(t *testing.T) {
	sentinels := map[string]string{
		"career_progression_rebuilt_v1":    "../games/halo_infinite/migrations/steps_player_repairs.go",
		"career_xp_total_default_fixed_v1": "../games/halo_infinite/migrations/steps_player.go",
	}
	for key, path := range sentinels {
		src, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("lecture %s: %v — fichier de migration déplacé : réviser ce garde-rail", path, err)
		}
		if !strings.Contains(string(src), `"`+key+`"`) {
			t.Errorf("sentinelle %q introuvable dans %s : entrée périmée de demoSyncMetaAllowedKeys "+
				"(la migration correspondante rejouerait sur la base démo)", key, path)
		}
		if !demoSyncMetaKeyAllowed(key) {
			t.Errorf("sentinelle %q non autorisée alors qu'elle est attendue par ce garde-rail", key)
		}
	}
}
