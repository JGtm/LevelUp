/**
 * i18n FR/EN du système de notifications.
 *
 * Pattern : dictionnaire typé par locale, accessible via getNotificationsText(locale).
 * Aligné avec features/{settings,help,...}/i18n.ts.
 *
 * Les clés notif.<category>.title et notif.<category>.body correspondent aux
 * title_key/body_key reçus du backend. Résolues par format.ts en injectant
 * les params (interpolation simple {name}, {count} sans plurals ICU pour le MVP).
 */
import type { NotificationCategory } from './types'
import type { Locale } from '@/lib/i18n/locale'

export interface NotificationsText {
  // Cloche & badge
  bellAriaLabelEmpty: string
  bellAriaLabelWithCount: string // "{count} notifications non lues"

  // Dropdown
  dropdownTitle: string
  dropdownUnread: string
  dropdownOlder: string
  dropdownEmpty: string
  dropdownMarkAllRead: string
  dropdownViewAll: string
  dropdownErrorLoading: string

  // Actions item
  actionMarkAsRead: string
  actionMarkAsUnread: string
  actionDismiss: string
  actionView: string

  // Page dédiée
  pageTitle: string
  pageFilterAll: string
  pageFilterUnread: string
  pageGroupToday: string
  pageGroupYesterday: string
  pageGroupThisWeek: string
  pageGroupOlder: string
  pageBulkMarkRead: string
  pageBulkDismiss: string
  pageLoadMore: string
  pageEmpty: string

  // Settings tab
  settingsTitle: string
  settingsDescription: string
  settingsMaster: string
  settingsMasterDescription: string
  settingsToasts: string
  settingsToastsDescription: string
  settingsCategoriesTitle: string
  settingsCategoriesDescription: string
  settingsDeliveryBoth: string
  settingsDeliveryInApp: string
  settingsDeliveryToast: string
  settingsDeliveryOff: string
  settingsSubscriptionsTitle: string
  settingsSubscriptionsDescription: string
  settingsRetentionTitle: string
  settingsRetentionLabel: string
  settingsTestButton: string
  settingsTestSent: string

  // Catégories — label + description (pour Settings)
  categoryLabel: Record<NotificationCategory, string>
  categoryDescription: Record<NotificationCategory, string>

  // Mapping clé métrique (envoyée par le backend) → libellé localisé.
  // Sert à éviter tout hardcode "KD"/"Winrate" dans les templates.
  metricLabel: Record<string, string>

  // Mapping fenêtre temporelle records V2 → libellé localisé (30d/90d/all_time).
  periodLabel: Record<string, string>

  // Templates de notif (rendus à partir de title_key/body_key + params)
  // Convention : "notif.<category>.title" / ".body"
  // Gardé sous forme de Record<string,string> pour permettre extension future
  // sans modifier le typage.
  templates: Record<string, string>

  // Temps relatif
  relJustNow: string
  relMinutesAgo: (n: number) => string
  relHoursAgo: (n: number) => string
  relDaysAgo: (n: number) => string
  relOnDate: (iso: string) => string
}

