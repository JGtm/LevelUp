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
- D2 Tiers d'armes, partout : puissance en tete, puis bonus (camouflage, surbouclier — socles de
  bonus, game-changers), puis terrain, puis un depliable ferme par defaut avec base. La ligne
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

## Lots

### Lot A1 — chrome des graphes, Escouade, formes retenues (worktree `LevelUp-wt-ajsup-a1`)
- [ ] Six cartes de l'echange + donuts + tout ChartCard monte dans une SectionCard : `frameless`
- [ ] Titres D12 (FR + EN)
- [ ] Taux d'echange par session : rendu inchange (attend D13)
- [ ] Frises : retirer « Affiches : les XX derniers matchs… » et « Hors de cette barre… »
- [ ] Formes retenues (solo + squad) : titre de section + HeaderStrip retires (D5)
- [ ] Notes sous legende des FormesCard -> InfoTooltip (i) a droite du titre de carte
- [ ] Footers/notes des cartes de l'echange -> InfoTooltip (i)
- [ ] Intertitre « Objectifs » orphelin corrige ; politique D8 sur les blocs des formes
- [ ] Etiquettes blanches sur rayures (Piste100Form) — « Ce que mon camp prend du lobby »
- [ ] « geste » -> « usages » dans `formes/cardsI18n.ts` et `formes/i18n.ts` ; « Bastion » -> « Bases »
- [ ] Colonne « Objets laches au sol » (D9)

### Lot A2 — usage partage + Sessions (worktree `LevelUp-wt-ajsup-a2`)
- [ ] Sessions : retirer « Matchs mesures X/X », pied du controle des armes, texte D1
- [ ] Sessions : Regularite a droite de Parts et parites en pleine page ; legende unique
- [ ] Legende raye/plein : composant unique pose sur UsageGaugeGrid et UsageLobbyTrack (D7)
- [ ] Etiquettes blanches sur UsageLobbyTrack (verifier sur capture) ; epaisseurs homogenes
- [ ] Controle des armes speciales (Sessions) scinde (D6)
- [ ] Ordre des tiers D2 dans `usagePadTiersModel.ts` + depliable base + « non identifie » retire
- [ ] Centrage vertical des grilles dans EquipmentUsageSection (Timeseries/Escouade/Synthese)
- [ ] Donuts d'usage sortis de leur cadre (`frameless`)
- [ ] Notes de EquipmentUsageSection -> InfoTooltip (i)
- [ ] Etats vides D8 sur les blocs d'usage
- [ ] « Bastion(s) » -> « Bases » dans `usageI18n.ts`

### Lot B — comparaison Sessions alignee (worktree `LevelUp-wt-ajsup-b`)
- [ ] Rangees partagees gauche/droite, placeholder D16

### Lot C — Explorer, Relations, Synthese, Timeseries intensite (worktree `LevelUp-wt-ajsup-c`)
- [ ] Top medailles : dans Profil de combat, suit le switch En direct / Local ; retire de la carte cible
- [ ] Noyau dur : plus de repli, tout affiche
- [ ] Rythme des rencontres : cellules plus hautes (hauteur locale, liseret partage intact)
- [ ] Synthese : sous-titre « Frag le plus lointain… » retire
- [ ] Intensite : sous-titre et titre de panneau « Joueur » retires

### Lot D — Match view (worktree `LevelUp-wt-ajsup-d`)
- [ ] Occupation du terrain : +20 %, zoom et deplacement (hooks du rejeu 2D)
- [ ] Notes sous legende -> InfoTooltip (i)
- [ ] Usages d'equipement : deux cartes sur la meme rangee (rendu inchange, attend D15)
- [ ] Ordre des tiers D2 dans `weaponTier.ts` / MatchPadControlSection ; « geste » -> « usages »

### Lot E — Empaleur (worktree `LevelUp-wt-ajsup-e`)
- [x] Diagnostic sur b1ad85eb : pourquoi Skewer = base ; cause prouvee sur pieces
- [x] Correctif Go + jumeau TS + tests ; aucun reclassement heuristique

**Journal du lot E (2026-09-21).** Cause prouvee : le canal `loadouts` n'est PAS publie au
spawn mais sur une grille d'images-cles GLOBALE — sur `b1ad85eb`, 25 instants d'emission
(t = 12, 212, 412 … 5014, un toutes les 200 frames = 20 s) pour 73 vies, l'ecart entre le debut
d'une vie et sa premiere emission allant de 0 a 192 frames (19,2 s ; mediane 60). Cinq vies sur
73 (6,85 %, au-dessus de `BaseShareMin = 0,05`) avaient donc leur premiere emission APRES avoir
ramasse un Empaleur ; trois sont attestees par un evenement de prise date (slots 547, 563, 595),
les deux autres par la chaine des objets au sol (slots 573 et 594, vies non nommees, qui LACHENT
un Empaleur a leur mort). Correctif : une emission qui SUIT une prise d'arme de la meme vie
n'est plus lue comme un equipement de depart, et la vie quitte alors numerateur ET denominateur.
Les prises sont l'union de trois canaux (`pickups` nature arme, `weaponChanges` taken/swapped,
`groundWeapons.picker`), filtree de la dotation de reapparition (prise datee du debut de la vie).
Effet mesure : b1ad85eb 47 vies retenues, Empaleur 4,25 % -> plus « base » (verifie par le code
Go sur l'artefact reel : `base` avant, hors-base apres ; MA40 et Sidekick restent base) ; parc de
92 artefacts, 6 385 vies retenues sur 7 844, 34 matchs changent d'ensemble d'armes de base.
`BaseShareMin` CONSERVE (justification datee dans le code) : apres correctif la queue plafonne a
4,94 % et la plus faible vraie arme de base tient 5,06 %. Ratchet de surface `film/replay`
260 -> 263 (trois natures de prise nommees), justifie et date dans le test.

### Lot F — maquette (fichier `.ai/V7.5/MAQUETTE_RENDUS_AJSUP_2026-09-21.html`)
- [ ] 5 sujets x 3 propositions, jetons de l'app, clair/sombre

## Decouvertes (hors perimetre, ne pas traiter)

- **Lot E — la table `match_pad_tiers` porte encore l'ancienne regle.** La vue Match recalcule
  les niveaux A LA REQUETE depuis l'artefact : elle est corrigee des le deploiement. Les pages
  d'AGREGAT (Sessions, Escouade, Timeseries) lisent la projection persistee, ecrite avec
  l'ancienne regle : 34 des 92 matchs rangés y portent un ensemble d'armes de base faux. Le
  rattrapage existe deja (`levelup backfill-pad-tiers`, il rejoue EXACTEMENT
  `ProjeterNiveauxDArmes` sur les artefacts deja ranges, sans recuisson) — A DEMANDER a
  l'utilisateur, non lance.
- **Lot E — neuf vies de `b1ad85eb` ne sont pas nommees** (`index_hors_table`), dont les deux
  qu'aucun canal de prise ne couvre (slots 573 et 594). Sans nom, leurs prises de socle ne sont
  de toute facon attribuees a personne ; c'est ce qui laisse une queue residuelle sous le seuil.
