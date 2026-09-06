/**
 * ReplaySettingsDrawer — LE TIROIR DE RÉGLAGES du rejeu (décision utilisateur du 16/08) :
 * lecture, calques, effets d'événement, son (+ filtre par catégorie), vitesse. Regroupe ce qui
 * vivait éparpillé dans la barre du canvas — AUCUN réglage n'est réinventé ici, chacun garde sa
 * règle et sa persistance (lecture/calques/effets/vitesse : useReplaySettings ; son et
 * catégories : useReplaySound).
 *
 * PANNEAU EN SURIMPRESSION, PAS UNE MODALE (retour de planche du 16/08 : « je vois plus un
 * panneau par dessus »). Il se pose SUR le rejeu — le canvas ne se retaille donc pas à
 * l'ouverture, et le rendu ne saute pas. Ce qui reste d'une modale : on en sort par Échap, par
 * le bouton, ou en cliquant dehors, et le focus entre au panneau à l'ouverture. Ce qui n'en est
 * pas : ni fond assombri, ni piège de focus, ni lecture suspendue — le rejeu tourne derrière.
 *
 * IL A QUITTÉ LE CADRE LE 2026-09-02 (demande utilisateur : « est-ce que ce tiroir sort du
 * cadre ? […] si la taille de l'écran le permet, c'est bien qu'il puisse s'afficher au-dessus
 * du killfeed ou des fiches »). Il vivait en `absolute inset-y-0 right-0` DANS la carte du
 * rejeu, que son `overflow-hidden` découpait au bord : ni débordement sur la colonne de droite,
 * ni largeur supérieure à ce que la carte lui laissait. Il se rend désormais en PORTAIL sur
 * `body`, en position fixe, centré sur le bouton qui l'ouvre (`useAnchoredPanel`) — aucun
 * `z-index` ne fait sortir un enfant d'un ancêtre en `overflow-hidden`, il faut quitter le
 * sous-arbre. C'est ce qui a rendu la largeur de 416 px possible, donc les calques sur deux
 * colonnes, qui butaient depuis le 2026-08-29 sur les 264 px utiles de l'ancien tiroir.
 *
 * DES INTERRUPTEURS, PLUS DES BOUTONS (retour utilisateur du 2026-08-29 : « pour les réglages
 * je préfère un toggle plutôt que des boutons comme aujourd'hui »). Tout ce qui est un OUI/NON
 * passe par `SettingsToggle`, devenu un vrai interrupteur `role="switch"` ; les CHOIX EXCLUSIFS
 * (lecture et portée de la carte de chaleur, couleur des points) gardent le bouton pressé sous
 * le nom `SettingsChoice` — la nuance et sa raison sont dans `ReplaySettingsToggle.tsx`.
 *
 * DÉCOUPÉ EN SECTIONS (Playback/Layers/Effects/Heatmap/Colors/Sound), chacune sa propre
 * fonction : un seul corps de composant pour toutes dépassait le seuil de lisibilité
 * (CLAUDE.md n°5, fonction ≤ 80 lignes) sans y gagner en clarté — des blocs indépendants s'y
 * prêtent mieux. Les deux plus lourdes vivent dans leur propre fichier (`ReplayHeatmapSection`
 * le 2026-08-18, `ReplaySettingsLayers` le 2026-08-29) : c'est ce qui tient ce fichier sous
 * les 500 lignes du dépôt.
 */
import { useEffect, useRef, type RefObject } from 'react'
import { createPortal } from 'react-dom'

import { InfoMark, SettingsChoice, SettingsToggle } from './ReplaySettingsToggle'
import { useAnchoredPanel } from './useAnchoredPanel'
import { REPLAY_TEXT, type ReplayLocale } from './i18n/i18n'
import { HeatmapSection, type ReplayHeatmapControls } from './ReplayHeatmapSection'
import { LayersSection } from './ReplaySettingsLayers'
import { SOUND_CATEGORIES } from './replaySound'
import { MARKER_COLORS_MODES, type MarkerColorsMode } from './useReplaySettings'
import type { ReplaySound } from './useReplaySound'

/**
 * LA LARGEUR DU PANNEAU, en px et pas en classe : le hook de placement en a besoin pour le
 * centrer sur son declencheur, et deux sources pour une meme largeur divergeraient au premier
 * ajustement. 416 px moins les marges internes laissent ~190 px par colonne de calques — c est
 * ce qui rend enfin possible la grille a deux colonnes (cf. l en-tete de LayerGroup).
 */
