/**
 * ReplaySettingsDrawer — LE TIROIR DE RÉGLAGES du rejeu (décision utilisateur du 16/08) :
 * calques, effets d'événement, son (+ filtre par catégorie), vitesse. Regroupe ce qui vivait
 * éparpillé dans la barre du canvas — AUCUN réglage n'est réinventé ici, chacun garde sa
 * règle et sa persistance (calques/effets/vitesse : useReplaySettings ; son et catégories :
 * useReplaySound).
 *
 * PANNEAU EN SURIMPRESSION, PAS UNE MODALE (retour de planche du 16/08 : « je vois plus un
 * panneau par dessus »). Il se pose SUR la carte, dans le cadre du rejeu — le canvas ne se
 * retaille donc plus à l'ouverture, et le rendu ne saute pas. Ce qui reste d'une modale : on
 * en sort par Échap, par le bouton, ou en cliquant dehors, et le focus entre au panneau à
 * l'ouverture. Ce qui n'en est pas : ni fond assombri, ni piège de focus, ni lecture
 * suspendue — le rejeu continue de tourner derrière.
 *
 * DÉCOUPÉ EN SECTIONS (Layers/Effects/Heatmap/Speed/Sound), chacune sa propre fonction : un
 * seul corps de composant pour toutes dépassait le seuil de lisibilité (CLAUDE.md n°5,
 * fonction ≤ 80 lignes) sans y gagner en clarté — des blocs indépendants s'y prêtent mieux.
 */
import { useEffect, useRef, type RefObject } from 'react'

import { Button } from '@/components/ui/button'

import type { HeatmapMode } from './heatmapLayer'
import { REPLAY_TEXT, type ReplayLocale } from './i18n'
import { ReplaySoundControls } from './ReplaySoundControls'
import { SOUND_CATEGORIES } from './replaySound'
import { HEATMAP_MODES, SPEED_MULTIPLIERS } from './useReplaySettings'
import type { ReplaySound } from './useReplaySound'

/** Ce que le tiroir sait de la carte de chaleur : son état, et ce qu'elle peut mesurer. */
export interface ReplayHeatmapControls {
  show: boolean
  onToggle: () => void
  mode: HeatmapMode
  onSetMode: (mode: HeatmapMode) => void
  /** Faux quand aucune mort du match n'a pu être localisée : la lecture « éliminations »
   *  ne commande alors rien et n'est pas proposée (même règle que le bouton Zones). */
  killsAvailable: boolean
}

interface ReplaySettingsDrawerProps {
  locale: ReplayLocale
  onClose: () => void
  showAim: boolean
  onToggleAim: () => void
  showZones: boolean
  onToggleZones: () => void
  showNames: boolean
  onToggleNames: () => void
  /** Le calque zones n'existe que si la carte a des zones nommées (même règle que le
   *  bouton d'origine : un interrupteur qui ne commande rien tromperait plus qu'il n'informe). */
  zonesAvailable: boolean
  /** Les POSES d'équipement (schéma 9) : calque + objets non identifiés. */
  placements: ReplayPlacementControls
  heatmap: ReplayHeatmapControls
  /** Éclairs de bouche (tous les tirs) et trait tueur -> victime : deux réglages distincts. */
  showShotFx: boolean
  onToggleShotFx: () => void
  showKillFx: boolean
  onToggleKillFx: () => void
  sound: ReplaySound
  speed: number
  onSetSpeed: (speed: number) => void
  /**
   * Le bouton qui a ouvert le panneau. Il est EXCLU du « clic dehors » — sans quoi le clic
   * qui referme fermerait puis rouvrirait aussitôt (le même clic atteint ensuite le bouton)
   * — et il RÉCUPÈRE le focus à la fermeture, côté appelant.
   */
  triggerRef?: RefObject<HTMLElement | null>
}

/**
 * InfoMark — la marque « (i) » d'une RÉSERVE DE MESURE, en SVG et jamais en caractère : le
 * « i cerclé » typographique dépend de la police installée et se rend en carré vide sur les
 * machines qui ne l'ont pas. Elle ne prend pas le focus et n'ouvre rien — tout ce qu'elle a
 * à dire tient dans son infobulle, et le lecteur d'écran lit la même phrase.
 */
