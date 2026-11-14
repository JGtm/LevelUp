package service

// kill_source_icon_service_test.go — le référentiel des pictogrammes d'arme : même résolution que
// les éliminations, nom dans la langue de la requête, dégradation propre d'un titre sans capacité.

import (
	"context"
	"errors"
	"testing"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/games/halo_infinite"
	"levelup/go-api/internal/games/mappings"
	"levelup/go-api/internal/i18n"
	"levelup/go-api/internal/testutil"

	"github.com/JGtm/halo-infinite-film-decoder/killicon"
)

// adapterSansCapacite : un adapter d'assets qui n'énumère rien (titre sans décodeur de film).
// Le champ nommé porte l'interface : seules les méthodes de l'adapter réel sont utilisées ailleurs.
type adapterSansCapacite struct{ games.TitleAssetURLAdapter }

// adapterEnumerant : un adapter qui énumère une table fixe, sans aucun titre réel.
type adapterEnumerant struct {
	games.TitleAssetURLAdapter
	icons []canonical.KillSourceIcon
}

func (a adapterEnumerant) KillSourceIconsByKey() []canonical.KillSourceIcon { return a.icons }

func resolveurDe(a games.TitleAssetURLAdapter) func(string) (games.TitleAssetURLAdapter, error) {
	return func(string) (games.TitleAssetURLAdapter, error) { return a, nil }
}

func cheminsDuDepot(t *testing.T) *title.PathResolver {
	t.Helper()
	root, err := testutil.RepoRoot()
	if err != nil {
		t.Fatalf("racine du dépôt : %v", err)
	}
	return title.NewPathResolver(root)
}

// L'invariant de bout en bout : pour toute entrée, la clé et l'image sont celles que KillSourceIcon rend
// pour une élimination.
func TestKillSourceIconService_MemeResolutionQueLesEliminations(t *testing.T) {
	adapter := halo_infinite.NewAssetURLAdapter()
	svc := NewKillSourceIconService(resolveurDe(adapter), cheminsDuDepot(t))
	entries, err := svc.ListKillSourceIcons(context.Background(), title.DefaultSlug)
	if err != nil {
		t.Fatalf("ListKillSourceIcons : %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("référentiel vide pour le titre par défaut")
	}
	byKey := map[string]string{}
	for _, e := range entries {
		if e.Key == "" || e.ImageURL == "" || e.Label == "" {
			t.Errorf("entrée incomplète : %+v", e)
		}
		byKey[e.Key] = e.ImageURL
	}
	// Une élimination par chacune de ces sources rend la même clé et la même image.
	for _, ic := range adapter.KillSourceIconsByKey() {
		got, ok := adapter.KillSourceIcon(sourceDeLaCle(t, adapter, ic.WeaponKey))
		if !ok || got.WeaponKey != ic.WeaponKey || got.ImageURL != byKey[ic.WeaponKey] {
			t.Errorf("clé %q : élimination (%q, %q), référentiel %q", ic.WeaponKey, got.WeaponKey, got.ImageURL, byKey[ic.WeaponKey])
		}
	}
}

// sourceDeLaCle retrouve une source de dégât de l'arme `key` : le service n'expose aucun tag, le
// test passe par l'énumération de l'adapter, qui est exactement ce que le service consomme.
func sourceDeLaCle(t *testing.T, a *halo_infinite.AssetURLAdapter, key string) uint32 {
	t.Helper()
	for _, tag := range killicon.ResolvedTags() {
		if ic, ok := a.KillSourceIcon(tag); ok && ic.WeaponKey == key {
			return tag
		}
	}
	t.Fatalf("aucune source de dégât pour la clé %q", key)
	return 0
}

// Le nom est celui de la langue de la requête, lu dans weapon_names.toml du titre.
func TestKillSourceIconService_LibelleDansLaLangueDeLaRequete(t *testing.T) {
	paths := cheminsDuDepot(t)
	set, err := mappings.LoadWeaponNamesInDir(paths.TitleMappingsDir(title.DefaultSlug))
	if err != nil {
		t.Fatalf("weapon_names.toml : %v", err)
	}
	names := set.Names()
	svc := NewKillSourceIconService(resolveurDe(halo_infinite.NewAssetURLAdapter()), paths)

	differeSelonLaLangue := false
	parLangue := map[i18n.Locale]map[string]string{}
	for _, loc := range i18n.Locales() {
		entries, err := svc.ListKillSourceIcons(i18n.WithContext(context.Background(), loc), title.DefaultSlug)
		if err != nil {
			t.Fatalf("langue %s : %v", loc, err)
		}
		parLangue[loc] = map[string]string{}
		for _, e := range entries {
			parLangue[loc][e.Key] = e.Label
			if want := names[e.Key].Pick(loc); want != "" && e.Label != want {
				t.Errorf("langue %s, clé %q : libellé %q, attendu %q", loc, e.Key, e.Label, want)
			}
		}
	}
	locs := i18n.Locales()
	for key, a := range parLangue[locs[0]] {
		for _, loc := range locs[1:] {
			if parLangue[loc][key] != a {
				differeSelonLaLangue = true
			}
		}
	}
	if !differeSelonLaLangue {
		t.Error("aucun libellé ne change avec la langue : le nom n'est pas résolu dans la langue de la requête")
	}
}

// Un titre dont l'adapter n'énumère pas (pas de décodeur de film) rend une liste vide, sans erreur.
func TestKillSourceIconService_TitreSansCapaciteListeVide(t *testing.T) {
	svc := NewKillSourceIconService(resolveurDe(adapterSansCapacite{}), cheminsDuDepot(t))
	entries, err := svc.ListKillSourceIcons(context.Background(), "titre_sans_capacite")
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}
	if entries == nil || len(entries) != 0 {
		t.Errorf("entries = %#v, attendu une liste vide non nulle", entries)
	}
}

