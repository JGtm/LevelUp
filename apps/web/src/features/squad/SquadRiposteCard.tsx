/**
 * SquadRiposteCard — « Riposte », la première carte de la section Coordination.
 *
 * ELLE ABSORBE SIX BLOCS (D19, variante A de la maquette du 2026-09-21). L'onglet Synergies
 * montait huit SectionCard de même poids visuel dont SEPT portaient la même notion : le
 * taux y était énoncé trois fois (en prose, en tuile, en ligne d'habituel), le délai deux
 * fois, le détail par joueur deux fois. Le vocabulaire changeait à chaque carte — échange,
 * vengeance, assistance croisée — sans que la mesure change.
 *
 * QUATRE ÉTAGES, ET UN SEUL RÉCIT :
 *
 *   1. une PHRASE DE LECTEUR, toujours rendue (l'ancien « Constat du moment » se taisait
 *      sous 5 points d'écart — un seuil qui faisait disparaître la seule carte qui parlait
 *      français) ; sa dernière proposition, l'écart à l'habituel, SE TAIT quand le filtre
 *      couvre déjà tout l'historique : l'écart y est nul par tautologie ;
 *   2. le CHIFFRE D'APPEL face à l'habituel, et le délai médian à côté ;
 *   3. la FRISE soirée par soirée (bâtons + tendance + repère d'habituel + volumes) ;
 *   4. deux REPLIS fermés par défaut — le détail du délai, puis le détail par couple.
 *
 * LES CHIFFRES DE LA MAQUETTE NE SONT JAMAIS REPRIS : tout vient de `appelRiposte`.
 */
import { useMemo } from 'react'

import { TooltipParagraphs } from '@/components/ui/info-tooltip'
import { titleWithInfo } from '@/components/ui/title-with-info'
import { SectionCard } from '@/components/ui/section-card'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import { tokenCssVar } from '@/lib/accessibility'
import { formatPoints } from '@/lib/baseline'
import { withLowSampleNote } from '@/lib/formatters/lowSampleNote'
import { intlLocale } from '@/lib/formatters'
import type { SquadEchange } from '@/lib/api/types'
import { useAppShellStore } from '@/stores/appShellStore'

import { SquadRiposteDelaiPanel } from './SquadRiposteDelaiPanel'
import { SquadRiposteMatricePanel } from './SquadRiposteMatricePanel'
import { SquadRiposteSessionsChart } from './SquadRiposteSessionsChart'
import { appelRiposte, friseRiposte, FENETRE_TENDANCE, PLANCHER_MORTS } from './squadRiposte.logic'
import { getSquadRiposteText } from './squadRiposteStrings'

export interface SquadRiposteCardProps {
  echange: SquadEchange
}

export function SquadRiposteCard({ echange }: SquadRiposteCardProps) {
  const locale = useAppShellStore((s) => s.locale)
  const t = getSquadRiposteText(locale)
  const numLoc = intlLocale(locale)

  const pctFmt = useMemo(
    () =>
      new Intl.NumberFormat(numLoc, {
        style: 'percent',
        minimumFractionDigits: 1,
        maximumFractionDigits: 1,
      }),
    [numLoc],
  )
  const secFmt = useMemo(
    () => new Intl.NumberFormat(numLoc, { minimumFractionDigits: 1, maximumFractionDigits: 1 }),
    [numLoc],
  )

  const secondes = echange.fenetre_ms / 1000
  const appel = useMemo(() => appelRiposte(echange), [echange])
  const frise = useMemo(() => friseRiposte(echange), [echange])

  // UNE SEULE INFOBULLE PAR CARTE : la définition (la fenêtre), la règle de l'habituel, et
  // la réserve d'échantillon quand elle s'applique. Tout ce qui dit la MÉTHODE y tient —
  // rien de cela ne doit être du gris entre la phrase et le graphe.
  const aide = (
    <TooltipParagraphs
      items={[
        t.definition(secondes),
        t.helpUsual,
        appel.echantillonFaible
          ? withLowSampleNote(t.lowSampleHint(PLANCHER_MORTS), true, t.lowSample)
          : null,
      ]}
    />
  )

  return (
    <SectionCard
      title={t.title}
      label={t.label}
      titleAdornment={titleWithInfo(aide, { testId: 'squad-riposte-low-sample' })}
    >
      <div className="space-y-3 px-3 py-2" data-testid="squad-riposte">
        <PhraseDeLecteur appel={appel} secondes={secondes} secFmt={secFmt} t={t} />
        <ChiffreDAppel appel={appel} pctFmt={pctFmt} secFmt={secFmt} t={t} />
        {frise.soirees.length > 0 ? (
          <SquadRiposteSessionsChart
            frise={frise}
            title={t.friseTitle(secondes)}
            emptyMessage={t.friseEmpty}
            aboveLabel={t.friseAbove}
            belowLabel={t.friseBelow}
            trendLabel={t.friseTrend(FENETRE_TENDANCE)}
            usualLabel={t.friseUsual(pctFmt.format(appel.habituel))}
            yAxisLabel={t.friseYAxis}
            volumeAxisLabel={t.friseVolumeAxis}
            volumeTooltip={t.friseVolume}
          />
        ) : (
          <EmptyStateNotice title={t.emptyTitle} description={t.friseEmpty} />
        )}
      </div>
      <details className="border-t border-border pb-2" data-testid="squad-riposte-fold-delai">
        <summary className="cursor-pointer px-3 py-1 text-xs text-muted-foreground">
          {t.foldDelay}
        </summary>
        <div className="px-3 pb-1">
          <SquadRiposteDelaiPanel echange={echange} />
        </div>
      </details>
      <details className="border-t border-border pb-2" data-testid="squad-riposte-fold-matrice">
        <summary className="cursor-pointer px-3 py-1 text-xs text-muted-foreground">
          {t.foldMatrix}
        </summary>
        <div className="px-3 pb-1">
          <SquadRiposteMatricePanel echange={echange} />
        </div>
      </details>
    </SectionCard>
  )
}

