/**
 * RelationBadgeLegend — légende détaillée des pastilles de relation, rendue dans
 * l'aide ⓘ de l'en-tête « Joueur » du tableau. Chaque entrée = la pastille réelle
 * (NarrativeBadge solid, même couleur que dans le tableau) + sa signification.
 *
 * Libellés résolus via le manifest squad (`narrative.encounter.*`, source unique
 * partagée avec RelationBadges) ; descriptions via le manifest palmares. Les
 * couleurs passent par les tokens accessibilité (jamais de hex).
 */
import { NarrativeBadge } from '@/components/feedback/NarrativeBadge'
import { tokenVar } from '@/lib/accessibility'
import type { SemanticToken } from '@/lib/accessibility/semantic-tokens'
import { formatMessage } from '@/lib/i18n/format'
import { palmaresManifest, type PalmaresManifestKey } from '@/lib/i18n/generated/palmares'
import { squadManifest, type SquadManifestKey } from '@/lib/i18n/generated/squad'
import { useAppShellStore } from '@/stores/appShellStore'

import { normalizePalmaresLocale } from './i18n'

interface LegendEntry {
  /** Identifiant du badge (`narrative.encounter.<kind>`) — sert au filtre `only`. */
  kind: string
  token: SemanticToken
  /** Clé du libellé (manifest squad). Absente ⇒ utiliser staticLabelKey. */
  labelKey?: SquadManifestKey
  /** Variables ICU du libellé (ex. ordinal). */
  labelVars?: Record<string, unknown>
  /** Libellé statique (manifest palmares) quand le libellé réel est dynamique (cross-jeu). */
  staticLabelKey?: PalmaresManifestKey
  descKey: PalmaresManifestKey
}

// Ordre de lecture : alliés d'abord, puis adversaires, puis transverses.
const ENTRIES: LegendEntry[] = [
  { kind: 'ally_plus', token: 'narrative-encounter-ally-plus', labelKey: 'narrative.encounter.ally_plus', descKey: 'palmares.relations.badge_legend.ally_plus' },
  { kind: 'duo_gagnant', token: 'narrative-encounter-duo-gagnant', labelKey: 'narrative.encounter.duo_gagnant', descKey: 'palmares.relations.badge_legend.duo_gagnant' },
  { kind: 'tough_enemy', token: 'narrative-encounter-tough-enemy', labelKey: 'narrative.encounter.tough_enemy', descKey: 'palmares.relations.badge_legend.tough_enemy' },
  { kind: 'coriace', token: 'narrative-encounter-coriace', labelKey: 'narrative.encounter.coriace', descKey: 'palmares.relations.badge_legend.coriace' },
  { kind: 'proie_favorite', token: 'narrative-encounter-proie-favorite', labelKey: 'narrative.encounter.proie_favorite', descKey: 'palmares.relations.badge_legend.proie_favorite' },
  { kind: 'cameleon', token: 'narrative-encounter-cameleon', labelKey: 'narrative.encounter.cameleon', descKey: 'palmares.relations.badge_legend.cameleon' },
  { kind: 'de_longue_date', token: 'narrative-encounter-de-longue-date', labelKey: 'narrative.encounter.de_longue_date', descKey: 'palmares.relations.badge_legend.de_longue_date' },
  { kind: 'recrue', token: 'narrative-encounter-recrue', labelKey: 'narrative.encounter.recrue', descKey: 'palmares.relations.badge_legend.recrue' },
  { kind: 'ordinal', token: 'narrative-encounter-ordinal', labelKey: 'narrative.encounter.ordinal', labelVars: { ordinal: 'N' }, descKey: 'palmares.relations.badge_legend.ordinal' },
  { kind: 'cross_game', token: 'narrative-encounter-cross-game', staticLabelKey: 'palmares.relations.badge_legend.cross_game_name', descKey: 'palmares.relations.badge_legend.cross_game' },
]

/**
 * RelationBadgeLegend — `only` restreint la légende à un sous-ensemble de badges
 * (ex. match view : seuls ally_plus/tough_enemy/coriace/ordinal sont produits par
 * le backend). Absent ⇒ tous les badges (page Relations).
 */
export function RelationBadgeLegend({ intro, only }: { intro: string; only?: string[] }) {
  const locale = normalizePalmaresLocale(useAppShellStore((s) => s.locale))
  const entries = only ? ENTRIES.filter((e) => only.includes(e.kind)) : ENTRIES
  return (
    <div className="flex flex-col gap-2 text-left">
      <p className="text-2xs text-muted-foreground">{intro}</p>
      <ul className="flex flex-col gap-1.5">
        {entries.map((e) => {
          const label = e.staticLabelKey
            ? formatMessage(palmaresManifest, e.staticLabelKey, locale)
            : formatMessage(squadManifest, e.labelKey as SquadManifestKey, locale, e.labelVars)
          const desc = formatMessage(palmaresManifest, e.descKey, locale)
          return (
            <li key={e.token} className="flex items-start gap-2">
              <span className="mt-px shrink-0">
                <NarrativeBadge label={label} colorVar={tokenVar(e.token)} solid size="sm" />
              </span>
              <span className="text-2xs leading-snug text-foreground">{desc}</span>
            </li>
          )
        })}
      </ul>
    </div>
  )
}
