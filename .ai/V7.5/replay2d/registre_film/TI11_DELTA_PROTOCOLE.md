# PROTOCOLE — ti=11 SUR LE FLUX DELTA (COMMITE AVANT MESURE)

> Ecrit le 2026-08-27, branche `wt/ti11-cadre`, base `84ed417c7` (tete du lot remesure :
> gate T1 cadre TENU a 90,32 % keyframe, mais le keyframe ne porte que l'ETAT PAR DEFAUT —
> i3 porteur = 0xFFFFFFFF null, i12/i13 = 0, i1 = gris, i16-31 = null). Ce document FIGE le
> corpus, les seuils et les temoins AVANT toute mesure. Les seuils sont recopies du prompt du
> lot SANS modification. Aucune mesure ne modifie ce fichier ; les resultats vont aux logs
> figes (`TI11_DELTA_D1.log`, et logs D2 par confrontation) et au compte-rendu.

## 0. Objet et instrument

Question unique de D1 : la grammaire ti=11 34-feuilles MAINTENANT VALIDEE (les 34 desers de
`consumeByName`, dont l'atterrissage keyframe est 90,32 % cumule sous le cadre C5) fait-elle
PARSER proprement les records ti=11 du flux DELTA — et surtout, les champs VIVANTS (i3
object-reference, i1 couleur, i12/i13 progression, i16-31 sous-entites) y sont-ils PEUPLES
(non-sentinelle) et EVOLUENT-ils dans le temps ? La keyframe a prouve que le cadre est juste
mais qu'elle ne stocke que l'etat par defaut (T2 du lot precedent) : le vivant, s'il existe
dans ti=11, est DELTA-porte.

### DISTINCTION TECHNIQUE HONNETE : le CADRE delta n'est PAS le cadre keyframe C5

Le lot precedent a valide le cadre **C5** (en-tete 108 + mots de taille + etat par defaut +
LevelShift) : c'est le cadre de l'IMAGE-CLE (payload type-2), ou la boucle d'etat complet
(`WalkKeyframeFullState` -> `traverseComponentLoop`) itere TOUS les composants de l'archetype
dans l'ordre, gates par l'etat par defaut. Le flux DELTA a un cadre DIFFERENT : le record
d'OBJET DU MONDE masque (`matchWorldObjectRecord` : prefixe(1) + slot(13) + gen(2) +
porte(2==0) + nb-composants(3) + indices croissants(6b)), suivi de la marche des SEULS
composants du masque, dans l'ORDRE DU MASQUE. Ce qui est PARTAGE entre les deux chemins, et
qui a ete valide en T1, c'est la GRAMMAIRE : les desers de `consumeByName`
(`components_managed_objective.go`). Le cadre C5 lui-meme ne s'applique PAS au delta ; ce qui
s'y applique, c'est la grammaire feuille par feuille.

### Le chemin delta ti=11 est-il celui de l'objet du monde ? DEJA prouve pour le frere ti=13

