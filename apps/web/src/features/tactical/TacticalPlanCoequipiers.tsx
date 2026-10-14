/**
 * TacticalPlanCoequipiers — ce que la lecture des coéquipiers pose SUR LE PLAN PRINCIPAL (amendement
 * A1 : même fond, même projection, même cadrage et même zoom que la chaleur et la couche des tueurs) :
 *
 *   - `CoucheDesCoequipiers` : la toile de la scène (nuage de la zone, instantané d'une mort, étape de
 *     la séquence) et, sur l'instantané, la bulle de l'échange (« Tueur abattu 1,8 s après la mort ») ;
 *   - `PiedDeLaMort` : la légende par forme et la barre des étapes de la séquence (D), rendue
 *     seulement quand la séquence est servie.
 *
 * La toile n'intercepte aucun geste. Couleurs : jetons d'équipe de l'app (`team-ally`, `team-enemy`),
 * encre et fond du thème lus sur des éléments du DOM — ils suivent le thème et les couleurs d'équipe
 * choisies. Aucune teinte de la rampe de chaleur.
 */
import { useEffect, useRef } from 'react'

import { resolveToken } from '@/lib/accessibility/resolveToken'
import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'
import { useColorPaletteVersion } from '@/lib/accessibility/useColorPaletteVersion'
import type { Locale } from '@/lib/i18n/locale'

import { peindreLaScene } from './coequipiersPaint'
import type { TacticalText } from './i18n'
import {
  echangeDeLaMort,
  libelleDeDecalage,
  secondesDeFenetre,
  type SceneDuPlan,
} from './mort.logic'
import { projectionDuPlan, type FenetreDuPlan, type RepereTactique } from './tacticalView.logic'
import { dimensionsDeLaCouche } from './tueurs.logic'
import { useTailleDuCanvas } from './useTailleDuCanvas'
import { entreesDeLegende, modeDeLaScene, type FormeDeLegende, type VueDeMort } from './vueDeMort'
import { decimale } from './zone.logic'

interface CoucheDesCoequipiersProps {
  t: TacticalText
  locale: Locale
  repere: RepereTactique
  fenetre: FenetreDuPlan
  scene: SceneDuPlan
  /** La mort choisie, pour la bulle de l'échange ; `null` = vue de la zone. */
  vue: VueDeMort | null
  estompe: string
}

/** CoucheDesCoequipiers — la toile de la scène et la bulle de l'échange. */
export function CoucheDesCoequipiers({ t, locale, repere, fenetre, scene, vue, estompe }: CoucheDesCoequipiersProps) {
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const fondRef = useRef<HTMLSpanElement>(null)
  const taille = useTailleDuCanvas(canvasRef)
  const paletteVersion = useColorPaletteVersion()

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
    const echelle = width / (fenetre.maxX - fenetre.minX)
    const fond = fondRef.current ? getComputedStyle(fondRef.current).backgroundColor : ''
    peindreLaScene(
      ctx,
      scene,
      projection,
      echelle,
      {
        encre: getComputedStyle(canvas).color,
        fond,
        allie: resolveToken('team-ally'),
        adverse: resolveToken('team-enemy'),
      },
      dimensionsDeLaCouche(echelle),
    )
  }, [scene, repere, fenetre, taille, paletteVersion])

  const echange = vue && modeDeLaScene(scene, vue) === 'instantane' ? echangeDeLaMort(t, vue.contribution, locale) : null
  return (
    <>
      <span ref={fondRef} aria-hidden className="hidden bg-background" />
      <canvas
        ref={canvasRef}
        aria-hidden
        className={`pointer-events-none absolute inset-0 h-full w-full text-foreground${estompe}`}
        data-testid="tactical-plan-coequipiers"
      />
      {echange && (
        <div
          className="pointer-events-none absolute bottom-3 left-3 z-[3] flex max-w-[70%] flex-col gap-0.5 rounded-lg border border-border bg-card/90 px-3 py-2 text-xs"
          data-testid="tactical-echange"
        >
          <span className="font-semibold text-foreground">{echange.titre}</span>
          {echange.detail && <span className="text-muted-foreground">{echange.detail}</span>}
        </div>
      )}
    </>
  )
}


