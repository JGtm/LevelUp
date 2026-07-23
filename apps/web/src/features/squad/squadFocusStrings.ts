/**
 * squadFocusStrings — accès aux libellés du bandeau « Objectifs d'escouade » et
 * de son panneau de défis, résolus depuis le manifest i18n `squad.focus.*`
 * (source unique FR/EN + parité vérifiée par le build manifest, ADR 0003).
 *
 * `getSquadFocusText(locale)` adapte le manifest à un objet `t` ergonomique
 * (statiques + fonctions d'interpolation ICU) consommé par les composants du
 * bandeau. Ne pas remettre de littéraux FR/EN en dur ici : tout vit dans
 * `lib/i18n/manifests/squad.toml`.
 */
import { formatMessage } from '@/lib/i18n/format'
import { squadManifest, type SquadManifestKey } from '@/lib/i18n/generated/squad'
import type { Locale } from '@/lib/i18n/locale'

export function getSquadFocusText(locale: Locale) {
  const m = (key: SquadManifestKey, values?: Record<string, unknown>) =>
    formatMessage(squadManifest, key, locale, values)
  return {
    title: m('squad.focus.title'),
    saved: (name: string) => m('squad.focus.saved', { name }),
    manage: m('squad.focus.manage'),
    hide: m('squad.focus.hide'),
    saveCta: m('squad.focus.save_cta'),
    saving: m('squad.focus.saving'),
    saveSuccess: m('squad.focus.save_success'),
    saveError: m('squad.focus.save_error'),
    renameCta: m('squad.focus.rename_cta'),
    deleteCta: m('squad.focus.delete_cta'),
    confirmDelete: m('squad.focus.confirm_delete'),
    ok: m('squad.focus.ok'),
    renamed: m('squad.focus.renamed'),
    deleted: m('squad.focus.deleted'),
    noObjectives: m('squad.focus.no_objectives'),
    join: m('squad.focus.join'),
    joined: m('squad.focus.joined'),
    joinedToast: m('squad.focus.joined_toast'),
    joinError: m('squad.focus.join_error'),
    evaluate: m('squad.focus.evaluate'),
    evaluated: m('squad.focus.evaluated'),
    evalError: m('squad.focus.eval_error'),
    created: m('squad.focus.created'),
    createError: m('squad.focus.create_error'),
    challengeDeleted: m('squad.focus.challenge_deleted'),
    challengeDeleteError: m('squad.focus.challenge_delete_error'),
    reached: m('squad.focus.reached'),
    expired: m('squad.focus.expired'),
    target: (n: number) => m('squad.focus.target', { n }),
    matchesN: (n: number) => m('squad.focus.matches_n', { n }),
    challenge: m('squad.focus.challenge'),
    propose: m('squad.focus.propose'),
    proposing: m('squad.focus.proposing'),
    create: m('squad.focus.create'),
    creating: m('squad.focus.creating'),
    poolEmpty: m('squad.focus.pool_empty'),
    poolCaption: m('squad.focus.pool_caption'),
    poolError: m('squad.focus.pool_error'),
    orientation: (axis: string) => m('squad.focus.orientation', { axis }),
    axisLabels: {
      combat: m('squad.focus.axis_combat'),
      survival: m('squad.focus.axis_survival'),
      support: m('squad.focus.axis_support'),
      score: m('squad.focus.axis_score'),
      objective: m('squad.focus.axis_objective'),
      impact: m('squad.focus.axis_impact'),
    },
  }
}

/** Forme de l'objet de libellés résolus (consommé par les composants du bandeau). */
export type SquadFocusText = ReturnType<typeof getSquadFocusText>
