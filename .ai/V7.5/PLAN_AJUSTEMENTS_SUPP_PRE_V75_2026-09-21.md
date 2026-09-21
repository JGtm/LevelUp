# Plan — Ajustements supplementaires pre-v7.5 (deuxieme vague, 2026-09-21)

Source : `C:\Users\Guillaume\Downloads\Ajustements-supp-prev75.txt` (retours utilisateur) + arbitrages
du 2026-09-21 (conversation). Branche de chantier : `feat/ajustements-supp-v75` (depuis
`origin/feat/v75` = 28b93cca5 + prop `frameless` de ChartCard b8166c9bf). Contrat : skill
`plan-execution`. Chaque lot = un worktree, un executeur Opus, une branche `feat/ajsup-<lot>`,
fusion par le pilote dans la branche de chantier. Statuts : `[x]` fait, `[~]` couvert ailleurs,
`[!]` non traite (justifie).

## Decisions utilisateur (tranchees, ne pas rouvrir)

- D1 « Prises nettes de drapeau » (Sessions) : retirer EXACTEMENT le texte cite (de « Prises nettes
  de drapeau » a « attribuees a un joueur ») ; le pilote lit : les phrases explicatives du bloc.
  Le titre de vue et la jauge du role « prendre » restent.
- D2 Tiers d'armes, partout : puissance en tete, puis terrain, puis un depliable ferme par defaut
  avec base. Les socles de BONUS (camouflage, surbouclier) sont des EQUIPEMENTS (decision du
  2026-09-21) : ils quittent la section armes, deja comptes dans « Usages d'equipement ». La ligne
  « non identifie » disparait des grilles (son compte passe dans l'infobulle (i) du titre).
- D3 Empaleur : ERREUR de classement constatee sur b1ad85eb (depart a loadout classique et egal
  pour tous) — corriger la cause AVANT tout reclassement.
- D4 « geste » -> « usages » partout dans l'UI.
- D5 « Les formes retenues » (Timeseries + Escouade) : supprimer le titre de section ET la rangee
  de tuiles de couverture ; les blocs Equipement / Armes speciales / Objectifs restent.
- D6 Sessions « Controle des armes speciales » : scinde en cartes dediees comme Synthese/Escouade.
- D7 Legende raye/plein : sur TOUTES les grilles de jauges et les deux pistes du lobby.
- D8 Etats vides : bloc dans une rangee = reste affiche avec un message nommant la cause (aucun
  film decode / aucun socle ni objectif dans les modes selectionnes / erreur de chargement) ;
  section entiere sans objet = masquee ; intertitre orphelin « Objectifs » corrige.
- D9 Colonne « Objets laches au sol » : detail par famille si le film le sert, sinon suppression.
- D10 Explorer « Il s'est passe quoi sur le bloc » : ignore.
- D11/D13/D14/D15 : MAQUETTE de 3 propositions chacune (pas de rond, pas de plat/mince) pour :
  Part des assistances, Repartition des resultats, Taux d'echange par session, Controle des armes
  speciales (Match view), Usages d'equipement (Match view : « par joueur » + « part de chaque
  equipe » cote a cote). Aucun code avant le verdict.
- D12 Titres Escouade : « Assistances croisees », « Delai de vengeance », « Frags non venges »,
  « Assistances par coequipier », « Part de mon camp », « Detail des prises d'armes speciales ».
- D16 Alignement comparaison : rangees partagees ; placeholder « Sans equivalent dans cette session ».
- D17 (2026-09-21, apres maquette) : Explorer « Part des assistances » = 1.A et « Repartition des resultats » = 2.A, TOUS DEUX A L HORIZONTALE. Sujets 3, 4, 5 : pas encore tranches.
- D18 (2026-09-21) : Match view « Controle des armes speciales » = 4.A, colonnes verticales empilees par socle, hauteur = prises reelles, echelle commune ; groupes PUISSANCE / TERRAIN separes par un trait vertical discret, nom du groupe + sous-total CENTRE EN HAUT DANS LE GRAPHIQUE entre l axe Y et le trait (pas de bandeau) ; base repliee.
- D19 (2026-09-21) : section « echange » de l Escouade = variante A de la maquette MAQUETTE_ECHANGE_ESCOUADE ; vocabulaire tactique : vengeance -> « Riposte », assistance -> « Appui », A APPLIQUER PARTOUT dans l app ; nom de la section : « Coordination » (valide), « taux d echange » -> « taux de riposte ».
- D20 (2026-09-21) : Match view « Part de chaque equipe » = 5.A : barres horizontales epaisses PAR FAMILLE (memes familles et meme ordre que la grille « Usages par joueur », inchangee), longueur = usages reels sur une echelle commune, segment mon camp / eux avec comptes ecrits, total en bout de ligne.
- D21 (2026-09-21, soir) : sections transverses — Riposte sur Sessions (jauges a parite, normalisation par mort de camp), Match view (par camp + par joueur, comptes pas taux) et Timeseries (frise) ; Portee des engagements sur Sessions (p10-p90 par classe vs habituel, reserve echantillon) et Escouade (angle narratif : roles de portee / records / duos — a choisir) ; Balance des degats cumulee sur Timeseries (front seul, lot L) ; Appui cote recu sur Sessions et Timeseries. Reserves a verifier sur pieces : taille d equipe par match ; appuis des coequipiers non suivis dans assist_pairs.

