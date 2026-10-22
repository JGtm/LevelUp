/**
 * Texte des conditions d'utilisation — FR et EN.
 *
 * Même doctrine que `./i18n` (politique de confidentialité) : ce texte décrit ce que le code
 * fait RÉELLEMENT. La section « Mise en commun des jetons » doit rester cohérente avec la
 * section du même nom de la politique de confidentialité et avec les textes du consentement
 * (`features/settings/poolSharingText.ts`) : toute évolution de l'un se répercute sur les
 * deux autres DANS LE MÊME COMMIT, et fait avancer `TERMS_UPDATED_AT`.
 *
 * Points vérifiés sur pièces au 2026-10-16 :
 *   - le jeton n'entre dans le round-robin public que sur un accord enregistré
 *     (`auth.UserTokens.PoolSharing`, `pool.CredentialSource.PoolExcluded`) ; sans décision
 *     ou après un refus, il ne sert que son propriétaire (`PooledHaloClient.acquire`)
 *   - le retrait est appliqué à la requête suivante (`Pool.SetShared`)
 *   - les joueurs antérieurs au réglage sont marqués `grandfathered_explicit`
 *     (`levelup pool-sharing grandfather`)
 */
import type { Locale } from '@/lib/i18n/locale'
import { CONTACT_TOKEN, type PrivacyText } from './i18n'

/** Date de dernière révision de fond, affichée en tête de page. */
export const TERMS_UPDATED_AT: Record<Locale, string> = {
  fr: '20 octobre 2026',
  en: '20 October 2026',
}

const FR_TEXT: PrivacyText = {
  title: "Conditions d'utilisation",
  updatedLabel: 'Dernière mise à jour',
  intro: [
    "LevelUp est un projet personnel et non commercial. Ces conditions s'appliquent à l'instance publique opérée par l'auteur du projet ; une instance que vous hébergez vous-même est sous votre responsabilité. En connectant votre compte Microsoft ou en utilisant le service, vous les acceptez.",
    "La façon dont vos données sont traitées est décrite dans la politique de confidentialité. Ces conditions décrivent ce que vous pouvez attendre du service, ce qu'il attend de vous, et en particulier la mise en commun des jetons.",
  ],
  sections: [
    {
      heading: 'Le service',
      paragraphs: [
        "LevelUp affiche des statistiques de jeu Halo à partir de l'API de Microsoft et de la lecture des films de parties. C'est un projet indépendant, sans lien avec Microsoft ni Halo Studios. Halo et ses contenus appartiennent à leurs détenteurs.",
      ],
    },
    {
      heading: 'Votre compte',
      bullets: [
        "la connexion se fait chez Microsoft : votre mot de passe ne passe jamais par LevelUp",
        "vous restez responsable de votre compte Microsoft et de ses propres conditions d'utilisation",
        "vous pouvez retirer l'autorisation à tout moment depuis la page de gestion des accès de votre compte Microsoft, et demander la suppression de vos données (voir la politique de confidentialité)",
        "une instance peut être fermée aux nouvelles inscriptions",
      ],
    },
    {
      heading: 'Mise en commun des jetons',
      paragraphs: [
        "La synchronisation des parties est limitée par un débit que Microsoft impose à chaque compte. Pour la rendre plus rapide, LevelUp peut répartir ses requêtes sur les jetons des joueurs qui y ont consenti. C'est un choix, jamais une condition d'accès au service. Les profils suivis sans compte connecté, eux, ne se synchronisent que grâce aux jetons mis en commun : si personne ne partage le sien, ils ne sont plus synchronisés.",
        "Ces requêtes sont faites au nom de votre compte Microsoft. Si Microsoft restreint un jour ce compte ou l'accès à son API, cela peut vous concerner : l'auteur ne peut pas l'empêcher et ne le garantit pas.",
        "L'auteur peut retirer un jeton du pool s'il est révoqué, défaillant ou s'il provoque des erreurs répétées.",
        "Les joueurs qui participaient au projet avant l'existence de ce réglage ont donné leur accord directement à l'auteur ; il est enregistré comme tel, et ils peuvent le retirer comme n'importe qui.",
      ],
      bullets: [
        "ce que vous autorisez : qu'une requête de synchronisation, y compris pour un autre joueur, soit faite avec votre jeton pour lire des données publiques de parties",
        "ce que vous n'autorisez pas : lire vos données privées pour un autre joueur, ou montrer, copier ou transmettre votre jeton à quiconque",
        "en contrepartie : vos propres parties sont synchronisées par l'ensemble des jetons mis en commun",
        "si vous refusez ou ne choisissez pas : votre jeton ne sert que votre propre synchronisation, qui reste possible mais plus lente, limitée au débit de votre seul compte",
        "retrait : à tout moment dans Réglages, onglet Compte, avec effet à la requête suivante et sans conséquence sur le reste du service",
      ],
    },
    {
      heading: 'Usage attendu',
      bullets: [
        "ne pas utiliser l'instance pour récupérer des données en masse, contourner les limites de débit de l'API ou perturber le service",
        "ne pas chercher à accéder aux données d'un autre joueur sans son accord",
        "ne rien écrire dans le panneau « Signaler un problème » qui ne doive pas être public",
      ],
    },
    {
      heading: 'Disponibilité et garanties',
      paragraphs: [
        "Le service est fourni en l'état, sans garantie de disponibilité ni d'exactitude. Il repose sur des interfaces de Microsoft qui ne sont pas publiquement documentées et peuvent changer ou se fermer sans préavis : une fonction peut cesser de marcher, et une partie dont le film a expiré chez Microsoft ne peut plus être analysée au-delà de ses statistiques. L'auteur peut modifier, suspendre ou arrêter l'instance.",
      ],
    },
    {
      heading: 'Code et contenus',
      paragraphs: [
        "Le code est publié sous licence AGPL-3.0 et la documentation sous licence CC BY 4.0 (voir le fichier NOTICE du dépôt). Cette licence ne couvre pas les contenus de Halo (images, sons, données du jeu), qui restent à Microsoft et à ses ayants droit.",
      ],
    },
    {
      heading: 'Modifications',
      paragraphs: [
        "Ces conditions portent la date de leur dernière mise à jour. Un changement qui élargirait l'usage de votre jeton ne s'applique qu'avec un nouvel accord de votre part.",
      ],
    },
    {
      heading: 'Contact',
      paragraphs: [`Pour toute question sur ces conditions, écrivez à ${CONTACT_TOKEN}.`],
    },
  ],
  backToApp: "Retour à l'application",
}

