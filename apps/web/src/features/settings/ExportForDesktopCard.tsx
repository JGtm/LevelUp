/**
 * ExportForDesktopCard — « Exporter vers l'application de bureau » (lot L22, capacité
 * `can_export_for_desktop`, vraie sur le site hébergé seulement). Génère un jeton d'export à usage
 * unique (15 minutes) que l'on colle dans l'application de bureau, Réglages > Application.
 *
 * Le jeton s'affiche une fois, avec « Copier » et son échéance ; « Masquer » l'efface de l'écran.
 * Un refus (un export par heure) dit sa cause par son code, écrite dans chaque langue.
 */
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { apiErrorCode } from '@/lib/api/client'
import { useCopyToClipboard } from '@/lib/clipboard/useCopyToClipboard'
import { formatDateTime, intlLocale } from '@/lib/formatters'
import { useAppShellStore } from '@/stores/appShellStore'
import { useDeploymentCapability } from '@/stores/useDeploymentCapability'
import { fill } from './fillTemplate'
import { useGenerateExportToken } from './exportForDesktopQueries'
import { getExportForDesktopText } from './exportForDesktopText'

export function ExportForDesktopCard() {
  const canExport = useDeploymentCapability('can_export_for_desktop')
  return canExport ? <ExportForDesktopBody /> : null
}

function ExportForDesktopBody() {
  const locale = useAppShellStore((s) => s.locale)
  const t = getExportForDesktopText(locale)
  const generate = useGenerateExportToken()
  const { copy, copied } = useCopyToClipboard()
  const issued = generate.data
  const errorCode = generate.isError ? apiErrorCode(generate.error) : undefined
  const errorText = generate.isError ? ((errorCode && t.errors[errorCode]) ?? t.unknownError) : undefined

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">{t.title}</CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        <p className="text-sm text-muted-foreground">{t.intro}</p>
        {!issued && (
          <Button size="sm" variant="outline" disabled={generate.isPending} onClick={() => generate.mutate()}>
            {generate.isPending ? t.generating : t.generate}
          </Button>
        )}
        {errorText && (
          <p role="alert" className="text-xs text-destructive">
            {errorText}
          </p>
        )}
        {issued && (
          <div className="space-y-2">
            <p className="text-sm text-foreground">{t.tokenLabel}</p>
            <code
              className="block break-all rounded border border-border bg-muted px-2 py-1 font-mono text-xs text-foreground"
              data-testid="export-token"
            >
              {issued.token}
            </code>
            <div className="flex flex-wrap gap-2">
              <Button size="sm" variant="outline" onClick={() => void copy(issued.token)}>
                {copied ? t.copied : t.copy}
              </Button>
              <Button size="sm" variant="ghost" onClick={() => generate.reset()}>
                {t.hide}
              </Button>
            </div>
            <p className="text-xs text-foreground">
              {fill(t.expiry, { time: formatDateTime(issued.expires_at, intlLocale(locale)) })}
            </p>
            <p className="text-xs text-muted-foreground">{t.instruction}</p>
            <p className="text-xs text-muted-foreground">{t.shownOnce}</p>
          </div>
        )}
      </CardContent>
    </Card>
  )
}
