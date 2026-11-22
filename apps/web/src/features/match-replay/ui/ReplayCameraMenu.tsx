/**
 * ReplayCameraMenu — LE MENU DE LA CAMÉRA de la barre de lecture compacte (Cinéma et Plein cadre) : un
 * bouton icône seule, à gauche du son, qui ouvre un petit panneau à DEUX listes.
 *
 *  - « Suivre » : « Aucun », puis les joueurs du match rangés par camp, le joueur suivi marqué. Choisir un
 *    joueur est EXACTEMENT le geste du bouton « Suivre » de la fiche (`CameraFollow.onFollow`) ; « Aucun »
 *    éteint le suivi. Aucune règle neuve : le modèle du panneau de joueur décide (suivre un joueur masqué
 *    lève son masquage, un geste manuel sur la carte éteint le suivi).
 *  - « Point de vue » : par les yeux de qui se lit le match, la liste que la frise propose déjà
 *    (`timeline.viewpointGroups`, jamais reconstruite ici). Une option inerte (joueur sans ligne de tableau
 *    de score) l'est ici comme là-bas.
 *
 * Les deux listes ne se mêlent pas : suivre un joueur ne change pas le point de vue, et inversement. Une liste
 * vide n'est pas rendue ; sans aucune liste, le bouton non plus (jamais une commande qui n'ouvre rien).
 *
 * LE NOM DU BOUTON dit l'état : « Caméra : suivre <nom> » quand un joueur est suivi, « Caméra » sinon ; le
 * bouton prend le fond d'accent tant que la caméra suit quelqu'un.
 *
 * LA FERMETURE est celle de `useDismissable` (Échap, clic dehors au `pointerdown`, comme les autres menus de
 * la barre). Le panneau se signale au mode plein écran (`data-replay-escape-layer`) : Échap le ferme AVANT de
 * sortir d'un Plein cadre. Un choix ferme le panneau ; après un choix ou Échap, le FOCUS REVIENT AU BOUTON — un
 * bouton ne retient pas les raccourcis du lecteur (Espace lance ou arrête toujours la lecture). Un clic dehors
 * laisse le focus où l'on a cliqué. Panneau ouvert, les touches utilisées dans ses listes (Entrée, Espace,
 * flèches) ne traversent pas vers le clavier du lecteur ; Échap et Tab passent.
 *
 * Le composant est monté par la fonction de rendu de l'emplacement de la barre (sans hook) : l'état ouvert / fermé
 * vit ici, les données arrivent par props.
 */
import { useId, useRef, useState, type KeyboardEvent, type ReactNode } from 'react'

import { isEscapeKey, useDismissable } from '@/lib/hooks/useDismissable'

import type { ReplayTimeline } from '../hooks/useReplayTimeline'
import { REPLAY_TEXT, type ReplayLocale } from '../i18n/i18n'
import { followedLabelOf, type CameraFollow } from '../model/cameraMenu'
import { COTES_MENU_CAMERA as C } from '../model/cotesMenuCamera'
import type { ViewpointOptionGroup } from '../model/viewpointOptions'
import { COMPACT_BUTTON_CLASS, COMPACT_BUTTON_SQUARE } from './compactButton'
import { VideoCameraIcon } from './ReplayTransportIcons'

export interface ReplayCameraMenuProps {
  /** Le point de vue courant, les camps du match et le geste qui pose le point de vue (la source de la frise). */
  timeline: Pick<ReplayTimeline, 'viewpoint' | 'viewpointGroups' | 'onSelectViewpoint'>
  /** Le joueur que la caméra suit et le geste « Suivre ce joueur ». */
  cameraFollow: CameraFollow
  locale: ReplayLocale
}

/** Une touche du panneau reste au panneau, sauf Échap (qui le ferme) et Tab (qui en sort). */
function keepKeysInPanel(e: KeyboardEvent<HTMLElement>) {
  if (!isEscapeKey(e) && e.key !== 'Tab') e.stopPropagation()
}

function CheckGlyph() {
  return (
    <svg viewBox="0 0 16 16" className="ml-auto h-3 w-3 shrink-0" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="M3.5 8.5 6.5 11.5 12.5 4.5" />
    </svg>
  )
}

interface EntryProps {
  label: string
  title?: string
  selected: boolean
  disabled?: boolean
  onChoose: () => void
}

/** Une entrée de liste : un bouton à état (`aria-pressed`), la coche à droite quand elle est courante. */
function Entry({ label, title, selected, disabled = false, onChoose }: EntryProps) {
  return (
    <button
      type="button"
      onClick={onChoose}
      aria-pressed={selected}
      disabled={disabled}
      title={title ?? label}
      style={{ height: C.entreeHauteurPx }}
      className={`flex w-full items-center gap-1.5 rounded px-2 text-left text-xs transition-colors ${
        disabled ? 'cursor-not-allowed opacity-50' : 'cursor-pointer'
      } ${selected ? 'bg-accent text-accent-foreground' : disabled ? '' : 'hover:bg-accent'}`}
    >
      <span className="min-w-0 flex-1 truncate">{label}</span>
      {selected && <CheckGlyph />}
    </button>
  )
}

