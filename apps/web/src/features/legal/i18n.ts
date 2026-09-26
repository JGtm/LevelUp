/**
 * Texte de la politique de confidentialité — FR et EN.
 *
 * Dictionnaire de feature (couche 1 du skill frontend-patterns) plutôt que
 * manifest TOML : ce sont des paragraphes de prose juridique, pas des libellés
 * d'interface. Les deux langues vivent côte à côte pour qu'une modification de
 * fond ne puisse pas n'atterrir que dans une seule.
 *
 * DOCTRINE — ce texte décrit ce que le code fait RÉELLEMENT. Toute évolution qui
 * change la collecte, la conservation ou les destinataires (nouvel appel réseau
 * sortant, nouvelle donnée persistée, service tiers ajouté) doit mettre ce
 * fichier à jour DANS LE MÊME COMMIT, et faire avancer `updatedAt`.
 *
 * Points vérifiés sur pièces au 2026-08-31 :
 *   - cookie `levelup_session` : HttpOnly, SameSite=Lax, Secure en HTTPS, TTL 7 j
 *     (`internal/platform/session/store.go`, `internal/api/middleware/session.go`)
 *   - aucun traceur ni service d'analyse dans le bundle web (grep analytics /
 *     gtag / plausible / matomo / posthog / sentry : aucun branchement)
 *   - jetons Microsoft : `data/auth/watcher_tokens/{xuid}.json`, source unique
 *     (ADR 0023)
 *   - appel navigateur direct vers `api.github.com` depuis le panneau de retour
 *     (`features/feedback-drawer/queries.ts`, `credentials: 'omit'`)
 */
import type { Locale } from '@/lib/i18n/locale'

/**
 * Jeton remplacé à l'affichage par un lien `mailto:` vers l'alias de contact
 * (`privacyContactEmail()`). Même mécanique que `{{HP}}` dans l'aide : le texte
 * reste une chaîne de prose, la page se charge de l'insertion.
 */
export const CONTACT_TOKEN = '{{CONTACT}}'

export interface PrivacySection {
  heading: string
  paragraphs?: string[]
  bullets?: string[]
}

export interface PrivacyText {
  title: string
  updatedLabel: string
  intro: string[]
  sections: PrivacySection[]
  backToApp: string
}

/** Date de dernière révision de fond, affichée en tête de page. */
export const PRIVACY_UPDATED_AT: Record<Locale, string> = {
  fr: '31 août 2026',
  en: '31 August 2026',
}

