# TODO

## Migration NMEA

- Reevaluer a terme le fork `github.com/jgrelet/go-nmea`.
- La librairie upstream `github.com/adrianmo/go-nmea` semble aujourd'hui assez complete.
- Avant migration, verifier les ecarts d'API qui comptent pour `geo-acq`, en particulier :
- parsing des trames actuellement utilisees
- mutation des structs parsees
- re-serialisation des trames cote simulateurs
- gestion du type `LatLong`

## Sauvegarde acquisition

### Constat

- Aujourd'hui, toute l'acquisition est stockee dans une seule base SQLite brute.
- La table `raw_frames` contient a la fois :
- la trame NMEA brute (`payload`)
- le type de phrase (`sentence_type`)
- le JSON decode (`decoded_json`)
- Ce mode est utile pour du rejeu complet, mais il peut produire rapidement de gros volumes.

### Objectif

- Rendre la sauvegarde configurable selon le besoin :
- sauvegarde brute pour rejeu / audit
- sauvegarde des donnees decodees pour post-traitement plus leger
- possibilite d'activer l'un, l'autre, ou les deux

### Configuration TOML a faire evoluer

- Renommer le paragraphe `[acq]`.
- Nom propose : `[backup]`
- Raison : le bloc ne decrit pas l'acquisition elle-meme mais la politique de persistance.
- Alternative possible si besoin plus explicite : `[storage]`

Proposition de structure :

```toml
[backup]
raw     = true
process = true
```

Remarques :

- `raw` : sauvegarde des trames brutes NMEA
- `process` : sauvegarde des donnees decodees
- Le nom `process` est acceptable, mais `processed` serait sans doute plus clair et plus idiomatique.
- A confirmer au moment de l'implementation : preferer `processed` a `process` sauf contrainte de compatibilite voulue.

### Nommage des fichiers

- Deriver les noms de fichiers du nom de mission.
- Exemple pour mission `Test` :
- `Test-raw.sqlite`
- `Test-data.sqlite`

Questions a trancher pendant l'implementation :

- sanitisation du nom de mission pour le nom de fichier
- dossier de sortie par defaut
- comportement si le nom de mission est vide

### Travail technique a prevoir

- Faire evoluer `config.Config` pour remplacer `[acq].file` par une structure de sauvegarde plus riche.
- Adapter les TOML racine et les exemples.
- Adapter l'application et le CLI pour ouvrir zero, une ou deux bases selon la config.
- Decoupler le store SQLite actuel, aujourd'hui centre sur `raw_frames`.
- Conserver la session et les metadonnees de mission dans les deux bases, ou definir une relation claire entre elles.

### Base brute

- Garder une base brute append-only pour le rejeu.
- Cette base peut rester proche du schema actuel :
- `missions`
- `acquisition_sessions`
- `raw_frames`

### Base decodee

- Concevoir une nouvelle structure SQLite dediee aux donnees decodees.
- Eviter de stocker uniquement un blob JSON si l'objectif principal est le post-traitement scientifique.
- Preferer un schema interrogeable et compact.

Pistes de schema :

- Option 1 : une table normalisee commune
- colonnes communes : `session_id`, `mission_id`, `received_at_utc`, `device_name`, `sentence_type`
- charge utile decodee en JSON
- simple a faire, souple, mais moins performante pour les requetes metier

- Option 2 : une table par type de phrase utile
- ex. `gga_records`, `rmc_records`, `vtg_records`, `dbt_records`, etc.
- plus de travail, mais mieux pour requetage, indexation et export scientifique

- Option 3 : hybride
- table commune d'evenements decodees
- plus tables specialisees pour les phrases prioritaires

### Recommendation actuelle

- Demarrer par une base decodee hybride, avec priorite sur les phrases deja bien gerees par le projet :
- `GGA`
- `RMC`
- `VTG`
- `DBT`
- eventuellement `GLL`, `GSA`, `GSV` plus tard selon l'usage reel

- Stocker :
- horodatage de reception
- nom du device
- transport / port logique si utile
- type de phrase
- champs utiles exploses en colonnes
- eventuellement un JSON complementaire pour les champs non modelises au premier passage

### Questions ouvertes avant implementation complete

- Quelle est la liste des phrases NMEA qui doivent alimenter la base decodee en priorite ?
- Quels exports / post-traitements cibles veut-on produire a partir de cette base ?
- Veut-on une base decodee unique par mission, ou une base par session d'acquisition ?
- Faut-il conserver `decoded_json` dans la base brute si la base decodee existe deja ?
- Faut-il permettre une acquisition sans aucune persistance disque ?

## Etat de maturite de l'idee

- Oui, il y a deja assez d'information pour commencer un premier refactoring de configuration et de persistance.
- Non, il n'y a pas encore assez d'information pour figer proprement le schema final de la base decodee sans risque de refaire une migration juste apres.

## Plan de mise en oeuvre suggere

- Etape 1 : renommer `[acq]` en `[backup]` ou `[storage]`
- Etape 2 : ajouter les booleens `raw` et `processed`
- Etape 3 : separer conceptuellement le store brut et le store decode
- Etape 4 : implementer le store brut compatible avec l'existant
- Etape 5 : definir un premier schema decode minimal sur `GGA/RMC/VTG/DBT`
- Etape 6 : adapter l'export et le post-traitement pour utiliser la base decodee quand disponible