const EN_TEXT: PrivacyText = {
  title: 'Terms of use',
  updatedLabel: 'Last updated',
  intro: [
    'LevelUp is a personal, non-commercial project. These terms apply to the public instance operated by the project author; an instance you host yourself is your own responsibility. By connecting your Microsoft account or using the service, you accept them.',
    'How your data is handled is described in the privacy policy. These terms describe what you can expect from the service, what it expects from you, and in particular how tokens are pooled.',
  ],
  sections: [
    {
      heading: 'The service',
      paragraphs: [
        "LevelUp shows Halo game statistics from Microsoft's API and from reading match films. It is an independent project, not affiliated with Microsoft or Halo Studios. Halo and its content belong to their respective owners.",
      ],
    },
    {
      heading: 'Your account',
      bullets: [
        'sign-in happens at Microsoft: your password never passes through LevelUp',
        'you remain responsible for your Microsoft account and its own terms of use',
        'you can withdraw the authorisation at any time from your Microsoft account access management page, and ask for your data to be deleted (see the privacy policy)',
        'an instance may be closed to new sign-ups',
      ],
    },
    {
      heading: 'Token pooling',
      paragraphs: [
        'Match syncing is limited by a rate that Microsoft imposes on each account. To make it faster, LevelUp can spread its requests across the tokens of players who consented. It is a choice, never a condition for using the service. Profiles followed without a connected account, however, are synced only through pooled tokens: if nobody shares theirs, they are no longer synced.',
        "These requests are made in the name of your Microsoft account. If Microsoft ever restricts that account or access to its API, you may be affected: the author cannot prevent it and does not guarantee against it.",
        'The author may remove a token from the pool if it is revoked, failing, or causing repeated errors.',
        'Players who took part in the project before this setting existed gave their agreement to the author directly; it is recorded as such, and they can withdraw it like anyone else.',
      ],
      bullets: [
        'what you allow: a sync request, including one for another player, being made with your token to read public match data',
        'what you do not allow: reading your private data for another player, or showing, copying or passing your token to anyone',
        'in return: your own matches are synced by all the pooled tokens',
        'if you decline or do not choose: your token only serves your own sync, which still works but is slower, limited to the rate of your single account',
        'withdrawal: at any time in Settings, Account tab, effective from the next request and with no effect on the rest of the service',
      ],
    },
    {
      heading: 'Expected use',
      bullets: [
        'do not use the instance to harvest data in bulk, get around the API rate limits, or disrupt the service',
        "do not try to reach another player's data without their agreement",
        'do not write anything in the "Report an issue" panel that should not be public',
      ],
    },
    {
      heading: 'Availability and warranties',
      paragraphs: [
        "The service is provided as is, with no guarantee of availability or accuracy. It relies on Microsoft interfaces that are not publicly documented and may change or close without notice: a feature may stop working, and a match whose film has expired at Microsoft can no longer be analysed beyond its statistics. The author may change, suspend or shut down the instance.",
      ],
    },
    {
      heading: 'Code and content',
      paragraphs: [
        "The code is published under the AGPL-3.0 licence and the documentation under CC BY 4.0 (see the NOTICE file in the repository). That licence does not cover Halo content (images, sounds, game data), which remains the property of Microsoft and its licensors.",
      ],
    },
    {
      heading: 'Changes',
      paragraphs: [
        'These terms carry the date of their last update. A change that would widen the use of your token applies only with a new agreement from you.',
      ],
    },
    {
      heading: 'Contact',
      paragraphs: [`For any question about these terms, write to ${CONTACT_TOKEN}.`],
    },
  ],
  backToApp: 'Back to the application',
}

const TEXT: Record<Locale, PrivacyText> = { fr: FR_TEXT, en: EN_TEXT }

export function getTermsText(locale: Locale): PrivacyText {
  return TEXT[locale]
}
