package weaponregistry

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
)

// LoadFromDB charge le registre depuis metadata.duckdb (3 tables) en mémoire.
// Appelé une fois au boot (après la migration add_weapon_registry). Émet le log
// boot weapon_registry_loaded avec les cardinalités.
func LoadFromDB(ctx context.Context, db *sql.DB) (*MemRegistry, error) {
	r := &MemRegistry{
		byKey:    make(map[string]Weapon),
		byID:     make(map[string]string),
		families: make(map[string]FamilyLabels),
	}
	if err := r.loadWeapons(ctx, db); err != nil {
		return nil, err
	}
	if err := r.loadIDs(ctx, db); err != nil {
		return nil, err
	}
	if err := r.loadFamilies(ctx, db); err != nil {
		return nil, err
	}
	w, i, f := r.Counts()
	slog.InfoContext(ctx, "weapon_registry_loaded", "weapons", w, "ids", i, "families", f)
	return r, nil
}

func (r *MemRegistry) loadWeapons(ctx context.Context, db *sql.DB) error {
	// name_fr retiré du registre (V72-06) : le nom d'affichage est la source unique
	// keyée par weapon_key (weapon_name_labels). Le registre ne porte que dimensions.
	rows, err := db.QueryContext(ctx, `SELECT weapon_key, title_slug, name, class, role,
		family_key, faction, damage_type, manufacturer, extra FROM weapons`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var w Weapon
		var class, role, family, faction, dmg, manu, extra sql.NullString
		if err := rows.Scan(&w.Key, &w.TitleSlug, &w.Name, &class, &role,
			&family, &faction, &dmg, &manu, &extra); err != nil {
			return err
		}
		w.Class, w.Role = class.String, role.String
		w.FamilyKey, w.Faction = family.String, faction.String
		w.DamageType, w.Manufacturer = dmg.String, manu.String
		if extra.Valid && extra.String != "" {
			_ = json.Unmarshal([]byte(extra.String), &w.Extra)
		}
		r.byKey[w.Key] = w
	}
	return rows.Err()
}

func (r *MemRegistry) loadIDs(ctx context.Context, db *sql.DB) error {
	rows, err := db.QueryContext(ctx, `SELECT title_slug, id_kind, id_value, weapon_key FROM weapon_ids`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var title, kind, value, key string
		if err := rows.Scan(&title, &kind, &value, &key); err != nil {
			return err
		}
		r.byID[idCacheKey(title, kind, value)] = key
	}
	return rows.Err()
}

func (r *MemRegistry) loadFamilies(ctx context.Context, db *sql.DB) error {
	rows, err := db.QueryContext(ctx, `SELECT family_key, name_en, name_fr FROM weapon_families`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var f FamilyLabels
		if err := rows.Scan(&f.Key, &f.NameEN, &f.NameFR); err != nil {
			return err
		}
		r.families[f.Key] = f
	}
	return rows.Err()
}
