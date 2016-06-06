package playedplugin

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/iopred/bruxism"
	"github.com/iopred/discordgo"
	"github.com/syndtr/goleveldb/leveldb"
	msgpack "gopkg.in/vmihailenco/msgpack.v2"
)

type playedEntry struct {
	Name     string
	Duration time.Duration
}

type byDuration []*playedEntry

func (a byDuration) Len() int           { return len(a) }
func (a byDuration) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byDuration) Less(i, j int) bool { return a[i].Duration >= a[j].Duration }

type playedUser struct {
	Entries     map[string]*playedEntry
	Current     string
	LastChanged time.Time
	FirstSeen   time.Time
}

func (p *playedUser) Update(name string, now time.Time) {
	if p.Current != "" {
		pe := p.Entries[p.Current]
		if pe == nil {
			pe = &playedEntry{
				Name: p.Current,
			}
			p.Entries[p.Current] = pe
		}
		pe.Duration += now.Sub(p.LastChanged)
	}

	p.Current = name
	p.LastChanged = now
}

type oldData struct {
	Users map[string]*playedUser
}

type playedPlugin struct {
	sync.RWMutex
	db    *leveldb.DB
	queue []*discordgo.Presence
}

// Load will load plugin state from a byte array.
func (p *playedPlugin) Load(bot *bruxism.Bot, service bruxism.Service, data []byte) error {
	if service.Name() != bruxism.DiscordServiceName {
		panic("Carbonitex Plugin only supports Discord.")
	}

	dbFilename := service.Name() + "/" + p.Name() + "DB"

	migrate := false
	if _, err := os.Stat(dbFilename); os.IsNotExist(err) {
		migrate = true
	}

	var err error
	p.db, err = leveldb.OpenFile(dbFilename, nil)
	if err != nil {
		return err
	}

	bot.AddCloseFunc(func() {
		p.flush()
		p.db.Close()
	})

	if data != nil && migrate {
		batch := new(leveldb.Batch)

		od := &oldData{}
		if err := json.Unmarshal(data, od); err == nil {
			for k, u := range od.Users {
				b, err := msgpack.Marshal(u)
				if err == nil {
					batch.Put([]byte(k), b)
				}
			}
		}

		err = p.db.Write(batch, nil)
	}

	go p.Run(bot, service)
	return nil
}

// Save will save plugin state to a byte array.
func (p *playedPlugin) Save() ([]byte, error) {
	return nil, nil
}

func (p *playedPlugin) Update(user string, entry string, batch *leveldb.Batch) {
	t := time.Now()

	var u *playedUser
	b, err := p.db.Get([]byte(user), nil)
	if err == leveldb.ErrNotFound {
		u = &playedUser{
			Entries:     map[string]*playedEntry{},
			Current:     entry,
			LastChanged: t,
			FirstSeen:   t,
		}
	} else {
		u = &playedUser{}
		err = msgpack.Unmarshal(b, u)
		if err != nil {
			return
		}
	}

	u.Update(entry, t)

	b, err = msgpack.Marshal(u)

	if batch == nil {
		p.db.Put([]byte(user), b, nil)
	} else {
		batch.Put([]byte(user), b)
	}
}

func (p *playedPlugin) updatePresences(presences []*discordgo.Presence, batch *leveldb.Batch) {
	for _, pu := range presences {
		e := ""
		if pu.Game != nil {
			e = pu.Game.Name
		}
		p.Update(pu.User.ID, e, batch)
	}
}

var FLUSH = 500

