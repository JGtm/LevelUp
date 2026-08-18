// Package replaylabels assemble le CATALOGUE DE LIBELLÉS du rejeu 2D d'Halo Infinite à
// partir des mappings versionnés du titre.
//
// C'EST LE POINT DE JONCTION DES TROIS SOURCES, ET IL N'Y EN A QU'UN :
//
//	famille d'arme (high-32)  --registre d'armes-->  weapon_key
//	weapon_key                --weapon_names.toml-->  {en, fr}
//	weapon_key                --replay_labels.toml--> effet de rendu
//	rang de grenade / index de capacité --replay_labels.toml--> {en, fr}
//
// POURQUOI ICI. Le décodage (internal/analysis) est title-agnostic : il sait lire un
// film, pas ce qu'est un Ravager. Les catalogues appartiennent au TITRE — c'est la
// frontière posée par l'ADR 0011. Jusqu'au 2026-08-02 ils vivaient dans le code du
// rejeu, dont deux EN FRANÇAIS, ce qui interdisait l'anglais autant qu'un second titre.
//
// HORS LIGNE : ce chargement lit des fichiers ; il appartient à cmd/replay-build, jamais
// à un chemin de requête (l'API sert l'artefact déjà nommé).
package replaylabels

import (
	"fmt"
	"path/filepath"

	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/assets/static"
	"levelup/go-api/internal/domain/title"
	halo "levelup/go-api/internal/games/halo_infinite"
	"levelup/go-api/internal/games/mappings"
	"levelup/go-api/internal/games/weapons"
)

// Load assemble le catalogue de libellés d'un titre depuis ses mappings versionnés.
//
// Les deux fichiers sont OBLIGATOIRES quand on demande le catalogue : un rejeu à moitié
// nommé se lit comme une donnée absente alors que c'est une configuration incomplète.
// L'appelant qui accepte un rejeu anonyme passe simplement un catalogue vide.
func Load(repoRoot, titleSlug string) (replay.LabelCatalog, error) {
	dir := title.NewPathResolver(repoRoot).TitleMappingsDir(titleSlug)

	names, err := mappings.LoadWeaponNamesFromFile(filepath.Join(dir, "weapon_names.toml"))
	if err != nil {
		return replay.LabelCatalog{}, fmt.Errorf("noms d'armes du titre %s: %w", titleSlug, err)
	}
	labels, err := mappings.LoadReplayLabelsFromFile(filepath.Join(dir, "replay_labels.toml"))
	if err != nil {
		return replay.LabelCatalog{}, fmt.Errorf("libellés de rejeu du titre %s: %w", titleSlug, err)
	}

	cat := replay.NewLabelCatalog(
		weapons.FilmshellWeaponKeysByFamily(),
		weaponNames(names.Names()),
		labels.ShotEffects(),
		toLabels(titleSlug, labels.GrenadeRanks()),
		abilityPalettes(titleSlug, labels.AbilityPalettes()),
	)
	cat.Icons = weaponIcons(weapons.FilmshellWeaponKeysByFamily())
	// La TEINTE d'un tir (nature de la décharge) est posée après construction, comme les
	// icônes : elle n'entre dans aucune jointure du catalogue, elle voyage jusqu'au
	// document telle quelle (cf. replay_labels.toml, [shot_tints]).
	cat.Tints = labels.ShotTints()
	// La FAMILLE d'un objet d'équipement posé se pose après construction, comme les icônes
	// et les teintes : elle est keyée par GlobalID de tag `eqip` (lu dans le film), et
	// n'entre dans aucune jointure du catalogue.
	cat.EquipmentFamilies = labels.EquipmentObjects()
	// Le DRAPEAU : les identifiants d'objet du monde que le manifeste déclare de la famille
	// `flag`, projetés vers la table d'identité du rejeu. Le filtrage par famille se fait ICI
	// — c'est la couche titre qui sait ce que `flag` veut dire dans son manifeste ; le paquet
	// `replay` ne reçoit que « ces identifiants-là sont des drapeaux », jamais la chaîne.
	cat.FlagObjects = flagObjects(labels.ObjectiveObjects())
	return cat, nil
}