const FR_TEXT: PrivacyText = {
  title: 'Confidentialité',
  updatedLabel: 'Dernière mise à jour',
  intro: [
    "LevelUp est un projet personnel et non commercial, publié en source ouverte sous licence MIT. Cette page décrit l'instance publique opérée par l'auteur du projet. Une instance que vous hébergez vous-même est sous votre propre responsabilité : le code est le même, l'exploitation ne l'est pas.",
    "Le principe : LevelUp ne collecte que ce dont il a besoin pour afficher vos statistiques de jeu. Il n'y a ni publicité, ni service d'analyse d'audience, ni revente de données.",
  ],
  sections: [
    {
      heading: 'Ce qui est conservé',
      paragraphs: [
        "Lorsque vous connectez votre compte Microsoft, LevelUp enregistre votre identifiant Xbox (XUID), votre gamertag et un jeton de rafraîchissement délivré par Microsoft. Ce jeton sert uniquement à interroger l'API Halo officielle en votre nom. Votre mot de passe Microsoft ne transite jamais par LevelUp : l'authentification se fait chez Microsoft.",
        "À partir de cette autorisation, l'application récupère et conserve vos données de jeu telles que l'API officielle les expose :",
      ],
      bullets: [
        "historique de parties, statistiques par partie, modes, cartes et listes de lecture",
        "médailles, citations, rang de carrière et classement compétitif (CSR)",
        "données sociales Xbox (abonnements, activité) lorsque la fonctionnalité est utilisée",
        "indicateurs dérivés calculés localement à partir de ce qui précède, dont la note interne de niveau",
      ],
    },
    {
      heading: 'Les autres joueurs de vos parties',
      paragraphs: [
        "Une partie a des coéquipiers et des adversaires. Leurs gamertags, identifiants Xbox et statistiques dans ces parties sont conservés au même titre que les vôtres : c'est indissociable du fait d'afficher le déroulé d'un match. Ces informations proviennent de la même API officielle et ne sont pas enrichies par d'autres sources.",
      ],
    },
    {
      heading: 'Cookies et stockage du navigateur',
      paragraphs: [
        "Un seul cookie est posé : « levelup_session ». Il identifie votre session, il est strictement nécessaire au fonctionnement du site, et il n'a aucune finalité publicitaire ou de mesure d'audience. Il est inaccessible au JavaScript de la page (HttpOnly), limité aux navigations depuis le site (SameSite=Lax), transmis uniquement en HTTPS, et il expire au bout de sept jours.",
        "Vos préférences d'affichage (thème, palette de couleurs, filtres, réglages du rejeu) sont enregistrées dans le stockage local de votre navigateur. Elles ne sont jamais envoyées au serveur et disparaissent si vous videz les données du site.",
      ],
    },
    {
      heading: "Ce que LevelUp ne fait pas",
      bullets: [
        "aucun traceur publicitaire, aucun service de mesure d'audience, aucun cookie tiers",
        "aucune revente, aucun partage commercial, aucun transfert à un annonceur",
        "aucun profilage en dehors des indicateurs de jeu affichés dans l'application",
      ],
    },
    {
      heading: 'Services tiers réellement contactés',
      paragraphs: [
        "Microsoft — authentification et source de toutes les données de jeu (API Xbox Live et Halo). Vous pouvez retirer l'autorisation à tout moment depuis la page de gestion des accès de votre compte Microsoft ; LevelUp perd alors la possibilité de synchroniser vos parties.",
        "GitHub — uniquement si vous ouvrez le panneau « Signaler un problème » : votre navigateur interroge directement l'interface publique de GitHub pour chercher des signalements similaires, ce qui rend votre adresse IP visible par GitHub. Si vous validez l'envoi, un ticket est créé sur le dépôt public du projet et contient ce que vous avez écrit ainsi qu'un contexte technique (page consultée, filtres actifs, erreurs du navigateur). Ne rien y écrire que vous ne voudriez pas voir publié.",
        "Discord — uniquement si l'administrateur de l'instance a configuré un lien de notification. Dans ce cas, des notifications de l'application sont envoyées vers le salon correspondant.",
      ],
    },
    {
      heading: 'Hébergement et durée de conservation',
      paragraphs: [
        "Les données sont stockées sur le serveur qui héberge l'instance, dans des fichiers de base de données locaux. Les sessions expirent au bout de sept jours. Le jeton Microsoft est conservé tant que votre compte reste connecté à l'application ; il est renouvelé automatiquement et n'est jamais redemandé tant qu'il reste valide. Les données de jeu sont conservées tant que vous utilisez l'application, puisqu'elles constituent précisément l'historique qu'elle affiche.",
      ],
    },
    {
      heading: 'Vos droits',
      paragraphs: [
        "Vous pouvez demander l'accès à vos données, leur correction, leur portabilité ou leur suppression. La suppression retire votre base de joueur et votre jeton Microsoft ; les parties dans lesquelles vous apparaissez chez d'autres joueurs relèvent de la même API publique et suivent le sort de leurs comptes.",
        `Pour exercer ces droits, écrivez à ${CONTACT_TOKEN}, l'adresse de contact du projet. Vous pouvez aussi ouvrir un ticket sur le dépôt public, en gardant à l'esprit qu'il sera visible de tous — l'adresse est la voie discrète.`,
        "Si vous utilisez une instance hébergée par quelqu'un d'autre, adressez-vous à la personne qui l'opère : c'est elle qui détient les données.",
      ],
    },
    {
      heading: 'Sans lien avec Microsoft',
      paragraphs: [
        "LevelUp est un projet indépendant, sans lien avec Microsoft ni Halo Studios. Halo et les contenus associés appartiennent à leurs détenteurs respectifs.",
      ],
    },
  ],
  backToApp: "Retour à l'application",
}

