package grimhold

import (
	"testing"

	"github.com/kylyon/grimhold/internal/open5e"
	"github.com/kylyon/grimhold/internal/world"
)

func TestRollDice(t *testing.T) {
	for range 20 {
		got, err := RollDice("2d6+3")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got < 5 || got > 15 {
			t.Fatalf("2d6+3 out of range [5,15]: %d", got)
		}
	}

	if _, err := RollDice("invalid"); err == nil {
		t.Fatal("expected error for invalid dice expression")
	}
}

func TestPickFoeDangerKeepsWorst(t *testing.T) {
	region := world.Region{
		Name:   "Crypt",
		Danger: 3,
		Foes:   []string{"goblin", "ghoul"},
	}

	seen := map[string]int{}
	fetch := func(slug string) (open5e.Monster, error) {
		seen[slug]++
		if slug == "ghoul" {
			return open5e.Monster{Slug: "ghoul", Name: "Ghoul", CR: 1, HitPoints: 22}, nil
		}
		return open5e.Monster{Slug: "goblin", Name: "Goblin", CR: 0.25, HitPoints: 7}, nil
	}

	m, err := pickFoe(region, fetch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total := seen["goblin"] + seen["ghoul"]; total != 3 {
		t.Fatalf("expected 3 rolls for danger 3, got %d", total)
	}
	if seen["ghoul"] > 0 && m.Slug != "ghoul" {
		t.Fatalf("expected worst foe ghoul when rolled, got %s", m.Slug)
	}
}

func TestVictoryAndDefeat(t *testing.T) {
	Player = newDefaultPlayer()
	World.CurrentLocation = "ashen-marsh"

	enc := &Encounter{
		Monster:   open5e.Monster{Name: "Goblin", CR: 0.25, HitPoints: 7},
		CurrentHP: 0,
	}
	handleVictory(enc)

	if Player.XP != 50 {
		t.Fatalf("expected 50 XP for CR 1/4, got %d", Player.XP)
	}
	if Player.Level != 2 {
		t.Fatalf("expected level up to 2 at 50 XP, got %d", Player.Level)
	}
	if Player.MaxHP != 38 || Player.AttackBonus != 6 {
		t.Fatalf("expected MaxHP=38 and AttackBonus=6 after level up, got MaxHP=%d AttackBonus=%d",
			Player.MaxHP, Player.AttackBonus)
	}
	if Player.Gold <= 0 {
		t.Fatalf("expected positive gold reward, got %d", Player.Gold)
	}
	if Player.Slain["Goblin"] != 1 {
		t.Fatalf("expected Goblin kill count 1, got %d", Player.Slain["Goblin"])
	}
	if activeEncounter != nil {
		t.Fatal("expected activeEncounter to be cleared after victory")
	}

	Player.HP = 0
	activeEncounter = &Encounter{Monster: open5e.Monster{Name: "Ghoul"}, CurrentHP: 10}
	handleDefeat()

	if Player.HP != 1 {
		t.Fatalf("expected 1 HP after defeat, got %d", Player.HP)
	}
	if World.CurrentLocation != "crossroads" {
		t.Fatalf("expected respawn at crossroads, got %q", World.CurrentLocation)
	}
	if activeEncounter != nil {
		t.Fatal("expected activeEncounter to be cleared after defeat")
	}
}

func TestRestOncePerVisit(t *testing.T) {
	Player = newDefaultPlayer()
	activeEncounter = nil
	Player.HP = 10

	if err := callbackRest(nil); err != nil {
		t.Fatalf("expected first rest to succeed, got %v", err)
	}
	if Player.HP != Player.MaxHP {
		t.Fatalf("expected full heal to %d, got %d", Player.MaxHP, Player.HP)
	}

	if err := callbackRest(nil); err == nil {
		t.Fatal("expected second rest in same visit to fail")
	}

	Player.RestedThisVisit = false
	Player.HP = 15
	if err := callbackRest(nil); err != nil {
		t.Fatalf("expected rest after new visit to succeed, got %v", err)
	}
}
