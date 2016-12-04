package comicplugin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image/png"
	"log"
	"math"
	"math/rand"
	"strconv"
	"strings"
	"sync"

	"github.com/matannoam/comicjerk"
	"github.com/iopred/comicgen"
	"github.com/iopred/discordgo"
)

type comicPlugin struct {
	sync.Mutex

	comicjerk.SimplePlugin
	log    map[string][]comicjerk.Message
	Comics int
}

func (p *comicPlugin) Load(bot *comicjerk.Bot, service comicjerk.Service, data []byte) error {
	if data != nil {
		if err := json.Unmarshal(data, p); err != nil {
			log.Println("Error loading data", err)
		}
	}

	return nil
}

func (p *comicPlugin) Save() ([]byte, error) {
	return json.Marshal(p)
}

func (p *comicPlugin) Help(bot *comicjerk.Bot, service comicjerk.Service, message comicjerk.Message, detailed bool) []string {
	help := comicjerk.CommandHelp(service, "comic", "[1-10]", "Creates a comic from recent messages, or a number of messages if provided.")

	ticks := ""
	if service.Name() == comicjerk.DiscordServiceName {
		ticks = "`"
	}
	if detailed {
		help = append(help, []string{
			comicjerk.CommandHelp(service, "customcomic", "[id|name:] <text> | [id|name:] <text>", fmt.Sprintf("Creates a custom comic. Available names: %s%s%s", ticks, strings.Join(comicgen.CharacterNames, ", "), ticks))[0],
			"Examples:",
			comicjerk.CommandHelp(service, "comic", "5", "Creates a comic from the last 5 messages")[0],
			comicjerk.CommandHelp(service, "customcomic", "A | B | C", "Creates a comic with 3 lines.")[0],
			comicjerk.CommandHelp(service, "customcomic", "0: Hi! | 1: Hello! | 0: Goodbye.", "Creates a comic with 3 lines, the second line spoken by a different character")[0],
			comicjerk.CommandHelp(service, "customcomic", "tiki: Hi! | jordy: Hello! | tiki: Goodbye.", "Creates a comic with 3 lines, containing tiki and jordy.")[0],
		}...)
	}

	return help
}

func makeScriptFromMessages(service comicjerk.Service, message comicjerk.Message, messages []comicjerk.Message) *comicgen.Script {
	speakers := make(map[string]int)
	avatars := make(map[int]string)

	script := []*comicgen.Message{}

	for _, message := range messages {
		speaker, ok := speakers[message.UserName()]
		if !ok {
			speaker = len(speakers)
			speakers[message.UserName()] = speaker
			avatars[speaker] = message.UserAvatar()
		}

		script = append(script, &comicgen.Message{
			Speaker: speaker,
			Text:    message.Message(),
			Author:  message.UserName(),
		})
	}
	return &comicgen.Script{
		Messages: script,
		Author:   fmt.Sprintf(service.UserName()),
		Avatars:  avatars,
		Type:     comicgen.ComicTypeChat,
	}
}

func (p *comicPlugin) makeComic(bot *comicjerk.Bot, service comicjerk.Service, message comicjerk.Message, script *comicgen.Script) {
	p.Comics++
	comic := comicgen.NewComicGen("comic", service.Name() != comicjerk.DiscordServiceName)
	image, err := comic.MakeComic(script)
	if err != nil {
		service.SendMessage(message.Channel(), fmt.Sprintf("Sorry %s, there was an error creating the comic. %s", message.UserName(), err))
	} else {
		go func() {
			b := &bytes.Buffer{}
			err = png.Encode(b, image)
			if err != nil {
				service.SendMessage(message.Channel(), fmt.Sprintf("Sorry %s, there was a problem creating your comic.", message.UserName()))
				return
			}

			if service.Name() == comicjerk.DiscordServiceName {
				discord := service.(*comicjerk.Discord)
				p, err := discord.UserChannelPermissions(message.UserID(), message.Channel())
				if err == nil && p&discordgo.PermissionAttachFiles != 0 {
					service.SendFile(message.Channel(), "comic.png", b)
					return
				}
			}

			url, err := bot.UploadToImgur(b, "comic.png")
			if err != nil {
				service.SendMessage(message.Channel(), fmt.Sprintf("Sorry %s, there was a problem uploading the comic to imgur.", message.UserName()))
				log.Println("Error uploading comic: ", err)
				return
			}

			if service.Name() == comicjerk.DiscordServiceName {
				service.SendMessage(message.Channel(), fmt.Sprintf("Here's your comic <@%s>: %s", message.UserID(), url))
			} else {
				service.SendMessage(message.Channel(), fmt.Sprintf("Here's your comic %s: %s", message.UserName(), url))
			}
		}()
	}
}

