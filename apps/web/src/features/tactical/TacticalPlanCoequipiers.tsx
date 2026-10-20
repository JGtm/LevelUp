/**
 * TacticalPlanCoequipiers — ce que la lecture des coéquipiers pose SUR LE PLAN PRINCIPAL (amendement
 * A1 : même fond, même projection, même cadrage et même zoom que la chaleur et la couche des tueurs) :
 *
 *   - `CoucheDesCoequipiers` : la toile de la scène (nuage de la zone, instantané d'une mort, étape ou
 *     instant de la séquence en lecture) ;
 *   - `PiedDuPlan` : la légende par forme et la zone « séquence » (lecteur, ou état qui dit pourquoi
 *     elle manque — `TacticalSequenceSlot`).
 *
 * La toile n'intercepte aucun geste. Couleurs : jetons d'équipe de l'app (`team-ally`, `team-enemy`),
 * encre et fond du thème lus sur des éléments du DOM — ils suivent le thème et les couleurs d'équipe
 * choisies. Aucune teinte de la rampe de chaleur.
 *
 * PENDANT LA LECTURE (`lecture`), la scène n'est pas une prop : la toile s'abonne à la tête de lecture
 * (`useSyncExternalStore`) et repeint la scène de l'instant, sans repeindre le reste du plan.
 */
import { useEffect, useMemo, useRef, useSyncExternalStore } from 'react'

import { resolveToken } from '@/lib/accessibility/resolveToken'
import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'
import { useColorPaletteVersion } from '@/lib/accessibility/useColorPaletteVersion'
import type { TacticalSequence } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'

import { peindreLaScene } from './coequipiersPaint'
import type { TacticalText } from './i18n'
import { secondesDeTraine, type SceneDuPlan } from './mort.logic'
import type { EtatDeLaSequence } from './sequenceCause.logic'
import { sceneALInstant } from './sequenceLecture.logic'
import { projectionDuPlan, type FenetreDuPlan, type RepereTactique } from './tacticalView.logic'
import { ZoneSequence } from './TacticalSequenceSlot'
import { dimensionsDeLaCouche } from './tueurs.logic'
import type { LectureDeSequence } from './useLectureDeSequence'
import { useTailleDuCanvas } from './useTailleDuCanvas'
import { entreesDeLegende, modeDeLaScene, type FormeDeLegende, type VueDeMort } from './vueDeMort'

/** La lecture animée que la toile suit : la séquence et sa tête de lecture. */
export interface LectureDeLaToile {
  sequence: TacticalSequence
  lecture: LectureDeSequence
}

interface CoucheDesCoequipiersProps {
  repere: RepereTactique
  fenetre: FenetreDuPlan
  scene: SceneDuPlan
  estompe: string
  /** Quand elle est posée, la scène est celle de l'instant de la tête de lecture, et `scene` n'est pas peinte. */
  lecture?: LectureDeLaToile | null
}

const SANS_ABONNEMENT = () => () => {}

/** CoucheDesCoequipiers — la toile de la scène. */
export function CoucheDesCoequipiers({ repere, fenetre, scene: sceneFixe, estompe, lecture = null }: CoucheDesCoequipiersProps) {
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const fondRef = useRef<HTMLSpanElement>(null)
  const taille = useTailleDuCanvas(canvasRef)
  const paletteVersion = useColorPaletteVersion()
  const instant = useSyncExternalStore(lecture?.lecture.horloge.abonner ?? SANS_ABONNEMENT, lecture?.lecture.horloge.lire ?? (() => 0))
  const scene = useMemo(
    () => (lecture ? sceneALInstant(lecture.sequence, instant) : sceneFixe),
    [lecture, instant, sceneFixe],
  )

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

  return (
    <>
      <span ref={fondRef} aria-hidden className="hidden bg-background" />
      <canvas
        ref={canvasRef}
        aria-hidden
        className={`pointer-events-none absolute inset-0 h-full w-full text-foreground${estompe}`}
        data-testid="tactical-plan-coequipiers"
      />
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
  }
}

interface PiedDuPlanProps {
  /** Les rôles que la couche des tueurs et victimes peint (une entrée de légende chacun). */
  roles: readonly ('tueur' | 'victime')[]
  t: TacticalText
  locale: Locale
  scene: SceneDuPlan
  vue: VueDeMort | null
  /** L'état de la zone « séquence » de la contribution choisie ; `null` = rien de choisi ou fonction fermée. */
  etatSequence?: EtatDeLaSequence | null
}

/**
 * PiedDuPlan — LA légende du plan (une forme = une entrée, les deux couches ensemble), puis la zone
 * « séquence » : le lecteur quand la séquence est servie, sinon l'état qui dit pourquoi elle manque.
 */
export function PiedDuPlan({ t, locale, scene, vue, roles, etatSequence = null }: PiedDuPlanProps) {
  const mode = modeDeLaScene(scene, vue)
  const entrees = entreesDeLegende(t, mode, scene, locale, roles)
  const seq = vue?.sequence ?? null
  const avecZone = seq !== null || (etatSequence !== null && etatSequence.kind !== 'servie')
  if (entrees.length === 0 && !avecZone) return null
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
          {mode === 'sequence' && seq && <li className="ml-auto">{t.seqTrail(secondesDeTraine(seq.traine_ms, locale))}</li>}
        </ul>
      )}
      {avecZone && <ZoneSequence t={t} locale={locale} vue={vue} etat={etatSequence} />}
    </div>
  )
}