function Glyphe({ forme }: { forme: FormeDeLegende }) {
  const allie = { backgroundColor: tokenCssVar('team-ally') }
  switch (forme) {
    case 'mort':
      return <span aria-hidden className="inline-block size-3 rounded-full border-2 border-foreground bg-background" data-forme="mort" />
    case 'tueur':
      return <span aria-hidden className="inline-block size-2 rotate-45" style={{ backgroundColor: tokenCssVar('team-enemy') }} data-forme="losange" />
    case 'victime':
      return <span aria-hidden className="inline-block size-2 rounded-full" style={{ backgroundColor: tokenCssVar('team-enemy') }} data-forme="rond" />
    case 'plein':
      return <span aria-hidden className="inline-block size-2.5 rounded-full" style={allie} data-forme="plein" />
    case 'creux':
      return <span aria-hidden className="inline-block size-2.5 rounded-full border-2 bg-transparent" style={{ borderColor: tokenCssVar('team-ally') }} data-forme="creux" />
    case 'nuage':
      return <span aria-hidden className="inline-block size-1.5 rounded-full opacity-60" style={allie} data-forme="nuage" />
    case 'cercle':
      return <span aria-hidden className="inline-block w-4 border-t-[1.5px] border-dashed border-foreground/60" data-forme="cercle" />
    case 'trait':
      return <span aria-hidden className="inline-block h-0.5 w-4" style={allie} data-forme="trait" />
  }
}

interface PiedDuPlanProps {
  /** Les rôles que la couche des tueurs et victimes peint (une entrée de légende chacun). */
  roles: readonly ('tueur' | 'victime')[]
  t: TacticalText
  locale: Locale
  scene: SceneDuPlan
  vue: VueDeMort | null
}

/** PiedDuPlan — LA légende du plan (une forme = une entrée, les deux couches ensemble), puis la barre des étapes quand la séquence est servie. */
export function PiedDuPlan({ t, locale, scene, vue, roles }: PiedDuPlanProps) {
  const mode = modeDeLaScene(scene, vue)
  const entrees = entreesDeLegende(t, mode, scene, locale, roles)
  const seq = vue?.sequence ?? null
  if (entrees.length === 0 && !seq) return null
  return (
    <div className="mx-3 mb-3 -mt-1 flex flex-col gap-2" data-testid="tactical-plan-pied">
      {entrees.length > 0 && (
        <ul className="flex flex-wrap gap-x-4 gap-y-1 text-[11.5px] text-muted-foreground" data-testid="tactical-plan-formes">
          {entrees.map((e) => (
            <li key={e.forme} className="flex items-center gap-1.5">
              <Glyphe forme={e.forme} />
              {e.texte}
            </li>
          ))}
          {mode === 'sequence' && seq && <li className="ml-auto">{t.seqTrail(secondesDeFenetre(seq.traine_ms, locale))}</li>}
        </ul>
      )}
      {vue && seq && <BarreDEtapes t={t} locale={locale} vue={vue} />}
    </div>
  )
}

const BOUTON =
  'min-h-8 rounded-md border px-1.5 py-1 font-mono text-[11px] tabular-nums transition-colors focus-visible:outline focus-visible:outline-2 focus-visible:outline-ring'

/** BarreDEtapes — « Instantané » puis une étape par seconde, de −4 s à +3 s ; la traîne en note. */
function BarreDEtapes({ t, locale, vue }: { t: TacticalText; locale: Locale; vue: VueDeMort }) {
  const etapes = vue.sequence?.etapes ?? []
  return (
    <div role="group" aria-label={t.seqGroup} className="flex flex-wrap items-center gap-1" data-testid="tactical-sequence-etapes">
      <button
        type="button"
        aria-pressed={vue.etape === null}
        onClick={() => vue.onEtape(null)}
        className={`${BOUTON} ${vue.etape === null ? 'border-foreground bg-foreground text-background' : 'border-border bg-card text-muted-foreground hover:bg-accent'}`}
      >
        {t.seqSnapshot}
      </button>
      {etapes.map((e, i) => (
        <button
          key={e.offset_ms}
          type="button"
          aria-pressed={vue.etape === i}
          aria-label={t.seqStage(decimale(e.offset_ms / 1000, locale))}
          onClick={() => vue.onEtape(i)}
          className={`${BOUTON} ${vue.etape === i ? 'border-foreground bg-foreground text-background' : e.offset_ms === 0 ? 'border-foreground bg-card text-foreground' : 'border-border bg-card text-muted-foreground hover:bg-accent'}`}
          data-testid="tactical-sequence-etape"
        >
          {libelleDeDecalage(e.offset_ms, locale)}
        </button>
      ))}
    </div>
  )
}
