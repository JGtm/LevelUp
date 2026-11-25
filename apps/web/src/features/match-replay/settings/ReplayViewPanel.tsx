/**
 * ReplayViewPanel — LE PANNEAU DE LA VUE du rejeu : les préréglages souples et Mes vues en tête, puis cinq rubriques
 * en onglets verticaux (Joueurs, Carte, Objets, Fil, Son), puis, au Plein cadre, la ligne « Lecture » (barre fine),
 * et le pied « Réinitialiser ». Il REMPLACE les dix sections d'avant (plan « réglages d'affichage », variante A de la
 * maquette validée) : AUCUN réglage n'est réinventé, chacun garde sa règle, sa clé et son effet — seul change ce qui les
 * commande. Tout se déduit de la table des réglages (`settingsTable`) et du modèle du panneau (`viewPanelModel`).
 *
 * PANNEAU EN SURIMPRESSION, PAS UNE MODALE. Il se pose SUR le rejeu : le canvas ne se retaille pas à l'ouverture. On
 * en sort par Échap, par le bouton ou en cliquant dehors, et le focus entre au panneau à l'ouverture ; ni fond
 * assombri, ni piège de focus, ni lecture suspendue. Échap ferme la couche la plus intérieure d'abord (le champ de nom
 * d'une vue, la confirmation de la remise à zéro), le panneau à la frappe suivante.
 *
 * IL SORT DU CADRE : la carte porte `overflow-hidden`, que ne franchit aucun `z-index`. Il se rend en PORTAIL sur
 * `body`, en position fixe, sous le bouton « Vue » (`useAnchoredPanel`), et défile EN INTERNE quand l'écran est bas.
 */
import { useEffect, useRef, useState, type RefObject } from 'react'
import { createPortal } from 'react-dom'

import { useDismissable } from '@/lib/hooks/useDismissable'

import { REPLAY_TEXT } from '../i18n/i18n'
import { ToggleRow } from './ReplaySettingRows'
import { RubricTabs } from './ReplayRubricTabs'
import { ViewChips } from './ReplayViewChips'
import { defOf, type RubricId, type SettingDef } from './settingsTable'
import { persistViewTab, readViewTab } from './settingsStore'
import { useAnchoredPanel } from './useAnchoredPanel'
import type { ReplayViewPanelModel } from './viewPanelModel'

/**
 * LA LARGEUR DU PANNEAU, en px et pas en classe : le hook de placement en a besoin pour aligner son bord droit sur
 * celui du bouton. 104 px d'onglets, 10 px d'écart et le reste à la rubrique.
 */
const PANEL_WIDTH = 416

interface Props {
  model: ReplayViewPanelModel
  onClose: () => void
  /** Le bouton qui a ouvert le panneau : exclu du « clic dehors », il reprend le focus à la fermeture (côté appelant). */
  triggerRef?: RefObject<HTMLElement | null>
}

/**
 * La confirmation de la remise à zéro : une couche d'Échap (la pile de `useDismissable` la ferme SEULE, le panneau à la
 * frappe suivante). Aucun clic dehors ne la ferme.
 */
function ResetConfirm({ model, onDone, onCancel }: { model: ReplayViewPanelModel; onDone: () => void; onCancel: () => void }) {
  const t = REPLAY_TEXT[model.locale].settings
  const ref = useDismissable<HTMLDivElement>(true, onCancel, { outside: null })
  return (
    <div ref={ref} data-replay-escape-layer="" className="flex flex-1 items-center gap-2">
      <span>{t.resetAsk}</span>
      <span className="flex-1" />
      <button
        type="button"
        autoFocus
        onClick={() => {
          model.views.reset()
          onDone()
        }}
        className="inline-flex h-6 cursor-pointer items-center rounded-md border border-primary bg-primary px-2 text-[11.5px] text-primary-foreground hover:bg-primary/90 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/40"
      >
        {t.resetConfirm}
      </button>
      <button
        type="button"
        onClick={onCancel}
        className="inline-flex h-6 cursor-pointer items-center rounded-md px-2 text-[11.5px] text-muted-foreground hover:bg-accent hover:text-accent-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/40"
      >
        {t.cancel}
      </button>
    </div>
  )
}

