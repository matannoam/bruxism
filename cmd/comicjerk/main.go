package main

import (
	"flag"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/matannoam/comicjerk"
	"github.com/matannoam/comicjerk/carbonitexplugin"
	"github.com/matannoam/comicjerk/chartplugin"
	"github.com/matannoam/comicjerk/comicplugin"
	"github.com/matannoam/comicjerk/directmessageinviteplugin"
	"github.com/matannoam/comicjerk/discordavatarplugin"
	"github.com/matannoam/comicjerk/inviteplugin"
	"github.com/matannoam/comicjerk/reminderplugin"
	"github.com/matannoam/comicjerk/statsplugin"
)

var discordToken string
var discordEmail string
var discordPassword string
var discordApplicationClientID string
var discordOwnerUserID string
var discordShards int
var ircServer string
var ircUsername string
var ircPassword string
var ircChannels string
var slackToken string
var slackOwnerUserID string
var imgurID string
var imgurAlbum string
var mashableKey string
var carbonitexKey string

func init() {
	flag.StringVar(&discordToken, "discordtoken", "", "Discord token.")
	flag.StringVar(&discordEmail, "discordemail", "", "Discord account email.")
	flag.StringVar(&discordPassword, "discordpassword", "", "Discord account password.")
	flag.StringVar(&discordOwnerUserID, "discordowneruserid", "", "Discord owner user id.")
	flag.StringVar(&discordApplicationClientID, "discordapplicationclientid", "", "Discord application client id.")
	flag.IntVar(&discordShards, "discordshards", 1, "Number of discord shards.")
	flag.StringVar(&ircServer, "ircserver", "", "IRC server.")
	flag.StringVar(&ircUsername, "ircusername", "", "IRC user name.")
	flag.StringVar(&ircPassword, "ircpassword", "", "IRC password.")
	flag.StringVar(&ircChannels, "ircchannels", "", "Comma separated list of IRC channels.")
	flag.StringVar(&slackToken, "slacktoken", "", "Slack token.")
	flag.StringVar(&slackOwnerUserID, "slackowneruserid", "", "Slack owner user id.")
	flag.StringVar(&imgurID, "imgurid", "", "Imgur client id.")
	flag.StringVar(&imgurAlbum, "imguralbum", "", "Imgur album id.")
	flag.StringVar(&mashableKey, "mashablekey", "", "Mashable key.")
	flag.StringVar(&carbonitexKey, "carbonitexkey", "", "Carbonitex key for discord server count tracking.")
	flag.Parse()

	rand.Seed(time.Now().UnixNano())
}

func main() {
	q := make(chan bool)

	// Set our variables.
	bot := comicjerk.NewBot()
	bot.ImgurID = imgurID
	bot.ImgurAlbum = imgurAlbum
	bot.MashableKey = mashableKey

	// Generally CommandPlugins don't hold state, so we share one instance of the command plugin for all services.
	cp := comicjerk.NewCommandPlugin()
	cp.AddCommand("invite", inviteplugin.InviteCommand, inviteplugin.InviteHelp)
	cp.AddCommand("join", inviteplugin.InviteCommand, nil)
	cp.AddCommand("stats", statsplugin.StatsCommand, statsplugin.StatsHelp)
	cp.AddCommand("info", statsplugin.StatsCommand, nil)
	cp.AddCommand("stat", statsplugin.StatsCommand, nil)
	cp.AddCommand("quit", func(bot *comicjerk.Bot, service comicjerk.Service, message comicjerk.Message, args string, parts []string) {
		if service.IsBotOwner(message) {
			q <- true
		}
	}, nil)

	if (discordEmail != "" && discordPassword != "") || discordToken != "" {
		var discord *comicjerk.Discord
		if discordToken != "" {
			discord = comicjerk.NewDiscord(discordToken)
		} else {
			discord = comicjerk.NewDiscord(discordEmail, discordPassword)
		}
		discord.ApplicationClientID = discordApplicationClientID
		discord.OwnerUserID = discordOwnerUserID
		discord.Shards = discordShards
		bot.RegisterService(discord)

		bot.RegisterPlugin(discord, cp)
		bot.RegisterPlugin(discord, chartplugin.New())
		bot.RegisterPlugin(discord, comicplugin.New())
		bot.RegisterPlugin(discord, directmessageinviteplugin.New())
		bot.RegisterPlugin(discord, reminderplugin.New())
		bot.RegisterPlugin(discord, discordavatarplugin.New())
		if carbonitexKey != "" {
			bot.RegisterPlugin(discord, carbonitexplugin.New(carbonitexKey))
		}
	}

	// Register the IRC service if we have an IRC server and Username.
	if ircServer != "" && ircUsername != "" {
		irc := comicjerk.NewIRC(ircServer, ircUsername, ircPassword, strings.Split(ircChannels, ","))
		bot.RegisterService(irc)

		bot.RegisterPlugin(irc, cp)
		bot.RegisterPlugin(irc, chartplugin.New())
		bot.RegisterPlugin(irc, comicplugin.New())
		bot.RegisterPlugin(irc, reminderplugin.New())
	}

	if slackToken != "" {
		slack := comicjerk.NewSlack(slackToken)
		slack.OwnerUserID = slackOwnerUserID
		bot.RegisterService(slack)

		bot.RegisterPlugin(slack, cp)
	}

	// Start all our services.
	bot.Open()

	// Wait for a termination signal, while saving the bot state every minute. Save on close.
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)

	t := time.Tick(1 * time.Minute)

out:
	for {
		select {
		case <-q:
			break out
		case <-c:
			break out
		case <-t:
			bot.Save()
		}
	}

	bot.Save()
	bot.Close()
}
