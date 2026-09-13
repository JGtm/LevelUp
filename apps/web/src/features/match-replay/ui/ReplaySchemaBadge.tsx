/**
 * ReplaySchemaBadge — le badge ADMIN « version de schéma » (lot A, 2026-09-11).
 *
 * RIEN, PAS MÊME UN ÉLÉMENT VIDE, POUR UN NON-ADMIN : `isAdmin` gate le composant tout
 * entier — un joueur ordinaire ne doit voir aucune trace de la mécanique de recuisson.
 * `isAdmin` est passé en PROP (la route le lit sur `useAppShellStore`, même convention que
 * `locale`/`theme` déjà redescendus par `replay.tsx`) : ce composant reste un pur
 * présentateur, testable sans monter le store.
 *
 * QUATRE RENDUS, un par `ReplaySchemaStatus.kind` (`model/replaySchemaStatusLogic.ts`) :
 *  - `upToDate` : ton `success` — l'artefact porte déjà la version courante du producteur ;
 *  - `stale` : ton `warning` — recuisson à faire ; le second nombre est la version cible
 *    QUAND on en connaît une (aucune sous la version minimale affichable, en-tête absent) ;
 *  - `invalid` : ton `destructive` (2026-09-13, lot 0.B) — le document ne respecte pas le
 *    contrat, et le badge NOMME le premier manquement. C'est la seule trace visible d'une
 *    dérive que TypeScript ne voit plus une fois compilé ; le rendu, lui, continue — une page
 *    blanche apprendrait moins qu'un rejeu incomplet ;
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

import type { ReplayContractIssue } from '@/lib/replay/replayDocumentSchema'

import { computeReplaySchemaStatus, type ReplaySchemaStatus } from '../model/replaySchemaStatusLogic'
import { REPLAY_TEXT, type ReplayLocale } from '../i18n/i18n'

/**
 * detailDe met le manquement EN MOTS, dans la langue du lecteur (ronde 2 de la revue, constat
 * R2-2). Il était fabriqué côté schéma, en français, et se retrouvait donc au milieu de la
 * phrase anglaise du badge : `contract violated (cle(s) inconnue(s) : shotz)`. La liste des
 * clés et le texte de zod, eux, restent bruts — ce sont des données, pas de la langue.
 */
function detailDe(
  issue: ReplayContractIssue,
  t: (typeof REPLAY_TEXT)[ReplayLocale],
): string {
  switch (issue.kind) {
    case 'unknownKeys':
      return t.contractUnknownKeysFmt(issue.keys.join(', '))
    case 'invalidField':
      return t.contractInvalidFieldFmt(issue.path, issue.detail)
    default:
      return t.contractMalformed
  }
}

/** Le libellé d'un statut, dans la langue courante. Pur : c'est ce qui le rend testable seul. */
function libelleDe(status: ReplaySchemaStatus, t: (typeof REPLAY_TEXT)[ReplayLocale]): string {
  switch (status.kind) {
    case 'upToDate':
      return t.schemaBadgeUpToDateFmt(status.schemaVersion)
    case 'invalid':
      return t.schemaBadgeInvalidFmt(status.schemaVersion, detailDe(status.issue, t))
    case 'stale':
      return status.latestSchemaVersion === undefined
        ? t.schemaBadgeStaleNoTargetFmt(status.schemaVersion)
        : t.schemaBadgeStaleFmt(status.schemaVersion, status.latestSchemaVersion)
    default:
      return t.schemaBadgeUnknownFmt(status.schemaVersion)
  }
}

export function ReplaySchemaBadge({
  isAdmin,
  schemaVersion,
  latestSchemaVersion,
  contractIssue,
  locale,
}: {
  isAdmin: boolean
  schemaVersion: number
  latestSchemaVersion: number | undefined
  contractIssue?: ReplayContractIssue
  locale: ReplayLocale
}) {
  if (!isAdmin) return null
  const t = REPLAY_TEXT[locale]
  const status = computeReplaySchemaStatus(schemaVersion, latestSchemaVersion, contractIssue)
  const label = libelleDe(status, t)
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
  const color =
    status.kind === 'upToDate' ? 'success' : status.kind === 'invalid' ? 'destructive' : 'warning'
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