`zone_state_scan.go` (EN PRODUCTION) balaie le flux DELTA de l'archetype FRERE ti=13
(managed-property, les zones) par exactement `worldObjectSlotBand` + `matchWorldObjectRecord`
+ `consumeByName` + temoin de chainage `worldObjectHeaderAt` — zero copie de grammaire. Le
canal scalaire y chaine a 97 %. ti=11 (managed-objective) est un archetype voisin ; l'HYPOTHESE
mesuree ici est qu'il partage ce cadre delta. La sonde R4 `TestTI11MasqueEtBande`
(`sonde_ti11_objectifs_test.go`) balayait DEJA le delta ti=11 par `matchWorldObjectRecord`,
mais ne faisait que lire le MASQUE (indice hors-grammaire 45,9 % sur la bande observee) : elle
ne MARCHAIT PAS les records avec la grammaire (incomplete a l'epoque) ni ne moissonnait les
champs. C'est ce que D1 ajoute : marche complete + hook + evolution temporelle.

### Instrument (D1)

Harnais de MESURE sous garde d'environnement (`TI11_ROOT` / `TI11_FILMS`), jamais en CI, un
seul decodage filmdec par process (`LockProcessDecode`), un film charge a la fois (borne
memoire). Il NE PUBLIE RIEN (schema/contrat/calque intacts en D1/D2). Mecanique, par film :

1. Bande = slots ti=11 OBSERVES en keyframe (`worldObjectSlotBand` donne la bande comblee ;
   pour un objectif — qui vit toute la partie et apparait a CHAQUE keyframe — l'OBSERVE est
   deja complet, donc on rapporte les DEUX : observe pur ET comble, l'observe faisant foi,
   exactement le choix documente par la sonde R4).
2. Pour chaque paquet DELTA (avec son `TimestampUS`, meme horloge que les positions de bipede) :
   ancrer les records par `matchWorldObjectRecord`, MARCHER le masque par `consumeByName`
   (typeIndex=11, `arch.Level(id)`), moissonner les champs par `SetManagedObjectiveHook`,
   mesurer le CHAINAGE (`worldObjectHeaderAt` a la fin) comme temoin de largeur.
3. Agreger : records ancres / marches (walked) / casses (broken) / chaines ; part hors-grammaire
   (indice masque > 33) ; par champ vivant, valeurs distinctes non-sentinelle et EVOLUTION dans
   le temps par (Gen,Slot).

## 1. Corpus admis (verifie present dans `data/cache/film_chunks` du principal, 2026-08-27)

| famille | films (prefixe 8 hex) | chunks |
|---|---|---|
| Oddball (corpus 5) | 24dbb67d, 43716616, 92f18088, d9781168, c88ec007 | 29/20/34/39/36 |
| CTF | 64e8adfa (Catalyst), 530820e5, 53ce4390 | 45/27/41 |
| KOTH | 01e1f945 (Catalyst), 606d9844, 8076f97f, 0a247154 | 30/15/21/42 |
| Strongholds/Bastion | 7344d24f, 696a9d7c, 10ed320d | 33/31/34 |

Corpus de MESURE D1 (parse + vivant) : les 5 Oddball + CTF 64e8adfa + KOTH 01e1f945 (>= 1 CTF
et >= 1 KOTH exiges), etendu aux autres films des familles pour le denominateur si besoin.
Corpus PORTEUR (D2 i3) : les 5 Oddball (oracle `time_as_skull_carrier_seconds`). Corpus ZONES
(D2 i16-31 / i1 / i12-13) : Bastion 7344d24f/696a9d7c/10ed320d + KOTH pour i1.

RESERVE reportee du lot precedent (decouverte D1 du plan cadre) : les 3 films Strongholds
rendent ZERO record ti=11 BORNE par `WalkKeyframeWorld` en keyframe. Sur le flux DELTA la
question se re-pose independamment (le delta ancre par bande, pas par l'oracle keyframe) : la
mesure D1 dira si Bastion porte des records ti=11 en delta. Si zero, D2.3/D2 progression
basculent sur un autre film Bastion ou restent [!] documente.

## 2. Temoins (figes)

1. TEMOIN BANDE FANTOME : bande de meme cardinalite faite de slots JAMAIS vus porter ti=11
   (`ti11GhostBand`), passee par le MEME balayage. Le taux de records ancres/chaines sur le
   fantome doit s'effondrer face au signal (regle du lot armes au sol : le temoin passe par le
   meme code, sinon il ne controle pas le decodeur mais une variante de lui).
2. TEMOIN GRAMMAIRE (hors-grammaire) : part des records ancres dont le masque porte un indice
   > 33 (un record ti=11 ne PEUT porter que i0..i33). Baseline R4 = 45,9 % sur la bande
   observee : ce que D1 doit expliquer (contamination de bande vs vrai record ti=11).
3. TEMOIN ALEATOIRE (champs vivants) : 4096 GlobalID 32b tires au hasard (graine fixe) ne
   doivent PAS tomber dans l'ensemble des i3/i16-31 observes (<= 1 %) — controle que le vivant
   observe est un ensemble creux d'entites reelles, pas du bruit dense.
4. TEMOIN DECALE 12 m (D2 zones, repris du gate Oddball historique) : decaler le lieu de repos
   de reference de 12 m doit effondrer l'appariement (<= 20 %).

## 3. SEUILS — recopies du prompt du lot, NON MODIFIABLES

### GATE D1 (le gate maitre, joue en premier ; conditionne D2)

> **>= 80 %** des records delta ti=11 PARSENT (marche aboutie + coherente ; le chainage est le
> temoin de largeur, comme ti=13) ET **>= 1** champ vivant non-sentinelle EVOLUE dans le temps.
> Denominateur = records delta ti=11 ancres par la bande. Log fige `TI11_DELTA_D1.log`.
>
> - SI les champs restent des SENTINELLES meme en delta (le vivant n'est nulle part dans
>   ti=11) : verdict **[!] chiffre, STOP**. La reprise bascule sur la voie STATBORG
>   (`skull_grabs` analogue au drapeau) pour le porteur, et l'owner deja publie pour les zones.
>   NE PAS s'acharner. NE RIEN PUBLIER.
> - SI D1 passe : enchainer D2.

### GATES D2 (seulement si D1 passe), par champ, chacun contre son oracle deja en prod

- **D2 i1 couleur -> camp** (clustering RGBA) vs `zone_states` (owner publie 93 %) /
  `hillStates` (88-89 %) : owner de zone/colline. Seuil **>= 90 % global**, **>= 85 % par
  equipe** prise SEPAREMENT.
- **D2 i12/i13 progression** vs la jauge de capture publiee (Bastion). Seuil accord **>= 85 %**.
- **D2 i3 object-reference -> objet physique** (crane/drapeau ti=42) : croiser au calque objet
  DEJA publie (vies libres du crane/drapeau). Si i3 pointe le crane, sa POSITION (via l'objet
  ti=42) donne le lieu, et le PORTEUR = le joueur a cette position pendant un trou (recette
  existante). Confronter au gate historique Oddball `time_as_skull_carrier_seconds`
  (`match_objective_stats_latest`) **>= 80 % par joueur** ET **porteur principal >= 3/5 films**.
  RESERVE HONNETE : i3 nomme l'OBJET, pas forcement le JOUEUR — le CR dira ce que i3 donne
  exactement et ce qu'il faut encore pour le porteur individuel.
- **D2 i16-31 sous-entites -> identite de zone A/B/C** (Bastion cardinal 3, Total Control 3
  actives). Seuil appariement **>= 90 %**, temoin decale 12 m **<= 20 %**.

Logs figes par confrontation.

### D3 PUBLICATION (seulement les champs dont le gate D2 tient)

Owner de zone/colline, identite de zone, progression native, et/ou porteur si le gate
historique tient. Triplet schema Go/contrat/web + chronique, i18n FR+EN, calque au patron
`zoneStatesLayer`/`flagCarriesLayer`, re-cuisson TEMOINS avec verification de CONTENU. Gates :
go test paquets touches + contracttest, tsc -b (cache purge), vitest match-replay, lint web,
parite schema. Ne publie QUE ce qui passe son gate ; le reste reste [!] documente.

## 4. Gates techniques du lot

Protocole commite (ce fichier, un seul commit). Logs figes. `go vet` / `go build` exit 0 sur
les paquets touches. Arbre propre. Pas de push. `git add` par fichier (jamais `-A`).
thought_log/REGISTRE non touches (textes au CR). Si publication (D3) : gates web+contrat verts.
Donnees du principal en LECTURE via `LEVELUP_REPO_ROOT` / `TI11_ROOT` pointant
`c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks`.
