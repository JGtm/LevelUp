/**
 * ReplaySchemaBadge — le badge ADMIN « version de schéma » (lot A, 2026-09-11).
 *
 * RIEN, PAS MÊME UN ÉLÉMENT VIDE, POUR UN NON-ADMIN : `isAdmin` gate le composant tout
 * entier — un joueur ordinaire ne doit voir aucune trace de la mécanique de recuisson.
 * `isAdmin` est passé en PROP (la route le lit sur `useAppShellStore`, même convention que
 * `locale`/`theme` déjà redescendus par `replay.tsx`) : ce composant reste un pur
 * présentateur, testable sans monter le store.
 *
 * TROIS RENDUS, un par `ReplaySchemaStatus.kind` (`model/replaySchemaStatusLogic.ts`) :
 *  - `upToDate` : ton `success` — l'artefact porte déjà la version courante du producteur ;
 *  - `stale` : ton `warning` — recuisson à faire, le second nombre est la version cible ;
 *  - `unknown` : NEUTRE — aucun en-tête `X-Replay-Latest-Schema-Version` à comparer
 *    (artefact antérieur à ce lot, ou réponse qui l'aurait perdu en route) ; le badge dit la
 *    seule chose qu'il sait, la version LUE. Pas de jeton `tokenCssVar` pour ce cas : aucun
 *    jeton « neutral » n'existe dans `semantic-tokens.ts` (uniquement des variantes composées
 *    — `zone-neutral`, `divergent-neutral` — hors sujet ici), et les classes Tailwind
 *    `text-muted-foreground` / `border-border` sont déjà le vocabulaire NEUTRE du dépôt
 *    (utilisées telles quelles dans `ReplayCountersBadge.tsx`, `replay.tsx`…), distinct des
 *    classes de COULEUR que le skill `color-tokens` proscrit (`text-red-*` etc.).
 */
import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'

import { computeReplaySchemaStatus } from '../model/replaySchemaStatusLogic'
import { REPLAY_TEXT, type ReplayLocale } from '../i18n/i18n'

export function ReplaySchemaBadge({
  isAdmin,
  schemaVersion,
  latestSchemaVersion,
  locale,
}: {
  isAdmin: boolean
  schemaVersion: number
  latestSchemaVersion: number | undefined
  locale: ReplayLocale
}) {
  if (!isAdmin) return null
  const t = REPLAY_TEXT[locale]
  const status = computeReplaySchemaStatus(schemaVersion, latestSchemaVersion)
  const label =
    status.kind === 'upToDate'
      ? t.schemaBadgeUpToDateFmt(status.schemaVersion)
      : status.kind === 'stale'
        ? t.schemaBadgeStaleFmt(status.schemaVersion, status.latestSchemaVersion)
        : t.schemaBadgeUnknownFmt(status.schemaVersion)
  if (status.kind === 'unknown') {
    return (
      <span
        className="inline-flex items-center rounded-full border border-border px-2 py-0.5 text-[10px] font-medium text-muted-foreground"
        title={label}
      >
        {label}
      </span>
    )
  }
  const color = status.kind === 'upToDate' ? 'success' : 'warning'
  return (
    <span
      className="inline-flex items-center rounded-full border px-2 py-0.5 text-[10px] font-medium"
      style={{ borderColor: tokenCssVar(color), color: tokenCssVar(color) }}
      title={label}
    >
      {label}
    </span>
  )
}
