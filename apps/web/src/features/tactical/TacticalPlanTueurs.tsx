/**
 * TacticalPlanTueurs — la couche « autre joueur » du plan et sa légende : tant qu'une zone est
 * sélectionnée, un trait relie chaque événement de la zone à l'autre joueur (le tueur d'une mort, la
 * victime d'un frag) et un point marque l'autre joueur. Aucun réglage : le clic sur la zone EST le
 * réglage.
 *
 * LA COUCHE A SA TOILE, superposée à celle de la chaleur et de même taille : le survol (tuile ou
 * point) la repeint seule, sans refaire la chaleur lissée. Elle partage la projection du plan
 * (`projectionDuPlan`), donc le cadrage et le zoom de la chaleur ; ses tailles suivent l'échelle du
 * plan (`dimensionsDeLaCouche`). La toile n'intercepte aucun geste ; le survol et le clic d'un point
 * se lisent sur le cadre qui la porte (écouteurs natifs : ceux de la toile de la chaleur — clic,
 * glisser, molette — ne sont pas touchés, sauf le clic sur un point, qui choisit sa tuile au lieu de
 * changer de zone).
 *
 * COULEURS : aucune teinte de la rampe de chaleur. Marqueurs et traits sont à l'encre de premier
 * plan du thème (`text-foreground`), cernés de la couleur du fond (`bg-background`), lues sur
 * deux éléments du DOM : elles suivent donc le thème clair ou sombre. Le tueur est un losange, la
 * victime un rond (`formeDuRole`) ; la légende dit les deux formes.
 */
import { useEffect, useRef, useState } from 'react'

import { useColorPaletteVersion } from '@/lib/accessibility/useColorPaletteVersion'
import type { TacticalContribution } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'

import type { TacticalText } from './i18n'
import { projectionDuPlan, type FenetreDuPlan, type RepereTactique } from './tacticalView.logic'
import {
  dimensionsDeLaCouche,
  formeDuRole,
  misEnAvant,
  pointLePlusProche,
  rolesPresents,
  SEUIL_CLIC_PX,
  type PointDeLaCouche,
  type RoleDuPoint,
} from './tueurs.logic'
import { peindreLaCouche } from './tueursPaint'
import { useTailleDuCanvas } from './useTailleDuCanvas'
import { infobulleDuPoint } from './zone.logic'

/** Décalage de l'infobulle par rapport au curseur, en px. */
const DECALAGE_INFOBULLE_PX = 12

export interface CoucheDesTueursProps {
  t: TacticalText
  locale: Locale
  repere: RepereTactique
  fenetre: FenetreDuPlan
  contributions: readonly TacticalContribution[]
  points: readonly PointDeLaCouche[]
  /** L'index de contribution survolé (tuile ou point), `null` sinon. */
  survol: number | null
  onSurvol: (index: number | null) => void
  /** L'index de contribution choisi (clic sur une tuile ou un point), `null` sinon. */
  choisie: number | null
  /** Un clic sur un point du plan : la vue bascule le choix de sa contribution. */
  onChoisir: (index: number) => void
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
  choisie,
  onChoisir,
  estompe,
}: CoucheDesTueursProps) {
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const fondRef = useRef<HTMLSpanElement>(null)
  const taille = useTailleDuCanvas(canvasRef)
  const paletteVersion = useColorPaletteVersion()
  // Où est le curseur sur le plan : l'infobulle ne suit que le survol d'un POINT, pas celui d'une tuile.
  const [curseur, setCurseur] = useState<{ x: number; y: number } | null>(null)
  const enAvant = misEnAvant(survol, choisie)
  // Les écouteurs natifs lisent les derniers rappels sans se réinstaller à chaque rendu.
  const rappels = useRef({ onSurvol, onChoisir })
  useEffect(() => {
    rappels.current = { onSurvol, onChoisir }
  })

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
    const dim = dimensionsDeLaCouche(width / (fenetre.maxX - fenetre.minX))
    const fond = fondRef.current ? getComputedStyle(fondRef.current).backgroundColor : ''
    peindreLaCouche(ctx, points, projection, { encre: getComputedStyle(canvas).color, fond }, dim, enAvant)
  }, [points, enAvant, repere, fenetre, taille, paletteVersion])

  useEffect(() => {
    const canvas = canvasRef.current
    const cadre = canvas?.parentElement
    if (!canvas || !cadre) return
    // Le point sous le curseur (px relatifs à la toile), à l'échelle courante du plan.
    const pointSous = (event: MouseEvent) => {
      const rect = canvas.getBoundingClientRect()
      const largeur = canvas.clientWidth
      const projection = projectionDuPlan(repere, largeur, fenetre)
      if (!projection) return null
      const position = { x: event.clientX - rect.left, y: event.clientY - rect.top }
      const dim = dimensionsDeLaCouche(largeur / (fenetre.maxX - fenetre.minX))
      return { position, index: pointLePlusProche(points, position, projection, dim.rayonSurvol) }
    }
    let appui: { x: number; y: number } | null = null
    const surAppui = (event: PointerEvent) => {
      appui = { x: event.clientX, y: event.clientY }
    }
    const surMouvement = (event: PointerEvent) => {
      const sous = pointSous(event)
      if (!sous) return
      setCurseur(sous.index === null ? null : sous.position)
      rappels.current.onSurvol(sous.index)
    }
    const surSortie = () => {
      setCurseur(null)
      rappels.current.onSurvol(null)
    }
    // Phase de capture : un clic sur un point choisit sa tuile et n'atteint jamais la toile de la
    // chaleur (qui changerait de zone). Un glisser n'est pas un clic.
    const surClic = (event: MouseEvent) => {
      const depart = appui
      appui = null
      if (depart && Math.hypot(event.clientX - depart.x, event.clientY - depart.y) > SEUIL_CLIC_PX) return
      const sous = pointSous(event)
      if (!sous || sous.index === null) return
      event.stopPropagation()
      rappels.current.onChoisir(sous.index)
    }
    cadre.addEventListener('pointerdown', surAppui, true)
    cadre.addEventListener('pointermove', surMouvement)
    cadre.addEventListener('pointerleave', surSortie)
    cadre.addEventListener('click', surClic, true)
    return () => {
      cadre.removeEventListener('pointerdown', surAppui, true)
      cadre.removeEventListener('pointermove', surMouvement)
      cadre.removeEventListener('pointerleave', surSortie)
      cadre.removeEventListener('click', surClic, true)
    }
  }, [points, repere, fenetre])

  const survole = curseur && survol !== null ? contributions[survol] : undefined
  return (
    <>
      {/* Porte la couleur du fond du thème, lue pour le liseré des marqueurs. */}
      <span ref={fondRef} aria-hidden className="hidden bg-background" />
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
          <span
            aria-hidden
            className={`inline-block size-2 bg-foreground ${formeDuRole(r) === 'rond' ? 'rounded-full' : 'rotate-45'}`}
            data-forme={formeDuRole(r)}
          />
          {libelle[r]}
        </li>
      ))}
    </ul>
  )
}
