package grimhold

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"sort"
	"strconv"
	"strings"

	"github.com/kylyon/grimhold/internal/open5e"
	"github.com/kylyon/grimhold/internal/world"
)

// PlayerState tracks the player's combat stats, progression, and slain bestiary.
type PlayerState struct {
	HP              int
	MaxHP           int
	Level           int
	XP              int
	ArmorClass      int
	AttackBonus     int
	Weapon          string
	DamageDice      string
	DamageBonus     int
	Dexterity       int
	Gold            int
	Slain           map[string]int
	RestedThisVisit bool
}

// Encounter represents an active fight with a monster.
type Encounter struct {
	Monster   open5e.Monster
	CurrentHP int
}

var Player = newDefaultPlayer()
var activeEncounter *Encounter

func newDefaultPlayer() PlayerState {
	return PlayerState{
		HP:          30,
		MaxHP:       30,
		Level:       1,
		ArmorClass:  14,
		AttackBonus: 5,
		Weapon:      "Longsword",
		DamageDice:  "1d8",
		DamageBonus: 3,
		Dexterity:   14,
		Slain:       make(map[string]int),
	}
}

// XPToNextLevel returns the total XP required to advance past the given level.
func XPToNextLevel(level int) int {
	if level < 1 {
		level = 1
	}
	return level * level * 50
}

// RollD rolls a single die with the given number of sides.
func RollD(sides int) int {
	if sides <= 1 {
		return 1
	}
	return rand.IntN(sides) + 1
}

// RollDice parses and rolls a dice expression like "2d6", "1d8+3", or "1d6-1".
func RollDice(expr string) (int, error) {
	cleaned := strings.ReplaceAll(strings.ToLower(strings.TrimSpace(expr)), " ", "")
	if cleaned == "" {
		return 0, errors.New("empty dice expression")
	}

	bonus := 0
	dicePart := cleaned
	if idx := strings.IndexAny(cleaned, "+-"); idx != -1 {
		b, err := strconv.Atoi(cleaned[idx:])
		if err != nil {
			return 0, fmt.Errorf("invalid dice bonus in %q: %w", expr, err)
		}
		bonus = b
		dicePart = cleaned[:idx]
	}

	parts := strings.Split(dicePart, "d")
	if len(parts) != 2 {
		return 0, fmt.Errorf("invalid dice expression %q", expr)
	}

	count := 1
	if parts[0] != "" {
		c, err := strconv.Atoi(parts[0])
		if err != nil || c <= 0 {
			return 0, fmt.Errorf("invalid dice count in %q", expr)
		}
		count = c
	}

	sides, err := strconv.Atoi(parts[1])
	if err != nil || sides <= 0 {
		return 0, fmt.Errorf("invalid die sides in %q", expr)
	}

	total := bonus
	for range count {
		total += RollD(sides)
	}
	if total < 1 {
		total = 1
	}
	return total, nil
}

func abilityModValue(score int) int {
	mod := (score - 10) / 2
	if score < 10 && (score-10)%2 != 0 {
		mod--
	}
	return mod
}

// XPForCR returns the D&D 5e XP reward for a given challenge rating.
func XPForCR(cr float64) int {
	table := map[float64]int{
		0:     10,
		0.125: 25,
		0.25:  50,
		0.5:   100,
		1:     200,
		2:     450,
		3:     700,
		4:     1100,
		5:     1800,
		6:     2300,
		7:     2900,
		8:     3900,
		9:     5000,
		10:    5900,
	}
	if xp, ok := table[cr]; ok {
		return xp
	}
	if cr <= 0 {
		return 10
	}
	return int(cr * 600)
}

func pickFoe(region world.Region, fetch func(string) (open5e.Monster, error)) (open5e.Monster, error) {
	if len(region.Foes) == 0 {
		return open5e.Monster{}, fmt.Errorf("there are no foes lurking in %s", region.Name)
	}

	rolls := 1
	if region.Danger > 1 {
		rolls = region.Danger
	}

	var worst open5e.Monster
	for i := range rolls {
		slug := region.Foes[rand.IntN(len(region.Foes))]
		m, err := fetch(slug)
		if err != nil {
			return open5e.Monster{}, fmt.Errorf("fetching foe %q: %w", slug, err)
		}
		if i == 0 || m.CR > worst.CR || (m.CR == worst.CR && m.HitPoints > worst.HitPoints) {
			worst = m
		}
	}
	return worst, nil
}