/** Le pied : « Réinitialiser » et sa confirmation, qui garde Mes vues, la vue d'ouverture et les épingles. */
function PanelFooter({ model, onStatus }: { model: ReplayViewPanelModel; onStatus: (m: string) => void }) {
  const t = REPLAY_TEXT[model.locale].settings
  const [confirming, setConfirming] = useState(false)
  return (
    <div className="flex items-center gap-2 border-t border-border pt-2.5 text-xs">
      {confirming ? (
        <ResetConfirm
          model={model}
          onDone={() => {
            setConfirming(false)
            onStatus(t.statusReset)
          }}
          onCancel={() => setConfirming(false)}
        />
      ) : (
        <button
          type="button"
          onClick={() => setConfirming(true)}
          className="inline-flex h-6 cursor-pointer items-center rounded-md border border-border px-2 text-[11.5px] hover:bg-accent hover:text-accent-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/40"
        >
          {t.reset}
        </button>
      )}
    </div>
  )
}

export function ReplayViewPanel({ model, onClose, triggerRef }: Props) {
  const t = REPLAY_TEXT[model.locale]
  const panelRef = useRef<HTMLDivElement>(null)
  const [tab, setTab] = useState<RubricId>(readViewTab)
  const [status, setStatus] = useState('')
  const pos = useAnchoredPanel(triggerRef, PANEL_WIDTH)

  // LE FOCUS ENTRE À L'OUVERTURE : le panneau se pose SUR la carte ; sans cela un lecteur au clavier resterait derrière lui.
  useEffect(() => {
    panelRef.current?.focus({ preventScroll: true })
  }, [])
  // ÉCHAP ET CLIC DEHORS, écoutés au document : un voile transparent avalerait le premier clic et couvrirait la barre.
  useDismissable(true, onClose, { ref: panelRef, outside: 'pointerdown', ignore: triggerRef })

  function onPin(def: SettingDef) {
    if (model.views.pin(def.id) === 'full') setStatus(t.settings.statusPinFull)
  }

  const thin = model.ctx.thinBarAvailable
  // LE PORTAIL EST LA CONDITION, PAS UN DÉTAIL D'IMPLÉMENTATION : aucun `z-index` ne fait sortir un enfant d'un ancêtre qui découpe.
  return createPortal(
    <div
      ref={panelRef}
      tabIndex={-1}
      role="dialog"
      // UNE COUCHE QU'ÉCHAP FERME AVANT LE MODE PLEIN ÉCRAN (cf. useReplayFullscreen).
      data-replay-escape-layer=""
      aria-label={t.viewButton}
      style={{ left: pos.left, top: pos.top, width: PANEL_WIDTH, maxHeight: pos.maxHeight }}
      className="fixed z-50 flex max-w-[calc(100vw-1rem)] flex-col gap-2.5 overflow-y-auto rounded-lg border border-border bg-card px-3 py-3 text-sm shadow-xl outline-none"
    >
      <div className="flex items-center justify-between">
        <h2 className="text-sm font-semibold">{t.viewButton}</h2>
        <button
          type="button"
          onClick={onClose}
          className="flex h-6 w-6 cursor-pointer items-center justify-center rounded text-muted-foreground transition-colors hover:bg-muted hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/40"
          aria-label={t.viewClose}
        >
          ×
        </button>
      </div>
      <ViewChips model={model} onStatus={setStatus} />
      <div className="h-px bg-border" />
      <RubricTabs
        model={model}
        tab={tab}
        onTab={(r) => {
          setTab(r)
          persistViewTab(r)
        }}
        onPin={onPin}
      />
      {thin && (
        <section className="space-y-0.5">
          <div className="h-px bg-border" />
          <h3 className="pt-2 text-xs font-medium text-muted-foreground">{t.settings.readingTitle}</h3>
          <ToggleRow def={defOf('thinBar')} model={model} onPin={onPin} />
        </section>
      )}
      <p role="status" aria-live="polite" className={status ? 'text-[11px] text-muted-foreground' : 'sr-only'}>{status}</p>
      <PanelFooter model={model} onStatus={setStatus} />
    </div>,
    document.body,
  )
}
