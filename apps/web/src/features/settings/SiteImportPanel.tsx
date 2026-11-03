/**
 * SiteImportPanel — importer ses données depuis l'instance hébergée (lot L22, capacité
 * `can_import_from_site`, application de bureau seulement). Le même panneau sert les Réglages
 * (onglet Application, `variant="card"`) et l'assistant de premier lancement (`variant="optional"` :
 * un bloc replié que l'on peut ignorer, qui ne bloque jamais l'assistant).
 *
 * L'utilisateur colle l'adresse du site et le jeton d'export généré sur le site. Le jeton est à usage
 * unique : il quitte le champ dès l'envoi et n'est jamais conservé. La progression se relit toute
 * seule ; le bilan dit combien de matchs ont été ajoutés, un échec dit sa cause par son code.
 */
import { useId, useState } from 'react'
import { intlLocale } from '@/lib/formatters'
import type { Locale } from '@/lib/i18n/locale'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Spinner } from '@/components/ui/spinner'
import { useAppShellStore } from '@/stores/appShellStore'
import { useDeploymentCapability } from '@/stores/useDeploymentCapability'
import { fill } from './fillTemplate'
import { getSiteImportText, type SiteImportText } from './siteImportText'
import { useSiteImportForm } from './useSiteImportForm'

const INPUT_CLASS =
  'w-full rounded border border-border bg-background px-2 py-1 text-sm text-foreground disabled:opacity-60'

interface SiteImportPanelProps {
  variant?: 'card' | 'optional'
}

export function SiteImportPanel({ variant = 'card' }: SiteImportPanelProps) {
  const canImport = useDeploymentCapability('can_import_from_site')
  return canImport ? <SiteImportBody variant={variant} /> : null
}

/** Progression, bilan ou échec de l'import ; rien tant qu'aucun import n'a eu lieu. */
function ImportOutcome({
  t,
  locale,
  phase,
  matches,
  errorMessage,
}: {
  t: SiteImportText
  locale: Locale
  phase: string | undefined
  matches: number
  errorMessage: string | undefined
}) {
  if (errorMessage) {
    return (
      <p role="alert" className="text-xs text-destructive">
        {errorMessage}
      </p>
    )
  }
  if (phase === 'downloading' || phase === 'importing') {
    return (
      <div role="status">
        <Spinner size="sm" label={phase === 'downloading' ? t.downloading : t.importing} />
      </div>
    )
  }
  if (phase === 'done') {
    return (
      <p role="status" className="text-sm text-success">
        {fill(new Intl.PluralRules(intlLocale(locale)).select(matches) === 'one' ? t.addedOne : t.addedMany, {
          count: matches.toLocaleString(intlLocale(locale)),
        })}
      </p>
    )
  }
  return null
}

/** Un champ étiqueté de la saisie : le texte (adresse, gamertag) ou le jeton (type password, jamais mémorisé). */
function TextField(props: {
  label: string
  value: string
  disabled: boolean
  onChange: (v: string) => void
  secret?: boolean
  placeholder?: string
  hint?: string
}) {
  const id = useId()
  return (
    <div className="space-y-1">
      <label className="text-sm text-foreground" htmlFor={id}>
        {props.label}
      </label>
      <input
        id={id}
        type={props.secret ? 'password' : 'text'}
        autoComplete="off"
        className={INPUT_CLASS}
        value={props.value}
        placeholder={props.placeholder}
        disabled={props.disabled}
        onChange={(e) => props.onChange(e.target.value)}
      />
      {props.hint && <p className="text-xs text-muted-foreground">{props.hint}</p>}
    </div>
  )
}

function ImportForm({ t, locale }: { t: SiteImportText; locale: Locale }) {
  const form = useSiteImportForm(t)
  return (
    <form className="space-y-3" onSubmit={form.submit} autoComplete="off">
      <p className="text-sm text-muted-foreground">{t.intro}</p>
      <TextField
        label={t.urlLabel}
        value={form.url}
        placeholder={t.urlPlaceholder}
        disabled={form.running}
        onChange={form.setUrl}
      />
      <TextField
        secret
        label={t.tokenLabel}
        value={form.token}
        hint={t.tokenHint}
        disabled={form.running}
        onChange={form.setToken}
      />
      {form.askGamertag && (
        <TextField label={t.gamertagLabel} value={form.gamertag} disabled={form.running} onChange={form.setGamertag} />
      )}
      <Button type="submit" size="sm" variant="outline" disabled={!form.canSubmit}>
        {form.starting ? t.starting : t.importButton}
      </Button>
      <ImportOutcome
        t={t}
        locale={locale}
        phase={form.status?.phase}
        matches={form.status?.matches ?? 0}
        errorMessage={form.errorMessage}
      />
    </form>
  )
}

function SiteImportBody({ variant }: SiteImportPanelProps) {
  const locale = useAppShellStore((s) => s.locale)
  const t = getSiteImportText(locale)
  const [open, setOpen] = useState(false)
  if (variant === 'optional') {
    return (
      <div className="rounded-md border border-border">
        <button
          type="button"
          aria-expanded={open}
          onClick={() => setOpen((v) => !v)}
          className="w-full px-3 py-2 text-left text-sm text-foreground transition-colors hover:bg-accent"
        >
          {t.optionalToggle}
        </button>
        {open && (
          <div className="border-t border-border p-3">
            <ImportForm t={t} locale={locale} />
          </div>
        )}
      </div>
    )
  }
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">{t.title}</CardTitle>
      </CardHeader>
      <CardContent>
        <ImportForm t={t} locale={locale} />
      </CardContent>
    </Card>
  )
}