func monsterAttack(m open5e.Monster) (name string, atkBonus int, dmgDice string, dmgBonus int) {
	for _, a := range m.Actions {
		if a.DamageDice != "" || a.AttackBonus != 0 {
			dice := a.DamageDice
			if dice == "" {
				dice = "1d6"
			}
			return a.Name, a.AttackBonus, dice, a.DamageBonus
		}
	}
	fallbackBonus := max(2, int(m.CR)+2)
	return "Strike", fallbackBonus, "1d6", 1
}

func handleVictory(enc *Encounter) {
	xp := XPForCR(enc.Monster.CR)
	gold := RollD(max(2, int(enc.Monster.CR*8)+4))
	Player.XP += xp
	Player.Gold += gold
	Player.Slain[enc.Monster.Name]++
	activeEncounter = nil

	fmt.Printf("Victory! You defeated the %s!\n", enc.Monster.Name)
	fmt.Printf("Gained %d XP and %d gold pieces. (Total: %d XP, %d gp)\n", xp, gold, Player.XP, Player.Gold)

	for Player.XP >= XPToNextLevel(Player.Level) {
		Player.Level++
		Player.MaxHP += 8
		Player.HP = min(Player.MaxHP, Player.HP+8)
		Player.AttackBonus++
		fmt.Printf("LEVEL UP! You reached Level %d! (Max HP: %d, Attack Bonus: +%d)\n",
			Player.Level, Player.MaxHP, Player.AttackBonus)
	}
}

func handleDefeat() {
	Player.HP = 1
	Player.RestedThisVisit = false
	activeEncounter = nil

	respawn := "crossroads"
	if _, ok := World.Regions[respawn]; !ok {
		respawn = World.Start
	}
	World.CurrentLocation = respawn

	fmt.Println("You collapse as your vision fades to black...")
	fmt.Printf("You wake up at %s with 1 HP and your pride wounded.\n", World.Regions[respawn].Name)
}

func performMonsterStrike(m open5e.Monster) {
	atkName, atkBonus, dmgDice, dmgBonus := monsterAttack(m)
	roll := RollD(20)
	total := roll + atkBonus

	if roll == 1 || (roll != 20 && total < Player.ArmorClass) {
		fmt.Printf("%s uses %s (rolled %d + %d = %d vs AC %d) — misses!\n",
			m.Name, atkName, roll, atkBonus, total, Player.ArmorClass)
		return
	}

	dmg, err := RollDice(dmgDice)
	if err != nil {
		dmg = RollD(6)
	}
	dmg += dmgBonus
	if roll == 20 {
		bonusCrit, err := RollDice(dmgDice)
		if err != nil {
			bonusCrit = RollD(6)
		}
		dmg += bonusCrit
	}
	if dmg < 1 {
		dmg = 1
	}

	Player.HP -= dmg
	if Player.HP < 0 {
		Player.HP = 0
	}
	critNote := ""
	if roll == 20 {
		critNote = " (CRITICAL HIT!)"
	}
	fmt.Printf("%s uses %s (rolled %d + %d = %d vs AC %d)%s — hits for %d damage! (Your HP: %d/%d)\n",
		m.Name, atkName, roll, atkBonus, total, Player.ArmorClass, critNote, dmg, Player.HP, Player.MaxHP)

	if Player.HP <= 0 {
		handleDefeat()
	}
}

func callbackExplore(args []string) error {
	if activeEncounter != nil {
		return fmt.Errorf("you are already fighting a %s (HP: %d/%d) — use 'fight' or 'flee'",
			activeEncounter.Monster.Name, activeEncounter.CurrentHP, activeEncounter.Monster.HitPoints)
	}

	region := World.Regions[World.CurrentLocation]
	foe, err := pickFoe(region, apiClient.GetMonster)
	if err != nil {
		return err
	}

	activeEncounter = &Encounter{
		Monster:   foe,
		CurrentHP: foe.HitPoints,
	}

	fmt.Printf("You explore %s...\n", region.Name)
	fmt.Printf("A wild %s (CR %s, AC %d, HP %d) blocks your path!\n",
		foe.Name, foe.ChallengeRating, foe.ArmorClass, foe.HitPoints)
	fmt.Println("Use 'fight' to attack or 'flee' to attempt an escape.")
	return nil
}

