/**
 * ExplorerTargetAssists — bloc « Part des assistances » de la section « matchs joués
 * ensemble » de l'encart adversaire (3e rangée, sous « Répartition des résultats »).
 *
 * MÊME figure que la carte Binôme du hub Relations (AssistExchangeSummary) : barre
 * papillon sous deux têtes, puis « total · part » de chaque côté. Même agrégat côté
 * backend (RelationAssists), même borne d'échelle logarithmique — servie par l'API
 * (`assist_volume_max` = plus gros volume d'un sens parmi TOUTES les relations du
 * joueur), parce qu'ici une seule paire est affichée et que se borner à elle-même
 * remplirait toujours la demi-barre.
 *
 * L'assistance n'est mesurée que sur les matchs joués dans la même équipe dont le film
 * a été décodé : sans aucun match mesuré, le backend n'envoie pas d'objet et le bloc
 * affiche « — » (jamais « 0 assistance »).
 *
 * Pas de `h-full` : empilé sous « Répartition des résultats », chaque bloc garde la
 * hauteur de son contenu ; c'est leur SOMME qui donne la hauteur de la rangée, à
 * laquelle « Portée des frags » s'étire.
 */
import { AssistExchangeSummary } from '@/features/_shared/assists/AssistExchangeSummary'
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
  const volumeMax = encounterStats?.assist_volume_max ?? 0
  return (
    <div className="overflow-hidden rounded-lg border border-border bg-card" data-testid="explorer-target-assists">
      <div className="border-b border-border px-3 py-2 text-sm font-medium">
        {t('explorer.target_profile.assists_title')}
      </div>
      <div className="p-3">
        {assists ? (
          <AssistExchangeSummary assists={assists} volumeMax={volumeMax} locale={appLocale} />
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
