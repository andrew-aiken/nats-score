package discord

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/oauth2"
)

const (
	apiBaseURL = "https://discord.com/api"
)

// Client handles Discord API operations
type Client struct {
	httpClient *http.Client
}

// NewClient creates a new Discord API client from an OAuth2 token
func NewClient(ctx context.Context, conf *oauth2.Config, token *oauth2.Token) *Client {
	return &Client{
		httpClient: conf.Client(ctx, token),
	}
}

// GetCurrentUser fetches the current user's info
func (c *Client) GetCurrentUser() (*User, error) {
	res, err := c.httpClient.Get(apiBaseURL + "/users/@me")
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get user: %s", res.Status)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var user User
	if err := json.Unmarshal(body, &user); err != nil {
		return nil, err
	}

	return &user, nil
}

// GetUserGuilds fetches the current user's guilds
func (c *Client) GetUserGuilds() ([]Guild, error) {
	res, err := c.httpClient.Get(apiBaseURL + "/users/@me/guilds")
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get guilds: %s", res.Status)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var guilds []Guild
	if err := json.Unmarshal(body, &guilds); err != nil {
		return nil, err
	}

	return guilds, nil
}

// GetGuildMember fetches the current user's membership in a guild
func (c *Client) GetGuildMember(guildID string) (*GuildMember, error) {
	url := fmt.Sprintf("%s/users/@me/guilds/%s/member", apiBaseURL, guildID)
	res, err := c.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get guild member: %s", res.Status)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var member GuildMember
	if err := json.Unmarshal(body, &member); err != nil {
		return nil, err
	}

	return &member, nil
}

// FindGuild finds a guild by ID in a list of guilds
func FindGuild(guilds []Guild, guildID string) *Guild {
	for _, guild := range guilds {
		if guild.ID == guildID {
			return &guild
		}
	}
	return nil
}

// HasRole checks if a role ID is in the list of roles
func HasRole(roles []string, roleID string) bool {
	for _, role := range roles {
		if role == roleID {
			return true
		}
	}
	return false
}

// GetMatchingRoles returns a map of role IDs to names for roles the user has
func GetMatchingRoles(userRoles []string, requiredRoles map[string]string) map[string]string {
	matched := make(map[string]string)
	for _, roleID := range userRoles {
		if roleName, ok := requiredRoles[roleID]; ok {
			matched[roleID] = roleName
		}
	}
	return matched
}