type Appel = ReturnType<typeof appelRiposte>
type Texte = ReturnType<typeof getSquadRiposteText>

/**
 * PhraseDeLecteur — le fait, puis l'écart à l'ordinaire.
 *
 * TOUJOURS RENDUE, et c'est le changement de fond : l'ancienne carte « Constat du moment »
 * ne s'affichait qu'au-dessus de 30 morts ET 5 points d'écart, si bien que la seule carte
 * en français disparaissait le plus souvent. Seule la DERNIÈRE PROPOSITION est
 * conditionnelle — elle se tait quand l'écart est une tautologie (filtre = tout
 * l'historique) ou quand il est nul.
 */
function PhraseDeLecteur({
  appel,
  secondes,
  secFmt,
  t,
}: {
  appel: Appel
  secondes: number
  secFmt: Intl.NumberFormat
  t: Texte
}) {
  const fait =
    appel.surCombien != null && appel.delaiMedianS != null
      ? t.sayFact({
          seconds: secondes,
          surCombien: appel.surCombien,
          delai: secFmt.format(appel.delaiMedianS),
        })
      : t.sayFactNever(secondes)
  const ecart =
    appel.pleinHistorique || appel.ecartPoints === 0
      ? null
      : appel.ecartPoints < 0
        ? t.sayBelow
        : t.sayAbove
  return (
    <p
      className="border-l-2 border-info pl-3 text-sm text-foreground"
      data-testid="squad-riposte-phrase"
    >
      {ecart ? `${fait} ${ecart}` : fait}
    </p>
  )
}

/**
 * ChiffreDAppel — « 19,4 % ripostées » face à l'habituel, et « 2,4 s de délai médian ».
 *
 * L'ÉCART PREND LA COULEUR DE SON SIGNE (`outcome-win` / `outcome-loss`) : c'est un verdict
 * sur la période, pas une identité de série. Il DISPARAÎT quand le filtre couvre tout
 * l'historique — « ±0 pts vs habituel » ferait croire à une mesure là où il n'y a qu'une
 * tautologie.
 */
function ChiffreDAppel({
  appel,
  pctFmt,
  secFmt,
  t,
}: {
  appel: Appel
  pctFmt: Intl.NumberFormat
  secFmt: Intl.NumberFormat
  t: Texte
}) {
  const sousLHabituel = appel.ecartPoints < 0
  const ecart = appel.pleinHistorique
    ? null
    : sousLHabituel
      ? t.appelBelow(formatPoints(appel.ecart), pctFmt.format(appel.habituel))
      : t.appelAbove(formatPoints(appel.ecart), pctFmt.format(appel.habituel))
  return (
    <div className="flex flex-wrap gap-x-10 gap-y-3">
      <div>
        <p className="text-2xs uppercase tracking-wide text-muted-foreground">
          {t.appelLabel(appel.mortsEquipe)}
        </p>
        <p className="text-2xl font-bold tabular-nums text-foreground" data-testid="squad-riposte-taux">
          {pctFmt.format(appel.taux)}{' '}
          <span className="text-sm font-normal text-muted-foreground">{t.appelUnit}</span>
        </p>
        {ecart && (
          <p
            className="text-xs font-medium"
            style={{ color: tokenCssVar(sousLHabituel ? 'outcome-loss' : 'outcome-win') }}
            data-testid="squad-riposte-ecart"
          >
            {ecart}
          </p>
        )}
      </div>
      <div>
        <p className="text-2xs uppercase tracking-wide text-muted-foreground">{t.delaiLabel}</p>
        <p className="text-2xl font-bold tabular-nums text-foreground" data-testid="squad-riposte-delai-median">
          {appel.delaiMedianS != null ? t.delaiValue(secFmt.format(appel.delaiMedianS)) : t.noValue}{' '}
          <span className="text-sm font-normal text-muted-foreground">{t.delaiUnit}</span>
        </p>
        <p className="text-xs text-muted-foreground">
          {t.delaiSub(appel.ripostes, appel.sansReponse)}
        </p>
      </div>
    </div>
  )
}
