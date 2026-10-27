package duckdb

import (
	"context"
	"testing"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/i18n"
)

// nomsDeCarte rend le nom de chaque langue et pose toujours celui de la langue par défaut (clé des
// filtres) ; libelleDeCarte en est le nom dans la langue par défaut.
func TestNomsDeCarte_NomParLangueEtCleParDefaut(t *testing.T) {
	registre := i18n.Text{i18n.Fallback: "Streets"}
	trad := i18n.Text{i18n.Default: "Ruelles"}
	noms := nomsDeCarte(registre, trad)
	if noms[i18n.Default] != "Ruelles" || noms[i18n.Fallback] != "Streets" {
		t.Errorf("noms = %v", noms)
	}
	if got := libelleDeCarte(registre, trad); got != "Ruelles" {
		t.Errorf("libelleDeCarte = %q", got)
	}
	// Sans traduction : la clé retombe sur la langue de repli.
	if got := libelleDeCarte(registre, nil); got != "Streets" {
		t.Errorf("sans traduction : %q, attendu Streets", got)
	}
	if got := nomsDeCarte(registre, nil).Pick(i18n.Fallback); got != "Streets" {
		t.Errorf("en sans traduction : %q", got)
	}
	// Le registre d'entrée n'est pas modifié.
	if registre.Has(i18n.Default) {
		t.Errorf("le registre ne doit pas être modifié : %v", registre)
	}
}

// Les duels : la carte et le mode de chaque ligne suivent la langue de la requête. En français la carte
// prend sa traduction d'actif et le mode sa traduction de mode_name_tr (le registre les nomme en anglais) ;
// en anglais le nom du registre reste ; une ligne sans identifiant ni traduction garde son nom.
func TestNameDuels_SuitLaLangueDeLaRequete(t *testing.T) {
	meta := metaDesVariantes(t)
	for _, q := range []string{
		`CREATE TABLE mode_name_tr (mode_en VARCHAR, lang VARCHAR, name VARCHAR)`,
		`INSERT INTO mode_name_tr VALUES ('Slayer','fr','Assassin')`,
		`INSERT INTO asset_translations VALUES ('pair-slayer','pair','fr-FR','Arène:Assassin sur Tonnerre','',now())`,
	} {
		if _, err := meta.Exec(context.Background(), q); err != nil {
			t.Fatalf("seed %q: %v", q, err)
		}
	}
	r := &CareerRepo{pdb: &PlayerDB{Metadata: meta, TitleSlug: "halo_infinite"}}
	names := []duelNames{
		{en: "Thunderhead", id: "base-thunder", pairID: ""},
		{en: "Sans carte", id: "", pairID: ""},
		{en: "Live Fire", id: "", pairID: "pair-slayer"},
	}
	build := func() []domain.RelationDuelRawRow {
		return []domain.RelationDuelRawRow{
			{MapName: "Thunderhead", Mode: "Arena:Slayer on Thunderhead"},
			{MapName: "Sans carte", Mode: "Arena:Unknown on Nowhere"},
			{MapName: "Live Fire", Mode: ""},
		}
	}

	fr := build()
	r.nameDuels(ctxkeys.WithLocale(context.Background(), "fr"), fr, names)
	if fr[0].MapName != "Tonnerre" || fr[0].Mode != "Assassin" {
		t.Errorf("fr, ligne 1 : carte %q, mode %q, attendu Tonnerre / Assassin", fr[0].MapName, fr[0].Mode)
	}
	if fr[1].MapName != "Sans carte" || fr[1].Mode != "Arena:Unknown on Nowhere" {
		t.Errorf("fr, ligne 2 : sans traduction le nom du registre reste : %+v", fr[1])
	}
	if fr[2].Mode != "Arène:Assassin sur Tonnerre" {
		t.Errorf("fr, ligne 3 : mode %q, attendu le nom d'actif de la paire (registre vide)", fr[2].Mode)
	}

	en := build()
	r.nameDuels(ctxkeys.WithLocale(context.Background(), "en"), en, names)
	if en[0].MapName != "Thunderhead" || en[0].Mode != "Arena:Slayer on Thunderhead" || en[1].MapName != "Sans carte" || en[2].Mode != "" {
		t.Errorf("en : le nom du registre ne doit pas changer : %+v", en)
	}
}

