/**
 * TacticalReapparitionsView — le centre et la droite du cockpit pour la lecture « Réapparitions adverses » :
 * le plan annoté (zones teintées selon la part des réapparitions adverses, zone tenue hachurée) et, à
 * sa droite, les barres comparées. Elle remplace `TacticalPlanCard` + `TacticalZoneCard` tant que la pilule
 * « Lecture » est sur cette lecture.
 *
 * LA LECTURE DE PLACEMENT RESTE LA SOURCE DU CATALOGUE : les zones nommées, les bornes et le calage viennent
 * de `useTacticalRaster` (déjà servie par la vue d'analyse, sur le même périmètre). La lecture des
 * réapparitions n'y ajoute qu'une requête, par zone tenue (`useReapparitionsAdverses`).
 *
 * LA ZONE TENUE se choisit d'un clic sur le plan ou dans la pilule ; un re-clic la retire. Tant qu'aucune
 * n'est choisie, la répartition sans condition se lit par une zone de SONDE (`volumeDeSonde`) : la référence
 * ne dépend pas de la zone, et le plan montre déjà où l'adversaire réapparaît.
 */
import { useMemo, useState, type ReactNode } from 'react'

import type { TacticalRaster } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'
import { normalizeCalloutZones } from '@/lib/replay/calloutsPaint'

import { carteLue, type CarteEffective } from './cockpit.logic'
import type { TacticalText } from './i18n'
import type { EtatDuPlan } from './plan.logic'
import { usePlanDuRognage, useReapparitionsAdverses } from './queriesReapparitions'
import { construireLaLecture, volumeDeSonde, zonesProposees } from './reapparitions.logic'
import { TacticalReapparitionsColonne } from './TacticalReapparitionsColonne'
import { TacticalReapparitionsPlan } from './TacticalReapparitionsPlan'
import { ESTOMPE } from './tacticalLecture.logic'

export interface TacticalReapparitionsViewProps {
  playerSlug: string
  carte: CarteEffective
  mapName: string
  locale: Locale
  t: TacticalText
  /** Périmètre résolu par la barre L2 — le même que celui de la lecture de placement. `null` = pas encore. */
  matchIds: string[] | null
  coequipiers: string[]
  /** La lecture de placement affichée (zones, bornes) et l'état du plan qui en découle. */
  lecture: TacticalRaster | undefined
  etat: EtatDuPlan
  inconnus: string[] | null
  /** La pilule « Lecture », construite par la vue d'analyse (une seule liste de lectures). */
  pilule: ReactNode
}

export function TacticalReapparitionsView({
  playerSlug,
  carte,
  mapName,
  locale,
  t,
  matchIds,
  coequipiers,
  lecture,
  etat,
  inconnus,
  pilule,
}: TacticalReapparitionsViewProps) {
  const [choisie, setChoisie] = useState<number | null>(null)
  const lue = etat === 'pret' || etat === 'relecture' ? lecture : undefined
  const zones = useMemo(() => normalizeCalloutZones(lue?.zones), [lue?.zones])
  const propositions = useMemo(() => zonesProposees(zones, locale), [zones, locale])
  const sonde = volumeDeSonde(zones)
  const volume = choisie ?? sonde
  const requete = useReapparitionsAdverses(playerSlug, carte.mapId, {
    match_ids: carteLue(carte) ? matchIds : null,
    coequipiers,
    volume_index: volume,
  })
  const modele = useMemo(() => construireLaLecture(requete.data, choisie), [requete.data, choisie])
  const plan = usePlanDuRognage(playerSlug, carte.mapId)
  const estompe = etat === 'relecture' ? ` ${ESTOMPE}` : ''
  const sansZones = etat === 'pret' && sonde === null

  return (
    <div
      className="grid min-w-0 grid-cols-1 items-start gap-3 lg:grid-cols-[minmax(0,1fr)_var(--tac-zone-l)]"
      aria-busy={etat === 'attente' || etat === 'relecture'}
      data-testid="tactical-analysis-body"
    >
      <TacticalReapparitionsPlan
        t={t}
        locale={locale}
        playerSlug={playerSlug}
        mapId={carte.mapId}
        titre={mapName}
        lecture={lecture}
        etat={etat}
        inconnus={inconnus}
        modele={modele}
        zones={zones}
        pilule={pilule}
        zonesProposees={propositions}
        choisie={choisie}
        onChoisir={setChoisie}
        plan={plan}
      />
      <TacticalReapparitionsColonne
        t={t}
        locale={locale}
        modele={modele}
        enAttente={volume !== null ? requete.isPending : etat === 'attente'}
        enErreur={requete.isError}
        sansZones={sansZones}
        estompe={estompe}
      />
    </div>
  )
}
