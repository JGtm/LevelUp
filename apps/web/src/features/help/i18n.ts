export type HelpLocale = 'fr' | 'en'

export type HelpTab = 'glossary' | 'release-notes'

export interface GlossaryEntry {
  term: string
  definition: string
  formula?: string
  example?: string
}

export interface GlossarySection {
  title: string
  entries: GlossaryEntry[]
}

export interface HelpText {
  tabs: {
    glossary: string
    releaseNotes: string
  }
  page: {
    title: string
    subtitle: string
  }
  releaseNotes: {
    loading: string
    error: string
    title: string
  }
  glossary: {
    title: string
    sections: GlossarySection[]
  }
}

const FR_TEXT: HelpText = {
  tabs: {
    glossary: 'Glossaire & Concepts',
    releaseNotes: 'Notes de version',
  },
  page: {
    title: 'Aide',
    subtitle: 'Documentation et concepts de LevelUp',
  },
  releaseNotes: {
    loading: 'Chargement des notes de version…',
    error: 'Impossible de charger les notes de version.',
    title: 'Notes de version',
  },
  glossary: {
    title: 'Glossaire & Concepts',
    sections: [
      {
        title: 'Métriques de performance',
        entries: [
          {
            term: 'LUSR',
            definition:
              "Indice de compétence fondé sur TrueSkill 2 (Microsoft Research), adapté à Halo Infinite. C'est l'équivalent du CSR (rang classé officiel) pour les modes non classés : là où le CSR est attribué par Halo pour les listes de jeu compétitives, le LUSR est calculé localement par LevelUp sur tous les autres modes (arène, grande équipe, Firefight, parties personnalisées…), en appliquant la même logique TrueSkill.\n\nChaque joueur est représenté par deux nombres : sa moyenne estimée μ (talent supposé) et son incertitude σ (à quel point on est sûr de cette estimation). L'affichage public est μ − 3σ : on retire volontairement l'incertitude pour éviter de gonfler le rang d'un joueur dont on connaît mal le niveau.\n\nLe calcul est séparé par famille de listes de jeu (compétitif, arène, grande équipe, Firefight, parties personnalisées). Une famille = un état μ/σ indépendant, pour qu'une mauvaise série en social ne pollue pas votre niveau classé.\n\nÀ chaque partie, LevelUp calcule un score composite [0 ; 1] qui mesure votre performance « objective », puis l'injecte dans la formule TrueSkill pour mettre à jour μ et σ. Le score composite agrège 8 mesures (voir le tableau dans la formule) avec deux particularités importantes :\n\n• Cinq de ces mesures (précision, efficacité de dégâts, exploit médailles, conversion offensive, résistance défensive) sont comparées à votre **propre moyenne glissante sur les 50 dernières parties** — et non à une référence absolue. Le score s'ajuste donc à votre niveau actuel : progresser devient plus difficile à mesure que vous progressez.\n\n• Le niveau adverse est estimé via le ratio des « kills attendus » (mesure de niveau retournée par l'API Halo) entre les deux équipes, plafonné dans [0,5 × ; 2,0 ×] votre μ. Battre une équipe plus forte vous fait monter davantage qu'écraser une équipe faible.\n\nRésultats pris en compte : Victoire (poids 1,0), Égalité (0,5), Défaite (0,0). Les abandons comptent faiblement (0,15) pour ne pas vous pénaliser sur une déconnexion subie sans pour autant les ignorer totalement.\n\nDérive d'inactivité : au-delà d'une journée sans jouer dans la famille concernée, σ remonte d'1 point par jour (plafonné à 14 jours). Conséquence : la partie suivante a plus d'impact sur μ (positif comme négatif), pour rattraper l'éventuelle évolution de votre vrai niveau pendant la pause.",
            formula:
              'LUSR affiché = μ − 3σ\n\nConstantes du moteur TrueSkill :\n• μ initial : 1 500 · σ initial : 350\n• σ minimum : 60 · σ maximum : 350\n• β (variabilité par partie) : 200 · τ (dynamique du talent) : 25\n• K (Elo) : 32 · probabilité d\'égalité : 0,06\n• Dérive d\'inactivité : 1 point/jour au-delà d\'1 jour, plafonné à 14 jours\n• Force adverse : μ adversaire = μ × clamp(KE_adverse / KE_équipe, 0,5 ; 2,0)\n  (KE = kills attendus, mesure de niveau renvoyée par l\'API Halo)\n• σ adversaire : 150 (constant)\n\nScore composite = Σ (composante × poids) / Σ (poids actifs), borné [0 ; 1]\nSi une composante est absente sur la partie, son poids est exclu du dénominateur.\n\nTableau des 8 composantes (somme = 1,02, renormalisée automatiquement) :\n\nPoids  Composante                Calcul                              Référence\n─────  ────────────────────────  ──────────────────────────────────  ─────────────────────\n0,27   Frags vs attendus         frags / kills_expected (API)        sigmoïde autour de 1\n0,24   Morts vs attendues        deaths_expected (API) / morts       sigmoïde autour de 1\n0,16   Conversion offensive      225 × (frags + ass/3) / dégâts      moyenne 50 parties (P80=0,83)\n0,10   Efficacité de dégâts      dégâts infligés / total dégâts      moyenne 50 parties\n0,10   Écart de précision        précision − moyenne 50 parties      0,5 + delta × 2, borné [0;1]\n0,06   Résistance défensive      dégâts reçus / (225 × morts)        moyenne 50 parties (P80=1,59)\n0,05   Facteur de victoire       Victoire 1,0 · Égalité 0,5 ·        — (valeur fixe)\n                                 Défaite 0,0 · Abandon 0,15\n0,04   Exploit médailles         score d\'exploit pondéré            moyenne 50 parties (défaut 5,0)\n\nMise à jour TrueSkill (post-calcul du composite) :\nΔμ = K × (composite − 0,5)\nΔσ ajustée selon la formule TrueSkill (atténuation v(t), w(t) avec marge d\'égalité)\npuis σ recombinée avec τ pour réinjecter une part de dynamique.\n\nFamilles de listes de jeu (états μ/σ séparés) :\nranked · ranked challenge · big_team_battle · firefight · custom · arena (défaut)',
            example:
              "Un joueur avec μ = 1 700 et σ = 80 affiche un LUSR de 1 460 (= 1 700 − 3 × 80).\n\nAprès deux semaines sans jouer, σ remonte d'environ 13 points (1 point par jour, plafonné à 14 jours) pour atteindre 93. Le LUSR affiché tombe alors à 1 421 (= 1 700 − 3 × 93) sans qu'aucune partie n'ait été jouée — l'incertitude grimpe parce que LevelUp ne sait plus aussi bien où se situe le joueur.\n\nLa partie de retour aura plus d'impact qu'une partie habituelle (positif comme négatif), pour recaler μ vers son nouveau niveau réel.",
          },
          {
            term: 'Score de performance',
            definition:
              "Score relatif de 0 à 100 qui mesure votre contribution sur une partie par rapport à votre propre historique. Calculé à partir de 13 mesures pondérées — chacune est convertie en rang percentile sur vos parties passées (votre place dans votre propre distribution).\n\nLe principe : pour chaque mesure, on regarde quelle proportion de vos parties précédentes affichaient une valeur inférieure ou égale à la vôtre. Une partie où vous obtenez votre meilleur score depuis 100 parties donne un percentile proche de 100 ; une partie médiane donne un percentile autour de 50. Le score final est la moyenne pondérée de ces 13 percentiles, normalisée entre 0 et 100.\n\nÉvolution dans le temps — le score n'est pas figé. Pour chaque partie, l'historique de référence est constitué de **toutes vos parties précédentes** (jusqu'à la veille de celle qu'on évalue). Plus vous jouez, plus l'échantillon de comparaison s'agrandit et se précise.\n\nÉvolution avec votre niveau — comme la référence est votre propre historique, le score s'auto-recalibre quand vous progressez. Une partie « excellente » d'il y a un an peut devenir simplement « correcte » comparée à votre niveau actuel : si vos performances montent, le seuil pour décrocher un percentile élevé monte avec elles. Conséquence directe : un débutant en progression rapide voit ses scores stagner ou baisser même quand il joue mieux en valeur absolue, parce que sa propre distribution se déplace plus vite que ses gains. À l'inverse, un joueur expérimenté dont le niveau plafonne aura des scores plus stables.\n\nRecalcul rétroactif — chaque score affiché correspond à l'historique tel qu'il était au moment où la partie a été jouée. Cependant, lors d'une synchronisation avec recalcul activé (option « rafraîchir les scores de performance »), toute la série est recalculée depuis le début avec l'historique à jour. Une partie passée peut donc voir son score bouger légèrement après un tel recalcul, surtout si elle se trouvait près du seuil des 10 parties d'activation.\n\nLes mesures « inversées » (morts par minute) sont traitées dans le sens contraire : moins vous mourez, plus le percentile est élevé. Les mesures optionnelles (précision, dégâts, médailles, etc.) ne sont prises en compte que si la donnée est disponible — leur poids est exclu du dénominateur sinon, ce qui évite de pénaliser un mode sans précision tracée.\n\nIl faut au moins 10 parties dans l'historique pour activer le score. En dessous de ce seuil, un repli simple sur le rang percentile du seul ratio FDA est utilisé.\n\nRésistance aux robots : si des robots étaient présents dans votre équipe, un bonus correctif est appliqué après le calcul. Il dépend de l'écart de niveau estimé entre les deux équipes (delta des MMR rapporté à 30 %, plafonné à ±1). Sur une victoire le bonus vaut au minimum 0,5 point et au maximum 5 ; sur une défaite il monte jusqu'à 7. Cette correction reflète le fait qu'un robot coéquipier alourdit votre équipe — il serait injuste de pénaliser votre percentile pour cela.",
            formula:
              'Étape 1 — normalisation par minute pour les mesures de volume.\nÉtape 2 — calcul du rang percentile pour chaque mesure sur votre historique.\nÉtape 3 — moyenne pondérée des percentiles :\nScore brut = Σ(percentile × poids) / Σ(poids actifs)\nÉtape 4 — bornage [0 ; 100] et arrondi à 0,1.\n\nSi une mesure optionnelle est absente, son poids n\'entre pas dans le dénominateur.\n\nTableau des 13 mesures (somme = 1,01, renormalisée automatiquement) :\n\nPoids  Mesure                          Sens       Type\n─────  ──────────────────────────────  ─────────  ──────────\n0,14   Frags par minute                normal     requise\n0,11   Ratio FDA                       normal     requise\n0,10   Morts par minute                inversé    requise\n0,10   Score personnel par minute      normal     optionnelle\n0,09   Frags / frags attendus          normal     optionnelle\n0,09   Rendement offensif              normal     optionnelle\n0,07   Morts attendues / morts         normal     optionnelle\n0,06   Dégâts par minute               normal     optionnelle\n0,06   Score d\'exploit médailles       normal     optionnelle\n0,06   Assistances par minute          normal     requise\n0,05   Résistance défensive            normal     optionnelle\n0,04   Précision                       normal     optionnelle\n0,04   Rang final vs rang attendu      normal     optionnelle\n\nSens « inversé » = moins c\'est élevé, mieux c\'est (rang percentile inversé).\n\nBonus robot coéquipier (appliqué après le calcul) :\nfacteur = ((MMR_adverse − MMR_équipe) / MMR_équipe) / 0,30, borné [−1 ; 1]\nVictoire : bonus = max(0,5 ; 3,0 + facteur × 2,0)\nDéfaite : bonus = 5,0 + facteur × 2,0\nBonus ajouté avant le bornage final.',
            example:
              "Vous avez 200 parties dans l'historique. Sur la partie courante :\n• Frags par minute : meilleur que 78 % de vos parties → percentile 78 (poids 0,14)\n• Ratio FDA : meilleur que 65 % → percentile 65 (poids 0,11)\n• Morts par minute (inversé) : vous êtes mort moins souvent que dans 82 % de vos parties → percentile 82 (poids 0,10)\n• Précision absente sur ce mode → mesure ignorée, son poids 0,04 est retiré du dénominateur\n• … (les 9 autres mesures suivent le même calcul)\n\nMoyenne pondérée des percentiles ≈ 72 → score affiché 72/100.\nLecture : sur ces 12 mesures, vous étiez en moyenne dans les 28 % les plus performants de votre propre historique.",
          },
          {
            term: 'Rendement offensif',
            definition:
              "Efficacité offensive : mesure la quantité de dégâts nécessaire pour convertir une élimination. Le coefficient 225 repose sur le postulat qu'un Spartan possède 225 points de vie au total (90 de vie de base et 135 de bouclier), convention officielle de Halo Infinite.\n\nAu-dessus de 1,0 : vous éliminez avec moins de dégâts qu'attendu (précision, tirs à la tête qui ignorent le bouclier).\nEn dessous de 1,0 : vous gaspillez des dégâts (assistances non converties, suivi insuffisant).",
            formula:
              'Rendement offensif = 225 × (éliminations + assistances/3) / dégâts infligés\n\nP80 de référence (données réelles) : 0,83\n(soit 83 % des parties ont un rendement ≤ 0,83)',
            example:
              "10 éliminations, 6 assistances, 2 800 dégâts → 225 × (10 + 2) / 2 800 ≈ 0,96 : au-dessus du P80, vous convertissez efficacement.\n6 éliminations, 2 assistances, 2 800 dégâts → 225 × (6 + 0,67) / 2 800 ≈ 0,54 : beaucoup de dégâts pour peu d'éliminations.",
          },
          {
            term: 'Résistance défensive',
            definition:
              "Mesure la quantité de dégâts que vous absorbez par mort. Repose sur le même postulat : un Spartan possède 225 points de vie (90 de base et 135 de bouclier).\n\nAu-dessus de 1,0 : vous mourez après avoir encaissé plus qu'une vie de Spartan (bonne résistance, vous obligez les adversaires à vider leur chargeur).\nEn dessous de 1,0 : vous mourez rapidement, souvent surpris ou mal positionnés.",
            formula:
              'Résistance défensive = dégâts reçus / (225 × morts)\n\nP80 de référence (données réelles) : 1,59',
            example:
              "5 morts, 1 400 dégâts reçus → 1 400 / (225 × 5) ≈ 1,24 : au-dessus de 1,0, vous absorbez en moyenne 1,24 fois la vie d'un Spartan avant de mourir.\n5 morts, 750 dégâts reçus → 750 / 1 125 ≈ 0,67 : vous mourez tôt dans les échanges.",
          },
          {
            term: 'Ratio FDA',
            definition:
              "Ratio Frags / Décès / Assistances retourné directement par l'API Halo. C'est la mesure « tout-en-un » que Halo affiche pour évaluer un joueur sur une partie : on additionne les frags et un tiers des assistances, puis on divise par le nombre de décès (avec un plancher d'un décès pour éviter la division par zéro).\n\nLa lettre D du sigle vient de « Décès » mais Halo affiche le mot « Morts » dans son interface — les deux désignent la même chose. Le poids d'un tiers pour une assistance reflète le fait qu'aider à abattre un adversaire vaut moins que le tuer soi-même.\n\nMesure de référence mais sensible au volume : un joueur très actif (beaucoup de frags et de morts) peut afficher le même ratio FDA qu'un joueur passif (peu d'engagements). Le score de performance complète cette lecture en intégrant 12 autres mesures.",
            formula: 'FDA = (frags + assistances / 3) / max(1, morts)',
            example:
              "15 frags, 4 assistances, 6 morts → FDA = (15 + 4/3) / 6 ≈ 2,72\n20 frags, 0 assistance, 5 morts → FDA = 20 / 5 = 4,00\n0 frag, 0 assistance, 0 mort → FDA = 0 / 1 = 0 (plancher d'une mort)",
          },
          {
            term: "Score d'engagement",
            definition:
              "Mesure votre dynamisme dans un match : écart entre votre engagement observé et votre engagement attendu (vu votre style historique et le contexte d'équipe), normalisé en percentile vs votre propre historique. 50 = comme d'habitude. >50 = plus engagé. <50 = moins engagé.\n\nDistinct du Score de performance : le PerfScore mesure la qualité de vos actions, l'Engagement mesure leur quantité, leur régularité et votre état d'esprit. Les deux axes sont indépendants — un joueur peut être compétent mais atone (perf élevée, engagement bas) ou moyen mais investi à fond (perf normale, engagement haut).\n\nCalcul intra-match : pour chaque instant t (échantillonné toutes les 10s, fenêtre glissante 90s), on compare votre pace d'événements (kills, deaths, assists, médailles) au pace de votre équipe alliée multiplié par votre coefficient personnel. La moyenne des écarts sur le match donne le résidu brut, qui est ensuite normalisé en percentile vs vos 200 derniers matchs sur la même catégorie de mode.\n\nNiveaux de confiance : insufficient_history (<10 matchs, score nil), partial (10-29 matchs), full (≥30 matchs). PvE non couvert v1.",
            formula:
              "Pour chaque instant t :\n  pace_attendu(t) = coef_team_share × pace_team_per_player(t)\n  écart(t) = pace_joueur(t) − pace_attendu(t)\n\nRésidu brut du match = mean_t écart(t)\nScore final = percentile(résidu brut, distribution historique 200 matchs)\n\nFenêtre glissante : 90s, échantillonnage : 10s.\nFallback FFA/1v1 : pace_attendu calculé via lobby_share au lieu de team_share.\nMin match duration : 3 minutes (sinon score = null).",
            example:
              "Vous prenez habituellement 28% du travail dans vos équipes (coef_team_share = 1.12 dans un 4v4). Sur ce match, votre équipe alliée fait 12 events/min/joueur en moyenne. Votre pace attendu = 1.12 × 12 = 13.4 events/min. Si vous faites 16 events/min, vous êtes au-dessus de l'attendu de 2.6. Sur l'ensemble du match, écart moyen = +1.8. Vs votre historique (médiane résidus = 0), c'est meilleur que 78% de vos parties → Engagement Score = 78/100.",
          },
          {
            term: "Coefficient d'engagement (team_share, lobby_share)",
            definition:
              "Caractérisent votre style de jeu sur la durée. Médianes glissantes sur vos 200 derniers matchs par catégorie de mode (PvP_ranked, PvP_unranked).\n\nCoef_team_share : votre part historique du travail de vos équipes. Comparable inter-joueurs (caractéristique personnelle pure indépendante du contexte). >1.0 = leader intra-équipe. <1.0 = profil support / passif.\n\nCoef_lobby_share : votre part historique de l'action totale du lobby. Mixe style + skill + qualité des équipes habituelles.\n\nLecture combinée :\n• team=1.3 / lobby=1.0 : leader d'équipes moyennes (vous portez)\n• team=1.0 / lobby=1.4 : passager d'équipes fortes (vous bénéficiez du contexte)\n• team=0.7 / lobby=1.1 : sous-engagé en équipe forte (l'équipe domine sans vous)",
            formula:
              "coef_team_share  = mediane(pace_joueur / pace_team_per_player) sur 200 matchs\ncoef_lobby_share = mediane(pace_joueur / pace_lobby_per_player) sur 200 matchs\n\nRecalculés périodiquement au sync. Stale au-delà de 30 jours d'inactivité.",
            example:
              "Sur vos 200 dernières parties PvP rankées, votre pace moyen est 1.18× le pace per-player de vos équipes (coef_team_share = 1.18). Vous prenez donc 29.5% du travail dans un 4v4 (au lieu de la fair share 25%). Sur la même période, vous faites 1.05× le pace per-player du lobby (coef_lobby_share = 1.05) → vous êtes légèrement plus actif que la moyenne du lobby, mais surtout fort dans votre équipe vs vos coéquipiers.",
          },
          {
            term: 'Intensité du match',
            definition:
              "Caractéristique objective d'un match, indépendante du joueur : events/min/joueur du lobby sur la durée totale. Permet de classer les matchs entre eux (calme / standard / chaotique) et d'interpréter le score d'engagement dans son contexte.\n\nMatchs typiques :\n• Slayer 12 min équilibré : ~10-13 events/min/joueur\n• Slayer 4 min steamroll : ~8-12 (compressé)\n• Strongholds tactique : ~6-9 (longues phases calmes)\n• BTB chaotique : ~12-16",
            formula:
              'match_intensity = total_events_lobby / N_humains_lobby / durée_minutes\n\nStockée au niveau du match (1 valeur par match, ne change jamais après ingestion).',
            example:
              "Match Slayer 11:30, 8 humains au lobby, 95 events highlight (kills + deaths + assists + medals). Intensité = 95 / 8 / 11.5 = 1.03 events/min/joueur côté lobby per_player.\n\nUn match avec 14 events/min/joueur correspond à un cas chaotique (P88 vs votre historique). Un match à 6 events/min/joueur est à l'inverse calme/tactique.",
          },
        ],
      },
      {
        title: 'Badges narratifs de partie',
        entries: [
          {
            term: 'Comment les badges sont calculés',
            definition:
              "Les badges sont déterminés à partir de la courbe de score reconstruite partie par partie, en analysant l'évolution du score élimination après élimination. Seules les Victoires et Défaites reçoivent un badge (Égalités et abandons sont exclus).\n\nLes parties avec des robots coéquipiers peuvent être exclues selon vos préférences. La sensibilité par défaut est « standard » : seuil d'avance fixé à 40 % et seuil de retournement à 35 %.",
            formula:
              "Seuils (sensibilité standard) :\n• Seuil d'avance = score_final_max × 0,40\n• Seuil de retournement = score_final_max × 0,35\n\nPriorité de détection :\n1. Contre-Remontada (priorité maximale)\n2. Remontada\n3. Débandade\n4. Domination\n5. Humiliation",
            example:
              "Partie finale 50–32 (score_final_max = 50) :\nSeuil d'avance = 50 × 0,40 = 20 éliminations d'écart\nSeuil de retournement = 50 × 0,35 = 17,5 éliminations d'écart avant qu'un retournement soit reconnu",
          },
          {
            term: 'Domination',
            definition:
              "Victoire dans laquelle votre équipe a maintenu une avance importante et constante tout au long de la partie, sans jamais être réellement menacée. Badge de maîtrise totale.",
            formula:
              "Conditions : Victoire + avance maximale du joueur ≥ seuil d'avance\n(et aucun retournement ≥ seuil de retournement)",
            example:
              "Score final 50–28. Votre équipe a toujours mené d'au moins 20 éliminations → Domination. Si à un moment l'adversaire avait remonté à 5 éliminations d'écart, c'est trop peu pour déclencher un autre badge.",
          },
          {
            term: 'Humiliation',
            definition:
              "Défaite dans laquelle l'adversaire a maintenu une avance écrasante du début à la fin. Votre équipe n'a jamais vraiment été en position de renverser la tendance.",
            formula:
              "Conditions : Défaite + avance maximale adverse ≥ seuil d'avance\n(et aucun retournement ≥ seuil de retournement)",
            example:
              "Score final 32–50. L'adversaire a toujours mené de 20 éliminations ou plus → Humiliation.",
          },
          {
            term: 'Remontada',
            definition:
              "Victoire arrachée après avoir été nettement menés. Votre équipe était en position défavorable (retard ≥ seuil) à un moment de la partie avant de renverser la situation et de l'emporter.",
            formula:
              "Conditions : Victoire + l'adversaire avait une avance ≥ seuil de retournement avant la fin de la partie",
            example:
              "Score final 50–45. À mi-partie vous étiez en retard de 18 éliminations. Vous avez renversé la tendance → Remontada.",
          },
          {
            term: 'Débandade',
            definition:
              "Défaite après avoir été en position de force. Votre équipe menait nettement à un moment de la partie avant de s'effondrer et de perdre.",
            formula:
              'Conditions : Défaite + votre équipe avait une avance ≥ seuil de retournement avant la fin de la partie',
            example:
              "Score final 45–50. À mi-partie vous meniez de 18 éliminations. L'adversaire a rattrapé → Débandade.",
          },
          {
            term: 'Contre-Remontada',
            definition:
              "Le badge le plus rare : victoire après un double retournement. Votre équipe a d'abord mené, puis a été rattrapée et dépassée, avant de reprendre l'avantage et de gagner.\n\nLes deux équipes ont eu une avance significative à un moment de la partie.",
            formula:
              "Conditions : Victoire + votre équipe avait une avance ≥ seuil de retournement à un moment\n+ l'adversaire avait aussi une avance ≥ seuil de retournement à un autre moment\n(priorité maximale — détectée avant Remontada)",
            example:
              "Vous meniez de 20 éliminations en première moitié. L'adversaire est remonté et vous a dépassés de 18. Vous êtes repartis et avez gagné 50–47 → Contre-Remontada.",
          },
        ],
      },
      {
        title: 'Données et synchronisation',
        entries: [
          {
            term: 'Synchronisation',
            definition:
              "Processus de récupération des parties depuis l'interface Halo Waypoint et d'écriture dans la base DuckDB locale. En mode incrémental (par défaut), seules les nouvelles parties non encore enregistrées sont récupérées ; une synchronisation complète peut être forcée depuis les Paramètres.\n\nTrois déclencheurs coexistent :\n\n• Synchronisation manuelle — vous appuyez sur le bouton « Synchroniser » dans les Paramètres. C'est la méthode de référence après une longue session ou pour rattraper un retard ponctuel.\n\n• Synchronisation planifiée — un ordonnanceur interne déclenche une synchronisation incrémentale à intervalle fixe (par exemple toutes les 30 minutes). Activable et configurable depuis Paramètres › Synchronisation. Utile pour garder la base à jour en arrière-plan tant que l'application tourne, indépendamment de votre activité de jeu.\n\n• Détection de présence — un veilleur écoute les notifications Xbox en temps réel via un canal RTA (authentifié par jeton XSTS) et déclenche automatiquement une synchronisation dès qu'il détecte la fin d'une de vos parties. Réaction quasi immédiate, sans attendre le prochain tour de l'ordonnanceur.\n\nPrérequis côté Xbox : pour que les notifications RTA soient effectivement reçues, votre profil Xbox doit autoriser la lecture publique de votre statut de connexion et de votre activité de jeu en cours (paramètres « Vous voir en ligne » et « Vous voir jouer à un jeu » réglés sur « Tout le monde »). Si ces paramètres sont restreints (Amis uniquement, Bloqué), le canal RTA refuse l'abonnement et la détection de présence reste silencieuse — la synchronisation planifiée et la synchronisation manuelle continuent en revanche de fonctionner normalement.",
            example:
              "Vous lancez Halo, jouez 5 parties, puis fermez le jeu :\n• Détection de présence active → chaque fin de partie déclenche une synchronisation, vos 5 parties sont remontées au fil de l'eau.\n• Synchronisation planifiée seule (toutes les 30 min) → vos 5 parties sont remontées en 1 ou 2 passes, avec un délai maximum d'environ 30 minutes.\n• Aucune des deux activée → vous ouvrez LevelUp plus tard et appuyez sur « Synchroniser » : les 5 parties sont récupérées d'un coup, sans retoucher aux 500 déjà stockées.",
          },
          {
            term: 'Recalcul rétroactif',
            definition:
              "Recalcul ou remplissage rétroactif de données manquantes sur l'historique déjà synchronisé. Utile après une mise à jour qui introduit un nouveau champ (par exemple : tirs effectués, rang de compétence, médailles) : le recalcul rétroactif applique ce champ à toutes les parties existantes.",
            example:
              "Après l'ajout du calcul de badges en version 6.2, un recalcul rétroactif a été lancé pour produire les badges Remontada, Débandade et Contre-Remontada sur tout l'historique déjà synchronisé.",
          },
          {
            term: 'Normalisation des modes',
            definition:
              "Résolution d'un nom d'affichage unique et lisible en français à partir des variantes brutes renvoyées par l'interface Halo Waypoint. Une même playlist peut remonter sous plusieurs intitulés selon la version du jeu, la saison ou le canal d'où elle est lue.\n\nLevelUp consolide ces variantes en un seul nom français via une table de correspondance et applique le même traitement aux noms de cartes lorsque c'est nécessaire. Cela garantit qu'une partie de « Capture du drapeau » sur « Aquarius » apparaît toujours sous le même libellé dans vos statistiques, votre Explorer et votre Palmarès, peu importe sous quel intitulé brut Halo l'a renvoyée.",
            example:
              "L'interface Halo peut renvoyer « Massacre Grande équipe », « Grande équipe Massacre » ou « Massacre — Grande équipe » pour la même playlist : les trois sont unifiées en « Grande équipe — Massacre ».\n« Capture du drapeau », « CDD » et « Drapeau » → unifiés en « Capture du drapeau ».\nLa table de correspondance (mode_pair_overrides) contient une trentaine d'entrées en français et en anglais pour couvrir les cas ambigus.",
          },
          {
            term: 'Fréquences de rafraîchissement',
            definition:
              "Toutes les données affichées par LevelUp ne sont pas rafraîchies au même rythme. Le choix dépend de trois critères : le coût de l'appel à l'interface Halo (certaines requêtes sont lentes ou limitées en débit), la fraîcheur réellement attendue par l'utilisateur, et la possibilité ou non de calculer la donnée à partir de la base locale.\n\nQuatre canaux coexistent :\n\n• Lecture directe Halo à chaque ouverture de page — pour les données « live » qui changent à chaque partie et que l'utilisateur s'attend à voir à jour immédiatement.\n\n• Lecture directe Halo avec cache disque 24 h — pour les données qui changent peu (seuils de rangs, libellés de cartes, descriptions de défis), mais qui sont lourdes à récupérer.\n\n• Cache mémoire court de 5 à 10 minutes sur la base DuckDB locale — pour les agrégats et listes consultés en boucle pendant une session de navigation.\n\n• Mise à jour à la synchronisation uniquement — pour le détail des parties et les calculs dérivés (LUSR, score de performance, citations, agrégats coéquipiers). La synchronisation elle-même a ses propres déclencheurs : voir l'entrée « Synchronisation » pour les trois modes (manuelle, planifiée, détection de présence) et l'entrée « Recalcul rétroactif » pour les opérations de reprise sur l'historique.",
            example:
              "Lecture directe Halo à chaque ouverture (page Accueil) :\n• Passe de combat saisonnier — niveau et points actuels, repris à chaque rafraîchissement de page.\n• Défis en cours (quotidiens, hebdomadaires, événements) — état d'avancement et récompenses.\n• Succès Xbox Halo Infinite — liste des succès débloqués et progression.\n• Données de carrière — rang Spartan courant, progression vers le palier suivant.\n\nLecture directe Halo avec cache disque 24 h :\n• Référentiel des paliers de carrière (noms, seuils de points)\n• Référentiel des cartes et modes (libellés français, images)\n• Métadonnées du passe de combat saisonnier (récompenses par palier)\nLe cache disque évite de retaper l'interface Halo à chaque ouverture pour des données quasi statiques.\n\nCache mémoire court (5 à 10 minutes) sur DuckDB local :\n• Pages Statistiques et Palmarès (agrégats sur tout l'historique)\n• Page Escouade (synergies, contributions par coéquipier)\n• Listes de parties paginées dans l'Explorer\n\nMise à jour à la synchronisation uniquement :\n• Détail des parties (frags, dégâts, médailles, événements filmés)\n• Rang LUSR par partie (calculé localement) et CSR officiel par partie\n• Score de performance par partie\n• Citations et badges narratifs\n• Statistiques de coéquipiers sur les parties communes\n→ Voir « Synchronisation » pour les trois modes de déclenchement (manuelle, planifiée, détection de présence).\n\nMise à jour au recalcul rétroactif :\n• Champs ajoutés par une nouvelle version sur des parties anciennes (ex. badges Remontada après v6.2)\n→ Voir « Recalcul rétroactif » pour le déclenchement et la portée.\n\nIndexation automatique en arrière-plan :\n• Les fichiers médias locaux (captures, clips) sont ré-indexés après chaque synchronisation.",
          },
        ],
      },
      {
        title: 'Progression & Gamification',
        entries: [
          {
            term: 'Objectifs',
            definition:
              "Système de défis propre à LevelUp, distinct des défis Mission Control de Halo. Chaque objectif cible une métrique Halo (kills, KDA, précision, dégâts…) sur une fenêtre temporelle définie — quotidienne, hebdomadaire, mensuelle ou libre.\n\nLes objectifs sont organisés en **arcs** : des séquences thématiques qui structurent un parcours progressif. Un arc regroupe plusieurs objectifs liés (par exemple, « améliorer la précision sur 30 jours » puis « maintenir un KDA ≥ 3 la semaine suivante »).\n\nDeux modes d'évaluation :\n• **Seuil (threshold)** — atteindre la cible en une seule partie (ex. « 20 kills dans un match »).\n• **Cumulatif** — accumuler sur toute la période (ex. « 500 kills en un mois »).\n\nDeux modes de création :\n• **Libre** — vous choisissez la métrique, la cible et la fenêtre vous-même.\n• **Piloté** — LevelUp suggère un objectif calibré sur votre historique.\n\nLes objectifs peuvent être **individuels** (gérés dans votre profil joueur) ou **d'escouade** (partagés entre les membres). Les défis d'escouade proposent deux dynamiques :\n• **Collectif** — chaque membre contribue à un objectif commun (ex. « 1 000 kills au total cette semaine »).\n• **Compétitif** — les membres s'affrontent sur la même métrique et se classent entre eux.\n\nChaque objectif complété récompense des **Prestige Points (PP)** selon son palier (Normal, Heroic, Legendary, Mythic).",
            example:
              "Vous créez un objectif libre : « 3 000 dégâts par partie en moyenne sur les 10 prochaines parties », mode cumulatif, palier Heroic. Une fois atteint, vous gagnez les PP associés et l'objectif passe au statut Complété dans votre parcours.",
          },
          {
            term: 'Prestige',
            definition:
              "Système de progression propre à LevelUp, indépendant du rang Halo officiel. Chaque objectif complété rapporte des **Prestige Points (PP)** dont le montant dépend du palier de l'objectif.\n\nLes PP s'accumulent dans votre profil et déterminent votre palier Prestige :\n• **Normal** (gris) — premiers objectifs\n• **Heroic** (bleu) — engagement régulier\n• **Legendary** (violet) — maîtrise avancée\n• **Mythic** (or) — niveau d'élite\n\nLe **Leaderboard PP** (sous-onglet Communauté dans Palmarès) compare vos PP à ceux des joueurs de votre escouade et de vos relations — dérivé automatiquement des données partagées, sans gestion d'amis manuelle.\n\nAccès : barre de navigation → Objectifs pour les défis et le parcours ; Palmarès → Communauté pour le classement.",
            example:
              "Vous avez complété 12 objectifs Heroic et 3 Legendary. Vos PP totaux vous placent deuxième dans le leaderboard de votre escouade, avec le badge Legendary affiché sur votre profil.",
          },
        ],
      },
      {
        title: 'Navigation et organisation',
        entries: [
          {
            term: 'Sessions',
            definition:
              "Regroupement automatique de parties consécutives séparées par moins de 2 heures d'inactivité. Une session représente une « soirée de jeu » continue.\n\nL'analyse des sessions permet d'étudier comment votre performance évolue au fil d'une même session (fatigue, échauffement, etc.).",
            example:
              '5 parties jouées entre 20 h 00 et 22 h 30 → 1 session.\nUne 6ᵉ partie jouée à 1 h 00 → nouvelle session distincte.',
          },
          {
            term: 'Escouade',
            definition:
              "Groupe de joueurs synchronisés sur LevelUp qui partagent des parties communes. Les pages Escouade calculent des statistiques agrégées sur les parties jouées ensemble : synergies, contributions, carte de chaleur d'intensité.\n\nUn joueur n'apparaît dans votre escouade que s'il est lui-même synchronisé dans l'application.",
            example:
              "Si trois de vos coéquipiers réguliers utilisent LevelUp, votre page Escouade affichera vos parties jouées avec eux et leurs statistiques sur cet échantillon commun.",
          },
          {
            term: 'Explorer',
            definition:
              "Vue d'exploration détaillée de toutes vos parties avec filtres en cascade : carte, mode, liste de jeu, résultat, date, session. Permet une analyse fine et la navigation vers la vue détaillée de chaque partie.",
            example:
              "Filtrer par carte « Streets », mode « Capture du drapeau », résultat « Victoire » sur les 30 derniers jours pour isoler vos parties gagnées récemment dans cette configuration.",
          },
          {
            term: 'Accessibilité couleurs',
            definition:
              "Paramètre de l'onglet Accessibilité (Paramètres) permettant de basculer toute l'interface sur la palette Okabe-Ito (2008). Cette palette utilise 8 couleurs distinguables par les trois principaux types de daltonisme (protanopie, deutéranopie, tritanopie). Le choix est persisté localement et n'affecte pas les autres joueurs.",
            example:
              "Si les barres de performance rouge/vert se ressemblent sur votre écran, activez la palette Okabe-Ito : les mêmes niveaux de performance sont alors encodés en bleu ciel, bleu-vert, jaune, orange et vermillon — distincts pour tous les types de vision.",
          },
          {
            term: 'Palmarès',
            definition:
              "Section regroupant vos classements (tableau local par liste de jeu et par saison), vos relations (alliés fréquents, ennemis récurrents, victimes habituelles), la comparaison face-à-face avec un autre joueur, et votre passe saisonnier Halo.",
            example:
              "Le sous-onglet Relations met en évidence vos cinq alliés les plus fréquents et leur taux de victoire commun, ainsi que vos cinq ennemis récurrents et le solde des affrontements.",
          },
        ],
      },
    ],
  },
}

