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
 * Ce que le tiroir sait des ARMES AU SOL : une bascule, et si le film en porte.
 *
 *  suit la même règle que les socles — un film dont aucune arme ne tombe (ou un
 * artefact antérieur au schéma 27) ne montre pas la bascule. Le calque est SÉPARÉ de celui des
 * socles et ce n'est pas un doublon : un socle est un LIEU qui réapprovisionne, une arme au sol
 * est un OBJET qui ne revient pas. On peut vouloir l'un sans l'autre.
 */
export interface ReplayGroundWeaponControls {
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

/** LA BOMBE d'Assaut (schéma 30) : portée et posée, un seul calque, allumé par défaut. */
export interface ReplayBombCarrierControls {
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
  showTrail: boolean
  onToggleTrail: () => void
  zonesAvailable: boolean
  placements: ReplayPlacementControls
  weaponPads: ReplayWeaponPadControls
  groundWeapons: ReplayGroundWeaponControls
  flagCarries: ReplayFlagControls
  vipCrown: ReplayVipCrownControls
  skullCarrier: ReplaySkullCarrierControls
  bombCarrier: ReplayBombCarrierControls
}

export function LayersSection({
  locale, showAim, onToggleAim, showZones, onToggleZones,
  showTrail, onToggleTrail, zonesAvailable, placements, weaponPads, groundWeapons, flagCarries,
  vipCrown, skullCarrier, bombCarrier,
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
        {/* LE CALQUE DES NOMS N'A PLUS DE BASCULE (2026-09-02, demande utilisateur) : il est
            toujours allumé. Un nom sous un marqueur n'est pas un habillage dont on débat, c'est
            ce qui rend le rejeu lisible — et la bascule coûtait une ligne de tiroir à tout le
            monde pour un réglage que personne ne change. Le flag, sa clé de stockage et ses
            libellés sont partis avec elle. */}
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
        {/* LES ARMES AU SOL sont des OBJETS, pas des lieux : elles ne réapparaissent pas, elles
            gisent là où elles sont tombées. D'où une bascule à part de celle des emplacements —
            on peut vouloir voir les socles sans le fouillis des armes lâchées, et l'inverse. */}
        {groundWeapons.available && (
          <SettingsToggle
            label={t.layerGroundWeapons}
            pressed={groundWeapons.show}
            onToggle={groundWeapons.onToggle}
            hint={t.layerGroundWeaponsHint}
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
        {/* LA BOMBE est l'ENJEU d'Assaut : portée elle suit son porteur, lâchée elle reste au
            dernier point de son lâcheur jusqu'à la reprise ou l'explosion. Un film hors
            Assaut n'en publie aucune. */}
        {bombCarrier.available && (
          <SettingsToggle
            label={t.layerBombCarrier}
            pressed={bombCarrier.show}
            onToggle={bombCarrier.onToggle}
            hint={t.layerBombCarrierHint}
          />
        )}
      </div>
    </section>
  )
}
