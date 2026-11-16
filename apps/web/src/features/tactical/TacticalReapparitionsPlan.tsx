/**
 * TacticalReapparitionsPlan — la carte du plan de la lecture « Réapparitions adverses » : bandeau (nom de la
 * carte, aide ⓘ, pilules « Lecture » et « Zone tenue »), fond de carte, couche des zones teintées
 * (`TacticalReapparitionsZones`), noms et parts des zones principales, et légende en bas.
 *
 * LE FOND ET SON CADRAGE SONT CEUX DE LA CARTE DU PLAN (`TacticalPlanFond`, `cadrageDuFond`, orientation
 * du rejeu 2D) : la lecture remplace le calque de chaleur, pas le cadre. Pas de zoom ici : la couche se
 * lit sur la carte entière.
 *
 * L'ÉTAT VIDE : le plan reste, les zones passent en gris (répartition sans condition) et une note pointillée
 * dit ce qui manque ; l'explication complète et les jauges sont dans la colonne de droite.
 */
import { useMemo, useRef, type ReactNode } from 'react'

import { SectionCard } from '@/components/ui/section-card'
import { titleWithInfo } from '@/components/ui/title-with-info'
import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'
import type { TacticalRaster } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'
import type { CalloutZoneReady } from '@/lib/replay/calloutsPaint'

import { useMapViewPreferences } from '@/features/match-replay/settings/useMapViewPreferences'

import type { TacticalText } from './i18n'
import { cadrageDuFond, MARGE_LEGENDE, type EtatDuPlan } from './plan.logic'
import { PILULE, ETIQUETTE } from './pilules'
import { useTacticalMapBackgroundFrame } from './queries'
import type { PlanDuRognage } from './queriesReapparitions'
import {
  cadreDeLaCouche,
  formaterLaPart,
  lignesEtiquetees,
  positionDeLEtiquette,
  type LectureReapparitions,
  type ZoneProposee,
} from './reapparitions.logic'
import { PuceDEcart } from './TacticalReapparitionsPuce'
import { TacticalReapparitionsZones } from './TacticalReapparitionsZones'
import { AvisSurLeFond, IndicateurDeLecture } from './TacticalPlanAvis'
import { TacticalPlanFond } from './TacticalPlanFond'
import { aspectDuPlan, ESTOMPE } from './tacticalLecture.logic'
import { repereDuPlan } from './tacticalView.logic'
import { useCadrageDuPlan } from './useCadrageDuPlan'
import { useHauteurDuPlan } from './useHauteurDuPlan'

export interface TacticalReapparitionsPlanProps {
  t: TacticalText
  locale: Locale
  playerSlug: string
  mapId: string
  /** Le nom de la carte affichée ; '' = aucun titre. */
  titre: string
  /** La lecture de placement qui porte les zones nommées, les bornes et le calage (courante ou précédente). */
  lecture: TacticalRaster | undefined
  etat: EtatDuPlan
  inconnus: string[] | null
  /** La lecture des réapparitions adverses ; `null` tant que la réponse n'est pas là. */
  modele: LectureReapparitions | null
  /** Les zones nommées de la carte, normalisées une fois par la vue (`normalizeCalloutZones`). */
  zones: readonly CalloutZoneReady[]
  /** La pilule « Lecture » (partagée avec la carte du plan). */
  pilule: ReactNode
  zonesProposees: readonly ZoneProposee[]
  choisie: number | null
  onChoisir: (volumeIndex: number | null) => void
  plan: PlanDuRognage
}

