/**
 * scope-empty-text — les textes d'un bloc qui reste AFFICHÉ alors qu'aucun match du périmètre n'est
 * concerné (« Aucun match à objectif sur ce périmètre », « Aucun frag sur ce périmètre »).
 *
 * Règle produit (2026-10-17) : un manque de fonction (capability du titre absente) retire le bloc ;
 * un manque de données (aucun match du périmètre n'est concerné) le laisse, avec son titre et un état
 * vide dans un bloc parent — un bloc qui disparaît se lit comme un bug. Une lecture en échec (`objectiveUnavailable`) se dit à part : elle n'est pas « aucun match concerné »,
 * mais le bloc reste de même affiché. Constats factuels : ni
 * personne, ni action possible, ni mot interne (garde `textesSansPersonne.test.ts`).
 *
 * Le titre du constat des frags est celui du manifeste partagé (`frags.empty.no_data`), lu aussi par
 * « Frags par arme » : une seule phrase pour le même constat.
 */
import { formatMessage } from '@/lib/i18n/format'
import { fragsManifest } from '@/lib/i18n/generated/frags'
import type { Locale } from '@/lib/i18n/locale'

/**
 * Ce que le périmètre n'a pas : des matchs à objectif, des actions d'objectif, des frags, une assistance ou une portée lues
 * au film ; ou ce que la lecture n'a pas rendu (`objectiveUnavailable` : lecture de l'objectif en échec, qui n'est pas
 * « aucun match à objectif »).
 */
export type ScopeEmptyKind = 'objective' | 'objectiveActions' | 'objectiveUnavailable' | 'frags' | 'support'

export interface ScopeEmptyText {
  title: string
  description: string
}

const TEXT: Record<Locale, Record<Exclude<ScopeEmptyKind, 'frags'>, ScopeEmptyText> & { fragsDescription: string }> = {
  fr: {
    objective: {
      title: 'Aucun match à objectif sur ce périmètre',
      description: 'Aucun des matchs de ce périmètre n’est joué dans un mode à objectif.',
    },
    objectiveActions: {
      title: 'Aucune action d’objectif sur ce périmètre',
      description: 'Aucune action de drapeau, de zone ou de crâne n’est enregistrée sur les matchs de ce périmètre.',
    },
    objectiveUnavailable: {
      title: 'Objectif indisponible pour ce périmètre',
      description: 'Les données d’objectif de ce périmètre n’ont pas pu être chargées.',
    },
    support: {
      title: 'Aucune assistance ni portée de frag sur ce périmètre',
      description: 'Aucun match de ce périmètre n’a d’assistance ni de portée de frag lue au film.',
    },
    fragsDescription: 'Aucun frag n’est enregistré sur les matchs de ce périmètre.',
  },
  en: {
    objective: {
      title: 'No objective match in this scope',
      description: 'None of the matches in this scope was played in an objective mode.',
    },
    objectiveActions: {
      title: 'No objective action in this scope',
      description: 'No flag, zone or skull action is recorded on the matches in this scope.',
    },
    objectiveUnavailable: {
      title: 'Objective unavailable for this scope',
      description: 'The objective data for this scope could not be loaded.',
    },
    support: {
      title: 'No assist or kill range in this scope',
      description: 'None of the matches in this scope has an assist or a kill range read from the film.',
    },
    fragsDescription: 'No kill is recorded on the matches in this scope.',
  },
}

/** Le constat d'un bloc sans match concerné, à la langue de l'application. */
export function scopeEmptyText(kind: ScopeEmptyKind, locale: Locale): ScopeEmptyText {
  const t = TEXT[locale]
  if (kind === 'frags') {
    return { title: formatMessage(fragsManifest, 'frags.empty.no_data', locale), description: t.fragsDescription }
  }
  return t[kind]
}

/** Tous les constats d'une langue : ce que la garde des textes sans personne parcourt. */
export function allScopeEmptyTexts(locale: Locale): ScopeEmptyText[] {
  return (['objective', 'objectiveActions', 'objectiveUnavailable', 'frags', 'support'] as const).map((k) => scopeEmptyText(k, locale))
}
