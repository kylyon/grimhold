package open5e

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/kylyon/grimhold/internal/cache"
)

const BaseMonstersURL = "https://api.open5e.com/v1/monsters/"

// Action represents a monster's special ability, action, or legendary action.
type Action struct {
	Name        string `json:"name"`
	Desc        string `json:"desc"`
	AttackBonus int    `json:"attack_bonus"`
	DamageDice  string `json:"damage_dice"`
	DamageBonus int    `json:"damage_bonus"`
}

// Monster holds the decoded stat block from the Open5e monsters API.
type Monster struct {
	Slug             string         `json:"slug"`
	Name             string         `json:"name"`
	Size             string         `json:"size"`
	Type             string         `json:"type"`
	Alignment        string         `json:"alignment"`
	ArmorClass       int            `json:"armor_class"`
	ArmorDesc        string         `json:"armor_desc"`
	HitPoints        int            `json:"hit_points"`
	HitDice          string         `json:"hit_dice"`
	Speed            map[string]any `json:"speed"`
	Strength         int            `json:"strength"`
	Dexterity        int            `json:"dexterity"`
	Constitution     int            `json:"constitution"`
	Intelligence     int            `json:"intelligence"`
	Wisdom           int            `json:"wisdom"`
	Charisma         int            `json:"charisma"`
	Senses           string         `json:"senses"`
	Languages        string         `json:"languages"`
	ChallengeRating  string         `json:"challenge_rating"`
	CR               float64        `json:"cr"`
	SpecialAbilities []Action       `json:"special_abilities"`
	Actions          []Action       `json:"actions"`
	LegendaryActions []Action       `json:"legendary_actions"`
}

// MonsterListResponse represents a paginated page of monsters from Open5e.
type MonsterListResponse struct {
	Count    int       `json:"count"`
	Next     *string   `json:"next"`
	Previous *string   `json:"previous"`
	Results  []Monster `json:"results"`
}

// Client fetches monster data from Open5e with an in-memory TTL cache.
type Client struct {
	httpClient *http.Client
	cache      *cache.Cache
}

// NewClient creates an Open5e API client with the given HTTP timeout and cache TTL.
func NewClient(timeout, cacheTTL time.Duration) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: timeout},
		cache:      cache.NewCache(cacheTTL),
	}
}

func (c *Client) get(rawURL string) ([]byte, error) {
	if cached, ok := c.cache.Get(rawURL); ok {
		return cached, nil
	}

	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request for %s: %w", rawURL, err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("requesting %s: %w", rawURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("open5e API returned status %d for %s", resp.StatusCode, rawURL)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response from %s: %w", rawURL, err)
	}

	c.cache.Add(rawURL, data)
	return data, nil
}

// ListMonsters fetches a page of monsters. If pageURL is empty, it fetches the first page.
func (c *Client) ListMonsters(pageURL string) (MonsterListResponse, error) {
	if pageURL == "" {
		pageURL = BaseMonstersURL + "?limit=20"
	}

	data, err := c.get(pageURL)
	if err != nil {
		return MonsterListResponse{}, err
	}

	var list MonsterListResponse
	if err := json.Unmarshal(data, &list); err != nil {
		return MonsterListResponse{}, err
	}
	return list, nil
}

// GetMonster fetches a single monster by slug or name.
func (c *Client) GetMonster(nameOrSlug string) (Monster, error) {
	trimmed := strings.TrimSpace(nameOrSlug)
	if trimmed == "" {
		return Monster{}, fmt.Errorf("monster name cannot be empty")
	}

	slug := toSlug(trimmed)
	directURL := BaseMonstersURL + url.PathEscape(slug) + "/"
	if data, err := c.get(directURL); err == nil {
		var m Monster
		if err := json.Unmarshal(data, &m); err == nil && m.Name != "" {
			return m, nil
		}
	}

	// Fallback to search endpoint if the exact slug didn't match.
	searchURL := BaseMonstersURL + "?search=" + url.QueryEscape(trimmed)
	data, err := c.get(searchURL)
	if err != nil {
		return Monster{}, err
	}

	var list MonsterListResponse
	if err := json.Unmarshal(data, &list); err != nil {
		return Monster{}, err
	}
	if len(list.Results) == 0 {
		return Monster{}, fmt.Errorf("monster %q not found", trimmed)
	}

	for _, m := range list.Results {
		if strings.EqualFold(m.Name, trimmed) || strings.EqualFold(m.Slug, slug) {
			return m, nil
		}
	}
	return list.Results[0], nil
}

func toSlug(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, "_", "-")
	return strings.Join(strings.Fields(s), "-")
}
