/**
 * firstBlood — adaptation du payload API vers les props du chart `FirstBloodLanes`.
 *
 * Trois surfaces produit consomment le MÊME bloc `first_blood` (Escouade/Dynamique
 * multi-joueurs, Timeseries solo, Session solo) : la conversion snake_case → props
 * du composant et le calcul de la fenêtre d'axe vivent ici, et nulle part ailleurs.
 * Le composant `FirstBloodLanes` reste ignorant du contrat HTTP.
 */
import type { FirstBloodPlayerSeries } from '@/components/charts/FirstBloodLanes'
import type { FirstBloodPlayerSeriesDTO } from '@/lib/api/types'

/** Fenêtre d'axe minimale (5 min) — sous ce seuil les bandes deviennent illisibles. */
const MIN_MAX_SEC = 300
/** Granularité d'arrondi de la fenêtre d'axe : la minute (l'axe tick toutes les 60 s). */
const AXIS_STEP_SEC = 60

/** Projette le payload API en séries de props (camelCase). */
export function toFirstBloodSeries(
  rows: FirstBloodPlayerSeriesDTO[] | null | undefined,
): FirstBloodPlayerSeries[] {
  if (!rows || rows.length === 0) return []
  return rows.map((r) => ({
    player: r.player,
    matches: (r.matches ?? []).map((m) => ({
      matchId: m.match_id,
      firstKillSec: m.first_kill_sec ?? null,
      firstDeathSec: m.first_death_sec ?? null,
      // DEC-4 (retours utilisateur 2026-08-29) : carte/mode/date pour le
      // tooltip — map_ui/mode_ui optionnels au contrat (dégradation propre
      // s'ils manquent, cf. FirstBloodLanes), start_time toujours renseigné.
      mapUI: m.map_ui ?? undefined,
      modeUI: m.mode_ui ?? undefined,
      startTime: m.start_time,
    })),
  }))
}

/**
 * Fenêtre de l'axe X : p99 des événements arrondi à la minute supérieure, plancher
 * à 5 min. Le p99 (et non le max) évite qu'UNE partie interminable — un joueur qui
 * ne meurt qu'à 9 minutes — écrase toutes les autres bandes sur la gauche de l'axe ;
 * le point hors fenêtre reste tracé par ECharts, simplement collé au bord droit.
 */
export function firstBloodMaxSec(series: FirstBloodPlayerSeries[]): number {
  const values: number[] = []
  for (const s of series) {
    for (const m of s.matches) {
      if (m.firstKillSec != null && Number.isFinite(m.firstKillSec)) values.push(m.firstKillSec)
      if (m.firstDeathSec != null && Number.isFinite(m.firstDeathSec)) values.push(m.firstDeathSec)
    }
  }
  if (values.length === 0) return MIN_MAX_SEC
  values.sort((a, b) => a - b)
  const p99 = values[Math.min(values.length - 1, Math.ceil(values.length * 0.99) - 1)]
  return Math.max(MIN_MAX_SEC, Math.ceil(p99 / AXIS_STEP_SEC) * AXIS_STEP_SEC)
}