function InfoMark({ text }: { text: string }) {
  return (
    <span
      role="img"
      aria-label={text}
      title={text}
      className="inline-flex shrink-0 items-center text-muted-foreground"
    >
      <svg
        width="12"
        height="12"
        viewBox="0 0 12 12"
        fill="none"
        stroke="currentColor"
        strokeWidth="1.1"
        strokeLinecap="round"
        aria-hidden="true"
      >
        <circle cx="6" cy="6" r="4.6" />
        <path d="M6 5.4v3" />
        <path d="M6 3.6v.2" />
      </svg>
    </span>
  )
}

/**
 * Une ligne de bascule : même gabarit pour les calques, les effets et les catégories de son —
 * huit usages dans ce seul fichier, un seul rendu plutôt que huit copies presque identiques
 * (CLAUDE.md règle 6, « à la 3e copie, centraliser »).
 *
 * `info` ajoute la marque de réserve À CÔTÉ du bouton, pas dedans : elle porte une phrase
 * longue (la couverture des tirs) que l'infobulle du bouton, déjà prise par son propre
 * `hint`, ne peut pas dire — et une marque cliquable serait un bouton de plus à comprendre.
 */
function SettingsToggle({
  label, pressed, onToggle, hint, info,
}: {
  label: string
  pressed: boolean
  onToggle: () => void
  hint?: string
  info?: string
}) {
  const button = (
    <Button
      type="button"
      variant={pressed ? 'default' : 'ghost'}
      size="sm"
      onClick={onToggle}
      className="h-7 justify-start px-2 text-xs"
      title={hint}
      aria-pressed={pressed}
    >
      {label}
    </Button>
  )
  if (!info) return button
  return (
    <div className="flex items-center gap-1">
      <span className="flex min-w-0 flex-1 flex-col">{button}</span>
      <InfoMark text={info} />
    </div>
  )
}

/**
 * Ce que le tiroir sait des POSES d'équipement : les deux bascules, et ce que le film porte.
 *
 * `available` et `unnamedAvailable` suivent la même règle que le bouton Zones — pas de
 * commande qui ne commande rien. Un film sans pose publiée (largeur de bloc non tranchée,
 * ou match sans équipement posé) ne montre pas le calque ; un film dont TOUTES les poses
 * sont nommées ne montre pas la bascule des objets non identifiés.
 */
export interface ReplayPlacementControls {
  available: boolean
  show: boolean
  onToggle: () => void
  unnamedAvailable: boolean
  showUnnamed: boolean
  onToggleUnnamed: () => void
}

interface LayersSectionProps {
  locale: ReplayLocale
  showAim: boolean
  onToggleAim: () => void
  showZones: boolean
  onToggleZones: () => void
  showNames: boolean
  onToggleNames: () => void
  zonesAvailable: boolean
  placements: ReplayPlacementControls
}

function LayersSection({
  locale, showAim, onToggleAim, showZones, onToggleZones, showNames, onToggleNames, zonesAvailable,
  placements,
}: LayersSectionProps) {
  const t = REPLAY_TEXT[locale]
  return (
    <section className="space-y-1">
      <h3 className="text-xs font-medium text-muted-foreground">{t.layers}</h3>
      <div className="flex flex-col gap-1">
        <SettingsToggle label={t.layerAim} pressed={showAim} onToggle={onToggleAim} hint={t.layerAimHint} />
        <SettingsToggle
          label={t.layerNames}
          pressed={showNames}
          onToggle={onToggleNames}
          hint={t.layerNamesHint}
        />
        {zonesAvailable && (
          <SettingsToggle
            label={t.layerZones}
            pressed={showZones}
            onToggle={onToggleZones}
            hint={t.layerZonesHint}
          />
        )}
        {/* Les POSES sont un calque, pas un effet : elles montrent un ÉTAT du terrain (un mur
            EST là de t0 à t1), là où un éclair de bouche montre un instant. La bascule des
            objets non identifiés n'apparaît qu'avec elles — elle ne commanderait rien sinon. */}
        {placements.available && (
          <>
            <SettingsToggle
              label={t.layerPlacements}
              pressed={placements.show}
              onToggle={placements.onToggle}
              hint={t.layerPlacementsHint}
            />
            {placements.show && placements.unnamedAvailable && (
              <SettingsToggle
                label={t.layerPlacementsUnnamed}
                pressed={placements.showUnnamed}
                onToggle={placements.onToggleUnnamed}
                hint={t.layerPlacementsUnnamedHint}
              />
            )}
          </>
        )}
      </div>
    </section>
  )
}