const FR: NotificationsText = {
  bellAriaLabelEmpty: 'Aucune notification non lue',
  bellAriaLabelWithCount: '{count} notifications non lues',

  dropdownTitle: 'Notifications',
  dropdownUnread: 'Non lues',
  dropdownOlder: 'Plus récentes',
  dropdownEmpty: 'Tout est calme par ici',
  dropdownMarkAllRead: 'Tout marquer comme lu',
  dropdownViewAll: 'Voir tout',
  dropdownErrorLoading: 'Impossible de charger les notifications',

  actionMarkAsRead: 'Marquer comme lu',
  actionMarkAsUnread: 'Marquer comme non lu',
  actionDismiss: 'Ignorer',
  actionView: 'Voir',

  pageTitle: 'Notifications',
  pageFilterAll: 'Toutes',
  pageFilterUnread: 'Non lues uniquement',
  pageGroupToday: "Aujourd'hui",
  pageGroupYesterday: 'Hier',
  pageGroupThisWeek: 'Cette semaine',
  pageGroupOlder: 'Plus ancien',
  pageBulkMarkRead: 'Marquer comme lues ({count})',
  pageBulkDismiss: 'Ignorer ({count})',
  pageLoadMore: 'Charger plus',
  pageEmpty: 'Aucune notification pour le moment',

  settingsTitle: 'Notifications',
  settingsDescription:
    'Gère les notifications dans le panneau, les pop-ups et les abonnements par catégorie.',
  settingsMaster: 'Activer les notifications',
  settingsMasterDescription:
    "Désactive complètement la cloche et les pop-ups. Les événements restent enregistrés en base.",
  settingsToasts: 'Activer les pop-ups',
  settingsToastsDescription:
    "Affiche les nouvelles notifications en pop-up à l'écran (en plus du panneau cloche).",
  settingsCategoriesTitle: 'Catégories',
  settingsCategoriesDescription:
    "Choisis pour chaque type d'événement comment tu veux être notifié.",
  settingsDeliveryBoth: 'Panneau + pop-up',
  settingsDeliveryInApp: 'Panneau uniquement',
  settingsDeliveryToast: 'Pop-up uniquement',
  settingsDeliveryOff: 'Désactivé',
  settingsSubscriptionsTitle: 'Abonnements',
  settingsSubscriptionsDescription:
    'Active ou désactive le suivi automatique pour le joueur courant.',
  settingsRetentionTitle: 'Rétention',
  settingsRetentionLabel: 'Conserver les {n} dernières notifications',
  settingsTestButton: 'Envoyer une notification de test',
  settingsTestSent: 'Notification de test envoyée',

  categoryLabel: {
    app_release: 'Nouveauté app',
    match_synced: 'Match synchronisé',
    media_added: 'Média ajouté',
    media_liked: 'Média aimé',
    objective_assigned: 'Nouvel objectif',
    objective_completed: 'Objectif complété',
    challenge_added: 'Nouveau défi',
    challenge_completed: 'Défi complété',
    season_pass_level: 'Niveau de passe saisonnier (déprécié)',
    sync_error: 'Erreur de synchronisation',
    personal_record: 'Record personnel',
    threshold_crossed: 'Palier franchi',
    friend_added: 'Ami ajouté',
    friend_sync_completed: 'Sessions amis mises à jour',
    data_health_warning: 'Audit base de données',
    career_rank: 'Rang de carrière',
    skill_tier: 'Palier CSR / LUSR',
    battlepass_completed: 'Pass saisonnier terminé',
    citation_tier: 'Palier de citation',
    citation_mastery: 'Citation maîtrisée',
    record_near_miss: 'Approche d\'un record',
    milestone_unlocked: 'Jalon débloqué',
    milestone_near_miss: 'Approche d\'un jalon',
    lusr_tier_approach: 'Approche d\'un palier LUSR',
    streak_milestone: 'Palier de série',
    comeback_welcome: 'Bienvenue de retour',
    trend_consolidate: 'Axe à consolider',
    title_ready: 'Titre prêt',
    rival_encounter: 'Rival croisé',
  },
  categoryDescription: {
    app_release: 'Une nouvelle version de LevelUp est disponible.',
    match_synced: 'Tes nouveaux matchs ont été synchronisés.',
    media_added: "Une vidéo ou capture d'écran a été ajoutée.",
    media_liked: 'Quelqu\'un a aimé un de tes médias.',
    objective_assigned: 'Un objectif a été créé automatiquement.',
    objective_completed: 'Tu as terminé un objectif.',
    challenge_added: 'Un nouveau défi est disponible.',
    challenge_completed: 'Tu as terminé un défi quotidien ou hebdomadaire.',
    season_pass_level: 'Catégorie historique remplacée par « Rang de carrière » et « Pass saisonnier terminé ».',
    sync_error: 'La synchronisation a échoué — relance manuelle conseillée.',
    personal_record: 'Tu as battu un record personnel.',
    threshold_crossed: 'Un palier de ratio FDA ou de taux de victoire a été franchi.',
    friend_added: 'Un gamertag a été ajouté à ta liste d\'amis.',
    friend_sync_completed: 'Des matchs ont été reclassés en escouade après ajout d\'ami.',
    data_health_warning: 'Anomalies détectées dans la base par l\'audit périodique (UUIDs bruts, bits incohérents, URLs résiduelles).',
    career_rank: 'Tu as gagné un nouveau rang de carrière Halo (cumul de carrière).',
    skill_tier: 'Tu as changé de palier compétitif sur une sélection (CSR ou LUSR).',
    battlepass_completed: 'Tu as terminé un pass saisonnier (rang max atteint).',
    citation_tier: 'Tu as franchi un nouveau palier sur une citation.',
    citation_mastery: 'Tu as maîtrisé une citation à 100 %.',
    record_near_miss: 'Ton score courant approche d\'un de tes records personnels.',
    milestone_unlocked: 'Tu viens de débloquer un jalon (palier cumulatif).',
    milestone_near_miss: 'Tu es proche de débloquer un jalon.',
    lusr_tier_approach: 'Ton score LUSR approche du prochain sous-palier.',
    streak_milestone: 'Ta série atteint un palier (multiplicateur PP).',
    comeback_welcome: 'Tu reviens après une pause — bienvenue !',
    trend_consolidate: 'Une composante de ta performance fléchit sur la durée — une occasion de la renforcer.',
    title_ready: 'Un titre fraîchement activé a terminé sa première synchronisation.',
    rival_encounter: 'Une sync a ramené un nouveau duel contre un de tes principaux rivaux.',
  },

  // metricLabel : mapping des clés métriques (envoyées par le backend dans
  // params.metric_key OU params.metric) vers leur libellé localisé. Évite
  // tout hardcode "KD", "Winrate" dans les templates ou le code Go.
  metricLabel: {
    kd_ratio: 'ratio K/D',
    winrate: 'taux de victoire',
    kda: 'KDA',
    // Progression V2 (envoyés par le coach generator avec la clé `metric`).
    performance_score: 'score de performance',
    kpm: 'tueries / minute',
    accuracy: 'précision',
    pspm: 'score perso / minute',
  },

  // periodLabel : mapping des fenêtres temporelles de records V2 vers leur
  // libellé localisé. Le backend envoie `params.period` = "30d"/"90d"/"all_time".
  periodLabel: {
    '30d': '30 jours',
    '90d': '90 jours',
    all_time: 'carrière',
  },

  templates: {
    'notif.app_release.title': 'Nouvelle version : {version}',
    'notif.app_release.body': 'Découvre les nouveautés dans le journal des modifications.',
    'notif.match_synced.title': '{count} match(s) synchronisé(s)',
    'notif.match_synced.body': 'Tes statistiques sont à jour.',
    'notif.media_added.title': 'Média ajouté par {actor_name}',
    'notif.media_added.body': '{count} fichier(s) associé(s) à un match.',
    'notif.media_liked.title': '{actor_name} a aimé ton média',
    'notif.media_liked.body': 'Va voir lequel sur la galerie.',
    'notif.objective_assigned.title': 'Nouvel objectif',
    'notif.objective_assigned.body': '{count} nouvel(s) objectif(s) attribué(s).',
    'notif.objective_completed.title': 'Objectif complété',
    'notif.objective_completed.body': '{count} objectif(s) terminé(s) — bravo !',
    'notif.challenge_added.title': 'Nouveau(x) défi(s) disponible(s)',
    'notif.challenge_added.body': '{count} nouveau(x) défi(s).',
    'notif.challenge_completed.title': 'Défi complété',
    'notif.challenge_completed.body': '{count} citation(s) gagnée(s).',
    'notif.season_pass_level.title': 'Niveau {level} atteint',
    'notif.season_pass_level.body': 'Tu progresses sur le passe saisonnier.',
    'notif.sync_error.title': 'Erreur de synchronisation',
    'notif.sync_error.body': '{message}',
    'notif.personal_record.title': 'Record personnel',
    'notif.personal_record.body': 'Nouveau record sur {metric_label} : {value}.',
    'notif.threshold_crossed.title': 'Palier franchi',
    'notif.threshold_crossed.body': 'Tu as franchi un palier de {metric_label} : {value}.',
    'notif.trend_consolidate.title': 'Axe à consolider',
    'notif.trend_consolidate.body': 'Une composante de ta performance baisse depuis quelque temps — l\'occasion de la renforcer.',
    'notif.friend_added.title': '{gamertag} ajouté à tes amis',
    'notif.friend_added.body': 'Les sessions communes seront reclassées en escouade en arrière-plan.',
    'notif.friend_sync_completed.title': 'Sessions amis mises à jour',
    'notif.friend_sync_completed.body': '{promoted} match(s) reclassé(s) en escouade-amis.',
    'notif.test.title': 'Notification de test',
    'notif.test.body': 'Le pipeline de notifications fonctionne correctement.',
    'notif.data_health_warning.title': 'Audit base : {warnings_total} anomalie(s) détectée(s)',
    'notif.data_health_warning.body': '{uuids_raw} UUID(s) brut(s), {lying_bits_events} bit(s) menteur(s), {garbage_banner_urls} URL(s) résiduelle(s). {hint}',
    'notif.career_rank.title': 'Rang {rank} atteint',
    'notif.career_rank.body': 'Tu viens de débloquer {rank_name} (depuis le rang {previous}).',
    'notif.skill_tier.title': 'Nouveau palier {tier} ({rating_type})',
    'notif.skill_tier.body': 'Sélection {playlist_group} — {tier} {sub_tier} (avant : {previous_tier} {previous_sub_tier}).',
    'notif.battlepass_completed.title': 'Pass saisonnier terminé',
    'notif.battlepass_completed.body': '{count} pass saisonnier(s) au rang max — félicitations !',
    'notif.citation_tier.title': 'Nouveau palier de citation',
    'notif.citation_tier.body': '{count} palier(s) franchi(s) depuis la dernière synchronisation.',
    'notif.citation_mastery.title': 'Citation maîtrisée',
    'notif.citation_mastery.body': '{count} citation(s) à 100 % — bravo !',
    // ─── Progression V2 (Ascension) — coach proactif ─────────────────────
    'notif.record_near_miss.title': 'Tu approches d\'un record',
    'notif.record_near_miss.body': 'Ton {metric_label} sur {period_label} approche de ton PB ({value} vs {target}).',
    'notif.milestone_unlocked.title': 'Jalon débloqué : {title_fr}',
    'notif.milestone_unlocked.body': 'Tu viens de débloquer « {title_fr} » — bravo !',
    'notif.milestone_near_miss.title': 'À deux doigts d\'un jalon',
    'notif.milestone_near_miss.body': 'Plus que quelques pas pour débloquer « {title_fr} ».',
    'notif.lusr_tier_approach.title': 'À {gap} pts de {next_tier_name}',
    'notif.lusr_tier_approach.body': 'Ton score LUSR approche du prochain sous-palier {next_tier_name}.',
    'notif.streak_milestone.title': 'Série de {length} jours !',
    'notif.streak_milestone.body': 'Tu atteins le palier {length} j — multiplicateur PP ×{multiplier}.',
    'notif.comeback_welcome.title': 'Bon retour parmi nous !',
    'notif.comeback_welcome.body': 'Tu as repris après {days_away} jours d\'absence — ton bouclier de série est prêt.',
    'notif.title_ready.title': '{title_name} est prêt',
    'notif.title_ready.body': 'Tes données {title_name} sont synchronisées — explore tes stats.',
    'notif.rival_encounter.title': 'Rival croisé',
    'notif.rival_encounter.body': 'Tu as recroisé {gamertag} : {kills} frags / {deaths} morts.',
  },

  relJustNow: 'à l’instant',
  relMinutesAgo: (n) => (n <= 1 ? 'il y a 1 min' : `il y a ${n} min`),
  relHoursAgo: (n) => (n <= 1 ? 'il y a 1 h' : `il y a ${n} h`),
  relDaysAgo: (n) => (n <= 1 ? 'il y a 1 j' : `il y a ${n} j`),
  relOnDate: (iso) => new Date(iso).toLocaleDateString('fr-FR'),
}

