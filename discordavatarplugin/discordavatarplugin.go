package discordavatarplugin

import (
	"regexp"
	"strings"

	"github.com/matannoam/comicjerk"
	"github.com/iopred/discordgo"
)

var userIDRegex = regexp.MustCompile("<@!?([0-9]*)>")

func avatarMessageFunc(bot *comicjerk.Bot, service comicjerk.Service, message comicjerk.Message) {
	if service.Name() == comicjerk.DiscordServiceName && !service.IsMe(message) {
		if comicjerk.MatchesCommand(service, "avatar", message) {
			query := strings.Join(strings.Split(message.RawMessage(), " ")[1:], " ")

			id := message.UserID()
			match := userIDRegex.FindStringSubmatch(query)
			if match != nil {
				id = match[1]
			}

			discord := service.(*comicjerk.Discord)

			u, err := discord.Session.User(id)
			if err != nil {
				return
			}

			service.SendMessage(message.Channel(), discordgo.EndpointUserAvatar(u.ID, u.Avatar))
		}
	}
}

func avatarHelpFunc(bot *comicjerk.Bot, service comicjerk.Service, message comicjerk.Message, detailed bool) []string {
	if detailed {
		return nil
	}
	return comicjerk.CommandHelp(service, "avatar", "[@username]", "Returns a big version of your avatar, or a users avatar if provided.")
}

// New creates a new discordavatar plugin.
func New() comicjerk.Plugin {
	p := comicjerk.NewSimplePlugin("discordavatar")
	p.MessageFunc = avatarMessageFunc
	p.HelpFunc = avatarHelpFunc
	return p
}