/** Une liste du panneau : son titre, puis ses entrées. */
function List({ heading, children }: { heading: string; children: ReactNode }) {
  return (
    <div role="group" aria-label={heading} className="flex min-w-0 flex-col" style={{ width: C.listeLargeurPx }}>
      <div className="px-2 pb-1 pt-1 text-3xs font-semibold uppercase tracking-wider text-muted-foreground" aria-hidden="true">
        {heading}
      </div>
      {children}
    </div>
  )
}

/** Le titre d'un camp, dans une liste. */
function CampHeading({ label }: { label: string }) {
  return <div className="truncate px-2 pb-0.5 pt-1.5 text-3xs uppercase tracking-wider text-muted-foreground">{label}</div>
}

interface PanelProps {
  id: string
  groups: readonly ViewpointOptionGroup[]
  viewpoint: string | null
  followedKey: string | null
  locale: ReplayLocale
  onFollow: (key: string | null) => void
  onViewpoint: (value: string) => void
}

/** Le panneau : « Suivre » (avec « Aucun ») puis « Point de vue », côte à côte. */
function CameraPanel({ id, groups, viewpoint, followedKey, locale, onFollow, onViewpoint }: PanelProps) {
  const t = REPLAY_TEXT[locale]
  return (
    <div
      id={id}
      role="group"
      aria-label={t.roster.camera}
      // UNE COUCHE QU'ÉCHAP FERME AVANT LE MODE PLEIN ÉCRAN (cf. useReplayFullscreen).
      data-replay-escape-layer=""
      onKeyDown={keepKeysInPanel}
      className="absolute left-0 z-20 flex overflow-y-auto rounded-lg border border-border bg-popover p-1 shadow-xl"
      style={{ bottom: `calc(100% + ${C.panneauEcartPx}px)`, columnGap: C.listeEcartPx, maxHeight: C.panneauHauteurMaxPx }}
    >
      <List heading={t.playerPanel.action.follow}>
        <Entry label={t.roster.cameraNone} selected={followedKey === null} onChoose={() => onFollow(null)} />
        {groups.map((group) => (
          <div key={group.key}>
            <CampHeading label={group.label} />
            {group.options.map((o) => (
              <Entry key={o.rosterKey} label={o.label} selected={o.rosterKey === followedKey} onChoose={() => onFollow(o.rosterKey)} />
            ))}
          </div>
        ))}
      </List>
      <List heading={t.roster.cameraViewpointHeading}>
        {groups.map((group) => (
          <div key={group.key}>
            <CampHeading label={group.label} />
            {group.options.map((o) => (
              <Entry key={o.value} label={o.label} title={o.title} selected={o.value === viewpoint} disabled={o.disabled} onChoose={() => onViewpoint(o.value)} />
            ))}
          </div>
        ))}
      </List>
    </div>
  )
}

export function ReplayCameraMenu({ timeline, cameraFollow, locale }: ReplayCameraMenuProps) {
  const t = REPLAY_TEXT[locale]
  const groups = timeline.viewpointGroups
  const available = groups.length > 0
  const [open, setOpen] = useState(false)
  const triggerRef = useRef<HTMLButtonElement>(null)
  const panelId = useId()

  // Échap et clic dehors (`pointerdown`, comme le menu de vitesse) : Échap rend le focus au bouton, le clic
  // dehors le laisse où l'on a cliqué.
  const wrapRef = useDismissable(
    open && available,
    (reason) => {
      setOpen(false)
      if (reason === 'escape') triggerRef.current?.focus()
    },
    { outside: 'pointerdown' },
  )

  if (!available) return null

  const followedKey = cameraFollow.player?.xuid ?? null
  const followedName = followedLabelOf(groups, followedKey)
  const label = followedName ? t.roster.cameraFollowFmt(followedName) : t.roster.camera

  /** Un choix ferme le panneau et rend le focus au bouton : les raccourcis du lecteur restent actifs. */
  function choose(apply: () => void) {
    apply()
    setOpen(false)
    triggerRef.current?.focus()
  }

  return (
    <div ref={wrapRef} className="relative">
      <button
        ref={triggerRef}
        type="button"
        onClick={() => setOpen((v) => !v)}
        aria-expanded={open}
        aria-controls={open ? panelId : undefined}
        aria-label={label}
        title={label}
        style={COMPACT_BUTTON_SQUARE}
        className={`${COMPACT_BUTTON_CLASS} cursor-pointer ${followedName !== null ? 'bg-accent text-accent-foreground' : 'text-muted-foreground'}`}
      >
        <VideoCameraIcon />
      </button>
      {open && (
        <CameraPanel
          id={panelId}
          groups={groups}
          viewpoint={timeline.viewpoint}
          followedKey={followedKey}
          locale={locale}
          onFollow={(key) => choose(() => cameraFollow.onFollow(key))}
          onViewpoint={(value) => choose(() => timeline.onSelectViewpoint(value))}
        />
      )}
    </div>
  )
}
