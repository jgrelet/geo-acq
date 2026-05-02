# Git Worktree Memo

Petit memo pour utiliser `git worktree` sur `geo-acq` sans se melanger entre `master`, `Wails` et `Fyne`.

## Idee simple

Avec les worktrees, on garde plusieurs repertoires de travail en parallele :

- `geo-acq` : depot principal, souvent sur `master`
- `geo-acq-wails` : branche `feature/gui-wails`
- `geo-acq-fyne` : branche `feature/gui-fyne`

Ca evite de faire des `git checkout` en permanence quand on compare plusieurs pistes.

## Voir les worktrees

```bash
git worktree list
```

Exemple attendu :

```bash
/c/github/jgrelet/golang/geo-acq        b101c01 [master]
/c/github/jgrelet/golang/geo-acq-wails  5907fcd [feature/gui-wails]
/c/github/jgrelet/golang/geo-acq-fyne   d443aa3 [feature/gui-fyne]
```

## Travailler sur une branche

On ne fait pas `git checkout feature/gui-wails` depuis `geo-acq` si cette branche est deja ouverte dans un autre worktree.

On se place directement dans le bon repertoire :

```bash
cd /c/github/jgrelet/golang/geo-acq-wails
git status
git branch --show-current
```

Ou :

```bash
cd /c/github/jgrelet/golang/geo-acq-fyne
git status
git branch --show-current
```

## Creer un nouveau worktree

Depuis le depot principal :

```bash
git worktree add ../geo-acq-wails -b feature/gui-wails master
```

Ou :

```bash
git worktree add ../geo-acq-fyne -b feature/gui-fyne master
```

## Committer dans un worktree

Depuis le bon repertoire :

```bash
cd /c/github/jgrelet/golang/geo-acq-wails
git add .
git commit -m "Improve Wails GUI workflow and tabbed monitoring layout"
```

Le commit va bien sur la branche du worktree courant.

## Merger vers master

Quand une branche est prete :

```bash
cd /c/github/jgrelet/golang/geo-acq
git checkout master
git merge feature/gui-wails
```

Ou :

```bash
git merge feature/gui-fyne
```

Le workflow Git reste le meme qu'avant :

1. branche de feature
2. commits
3. merge vers `master`

Le worktree change seulement le confort de travail.

## Supprimer un worktree

Quand on n'en a plus besoin :

```bash
git worktree remove /c/github/jgrelet/golang/geo-acq-fyne
```

Ou :

```bash
git worktree remove /c/github/jgrelet/golang/geo-acq-wails
```

Ensuite, si la branche est deja mergee :

```bash
git branch -d feature/gui-fyne
git branch -d feature/gui-wails
```

## Message d'erreur classique

Si tu vois :

```bash
fatal: 'feature/gui-fyne' is already used by worktree ...
```

Ca veut dire :

- la branche est deja ouverte dans un autre repertoire
- il faut aller dans ce repertoire
- ou supprimer ce worktree
- ou changer la branche active dans ce worktree

## Regle pratique pour ce projet

- `geo-acq` pour `master`
- `geo-acq-wails` pour `feature/gui-wails`
- `geo-acq-fyne` pour `feature/gui-fyne`

Ca suffit largement pour travailler proprement.
