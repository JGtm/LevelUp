/**
 * outline-colors.ts — Palette des couleurs d'outline Halo Infinite.
 *
 * Noms officiels EN/FR, codes hex officiels Halo Infinite.
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
  { id: 'grass',        nameEn: 'Grass',        nameFr: 'Herbe',        hex: '#BEFC77' },
  { id: 'citron',       nameEn: 'Citron',        nameFr: 'Citron',       hex: '#A8FF3A' },
  { id: 'jade',         nameEn: 'Jade',          nameFr: 'Jade',         hex: '#96FFC4' },
  { id: 'mint',         nameEn: 'Mint',          nameFr: 'Menthe',       hex: '#43FF93' },
  { id: 'sky-blue',     nameEn: 'Sky Blue',      nameFr: 'Bleu ciel',    hex: '#5DD4FF' },
  { id: 'cerulean',     nameEn: 'Cerulean',      nameFr: 'Céruléen',     hex: '#4DC0FF' },
  { id: 'sunshine',     nameEn: 'Sunshine',      nameFr: 'Soleil',       hex: '#FFFB6C' },
  { id: 'pineapple',    nameEn: 'Pineapple',     nameFr: 'Ananas',       hex: '#FFFA2E' },
  { id: 'carrot',       nameEn: 'Carrot',        nameFr: 'Carotte',      hex: '#FF734D' },
  { id: 'tangelo',      nameEn: 'Tangelo',       nameFr: 'Tangelo',      hex: '#FF4D0A' },
  { id: 'salmon',       nameEn: 'Salmon',        nameFr: 'Saumon',       hex: '#FF4C4C' },
  { id: 'vermilion',    nameEn: 'Vermilion',     nameFr: 'Vermillon',    hex: '#FF5756' },
  { id: 'cotton-candy', nameEn: 'Cotton Candy',  nameFr: 'Barbe à papa', hex: '#FFB0FF' },
  { id: 'cherry',       nameEn: 'Cherry',        nameFr: 'Cerise',       hex: '#FC4DDD' },
  { id: 'lavender',     nameEn: 'Lavender',      nameFr: 'Lavande',      hex: '#B986DA' },
  { id: 'aubergine',    nameEn: 'Aubergine',     nameFr: 'Aubergine',    hex: '#B64EFB' },
]

export function findOutlineColor(id: string | null): HaloOutlineColor | null {
  if (!id) return null
  return HALO_OUTLINE_COLORS.find((c) => c.id === id) ?? null
}
