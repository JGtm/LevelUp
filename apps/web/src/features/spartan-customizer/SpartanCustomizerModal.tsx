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

type BrowseTab = 'emblem' | 'nameplate'

interface SpartanCatalog {
  emblem_ids?: string[]
  nameplate_ids?: string[]
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

  const [emblemIds, setEmblemIds] = useState<string[]>([])
  const [nameplateIds, setNameplateIds] = useState<string[]>([])
  const [draft, setDraft] = useState<SpartanAppearance>(saved)
  const [tab, setTab] = useState<BrowseTab>('emblem')

  // Catalogue (asset statique same-origin) — listes distinctes emblèmes / bannières.
  useEffect(() => {
    let cancelled = false
    fetch(`${SPARTAN_BASE}/spartan-catalog.json`)
      .then((r) => r.json() as Promise<SpartanCatalog>)
      .then((cat) => {
        if (cancelled) return
        const eIds = cat.emblem_ids ?? []
        const nIds = cat.nameplate_ids ?? []
        setEmblemIds(eIds)
        setNameplateIds(nIds)
        // Présélection si rien encore choisi (emblème et bannière indépendants).
        setDraft((d) => ({
          ...d,
          emblemId: d.emblemId ?? eIds[0] ?? null,
          nameplateId: d.nameplateId ?? nIds[0] ?? null,
        }))
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

  const isEmblemTab = tab === 'emblem'
  const browseIds = isEmblemTab ? emblemIds : nameplateIds
  const browseActiveId = isEmblemTab ? draft.emblemId : draft.nameplateId
  const thumbDir = isEmblemTab ? 'emblems_thumb_colored' : 'nameplates_thumb_colored'

  function pickItem(id: string) {
    setDraft((d) => (isEmblemTab ? { ...d, emblemId: id } : { ...d, nameplateId: id }))
  }

  function handleSave() {
    setAppearance(draft)
    onClose()
  }

  function tabClass(active: boolean) {
    return [
      '-mb-px border-b-2 px-3 py-1.5 text-sm font-medium transition-colors',
      active
        ? 'border-primary text-foreground'
        : 'border-transparent text-muted-foreground hover:text-foreground',
    ].join(' ')
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

        {/* Aperçu : emblème (emblemId) + bannière (nameplateId) — choix indépendants */}
        <div className="flex flex-wrap items-center justify-center gap-8 border-b border-border bg-muted/30 px-5 py-5">
          {draft.emblemId && (
            <RecoloredMask
              src={`${SPARTAN_BASE}/emblems/${draft.emblemId}.png`}
              colors={colors}
              alt={t('home.spartan_customizer.emblem_preview')}
              className="h-28 w-28 object-contain"
            />
          )}
          {draft.nameplateId && (
            <RecoloredMask
              src={`${SPARTAN_BASE}/nameplates/${draft.nameplateId}.png`}
              colors={colors}
              alt={t('home.spartan_customizer.nameplate_preview')}
              className="h-28 w-auto max-w-full object-contain"
            />
          )}
          {!draft.emblemId && !draft.nameplateId && <div className="h-28" />}
        </div>

        {/* Corps : couleurs + onglets (Emblème / Bannière) + grille */}
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
            <div role="tablist" className="mb-3 flex gap-1 border-b border-border">
              <button
                type="button"
                role="tab"
                aria-selected={isEmblemTab}
                onClick={() => setTab('emblem')}
                className={tabClass(isEmblemTab)}
              >
                {t('home.spartan_customizer.tab_emblem')}
              </button>
              <button
                type="button"
                role="tab"
                aria-selected={!isEmblemTab}
                onClick={() => setTab('nameplate')}
                className={tabClass(!isEmblemTab)}
              >
                {t('home.spartan_customizer.tab_nameplate')}
              </button>
            </div>

            <div
              className={
                isEmblemTab
                  ? 'grid grid-cols-[repeat(auto-fill,minmax(56px,1fr))] gap-2'
                  : 'grid grid-cols-[repeat(auto-fill,minmax(150px,1fr))] gap-2'
              }
            >
              {browseIds.map((id) => {
                const active = browseActiveId === id
                return (
                  <button
                    key={id}
                    type="button"
                    onClick={() => pickItem(id)}
                    aria-pressed={active}
                    aria-label={`#${id}`}
                    className={[
                      'overflow-hidden rounded-lg border bg-muted/40 transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-ring',
                      isEmblemTab ? 'aspect-square' : 'aspect-[200/42]',
                      active
                        ? 'border-primary ring-2 ring-ring'
                        : 'border-border hover:border-primary/60',
                    ].join(' ')}
                  >
                    <img
                      src={`${SPARTAN_BASE}/${thumbDir}/${id}.png`}
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
              setDraft((d) => ({
                ...DEFAULT_SPARTAN_APPEARANCE,
                emblemId: d.emblemId,
                nameplateId: d.nameplateId,
              }))
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
