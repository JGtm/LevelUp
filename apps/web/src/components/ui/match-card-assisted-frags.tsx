/**
 * MatchCardAssistedFrags — sous la barre frags / assistances / décès de la tuile de match :
 * la barre à trois tons du sens « reçues » (un coéquipier t'a assisté), part de dégâts par
 * segment en infobulle, puis SA légende « N / M frags assistés » dessous (comme la barre
 * du dessus porte la sienne — la part en % ne s'écrit pas, la barre la montre).
 *
 * L'emplacement est RÉSERVÉ : sans `assisted_frags` (match non mesuré : pas de film
 * analysé) le composant occupe la même hauteur, vide — sur la grille de l'accueil, les
 * tuiles voisines gardent leurs stats alignées. Aucun « — » ni « 0 » fabriqué : la tuile
 * n'a pas de place pour l'incertitude, et un « 0 » se lirait comme une mesure.
 */
import { AssistTierBar, ASSIST_RECEIVED_TOKEN } from '@/features/_shared/assists/AssistTierBar'
import { assistTierTone } from '@/features/_shared/assists/assistTierTone'
import { assistShareSegments } from '@/features/_shared/assists/assistExchange'
import { ASSISTS_TEXT } from '@/features/_shared/assists/assistsI18n'
import { tokenCssVar } from '@/lib/accessibility'
import type { MatchAssistedFrags } from '@/lib/api/types'
import { formatMessage } from '@/lib/i18n/format'
import { commonManifest } from '@/lib/i18n/generated/common'
import type { Locale } from '@/lib/i18n/locale'

// Barre (h-2, comme la barre frags / assistances / décès) + interligne 6 px + légende text-2xs
// (10 px, leading-none) : la hauteur réservée est celle du bloc mesuré. `mt-3` le détache
// de la légende frags / assistances / morts (demande utilisateur 2026-09-19).
const SLOT_CLASS = 'mt-3 h-[24px] space-y-1.5'
const LEGEND_CLASS = 'text-center font-mono text-2xs tabular-nums leading-none'

export function MatchCardAssistedFrags({
  assisted,
  locale,
}: {
  assisted: MatchAssistedFrags | null | undefined
  locale: Locale
}) {
  if (!assisted) {
    return <div data-testid="match-card-assisted-frags-slot" className={SLOT_CLASS} aria-hidden="true" />
  }
  const frags = assisted.frags_measured
  const received = assisted.received.total
  const fmt = (n: number) => n.toLocaleString(locale)
  return (
    <div data-testid="match-card-assisted-frags" className={SLOT_CLASS}>
      <AssistTierBar
        segments={assistShareSegments(assisted.received, frags)}
        color={tokenCssVar(ASSIST_RECEIVED_TOKEN)}
        text={ASSISTS_TEXT[locale]}
        locale={locale}
        variant="tile"
        testId="match-card-assist-segment"
      />
      {/* Légende dans le ton FORT du sens (même teinte, clarté montée vers le premier plan du
          thème) : le jeton tel quel, pinné sombre par le contraste clair, est trop terne en
          texte sur la tuile (retour utilisateur 2026-09-19). */}
      <div className={LEGEND_CLASS} style={{ color: assistTierTone(tokenCssVar(ASSIST_RECEIVED_TOKEN), 'high') }}>
        {formatMessage(commonManifest, 'common.match_card.assisted_frags', locale, {
          assisted: fmt(received),
          frags: fmt(frags),
        })}
      </div>
    </div>
  )
}
