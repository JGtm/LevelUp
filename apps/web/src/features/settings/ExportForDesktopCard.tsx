/**
 * ExportForDesktopCard — « Exporter vers l'application de bureau » (lot L22, capacité
 * `can_export_for_desktop`, vraie sur le site hébergé seulement). Génère un jeton d'export à usage
 * unique (15 minutes) que l'on colle dans l'application de bureau, Réglages > Application.
 *
 * Le jeton s'affiche une fois, avec « Copier » et son échéance ; « Masquer » l'efface de l'écran.
 * Masquée pour un compte sans identité Xbox liée (comme PoolSharingCard) : le serveur ne délivre de jeton qu'à un
 * joueur relié à Xbox (401 `halo_auth_required` sinon), la carte n'offre donc pas un bouton qui ne peut qu'échouer.
 * Le jeton se délivre sans limite de débit : le quota d'un export par heure se tient au téléchargement, côté
 * application de bureau, pas ici.
 */
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { useCopyToClipboard } from '@/lib/clipboard/useCopyToClipboard'
import { formatDateTime, intlLocale } from '@/lib/formatters'
import { useAppShellStore } from '@/stores/appShellStore'
import { useDeploymentCapability } from '@/stores/useDeploymentCapability'
import { fillTemplate } from '@/lib/i18n/template'
import { useGenerateExportToken } from './exportForDesktopQueries'
import { getExportForDesktopText } from './exportForDesktopText'

export function ExportForDesktopCard() {
  const canExport = useDeploymentCapability('can_export_for_desktop')
  const linkedToXbox = useAppShellStore((s) => s.linkedHaloIdentity !== null)
  return canExport && linkedToXbox ? <ExportForDesktopBody /> : null
}

function ExportForDesktopBody() {
  const locale = useAppShellStore((s) => s.locale)
  const t = getExportForDesktopText(locale)
  const generate = useGenerateExportToken()
  const { copy, copied } = useCopyToClipboard()
  const issued = generate.data
  const errorText = generate.isError ? t.unknownError : undefined

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
              {fillTemplate(t.expiry, { time: formatDateTime(issued.expires_at, intlLocale(locale)) })}
            </p>
            <p className="text-xs text-muted-foreground">{t.instruction}</p>
            <p className="text-xs text-muted-foreground">{t.shownOnce}</p>
          </div>
        )}
      </CardContent>
    </Card>
  )
}
