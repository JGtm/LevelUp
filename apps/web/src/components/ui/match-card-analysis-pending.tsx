/**
 * Emplacements EN ATTENTE d'une tuile de match dont l'analyse du film n'est pas finie
 * (`RecentMatchItem.analysis_pending`) : à la place d'un vide, la tuile annonce ce qui
 * manque. Chaque emplacement n'est rendu que si la donnée correspondante manque ET que
 * l'analyse n'est pas finie — jamais sur une tuile normale.
 *
 *  - frags assistés : petit indicateur tournant + « Frags assistés : analyse en cours|en attente » ;
 *  - citations : trois cercles en pointillés + « Citations : analyse en cours|en attente » ;
 *  - rejeu : puce en pointillés « Rejeu 2D : analyse en cours|en attente », à la place du bouton absent.
 *
 * Tout mouvement est sous `motion-safe:` (`prefers-reduced-motion`). Jetons sémantiques
 * uniquement (`info`, `muted-foreground`).
 */
import { formatMessage } from '@/lib/i18n/format'
import { commonManifest, type CommonManifestKey } from '@/lib/i18n/generated/common'
import type { Locale } from '@/lib/i18n/locale'
import { analysisPendingLabelKey } from './match-card-presentation'

/** `inProgress` : un décodage porte sur CE match (`analysis_state`) ; sinon l'emplacement dit « en attente ». */
interface PendingSlotProps {
  locale: Locale
  inProgress: boolean
}

function textOf(locale: Locale) {
  return (key: CommonManifestKey) => formatMessage(commonManifest, key, locale)
}

/** Même hauteur réservée que l'emplacement des frags assistés (alignement des tuiles voisines). */
const ASSISTED_SLOT_CLASS = 'mt-3 h-[24px] flex items-center justify-center gap-1.5 text-2xs italic text-muted-foreground'

export function MatchCardAssistedFragsPending({ locale, inProgress }: PendingSlotProps) {
  const t = textOf(locale)
  return (
    <div data-testid="match-card-assisted-frags-pending" className={ASSISTED_SLOT_CLASS}>
      <span
        aria-hidden="true"
        className="inline-block h-3 w-3 shrink-0 rounded-full border-2 border-muted-foreground/30 border-t-info motion-safe:animate-spin"
      />
      <span>{t(analysisPendingLabelKey('assisted_frags', inProgress))}</span>
    </div>
  )
}

export function MatchCardCitationsPending({ locale, inProgress }: PendingSlotProps) {
  const t = textOf(locale)
  return (
    <div
      data-testid="match-card-citations-pending"
      className="px-3 pb-2.5 pt-2 flex flex-col items-center gap-1.5 border-t border-border/40 mt-2"
    >
      <div className="flex justify-center gap-3" aria-hidden="true">
        {[0, 1, 2].map((i) => (
          <span key={i} className="h-10 w-10 rounded-full border-2 border-dashed border-muted-foreground/40" />
        ))}
      </div>
      <span className="text-2xs italic text-muted-foreground">{t(analysisPendingLabelKey('citations', inProgress))}</span>
    </div>
  )
}

export function MatchCardReplayPending({ locale, inProgress }: PendingSlotProps) {
  const t = textOf(locale)
  return (
    <span
      data-testid="match-card-replay-pending"
      className="inline-flex items-center rounded-full border border-dashed border-muted-foreground/50 px-2.5 py-0.5 text-2xs italic leading-none text-muted-foreground"
    >
      {t(analysisPendingLabelKey('replay', inProgress))}
    </span>
  )
}