func (p *comicPlugin) Message(bot *comicjerk.Bot, service comicjerk.Service, message comicjerk.Message) {
	if service.IsMe(message) {
		return
	}

	p.Lock()
	defer p.Unlock()

	log, ok := p.log[message.Channel()]
	if !ok {
		log = []comicjerk.Message{}
	}

	if comicjerk.MatchesCommand(service, "customcomic", message) {
		ty := comicgen.ComicTypeChat

		service.Typing(message.Channel())

		str, _ := comicjerk.ParseCommand(service, message)

		messages := []*comicgen.Message{}

		splits := strings.Split(str, "|")
		for _, line := range splits {
			line := strings.Trim(line, " ")

			text := ""
			speaker := 0
			author := ""
			if strings.Index(line, ":") != -1 {
				lineSplit := strings.Split(line, ":")

				author = strings.ToLower(strings.Trim(lineSplit[0], " "))

				var err error
				speaker, err = strconv.Atoi(author)
				if err != nil {
					speaker = -1
				}

				text = strings.Trim(lineSplit[1], " ")
			} else {
				text = line
			}

			messages = append(messages, &comicgen.Message{
				Speaker: speaker,
				Text:    text,
				Author:  author,
			})
		}

		if len(messages) == 0 {
			service.SendMessage(message.Channel(), fmt.Sprintf("Sorry %s, you didn't add any text.", message.UserName()))
			return
		}

		p.makeComic(bot, service, message, &comicgen.Script{
			Messages: messages,
			Author:   fmt.Sprintf(service.UserName()),
			Type:     ty,
		})
	} else if comicjerk.MatchesCommand(service, "comic", message) {
		if len(log) == 0 {
			service.SendMessage(message.Channel(), fmt.Sprintf("Sorry %s, I don't have enough messages to make a comic yet.", message.UserName()))
			return
		}

		service.Typing(message.Channel())

		lines := 0
		linesString, parts := comicjerk.ParseCommand(service, message)
		if len(parts) > 0 {
			lines, _ = strconv.Atoi(linesString)
		}

		if lines <= 0 {
			lines = 1 + int(math.Floor((math.Pow(2*rand.Float64()-1, 3)/2+0.5)*float64(5)))
		}

		if lines > len(log) {
			lines = len(log)
		}

		p.makeComic(bot, service, message, makeScriptFromMessages(service, message, log[len(log)-lines:]))
	} else {
		// Don't append commands.
		if strings.HasPrefix(strings.ToLower(strings.Trim(message.Message(), " ")), strings.ToLower(service.CommandPrefix())) {
			return
		}

		switch message.Type() {
		case comicjerk.MessageTypeCreate:
			if len(log) < 10 {
				log = append(log, message)
			} else {
				log = append(log[1:], message)
			}
		case comicjerk.MessageTypeUpdate:
			for i, m := range log {
				if m.MessageID() == message.MessageID() {
					log[i] = message
					break
				}
			}
		case comicjerk.MessageTypeDelete:
			for i, m := range log {
				if m.MessageID() == message.MessageID() {
					log = append(log[:i], log[i+1:]...)
					break
				}
			}
		}
		p.log[message.Channel()] = log
	}
}

func (p *comicPlugin) Name() string {
	return "Comic"
}

// Stats will return the stats for a plugin.
func (p *comicPlugin) Stats(bot *comicjerk.Bot, service comicjerk.Service, message comicjerk.Message) []string {
	return []string{fmt.Sprintf("Comics created: \t%d\n", p.Comics)}
}

// New will create a new comic plugin.
func New() comicjerk.Plugin {
	return &comicPlugin{
		log: make(map[string][]comicjerk.Message),
	}
}