export function TacticalReapparitionsPlan({
  t,
  locale,
  playerSlug,
  mapId,
  titre,
  lecture,
  etat,
  inconnus,
  modele,
  zones,
  pilule,
  zonesProposees,
  choisie,
  onChoisir,
  plan,
}: TacticalReapparitionsPlanProps) {
  const cadreFond = useTacticalMapBackgroundFrame(playerSlug, mapId)
  const { turns } = useMapViewPreferences(mapId)
  const lue = etat === 'pret' || etat === 'relecture' ? lecture : undefined
  const repere = useMemo(() => (lue ? repereDuPlan(cadreFond, lue.bornes, lue.pas_m, turns) : null), [cadreFond, lue, turns])
  // La scène et la fenêtre du plan (1x : la carte entière) : le MÊME cadrage que la carte de placement.
  const { scene, fenetre } = useCadrageDuPlan(repere, cadreFond, turns)
  const cadrage = cadrageDuFond(scene, fenetre, turns)
  const cadre = useMemo(() => cadreDeLaCouche(cadreFond, lue?.bornes), [cadreFond, lue?.bornes])
  const carteRef = useRef<HTMLDivElement>(null)
  const cadreRef = useRef<HTMLDivElement>(null)
  const hauteurMax = useHauteurDuPlan(carteRef, cadreRef)
  const estompe = etat === 'relecture' ? ` ${ESTOMPE}` : ''
  const surLePlan = !!(repere && cadre && zones.length > 0 && etat !== 'echec')
  const info = modele
    ? t.reappPlanInfo(modele.plancher.joueursPourTenir, modele.plancher.matchs, modele.plancher.reapparitions)
    : null

  return (
    <div ref={carteRef} className="min-w-0" data-testid="tactical-reapp-plan">
      <SectionCard
        title={titre}
        label={titre || undefined}
        titleAdornment={titleWithInfo(info, {
          trailing: (
            <span className="flex flex-wrap items-center gap-2" data-testid="tactical-plan-pills">
              {pilule}
              <PiluleZoneTenue t={t} zones={zonesProposees} choisie={choisie} onChoisir={onChoisir} />
              {modele && (
                <span className="whitespace-nowrap text-[11.5px] font-normal text-muted-foreground">
                  {t.reappMeta(modele.reference.matchs, modele.reference.reapparitions)}
                </span>
              )}
            </span>
          ),
          testId: 'tactical-plan-title',
        })}
        footer={modele && <LegendeDuPlan t={t} locale={locale} modele={modele} />}
      >
        <div className="relative mx-3 mb-3 mt-2.5 flex justify-center" style={{ paddingRight: MARGE_LEGENDE }}>
          <TacticalPlanFond
            playerSlug={playerSlug}
            mapId={mapId}
            aspect={aspectDuPlan(cadreFond, repere, turns)}
            hauteurMax={hauteurMax}
            cadreRef={cadreRef}
            cadrage={cadrage}
          >
            {surLePlan && repere && cadre && (
              <>
                <TacticalReapparitionsZones
                  zones={zones}
                  lecture={modele}
                  cadre={cadre}
                  cadrage={cadrage}
                  plan={plan}
                  locale={locale}
                  choisie={choisie}
                  onChoisir={(volume) => onChoisir(volume === choisie ? null : volume)}
                  estompe={estompe}
                />
                <EtiquettesDeZones t={t} locale={locale} zones={zones} modele={modele} repere={repere} estompe={estompe} />
              </>
            )}
            <AvisSurLeFond t={t} etat={etat} inconnus={inconnus} estompe={estompe}>
              {null}
            </AvisSurLeFond>
            <IndicateurDeLecture t={t} etat={etat} />
            {modele?.etat === 'vide' && modele.zone && (
              <p
                className="pointer-events-none absolute bottom-2.5 left-1/2 max-w-[calc(100%-20px)] -translate-x-1/2 rounded-lg border-[1.5px] border-dashed border-placeholder-border bg-card/90 px-2.5 py-1.5 text-center text-[11.5px] text-muted-foreground"
                data-testid="tactical-reapp-plan-vide"
              >
                {t.reappPlanEmpty(modele.zone.nom)}
              </p>
            )}
          </TacticalPlanFond>
        </div>
      </SectionCard>
    </div>
  )
}

/** PiluleZoneTenue — la pilule « Zone tenue » : le choix de la zone, au clavier comme à la souris. */
function PiluleZoneTenue({
  t,
  zones,
  choisie,
  onChoisir,
}: {
  t: TacticalText
  zones: readonly ZoneProposee[]
  choisie: number | null
  onChoisir: (volumeIndex: number | null) => void
}) {
  return (
    <label className={PILULE}>
      <span className={ETIQUETTE}>{t.reappPillHeld}</span>
      <select
        value={choisie === null ? '' : String(choisie)}
        onChange={(e) => onChoisir(e.target.value === '' ? null : Number(e.target.value))}
        aria-label={t.reappPillHeld}
        className="w-[150px] bg-transparent pl-1.5 pr-0.5 text-foreground"
        data-testid="tactical-reapp-zone-select"
      >
        <option value="">{t.reappHeldNone}</option>
        {zones.map((z) => (
          <option key={z.volumeIndex} value={z.volumeIndex}>
            {z.nom}
          </option>
        ))}
      </select>
    </label>
  )
}

