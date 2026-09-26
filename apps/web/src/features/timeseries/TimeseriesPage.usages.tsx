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
 * AUCUNE REQUÊTE NEUVE : les blocs (`weapon_range`, `elevation`, `range_profiles`,
 * `equipment_usage`, `formes_retenues`) arrivent avec la MÊME réponse de page que les
 * autres onglets.
 *
 * LES RÔLES DE PORTÉE SUIVENT LA PORTÉE (D23-a du 2026-09-22) : la carte dit à quelle
 * distance je me tiens PAR RAPPORT AU LOBBY, la section au-dessus dit avec quelle arme et à
 * quels mètres. Même capability produit, donc même gate, et elles se lisent l'une après
 * l'autre — les séparer d'un onglet romprait la paire.
 *
 * CHAQUE SECTION SE RETIRE D'ELLE-MÊME quand son bloc est absent (film non décodé, titre
 * sans la capability `weapon_range`). Les trois retirées, l'onglet DIT pourquoi il est vide
 * au lieu de rester muet : un onglet sans contenu ni message se lit « bug ».
 */
import { DetailSection } from '@/components/ui/detail-section'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import { EquipmentUsageSection } from '@/features/_shared/usage/EquipmentUsageSection'
import { usageAvailabilityKind } from '@/features/_shared/usage/usageAvailability'
import { USAGE_TEXT } from '@/features/_shared/usage/usageI18n'
import { FormesRetenuesSection } from '@/features/squad/formes/FormesRetenuesSection'
import { useCapability } from '@/lib/capabilities/capabilities'
import type { TimeseriesPageResponse } from '@/lib/api/types'
import type { TimeseriesManifestKey } from '@/lib/i18n/generated/timeseries'
import type { Locale } from '@/lib/i18n/locale'

import { TimeseriesRangeRolesCard } from './TimeseriesRangeRolesCard'
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
    data.equipment_usage != null && usageAvailabilityKind(data.equipment_usage) !== 'hidden'
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
      {/* Portée des engagements — la section porte son propre titre standard. Le nuage
          « distance x hauteur d'engagement » vit DANS cette section, à côté de la portée. */}
      {showsRange && (
        <WeaponRangeSection
          range={data.weapon_range}
          elevation={data.elevation}
          matchRows={data.match_rows ?? []}
        />
      )}

      {/* Rôles de portée — juste après « Portée par arme » : la même famille de sujet, posée
          en écart au lobby plutôt qu'en mètres absolus. */}
      {hasWeaponRange && <TimeseriesRangeRolesCard bloc={data.range_profiles} />}

      {/* « Équipement et armes de socle » — la section ne monte que des cartes, sans titre à
          elle : le titre de section est posé ici, et il NOMME LES DEUX FAMILLES DE RAMASSAGES
          qu'il coiffe — l'équipement (usages, part du lobby) et les armes posées sur les
          socles (armes spéciales, niveaux). Il ne se confond donc ni avec « Portée des
          engagements » ci-dessus, ni avec sa première carte (« Usages d'équipement »), dont
          reprendre le libellé ferait lire deux fois la même ligne (arbitrage du
          2026-09-22). */}
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

      {/* « Les formes retenues », CONTEXTE SOLO — elle ne porte PLUS de titre de section
          depuis la décision D5 du 2026-09-21 (trois intertitres se suivaient) : chacun de ses
          blocs porte le sien. On ne lui en repose donc pas un ici. */}
      <FormesRetenuesSection block={data.formes_retenues} locale={locale} contexte="solo" />
    </div>
  )
}
