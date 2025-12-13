package discord

// User represents Discord user info
type User struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

// Guild represents a Discord guild/server
type Guild struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Icon        string `json:"icon"`
	Owner       bool   `json:"owner"`
	Permissions int    `json:"permissions"`
}

// GuildMember represents a user's membership in a guild
type GuildMember struct {
	Roles []string `json:"roles"`
	Nick  string   `json:"nick"`
	User  struct {
		ID       string `json:"id"`
		Username string `json:"username"`
	} `json:"user"`
}
