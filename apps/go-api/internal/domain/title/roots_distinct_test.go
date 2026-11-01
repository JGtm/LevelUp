package title

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// cheminsDeRessources : les méthodes du PathResolver dont le chemin est une RESSOURCE livrée avec
// le programme (amendement A-2). Toute autre méthode qui rend un chemin est de la DONNÉE : elle
// doit vivre sous la racine de données. Une méthode neuve non classée ici est donc exigée sous
// Data, et le test rougit si elle ne l'est pas : le classement est une décision, pas un oubli.
var cheminsDeRessources = map[string]bool{
	"RepoRoot": true, "ResourcesRoot": true,
	"TitleMappingsDir": true, "DemoManifestPath": true, "DemoFixturesDir": true,
	"PrebuiltMetadataZipPath": true,
	"MapQuantBoundsPath":      true, "EquipmentEnergyPath": true, "FilmProfilesPath": true,
	"MapFondReglagesPath": true, "MapObjectivesPath": true, "MapPlayedPositionsPath": true,
	"MapWeaponPadsPath": true, "MapCalloutsPath": true, "MapCalloutsLexiquePath": true,
	"MapGeometryDir": true, "MapStructurePath": true,
	"MapBackgroundDir": true, "MapBackgroundPath": true, "MapBackgroundImageFilePath": true,
	"MapBackgroundMetaPath": true,
}

// Les racines elles-mêmes et les méthodes qui ne rendent pas un chemin de fichier du programme.
var horsClassement = map[string]bool{"DataRoot": true, "LogsRoot": true, "Roots": true}

func appelle(pr *PathResolver) map[string]string {
	pt, pv := reflect.TypeOf(pr), reflect.ValueOf(pr)
	args := []string{"halo_infinite", "JGtm", ""} // 3e chaîne : baseDir de ResolveCapturesDir, vide = repli interne
	out := map[string]string{}
	for i := 0; i < pt.NumMethod(); i++ {
		m := pt.Method(i)
		if horsClassement[m.Name] || m.Type.NumOut() != 1 || m.Type.Out(0).Kind() != reflect.String {
			continue
		}
		var in []reflect.Value
		ok, si := true, 0
		for j := 1; j < m.Type.NumIn(); j++ {
			if m.Type.In(j).Kind() != reflect.String || si >= len(args) {
				ok = false
				break
			}
			in = append(in, reflect.ValueOf(args[si]))
			si++
		}
		if ok {
			out[m.Name] = pv.Method(i).Call(in)[0].String()
		}
	}
	return out
}

func sousRacine(chemin, racine string) bool {
	rel, err := filepath.Rel(racine, chemin)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// TROIS RACINES DISTINCTES : aucun chemin de ressource sous la racine de données, aucun chemin de
// donnée sous la racine de ressources (ADR 0008, amendement A-5, lot L11).
func TestPathResolver_TroisRacinesDistinctes_AucunChevauchement(t *testing.T) {
	base := t.TempDir()
	roots := Roots{
		Resources: filepath.Join(base, "ressources"), Data: filepath.Join(base, "donnees"),
		Logs: filepath.Join(base, "journaux"), DataSet: true,
	}
	chemins := appelle(NewPathResolverRoots(roots))
	if len(chemins) < 40 {
		t.Fatalf("le test n'appelle que %d méthodes : la réflexion ne voit plus le résolveur", len(chemins))
	}
	for nom, chemin := range chemins {
		switch {
		case cheminsDeRessources[nom]:
			if !sousRacine(chemin, roots.Resources) || sousRacine(chemin, roots.Data) {
				t.Errorf("%s = %s : une ressource doit vivre sous %s, jamais sous la racine de données", nom, chemin, roots.Resources)
			}
		default:
			if !sousRacine(chemin, roots.Data) || sousRacine(chemin, roots.Resources) {
				t.Errorf("%s = %s : une donnée doit vivre sous %s, jamais sous la racine de ressources "+
					"(si c'est une ressource, l'ajouter à cheminsDeRessources)", nom, chemin, roots.Data)
			}
		}
	}
}

// La forme relative d'une ressource sous Resources est celle d'un checkout (amendement A-2), et les
// overlays écrits à l'exécution à côté du référentiel sont de la donnée.
func TestPathResolver_RessourcesGardentLeurFormeRelative(t *testing.T) {
	roots := Roots{Resources: "/res", Data: "/donnees", Logs: "/journaux", DataSet: true}
	pr := NewPathResolverRoots(roots)
	for got, want := range map[string]string{
		pr.MapQuantBoundsPath("halo_infinite"):       filepath.Join("/res", "data", "titles", "halo_infinite", "reference", "map_quant_bounds.json"),
		pr.PrebuiltMetadataZipPath("halo_infinite"):  filepath.Join("/res", "data", "titles", "halo_infinite", "warehouse", "metadata-prebuilt.zip"),
		pr.TitleMappingsDir("halo_infinite"):         filepath.Join("/res", "config", "titles", "halo_infinite", "mappings"),
		pr.MapWeaponPadsOverlayPath("halo_infinite"): filepath.Join("/donnees", "titles", "halo_infinite", "reference", "generated", "map_weapon_pads.json"),
		pr.MapCalloutsOverlayPath("halo_infinite"):   filepath.Join("/donnees", "titles", "halo_infinite", "reference", "generated", "map_callouts.json"),
		pr.MetadataDBPath("halo_infinite"):           filepath.Join("/donnees", "titles", "halo_infinite", "warehouse", "metadata.duckdb"),
		pr.MigrationBackupsDir():                     filepath.Join("/donnees", "backups", "migrations"),
	} {
		if got != want {
			t.Errorf("chemin = %s, attendu %s", got, want)
		}
	}
}

// A-3 : sans LEVELUP_DATA_DIR, db_profiles.json et app_settings.json restent en tête du checkout ;
// avec, en tête de Data.
func TestPathResolver_FichiersDEtatSelonLaRacineDeDonnees(t *testing.T) {
	def := NewPathResolverRoots(Roots{Resources: "/res", Data: "/res/data"})
	if def.DBProfilesPath() != filepath.Join("/res", "db_profiles.json") || def.AppSettingsPath() != filepath.Join("/res", "app_settings.json") {
		t.Errorf("défaut : %s %s", def.DBProfilesPath(), def.AppSettingsPath())
	}
	set := NewPathResolverRoots(Roots{Resources: "/res", Data: "/donnees", DataSet: true})
	if set.DBProfilesPath() != filepath.Join("/donnees", "db_profiles.json") || set.AppSettingsPath() != filepath.Join("/donnees", "app_settings.json") {
		t.Errorf("Data explicite : %s %s", set.DBProfilesPath(), set.AppSettingsPath())
	}
}
