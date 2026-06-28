import { useEffect, useState } from 'react'

import { formatMessage } from '@/lib/i18n/format'
import { homeManifest, type HomeManifestKey } from '@/lib/i18n/generated/home'
import { SPARTAN_PALETTE } from '@/lib/accessibility/palettes/spartan'
import { useAppShellStore } from '@/stores/appShellStore'

import { RecoloredMask } from './RecoloredMask'
import {
  DEFAULT_SPARTAN_APPEARANCE,
  useSpartanAppearanceStore,
  type SpartanAppearance,
} from './store'

const SPARTAN_BASE = '/titles/halo_5/spartan'

interface SpartanCatalog {
  ids?: string[]
}

/** Ligne de sélection de couleur (palette de pastilles). */
function ColorRow({
  label,
  value,
  onPick,
}: {
  label: string
  value: string
  onPick: (hex: string) => void
}) {
  return (
    <div className="space-y-1.5">
      <p className="text-xs font-medium text-muted-foreground">{label}</p>
      <div className="flex flex-wrap gap-1.5">
        {SPARTAN_PALETTE.map((c) => {
          const active = value.toLowerCase() === c.hex.toLowerCase()
          return (
            <button
              key={c.id}
              type="button"
              onClick={() => onPick(c.hex)}
              aria-label={c.id}
              aria-pressed={active}
              className={[
                'h-6 w-6 rounded-full border transition-transform hover:scale-110 focus:outline-none focus-visible:ring-2 focus-visible:ring-ring',
                active ? 'ring-2 ring-ring ring-offset-1 ring-offset-card' : 'border-border',
              ].join(' ')}
              // Couleur de DONNÉES (palette joueur) — exception tokens documentée (cf. spartan.ts).
              style={{ backgroundColor: c.hex }}
            />
          )
        })}
      </div>
    </div>
  )
}