## Lots

### Lot A1 — chrome des graphes, Escouade, formes retenues (worktree `LevelUp-wt-ajsup-a1`) — FUSIONNE 57362a0d5
- [x] Six cartes de l'echange + donuts + tout ChartCard monte dans une SectionCard : `frameless`
- [x] Titres D12 (FR + EN)
- [x] Taux d'echange par session : rendu inchange (attend D13)
- [x] Frises : retirer « Affiches : les XX derniers matchs… » et « Hors de cette barre… »
- [x] Formes retenues (solo + squad) : titre de section + HeaderStrip retires (D5)
- [x] Notes sous legende des FormesCard -> InfoTooltip (i) a droite du titre de carte
- [x] Footers/notes des cartes de l'echange -> InfoTooltip (i)
- [x] Intertitre « Objectifs » orphelin corrige ; politique D8 sur les blocs des formes
- [x] Etiquettes blanches sur rayures (Piste100Form) — « Ce que mon camp prend du lobby »
- [x] « geste » -> « usages » dans `formes/cardsI18n.ts` et `formes/i18n.ts` ; « Bastion » -> « Bases »
- [x] Colonne « Objets laches au sol » (D9)

### Lot A2 — usage partage + Sessions (worktree `LevelUp-wt-ajsup-a2`) — FUSIONNE b62842439
- [x] Sessions : retirer « Matchs mesures X/X », pied du controle des armes, texte D1
- [x] Sessions : Regularite a droite de Parts et parites en pleine page ; legende unique
- [x] Legende raye/plein : composant unique pose sur UsageGaugeGrid et UsageLobbyTrack (D7)
- [x] Etiquettes blanches sur UsageLobbyTrack (verifier sur capture) ; epaisseurs homogenes
- [x] Controle des armes speciales (Sessions) scinde (D6)
- [x] Ordre des tiers D2 dans `usagePadTiersModel.ts` + depliable base + « non identifie » retire
- [x] Centrage vertical des grilles dans EquipmentUsageSection (Timeseries/Escouade/Synthese)
- [x] Donuts d'usage sortis de leur cadre (`frameless`)
- [x] Notes de EquipmentUsageSection -> InfoTooltip (i)
- [x] Etats vides D8 sur les blocs d'usage
- [x] « Bastion(s) » -> « Bases » dans `usageI18n.ts`

### Lot B — comparaison Sessions alignee (worktree `LevelUp-wt-ajsup-b`) — FUSIONNE 0fb899baa
- [x] Rangees partagees gauche/droite, placeholder D16

### Lot C — Explorer, Relations, Synthese, Timeseries intensite (worktree `LevelUp-wt-ajsup-c`) — FUSIONNE 95065fee3
- [x] Top medailles : dans Profil de combat, suit le switch En direct / Local ; retire de la carte cible
- [x] Noyau dur : plus de repli, tout affiche
- [x] Rythme des rencontres : cellules plus hautes (hauteur locale, liseret partage intact)
- [x] Synthese : sous-titre « Frag le plus lointain… » retire
- [x] Intensite : sous-titre et titre de panneau « Joueur » retires

### Lot D — Match view (worktree `LevelUp-wt-ajsup-d`) — FUSIONNE a9733a985
- [x] Occupation du terrain : +20 %, zoom et deplacement (hooks du rejeu 2D)
- [x] Notes sous legende -> InfoTooltip (i)
- [x] Usages d'equipement : deux cartes sur la meme rangee (rendu inchange, attend D15)
- [x] Ordre des tiers D2 dans `weaponTier.ts` / MatchPadControlSection ; « geste » -> « usages »