/**
 * Les EFFETS D'ÉVÉNEMENT ont leur propre section, séparée des calques : un calque montre un
 * ÉTAT du terrain (une visée, des zones, une chaleur), un effet montre un INSTANT (un tir,
 * une mort). Les mélanger ferait lire « éclairs de bouche » comme un fond de carte.
 *
 * LA RÉSERVE DE MESURE EST À L'ÉCRAN, pas dans un commentaire : le film n'enregistre un tir
 * que lorsqu'un dégât est appliqué, donc la couverture n'est pas garantie totale. C'est la
 * demande explicite du 16/08 — un (i) à côté de la bascule, sa phrase en infobulle.
 */
function EffectsSection({
  locale, showShotFx, onToggleShotFx, showKillFx, onToggleKillFx,
}: {
  locale: ReplayLocale
  showShotFx: boolean
  onToggleShotFx: () => void
  showKillFx: boolean
  onToggleKillFx: () => void
}) {
  const t = REPLAY_TEXT[locale]
  return (
    <section className="space-y-1">
      <h3 className="text-xs font-medium text-muted-foreground">{t.effects}</h3>
      <div className="flex flex-col gap-1">
        <SettingsToggle
          label={t.layerShotFx}
          pressed={showShotFx}
          onToggle={onToggleShotFx}
          hint={t.layerShotFxHint}
          info={t.layerShotFxCoverage}
        />
        <SettingsToggle
          label={t.layerKillFx}
          pressed={showKillFx}
          onToggle={onToggleKillFx}
          hint={t.layerKillFxHint}
        />
      </div>
    </section>
  )
}

/**
 * La CARTE DE CHALEUR a sa propre section : c'est un calque, mais qui porte un CHOIX de
 * lecture (ce qu'on mesure). Le noyer dans la liste des calques mettrait ce choix au même
 * rang qu'une bascule, alors qu'il change la grandeur affichée. Le choix ne s'affiche que
 * lorsque le calque est allumé — sinon il commanderait quelque chose d'invisible.
 */
function HeatmapSection({
  locale, heatmap,
}: {
  locale: ReplayLocale
  heatmap: ReplayHeatmapControls
}) {
  const t = REPLAY_TEXT[locale]
  const modes = heatmap.killsAvailable ? HEATMAP_MODES : HEATMAP_MODES.filter((m) => m !== 'kills')
  return (
    <section className="space-y-1">
      <h3 className="text-xs font-medium text-muted-foreground">{t.layerHeatmap}</h3>
      <div className="flex flex-col gap-1">
        <SettingsToggle
          label={t.layerHeatmap}
          pressed={heatmap.show}
          onToggle={heatmap.onToggle}
          hint={t.layerHeatmapHint}
        />
        {heatmap.show && modes.length > 1 && (
          <>
            <p className="pt-1 text-xs text-muted-foreground">{t.heatmapReading}</p>
            {modes.map((m) => (
              <SettingsToggle
                key={m}
                label={t.heatmapMode[m]}
                pressed={heatmap.mode === m}
                onToggle={() => heatmap.onSetMode(m)}
                hint={t.heatmapModeHint[m]}
              />
            ))}
          </>
        )}
      </div>
    </section>
  )
}

function SpeedSection({
  locale, speed, onSetSpeed,
}: {
  locale: ReplayLocale
  speed: number
  onSetSpeed: (speed: number) => void
}) {
  const t = REPLAY_TEXT[locale]
  return (
    <section className="space-y-1">
      <h3 className="text-xs font-medium text-muted-foreground">{t.speed}</h3>
      <div className="flex flex-wrap gap-1">
        {SPEED_MULTIPLIERS.map((m) => (
          <Button
            key={m}
            type="button"
            variant={speed === m ? 'default' : 'ghost'}
            size="sm"
            onClick={() => onSetSpeed(m)}
            className="h-7 px-2 text-xs"
            // La vitesse en cours est dite, pas seulement peinte : sans `aria-pressed` les
            // quatre boutons s'annoncent identiques a un lecteur d'ecran, alors que les
            // bascules voisines (SettingsToggle) le portent toutes.
            aria-pressed={speed === m}
          >
            {m < 1 ? `${m.toFixed(1)}×` : `${m.toFixed(0)}×`}
          </Button>
        ))}
      </div>
    </section>
  )
}

/** Le son n'apparaît qu'avec au moins un événement sonore dans ce match : même règle que
 *  partout ailleurs dans la barre — pas de commande qui ne commande rien. */
