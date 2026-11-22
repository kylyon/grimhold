package grimhold

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/kylyon/grimhold/internal/open5e"
	"github.com/kylyon/grimhold/internal/world"
)

var Commands map[string]CliCommand
var World world.World
var apiClient *open5e.Client
var nextPageURL *string
var prevPageURL *string

type CliCommand struct {
	Name        string
	Description string
	Callback    func(args []string) error
}

func callbackHelp(args []string) error {
	fmt.Println("Grimhold — a dungeon crawler")
	fmt.Printf("\nCommands : \n")

	for _, v := range Commands {
		fmt.Printf("  %s \t %s\n", v.Name, v.Description)
	}

	return nil
}

func callbackMap(args []string) error {
	for _, r := range World.Regions {

		fmt.Printf("%s ( key : %s )\n", r.Name, r.ID)
		fmt.Printf("  Danger level : %d\n", r.Danger)
		for _, e := range r.Exits {
			fmt.Printf("  Exits : %s ( key : %s )\n", World.Regions[e].Name, e)
		}
	}
	return nil
}

func callbackTravel(args []string) error {
	if activeEncounter != nil {
		return errors.New("you cannot travel while in combat — use 'fight' or 'flee'")
	}
	if len(args) <= 0 {
		return errors.New("You must specify a region to travel")
	}

	_, ok := World.Regions[args[0]]

	if !ok {
		return errors.New("Invalid region")
	}

	currentLocation := World.CurrentLocation
	if !slices.Contains(World.Regions[currentLocation].Exits, args[0]) {
		return errors.New("You can't travel to that region")
	}

	World.CurrentLocation = args[0]
	Player.RestedThisVisit = false
	return nil
}

func callbackLook(args []string) error {
	currentLocation := World.CurrentLocation
	fmt.Printf("%s :\n", World.Regions[World.CurrentLocation].Name)
	fmt.Println(World.Regions[currentLocation].Description)
	switch World.Regions[currentLocation].Danger {
	case 1:
		fmt.Println("I feel something strange")
	case 2:
		fmt.Println("Danger is surrounding me")
	default:
		fmt.Println("I'm feeling safe here")
	}
	fmt.Printf("You can go to :\n")
	for _, v := range World.Regions[currentLocation].Exits {
		fmt.Printf("  %s ( key : %s )\n", World.Regions[v].Name, v)
	}

	return nil
}

func fetchAndPrintBestiaryPage(pageURL string) error {
	resp, err := apiClient.ListMonsters(pageURL)
	if err != nil {
		return err
	}

	nextPageURL = resp.Next
	prevPageURL = resp.Previous

	fmt.Printf("Bestiary (%d monsters total):\n", resp.Count)
	for _, m := range resp.Results {
		fmt.Printf("  - %s (slug: %s, CR %s, HP %d, AC %d)\n", m.Name, m.Slug, m.ChallengeRating, m.HitPoints, m.ArmorClass)
	}

	if prevPageURL != nil || nextPageURL != nil {
		fmt.Println("\nUse 'next' or 'prev' to browse pages, or 'inspect <monster>' for details.")
	}
	return nil
}

func callbackBestiary(args []string) error {
	if len(args) > 0 {
		switch strings.ToLower(args[0]) {
		case "next":
			return callbackNext(nil)
		case "prev", "previous":
			return callbackPrev(nil)
		default:
			return fmt.Errorf("unknown bestiary subcommand %q (use 'bestiary', 'next', or 'prev')", args[0])
		}
	}
	return fetchAndPrintBestiaryPage("")
}

func callbackNext(args []string) error {
	if nextPageURL == nil {
		return errors.New("you are already on the last page (or run 'bestiary' first)")
	}
	return fetchAndPrintBestiaryPage(*nextPageURL)
}

func callbackPrev(args []string) error {
	if prevPageURL == nil {
		return errors.New("you are already on the first page (or run 'bestiary' first)")
	}
	return fetchAndPrintBestiaryPage(*prevPageURL)
}

func abilityMod(score int) string {
	mod := (score - 10) / 2
	if score < 10 && (score-10)%2 != 0 {
		mod--
	}
	if mod >= 0 {
		return fmt.Sprintf("+%d", mod)
	}
	return fmt.Sprintf("%d", mod)
}

func formatSpeed(speed map[string]any) string {
	if len(speed) == 0 {
		return "none"
	}
	keys := make([]string, 0, len(speed))
	for k := range speed {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		switch v := speed[k].(type) {
		case float64:
			parts = append(parts, fmt.Sprintf("%s %d ft.", k, int(v)))
		case int:
			parts = append(parts, fmt.Sprintf("%s %d ft.", k, v))
		case bool:
			if v {
				parts = append(parts, k)
			}
		default:
			parts = append(parts, fmt.Sprintf("%s %v", k, v))
		}
	}
	if len(parts) == 0 {
		return "none"
	}
	return strings.Join(parts, ", ")
}

