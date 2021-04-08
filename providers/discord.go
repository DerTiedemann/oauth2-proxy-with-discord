// stolen and modified from here: https://github.com/l1n/oauth2-proxy/blob/0efa4c9bdd436cdc408bc73b8d30ebf8931a3520/providers/discord.go
// original credit goes to https://github.com/l1n or https://git.sr.ht/~nova/

package providers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/bwmarrin/discordgo"
	"github.com/oauth2-proxy/oauth2-proxy/v7/pkg/apis/sessions"
	"github.com/oauth2-proxy/oauth2-proxy/v7/pkg/requests"
)

const (
	discordProviderName = "Discord"
	discordDefautScope  = "identify guilds"
)

var (
	// Default Login URL for GitHub.
	// Pre-parsed URL of https://discord.com/api/oauth2/authorize.
	discordDefaultLoginURL = &url.URL{
		Scheme: "https",
		Host:   "discord.com",
		Path:   "/api/oauth2/authorize",
	}

	// Default Redeem URL for GitHub.
	// Pre-parsed URL of https://discord.com/api/oauth2/token.
	discordDefaultRedeemURL = &url.URL{
		Scheme: "https",
		Host:   "discord.com",
		Path:   "/api/oauth2/token",
	}

	// Default profile URL for GitHub.
	// Pre-parsed URL of https://discord.com/api/users/@me.
	discordDefaultProfileURL = &url.URL{
		Scheme: "https",
		Host:   "discord.com",
		Path:   "/api/users/@me",
	}
)

type DiscordProvider struct {
	*ProviderData
}

func NewDiscordProvider(p *ProviderData) *DiscordProvider {
	p.setProviderDefaults(providerDefaults{
		name:        discordProviderName,
		loginURL:    discordDefaultLoginURL,
		redeemURL:   discordDefaultRedeemURL,
		profileURL:  discordDefaultProfileURL,
		validateURL: discordDefaultProfileURL,
		scope:       discordDefautScope,
	})
	p.Prompt = "none"
	return &DiscordProvider{ProviderData: p}
}

func getDiscordHeader(accessToken string) http.Header {
	header := make(http.Header)
	header.Set("Accept", "application/json")
	header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	return header
}

func (p *DiscordProvider) GetLoginURL(redirectURI, state string) string {
	a := *p.LoginURL
	params, _ := url.ParseQuery(a.RawQuery)
	params.Set("redirect_uri", redirectURI)
	params.Add("scope", p.Scope)
	params.Set("client_id", p.ClientID)
	params.Set("response_type", "code")
	params.Add("state", state)
	a.RawQuery = params.Encode()
	fmt.Println(a.String())
	return a.String()
}

func (p *DiscordProvider) EnrichSession(ctx context.Context, s *sessions.SessionState) error {
	if s.AccessToken == "" {
		return errors.New("missing access token")
	}

	// client := &http.Client{}

	discordHeader := getDiscordHeader(s.AccessToken)

	var user discordgo.User
	err := requests.New(p.ProfileURL.String()).
		WithContext(ctx).
		WithHeaders(discordHeader).
		Do().
		UnmarshalInto(&user)
	if err != nil {
		return err
	}

	// req, err := http.NewRequestWithContext(ctx, "GET", p.ProfileURL.String(), nil)
	// if err != nil {
	// 	return err
	// }

	// req.Header = discordHeader
	// res, err := client.Do(req)
	// if err != nil {
	// 	return err
	// }

	// data, err := io.ReadAll(res.Body)
	// if err != nil {
	// 	return err
	// }

	// fmt.Println(string(data))

	// json.Unmarshal(data, &user)

	s.Email = "dummy@example.com"

	/*
		{"id": "802160659160367104",
		"name": "\ud83c\udf49Meloney Squad\ud83c\udf49",
		"icon": "d634d6f7efebffe90ac0ac1f4277d625",
		"owner": false,
		"permissions": 104156737,
		"features": [],
		"permissions_new": "6546607681"}
	*/

	var guilds []struct {
		ID             string        `json:"id"`
		Name           string        `json:"name"`
		Icon           string        `json:"icon"`
		Owner          bool          `json:"owner"`
		Permissions    int           `json:"permissions"`
		Features       []interface{} `json:"features"`
		PermissionsNew int           `json:"permissions_new,string"`
	}
	err = requests.New(p.ProfileURL.String() + "/guilds").
		WithContext(ctx).
		WithHeaders(discordHeader).
		Do().
		UnmarshalInto(&guilds)

	if err != nil {
		return err
	}

	for _, guild := range guilds {
		s.Groups = append(s.Groups, guild.ID)
	}

	fmt.Println(s.Groups)

	return nil
}

func (p *DiscordProvider) ValidateSessionState(ctx context.Context, s *sessions.SessionState) bool {
	return validateToken(ctx, p, s.AccessToken, getDiscordHeader(s.AccessToken))
}
