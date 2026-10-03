/**
 * ExplorerTargetAssists — bloc « Part des assistances » de la section « matchs joués
 * ensemble » de l'encart adversaire (3e rangée, sous « Répartition des résultats »).
 *
 * Rendu 1.A HORIZONTAL (maquette du 2026-09-21, décision utilisateur D17) : deux pistes
 * épaisses superposées, un axe LINÉAIRE commun borné par le plus gros des deux totaux,
 * tranches empilées avec leurs comptes et trait de parité — cf.
 * `ExplorerAssistExchangeBars`. La carte Binôme du hub Relations garde le papillon
 * logarithmique (`_shared/assists/AssistExchangeSummary`) : elle compare des paires
 * entre elles, ce bloc n'en affiche qu'une. `assist_volume_max` n'est donc plus lu ici.
 *
 * L'assistance n'est mesurée que sur les matchs joués dans la même équipe dont le film
 * a été décodé : sans aucun match mesuré, le backend n'envoie pas d'objet et le bloc
 * affiche « — » (jamais « 0 assistance »).
 *
 * Hauteur : colonne d'une rangée `items-stretch` de trois cartes (cf.
 * ExplorerTargetProfileCard) — la carte prend `h-full` et sa zone de contenu `flex-1
 * justify-center`, de sorte que les pistes se CENTRENT verticalement quand la voisine de
 * rangée est plus haute (demande utilisateur du 2026-09-22). Le bandeau de titre reste
 * en haut, et aucune hauteur minimale n'est ajoutée.
 */
import { ExplorerAssistExchangeBars } from './ExplorerAssistExchangeBars'
import { ASSISTS_TEXT } from '@/features/_shared/assists/assistsI18n'
import { useAppShellStore } from '@/stores/appShellStore'
import { formatMessage } from '@/lib/i18n/format'
import { explorerManifest, type ExplorerManifestKey } from '@/lib/i18n/generated/explorer'
import type { ExplorerEncounterStats } from '@/lib/api/types'

interface Props {
  /** Stats de rencontre joueur↔cible. Absentes (ou sans assistances) → état « — ». */
  encounterStats?: ExplorerEncounterStats | null
}

export function ExplorerTargetAssists({ encounterStats }: Props) {
  const appLocale = useAppShellStore((s) => s.locale)
  const t = (key: ExplorerManifestKey) => formatMessage(explorerManifest, key, appLocale)
  const assists = encounterStats?.assists ?? null
  return (
    <div className="flex h-full flex-col overflow-hidden rounded-lg border border-border bg-card" data-testid="explorer-target-assists">
      <div className="border-b border-border px-3 py-2 text-sm font-medium">
        {t('explorer.target_profile.assists_title')}
      </div>
      <div className="flex flex-1 flex-col justify-center p-3">
        {assists ? (
          <ExplorerAssistExchangeBars assists={assists} locale={appLocale} />
        ) : (
          <div className="flex flex-col gap-1">
            <span className="font-mono text-2xl font-bold text-muted-foreground">
              {t('explorer.target_profile.value_unavailable')}
            </span>
            <span className="text-2xs text-muted-foreground">{ASSISTS_TEXT[appLocale].notMeasured}</span>
          </div>
        )}
      </div>
    </div>
  )
}
