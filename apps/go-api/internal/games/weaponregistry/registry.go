// Package weaponregistry — passage PRINCIPAL de la résolution d'arme (cf.
// .ai/PLAN_WEAPON_TAXONOMY.md). Charge le référentiel canonique
// weapons/weapon_ids/weapon_families (metadata.duckdb, seedé par la migration
// add_weapon_registry) en mémoire au boot, et résout n'importe quel id
// (filmshell/stock_id/…) vers l'arme canonique. Title-agnostic. Dégradation
// gracieuse : id/clé inconnu → (zéro-valeur, false), jamais de panic.
package weaponregistry

// Weapon — arme canonique résolue. Champs optionnels = "" / nil si non renseignés.
type Weapon struct {
	Key       string
	TitleSlug string
	Name      string
	// NameFR retiré V72-06 : le nom d'affichage (en/fr) est la source unique keyée par
	// weapon_key dans weapon_names.toml (→ weapon_name_labels). Le registre = dimensions.
	Class        string // sidearm|shoulder|heavy|melee|grenade (manipulation)
	Role         string // automatic|precision|sniper|shotgun|sidearm|power|special|melee|grenade
	FamilyKey    string
	Faction      string // human|covenant|forerunner|banished
	DamageType   string
	Manufacturer string
	Extra        map[string]any
}

// FamilyLabels — libellés d'une famille cross-titre.
type FamilyLabels struct {
	Key    string
	NameEN string
	NameFR string
}

// Registry — interface de résolution (le passage principal).
type Registry interface {
	// ByID résout n'importe quel id (filmshell, stock_id, module…) vers l'arme.
	ByID(titleSlug, idKind, idValue string) (Weapon, bool)
	// ByKey résout par clé canonique (ex. "hinf_br75").
	ByKey(weaponKey string) (Weapon, bool)
	// Family résout les libellés d'une famille (ex. "battle_rifle").
	Family(familyKey string) (FamilyLabels, bool)
}

// MemRegistry — implémentation en mémoire (cache chargé au boot via LoadFromDB).
type MemRegistry struct {
	byKey    map[string]Weapon
	byID     map[string]string // "title|kind|value" → weapon_key
	families map[string]FamilyLabels
}

var _ Registry = (*MemRegistry)(nil)

func idCacheKey(titleSlug, idKind, idValue string) string {
	return titleSlug + "|" + idKind + "|" + idValue
}

// ByID résout un id vers l'arme. Match exact (ids numériques en string).
func (r *MemRegistry) ByID(titleSlug, idKind, idValue string) (Weapon, bool) {
	key, ok := r.byID[idCacheKey(titleSlug, idKind, idValue)]
	if !ok {
		return Weapon{}, false
	}
	w, ok := r.byKey[key]
	return w, ok
}

// ByKey résout par clé canonique.
func (r *MemRegistry) ByKey(weaponKey string) (Weapon, bool) {
	w, ok := r.byKey[weaponKey]
	return w, ok
}

// Family résout les libellés d'une famille.
func (r *MemRegistry) Family(familyKey string) (FamilyLabels, bool) {
	f, ok := r.families[familyKey]
	return f, ok
}

// Counts — cardinalités (armes, ids, familles) pour le log boot / diagnostics.
func (r *MemRegistry) Counts() (weapons, ids, families int) {
	return len(r.byKey), len(r.byID), len(r.families)
}