const EN_TEXT: HelpText = {
  tabs: {
    glossary: 'Glossary & Concepts',
    releaseNotes: 'Release Notes',
  },
  page: {
    title: 'Help',
    subtitle: 'LevelUp documentation and concepts',
  },
  releaseNotes: {
    loading: 'Loading release notes…',
    error: 'Unable to load release notes.',
    title: 'Release Notes',
  },
  glossary: {
    title: 'Glossary & Concepts',
    sections: [
      {
        title: 'Performance Metrics',
        entries: [
          {
            term: 'LUSR',
            definition:
              "Skill rating based on TrueSkill 2, adapted for Halo Infinite. Think of it as the equivalent of CSR (the official ranked rating) for unranked modes: where CSR is assigned by Halo for ranked playlists, LUSR is computed locally by LevelUp across all other modes (arena, btb, social, fun…) using the same TrueSkill logic. Computed separately for each playlist group. Starts at 1 500 with an initial uncertainty (σ) of 350 that decreases with each match. Only Win/Loss/Tie outcomes count — DNF counts weakly (0.15). An inactivity drift is applied after 1+ day without playing to reflect potential rust.",
            formula:
              'Displayed LUSR = μ − 3σ\n\nΔμ = 32 × (composite_score − 0.5) × group_weight\nComposite score = 0.31×(kills/expected) + 0.28×(expected deaths/deaths) + 0.23×(damage/expected) + 0.13×(accuracy delta) + 0.05×(win factor)\n\nGroup weights: ranked 1.0 · arena 0.8 · btb 0.7 · fun 0.25\nCap: ±100 pts per match',
            example:
              'A player with μ = 1 700 and σ = 80 displays LUSR 1 460. After 2 weeks inactive, σ rises ~13 pts (1 pt/day, capped at 14 days), making the next match more impactful in both directions.',
          },
          {
            term: 'Performance Score',
            definition:
              'Relative 0–100 score measuring your contribution in a match vs your own history. Computed from 13 weighted metrics — each metric is converted to a percentile rank over your past matches. Requires at least 10 historical matches to activate. Bot-resistant: if bots were on your team, a corrective bonus is applied based on the MMR gap.',
            formula:
              'Score = Σ(percentile_rank × weight) / Σ(active weights), clamped 0–100\n\nMain weights:\n0.14 × kills/min · 0.11 × KDA · 0.10 × deaths/min (inverted)\n0.10 × personal score/min · 0.09 × kills vs expected\n0.09 × offensive conversion · 0.07 × deaths vs expected\n0.06 × damage/min · 0.06 × heroic medals · 0.05 × defensive resistance\n0.04 × accuracy · 0.04 × rank vs expected',
            example:
              '80/100 in a match = you were in the top 20 % of your own historical performances across those metrics.',
          },
          {
            term: 'Offensive Conversion',
            definition:
              'Offensive efficiency: measures how much damage you need to convert a kill. The 225 coefficient is based on the assumption that a Spartan has 225 total health points (90 base HP + 135 shields), the official Halo Infinite convention. Above 1.0 = you finish kills with less damage than a full Spartan\'s health (accuracy, headshots that skip the shield). Below = you waste damage (unconverted assists, poor follow-up).',
            formula:
              'Offensive conversion = 225 × (kills + assists/3) / damage_dealt\n\nP80 reference (real data): 0.83\n(= 83 % of matches have conversion ≤ 0.83)',
            example:
              '10 kills, 6 assists, 2 800 damage → 225 × (10 + 2) / 2 800 ≈ 0.96: above P80, efficient conversion.\n6 kills, 2 assists, 2 800 damage → 225 × (6 + 0.67) / 2 800 ≈ 0.54: lots of damage for few kills.',
          },
          {
            term: 'Defensive Resistance',
            definition:
              'Measures how much damage you absorb per death. Uses the same assumption: a Spartan has 225 total health (90 base HP + 135 shields). Above 1.0 = you die after absorbing more than a full Spartan\'s health (good resilience, forces enemies to commit a full magazine). Below = you die early in engagements, often surprised or poorly positioned.',
            formula:
              'Defensive resistance = damage_taken / (225 × deaths)\n\nP80 reference (real data): 1.59',
            example:
              '5 deaths, 1 400 damage taken → 1 400 / (225 × 5) ≈ 1.24: above 1.0, you absorb 1.24× a Spartan\'s health before dying.\n5 deaths, 750 damage taken → 750 / 1 125 ≈ 0.67: you die early in most engagements.',
          },
          {
            term: 'KDA',
            definition:
              'Classic kills/deaths/assists ratio, with a floor of 1 death to avoid division by zero. Assists count fully (unlike offensive conversion where they count as 1/3). A reference metric but volume-sensitive — a very active player may have the same KDA as a passive one.',
            formula: 'KDA = (kills + assists) / max(1, deaths)',
            example:
              '15 kills, 4 assists, 6 deaths → KDA = 19/6 ≈ 3.17\n0 kills, 0 assists, 0 deaths → KDA = 0/1 = 0 (floor at 1 death)',
          },
        ],
      },
      {
        title: 'Match Narrative Badges',
        entries: [
          {
            term: 'How Badges Are Computed',
            definition:
              'Badges are determined from the score curve reconstructed kill-by-kill for each match. Only Win or Loss matches receive a badge (Tie and DNF are excluded). Matches with bot teammates may be excluded based on your preferences. Default sensitivity: "standard" (leadPct = 40 %, comebackPct = 35 %).',
            formula:
              'Thresholds (standard sensitivity):\n• leadThreshold = final_max_score × 0.40\n• comebackThreshold = final_max_score × 0.35\n\nDetection priority:\n1. Counter-Comeback (highest priority)\n2. Comeback\n3. Collapse\n4. Domination\n5. Humiliation',
            example:
              'Final score 50–32 (final_max_score = 50):\nleadThreshold = 50 × 0.40 = 20 kills lead\ncomebackThreshold = 50 × 0.35 = 17.5 kills gap before a reversal triggers',
          },
          {
            term: 'Domination',
            definition:
              'Victory in which your team maintained a significant and consistent lead throughout the match, never truly threatened. A badge of total control.',
            formula:
              'Conditions: Win + max player lead ≥ leadThreshold\n(and no comeback swing ≥ comebackThreshold)',
            example:
              'Final 50–28. Your team always led by 20+ kills → Domination. If at some point the enemy closed to −5, that\'s too narrow to trigger another badge.',
          },
          {
            term: 'Humiliation',
            definition:
              "Defeat in which the enemy maintained an overwhelming lead from start to finish. Your team never had a real chance to turn things around.",
            formula:
              'Conditions: Loss + max enemy lead ≥ leadThreshold\n(and no comeback swing ≥ comebackThreshold)',
            example:
              'Final 32–50. The enemy always led by 20+ kills → Humiliation.',
          },
          {
            term: 'Comeback',
            definition:
              'Victory after being significantly behind. Your team was in a losing position (deficit ≥ threshold) at some point before turning the match around and winning.',
            formula:
              'Conditions: Win + the enemy had a lead ≥ comebackThreshold before the end of the match',
            example:
              'Final 50–45. At mid-match you were −18 kills behind. You reversed it → Comeback.',
          },
          {
            term: 'Collapse',
            definition:
              'Defeat after being in a strong position. Your team led significantly at some point before falling apart and losing.',
            formula:
              'Conditions: Loss + your team had a lead ≥ comebackThreshold before the end of the match',
            example:
              'Final 45–50. At mid-match you led by +18 kills. The enemy caught up → Collapse.',
          },
          {
            term: 'Counter-Comeback',
            definition:
              'The rarest badge: victory after a double reversal. Your team led, then was overtaken and fell behind, before reclaiming the advantage and winning. Both teams had a significant lead at some point.',
            formula:
              'Conditions: Win + your team had a lead ≥ comebackThreshold at some point\n+ the enemy also had a lead ≥ comebackThreshold at another point\n(highest priority — detected before Comeback)',
            example:
              'You led +20 in the first half. Enemy came back and overtook you by −18. You pushed back and won 50–47 → Counter-Comeback.',
          },
        ],
      },
      {
        title: 'Data & Sync',
        entries: [
          {
            term: 'Sync',
            definition:
              'Process that fetches matches from the Halo Waypoint API and writes them to the local DuckDB database. Delta mode (default) only fetches new matches not yet recorded. A full sync can be forced from Settings.',
            example:
              'You played 5 matches since the last sync. Delta sync fetches only those 5 matches, leaving the 500 already stored untouched.',
          },
          {
            term: 'Backfill',
            definition:
              'Retroactive recalculation or population of missing data on already-synced history. Useful after an update that introduces a new field (e.g. shots_fired, skill rank, medals): backfill computes that field for all existing matches.',
            example:
              'After adding badge computation in v6.2, a backfill was run to calculate Comeback/Collapse/Counter-Comeback badges on all previously synced history.',
          },
          {
            term: 'Mode Normalisation',
            definition:
              'Resolution of a unique display name from the raw variants returned by the Waypoint API. The API may return "BTB Slayer", "BTB-Slayer" or "Big Team Battle Slayer" for the same mode — normalisation unifies them as "BTB — Slayer" via the mode_pair_overrides table.',
            example:
              'The mode_pair_overrides table in metadata.duckdb contains ~29 FR/EN overrides for ambiguous cases.',
          },
          {
            term: 'Refresh Frequencies',
            definition: 'LevelUp uses several data freshness levels depending on the page.',
            example:
              'Live (every page open): Home, Last Match.\nQuery cache 5–10 min: Stats, Palmares, Squad.\nManual sync: triggered from the Sync button in Settings.\nAuto background: media is re-indexed after every sync.',
          },
        ],
      },
      {
        title: 'Progression & Gamification',
        entries: [
          {
            term: 'Objectives',
            definition:
              "LevelUp's challenge system, separate from Halo's Mission Control challenges. Each objective targets a Halo metric (kills, KDA, accuracy, damage…) over a defined time window — daily, weekly, monthly, or free.\n\nObjectives are organised into **arcs**: thematic sequences that structure a progressive journey. An arc groups several related objectives (e.g. \"improve accuracy over 30 days\" followed by \"sustain KDA ≥ 3 the following week\").\n\nTwo evaluation modes:\n• **Threshold** — reach the target in a single match (e.g. \"20 kills in one match\").\n• **Cumulative** — accumulate across the full window (e.g. \"500 kills in a month\").\n\nTwo creation modes:\n• **Free** — you choose the metric, target, and window yourself.\n• **Guided** — LevelUp suggests an objective calibrated to your history.\n\nObjectives can be **individual** (stored in your player profile) or **squad-level** (shared across squad members). Squad challenges offer two dynamics:\n• **Collective** — every member contributes toward a shared target (e.g. \"1 000 kills combined this week\").\n• **Competitive** — members race on the same metric and rank against each other.\n\nEach completed objective awards **Prestige Points (PP)** based on its tier (Normal, Heroic, Legendary, Mythic).",
            example:
              'You create a free objective: "Average 3 000 damage per match over the next 10 matches", cumulative mode, Heroic tier. Once reached, you earn the PP and the objective moves to Completed in your journey.',
          },
          {
            term: 'Prestige',
            definition:
              "LevelUp's own progression layer, independent of the official Halo rank. Each completed objective awards **Prestige Points (PP)** that scale with the objective's tier.\n\nPP accumulate in your profile and determine your Prestige tier:\n• **Normal** (grey) — first objectives\n• **Heroic** (blue) — consistent engagement\n• **Legendary** (purple) — advanced mastery\n• **Mythic** (gold) — elite level\n\nThe **PP Leaderboard** (Community sub-tab in Palmares) compares your PP against players in your squad and relations — derived automatically from shared match data, no manual friend management.\n\nAccess: navigation bar → Objectives for challenges and journey; Palmares → Community for the leaderboard.",
            example:
              'You completed 12 Heroic and 3 Legendary objectives. Your total PP rank you second in your squad leaderboard, with a "Legendary" badge on your profile.',
          },
        ],
      },
      {
        title: 'Navigation & Organisation',
        entries: [
          {
            term: 'Sessions',
            definition:
              'Automatic grouping of consecutive matches separated by less than 2 hours of inactivity. A session represents a continuous "gaming evening". Session analysis lets you study how performance evolves within a session (fatigue, warm-up, etc.).',
            example:
              '5 matches played between 8 PM and 10:30 PM → 1 session.\nA 6th match at 1 AM → a new, separate session.',
          },
          {
            term: 'Squad',
            definition:
              'Group of LevelUp-synced players who share common matches. Squad pages compute aggregated stats on matches played together: synergies, contributions, intensity heatmap. A player only joins your squad if they are themselves synced in the app.',
          },
          {
            term: 'Explorer',
            definition:
              'Drilldown view of all your matches with cascade filters: map, mode, playlist, outcome, date, session. Enables fine-grained analysis and navigation to each match detail view.',
          },
          {
            term: 'Color Accessibility',
            definition:
              'Setting in the Accessibility tab (Settings) that switches the entire interface to the Okabe-Ito (2008) color palette. This palette uses 8 colors distinguishable by the three main types of color blindness (protanopia, deuteranopia, tritanopia). The preference is stored locally and does not affect other players.',
            example:
              'If the red/green performance bars look the same on your screen, enable the Okabe-Ito palette: the same performance levels will then be encoded in sky blue, bluish green, yellow, orange and vermillion — distinct for all vision types.',
          },
          {
            term: 'Palmares',
            definition:
              'Section grouping your rankings (local leaderboard by playlist/season), your relationships (frequent allies, nemeses, frequent victims), player-vs-player comparison, and your Halo season pass.',
          },
        ],
      },
    ],
  },
}

const TEXT: Record<HelpLocale, HelpText> = { fr: FR_TEXT, en: EN_TEXT }

export function normalizeHelpLocale(locale?: string | null): HelpLocale {
  return locale === 'en' ? 'en' : 'fr'
}

export function getHelpText(locale?: string | null): HelpText {
  return TEXT[normalizeHelpLocale(locale)]
}
