package analysis

import "testing"

// TestAnnuaireGamertags_Resolve rejoue la cascade de v_gamertag_lookup niveau par niveau.
// Chaque cas ne laisse ouvert que le niveau qu'il vérifie : si l'ordre des niveaux se
// dérègle, le nom rendu change et le cas rougit.
func TestAnnuaireGamertags_Resolve(t *testing.T) {
	a := AnnuaireGamertags{
		Alias: map[string]string{
			"x_alias":      "NomAlias",
			"x_alias_vide": "",
			"bid(1.0)":     "AliasDeBot", // un bot connu garde son nom officiel
		},
		Participants: map[string]string{
			"x_alias":      "NomParticipant",
			"x_alias_vide": "NomParticipant",
			"x_part":       "NomParticipant",
		},
		KillFeed: map[string]string{
			"x_alias": "NomKillFeed",
			"x_part":  "NomKillFeed",
			"x_kf":    "NomKillFeed",
		},
	}
	cas := []struct {
		xuid, attendu, pourquoi string
	}{
		{"bid(1.0)", "343 Meowlnir", "bot connu : nom officiel, AVANT l'alias"},
		{"bid(99.0)", "bid(99.0)", "bot inconnu : le xuid tel quel (ELSE de BotSQLCase)"},
		{"bid(1.0", "bid(1.0", "bot sans parenthèse : clé exacte comme le SQL, pas la tolérance de BotDisplayName"},
		{"x_alias", "NomAlias", "l'alias passe avant les participants et le kill-feed"},
		{"x_alias_vide", "NomParticipant", "un alias vide ne nomme pas"},
		{"x_part", "NomParticipant", "les participants passent avant le kill-feed"},
		{"x_kf", "NomKillFeed", "le kill-feed nomme ce que rien d'autre ne nomme"},
		{"2533274823110022", "Joueur 0022", "inconnu de toutes les sources : libellé masqué"},
		{"x1", "Joueur x1", "xuid de moins de quatre caractères : entier"},
	}
	for _, c := range cas {
		if got := a.Resolve(c.xuid); got != c.attendu {
			t.Errorf("Resolve(%q) = %q, attendu %q (%s)", c.xuid, got, c.attendu, c.pourquoi)
		}
	}
}

// TestAnnuaireGamertags_ResolveSansCartes : un annuaire vide (cartes nil) ne panique pas et
// ne rend que les niveaux qui se déduisent du xuid seul.
func TestAnnuaireGamertags_ResolveSansCartes(t *testing.T) {
	var a AnnuaireGamertags
	if got := a.Resolve("bid(0.0)"); got != "343 Ritzy" {
		t.Errorf("bot connu = %q, attendu 343 Ritzy", got)
	}
	if got := a.Resolve("123456"); got != "Joueur 3456" {
		t.Errorf("xuid inconnu = %q, attendu Joueur 3456", got)
	}
}

// TestMaskedXuidLabel_CompteEnCaracteres : `right()` de DuckDB compte des caractères, pas des
// octets — le miroir Go aussi.
func TestMaskedXuidLabel_CompteEnCaracteres(t *testing.T) {
	if got := MaskedXuidLabel("abcdéfgh"); got != "Joueur éfgh" {
		t.Errorf("MaskedXuidLabel = %q, attendu Joueur éfgh", got)
	}
	if got := MaskedXuidLabel("xyzé"); got != "Joueur xyzé" {
		t.Errorf("MaskedXuidLabel = %q, attendu Joueur xyzé", got)
	}
	if got := MaskedXuidLabel(""); got != "Joueur " {
		t.Errorf("MaskedXuidLabel(\"\") = %q, attendu \"Joueur \"", got)
	}
}
