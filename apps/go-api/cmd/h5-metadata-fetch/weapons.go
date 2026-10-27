package main

// weapons.go — armes de Halo 5 : weapon_labels (icône, type) et weapon_label_names (un nom par
// langue de config/locales.toml pour laquelle l'API en donne un, ADR 0040 D-5). La langue de l'API
// (sans en-tête) est la langue de repli ; chaque autre langue est une passe Accept-Language.

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"levelup/go-api/internal/games/weapons"
	"levelup/go-api/internal/i18n"
)

type apiWeapon struct {
	Name              string `json:"name"`
	Type              string `json:"type"`
	LargeIconImageURL string `json:"largeIconImageUrl"`
	ID                string `json:"id"`
}

// fetchWeaponNames refetch /weapons dans la langue `lang` (l'API Metadata HONORE Accept-Language, cf.
// fetchMetaLang) et indexe le nom par id d'arme. Best-effort : endpoint en échec ou parse KO ->
// map vide ; seedWeapons n'écrit alors aucun nom pour cette langue (la lecture se replie sur la
// langue de repli).
func fetchWeaponNames(key, lang string) map[string]string {
	out := map[string]string{}
	body, err := fetchMetaLang(key, "weapons", lang)
	if err != nil {
		fmt.Printf("weapons[%s]: SKIP (%v)\n", lang, err)
		return out
	}
	var weapons []apiWeapon
	if err := json.Unmarshal(body, &weapons); err != nil {
		fmt.Printf("weapons[%s]: parse %v\n", lang, err)
		return out
	}
	for _, w := range weapons {
		if name := strings.TrimSpace(w.Name); name != "" && w.ID != "" {
			out[w.ID] = name
		}
	}
	return out
}

// weaponNamesByLanguage rend, par langue de l'application, les noms que l'API donne (id d'arme ->
// nom). La langue de repli vient de la passe sans en-tête déjà faite (`base`) ; les autres, d'une
// passe Accept-Language chacune.
func weaponNamesByLanguage(key string, base []apiWeapon) map[i18n.Locale]map[string]string {
	out := map[i18n.Locale]map[string]string{}
	for _, l := range i18n.Locales() {
		if l == i18n.Fallback {
			names := map[string]string{}
			for _, w := range base {
				if name := strings.TrimSpace(w.Name); name != "" {
					names[w.ID] = name
				}
			}
			out[l] = names
			continue
		}
		out[l] = fetchWeaponNames(key, acceptLanguage(l))
	}
	return out
}

// seedWeapons peuple weapon_labels (id -> icône, type) et weapon_label_names (id, langue -> nom)
// depuis l'API Metadata officielle. Une langue sans nom pour une arme n'écrit rien (jamais de nom
// inventé, ADR 0040 D-4) : la lecture se replie sur le nom de la langue de repli. Sans la passe
// par langue, les armes restaient en anglais (« lightrifle », « FRAG GRENADE ») côté match view.
//
// AUCUN override TOML ici (contre-revue V7.2) : la section [weapons] de
// asset_labels.toml a été retirée par V72-06 au profit de weapon_names.toml
// keyé par weapon_key — la branche d'override par nom EN était inatteignable.
func seedWeapons(db *sql.DB, key string) {
	body, err := fetchMeta(key, "weapons")
	if err != nil {
		fmt.Printf("weapons: SKIP (%v)\n", err)
		return
	}
	var weapons []apiWeapon
	if err := json.Unmarshal(body, &weapons); err != nil {
		fmt.Printf("weapons: parse %v\n", err)
		return
	}
	byLang := weaponNamesByLanguage(key, weapons)
	ctx := context.Background()
	n, names := 0, 0
	for _, w := range weapons {
		// id officiel = numérique (tient dans weapon_labels.weapon_id UBIGINT).
		id, perr := strconv.ParseUint(w.ID, 10, 64)
		if perr != nil {
			continue
		}
		perLang := i18n.Text{}
		for l, byID := range byLang {
			perLang.Set(l, byID[w.ID])
		}
		written, err := storeWeapon(ctx, db, id, w, perLang)
		if err != nil {
			fmt.Printf("weapons: insert %s: %v\n", w.ID, err)
			continue
		}
		n++
		names += written
	}
	fmt.Printf("weapons: %d seedées (sur %d), %d noms par langue écrits\n", n, len(weapons), names)
}

// storeWeapon écrit une arme : sa ligne de weapon_labels (icône, type) puis un nom par langue
// renseignée dans `names`. SELECT puis UPDATE ou INSERT, jamais d'écrasement d'une langue absente
// de `names`. Rend le nombre de noms écrits ou modifiés.
func storeWeapon(ctx context.Context, db *sql.DB, id uint64, w apiWeapon, names i18n.Text) (int, error) {
	if err := upsertWeaponLabel(ctx, db, id, w); err != nil {
		return 0, err
	}
	written := 0
	for l, name := range names {
		changed, err := weapons.SetLabelName(ctx, db, id, l, name)
		if err != nil {
			return written, err
		}
		if changed {
			written++
		}
	}
	return written, nil
}

// upsertWeaponLabel pose icône et type de l'arme : UPDATE si la ligne existe, INSERT sinon.
func upsertWeaponLabel(ctx context.Context, db *sql.DB, id uint64, w apiWeapon) error {
	var n int
	if err := db.QueryRowContext(ctx, //nolint:gosec
		fmt.Sprintf("SELECT COUNT(*) FROM weapon_labels WHERE weapon_id = %d", id)).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		_, err := db.ExecContext(ctx, //nolint:gosec
			fmt.Sprintf("UPDATE weapon_labels SET icon_url = ?, weapon_type = ? WHERE weapon_id = %d", id),
			w.LargeIconImageURL, w.Type)
		return err
	}
	_, err := db.ExecContext(ctx, //nolint:gosec
		fmt.Sprintf("INSERT INTO weapon_labels (weapon_id, icon_url, weapon_type) VALUES (%d, ?, ?)", id),
		w.LargeIconImageURL, w.Type)
	return err
}
