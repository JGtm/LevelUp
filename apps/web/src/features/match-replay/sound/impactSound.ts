/**
 * impactSound.ts — LE SON D'IMPACT D'UN PROJECTILE DE TIR D'UNE ARME À PIED (roquettes...).
 *
 * La table est keyée par le weapon_key du titre (`WeaponLabel.key`). Les armes de VÉHICULE n'y
 * figurent pas : leur son de détonation est déclaré par le registre du titre
 * (`vehicle_weapons.toml`, `impact_sound`) et publié sur `VehicleWeapon.impactSound` — le client
 * ne porte aucun tag d'arme de véhicule (garde-rail `vehicleWeaponRegistry.guard.test.ts`).
 *
 * Seuls les sons VALIDÉS À L'OREILLE sont déclarés (rendus extraits du jeu, événement sonore
 * reconstitué couche par couche) ; une arme absente de la table explose en silence, et n'emprunte
 * JAMAIS le son d'une explosion de grenade (ce serait affirmer une arme qu'on ignore). Pour en
 * déclarer un : ajouter `weapon_key -> stem` ci-dessous ; le fichier
 * `static/sounds/{slug}/<stem>.wav` doit exister (garde-rail `replaySoundAssets.guard.test.ts`), et
 * la lecture passe par `grenadeSoundEvents`.
 */
export const IMPACT_SOUND_STEMS: Readonly<Record<string, string>> = {}

/** Le stem du son d'impact d'une arme à pied, ou `undefined` (silence). */
export function impactSoundStem(weaponKey: string): string | undefined {
  return Object.hasOwn(IMPACT_SOUND_STEMS, weaponKey) ? IMPACT_SOUND_STEMS[weaponKey] : undefined
}
