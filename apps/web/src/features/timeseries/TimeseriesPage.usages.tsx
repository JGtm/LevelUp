/**
 * TimeseriesPage — onglet "Usages".
 *
 * UN AXE DE LECTURE : TOUT CE QUI VIENT DU FILM DÉCODÉ. Les trois sections réunies ici
 * étaient dispersées entre la Synthèse (« Portée des engagements ») et la fin de la
 * Progression (« Usages d'équipement », « Les formes retenues ») ; elles répondent pourtant
 * à la même question — ce que le film dit de la manière de jouer — et n'ont rien à voir avec
 * la courbe de progression d'un indicateur. Regroupées, elles se lisent ensemble et les deux
 * autres onglets retrouvent leur axe.
 *
 * AUCUNE REQUÊTE NEUVE : les trois blocs (`weapon_range`, `equipment_usage`,
 * `formes_retenues`) arrivent avec la MÊME réponse de page que les autres onglets.
 *
 * CHAQUE SECTION SE RETIRE D'ELLE-MÊME quand son bloc est absent (film non décodé, titre
 * sans la capability `weapon_range`). Les trois retirées, l'onglet DIT pourquoi il est vide
 * au lieu de rester muet : un onglet sans contenu ni message se lit « bug ».
 */
import { DetailSection } from '@/components/ui/detail-section'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import { EquipmentUsageSection } from '@/features/_shared/usage/EquipmentUsageSection'
import { usageAvailability } from '@/features/_shared/usage/usageAvailability'
import { USAGE_TEXT } from '@/features/_shared/usage/usageI18n'
import { FormesRetenuesSection } from '@/features/squad/formes/FormesRetenuesSection'
import { useCapability } from '@/lib/capabilities/capabilities'
import type { TimeseriesPageResponse } from '@/lib/api/types'
import type { TimeseriesManifestKey } from '@/lib/i18n/generated/timeseries'
import type { Locale } from '@/lib/i18n/locale'

import { WeaponRangeSection } from './WeaponRangeSection'

export interface TimeseriesUsagesTabProps {
  data: TimeseriesPageResponse
  locale: Locale
  t: (key: TimeseriesManifestKey) => string
}

export function TimeseriesUsagesTab({ data, locale, t }: TimeseriesUsagesTabProps) {
  // Portée et dénivelé mesurés des engagements : capability PRODUIT `weapon_range`
  // (title.CapWeaponRange). Halo 5 ne la déclare pas — ses événements de frag n'ont pas
  // d'arme, la jointure mesurée rendrait zéro ligne et la section serait vide.
  const hasWeaponRange = useCapability('weapon_range')
  const usageText = USAGE_TEXT[locale]

  // LES MÊMES PRÉDICATS QUE LES SECTIONS, relus ICI pour savoir EN AMONT si l'onglet a
  // quelque chose à montrer. Chaque section décide seule de se retirer (elle rend `null`) :
  // sans cette lecture en amont, l'onglet ne pourrait pas distinguer « trois sections
  // masquées » de « trois sections montées » et n'afficherait jamais son état vide.
  const showsRange = hasWeaponRange && data.weapon_range != null
  const showsEquipment =
    data.equipment_usage != null &&
    usageAvailability(data.equipment_usage, usageText).kind !== 'hidden'
  const showsFormes = data.formes_retenues != null && data.formes_retenues.available
  const hasAnything = showsRange || showsEquipment || showsFormes

  if (!hasAnything) {
    return (
      <EmptyStateNotice
        title={t('timeseries.usages.empty_title')}
        description={t('timeseries.usages.empty_description')}
      />
    )
  }

  return (
    <div className="space-y-8">
      {/* Portée des engagements — la section porte son propre titre standard. */}
      {showsRange && <WeaponRangeSection range={data.weapon_range} />}

      {/* Usages d'équipement — la section ne monte que des cartes, sans titre à elle : le
          titre de section est posé ici, et il coiffe bien plusieurs cartes (comptes, part du
          lobby, armes spéciales, niveaux). */}
      {showsEquipment && (
        <DetailSection title={t('timeseries.usages.equipment_title')}>
          <EquipmentUsageSection
            usage={data.equipment_usage}
            mode="solo"
            t={usageText}
            locale={locale}
          />
        </DetailSection>
      )}

      {/* « Les formes retenues », CONTEXTE SOLO — la section porte DÉJÀ son titre standard
          (`SectionTitle`), on ne le double pas. */}
      <FormesRetenuesSection block={data.formes_retenues} locale={locale} contexte="solo" />
    </div>
  )
}
