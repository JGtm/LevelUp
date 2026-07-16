/**
 * Rareté Halo Infinite — normalisation et mapping vers classes Tailwind.
 *
 * GameCMS renvoie une chaîne libre (`Common`, `Rare`, `Epic`, `Legendary`, `Mythic`)
 * — parfois absente. Ce module isole la logique de normalisation et fournit
 * les classes de couleur officielles utilisées par le carrousel et le lightbox.
 *
 * Pas de JSX ici : ce module est pur, testable sans DOM.
 */

export type RarityTier = 'common' | 'rare' | 'epic' | 'legendary' | 'mythic'

const RARITY_LABELS_FR: Record<RarityTier, string> = {
  common: 'Commun',
  rare: 'Rare',
  epic: 'Épique',
  legendary: 'Légendaire',
  mythic: 'Mythique',
}

const RARITY_LABELS_EN: Record<RarityTier, string> = {
  common: 'Common',
  rare: 'Rare',
  epic: 'Epic',
  legendary: 'Legendary',
  mythic: 'Mythic',
}

/**
 * Schéma de couleurs Halo Infinite officiel : gris / bleu / violet / or / rouge.
 * Une teinte par usage (bg, badge, glow) pour rester cohérent entre les deux
 * entrées (home compact + page Pass) sans dupliquer les classes.
 *
 * Convention : `bg` est un dégradé radial subtil — appliqué en fond derrière
 * l'image de la reward dans le carrousel ET le lightbox. Pas de bordure colorée
 * (visuel Halo Infinite : le tile est rempli, pas encadré).
 */
export interface RarityStyle {
  /** Dégradé de fond derrière l'image de la reward (carrousel + lightbox). */
  bg: string
  /** Variante de Badge dans le lightbox (rappel de la rareté). */
  badge: string
  /** Halo lumineux léger sur la carte (Légendaire/Mythique surtout). */
  glow: string
  /** Couleur solide pour les segments de la barre de distribution des raretés. */
  segment: string
}

const RARITY_STYLES: Record<RarityTier, RarityStyle> = {
  common: {
    bg: 'bg-gradient-to-br from-slate-500/40 via-slate-600/30 to-slate-800/50',
    badge: 'bg-slate-500/30 text-slate-100 border border-slate-400/40',
    glow: '',
    segment: 'bg-slate-500',
  },
  rare: {
    bg: 'bg-gradient-to-br from-sky-500/55 via-sky-600/40 to-sky-900/60',
    badge: 'bg-sky-500/30 text-sky-100 border border-sky-400/50',
    glow: 'shadow-[0_0_10px_-2px_rgba(56,189,248,0.45)]',
    segment: 'bg-sky-500',
  },
  epic: {
    bg: 'bg-gradient-to-br from-purple-500/60 via-purple-600/45 to-purple-900/65',
    badge: 'bg-purple-500/35 text-purple-100 border border-purple-400/50',
    glow: 'shadow-[0_0_12px_-2px_rgba(192,132,252,0.5)]',
    segment: 'bg-purple-500',
  },
  legendary: {
    bg: 'bg-gradient-to-br from-amber-400/65 via-amber-500/50 to-amber-800/70',
    badge: 'bg-amber-500/35 text-amber-100 border border-amber-400/55',
    glow: 'shadow-[0_0_14px_-2px_rgba(251,191,36,0.6)]',
    segment: 'bg-amber-400',
  },
  mythic: {
    bg: 'bg-gradient-to-br from-rose-500/65 via-rose-600/50 to-rose-900/70',
    badge: 'bg-rose-500/35 text-rose-100 border border-rose-500/55',
    glow: 'shadow-[0_0_16px_-2px_rgba(244,63,94,0.6)]',
    segment: 'bg-rose-500',
  },
}

/**
 * Normalise une chaîne brute GameCMS en `RarityTier` ou `null` si non reconnue.
 * Tolérant à la casse et aux variations (`Ultra Rare` → null par défaut, à étendre si besoin).
 */
export function normalizeRarity(raw?: string | null): RarityTier | null {
  if (!raw) return null
  const v = raw.trim().toLowerCase()
  switch (v) {
    case 'common':
    case 'normal':
      return 'common'
    case 'rare':
      return 'rare'
    case 'epic':
      return 'epic'
    case 'legendary':
      return 'legendary'
    case 'mythic':
      return 'mythic'
    default:
      return null
  }
}

export function rarityStyle(tier: RarityTier | null): RarityStyle | null {
  return tier == null ? null : RARITY_STYLES[tier]
}

// preferEN accepte une locale courte ('fr'/'en') OU Intl ('fr-FR'/'en-US') —
// certains appelants (PassContentSummary) passent l'Intl locale.
function preferEN(locale: string): boolean {
  return locale.toLowerCase().startsWith('en')
}

export function rarityLabel(tier: RarityTier, locale: string = 'fr'): string {
  return preferEN(locale) ? RARITY_LABELS_EN[tier] : RARITY_LABELS_FR[tier]
}

