/**
 * ReplayViewPanel — LE PANNEAU DE LA VUE du rejeu : les préréglages de calques d'abord, puis les
 * calques un par un, puis les effets d'événement, la carte de chaleur, la couleur des points et
 * le filtre de son. Il REMPLACE le tiroir de réglages (plan « maquettes du rejeu », lot 3,
 * décision du 2026-11-09) : AUCUN réglage n'est réinventé ni perdu, chacun garde sa règle et sa
 * persistance (calques, effets, chaleur, couleur : useReplaySettings ; son et catégories :
 * useReplaySound ; vue : useReplayViewMenu).
 *
 * PANNEAU EN SURIMPRESSION, PAS UNE MODALE. Il se pose SUR le rejeu — le canvas ne se retaille
 * donc pas à l'ouverture, et le rendu ne saute pas. Ce qui reste d'une modale : on en sort par
 * Échap, par le bouton, ou en cliquant dehors, et le focus entre au panneau à l'ouverture. Ce
 * qui n'en est pas : ni fond assombri, ni piège de focus, ni lecture suspendue — le rejeu
 * tourne derrière.
 *
 * IL SORT DU CADRE : la carte porte `overflow-hidden`, que ne franchit aucun `z-index`. Il se
 * rend en PORTAIL sur `body`, en position fixe, sous le bouton « Vue » qui l'ouvre
 * (`useAnchoredPanel`), et défile EN INTERNE quand l'écran est bas : il ne couvre de la carte que
 * ce qu'il lui faut.
 *
 * DES INTERRUPTEURS, PLUS DES BOUTONS pour tout OUI/NON (`SettingsToggle`, `role="switch"`) ; les
 * CHOIX EXCLUSIFS (préréglages, lecture et portée de la chaleur, couleur des points) gardent le
 * bouton pressé ou le segmenté — la nuance et sa raison sont dans `ReplaySettingsToggle.tsx`.
 *
 * DÉCOUPÉ EN SECTIONS, chacune sa propre fonction (CLAUDE.md n°5, fonction ≤ 80 lignes). Les
 * plus lourdes vivent dans leur fichier : `ReplayViewPresets`, `ReplayHeatmapSection`,
 * `ReplaySettingsLayers`.
 */
import { useEffect, useRef, type RefObject } from 'react'
import { createPortal } from 'react-dom'

import { InfoMark, SettingsChoice, SettingsToggle } from './ReplaySettingsToggle'
import { useAnchoredPanel } from './useAnchoredPanel'
import { REPLAY_TEXT, type ReplayLocale } from '../i18n/i18n'
import { HeatmapSection, type ReplayHeatmapControls } from '../ui/ReplayHeatmapSection'
import { LayersSection } from './ReplaySettingsLayers'
import { OrientationSection, type ReplayOrientationControls } from './ReplayOrientationSection'
import { ViewPresetsSection } from './ReplayViewPresets'
import type { ViewId, ViewPresetId } from './viewPresetLogic'
import { SOUND_CATEGORIES } from '../sound/replaySound'
import { MARKER_COLORS_MODES, type MarkerColorsMode } from './useReplaySettings'
import type { ReplaySound } from '../sound/useReplaySound'

/**
 * LA LARGEUR DU PANNEAU, en px et pas en classe : le hook de placement en a besoin pour aligner
 * son bord droit sur celui du bouton, et deux sources pour une meme largeur divergeraient au
 * premier ajustement. 416 px moins les marges internes laissent ~190 px par colonne de calques — c est
 * ce qui rend enfin possible la grille a deux colonnes (cf. l en-tete de LayerGroup).
 */
const PANEL_WIDTH = 416

/** Réexportés : les sections ont déménagé, la surface d'appel du panneau n'a pas bougé. */
export type { ReplayHeatmapControls } from '../ui/ReplayHeatmapSection'
export type { ReplayOrientationControls } from './ReplayOrientationSection'
export type {
  ReplayBombCarrierControls,
  ReplayFlagControls,
  ReplayGroundWeaponControls,
  ReplayModeObjectivesControls,
  ReplayPlacementControls,
  ReplaySkullCarrierControls,
  ReplayVehicleControls,
  ReplayVipCrownControls,
  ReplayWallPlanControls,
  ReplayWeaponPadControls,
} from './ReplaySettingsLayers'

import type {
  ReplayBombCarrierControls,
  ReplayFlagControls,
  ReplayGroundWeaponControls,
  ReplayModeObjectivesControls,
  ReplayPlacementControls,
  ReplaySkullCarrierControls,
  ReplayVehicleControls,
  ReplayVipCrownControls,
  ReplayWallPlanControls,
  ReplayWeaponPadControls,
} from './ReplaySettingsLayers'

