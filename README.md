# ComicJerk
A comic bot for Discord, IRC, and Slack. Originally forked from [iopred/bruxism](https://github.com/iopred/bruxism)

## Current plugin support:

Commands are prefixed with `@BotName `.

* `comic [1-10]` - Generates a comic from messages in the chat
* `help [<topic>]` - Returns generic help or help for a specific topic. Available topics: `comic,remind`
* `invite <id>` - Provides invite URL for the bot.
* `reminder <time> | <reminder>` - Sets a reminder.
* `stats` - Lists bot statistics.

eg: `@BotName help`

Also supports direct invites on Discord

## Usage:

### Installation:

`go get github.com/matannoam/comicjerk/cmd/comicjerk`

`go install github.com/matannoam/comicjerk/cmd/comicjerk`

`cd $GOPATH/bin`

### Run as a Discord bot

`comicjerk -discordtoken "Bot <discord bot token>"`

It is suggested that you set `-discordapplicationclientid` if you are running a bot account, this will make `inviteplugin` function correctly.

It is suggested that you set `-discordowneruserid` as this prevents anyone from calling `playingplugin`.

To invite your bot to a server, visit: `https://discordapp.com/oauth2/authorize?client_id=<discord client id>&scope=bot`

### Run as an IRC bot

`comicjerk -ircserver <irc server> -ircusername <irc username> -ircchannels <#channel1,#channel2>`

## Arguments:

* `discordtoken` - Sets the Discord token.
* `discordemail` - Sets the Discord account email.
* `discordpassword` - Sets the Discord account password.
* `discordclientid` - Sets the Discord client id.
* `ircserver` - Sets the IRC server.
* `ircusername` - Sets the IRC user name.
* `ircpassword` - Sets the IRC password.
* `ircchannels` - Comma separated list of IRC channels.
* `imgurid` - Sets the Imgur client id, used for uploading images to imgur.
* `imgurAlbum` - Sets an optional the Imgur album id, used for uploading images to imgur.
* `mashablekey` - Sets the mashable oauth key.

## Special Thanks

[Bruce Marriner](https://github.com/bwmarrin/discordgo) - For DiscordGo.
