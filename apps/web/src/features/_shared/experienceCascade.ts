/**
 * experienceCascade — helpers partagés du filtre « expérience » (cascade locale).
 *
 * Source unique de EXPERIENCE_TO_CASCADE + setsEqual, consommée par
 * useLocalFilterBar (hook réutilisable) ET SynthesisPage (H3, 2026-07-04 — dédup
 * d'une copie 13 L à l'identique). Un seul mapping = pas de dérive des libellés
 * canoniques backend.
 */
import type { Experience } from '@/features/_shared/ExperienceDropdown'

// Mapping experience → cascade.experience_types. Utilise les libellés canoniques
// backend « PVP classé » / « PVP non classé »
// (service/filters_service.go::experienceLabels). 'all' → [] = pas de filtre.
export const EXPERIENCE_TO_CASCADE: Record<Experience, string[]> = {
  all: [],
  ranked: ['PVP classé'],
  unranked: ['PVP non classé'],
}

// setsEqual — égalité ordre-insensible de deux Set<string>.
export function setsEqual(a: Set<string>, b: Set<string>): boolean {
  if (a.size !== b.size) return false
  for (const v of a) if (!b.has(v)) return false
  return true
}
