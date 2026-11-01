package infinite

import (
	"reflect"
	"strings"
	"testing"
)

// TestEtiquettesJSON_NomsDuService : les noms de champs JSON que le service Halo envoie, tels que
// les clients d'avant la brique les lisaient. Une étiquette changée ferait décoder un champ vide
// sans erreur : chaque nom est donc tenu ici, champ par champ (`chemin` : champs imbriqués séparés
// par des points ; une tranche est traversée vers son élément).
func TestEtiquettesJSON_NomsDuService(t *testing.T) {
	cas := []struct {
		typ    any
		chemin string
		tag    string
	}{
		{PrivacyDTO{}, "RankedMatchesPrivacy", "RankedMatchesPrivacy"},
		{PrivacyDTO{}, "AllMatchesPrivacy", "AllMatchesPrivacy"},
		{SeasonCalendarDTO{}, "Seasons.SeasonMetadataPath", "SeasonMetadataPath"},
		{DiscoveryAssetDTO{}, "AssetID", "AssetId"},
		{DiscoveryAssetDTO{}, "VersionID", "VersionId"},
		{EmblemMappingEntry{}, "NameplateCmsPath", "nameplateCmsPath"},
		{EmblemMappingEntry{}, "EmblemCmsPath", "emblemCmsPath"},
	}
	for _, c := range cas {
		typ := reflect.TypeOf(c.typ)
		var tag string
		for _, nom := range strings.Split(c.chemin, ".") {
			for typ.Kind() == reflect.Slice || typ.Kind() == reflect.Pointer {
				typ = typ.Elem()
			}
			champ, ok := typ.FieldByName(nom)
			if !ok {
				t.Fatalf("%T : champ %q introuvable", c.typ, c.chemin)
			}
			tag, typ = champ.Tag.Get("json"), champ.Type
		}
		if tag != c.tag {
			t.Errorf("%T.%s : étiquette json %q, attendu %q", c.typ, c.chemin, tag, c.tag)
		}
	}
}