const EN_TEXT: PrivacyText = {
  title: 'Privacy',
  updatedLabel: 'Last updated',
  intro: [
    'LevelUp is a personal, non-commercial project released as open source under the MIT licence. This page covers the public instance operated by the project author. An instance you host yourself is your own responsibility: the code is the same, the operation is not.',
    'The principle: LevelUp collects only what it needs to show your game statistics. There is no advertising, no audience measurement service, and no sale of data.',
  ],
  sections: [
    {
      heading: 'What is stored',
      paragraphs: [
        'When you connect your Microsoft account, LevelUp records your Xbox identifier (XUID), your gamertag, and a refresh token issued by Microsoft. That token is used solely to query the official Halo API on your behalf. Your Microsoft password never passes through LevelUp: authentication happens at Microsoft.',
        'From that authorisation, the application retrieves and stores your game data as the official API exposes it:',
      ],
      bullets: [
        'match history, per-match statistics, modes, maps and playlists',
        'medals, commendations, career rank and competitive rank (CSR)',
        'Xbox social data (follows, activity) when that feature is used',
        'derived indicators computed locally from the above, including the internal skill rating',
      ],
    },
    {
      heading: 'Other players in your matches',
      paragraphs: [
        'A match has teammates and opponents. Their gamertags, Xbox identifiers and statistics in those matches are stored alongside yours: that is inseparable from showing how a match unfolded. This information comes from the same official API and is not enriched from any other source.',
      ],
    },
    {
      heading: 'Cookies and browser storage',
      paragraphs: [
        'A single cookie is set: "levelup_session". It identifies your session, it is strictly necessary for the site to work, and it serves no advertising or audience-measurement purpose. It is unreadable by page JavaScript (HttpOnly), limited to navigation from the site (SameSite=Lax), sent over HTTPS only, and it expires after seven days.',
        'Your display preferences (theme, colour palette, filters, replay settings) are kept in your browser local storage. They are never sent to the server and disappear if you clear the site data.',
      ],
    },
    {
      heading: 'What LevelUp does not do',
      bullets: [
        'no advertising tracker, no audience measurement service, no third-party cookie',
        'no sale, no commercial sharing, no transfer to an advertiser',
        'no profiling beyond the game indicators shown in the application',
      ],
    },
    {
      heading: 'Third parties actually contacted',
      paragraphs: [
        'Microsoft — authentication and the source of all game data (Xbox Live and Halo APIs). You can withdraw the authorisation at any time from your Microsoft account access management page; LevelUp then loses the ability to sync your matches.',
        'GitHub — only if you open the "Report an issue" panel: your browser queries GitHub’s public interface directly to look for similar reports, which makes your IP address visible to GitHub. If you confirm the submission, a ticket is created on the public repository containing what you wrote plus technical context (current page, active filters, browser errors). Do not write anything there you would not want published.',
        'Discord — only if the instance administrator configured a notification link. In that case, application notifications are sent to the corresponding channel.',
      ],
    },
    {
      heading: 'Hosting and retention',
      paragraphs: [
        'Data is stored on the server hosting the instance, in local database files. Sessions expire after seven days. The Microsoft token is kept for as long as your account stays connected to the application; it is refreshed automatically and never requested again while it remains valid. Game data is kept for as long as you use the application, since it is precisely the history the application displays.',
      ],
    },
    {
      heading: 'Your rights',
      paragraphs: [
        'You may request access to your data, its correction, its portability or its deletion. Deletion removes your player database and your Microsoft token; matches in which you appear on other players’ accounts come from the same public API and follow the fate of those accounts.',
        `To exercise these rights, write to ${CONTACT_TOKEN}, the project’s contact address. You may also open a ticket on the public repository, bearing in mind that it will be visible to everyone — the address is the discreet route.`,
        'If you use an instance hosted by someone else, contact whoever operates it: they are the ones holding the data.',
      ],
    },
    {
      heading: 'Not affiliated with Microsoft',
      paragraphs: [
        'LevelUp is an independent project, not affiliated with Microsoft or Halo Studios. Halo and related content belong to their respective owners.',
      ],
    },
  ],
  backToApp: 'Back to the application',
}

const TEXT: Record<Locale, PrivacyText> = { fr: FR_TEXT, en: EN_TEXT }

export function getPrivacyText(locale: Locale): PrivacyText {
  return TEXT[locale]
}
