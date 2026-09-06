/**
 * TacticalFilterBar — la barre L2 de l'onglet Tactique.
 *
 * ELLE N'INVENTE AUCUN CONTRÔLE : elle ASSEMBLE ce qui existe (décision produit du
 * 2026-09-06, « un mix de l'Explorateur et de l'Escouade, sans rien inventer ») —
 *   période / saison / expérience / playlists / modes  `useLocalFilterBar`
 *   segmentation solo / escouade / mixte                `viewLabels` du même hook
 *   sessions épinglées                                  `SessionMultiSelect`
 *   composition (0 à 3 coéquipiers)                     `GamertagCombobox`
 *
 * Les deux derniers sont rendus DANS la barre du hook (`extras`), pas en dessous :
 * une seconde ligne de filtres donnerait deux zones pour un seul scope.
 *
 * ─── CE QUI S'APPLIQUE QUAND ────────────────────────────────────────────────
 *
 * Sessions et composition s'appliquent EN DIRECT, période et cascade au clic sur
 * « Analyser » — exactement le partage de la barre de l'Escouade. La raison est la
 * même : une session ou un coéquipier sont des choix ponctuels et lisibles, une
 * cascade se règle en plusieurs gestes et ne doit pas relancer une requête à chacun.
 *
 * Tout l'état committed vit dans l'URL (`usePageScope`) : le retour navigateur, le
 * rechargement et le lien partagé restaurent la même lecture.
 */
import { useMemo } from 'react'

import { GamertagCombobox } from '@/components/ui/GamertagCombobox'
import { SessionMultiSelect } from '@/components/ui/SessionMultiSelect'
import { useLocalFilterBar, type LocalFilterBarState } from '@/features/_shared/useLocalFilterBar'
import type { TeammateOption } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'

import type { TacticalText } from './i18n'
import { sessionsProposees } from './tacticalLogic'
import { MAX_COEQUIPIERS, type TacticalScope } from './tacticalScope'

export interface TacticalFilterBarProps {
  playerSlug: string
  locale: Locale
  t: TacticalText
  scope: TacticalScope
  setScope: (patch: Partial<TacticalScope>) => void
  /** Coéquipiers proposés au sélecteur de composition (avec leur xuid). */
  coequipierOptions: TeammateOption[]
}

export function TacticalFilterBar({
  playerSlug,
  locale,
  t,
  scope,
  setScope,
  coequipierOptions,
}: TacticalFilterBarProps) {
  // L'état committed du hook EST le scope d'URL : une seule vérité, donc le retour
  // navigateur remet les pills ET les requêtes dans le même mouvement.
  const committed = useMemo(
    () => ({
      value: {
        period: { start_date: scope.debut || null, end_date: scope.fin || null },
        experience: scope.experience,
        playlists: scope.playlists,
        modes: scope.modes,
        view: scope.vue,
      } satisfies LocalFilterBarState,
      onCommit: (next: LocalFilterBarState) =>
        setScope({
          debut: next.period.start_date ?? '',
          fin: next.period.end_date ?? '',
          experience: next.experience,
          playlists: next.playlists,
          modes: next.modes,
          vue: next.view,
        }),
    }),
    [scope.debut, scope.fin, scope.experience, scope.playlists, scope.modes, scope.vue, setScope],
  )

  const avecComposition = scope.coequipiers.length > 0

  const { bar } = useLocalFilterBar({
    playerSlug,
    labels: t.filterLabels,
    viewLabels: t.viewLabels,
    committed,
    // Fonction, et pas un nœud : les sessions proposées viennent de ce que le hook
    // charge pour ses propres counts — donc elles n'existent pas encore ici.
    extras: ({ sessionOptions }) => (
      <BarreExtras
        locale={locale}
        t={t}
        scope={scope}
        setScope={setScope}
        avecComposition={avecComposition}
        sessions={sessionOptions}
        coequipierOptions={coequipierOptions}
        playerSlug={playerSlug}
      />
    ),
  })

  return <>{bar}</>
}

/** Les deux contrôles que le hook ne porte pas : sessions et composition. */
function BarreExtras({
  playerSlug,
  locale,
  t,
  scope,
  setScope,
  avecComposition,
  sessions,
  coequipierOptions,
}: {
  playerSlug: string
  locale: Locale
  t: TacticalText
  scope: TacticalScope
  setScope: (patch: Partial<TacticalScope>) => void
  avecComposition: boolean
  sessions: Parameters<typeof sessionsProposees>[0]
  coequipierOptions: TeammateOption[]
}) {
  // Les sessions proposées SUIVENT la composition : escouade dès qu'un coéquipier
  // est choisi, solo sinon (même mécanique que la barre de l'Escouade).
  const proposees = useMemo(
    () => sessionsProposees(sessions, avecComposition),
    [sessions, avecComposition],
  )

  return (
    <>
      <SessionMultiSelect
        sessions={proposees}
        selected={scope.sessions}
        onChange={(labels) => setScope({ sessions: labels })}
        locale={locale}
        placeholder={t.sessions}
        triggerClassName="flex items-center gap-1.5 rounded-md border border-input bg-background px-2.5 py-1 text-xs font-medium hover:bg-muted whitespace-nowrap transition-colors"
      />
      <GamertagCombobox
        compact
        selected={scope.coequipiers}
        onChange={(gts) => setScope({ coequipiers: gts.slice(0, MAX_COEQUIPIERS) })}
        max={MAX_COEQUIPIERS}
        frequentOptions={coequipierOptions}
        excludeGamertag={playerSlug}
        placeholder={t.squadPlaceholder(coequipierOptions.length)}
      />
    </>
  )
}