// flagObjects retient les objets d'objectif de la famille DRAPEAU et les rend sous la forme
// que l'artefact connaît. nil quand le titre n'en déclare aucun : la chaîne des socles se
// comporte alors comme avant, et le calque des vies libres reste vide.
func flagObjects(in map[uint32]mappings.ObjectiveObject) map[uint32]replay.Label {
	out := map[uint32]replay.Label{}
	for id, o := range in {
		if o.Family != mappings.ObjectiveFamilyFlag {
			continue
		}
		out[id] = replay.Label{En: o.Label.En, Fr: o.Label.Fr}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// weaponIcons pointe, par famille, l'icône EXTRAITE que le titre sert (fiches joueur du
// rejeu). La clé d'icône EST la famille (le tag `weap` est la moitié haute de
// l'identifiant) : la jointure passe par l'adapter d'assets du titre, jamais par un nom.
// Une famille sans visuel n'entre pas — le client garde le libellé.
func weaponIcons(families map[uint32]string) map[uint32]replay.WeaponIconRef {
	adapter := halo.NewAssetURLAdapter()
	out := map[uint32]replay.WeaponIconRef{}
	for family := range families {
		id := int64(uint64(family) << 32) //nolint:gosec // recomposition voulue : le tag est la moitié haute
		url := adapter.WeaponImageURL(id)
		if url == "" {
			continue
		}
		out[family] = replay.WeaponIconRef{URL: url, Tinted: adapter.WeaponImageIsTinted(id)}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// weaponNames convertit les noms d'armes du loader vers le DTO d'artefact. La JOINTURE
// famille -> weapon_key -> nom, elle, vit dans `replay.NewLabelCatalog` : la recopier ici
// en ferait une seconde version, et un test ne pourrait plus rejouer la vraie.
func weaponNames(in map[string]mappings.WeaponName) map[string]replay.Label {
	out := make(map[string]replay.Label, len(in))
	for k, v := range in {
		out[k] = replay.Label{En: v.En, Fr: v.Fr}
	}
	return out
}

// toLabels convertit les libellés bilingues du loader vers le DTO d'artefact, en
// PRÉSERVANT L'ORDRE — pour les grenades, l'ordre EST le rang.
func toLabels(slug string, in []mappings.BilingualLabel) []replay.Label {
	if len(in) == 0 {
		return nil
	}
	out := make([]replay.Label, 0, len(in))
	for _, l := range in {
		out = append(out, toLabel(slug, l))
	}
	return out
}

// abilityPalettes convertit les palettes du loader vers le DTO d'artefact. Les MARQUEURS
// voyagent avec les noms : c'est l'assemblage (replay/abilities.go) qui classe le film, et
// il ne peut pas classer sans eux.
func abilityPalettes(slug string, in []mappings.AbilityPalette) []replay.AbilityPalette {
	if len(in) == 0 {
		return nil
	}
	out := make([]replay.AbilityPalette, 0, len(in))
	for _, p := range in {
		ranks := make(map[int]replay.Label, len(p.Ranks))
		for rank, l := range p.Ranks {
			ranks[rank] = toLabel(slug, l)
		}
		out = append(out, replay.AbilityPalette{ID: p.ID, Markers: p.Markers, Ranks: ranks})
	}
	return out
}

// toLabel joint le libellé et sa vignette de HUD quand le TOML en pointe une. Les
// vignettes sont des masques (blanc/gris + alpha) : Tinted, comme les icônes d'arme
// extraites — le client les teint à l'encre du thème.
func toLabel(slug string, l mappings.BilingualLabel) replay.Label {
	out := replay.Label{En: l.En, Fr: l.Fr}
	if l.Icon != "" {
		out.Img = static.URL(static.KindWeapon, slug, l.Icon, ".png")
		out.Tinted = true
	}
	return out
}