func callbackFight(args []string) error {
	if activeEncounter == nil {
		return errors.New("you are not in combat — use 'explore' to look for trouble")
	}

	enc := activeEncounter
	roll := RollD(20)
	total := roll + Player.AttackBonus

	if roll == 1 || (roll != 20 && total < enc.Monster.ArmorClass) {
		fmt.Printf("You attack %s (rolled %d + %d = %d vs AC %d) — miss!\n",
			enc.Monster.Name, roll, Player.AttackBonus, total, enc.Monster.ArmorClass)
	} else {
		dmg, err := RollDice(Player.DamageDice)
		if err != nil {
			dmg = RollD(8)
		}
		dmg += Player.DamageBonus
		if roll == 20 {
			bonusCrit, err := RollDice(Player.DamageDice)
			if err != nil {
				bonusCrit = RollD(8)
			}
			dmg += bonusCrit
		}
		if dmg < 1 {
			dmg = 1
		}

		enc.CurrentHP -= dmg
		if enc.CurrentHP < 0 {
			enc.CurrentHP = 0
		}
		critNote := ""
		if roll == 20 {
			critNote = " (CRITICAL HIT!)"
		}
		fmt.Printf("You attack %s (rolled %d + %d = %d vs AC %d)%s — hit for %d damage! (%s HP: %d/%d)\n",
			enc.Monster.Name, roll, Player.AttackBonus, total, enc.Monster.ArmorClass, critNote, dmg,
			enc.Monster.Name, enc.CurrentHP, enc.Monster.HitPoints)
	}

	if enc.CurrentHP <= 0 {
		handleVictory(enc)
		return nil
	}

	performMonsterStrike(enc.Monster)
	return nil
}

func callbackFlee(args []string) error {
	if activeEncounter == nil {
		return errors.New("you are not in combat")
	}

	enc := activeEncounter
	playerDex := abilityModValue(Player.Dexterity)
	foeDex := abilityModValue(enc.Monster.Dexterity)
	dc := 10 + foeDex

	roll := RollD(20)
	total := roll + playerDex

	if total >= dc {
		activeEncounter = nil
		fmt.Printf("You flee from %s! (DEX check %d + %d = %d vs DC %d)\n",
			enc.Monster.Name, roll, playerDex, total, dc)
		return nil
	}

	fmt.Printf("You fail to escape %s! (DEX check %d + %d = %d vs DC %d)\n",
		enc.Monster.Name, roll, playerDex, total, dc)
	performMonsterStrike(enc.Monster)
	return nil
}

func callbackCharacter(args []string) error {
	fmt.Println("=== Character Sheet ===")
	fmt.Printf("Level:    %d (XP: %d / %d)\n", Player.Level, Player.XP, XPToNextLevel(Player.Level))
	fmt.Printf("HP:       %d / %d\n", Player.HP, Player.MaxHP)
	fmt.Printf("AC:       %d\n", Player.ArmorClass)
	fmt.Printf("Weapon:   %s (%s+%d, Attack +%d)\n", Player.Weapon, Player.DamageDice, Player.DamageBonus, Player.AttackBonus)
	fmt.Printf("Gold:     %d gp\n", Player.Gold)
	fmt.Printf("Location: %s\n", World.Regions[World.CurrentLocation].Name)

	if len(Player.Slain) == 0 {
		fmt.Println("Slain Bestiary: (none yet)")
		return nil
	}

	names := make([]string, 0, len(Player.Slain))
	for name := range Player.Slain {
		names = append(names, name)
	}
	sort.Strings(names)

	fmt.Println("Slain Bestiary:")
	for _, name := range names {
		fmt.Printf("  - %s x%d\n", name, Player.Slain[name])
	}
	return nil
}

func callbackStatus(args []string) error {
	return callbackCharacter(args)
}

func callbackRest(args []string) error {
	if activeEncounter != nil {
		return errors.New("you cannot rest while in combat")
	}
	if Player.RestedThisVisit {
		return errors.New("you have already rested during this region visit — travel to another region first")
	}
	Player.HP = Player.MaxHP
	Player.RestedThisVisit = true
	fmt.Printf("You rest and recover to %d/%d HP.\n", Player.HP, Player.MaxHP)
	return nil
}