export function SpartanCustomizerModal({ onClose }: { onClose: () => void }) {
  const locale = useAppShellStore((s) => s.locale)
  const t = (key: HomeManifestKey) => formatMessage(homeManifest, key, locale)

  const saved = useSpartanAppearanceStore((s) => s.appearance)
  const setAppearance = useSpartanAppearanceStore((s) => s.setAppearance)

  const [ids, setIds] = useState<string[]>([])
  const [draft, setDraft] = useState<SpartanAppearance>(saved)

  // Catalogue d'emblèmes (asset statique same-origin)
  useEffect(() => {
    let cancelled = false
    fetch(`${SPARTAN_BASE}/spartan-catalog.json`)
      .then((r) => r.json() as Promise<SpartanCatalog>)
      .then((cat) => {
        if (cancelled) return
        const list = cat.ids ?? []
        setIds(list)
        // Présélection si aucun emblème encore choisi.
        setDraft((d) => (d.emblemId ? d : { ...d, emblemId: list[0] ?? null }))
      })
      .catch(() => {
        /* catalogue indisponible → grille vide, modale reste utilisable */
      })
    return () => {
      cancelled = true
    }
  }, [])

  // Fermeture au clavier
  useEffect(() => {
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') onClose()
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [onClose])

  const colors = {
    primary: draft.primary,
    secondary: draft.secondary,
    tertiary: draft.tertiary,
  }
  const selectedId = draft.emblemId

  function handleSave() {
    setAppearance(draft)
    onClose()
  }

  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-label={t('home.spartan_customizer.title')}
      data-testid="spartan-customizer-modal"
      className="fixed inset-0 z-50 flex items-center justify-center bg-background/85 p-4 backdrop-blur-sm"
      onClick={onClose}
    >
      <div
        className="relative flex max-h-[90vh] w-full max-w-4xl flex-col overflow-hidden rounded-2xl border border-border bg-card text-foreground shadow-2xl"
        onClick={(e) => e.stopPropagation()}
      >
        {/* En-tête */}
        <div className="flex items-start justify-between gap-4 border-b border-border bg-muted/60 px-5 py-3">
          <div className="min-w-0">
            <h2 className="text-lg font-semibold sm:text-xl">
              {t('home.spartan_customizer.title')}
            </h2>
            <p className="text-sm text-muted-foreground">
              {t('home.spartan_customizer.description')}
            </p>
          </div>
          <button
            type="button"
            onClick={onClose}
            aria-label={t('home.spartan_customizer.close')}
            className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full text-xl leading-none text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
          >
            ×
          </button>
        </div>

        {/* Aperçu : emblème + nameplate côte à côte (non collés) */}
        <div className="flex flex-wrap items-center justify-center gap-8 border-b border-border bg-muted/30 px-5 py-5">
          {selectedId ? (
            <>
              <RecoloredMask
                src={`${SPARTAN_BASE}/emblems/${selectedId}.png`}
                colors={colors}
                alt={t('home.spartan_customizer.emblem_preview')}
                className="h-28 w-28 object-contain"
              />
              <RecoloredMask
                src={`${SPARTAN_BASE}/nameplates/${selectedId}.png`}
                colors={colors}
                alt={t('home.spartan_customizer.nameplate_preview')}
                className="h-28 w-auto max-w-full object-contain"
              />
            </>
          ) : (
            <div className="h-28" />
          )}
        </div>

        {/* Corps : couleurs + grille d'emblèmes */}
        <div className="flex min-h-0 flex-1 flex-col gap-4 overflow-y-auto px-5 py-4">
          <div className="grid gap-3 sm:grid-cols-3">
            <ColorRow
              label={t('home.spartan_customizer.primary')}
              value={draft.primary}
              onPick={(hex) => setDraft((d) => ({ ...d, primary: hex }))}
            />
            <ColorRow
              label={t('home.spartan_customizer.secondary')}
              value={draft.secondary}
              onPick={(hex) => setDraft((d) => ({ ...d, secondary: hex }))}
            />
            <ColorRow
              label={t('home.spartan_customizer.tertiary')}
              value={draft.tertiary}
              onPick={(hex) => setDraft((d) => ({ ...d, tertiary: hex }))}
            />
          </div>

          <div>
            <p className="mb-2 text-sm font-medium">
              {t('home.spartan_customizer.choose_emblem')}
            </p>
            <div className="grid grid-cols-[repeat(auto-fill,minmax(56px,1fr))] gap-2">
              {ids.map((id) => {
                const active = selectedId === id
                return (
                  <button
                    key={id}
                    type="button"
                    onClick={() => setDraft((d) => ({ ...d, emblemId: id }))}
                    aria-pressed={active}
                    aria-label={`#${id}`}
                    className={[
                      'aspect-square overflow-hidden rounded-lg border bg-muted/40 transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-ring',
                      active ? 'border-primary ring-2 ring-ring' : 'border-border hover:border-primary/60',
                    ].join(' ')}
                  >
                    <img
                      src={`${SPARTAN_BASE}/emblems_thumb_colored/${id}.png`}
                      alt={`#${id}`}
                      loading="lazy"
                      decoding="async"
                      className="h-full w-full object-contain"
                    />
                  </button>
                )
              })}
            </div>
          </div>
        </div>

        {/* Pied : reset / annuler / enregistrer */}
        <div className="flex items-center justify-end gap-2 border-t border-border bg-muted/40 px-5 py-3">
          <button
            type="button"
            onClick={() =>
              setDraft((d) => ({ ...DEFAULT_SPARTAN_APPEARANCE, emblemId: d.emblemId }))
            }
            className="rounded-lg px-3 py-1.5 text-sm text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
          >
            {t('home.spartan_customizer.reset')}
          </button>
          <button
            type="button"
            onClick={onClose}
            className="rounded-lg border border-border px-3 py-1.5 text-sm font-medium transition-colors hover:bg-muted"
          >
            {t('home.spartan_customizer.cancel')}
          </button>
          <button
            type="button"
            onClick={handleSave}
            data-testid="spartan-customizer-save"
            className="rounded-lg bg-primary px-3 py-1.5 text-sm font-medium text-primary-foreground transition-opacity hover:opacity-90"
          >
            {t('home.spartan_customizer.save')}
          </button>
        </div>
      </div>
    </div>
  )
}