interface ReplayViewPanelProps {
  locale: ReplayLocale
  onClose: () => void
  /** La vue active (déduite de l'état des calques) et le choix d'un préréglage. */
  view: { active: ViewId; onSelect: (id: ViewPresetId) => void }
  showAim: boolean
  onToggleAim: () => void
  showZones: boolean
  onToggleZones: () => void
  /** La TRAÎNÉE des marqueurs (retour du 2026-08-18) : allumée par défaut, éteignable. */
  showTrail: boolean
  onToggleTrail: () => void
  /** Le calque zones n'existe que si la carte a des zones nommées (même règle que le
   *  bouton d'origine : un interrupteur qui ne commande rien tromperait plus qu'il n'informe). */
  zonesAvailable: boolean
  /** Les POSES d'équipement (schéma 9) : calque + objets non identifiés. */
  placements: ReplayPlacementControls
  /** Les EMPLACEMENTS D'ARME (schéma 11) : un seul calque, allumé par défaut. */
  weaponPads: ReplayWeaponPadControls
  /** Les ARMES AU SOL (schéma 27) : un seul calque, allumé par défaut. */
  groundWeapons: ReplayGroundWeaponControls
  /** Les OBJECTIFS DU MODE (2026-09-18) : trois calques, une bascule, allumée par défaut. */
  modeObjectives: ReplayModeObjectivesControls
  /** Les DRAPEAUX de capture (schéma 15) : un seul calque, allumé par défaut. */
  flagCarries: ReplayFlagControls
  /** La COURONNE VIP (schéma 22) : un seul calque, allumé par défaut. */
  vipCrown: ReplayVipCrownControls
  /** Le PORTEUR DU CRÂNE d'Oddball (schéma 23) : un seul calque, allumé par défaut. */
  skullCarrier: ReplaySkullCarrierControls
  /** LA BOMBE d'Assaut (schéma 30) : portée et posée, un seul calque, allumé par défaut. */
  bombCarrier: ReplayBombCarrierControls
  /** Les VÉHICULES (schéma 39) : un seul calque, allumé par défaut. */
  vehicles: ReplayVehicleControls
  /** Le PLAN DES MURS du fond (clé `replay-show-wall-plan`) : n'existe qu'avec un fond qui en porte un. */
  wallPlan: ReplayWallPlanControls
  /** L'ORIENTATION de la carte (quarts de tour), retenue par carte : n'existe qu'avec une carte connue. */
  orientation: ReplayOrientationControls
  heatmap: ReplayHeatmapControls
  /** Éclairs de bouche (tous les tirs) et trait tueur -> victime : deux réglages distincts. */
  showShotFx: boolean
  onToggleShotFx: () => void
  showKillFx: boolean
  onToggleKillFx: () => void
  /** Le panneau ne garde du son que le FILTRE PAR CATÉGORIE : l'interrupteur et le volume
   *  vivent à la barre de lecture depuis le 2026-08-24 (ReplayTransport). */
  sound: ReplaySound
  /** Couleur des points des joueurs : par équipe (défaut) ou distincte par joueur. */
  markerColors: MarkerColorsMode
  onSetMarkerColors: (mode: MarkerColorsMode) => void
  /**
   * Le bouton qui a ouvert le panneau. Il est EXCLU du « clic dehors » — sans quoi le clic
   * qui referme fermerait puis rouvrirait aussitôt (le même clic atteint ensuite le bouton)
   * — et il RÉCUPÈRE le focus à la fermeture, côté appelant.
   */
  triggerRef?: RefObject<HTMLElement | null>
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
      {/* La réserve de mesure (couverture des tirs) se pose sur le TITRE de la section
          (demande utilisateur du 2026-08-24), plus à côté d'une bascule. */}
      <h3 className="flex items-center gap-1 text-xs font-medium text-muted-foreground">
        {t.effects}
        <InfoMark text={t.layerShotFxCoverage} />
      </h3>
      {/* DEUX COLONNES, UNE RANGEE (demande utilisateur du 2026-09-02 : « les effets peuvent
          etre sur une rangee »). Ils sont exactement deux : les empiler coutait une ligne
          pour rien dans un panneau qui en fait desormais 416 de large. */}
      <div className="grid grid-cols-2 gap-x-3 gap-y-0.5">
        <SettingsToggle
          label={t.layerShotFx}
          pressed={showShotFx}
          onToggle={onToggleShotFx}
          hint={t.layerShotFxHint}
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
 * LA COULEUR DES POINTS (proposition utilisateur du 2026-08-24) : par équipe — le défaut,
 * la couleur dit le camp (D1) — ou distincte par joueur, pour suivre quelqu'un dans la
 * mêlée. Deux lectures exclusives, même grammaire que les choix de la carte de chaleur —
 * d'où `SettingsChoice` et non un interrupteur : exactement une des deux est vraie.
 */
function ColorsSection({
  locale, markerColors, onSetMarkerColors,
}: {
  locale: ReplayLocale
  markerColors: MarkerColorsMode
  onSetMarkerColors: (mode: MarkerColorsMode) => void
}) {
  const t = REPLAY_TEXT[locale]
  return (
    <section className="space-y-1">
      <h3 className="text-xs font-medium text-muted-foreground">{t.markerColorsTitle}</h3>
      <div className="grid grid-cols-2 gap-1">
        {MARKER_COLORS_MODES.map((mode) => (
          <SettingsChoice
            key={mode}
            label={t.markerColorsMode[mode]}
            pressed={markerColors === mode}
            onToggle={() => onSetMarkerColors(mode)}
            hint={t.markerColorsHint[mode]}
          />
        ))}
      </div>
    </section>
  )
}

/**
 * Le panneau ne garde du son que le FILTRE PAR CATÉGORIE (l'interrupteur et le volume sont à
 * la barre de lecture, 2026-08-24). Même règle que partout : pas de commande quand il n'y a
 * rien à commander — un match sans un seul son ne montre pas de filtre.
 *
 * CINQ OUI/NON INDÉPENDANTS, pas un choix : on peut couper les grenades ET les objectifs, ou
 * tout couper. Ce sont donc des interrupteurs, un par ligne comme les calques.
 */
function SoundSection({ locale, sound }: { locale: ReplayLocale; sound: ReplaySound }) {
  const t = REPLAY_TEXT[locale]
  if (!sound.available) return null
  return (
    <section className="space-y-1">
      <h3 className="text-xs font-medium text-muted-foreground">{t.soundCategoriesTitle}</h3>
      {/* DEUX COLONNES, DEUX RANGEES (meme demande) : quatre categories empilees faisaient
          quatre lignes la ou deux suffisent. */}
      <div className="grid grid-cols-2 gap-x-3 gap-y-0.5">
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
 * usePanelDismiss — LES TROIS SORTIES du panneau, et l'entrée du focus.
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
function usePanelDismiss(
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

export function ReplayViewPanel({
  locale,
  onClose,
  view,
  showAim,
  onToggleAim,
  showZones,
  onToggleZones,
  showTrail,
  onToggleTrail,
  zonesAvailable,
  placements,
  weaponPads,
  groundWeapons,
  modeObjectives,
  flagCarries,
  vipCrown,
  skullCarrier,
  bombCarrier,
  vehicles,
  wallPlan,
  orientation,
  heatmap,
  showShotFx,
  onToggleShotFx,
  showKillFx,
  onToggleKillFx,
  sound,
  markerColors,
  onSetMarkerColors,
  triggerRef,
}: ReplayViewPanelProps) {
  const t = REPLAY_TEXT[locale]
  // `tabIndex={-1}` rend le panneau focusable sans l'insérer dans l'ordre de tabulation.
  const panelRef = useRef<HTMLDivElement>(null)
  usePanelDismiss(panelRef, triggerRef, onClose)
  const pos = useAnchoredPanel(triggerRef, PANEL_WIDTH)

  // LE PORTAIL EST LA CONDITION, PAS UN DÉTAIL D'IMPLÉMENTATION : la carte porte
  // `overflow-hidden` et aucune valeur de `z-index` ne fait sortir un enfant d'un ancêtre qui
  // découpe. Il faut quitter le sous-arbre.
  return createPortal(
    <div
      ref={panelRef}
      tabIndex={-1}
      role="region"
      // UNE COUCHE QU'ÉCHAP FERME AVANT LE MODE PLEIN ÉCRAN (cf. useReplayFullscreen).
      data-replay-escape-layer=""
      aria-label={t.viewButton}
      style={{ left: pos.left, top: pos.top, width: PANEL_WIDTH, maxHeight: pos.maxHeight }}
      className="fixed z-50 flex max-w-[calc(100vw-1rem)] flex-col gap-3 overflow-y-auto rounded-lg border border-border bg-card px-3 py-3 text-sm shadow-xl outline-none"
    >
      <div className="flex items-center justify-between">
        <h2 className="text-sm font-semibold">{t.viewButton}</h2>
        <button
          type="button"
          onClick={onClose}
          className="flex h-6 w-6 items-center justify-center rounded text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
          aria-label={t.viewClose}
        >
          ×
        </button>
      </div>

      <ViewPresetsSection locale={locale} active={view.active} onSelect={view.onSelect} />
      <LayersSection
        locale={locale}
        showAim={showAim}
        onToggleAim={onToggleAim}
        showZones={showZones}
        onToggleZones={onToggleZones}
        showTrail={showTrail}
        onToggleTrail={onToggleTrail}
        zonesAvailable={zonesAvailable}
        placements={placements}
        weaponPads={weaponPads}
        groundWeapons={groundWeapons}
        modeObjectives={modeObjectives}
        flagCarries={flagCarries}
        vipCrown={vipCrown}
        skullCarrier={skullCarrier}
        bombCarrier={bombCarrier}
        vehicles={vehicles}
        wallPlan={wallPlan}
      />
      <OrientationSection locale={locale} orientation={orientation} />
      <EffectsSection
        locale={locale}
        showShotFx={showShotFx}
        onToggleShotFx={onToggleShotFx}
        showKillFx={showKillFx}
        onToggleKillFx={onToggleKillFx}
      />
      <HeatmapSection locale={locale} heatmap={heatmap} />
      <ColorsSection
        locale={locale}
        markerColors={markerColors}
        onSetMarkerColors={onSetMarkerColors}
      />
      <SoundSection locale={locale} sound={sound} />
    </div>,
    document.body,
  )
}
