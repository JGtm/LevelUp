/**
 * SquadAppuiPorteeSection — « Appui et portée » (Escouade › Synergies) : deux cartes, une rangée, même
 * hauteur. « Appui » à gauche (les assistances que les joueurs de l'escouade se donnent), « Rôles de
 * portée » à droite (la distance à laquelle chaque joueur frague, rapportée au lobby).
 *
 * Chaque carte se monte indépendamment (l'appui vient du résumé du film, les rôles des films décodés) :
 * la carte restante garde sa demi-largeur. Sans aucune des deux :
 *   - le titre ne lit pas le film (`filmExpected` faux, capability absente) : la section est absente ;
 *   - il le lit et le film d'au moins un match n'est pas lu (`filmGapOfBlock`) : la section garde son
 *     intertitre avec « Analyse du film en cours » ou « Film non analysé » ;
 *   - il le lit, tous les films sont lus et rien n'est mesuré : l'intertitre garde le constat
 *     « Aucune assistance ni portée de frag sur ce périmètre » (`ScopeEmptyCard`).
 * Jamais un trou qui laisse croire à un bug, jamais un cadre vide qui laisserait croire à une escouade sans entraide.
 */
import { SectionTitle } from '@/components/ui/detail-section'
import { InfoTooltip, TooltipParagraphs } from '@/components/ui/info-tooltip'
import { ScopeEmptyCard } from '@/components/ui/scope-empty'
import { FilmGapCard } from '@/features/_shared/filmGap/FilmGapSection'
import { filmExpected, filmGapOfBlock } from '@/features/_shared/filmGap/filmGap.logic'
import { FILM_GAP_TEXT } from '@/features/_shared/filmGap/filmGapStrings'
import type { MatchRangeBlock, SquadAssistPairs, SquadEmpriseBlock } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'

import type { SquadText } from './i18n'
import { SquadAppuiCard } from './SquadAppuiCard'
import { SquadRangeRolesCard } from './SquadRangeRolesCard'

interface Props {
  assistPairs: SquadAssistPairs | null | undefined
  rangeProfiles: MatchRangeBlock | null | undefined
  /** Le bloc d'Emprise de la page : il dit si le titre lit le film et si le film des matchs est lu. */
  emprise: SquadEmpriseBlock | null | undefined
  roster: string[]
  colorByPlayer: Record<string, string>
  locale: Locale
  t: SquadText
}

export function SquadAppuiPorteeSection({ assistPairs, rangeProfiles, emprise, roster, colorByPlayer, locale, t }: Props) {
  const measured = assistPairs != null || rangeProfiles != null
  if (!measured && !filmExpected(emprise)) return null
  const gap = filmGapOfBlock(emprise)
  return (
    <section className="space-y-4" aria-label={t.sections.appuiPortee} data-testid="squad-appui-portee">
      <SectionTitle className="flex items-center gap-1.5">
        {t.sections.appuiPortee}
        <InfoTooltip content={<TooltipParagraphs items={t.sections.appuiPorteeHelp} />} />
      </SectionTitle>
      {measured ? (
        <div className="grid grid-cols-1 gap-4 lg:grid-cols-2" data-testid="squad-appui-portee-row">
          {assistPairs && <SquadAppuiCard block={assistPairs} roster={roster} colorByPlayer={colorByPlayer} />}
          {rangeProfiles && <SquadRangeRolesCard bloc={rangeProfiles} roster={roster} colorByPlayer={colorByPlayer} />}
        </div>
      ) : gap ? (
        <FilmGapCard gap={gap} t={FILM_GAP_TEXT[locale]} />
      ) : (
        <ScopeEmptyCard kind="support" locale={locale} />
      )}
    </section>
  )
}