// Run is the background go routine that executes for the life of the plugin.
func (p *playedPlugin) Run(bot *bruxism.Bot, service bruxism.Service) {
	discord := service.(*bruxism.Discord)

	discord.Session.AddHandler(func(s *discordgo.Session, g *discordgo.GuildCreate) {
		if g.Unavailable == nil || *g.Unavailable {
			return
		}

		p.Lock()
		defer p.Unlock()

		batch := new(leveldb.Batch)

		p.updatePresences(g.Presences, batch)

		p.db.Write(batch, nil)
	})

	discord.Session.AddHandler(func(s *discordgo.Session, pr *discordgo.PresencesReplace) {
		p.Lock()
		defer p.Unlock()

		for _, pu := range *pr {
			p.queue = append(p.queue, pu)
		}

		if len(p.queue) > FLUSH {
			p.flush()
		}
	})

	discord.Session.AddHandler(func(s *discordgo.Session, pu *discordgo.PresenceUpdate) {
		p.Lock()
		defer p.Unlock()

		p.queue = append(p.queue, &pu.Presence)

		if len(p.queue) > FLUSH {
			p.flush()
		}
	})
}

func (p *playedPlugin) flush() {
	if p.queue == nil {
		return
	}

	batch := new(leveldb.Batch)

	p.updatePresences(p.queue, batch)

	p.db.Write(batch, nil)

	p.queue = nil
}

// Help returns a list of help strings that are printed when the user requests them.
func (p *playedPlugin) Help(bot *bruxism.Bot, service bruxism.Service, message bruxism.Message, detailed bool) []string {
	if detailed {
		return nil
	}

	return bruxism.CommandHelp(service, "played", "[@username]", "Returns your most played games, or a users most played games if provided.")
}

var userIDRegex = regexp.MustCompile("<@!?([0-9]*)>")

func (p *playedPlugin) Message(bot *bruxism.Bot, service bruxism.Service, message bruxism.Message) {
	defer bruxism.MessageRecover()
	if service.Name() == bruxism.DiscordServiceName && !service.IsMe(message) {
		if bruxism.MatchesCommand(service, "played", message) {
			query := strings.Join(strings.Split(message.RawMessage(), " ")[1:], " ")

			id := message.UserID()
			match := userIDRegex.FindStringSubmatch(query)
			if match != nil {
				id = match[1]
			}

			p.RLock()
			defer p.RUnlock()

			for _, pu := range p.queue {
				if pu.User.ID == id {
					p.flush()
					break
				}
			}

			var u *playedUser
			b, err := p.db.Get([]byte(id), nil)
			if err == leveldb.ErrNotFound {
				service.SendMessage(message.Channel(), "I haven't seen that user.")
				return
			} else {
				u = &playedUser{}
				err = msgpack.Unmarshal(b, u)
				if err != nil {
					service.SendMessage(message.Channel(), "I haven't seen that user.")
					return
				}
			}

			if len(u.Entries) == 0 {
				service.SendMessage(message.Channel(), "I haven't seen anything played by that user.")
				return
			}

			lc := humanize.Time(u.LastChanged)
			u.Update(u.Current, time.Now())

			pes := make(byDuration, len(u.Entries))
			i := 0
			for _, pe := range u.Entries {
				pes[i] = pe
				i++
			}

			sort.Sort(pes)

			messageText := fmt.Sprintf("*First seen %s, last update %s*\n", humanize.Time(u.FirstSeen), lc)
			for i = 0; i < len(pes) && i < 5; i++ {
				pe := pes[i]

				du := pe.Duration

				ds := ""
				hours := int(du / time.Hour)
				if hours > 0 {
					ds += fmt.Sprintf("%dh ", hours)
					du -= time.Duration(hours) * time.Hour
				}

				minutes := int(du / time.Minute)
				if minutes > 0 || len(ds) > 0 {
					ds += fmt.Sprintf("%dm ", minutes)
					du -= time.Duration(minutes) * time.Minute
				}

				seconds := int(du / time.Second)
				ds += fmt.Sprintf("%ds", seconds)

				messageText += fmt.Sprintf("**%s**: %s\n", pe.Name, ds)
			}
			service.SendMessage(message.Channel(), messageText)
		}
	}
}

// Name returns the name of the plugin.
func (p *playedPlugin) Name() string {
	return "Played"
}

// New will create a played plugin.
func New() bruxism.Plugin {
	return &playedPlugin{}
}