const EN: NotificationsText = {
  bellAriaLabelEmpty: 'No unread notifications',
  bellAriaLabelWithCount: '{count} unread notifications',

  dropdownTitle: 'Notifications',
  dropdownUnread: 'Unread',
  dropdownOlder: 'Recent',
  dropdownEmpty: "You're all caught up",
  dropdownMarkAllRead: 'Mark all as read',
  dropdownViewAll: 'View all',
  dropdownErrorLoading: 'Failed to load notifications',

  actionMarkAsRead: 'Mark as read',
  actionMarkAsUnread: 'Mark as unread',
  actionDismiss: 'Dismiss',
  actionView: 'View',

  pageTitle: 'Notifications',
  pageFilterAll: 'All',
  pageFilterUnread: 'Unread only',
  pageGroupToday: 'Today',
  pageGroupYesterday: 'Yesterday',
  pageGroupThisWeek: 'This week',
  pageGroupOlder: 'Older',
  pageBulkMarkRead: 'Mark as read ({count})',
  pageBulkDismiss: 'Dismiss ({count})',
  pageLoadMore: 'Load more',
  pageEmpty: 'No notifications yet',

  settingsTitle: 'Notifications',
  settingsDescription:
    'Manage in-app notifications, pop-ups and per-category subscriptions.',
  settingsMaster: 'Enable notifications',
  settingsMasterDescription:
    'Fully disables bell and pop-ups. Events still get persisted server-side.',
  settingsToasts: 'Enable pop-ups',
  settingsToastsDescription:
    'Show new notifications as on-screen pop-ups (in addition to the bell panel).',
  settingsCategoriesTitle: 'Categories',
  settingsCategoriesDescription:
    'Pick how you want to be notified for each event type.',
  settingsDeliveryBoth: 'Bell panel + pop-up',
  settingsDeliveryInApp: 'Bell panel only',
  settingsDeliveryToast: 'Pop-up only',
  settingsDeliveryOff: 'Off',
  settingsSubscriptionsTitle: 'Subscriptions',
  settingsSubscriptionsDescription:
    'Toggle automatic tracking for the current player.',
  settingsRetentionTitle: 'Retention',
  settingsRetentionLabel: 'Keep last {n} notifications',
  settingsTestButton: 'Send test notification',
  settingsTestSent: 'Test notification sent',

  categoryLabel: {
    app_release: 'App release',
    match_synced: 'Match synced',
    media_added: 'Media added',
    media_liked: 'Media liked',
    objective_assigned: 'New objective',
    objective_completed: 'Objective completed',
    challenge_added: 'New challenge',
    challenge_completed: 'Challenge completed',
    season_pass_level: 'Season pass level (deprecated)',
    sync_error: 'Sync error',
    personal_record: 'Personal record',
    threshold_crossed: 'Threshold crossed',
    friend_added: 'Friend added',
    friend_sync_completed: 'Friend sessions updated',
    data_health_warning: 'Database audit',
    career_rank: 'Career rank',
    skill_tier: 'CSR / LUSR tier',
    battlepass_completed: 'Battle pass completed',
    citation_tier: 'Commendation tier',
    citation_mastery: 'Commendation mastered',
    record_near_miss: 'Record near miss',
    milestone_unlocked: 'Milestone unlocked',
    milestone_near_miss: 'Milestone near miss',
    lusr_tier_approach: 'LUSR tier approach',
    streak_milestone: 'Streak milestone',
    comeback_welcome: 'Welcome back',
    trend_consolidate: 'Focus to consolidate',
    title_ready: 'Title ready',
    rival_encounter: 'Rival encountered',
  },
  categoryDescription: {
    app_release: 'A new LevelUp version is available.',
    match_synced: 'New matches were synchronized.',
    media_added: 'A video or screenshot was added.',
    media_liked: 'Someone liked one of your media.',
    objective_assigned: 'An objective was auto-assigned to you.',
    objective_completed: 'You completed an objective.',
    challenge_added: 'A new challenge is available.',
    challenge_completed: 'You completed a daily or weekly challenge.',
    season_pass_level: 'Legacy category superseded by "Career rank" and "Battle pass completed".',
    sync_error: 'The sync failed — manual retry recommended.',
    personal_record: 'You broke a personal record.',
    threshold_crossed: 'A K/D or winrate threshold was crossed.',
    friend_added: 'A gamertag was added to your friends list.',
    friend_sync_completed: 'Matches were reclassified as squad after a friend addition.',
    data_health_warning: 'Anomalies detected by the periodic DB audit (raw UUIDs, lying bits, stale URLs).',
    career_rank: 'You earned a new Halo career rank (lifetime).',
    skill_tier: 'Your competitive tier changed on a playlist (CSR or LUSR).',
    battlepass_completed: 'You completed a battle pass (max rank reached).',
    citation_tier: 'You crossed a new tier on a commendation.',
    citation_mastery: 'You mastered a commendation to 100%.',
    record_near_miss: 'Your current score is approaching one of your personal bests.',
    milestone_unlocked: 'You just unlocked a milestone (cumulative threshold).',
    milestone_near_miss: 'You are close to unlocking a milestone.',
    lusr_tier_approach: 'Your LUSR rating is approaching the next sub-tier.',
    streak_milestone: 'Your streak hit a milestone (PP multiplier).',
    comeback_welcome: 'You are back after a pause — welcome!',
    trend_consolidate: 'One of your performance areas has been trending down over time — a chance to shore it up.',
    title_ready: 'A newly activated title finished its first sync.',
    rival_encounter: 'A sync brought a new duel against one of your top rivals.',
  },

  metricLabel: {
    kd_ratio: 'K/D ratio',
    winrate: 'winrate',
    kda: 'KDA',
    // Progression V2 (sent by the coach generator under `metric` key).
    performance_score: 'performance score',
    kpm: 'kills per minute',
    accuracy: 'accuracy',
    pspm: 'personal score per minute',
  },

  periodLabel: {
    '30d': '30 days',
    '90d': '90 days',
    all_time: 'all-time',
  },

  templates: {
    'notif.app_release.title': 'New version: {version}',
    'notif.app_release.body': 'See what changed in the changelog.',
    'notif.match_synced.title': '{count} match(es) synced',
    'notif.match_synced.body': 'Your stats are up to date.',
    'notif.media_added.title': 'Media added by {actor_name}',
    'notif.media_added.body': '{count} file(s) linked to a match.',
    'notif.media_liked.title': '{actor_name} liked your media',
    'notif.media_liked.body': 'Check it out in the gallery.',
    'notif.objective_assigned.title': 'New objective',
    'notif.objective_assigned.body': '{count} new objective(s) assigned.',
    'notif.objective_completed.title': 'Objective completed',
    'notif.objective_completed.body': '{count} objective(s) completed — nice!',
    'notif.challenge_added.title': 'New challenge(s) available',
    'notif.challenge_added.body': '{count} new challenge(s).',
    'notif.challenge_completed.title': 'Challenge completed',
    'notif.challenge_completed.body': '{count} citation(s) earned.',
    'notif.season_pass_level.title': 'Level {level} reached',
    'notif.season_pass_level.body': 'You progress on the season pass.',
    'notif.sync_error.title': 'Sync error',
    'notif.sync_error.body': '{message}',
    'notif.personal_record.title': 'Personal record',
    'notif.personal_record.body': 'New record on {metric_label}: {value}.',
    'notif.threshold_crossed.title': 'Threshold crossed',
    'notif.threshold_crossed.body': 'You crossed a {metric_label} threshold: {value}.',
    'notif.trend_consolidate.title': 'Focus to consolidate',
    'notif.trend_consolidate.body': 'One of your performance areas has been dipping lately — a chance to shore it up.',
    'notif.friend_added.title': '{gamertag} added to your friends',
    'notif.friend_added.body': 'Shared sessions will be reclassified as squad in the background.',
    'notif.friend_sync_completed.title': 'Friend sessions updated',
    'notif.friend_sync_completed.body': '{promoted} match(es) reclassified as squad-friends.',
    'notif.test.title': 'Test notification',
    'notif.test.body': 'The notifications pipeline is working correctly.',
    'notif.data_health_warning.title': 'Database audit: {warnings_total} anomaly(ies) found',
    'notif.data_health_warning.body': '{uuids_raw} raw UUID(s), {lying_bits_events} lying bit(s), {garbage_banner_urls} stale URL(s). {hint}',
    'notif.career_rank.title': 'Rank {rank} reached',
    'notif.career_rank.body': 'You unlocked {rank_name} (from rank {previous}).',
    'notif.skill_tier.title': 'New tier {tier} ({rating_type})',
    'notif.skill_tier.body': 'Playlist {playlist_group} — {tier} {sub_tier} (was: {previous_tier} {previous_sub_tier}).',
    'notif.battlepass_completed.title': 'Battle pass completed',
    'notif.battlepass_completed.body': '{count} BP track(s) reached max rank — congrats!',
    'notif.citation_tier.title': 'New commendation tier',
    'notif.citation_tier.body': '{count} tier(s) crossed since last sync.',
    'notif.citation_mastery.title': 'Commendation mastered',
    'notif.citation_mastery.body': '{count} commendation(s) at 100% — nice!',
    // ─── Progression V2 (Ascension) — proactive coach ────────────────────
    'notif.record_near_miss.title': 'Approaching a record',
    'notif.record_near_miss.body': 'Your {metric_label} over {period_label} is approaching your PB ({value} vs {target}).',
    'notif.milestone_unlocked.title': 'Milestone unlocked: {title_en}',
    'notif.milestone_unlocked.body': 'You just unlocked « {title_en} » — congrats!',
    'notif.milestone_near_miss.title': 'Almost there on a milestone',
    'notif.milestone_near_miss.body': 'Just a few more steps to unlock « {title_en} ».',
    'notif.lusr_tier_approach.title': '{gap} pts from {next_tier_name}',
    'notif.lusr_tier_approach.body': 'Your LUSR rating is within reach of the next sub-tier {next_tier_name}.',
    'notif.streak_milestone.title': '{length}-day streak!',
    'notif.streak_milestone.body': 'You reached the {length}-day milestone — PP multiplier ×{multiplier}.',
    'notif.comeback_welcome.title': 'Welcome back!',
    'notif.comeback_welcome.body': 'You returned after {days_away} days away — your streak shield is ready.',
    'notif.title_ready.title': '{title_name} is ready',
    'notif.title_ready.body': 'Your {title_name} data is synced — explore your stats.',
    'notif.rival_encounter.title': 'Rival encountered',
    'notif.rival_encounter.body': 'You crossed paths with {gamertag} again: {kills} frags / {deaths} deaths.',
  },

  relJustNow: 'just now',
  relMinutesAgo: (n) => (n <= 1 ? '1 min ago' : `${n} min ago`),
  relHoursAgo: (n) => (n <= 1 ? '1 h ago' : `${n} h ago`),
  relDaysAgo: (n) => (n <= 1 ? '1 d ago' : `${n} d ago`),
  relOnDate: (iso) => new Date(iso).toLocaleDateString('en-US'),
}

const DICTS: Record<Locale, NotificationsText> = { fr: FR, en: EN }

export function getNotificationsText(locale: Locale): NotificationsText {
  return DICTS[locale] ?? FR
}