/**
 * Convertit un ItemType brut GameCMS (PascalCase) en libellé localisé.
 * Ex: `ArmorCoating` → `Revêtement d'armure`, `WeaponCharm` → `Breloque`.
 * Renvoie le brut si non mappé (lecture acceptable comme fallback).
 */
const ITEM_TYPE_LABELS_FR: Record<string, string> = {
  ArmorCoating: "Revêtement d'armure",
  ArmorHelmet: 'Casque',
  ArmorHelmetAttachment: 'Accessoire de casque',
  ArmorChestAttachment: 'Accessoire de torse',
  ArmorLeftShoulderPad: 'Épaulette gauche',
  ArmorRightShoulderPad: 'Épaulette droite',
  ArmorKneePad: 'Genouillère',
  ArmorHipAttachment: 'Accessoire de hanche',
  ArmorVisor: 'Visière',
  ArmorWristAttachment: 'Accessoire de poignet',
  ArmorGlove: 'Gants',
  ArmorMythicEffect: 'Effet mythique',
  WeaponCoating: "Revêtement d'arme",
  WeaponCharm: 'Breloque',
  WeaponEmblem: "Emblème d'arme",
  VehicleCoating: 'Revêtement de véhicule',
  VehicleEmblem: 'Emblème de véhicule',
  SpartanEmblem: 'Emblème Spartan',
  SpartanBackdropImage: "Image d'arrière-plan",
  SpartanActionPose: "Pose d'action",
  SpartanVoice: 'Voix Spartan',
  SpartanBody: 'Corps Spartan',
  AiTheme: 'Thème IA',
  AiModel: 'Modèle IA',
  Currency: 'Monnaie',
  XpBoost: 'Boost XP',
  ChallengeSwap: 'Relance défi',
}

// Miroir EN (termes officiels Halo Infinite). Sans lui, les types d'items du
// Battle Pass restaient en FR sous UI EN (« Revêtement d'armure » au lieu de
// « Armor Coating »).
const ITEM_TYPE_LABELS_EN: Record<string, string> = {
  ArmorCoating: 'Armor Coating',
  ArmorHelmet: 'Helmet',
  ArmorHelmetAttachment: 'Helmet Attachment',
  ArmorChestAttachment: 'Chest Attachment',
  ArmorLeftShoulderPad: 'Left Shoulder Pad',
  ArmorRightShoulderPad: 'Right Shoulder Pad',
  ArmorKneePad: 'Knee Pad',
  ArmorHipAttachment: 'Hip Attachment',
  ArmorVisor: 'Visor',
  ArmorWristAttachment: 'Wrist Attachment',
  ArmorGlove: 'Gloves',
  ArmorMythicEffect: 'Mythic Effect',
  WeaponCoating: 'Weapon Coating',
  WeaponCharm: 'Weapon Charm',
  WeaponEmblem: 'Weapon Emblem',
  VehicleCoating: 'Vehicle Coating',
  VehicleEmblem: 'Vehicle Emblem',
  SpartanEmblem: 'Spartan Emblem',
  SpartanBackdropImage: 'Backdrop Image',
  SpartanActionPose: 'Action Pose',
  SpartanVoice: 'Spartan Voice',
  SpartanBody: 'Spartan Body',
  AiTheme: 'AI Theme',
  AiModel: 'AI Model',
  Currency: 'Currency',
  XpBoost: 'XP Boost',
  ChallengeSwap: 'Challenge Swap',
}

/**
 * Catégorise un ItemType brut en "armure" (pièces et revêtements qui modifient
 * l'apparence du Spartan en jeu) ou "cosmétique" (le reste : armes, véhicules,
 * profil, IA). Permet de distinguer la personnalisation d'armure des cosmétiques
 * purs dans les résumés de pass saisonniers.
 */
const ARMOR_ITEM_TYPES = new Set([
  'ArmorCoating',
  'ArmorHelmet',
  'ArmorHelmetAttachment',
  'ArmorChestAttachment',
  'ArmorLeftShoulderPad',
  'ArmorRightShoulderPad',
  'ArmorKneePad',
  'ArmorHipAttachment',
  'ArmorVisor',
  'ArmorWristAttachment',
  'ArmorGlove',
  'ArmorMythicEffect',
  'SpartanBody',
])

export function isArmorItemType(raw?: string | null): boolean {
  if (!raw) return false
  return ARMOR_ITEM_TYPES.has(raw.trim())
}

export function itemTypeLabel(raw?: string | null, locale: string = 'fr'): string | null {
  if (!raw) return null
  const trimmed = raw.trim()
  if (!trimmed) return null
  const map = preferEN(locale) ? ITEM_TYPE_LABELS_EN : ITEM_TYPE_LABELS_FR
  if (map[trimmed]) return map[trimmed]
  // Fallback : transformer PascalCase en mots espacés ("ArmorVisor" → "Armor visor").
  const spaced = trimmed.replace(/([a-z])([A-Z])/g, '$1 $2')
  return spaced.charAt(0).toUpperCase() + spaced.slice(1).toLowerCase()
}
