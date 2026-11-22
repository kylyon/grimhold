# Grimhold

Grimhold is a terminal-based dungeon crawler written in Go. Explore connected regions loaded from `world.json`, inspect monsters fetched from the [Open5e API](https://api.open5e.com/v1/monsters/) (cached in memory with a 5-minute TTL), fight or flee in round-based D&D-style combat, level up your character, and save your progress.

## Build

```sh
go build ./cmd
```

## Run

Start a new game:

```sh
go run ./cmd
```

Resume from `save.json` (starts a new game cleanly if no save exists):

```sh
go run ./cmd --continue
```

## Save & Load

- `save` writes your current location, character stats, gold, slain bestiary, and per-visit rest state to `save.json`.
- `load` restores your saved game from `save.json`.
- `--continue` loads `save.json` automatically on startup.

## Sample Session

```text
grimhold > look
Grimhold Village :
A quiet village surrounded by mist.
I'm feeling safe here
You can go to :
  Dark Forest ( key : dark-forest )
  Ancient Crypt ( key : ancient-crypt )

grimhold > travel dark-forest
grimhold > explore
You explore Dark Forest...
A wild Goblin (CR 1/4, AC 15, HP 7) blocks your path!
Use 'fight' to attack or 'flee' to attempt an escape.

grimhold > fight
You attack Goblin (rolled 14 + 5 = 19 vs AC 15) — hit for 7 damage! (Goblin HP: 0/7)
Victory! You defeated the Goblin!
Gained 50 XP and 4 gold pieces. (Total: 50 XP, 4 gp)
LEVEL UP! You reached Level 2! (Max HP: 38, Attack Bonus: +6)

grimhold > character
=== Character Sheet ===
Level:    2 (XP: 50 / 200)
HP:       38 / 38
AC:       14
Weapon:   Longsword (1d8+3, Attack +6)
Gold:     4 gp
Location: Dark Forest
Slain Bestiary:
  - Goblin x1

grimhold > save
Game saved to save.json.
```
