/**
 * ReplaySettingsLayers — la section CALQUES du tiroir de réglages, et ce que chaque calque
 * demande au film pour seulement s'afficher.
 *
 * EXTRAITE DE `ReplaySettingsDrawer.tsx` LE 2026-08-29, deuxième extraction du tiroir après
 * la carte de chaleur (2026-08-18) et pour la même raison : le tiroir gagnait une section —
 * la LECTURE (lecture automatique) — et repassait au-dessus du seuil de 500 lignes du dépôt
 * (CLAUDE.md n°5). La règle du dépôt est d'extraire, pas de relever le plafond.
 *
 * LA DÉCOUPE TOMBE SUR UNE FRONTIÈRE NETTE : les calques sont la seule section dont l'affichage
 * dépend de CE QUE LE FILM PORTE, et les cinq interfaces `available` qui portent cette question
 * partent avec elle. Le tiroir les réexporte : sa surface d'appel ne change pas d'un octet.
 *
 * PAS DE COMMANDE QUI NE COMMANDE RIEN — c'est la règle du dépôt, née du bouton Zones : un film
 * sans zone nommée, sans socle publié, sans drapeau, sans couronne, sans crâne n'affiche pas la
 * bascule correspondante. Un interrupteur qui ne fait rien trompe plus qu'il n'informe.
 */
import { SettingsToggle } from './ReplaySettingsToggle'

import { REPLAY_TEXT, type ReplayLocale } from './i18n'

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

/** La COURONNE VIP (schéma 22) : un seul calque, allumé par défaut, comme les drapeaux. */
export interface ReplayVipCrownControls {
  available: boolean
  show: boolean
  onToggle: () => void
}

/** Le PORTEUR DU CRÂNE d'Oddball (schéma 23) : un seul calque, allumé par défaut. */
export interface ReplaySkullCarrierControls {
  available: boolean
  show: boolean
  onToggle: () => void
}

export interface LayersSectionProps {
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
  vipCrown: ReplayVipCrownControls
  skullCarrier: ReplaySkullCarrierControls
}

export function LayersSection({
  locale, showAim, onToggleAim, showZones, onToggleZones, showNames, onToggleNames,
  showTrail, onToggleTrail, zonesAvailable, placements, weaponPads, flagCarries, vipCrown,
  skullCarrier,
}: LayersSectionProps) {
  const t = REPLAY_TEXT[locale]
  return (
    <section className="space-y-1">
      <h3 className="text-xs font-medium text-muted-foreground">{t.layers}</h3>
      {/* UNE LIGNE PAR CALQUE depuis le 2026-08-29 (« je préfère un toggle plutôt que des
          boutons »). Ces bascules vivaient en GRILLE À DEUX COLONNES depuis le 2026-08-24
          (« un élément par ligne c'est inefficace ») : un interrupteur, lui, se lit sur son
          rail — libellé à gauche, état à droite — et deux rails côte à côte dans 130 px
          tronqueraient « Objets lâchés au sol » pour gagner une hauteur que le tiroir, qui
          défile déjà, n'avait pas besoin de gagner. */}
      <div className="flex flex-col gap-0.5">
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
        {/* LA COURONNE VIP est l'ENJEU du mode, comme les drapeaux : elle suit le porteur, sa
            présence EST la lecture du match. Un film hors VIP n'en publie aucune. */}
        {vipCrown.available && (
          <SettingsToggle
            label={t.layerVipCrown}
            pressed={vipCrown.show}
            onToggle={vipCrown.onToggle}
            hint={t.layerVipCrownHint}
          />
        )}
        {/* LE PORTEUR DU CRÂNE est l'ENJEU d'Oddball : il suit le porteur, sa présence EST la
            lecture du match. Un film hors Oddball n'en publie aucun. */}
        {skullCarrier.available && (
          <SettingsToggle
            label={t.layerSkullCarrier}
            pressed={skullCarrier.show}
            onToggle={skullCarrier.onToggle}
            hint={t.layerSkullCarrierHint}
          />
        )}
      </div>
    </section>
  )
}
