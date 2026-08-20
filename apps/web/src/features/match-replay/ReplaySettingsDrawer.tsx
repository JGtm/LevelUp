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

import { SettingsToggle } from './ReplaySettingsToggle'

import { REPLAY_TEXT, type ReplayLocale } from './i18n'
import { HeatmapSection, type ReplayHeatmapControls } from './ReplayHeatmapSection'
import { ReplaySoundControls } from './ReplaySoundControls'
import { SOUND_CATEGORIES } from './replaySound'
import { SPEED_MULTIPLIERS } from './useReplaySettings'
import type { ReplaySound } from './useReplaySound'

/** Réexporté : la section a déménagé (ReplayHeatmapSection), sa surface d'appel non. */
export type { ReplayHeatmapControls } from './ReplayHeatmapSection'

interface ReplaySettingsDrawerProps {
  locale: ReplayLocale
  onClose: () => void
  showAim: boolean
  onToggleAim: () => void
  showZones: boolean
  onToggleZones: () => void
  showNames: boolean
  onToggleNames: () => void
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
  /** Les DRAPEAUX de capture (schéma 15) : un seul calque, allumé par défaut. */
  flagCarries: ReplayFlagControls
  heatmap: ReplayHeatmapControls
  /** Éclairs de bouche (tous les tirs) et trait tueur -> victime : deux réglages distincts. */
  showShotFx: boolean
  onToggleShotFx: () => void
  showKillFx: boolean
  onToggleKillFx: () => void
  /** Fiches joueur COMPACTES (B2/R2-7) : une option, la validée reste le défaut. */
  compactCards: boolean
  onToggleCompactCards: () => void
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
  /**
   * Les objets de PUISSANCE lâchés à la mort. `droppedAvailable` ne pose plus qu'UNE
   * condition : le film en porte au moins un. La garde de mode qui l'annulait en Fiesta a été
   * retirée le 2026-08-20 (elle masquait 26 lâchers réels sur le témoin Fiesta) — la commande
   * s'affiche donc dans tous les modes dès qu'elle a de quoi commander.
   */
  droppedAvailable: boolean
  showDropped: boolean
  onToggleDropped: () => void
}

/**
 * Ce que le tiroir sait des EMPLACEMENTS D'ARME : une bascule, et si le film en porte.
 * `available` suit la même règle — un film sans socle publié (Super Fiesta sur variante
 * Forge : zéro socle mesuré) ne montre pas la bascule.
 */
export interface ReplayWeaponPadControls {
  available: boolean
  show: boolean
  onToggle: () => void
}

/**
 * Ce que le tiroir sait des DRAPEAUX de capture : une bascule, et si le film en porte.
 * `available` suit la même règle que les zones et les socles — un film qui n'est pas reconnu
 * comme de la capture de drapeau ne publie aucun drapeau, et ne montre donc pas la bascule.
 */
export interface ReplayFlagControls {
  available: boolean
  show: boolean
  onToggle: () => void
}

interface LayersSectionProps {
  locale: ReplayLocale
  showAim: boolean
  onToggleAim: () => void
  showZones: boolean
  onToggleZones: () => void
  showNames: boolean
  onToggleNames: () => void
  showTrail: boolean
  onToggleTrail: () => void
  zonesAvailable: boolean
  placements: ReplayPlacementControls
  weaponPads: ReplayWeaponPadControls
  flagCarries: ReplayFlagControls
}

function LayersSection({
  locale, showAim, onToggleAim, showZones, onToggleZones, showNames, onToggleNames,
  showTrail, onToggleTrail, zonesAvailable, placements, weaponPads, flagCarries,
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
        <SettingsToggle
          label={t.layerTrail}
          pressed={showTrail}
          onToggle={onToggleTrail}
          hint={t.layerTrailHint}
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
            {placements.show && placements.droppedAvailable && (
              <SettingsToggle
                label={t.layerPlacementsDropped}
                pressed={placements.showDropped}
                onToggle={placements.onToggleDropped}
                hint={t.layerPlacementsDroppedHint}
              />
            )}
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
        {/* Les EMPLACEMENTS D'ARME sont un calque du terrain eux aussi, mais leur donnée est
            une récurrence spatiale mesurée, pas un geste de joueur : d'où une bascule à part. */}
        {weaponPads.available && (
          <SettingsToggle
            label={t.layerWeaponPads}
            pressed={weaponPads.show}
            onToggle={weaponPads.onToggle}
            hint={t.layerWeaponPadsHint}
          />
        )}
        {/* Les DRAPEAUX sont l'ENJEU du mode, pas un meuble : ils bougent, ils changent de
            main, et leur position EST la lecture du match. Ils restent dans les calques —
            un drapeau au sol est un état du terrain, pas un instant. */}
        {flagCarries.available && (
          <SettingsToggle
            label={t.layerFlagCarries}
            pressed={flagCarries.show}
            onToggle={flagCarries.onToggle}
            hint={t.layerFlagCarriesHint}
          />
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
 * Les FICHES ont leur propre section, minuscule mais à part : elles ne vivent pas sur la
 * carte. Les ranger parmi les calques ferait croire qu'on allume ou éteint un dessin du
 * canvas, alors que le réglage change la COLONNE d'à côté.
 */
function CardsSection({
  locale, compactCards, onToggleCompactCards,
}: {
  locale: ReplayLocale
  compactCards: boolean
  onToggleCompactCards: () => void
}) {
  const t = REPLAY_TEXT[locale]
  return (
    <section className="space-y-1">
      <h3 className="text-xs font-medium text-muted-foreground">{t.cards}</h3>
      <SettingsToggle
        label={t.cardsCompact}
        pressed={compactCards}
        onToggle={onToggleCompactCards}
        hint={t.cardsCompactHint}
      />
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
  showTrail,
  onToggleTrail,
  zonesAvailable,
  placements,
  weaponPads,
  flagCarries,
  heatmap,
  showShotFx,
  onToggleShotFx,
  showKillFx,
  onToggleKillFx,
  compactCards,
  onToggleCompactCards,
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
        showTrail={showTrail}
        onToggleTrail={onToggleTrail}
        zonesAvailable={zonesAvailable}
        placements={placements}
        weaponPads={weaponPads}
        flagCarries={flagCarries}
      />
      <EffectsSection
        locale={locale}
        showShotFx={showShotFx}
        onToggleShotFx={onToggleShotFx}
        showKillFx={showKillFx}
        onToggleKillFx={onToggleKillFx}
      />
      <HeatmapSection locale={locale} heatmap={heatmap} />
      <CardsSection
        locale={locale}
        compactCards={compactCards}
        onToggleCompactCards={onToggleCompactCards}
      />
      <SpeedSection locale={locale} speed={speed} onSetSpeed={onSetSpeed} />
      <SoundSection locale={locale} sound={sound} />
    </div>
  )
}