// Un titre non résolu rend aussi une liste vide ; une autre erreur du résolveur remonte.
func TestKillSourceIconService_TitreNonResolu(t *testing.T) {
	paths := cheminsDuDepot(t)
	nonResolu := func(string) (games.TitleAssetURLAdapter, error) { return nil, games.ErrTitleNotResolved }
	entries, err := NewKillSourceIconService(nonResolu, paths).ListKillSourceIcons(context.Background(), "inconnu")
	if err != nil || entries == nil || len(entries) != 0 {
		t.Errorf("titre non résolu : (%#v, %v), attendu une liste vide sans erreur", entries, err)
	}
	panne := errors.New("panne du résolveur")
	enPanne := func(string) (games.TitleAssetURLAdapter, error) { return nil, panne }
	if _, err := NewKillSourceIconService(enPanne, paths).ListKillSourceIcons(context.Background(), "x"); !errors.Is(err, panne) {
		t.Errorf("erreur = %v, attendu la panne du résolveur", err)
	}
}

// Sans fichier de noms, les entrées gardent le nom propre de leur source : jamais vide, jamais d'erreur.
func TestKillSourceIconService_SansFichierDeNomsRepliSurLeNomPropre(t *testing.T) {
	adapter := adapterEnumerant{icons: []canonical.KillSourceIcon{
		{WeaponKey: "cle_a", Label: "Nom A", ImageURL: "/static/a.png", Tinted: true},
	}}
	paths := title.NewPathResolver(t.TempDir()) // aucun config/titles/…/weapon_names.toml
	entries, err := NewKillSourceIconService(resolveurDe(adapter), paths).ListKillSourceIcons(context.Background(), "t")
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}
	if len(entries) != 1 || entries[0].Label != "Nom A" || entries[0].Key != "cle_a" || !entries[0].Tinted {
		t.Errorf("entries = %+v", entries)
	}
}
