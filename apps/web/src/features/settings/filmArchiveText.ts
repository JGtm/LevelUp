/**
 * Textes de l'archive des films — FR et EN.
 *
 * Dictionnaire de feature (même couche que `poolSharingText.ts`) : la même carte sert l'assistant
 * de premier lancement et les réglages, les textes y restent donc identiques. Chaque langue de
 * `config/locales.toml` figure ici (parité par typage : `Record<Locale, FilmArchiveText>`).
 *
 * Les refus du serveur (combinaison « sans archive + rétention finie », déplacement, dossier non
 * vide) arrivent déjà rédigés dans la langue de la requête : la carte les affiche tels quels.
 */
import type { Locale } from '@/lib/i18n/locale'

export interface FilmArchiveText {
  title: string
  wizardIntro: string
  toggleLabel: string
  toggleHintOn: string
  toggleHintOff: string
  locationLabel: string
  locationPlaceholder: string
  locationHint: string
  locationSave: string
  moveHint: string
  restartRequired: string
  occupancyLabel: string
  filmOne: string
  filmMany: string
  volumeLabel: string
  volumeFree: string
  volumeWarn: string
  volumeCritical: string
  partial: string
  loading: string
  unavailable: string
  saveFailed: string
}

const FR: FilmArchiveText = {
  title: 'Archive des films',
  wizardIntro:
    "Les films des parties sont conservés sur ce poste pour pouvoir reconstruire les rejeux. Ce choix se change plus tard dans les réglages.",
  toggleLabel: 'Conserver les films',
  toggleHintOn:
    "Un film expire côté Halo et ne se retélécharge jamais : l'archive est la seule copie. Elle permet de reconstruire un rejeu à tout moment.",
  toggleHintOff:
    "Les rejeux sont construits puis les films sont supprimés du poste. Un rejeu perdu ne pourra plus être reconstruit : la durée de conservation des rejeux reste illimitée.",
  locationLabel: "Dossier de l'archive",
  locationPlaceholder: 'Emplacement par défaut',
  locationHint:
    "Chemin absolu d'un dossier vide. Vide : emplacement par défaut. Tant que l'archive est vide, le nouveau dossier sert tout de suite.",
  locationSave: 'Enregistrer le dossier',
  moveHint:
    "L'archive contient déjà des films : pour la déplacer, lancer « levelup film archive-move --to <dossier> » programme arrêté.",
  restartRequired: 'Redémarrer le programme pour utiliser le dossier enregistré.',
  occupancyLabel: 'Occupation',
  filmOne: 'film',
  filmMany: 'films',
  volumeLabel: 'Espace du volume',
  volumeFree: '{free} libres sur {total}',
  volumeWarn: 'Volume presque plein',
  volumeCritical: 'Volume saturé',
  partial: 'Mesure interrompue : les totaux sont des minimums.',
  loading: "Mesure de l'archive…",
  unavailable: "L'occupation de l'archive est indisponible.",
  saveFailed: "Le réglage n'a pas pu être enregistré.",
}

const EN: FilmArchiveText = {
  title: 'Film archive',
  wizardIntro:
    'Match films are kept on this computer so replays can be rebuilt. You can change this later in the settings.',
  toggleLabel: 'Keep films',
  toggleHintOn:
    'A film expires on the Halo side and can never be downloaded again: the archive is the only copy. It lets a replay be rebuilt at any time.',
  toggleHintOff:
    'Replays are built, then the films are deleted from this computer. A lost replay can no longer be rebuilt: replay retention stays unlimited.',
  locationLabel: 'Archive folder',
  locationPlaceholder: 'Default location',
  locationHint:
    'Absolute path of an empty folder. Empty: default location. While the archive is empty, the new folder is used right away.',
  locationSave: 'Save folder',
  moveHint:
    'The archive already holds films: to move it, run "levelup film archive-move --to <folder>" with the program stopped.',
  restartRequired: 'Restart the program to use the saved folder.',
  occupancyLabel: 'Usage',
  filmOne: 'film',
  filmMany: 'films',
  volumeLabel: 'Volume space',
  volumeFree: '{free} free of {total}',
  volumeWarn: 'Volume almost full',
  volumeCritical: 'Volume full',
  partial: 'Measurement interrupted: totals are lower bounds.',
  loading: 'Measuring the archive…',
  unavailable: 'Archive usage is unavailable.',
  saveFailed: 'The setting could not be saved.',
}

const TEXT: Record<Locale, FilmArchiveText> = { fr: FR, en: EN }

export function getFilmArchiveText(locale: Locale): FilmArchiveText {
  return TEXT[locale]
}