func printActionSection(title string, actions []open5e.Action) {
	if len(actions) == 0 {
		return
	}
	fmt.Printf("\n%s:\n", title)
	for _, a := range actions {
		fmt.Printf("  * %s: %s\n", a.Name, a.Desc)
	}
}

func callbackInspect(args []string) error {
	if len(args) == 0 {
		return errors.New("you must specify a monster name or slug (e.g. 'inspect goblin')")
	}

	m, err := apiClient.GetMonster(strings.Join(args, " "))
	if err != nil {
		return err
	}

	fmt.Printf("=== %s ===\n", m.Name)
	fmt.Printf("%s %s, %s\n", m.Size, m.Type, m.Alignment)
	if m.ArmorDesc != "" {
		fmt.Printf("Armor Class: %d (%s)\n", m.ArmorClass, m.ArmorDesc)
	} else {
		fmt.Printf("Armor Class: %d\n", m.ArmorClass)
	}
	if m.HitDice != "" {
		fmt.Printf("Hit Points:  %d (%s)\n", m.HitPoints, m.HitDice)
	} else {
		fmt.Printf("Hit Points:  %d\n", m.HitPoints)
	}
	fmt.Printf("Speed:       %s\n", formatSpeed(m.Speed))
	fmt.Printf("Abilities:   STR %d (%s) | DEX %d (%s) | CON %d (%s) | INT %d (%s) | WIS %d (%s) | CHA %d (%s)\n",
		m.Strength, abilityMod(m.Strength),
		m.Dexterity, abilityMod(m.Dexterity),
		m.Constitution, abilityMod(m.Constitution),
		m.Intelligence, abilityMod(m.Intelligence),
		m.Wisdom, abilityMod(m.Wisdom),
		m.Charisma, abilityMod(m.Charisma),
	)
	if m.Senses != "" {
		fmt.Printf("Senses:      %s\n", m.Senses)
	}
	if m.Languages != "" {
		fmt.Printf("Languages:   %s\n", m.Languages)
	}
	fmt.Printf("Challenge:   %s\n", m.ChallengeRating)

	printActionSection("Special Abilities", m.SpecialAbilities)
	printActionSection("Actions", m.Actions)
	printActionSection("Legendary Actions", m.LegendaryActions)

	return nil
}

func HandleCommand(command string) error {
	args := strings.Fields(command)

	if len(args) == 0 {
		return errors.New("Empty commmand")
	}

	value, ok := Commands[args[0]]
	if ok {
		if err := value.Callback(args[1:]); err != nil {
			return fmt.Errorf("command %q: %w", args[0], err)
		}
		return nil
	}

	return errors.New("Invalid commmand")
}

func init() {
	var err error
	World, err = world.Load("world.json")
	if errors.Is(err, os.ErrNotExist) {
		World, err = world.Load(filepath.Join("..", "..", "world.json"))
	}

	if err != nil {
		panic("Failed to load world: " + err.Error())
	}
	World.CurrentLocation = World.Start

	apiClient = open5e.NewClient(10*time.Second, 5*time.Minute)

	Commands = map[string]CliCommand{
		"help": {
			Name:        "help",
			Description: "Show this message",
			Callback:    callbackHelp,
		},
		"map": {
			Name:        "map",
			Description: "List the known regions",
			Callback:    callbackMap,
		},
		"travel": {
			Name:        "travel",
			Description: "Travel to a connected region",
			Callback:    callbackTravel,
		},
		"look": {
			Name:        "look",
			Description: "Describe your surroundings",
			Callback:    callbackLook,
		},
		"explore": {
			Name:        "explore",
			Description: "Wander the region (risk an encounter)",
			Callback:    callbackExplore,
		},
		"fight": {
			Name:        "fight",
			Description: "Attack the current foe in combat",
			Callback:    callbackFight,
		},
		"flee": {
			Name:        "flee",
			Description: "Attempt to escape from combat",
			Callback:    callbackFlee,
		},
		"character": {
			Name:        "character",
			Description: "Print your character sheet",
			Callback:    callbackCharacter,
		},
		"status": {
			Name:        "status",
			Description: "Show your HP, XP, gold, and slain bestiary",
			Callback:    callbackStatus,
		},
		"rest": {
			Name:        "rest",
			Description: "Recover full HP in a safe region",
			Callback:    callbackRest,
		},
		"bestiary": {
			Name:        "bestiary",
			Description: "List monsters page by page",
			Callback:    callbackBestiary,
		},
		"next": {
			Name:        "next",
			Description: "Show the next page of the bestiary",
			Callback:    callbackNext,
		},
		"prev": {
			Name:        "prev",
			Description: "Show the previous page of the bestiary",
			Callback:    callbackPrev,
		},
		"inspect": {
			Name:        "inspect",
			Description: "Inspect a monster's stat block",
			Callback:    callbackInspect,
		},
		"save": {
			Name:        "save",
			Description: "Save your game progress to save.json",
			Callback:    callbackSave,
		},
		"load": {
			Name:        "load",
			Description: "Load your saved game from save.json",
			Callback:    callbackLoad,
		},
	}
}