function SoundSection({ locale, sound }: { locale: ReplayLocale; sound: ReplaySound }) {
  const t = REPLAY_TEXT[locale]
  if (!sound.available) return null
  return (
    <section className="space-y-2">
      <h3 className="text-xs font-medium text-muted-foreground">{t.sound}</h3>
      <div className="flex flex-wrap items-center gap-1">
        <ReplaySoundControls sound={sound} locale={locale} />
      </div>
      <h3 className="text-xs font-medium text-muted-foreground">{t.soundCategoriesTitle}</h3>
      <div className="flex flex-col gap-1">
        {SOUND_CATEGORIES.map((category) => (
          <SettingsToggle
            key={category}
            label={t.soundCategory[category]}
            pressed={sound.categories[category]}
            onToggle={() => sound.toggleCategory(category)}
          />
        ))}
      </div>
    </section>
  )
}

/**
 * useDrawerDismiss — LES TROIS SORTIES du panneau, et l'entrée du focus.
 *
 *  - ÉCHAP, comme tout panneau du dépôt ;
 *  - CLIC DEHORS, écouté au document plutôt que par un voile transparent posé sur la page :
 *    un voile AVALERAIT le premier clic (fermer une commande demanderait deux clics) et
 *    couvrirait la barre de lecture. Ici aucun clic n'est intercepté, seule la fermeture
 *    s'ajoute. `pointerdown` et non `click` : le panneau part au geste, pas au relâché ;
 *  - le BOUTON DE FERMETURE, câblé par le composant.
 *
 * ET LE FOCUS ENTRE À L'OUVERTURE : le panneau se pose SUR la carte ; sans cela un lecteur
 * au clavier resterait derrière lui, à parcourir des commandes qu'il ne voit plus.
 */
function useDrawerDismiss(
  panelRef: RefObject<HTMLDivElement | null>,
  triggerRef: RefObject<HTMLElement | null> | undefined,
  onClose: () => void,
): void {
  useEffect(() => {
    panelRef.current?.focus({ preventScroll: true })
    function onKeyDown(event: KeyboardEvent) {
      if (event.key === 'Escape') onClose()
    }
    function onPointerDown(event: PointerEvent) {
      const target = event.target as Node | null
      if (!target) return
      if (panelRef.current?.contains(target)) return
      if (triggerRef?.current?.contains(target)) return
      onClose()
    }
    window.addEventListener('keydown', onKeyDown)
    document.addEventListener('pointerdown', onPointerDown)
    return () => {
      window.removeEventListener('keydown', onKeyDown)
      document.removeEventListener('pointerdown', onPointerDown)
    }
  }, [panelRef, triggerRef, onClose])
}

export function ReplaySettingsDrawer({
  locale,
  onClose,
  showAim,
  onToggleAim,
  showZones,
  onToggleZones,
  showNames,
  onToggleNames,
  zonesAvailable,
  placements,
  heatmap,
  showShotFx,
  onToggleShotFx,
  showKillFx,
  onToggleKillFx,
  sound,
  speed,
  onSetSpeed,
  triggerRef,
}: ReplaySettingsDrawerProps) {
  const t = REPLAY_TEXT[locale]
  // `tabIndex={-1}` rend le panneau focusable sans l'insérer dans l'ordre de tabulation.
  const panelRef = useRef<HTMLDivElement>(null)
  useDrawerDismiss(panelRef, triggerRef, onClose)

  return (
    <div
      ref={panelRef}
      tabIndex={-1}
      role="region"
      aria-label={t.settingsButton}
      className="absolute inset-y-0 right-0 z-20 flex w-64 flex-col gap-4 overflow-y-auto border-l border-border bg-card px-3 py-3 text-sm shadow-xl outline-none"
    >
      <div className="flex items-center justify-between">
        <h2 className="text-sm font-semibold">{t.settingsButton}</h2>
        <button
          type="button"
          onClick={onClose}
          className="flex h-6 w-6 items-center justify-center rounded text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
          aria-label={t.settingsClose}
        >
          ×
        </button>
      </div>

      <LayersSection
        locale={locale}
        showAim={showAim}
        onToggleAim={onToggleAim}
        showZones={showZones}
        onToggleZones={onToggleZones}
        showNames={showNames}
        onToggleNames={onToggleNames}
        zonesAvailable={zonesAvailable}
        placements={placements}
      />
      <EffectsSection
        locale={locale}
        showShotFx={showShotFx}
        onToggleShotFx={onToggleShotFx}
        showKillFx={showKillFx}
        onToggleKillFx={onToggleKillFx}
      />
      <HeatmapSection locale={locale} heatmap={heatmap} />
      <SpeedSection locale={locale} speed={speed} onSetSpeed={onSetSpeed} />
      <SoundSection locale={locale} sound={sound} />
    </div>
  )
}
