package mappings

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/replay/mapvar"
	"levelup/go-api/internal/testutil"
)

const objectiveRolesValide = `
[meta]
title_slug = "halo_infinite"
schema_version = 1

[[modes]]
match = ["CTF", "Neutral Flag"]
roles = ["flag_spawn", "flag_delivery"]

[[modes]]
match = ["Strongholds"]
roles = ["strongholds_zone"]
neutral = true
`

func TestObjectiveRoles_ChargementValide(t *testing.T) {
	set, err := LoadObjectiveRolesFromBytes("test.toml", []byte(objectiveRolesValide))
	if err != nil {
		t.Fatalf("chargement: %v", err)
	}
	if set.TitleSlug() != "halo_infinite" || set.SchemaVersion() != 1 {
		t.Errorf("meta: slug=%q v=%d", set.TitleSlug(), set.SchemaVersion())
	}
	modes := set.Modes()
	if len(modes) != 2 {
		t.Fatalf("modes = %d, attendu 2", len(modes))
	}
	ctf := modes[0]
	if len(ctf.Match) != 2 || ctf.Match[0] != "CTF" || ctf.Neutral {
		t.Errorf("entrée CTF inattendue: %+v", ctf)
	}
	if len(ctf.Roles) != 2 || ctf.Roles[0] != mapvar.RoleFlagSpawn || ctf.Roles[1] != mapvar.RoleFlagDelivery {
		t.Errorf("rôles CTF inattendus: %v", ctf.Roles)
	}
	// L'ordre du fichier est conservé, et le drapeau neutral est porté par l'entrée.
	if sh := modes[1]; !sh.Neutral || len(sh.Roles) != 1 || sh.Roles[0] != mapvar.RoleStrongholdZone {
		t.Errorf("entrée Strongholds inattendue: %+v", sh)
	}
}

// TestObjectiveRoles_ValidationStricte — chaque configuration fautive est REFUSÉE : un
// silence servirait un rejeu sans objectifs indistinguable d'un mode sans objectifs.
func TestObjectiveRoles_ValidationStricte(t *testing.T) {
	cas := []struct {
		nom     string
		toml    string
		fragmnt string
	}{
		{"role inconnu", `
[meta]
title_slug = "halo_infinite"
schema_version = 1
[[modes]]
match = ["CTF"]
roles = ["koth_hill"]
`, "inconnu du décodeur"},
		{"match vide", `
[meta]
title_slug = "halo_infinite"
schema_version = 1
[[modes]]
roles = ["flag_spawn"]
`, "match vide"},
		{"roles vide", `
[meta]
title_slug = "halo_infinite"
schema_version = 1
[[modes]]
match = ["CTF"]
`, "roles vide"},
		{"aucune entree", `
[meta]
title_slug = "halo_infinite"
schema_version = 1
`, "aucune entrée"},
		{"meta absente", `
[[modes]]
match = ["CTF"]
roles = ["flag_spawn"]
`, "title_slug manquant"},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			_, err := LoadObjectiveRolesFromBytes("test.toml", []byte(c.toml))
			if err == nil || !strings.Contains(err.Error(), c.fragmnt) {
				t.Errorf("err = %v, attendu contenant %q", err, c.fragmnt)
			}
		})
	}
}

// TestObjectiveRoles_FichierAbsentRemonteTelQuel — l'absence est un cas de l'appelant
// (titre sans objectifs statiques) : elle doit rester reconnaissable par os.IsNotExist.
func TestObjectiveRoles_FichierAbsentRemonteTelQuel(t *testing.T) {
	_, err := LoadObjectiveRolesFromFile(filepath.Join(t.TempDir(), "objective_roles.toml"))
	if err == nil {
		t.Fatal("attendu une erreur pour fichier absent")
	}
	if !os.IsNotExist(errUnwrapAll(err)) {
		t.Errorf("l'absence doit rester détectable (os.IsNotExist), reçu: %v", err)
	}
}

// errUnwrapAll déroule la chaîne d'enveloppes jusqu'à l'erreur d'origine.
func errUnwrapAll(err error) error {
	type unwrapper interface{ Unwrap() error }
	for {
		u, ok := err.(unwrapper)
		if !ok {
			return err
		}
		next := u.Unwrap()
		if next == nil {
			return err
		}
		err = next
	}
}

// TestObjectiveRoles_FichierDuDepot — le TOML VERSIONNÉ du titre par défaut charge, et
// porte les sept modes du plan (CTF, Strongholds, Oddball, Stockpile, Extraction, Assaut,
// King of the Hill — ce dernier depuis le lot C-ter volet 2).
func TestObjectiveRoles_FichierDuDepot(t *testing.T) {
	root, err := testutil.RepoRoot()
	if err != nil {
		t.Fatalf("racine du dépôt introuvable : %v", err)
	}
	path := filepath.Join(root, "config", "titles", "halo_infinite", "mappings", "objective_roles.toml")
	set, err := LoadObjectiveRolesFromFile(path)
	if err != nil {
		t.Fatalf("le fichier versionné doit charger: %v", err)
	}
	modes := set.Modes()
	if len(modes) != 7 {
		t.Fatalf("modes = %d, attendu 7 (CTF, Strongholds, Oddball, Stockpile, Extraction, Assaut, KOTH)", len(modes))
	}
	// La règle produit du lot 4 : Bastion et Extraction s'affichent NEUTRES (possession
	// dynamique non décodée) ; le drapeau, lui, garde ses couleurs d'équipe.
	neutres := map[mapvar.Role]bool{}
	for _, m := range modes {
		for _, r := range m.Roles {
			if m.Neutral {
				neutres[r] = true
			}
		}
	}
	if !neutres[mapvar.RoleStrongholdZone] || !neutres[mapvar.RoleExtractionZone] || !neutres[mapvar.RoleHill] {
		t.Errorf("strongholds_zone, extraction_zone et hill doivent être neutres, reçu: %v", neutres)
	}
	if neutres[mapvar.RoleFlagSpawn] || neutres[mapvar.RoleFlagDelivery] {
		t.Errorf("les rôles drapeau ne doivent PAS être neutres: %v", neutres)
	}
}

// reposRootDepuisTests SUPPRIMÉ (revue ronde 1, R1-1) : la remontée depuis le répertoire
// courant était le troisième mécanisme maison de localisation de la racine. Le mécanisme
// canonique est testutil.RepoRoot() (déduit de l'arbre source), gardé par
// internal/archlint.