// Média : le libellé affiché suit la langue ; celui de tri et de filtre reste celui de la langue par défaut.
func TestMediaComputedLabels_AfficheParLangueCleParDefaut(t *testing.T) {
	row := mediaEnrichedRow{Match: &mediaMatchRegistryInfo{
		MapName:      i18n.Text{i18n.Default: "Ruelles", i18n.Fallback: "Streets"},
		PairName:     i18n.Text{i18n.Default: "Assassin", i18n.Fallback: "Slayer"},
		PlaylistName: i18n.Text{i18n.Default: "Partie rapide", i18n.Fallback: "Quick Play"},
	}}
	if row.computedMapLabel() != "Ruelles" || row.computedMapLabelIn(i18n.Fallback) != "Streets" {
		t.Errorf("carte : %q / %q", row.computedMapLabel(), row.computedMapLabelIn(i18n.Fallback))
	}
	if row.computedModeLabel() != "Assassin" || row.computedModeLabelIn(i18n.Fallback) != "Slayer" {
		t.Errorf("mode : %q / %q", row.computedModeLabel(), row.computedModeLabelIn(i18n.Fallback))
	}
	if row.computedPlaylistLabel() != "Partie rapide" || row.computedPlaylistLabelIn(i18n.Fallback) != "Quick Play" {
		t.Errorf("playlist : %q / %q", row.computedPlaylistLabel(), row.computedPlaylistLabelIn(i18n.Fallback))
	}
	// Les options de carte et de playlist gardent le nom de la langue par défaut pour clé et ajoutent le
	// libellé de la langue de la requête pour le repli d'affichage.
	pairs := extractMapPairs([]mediaEnrichedRow{{Match: &mediaMatchRegistryInfo{MapID: "m", MapName: row.Match.MapName}}}, i18n.Fallback)
	if len(pairs) != 1 || pairs[0].label != "Ruelles" || pairs[0].display != "Streets" {
		t.Errorf("paires de cartes = %+v", pairs)
	}
}

// Un registre qui ne nomme rien (Halo 5) : la langue de repli vient de la traduction d'actif, de sorte
// qu'aucune langue, `en` comprise, ne soit servie vide. Un nom du registre n'est jamais écrasé, et une
// langue sans traduction reste absente (la lecture replie sur la langue de repli).
func TestFillUntranslated_PoseLaLangueDeRepliQuandLeRegistreEstVide(t *testing.T) {
	trad := i18n.Text{i18n.Fallback: "Streets", i18n.Default: "Ruelles"}

	var vide i18n.Text
	fillUntranslated(&vide, trad)
	if vide.Pick(i18n.Fallback) != "Streets" || vide.Pick(i18n.Default) != "Ruelles" {
		t.Errorf("registre vide : %v, attendu Streets / Ruelles", vide)
	}

	registre := i18n.Text{i18n.Fallback: "Streets (registre)"}
	fillUntranslated(&registre, trad)
	if registre[i18n.Fallback] != "Streets (registre)" {
		t.Errorf("le nom du registre ne doit pas etre ecrase : %v", registre)
	}
	if registre[i18n.Default] != "Ruelles" {
		t.Errorf("la langue par defaut prend sa traduction : %v", registre)
	}

	seuleLaLangueParDefaut := i18n.Text{}
	fillUntranslated(&seuleLaLangueParDefaut, i18n.Text{i18n.Default: "Ruelles"})
	if seuleLaLangueParDefaut.Has(i18n.Fallback) {
		t.Errorf("une langue sans traduction reste absente : %v", seuleLaLangueParDefaut)
	}
}
