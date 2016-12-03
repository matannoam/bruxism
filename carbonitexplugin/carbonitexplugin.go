package carbonitexplugin

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/matannoam/comicjerk"
)

type carbonitexPlugin struct {
	comicjerk.SimplePlugin
	key string
}

func (p *carbonitexPlugin) carbonitexPluginLoadFunc(bot *comicjerk.Bot, service comicjerk.Service, data []byte) error {
	if service.Name() != comicjerk.DiscordServiceName {
		panic("Carbonitex Plugin only supports Discord.")
	}

	go p.Run(bot, service)
	return nil
}

func (p *carbonitexPlugin) Run(bot *comicjerk.Bot, service comicjerk.Service) {
	for {
		<-time.After(5 * time.Minute)

		resp, err := http.PostForm("https://www.carbonitex.net/discord/data/botdata.php", url.Values{"key": {p.key}, "servercount": {fmt.Sprintf("%d", service.ChannelCount())}})

		if err == nil {
			htmlData, err := ioutil.ReadAll(resp.Body)
			if err == nil {
				resp.Body.Close()
				log.Println(string(htmlData))
			}
		}

		<-time.After(55 * time.Minute)
	}

}

// New will create a new carbonitex plugin.
// This plugin reports the server count to the carbonitex service.
func New(key string) comicjerk.Plugin {
	p := &carbonitexPlugin{
		SimplePlugin: *comicjerk.NewSimplePlugin("Carbonitex"),
		key:          key,
	}
	p.LoadFunc = p.carbonitexPluginLoadFunc
	return p
}
