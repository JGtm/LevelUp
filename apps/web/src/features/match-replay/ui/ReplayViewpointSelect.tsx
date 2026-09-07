/**
 * ReplayViewpointSelect — PAR LES YEUX DE QUI ON REGARDE, choisi sur la frise elle-même.
 *
 * # POURQUOI IL VIT DANS LA COLONNE DES LIBELLÉS (2026-09-07, lot L3)
 *
 * La première rangée de la frise disait « Toi ». C'était juste tant que la page n'avait qu'un
 * point de vue possible ; depuis le lot L2b elle en a autant qu'il y a de joueurs, et le mot
 * devenait faux dès qu'on en choisissait un autre. Le libellé est donc devenu la COMMANDE :
 * l'endroit où l'on lit de qui parle la piste est celui où l'on change de joueur, sans un
 * élément d'interface de plus dans une barre de lecture déjà pleine.
 *
 * # UN `<select>` NATIF, ET C'EST UN CHOIX
 *
 * Un menu maison aurait fallu clavier, rôles ARIA, gestion du focus et du survol. Le champ
 * natif les a tous, et il a mieux : sa LISTE OUVERTE n'est pas contrainte par la largeur du
 * champ fermé. La colonne fait 100 px — assez pour reconnaître un gamertag tronqué, pas pour
 * le lire — pendant que la liste déroulée montre les noms entiers. C'est exactement ce que la
 * décision 6 du plan demande, et aucun menu maison ne l'obtient sans repositionnement manuel.
 *
 * # LE FOCUS NE RESTE PAS SUR LE CHAMP APRÈS UN CHOIX
 *
 * `useReplayShortcuts` coupe TOUS les raccourcis quand l'élément actif est un `SELECT` (et c'est
 * la bonne règle : les flèches d'une liste déroulée lui appartiennent). Sans précaution, choisir
 * un joueur laisserait le focus ici, et la barre d'espace qui suit — le geste réflexe pour
 * mettre en pause — rouvrirait la liste au lieu d'arrêter la lecture. Le champ se retire donc
 * le focus une fois la valeur posée. Il n'est PAS exempté par `data-replay-timeline`, qui vise
 * nommément le curseur : tant que la liste est ouverte, ses flèches doivent rester à elle.
 *
 * # CE QU'IL NE FAIT PAS
 *
 * Il ne touche ni au curseur, ni à la lecture, ni à la vitesse (décision 1 du plan) : changer de
 * joueur change ce qu'on VOIT, jamais où l'on en est. Et il ne construit pas sa liste — elle
 * vient de `model/viewpointOptions.ts`, pur et testé à part, où vivent les deux règles qui
 * comptent (la valeur d'un bot, et l'option inerte d'un joueur sans ligne de tableau de score).
 */
import type { ViewpointOptionGroup } from '../model/viewpointOptions'

interface ReplayViewpointSelectProps {
  /**
   * Le point de vue COURANT — un xuid de la base. `null`, ou une valeur qu'aucune option ne
   * porte, laisse le champ vide plutôt que d'afficher le premier nom venu : mieux vaut ne rien
   * annoncer qu'annoncer quelqu'un d'autre que ce que la frise montre.
   */
  value: string | null
  /** Les sections du menu, un camp par section (cf. `buildViewpointOptions`). */
  groups: readonly ViewpointOptionGroup[]
  /** Nom accessible de la commande — le texte visible est un gamertag, il ne se nomme pas seul. */
  label: string
  /** Poser le point de vue. `null` revient au joueur de la page. */
  onSelect: (xuid: string | null) => void
}

export function ReplayViewpointSelect({ value, groups, label, onSelect }: ReplayViewpointSelectProps) {
  const courant = groups.flatMap((g) => g.options).find((o) => o.value === value)
  return (
    <select
      // MÊME ENCRE ET MÊME TAILLE QUE LES AUTRES LIBELLÉS de piste (`TrackLabel`) : la rangée
      // ne doit pas se distinguer par son poids visuel, seulement par le fait qu'elle s'ouvre.
      // `bg-transparent` la laisse sur le fond de la frise ; la flèche NATIVE est conservée —
      // c'est elle qui dit, sans un mot, que ce libellé-ci se déroule.
      //
      // LA HAUTEUR EST CONTRAINTE (`h-4`, `leading-none`) parce qu'un champ natif prend sinon
      // sa hauteur de formulaire — une vingtaine de pixels là où les pistes en font quatorze.
      // Le budget vertical de cette page est la ressource rare (chaque pixel rendu par le
      // transport devient un pixel de terrain) : la rangée ne doit pas grandir parce qu'elle
      // est devenue une commande. `truncate` fait le reste — la liste ouverte, elle, n'est pas
      // contrainte par cette largeur, et le nom entier vit dans `title`.
      className="h-4 w-full max-w-full cursor-pointer truncate rounded bg-transparent text-[9.5px] font-semibold uppercase leading-none tracking-[0.14em] text-muted-foreground transition-colors hover:text-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
      value={value ?? ''}
      aria-label={label}
      title={courant?.label ?? label}
      onChange={(e) => {
        onSelect(e.target.value || null)
        // LE FOCUS PART AVEC LE CHOIX (cf. l'en-tête) : sinon Espace rouvre la liste au lieu de
        // mettre le rejeu en pause. `currentTarget` et pas `target` — l'événement est déjà
        // remonté quand React le rejoue, et `target` peut avoir été recyclé.
        e.currentTarget.blur()
      }}
    >
      {groups.map((groupe) => (
        <optgroup key={groupe.key} label={groupe.label}>
          {groupe.options.map((option) => (
            <option
              key={`${groupe.key}:${option.value}`}
              value={option.value}
              disabled={option.disabled}
              title={option.title}
            >
              {option.label}
            </option>
          ))}
        </optgroup>
      ))}
    </select>
  )
}