### Lot E — Empaleur (worktree `LevelUp-wt-ajsup-e`) — FUSIONNE d9be16612
- [x] Diagnostic sur b1ad85eb : pourquoi Skewer = base ; cause prouvee sur pieces
- [x] Correctif Go + jumeau TS + tests ; aucun reclassement heuristique

**Journal du lot E (2026-09-21).** Cause prouvee : le canal `loadouts` n est PAS publie au spawn
mais sur une grille d images-cles GLOBALE (b1ad85eb : 25 instants d emission, un toutes les 200
frames = 20 s, pour 73 vies ; ecart debut de vie -> premiere emission de 0 a 192 frames, mediane
60). Cinq vies sur 73 (6,85 %, au-dessus de `BaseShareMin = 0,05`) avaient leur premiere emission
APRES avoir ramasse un Empaleur (slots 547, 563, 595 par evenement de prise date ; 573, 594 par la
chaine des objets au sol). Correctif : une emission qui SUIT une prise d arme de la meme vie n est
plus lue comme equipement de depart, la vie quitte numerateur ET denominateur. Prises = union de
`pickups` (arme), `weaponChanges` (taken/swapped), `groundWeapons.picker`, filtree de la dotation
de reapparition. Effet : b1ad85eb 47 vies retenues, Empaleur 4,25 % -> hors base (MA40, Sidekick
restent base) ; corpus 92 artefacts, 34 matchs changent d ensemble de base, dans le bon sens.
`BaseShareMin` conserve (queue residuelle 4,94 %, plus faible vraie base 5,06 %). Ratchet de
surface `film/replay` 260 -> 263 justifie.

### Lot F — maquette (fichier `.ai/V7.5/MAQUETTE_RENDUS_AJSUP_2026-09-21.html`) — FUSIONNE bed1a2600
- [x] 5 sujets x 3 propositions, jetons de l'app, clair/sombre

### Lot G — suites A1/D (worktree `LevelUp-wt-ajsup-g`) — FUSIONNE 9c9d7d337
- [x] Helper canonique titre + infobulle (i) + garde-rail ; copies migrees hors `_shared/usage` et `session-detail`
- [x] `DroppedByFamily` expose dans le bloc formes (Go + openapi + types) ; colonne par famille lachee dans « Usages d equipement par match »
- [x] Doubles cadres restants : compare/CompareWeaponsRange, match-view/MatchKillDistanceSection, tactical/TacticalCoordinationCard
- [x] Ordre d affichage des tiers decouple de l ordre de departage (`tierOfWeaponOf`)

### Lot H — Explorer, rendus 1.A et 2.A horizontaux (worktree `LevelUp-wt-ajsup-h`) — FUSIONNE 1176357a6
- [x] Part des assistances : deux barres epaisses horizontales, echelle lineaire commune, tranches empilees avec comptes, trait de parite
- [x] Repartition des resultats : une barre epaisse horizontale empilee (compte + part ecrits), taux en chiffre d appel, bande des resultats

### Lot I — Match view, controle des armes en 4.A (worktree `LevelUp-wt-ajsup-i`) — FUSIONNE 1e93ea6ee
- [x] Colonnes empilees par socle, groupes separes par un trait, noms centres en haut dans le graphique, base repliee (D18)

### Lot J — Escouade, section Coordination en variante A + vocabulaire Riposte / Appui partout (worktree `LevelUp-wt-ajsup-j`) — FUSIONNE bd8f290c3
- [x] Carte « Riposte » : phrase de lecteur, chiffre d appel face a l habituel, frise batons + tendance par soiree (volumes dessous), histogramme des delais et matrice « Qui riposte pour qui » en replis
- [x] Carte « Appui » : le bloc des assistances de l escouade, renomme, phrase de lecteur
- [x] Nuage intact, place apres Riposte, phrase d introduction reecrite
- [x] Blocs Constat, Compte, Donne/recu, Taux par session supprimes (0 code mort)
- [x] Vocabulaire : Riposte / Appui / Coordination applique dans toute l app (Escouade, Match view, Tactique, Timeseries, Synthese, Accueil, i18n FR+EN)
- [x] Nuage : medianes dessinees par taille decroissante (petites au premier plan)

