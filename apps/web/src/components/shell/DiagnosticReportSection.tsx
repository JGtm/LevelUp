/**
 * DiagnosticReportSection — le rapport de diagnostic de la fenêtre « À propos » (lot L22, capacité
 * `can_report_diagnostic`, application de bureau seulement). Un clic crée une archive des journaux et de
 * l'état de l'application (sans mot de passe ni jeton) ; la fenêtre en montre le chemin, que l'on copie
 * pour le joindre à un signalement.
 *
 * Un refus ou un échec dit sa cause par son code, jamais par le message du serveur. Quand le
 * presse-papier est indisponible, le chemin reste affiché et sélectionnable : aucune fausse
 * confirmation de copie.
 */
import { apiErrorCode } from '@/lib/api/client'
import { useCopyToClipboard } from '@/lib/clipboard/useCopyToClipboard'
import { formatMessage } from '@/lib/i18n/format'
import { commonManifest, type CommonManifestKey } from '@/lib/i18n/generated/common'
import { useAppShellStore } from '@/stores/appShellStore'
import { useDeploymentCapability } from '@/stores/useDeploymentCapability'
import { useDiagnosticReport } from './diagnosticReportQueries'

/** Code de refus d'un rapport demandé alors qu'un autre se crée. */
const ALREADY_RUNNING_CODE = 'diagnostic_already_running'

const BUTTON_CLASS =
  'rounded-md border border-border bg-card px-3 py-1.5 text-sm text-foreground transition-colors hover:bg-accent hover:text-accent-foreground disabled:opacity-60'

export function DiagnosticReportSection() {
  const canReport = useDeploymentCapability('can_report_diagnostic')
  return canReport ? <DiagnosticReportBody /> : null
}

function DiagnosticReportBody() {
  const locale = useAppShellStore((s) => s.locale)
  const t = (key: CommonManifestKey) => formatMessage(commonManifest, key, locale)
  const report = useDiagnosticReport()
  const { copy, copied } = useCopyToClipboard()
  const created = report.data
  const failure = report.isError
    ? t(
        apiErrorCode(report.error) === ALREADY_RUNNING_CODE
          ? 'common.about.diagnostic_already_running'
          : 'common.about.diagnostic_failed',
      )
    : undefined

  return (
    <div className="flex flex-col gap-2 rounded-lg border border-border bg-card px-3 py-2.5">
      <div className="flex flex-wrap items-center gap-3">
        <button type="button" onClick={() => report.mutate()} disabled={report.isPending} className={BUTTON_CLASS}>
          {report.isPending ? t('common.about.diagnostic_running') : t('common.about.diagnostic_button')}
        </button>
        <p className="min-w-0 flex-1 text-xs text-muted-foreground">{t('common.about.diagnostic_hint')}</p>
      </div>
      {failure && (
        <p role="alert" className="text-xs text-destructive">
          {failure}
        </p>
      )}
      {created && (
        <div className="flex flex-col gap-1" role="status">
          <p className="text-xs text-foreground">{t('common.about.diagnostic_done')}</p>
          <p className="break-all font-mono text-xs" data-testid="about-diagnostic-path" title={created.path}>
            {created.path}
          </p>
          <div className="flex flex-wrap items-center gap-3">
            <button type="button" onClick={() => void copy(created.path)} className={`${BUTTON_CLASS} px-2 py-0.5 text-xs`}>
              {copied ? t('common.about.diagnostic_copied_path') : t('common.about.diagnostic_copy_path')}
            </button>
            <p className="text-xs text-muted-foreground">{t('common.about.diagnostic_attach')}</p>
          </div>
        </div>
      )}
    </div>
  )
}