/**
 * EtiquettesDeZones — le nom et la part des zones principales (part pendant la tenue au-dessus du seuil) et le
 * nom de la zone tenue, en HTML par-dessus le plan : ils ne tournent pas avec la carte et restent lisibles
 * au bord du plan (jamais rognés).
 */
function EtiquettesDeZones({
  t,
  locale,
  zones,
  modele,
  repere,
  estompe,
}: {
  t: TacticalText
  locale: Locale
  zones: readonly { volumeIndex: number; labelX: number; labelY: number }[]
  modele: LectureReapparitions | null
  repere: NonNullable<ReturnType<typeof repereDuPlan>>
  estompe: string
}) {
  if (!modele) return null
  const ancre = (volume: number) => zones.find((z) => z.volumeIndex === volume)
  const style = (volume: number) => {
    const z = ancre(volume)
    return z ? positionDeLEtiquette({ x: z.labelX, y: z.labelY }, repere) : null
  }
  const classe =
    'absolute -translate-x-1/2 -translate-y-1/2 inline-flex items-center gap-1.5 whitespace-nowrap rounded-md border border-border bg-card/90 px-1.5 py-1 text-[11px] leading-none text-foreground shadow-sm'
  const tenue = modele.zone && style(modele.zone.volumeIndex)
  return (
    <div className={`pointer-events-none absolute inset-0${estompe}`} data-testid="tactical-reapp-etiquettes">
      {lignesEtiquetees(modele).map((l) => {
        const position = style(l.volumeIndex)
        return (
          position && (
            <span key={l.volumeIndex} className={classe} style={position} data-testid={`tactical-reapp-etiquette-${l.volumeIndex}`}>
              {l.nom}
              <b className="font-bold">{formaterLaPart(l.tenue ?? 0, locale)}</b>
              {l.ecart !== null && <PuceDEcart t={t} ecart={l.ecart} />}
            </span>
          )
        )
      })}
      {modele.zone && tenue && (
        <span
          className={`${classe} font-medium`}
          style={{ ...tenue, borderColor: tokenCssVar('team-ally') }}
          data-testid="tactical-reapp-etiquette-tenue"
        >
          <i className="h-2 w-2 rounded-sm" style={{ backgroundColor: tokenCssVar('team-ally') }} aria-hidden />
          {t.reappLabelHeld(modele.zone.nom)}
        </span>
      )}
    </div>
  )
}

/** LegendeDuPlan — la légende en bas du plan : dégradé de teinte, zone tenue hachurée, puce d'écart. */
function LegendeDuPlan({ t, locale, modele }: { t: TacticalText; locale: Locale; modele: LectureReapparitions }) {
  const ennemi = tokenCssVar('team-enemy')
  const allie = tokenCssVar('team-ally')
  const grise = modele.etat !== 'servie'
  const max = formaterLaPart(modele.maxTenue, locale)
  const degrade = grise
    ? 'linear-gradient(90deg, color-mix(in oklch, currentColor 8%, transparent), color-mix(in oklch, currentColor 45%, transparent))'
    : `linear-gradient(90deg, color-mix(in oklch, ${ennemi} 10%, transparent), ${ennemi})`
  let libelle = t.reappLegendGradient(max)
  if (modele.etat === 'vide') libelle = t.reappLegendGrey
  else if (modele.etat === 'sans_zone') libelle = t.reappLegendGreyNoZone
  return (
    <div
      className="flex flex-wrap gap-x-4 gap-y-1.5 border-t border-border px-3 py-2.5 text-[11.5px] text-muted-foreground"
      data-testid="tactical-reapp-legende"
    >
      <span className="inline-flex items-center gap-1.5">
        <i className="h-2 w-14 rounded-sm text-muted-foreground" style={{ background: degrade }} aria-hidden />
        {libelle}
      </span>
      {modele.zone && (
        <span className="inline-flex items-center gap-1.5">
          <i
            className="h-3.5 w-3.5 rounded-[3px] border-[1.5px]"
            style={{
              borderColor: allie,
              background: `repeating-linear-gradient(45deg, color-mix(in oklch, ${allie} 55%, transparent) 0 2px, color-mix(in oklch, ${allie} 12%, transparent) 2px 6px)`,
            }}
            aria-hidden
          />
          {t.reappLegendHeld(modele.plancher.joueursPourTenir)}
        </span>
      )}
      {modele.etat === 'servie' && (
        <span className="inline-flex items-center gap-1.5">
          <span aria-hidden>
            <PuceDEcart t={t} ecart={5} />
          </span>
          {t.reappLegendDelta}
        </span>
      )}
    </div>
  )
}
