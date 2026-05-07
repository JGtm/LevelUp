/**
 * outline-colors.ts — Palette des couleurs d'outline Halo Infinite.
 *
 * Noms officiels EN/FR, hex approximatifs calés sur les teintes in-game.
 * Utilisé dans les settings d'accessibilité pour mapper les couleurs
 * que l'utilisateur a choisies in-game sur les tokens team-ally / team-enemy.
 *
 * ⚠️  Ce fichier est une exception justifiée à la règle "zéro magic hex" —
 *     il centralise les définitions de couleurs Halo, pas les composants.
 */

export interface HaloOutlineColor {
  id: string
  nameEn: string
  nameFr: string
  hex: string
}

export const HALO_OUTLINE_COLORS: readonly HaloOutlineColor[] = [
  { id: 'red',           nameEn: 'Red',              nameFr: 'Rouge',          hex: '#E5231C' },
  { id: 'scarlet',       nameEn: 'Scarlet',           nameFr: 'Écarlate',       hex: '#AA1A1A' },
  { id: 'hot-pink',      nameEn: 'Hot Pink',          nameFr: 'Rose vif',       hex: '#FF1A7F' },
  { id: 'magenta',       nameEn: 'Magenta',           nameFr: 'Magenta',        hex: '#CC00AA' },
  { id: 'strawberry',    nameEn: 'Strawberry Crush',  nameFr: 'Framboise',      hex: '#FF405E' },
  { id: 'pink-lemonade', nameEn: 'Pink Lemonade',     nameFr: 'Limonade rosée', hex: '#FFB3C6' },
  { id: 'yellow',        nameEn: 'Yellow',            nameFr: 'Jaune',          hex: '#FFD600' },
  { id: 'sunshine',      nameEn: 'Sunshine',          nameFr: 'Soleil',         hex: '#FFA500' },
  { id: 'pineapple',     nameEn: 'Pineapple',         nameFr: 'Ananas',         hex: '#F5A623' },
  { id: 'tangelo',       nameEn: 'Tangelo',           nameFr: 'Tangelo',        hex: '#FF6600' },
  { id: 'lime',          nameEn: 'Lime',              nameFr: 'Citron vert',    hex: '#7FFF00' },
  { id: 'mint',          nameEn: 'Mint',              nameFr: 'Menthe',         hex: '#00FFCC' },
  { id: 'jade',          nameEn: 'Jade',              nameFr: 'Jade',           hex: '#00A86B' },
  { id: 'blue',          nameEn: 'Blue',              nameFr: 'Bleu',           hex: '#0057FF' },
  { id: 'purple',        nameEn: 'Purple',            nameFr: 'Violet',         hex: '#8000FF' },
  { id: 'aubergine',     nameEn: 'Aubergine',         nameFr: 'Aubergine',      hex: '#5C1A5C' },
]

export function findOutlineColor(id: string | null): HaloOutlineColor | null {
  if (!id) return null
  return HALO_OUTLINE_COLORS.find((c) => c.id === id) ?? null
}