const PANEL_WIDTH = 416

/** Réexportés : les sections ont déménagé, la surface d'appel du tiroir n'a pas bougé. */
export type { ReplayHeatmapControls } from './ReplayHeatmapSection'
export type {
  ReplayBombCarrierControls,
  ReplayFlagControls,
  ReplayGroundWeaponControls,
  ReplayPlacementControls,
  ReplaySkullCarrierControls,
  ReplayVehicleControls,
  ReplayVipCrownControls,
  ReplayWeaponPadControls,
} from './ReplaySettingsLayers'

import type {
  ReplayBombCarrierControls,
  ReplayFlagControls,
  ReplayGroundWeaponControls,
  ReplayPlacementControls,
  ReplaySkullCarrierControls,
  ReplayVehicleControls,
  ReplayVipCrownControls,
  ReplayWeaponPadControls,
} from './ReplaySettingsLayers'

interface ReplaySettingsDrawerProps {
  locale: ReplayLocale
  onClose: () => void
  /**
   * LA LECTURE DÉMARRE-T-ELLE SEULE à l'ouverture du rejeu (demande utilisateur du 2026-08-29,
   * point 22) ? ÉTEINT par défaut. Réglage persisté comme les autres, MAIS il ne commande pas
   * le lecteur ouvert : il décide de son état de départ, lu une fois au montage (cf.
   * `useReplayPlayback`).
   */
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
  heatmap: ReplayHeatmapControls
  /** Éclairs de bouche (tous les tirs) et trait tueur -> victime : deux réglages distincts. */
  showShotFx: boolean
  onToggleShotFx: () => void
  showKillFx: boolean
  onToggleKillFx: () => void
  /** Le tiroir ne garde du son que le FILTRE PAR CATÉGORIE : l'interrupteur et le volume
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
 * Le tiroir ne garde du son que le FILTRE PAR CATÉGORIE (l'interrupteur et le volume sont à
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
  showTrail,
  onToggleTrail,
  zonesAvailable,
  placements,
  weaponPads,
  groundWeapons,
  flagCarries,
  vipCrown,
  skullCarrier,
  bombCarrier,
  vehicles,
  heatmap,
  showShotFx,
  onToggleShotFx,
  showKillFx,
  onToggleKillFx,
  sound,
  markerColors,
  onSetMarkerColors,
  triggerRef,
}: ReplaySettingsDrawerProps) {
  const t = REPLAY_TEXT[locale]
  // `tabIndex={-1}` rend le panneau focusable sans l'insérer dans l'ordre de tabulation.
  const panelRef = useRef<HTMLDivElement>(null)
  useDrawerDismiss(panelRef, triggerRef, onClose)
  const pos = useAnchoredPanel(triggerRef, PANEL_WIDTH)

  // IL SORT DU CADRE DEPUIS LE 2026-09-02 (demande utilisateur). Il vivait en
  // `absolute inset-y-0 right-0` DANS la carte du rejeu, que son `overflow-hidden` découpait au
  // bord : il ne pouvait ni déborder sur la colonne de droite, ni être plus large que ce que la
  // carte lui laissait — et c'est cette largeur qui interdisait les calques sur deux colonnes.
  //
  // LE PORTAIL EST LA CONDITION, PAS UN DÉTAIL D'IMPLÉMENTATION : aucune valeur de `z-index` ne
  // fait sortir un enfant d'un ancêtre en `overflow-hidden`. Il faut quitter le sous-arbre.
  //
  return createPortal(
    <div
      ref={panelRef}
      tabIndex={-1}
      role="region"
      aria-label={t.settingsButton}
      style={{ left: pos.left, bottom: pos.bottom, width: PANEL_WIDTH, maxHeight: pos.maxHeight }}
      className="fixed z-50 flex flex-col gap-3 overflow-y-auto rounded-lg border border-border bg-card px-3 py-3 text-sm shadow-xl outline-none"
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
        showTrail={showTrail}
        onToggleTrail={onToggleTrail}
        zonesAvailable={zonesAvailable}
        placements={placements}
        weaponPads={weaponPads}
        groundWeapons={groundWeapons}
        flagCarries={flagCarries}
        vipCrown={vipCrown}
        skullCarrier={skullCarrier}
        bombCarrier={bombCarrier}
        vehicles={vehicles}
      />
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
