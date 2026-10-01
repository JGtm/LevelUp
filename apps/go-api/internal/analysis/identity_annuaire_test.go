package analysis

import (
	"strings"
	"testing"
)

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

// TestAnnuaireGamertags_Nomme : la frontière du port GamertagResolver — nommé par un niveau de la
// cascade (bot, alias, participant, kill-feed), ou laissé au libellé masqué.
func TestAnnuaireGamertags_Nomme(t *testing.T) {
	a := AnnuaireGamertags{
		Alias:        map[string]string{"x_alias": "NomAlias", "x_vide": ""},
		Participants: map[string]string{"x_part": "NomPart"},
		KillFeed:     map[string]string{"x_kf": "NomKF"},
	}
	for xuid, attendu := range map[string]bool{
		"x_alias": true, "x_part": true, "x_kf": true, "bid(1.0)": true, "bid(99.0)": true,
		"x_vide": false, "x_inconnu": false,
	} {
		if got := a.Nomme(xuid); got != attendu {
			t.Errorf("Nomme(%q) = %v, attendu %v (Resolve : %q)", xuid, got, attendu, a.Resolve(xuid))
		}
		if !attendu && a.Resolve(xuid) != MaskedXuidLabel(xuid) {
			t.Errorf("%q non nommé mais Resolve rend %q", xuid, a.Resolve(xuid))
		}
	}
}

// TestAnnuaireSQL_PorteeBase : la portée base lit les participants sur TOUTE la base — aucun
// `match_id` —, la portée de la lecture les borne à ses matchs (un seul gabarit par niveau) ; la
// localisation du repli (DA.10) ne projette que des match_id.
func TestAnnuaireSQL_PorteeBase(t *testing.T) {
	base := AnnuaireNomsBaseSQL()
	if strings.Contains(base, "match_id") {
		t.Errorf("AnnuaireNomsBaseSQL borne par match :\n%s", base)
	}
	if lecture := AnnuaireNomsSQL(); strings.Replace(lecture, " AND "+SQLDansListeParJointure("match_id"), "", 1) != base {
		t.Errorf("les deux portées divergent hors de la borne de match :\n%s\n---\n%s", lecture, base)
	}
	// DA.10 : la localisation ne rend QUE des match_id (lecture brute admise pour localiser,
	// jamais pour lire une valeur) — aucune colonne de nom dans ce qu'elle projette.
	loc := AnnuaireKillFeedLocaliserSQL()
	if n := strings.Count(loc, "SELECT match_id FROM"); n != 4 {
		t.Errorf("AnnuaireKillFeedLocaliserSQL : %d projections « SELECT match_id », attendu 4 :\n%s", n, loc)
	}
	for _, interdit := range []string{"SELECT xuid, gamertag", "MAX(", "_gamertag AS", "SELECT *"} {
		if strings.Contains(loc, interdit) {
			t.Errorf("AnnuaireKillFeedLocaliserSQL lit une valeur (%q) :\n%s", interdit, loc)
		}
	}
}
