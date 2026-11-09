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

import type { Locale } from '@/lib/i18n/locale'

export interface HaloOutlineColor {
  id: string
  name: Record<Locale, string>
  hex: string
}

export const HALO_OUTLINE_COLORS: readonly HaloOutlineColor[] = [
  { id: 'grass',        name: { en: 'Grass', fr: 'Herbe' },        hex: '#BEFC77' },
  { id: 'citron',       name: { en: 'Citron', fr: 'Citron' },       hex: '#A8FF3A' },
  { id: 'jade',         name: { en: 'Jade', fr: 'Jade' },         hex: '#96FFC4' },
  { id: 'mint',         name: { en: 'Mint', fr: 'Menthe' },       hex: '#43FF93' },
  { id: 'sky-blue',     name: { en: 'Sky Blue', fr: 'Bleu ciel' },    hex: '#5DD4FF' },
  { id: 'cerulean',     name: { en: 'Cerulean', fr: 'Céruléen' },     hex: '#4DC0FF' },
  { id: 'sunshine',     name: { en: 'Sunshine', fr: 'Soleil' },       hex: '#FFFB6C' },
  { id: 'pineapple',    name: { en: 'Pineapple', fr: 'Ananas' },       hex: '#FFFA2E' },
  { id: 'carrot',       name: { en: 'Carrot', fr: 'Carotte' },      hex: '#FF734D' },
  { id: 'tangelo',      name: { en: 'Tangelo', fr: 'Tangelo' },      hex: '#FF4D0A' },
  { id: 'salmon',       name: { en: 'Salmon', fr: 'Saumon' },       hex: '#FF4C4C' },
  { id: 'vermilion',    name: { en: 'Vermilion', fr: 'Vermillon' },    hex: '#FF5756' },
  { id: 'cotton-candy', name: { en: 'Cotton Candy', fr: 'Barbe à papa' }, hex: '#FFB0FF' },
  { id: 'cherry',       name: { en: 'Cherry', fr: 'Cerise' },       hex: '#FC4DDD' },
  { id: 'lavender',     name: { en: 'Lavender', fr: 'Lavande' },      hex: '#B986DA' },
  { id: 'aubergine',    name: { en: 'Aubergine', fr: 'Aubergine' },    hex: '#B64EFB' },
]

export function findOutlineColor(id: string | null): HaloOutlineColor | null {
  if (!id) return null
  return HALO_OUTLINE_COLORS.find((c) => c.id === id) ?? null
}