### Lot K — Match view, part de chaque equipe en 5.A (worktree `LevelUp-wt-ajsup-k`) — FUSIONNE 6c2391972
- [x] Barres horizontales par famille, echelle commune, deux camps, comptes ecrits, total (D20)

### Lot L — Timeseries, balance des degats cumulee (worktree `LevelUp-wt-ajsup-l`) — FUSIONNE 0da57c5b1
- [x] Aire divergente ancree a zero, meme grammaire que l Escouade, front seul

### Lot M — maquette sections transverses : Riposte, Portee, Appui (worktree `LevelUp-wt-ajsup-m`)
- [ ] Fichier `.ai/V7.5/MAQUETTE_TRANSVERSES_RIPOSTE_PORTEE_APPUI_2026-09-21.html` + verification des deux reserves de donnees

## Decouvertes (hors perimetre, ne pas traiter)
- J : `squad.toml` 998 L (dette) ; anglicismes FR dans `coaching_tips.toml` (revenge push, trader, aim assist) ; chaines de la statistique de jeu « assistances » laissees (liste dans le rapport du lot) ; `colorDistance.guard.test.ts` flake en suite complete (vert seul).
- I : `BarStackedChart.tsx` a 497 L brutes ; bande de trace des titres de groupe estimee (10-97 %), fragile si un appelant nomme ses axes avec des groupes ; deux chemins d encre (DOM color-mix / canvas opacite) sans garde-rail.
- K : une famille peut avoir une colonne de zeros dans la grille et aucune piste a droite (famille sans usage = ligne absente) -> verdict visuel.
- H : `AssistExchangeSummary` (papillon log) reste sur le hub Relations (`RelationAssistsCards.tsx:45`) -> verdict utilisateur ; `assist_volume_max` n a plus de lecteur Explorer ; `ExplorerTargetSampleStats.tsx` a 399 L.
- G : `explorer/ExplorerTargetFragRange.tsx:118` reecrit le chrome de SectionCard a la main ; alias courts de `deployedFamilyLabel` (`usageI18n.ts:507`) ne matchent aucune cle ; le hook go-vet de lefthook tourne sans CGO (60 lignes de bruit par commit).
- G : libelle des colonnes de lacher = « Laches : Surbouclier » (forme neutre, genres mixtes des familles) -> verdict utilisateur.
- E : la table persistee `match_pad_tiers` porte encore l ancienne regle (34 matchs sur 92 avec un ensemble de base faux dans les agregats Sessions/Escouade/Timeseries ; la Match view recalcule a la requete). Rattrapage existant `levelup backfill-pad-tiers` (rejoue `ProjeterNiveauxDArmes` sans recuisson) — DECISION utilisateur 2026-09-21 : OUI, a lancer en local A LA FIN du chantier (apres fusion de G et H) ; prevenir avant de le lancer en prod.
- E : neuf vies de b1ad85eb non nommees (`index_hors_table`), dont 573 et 594 : leurs prises ne sont attribuees a personne (queue residuelle sous le seuil).
- A2 : la vue « Prises nettes de drapeau » ne contient plus que son titre (D1 applique a la lettre) -> verdict utilisateur.
- A2 : `usage.powerup_pickups` servi par Go et plus lu par le web ; `usageI18n.ts` 562 L (dette reduite, non resorbee) ; `equipRift` cle morte.
- A2 : `usageCardTitle.tsx` = un second helper titre + infobulle, a reconcilier avec celui du lot G.
- D : `PAD_TIER_ORDER` sert aussi au departage de `tierOfWeaponOf` (`padControlLogic.ts:236`) : ordre d affichage et ordre de departage a decoupler (lot G).
- A1 : double cadre aussi dans `compare/CompareWeaponsRange.tsx`, `match-view/MatchKillDistanceSection.tsx`, `tactical/TacticalCoordinationCard.tsx`.
- A1 : motif titleAdornment + InfoTooltip recopie dans ~8 fichiers -> helper canonique + garde-rail (lot G du chantier).
- A1 : `DroppedByFamily` existe en amont (`film/replay/usage_summary.go:99`, `analysis/sessionusage`) mais s arrete a `analysis/squadformes/formes.go:229` (scalaire `Dropped`) : exposer la ventilation rendrait la colonne par famille (lot G).
- A1 : etiquettes sur rayures en `text-foreground` (pas blanc) car la hachure adverse laisse voir le fond de carte -> verdict visuel utilisateur.
