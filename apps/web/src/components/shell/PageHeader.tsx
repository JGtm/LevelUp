/**
 * PageHeader — entête uniforme pour les pages de contenu.
 */

interface PageHeaderProps {
  title: string
  subtitle?: string
  actions?: React.ReactNode
  inset?: boolean
}

export function PageHeader({ title, subtitle, actions, inset = true }: PageHeaderProps) {
  return (
    <div
      className={[
        'relative overflow-hidden rounded-[28px] border border-border bg-card/92 px-6 py-6 shadow-sm backdrop-blur sm:px-8',
        inset ? 'mx-6' : '',
      ].join(' ')}
    >
      <div className="pointer-events-none absolute inset-x-0 top-0 h-px bg-gradient-to-r from-transparent via-violet-300/80 to-transparent" />

      <div className="flex flex-col gap-4 md:flex-row md:items-end md:justify-between">
        <div className="max-w-3xl">
          <p className="text-[11px] font-semibold uppercase tracking-[0.24em] text-muted-foreground">
            LevelUp
          </p>
          <h1 className="mt-2 text-2xl font-semibold tracking-tight text-foreground sm:text-3xl">
            {title}
          </h1>
          {subtitle && <p className="mt-2 text-sm leading-6 text-muted-foreground">{subtitle}</p>}
        </div>

        {actions && <div className="flex flex-wrap items-center gap-2">{actions}</div>}
      </div>
    </div>
  )
}
