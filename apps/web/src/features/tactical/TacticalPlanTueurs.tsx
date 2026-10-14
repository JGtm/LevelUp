/**
 * TacticalPlanTueurs — la couche « autre joueur » du plan et sa légende : tant qu'une zone est
 * sélectionnée, un trait relie chaque événement de la zone à l'autre joueur (le tueur d'une mort, la
 * victime d'un frag) et un point marque l'autre joueur. Aucun réglage : le clic sur la zone EST le
 * réglage.
 *
 * LA COUCHE A SA TOILE, superposée à celle de la chaleur et de même taille : le survol (tuile ou
 * point) la repeint seule, sans refaire la chaleur lissée. Elle partage la projection du plan
 * (`projectionDuPlan`), donc le cadrage et le zoom de la chaleur. La toile n'intercepte aucun geste ;
 * le survol d'un point se lit sur le cadre qui la porte (écouteurs natifs : ceux de la toile de la
 * chaleur — clic, glisser, molette — ne sont pas touchés).
 *
 * COULEURS : les jetons de stats de combat — tueur = `stat-deaths`, victime = `stat-kills`.
 */
import { useEffect, useRef, useState } from 'react'

import { resolveToken } from '@/lib/accessibility/resolveToken'
import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'
import { useColorPaletteVersion } from '@/lib/accessibility/useColorPaletteVersion'
import type { TacticalContribution } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'

import type { TacticalText } from './i18n'
import { projectionDuPlan, type FenetreDuPlan, type RepereTactique } from './tacticalView.logic'
import { pointLePlusProche, rolesPresents, type PointDeLaCouche, type RoleDuPoint } from './tueurs.logic'
import { peindreLaCouche } from './tueursPaint'
import { useTailleDuCanvas } from './useTailleDuCanvas'
import { infobulleDuPoint } from './zone.logic'

/** Le jeton de chaque rôle : un tueur est une mort subie, une victime un frag. */
const JETON_DU_ROLE = { tueur: 'stat-deaths', victime: 'stat-kills' } as const

/** Décalage de l'infobulle par rapport au curseur, en px. */
const DECALAGE_INFOBULLE_PX = 12

export interface CoucheDesTueursProps {
  t: TacticalText
  locale: Locale
  repere: RepereTactique
  fenetre: FenetreDuPlan
  contributions: readonly TacticalContribution[]
  points: readonly PointDeLaCouche[]
  /** L'index de contribution mis en avant (tuile ou point survolé), `null` sinon. */
  survol: number | null
  onSurvol: (index: number | null) => void
  estompe: string
}

/** CoucheDesTueurs — la toile des traits et des points, et l'infobulle du point survolé. */
export function CoucheDesTueurs({
  t,
  locale,
  repere,
  fenetre,
  contributions,
  points,
  survol,
  onSurvol,
  estompe,
}: CoucheDesTueursProps) {
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const taille = useTailleDuCanvas(canvasRef)
  const paletteVersion = useColorPaletteVersion()
  // Où est le curseur sur le plan : l'infobulle ne suit que le survol d'un POINT, pas celui d'une tuile.
  const [curseur, setCurseur] = useState<{ x: number; y: number } | null>(null)

  useEffect(() => {
    const canvas = canvasRef.current
    if (!canvas) return
    const width = canvas.clientWidth
    const height = canvas.clientHeight
    if (width <= 0 || height <= 0) return
    canvas.width = width
    canvas.height = height
    const ctx = canvas.getContext('2d')
    if (!ctx) return
    ctx.clearRect(0, 0, width, height)
    const projection = projectionDuPlan(repere, width, fenetre)
    if (!projection) return
    void paletteVersion
    const roles = {
      tueur: resolveToken(JETON_DU_ROLE.tueur),
      victime: resolveToken(JETON_DU_ROLE.victime),
    }
    peindreLaCouche(ctx, points, projection, { roles, encre: getComputedStyle(canvas).color }, survol)
  }, [points, survol, repere, fenetre, taille, paletteVersion])

  useEffect(() => {
    const canvas = canvasRef.current
    const cadre = canvas?.parentElement
    if (!canvas || !cadre) return
    const surMouvement = (event: PointerEvent) => {
      const rect = canvas.getBoundingClientRect()
      const projection = projectionDuPlan(repere, canvas.clientWidth, fenetre)
      if (!projection) return
      const position = { x: event.clientX - rect.left, y: event.clientY - rect.top }
      const index = pointLePlusProche(points, position, projection)
      setCurseur(index === null ? null : position)
      onSurvol(index)
    }
    const surSortie = () => {
      setCurseur(null)
      onSurvol(null)
    }
    cadre.addEventListener('pointermove', surMouvement)
    cadre.addEventListener('pointerleave', surSortie)
    return () => {
      cadre.removeEventListener('pointermove', surMouvement)
      cadre.removeEventListener('pointerleave', surSortie)
    }
  }, [points, repere, fenetre, onSurvol])

  const survole = curseur && survol !== null ? contributions[survol] : undefined
  return (
    <>
      <canvas
        ref={canvasRef}
        aria-hidden
        className={`pointer-events-none absolute inset-0 h-full w-full text-foreground${estompe}`}
        data-testid="tactical-plan-tueurs"
      />
      {survole && curseur && (
        <span
          role="tooltip"
          className="pointer-events-none absolute z-[3] whitespace-nowrap rounded border border-border bg-card px-[5px] py-px text-[11px] leading-[14px] text-foreground"
          style={{ left: curseur.x + DECALAGE_INFOBULLE_PX, top: curseur.y + DECALAGE_INFOBULLE_PX }}
          data-testid="tactical-tueurs-tip"
        >
          {infobulleDuPoint(t, survole, locale)}
        </span>
      )}
    </>
  )
}

/** LegendeDesTueurs — une entrée par rôle présent, sous le plan (pastille de la couleur du rôle + mot). */
export function LegendeDesTueurs({ t, points }: { t: TacticalText; points: readonly PointDeLaCouche[] }) {
  const roles = rolesPresents(points)
  if (roles.length === 0) return null
  const libelle: Record<RoleDuPoint, string> = { tueur: t.layerKillers, victime: t.layerVictims }
  return (
    <ul className="mx-3 mb-3 -mt-1 flex flex-wrap gap-x-4 gap-y-1 text-[11.5px] text-muted-foreground" data-testid="tactical-tueurs-legend">
      {roles.map((r) => (
        <li key={r} className="flex items-center gap-1.5">
          <span aria-hidden className="inline-block size-2 rounded-full" style={{ backgroundColor: tokenCssVar(JETON_DU_ROLE[r]) }} />
          {libelle[r]}
        </li>
      ))}
    </ul>
  )
}
