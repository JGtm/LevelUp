package replay

//
// us2 (2026-09-06) : l'attribution passe du SLOT (« dernier gagnant ») à la VIE qui couvre
// l'instant du geste. Sur un slot recyclé, les tractions, épisodes et poses du premier
// occupant lui reviennent au lieu d'être créditées au second (constat C5 de la revue REG-R1).
// Tout résumé produit sous `us1` doit donc être refait, et cette montée est ce qui le déclenche.
// us3 (2026-09-07) — UNE VIE DONT LE NOMMAGE A ECHOUE OCCUPE QUAND MEME SON SLOT. Elle entre
// desormais dans la table par vie avec un xuid VIDE : elle n'ouvre aucune ligne (une ligne est
// keyee par xuid) mais elle rend l'instant NON ATTRIBUABLE au lieu de le laisser retomber sur
// `dernier[slot]`, le dernier occupant du match. Sur un slot recycle, la ligne d'un joueur
// recevait donc un geste qui n'est pas le sien — precisement la regle que le correctif us2
// declarait avoir supprimee. C'est deja le traitement des vies de BOT, pour la meme raison.
// Tout resume produit sous `us2` est donc a refaire sur un film a slot recycle.
// us4 (2026-09-09) — LES TROIS ISSUES D'UN OBJET PRIS entrent dans le resume (etape E3 du
// PLAN_EQUIPEMENT_GACHIS). La projection lit desormais `equipmentChanges` (canal jusqu'ici
// ignore de tout ecran d'usage) et publie quatre ventilations par famille : les prises
// (`taken`), les consommations (`spent`), les lachers (`dropped` — deja calcule en memoire,
// jamais persiste avant) et le GARDE SANS L'UTILISER, derive par `max(0, taken - utilise -
// lache)`. Sans cette montee, aucun match deja resume ne serait re-projete et les quatre
// colonnes neuves resteraient vides sur tout le corpus : la cle de reprise du backfill est
// (summary_rev, artifact_schema), pas la presence des colonnes.
// us5 (2026-09-10, lot 4.3) — LA JOINTURE RANG -> FAMILLE DU BILAN D'EQUIPEMENT ne passe
// plus par la RACINE DU LIBELLE : elle lit `abilityLabels[].family`, publiee par le
// document depuis le schema 51. La regle d'attribution change donc bel et bien — un
// artefact au schema 50 ne porte aucune famille, et son resume ne peut pas etre re-projete
// sans RECUISSON DE L'ARTEFACT lui-meme. (Cette entree vivait dans usage_summary.go
// jusqu'au lot 5.5 : la chronique n'a qu'un seul foyer, et c'est ce fichier.)
// us6 (2026-09-10, lot 5.5) — LE COTE « UTILISE » D'UN DEPLOYABLE SANS PIECE ENGENDREE
// (capteur, traqueur, ecran occultant, champ de reparation, balise du translocateur) se lit
// desormais sur ses CONSOMMATIONS DE CHARGE (`spent`) et non plus sur ses poses `deployed`.
// Une pose `deployed` sur un objet PORTE mesure un LACHER VOLONTAIRE a mi-vie, pas un
// deploiement : sur les 202 consommations de ces familles au parc, ZERO n'est couverte par
// une pose du meme joueur a moins de 2 s (le mur, seul equipement qui ENGENDRE UNE PIECE,
// est a 84 % — rapport E0 du 2026-09-10, question 5). Le mur garde donc ses poses de
// panneau et les deux bonus leurs episodes. AUCUNE RECUISSON D'ARTEFACT : `spent` et sa
// famille sont publies depuis le schema 51 — la seule bascule de revision suffit a faire
// reprendre les resumes par `levelup backfill-usage-summary` (sans `--force`).
