package chartplugin

import (
	"bytes"
	"fmt"
	"image/color"
	"log"
	"math/rand"
	"strings"

	"github.com/gonum/plot"
	"github.com/gonum/plot/plotter"
	"github.com/gonum/plot/plotutil"
	"github.com/gonum/plot/vg"
	"github.com/matannoam/comicjerk"
	"github.com/iopred/discordgo"
)

type chartPlugin struct {
	comicjerk.SimplePlugin
}

var randomDirection = []string{
	"up",
	"down",
	"flat",
}

var randomY = []string{
	"interest",
	"care",
	"success",
	"fail",
	"happiness",
	"sadness",
	"money",
}

var randomX = []string{
	"time",
	"releases",
	"days",
	"years",
}

func (p *chartPlugin) random(list []string) string {
	return list[rand.Intn(len(list))]
}

func (p *chartPlugin) randomChart(service comicjerk.Service) string {
	ticks := ""
	if service.Name() == comicjerk.DiscordServiceName {
		ticks = "`"
	}

	return fmt.Sprintf("%s%schart %s %s, %s%s", ticks, service.CommandPrefix(), p.random(randomDirection), p.random(randomY), p.random(randomX), ticks)
}

func (p *chartPlugin) helpFunc(bot *comicjerk.Bot, service comicjerk.Service, message comicjerk.Message, detailed bool) []string {
	help := comicjerk.CommandHelp(service, "chart", "<up|down|flat> <vertical message>, <horizontal message>", "Creates a chart trending in the desired direction.")

	if detailed {
		help = append(help, []string{
			"Examples:",
			comicjerk.CommandHelp(service, "chart", "down interest, time", "Creates a chart showing declining interest over time")[0],
		}...)
	}

	return help
}

func (p *chartPlugin) messageFunc(bot *comicjerk.Bot, service comicjerk.Service, message comicjerk.Message) {
	if service.IsMe(message) {
		return
	}

	if comicjerk.MatchesCommand(service, "chart", message) {
		query, parts := comicjerk.ParseCommand(service, message)
		if len(parts) == 0 {
			service.SendMessage(message.Channel(), fmt.Sprintf("Invalid chart eg: %s", p.randomChart(service)))
			return
		}

		start, end := 0.5, 0.5

		switch parts[0] {
		case "up":
			start, end = 0, 1
		case "down":
			start, end = 1, 0
		case "flat":
		case "straight":
		default:
			service.SendMessage(message.Channel(), fmt.Sprintf("Invalid chart direction. eg: %s", p.randomChart(service)))
			return
		}

		axes := strings.Split(query[len(parts[0]):], ",")
		if len(axes) != 2 {
			service.SendMessage(message.Channel(), fmt.Sprintf("Invalid chart axis labels eg: %s", p.randomChart(service)))
			return
		}

		pl, err := plot.New()
		if err != nil {
			service.SendMessage(message.Channel(), fmt.Sprintf("Error making chart, sorry! eg: %s", p.randomChart(service)))
			return
		}

		service.Typing(message.Channel())

		pl.Y.Label.Text = axes[0]
		pl.X.Label.Text = axes[1]

		num := 5 + rand.Intn(15)

		start *= float64(num)
		end *= float64(num)

		pts := make(plotter.XYs, num)
		for i := range pts {
			pts[i].X = float64(i) + rand.Float64()*0.5 - 0.2
			pts[i].Y = start + float64(end-start)/float64(num-1)*float64(i) + rand.Float64()*0.5 - 0.25
		}

		pl.X.Tick.Label.Color = color.Transparent
		pl.Y.Tick.Label.Color = color.Transparent

		pl.X.Min = -0.5
		pl.X.Max = float64(num) + 0.5

		pl.Y.Min = -0.5
		pl.Y.Max = float64(num) + 0.5

		lpLine, lpPoints, err := plotter.NewLinePoints(pts)
		if err != nil {
			service.SendMessage(message.Channel(), fmt.Sprintf("Sorry %s, there was a problem creating your chart.", message.UserName()))
		}
		lpLine.Color = plotutil.Color(rand.Int())
		lpLine.Width = vg.Points(1 + 0.5*rand.Float64())
		lpLine.Dashes = plotutil.Dashes(rand.Int())
		lpPoints.Shape = plotutil.Shape(rand.Int())
		lpPoints.Color = lpLine.Color

		pl.Add(lpLine, lpPoints)

		w, err := pl.WriterTo(320, 240, "png")
		if err != nil {
			service.SendMessage(message.Channel(), fmt.Sprintf("Sorry %s, there was a problem creating your chart.", message.UserName()))
			return
		}

		b := &bytes.Buffer{}
		w.WriteTo(b)

		go func() {
			if service.Name() == comicjerk.DiscordServiceName {
				discord := service.(*comicjerk.Discord)
				p, err := discord.UserChannelPermissions(message.UserID(), message.Channel())
				if err == nil && p&discordgo.PermissionAttachFiles != 0 {
					service.SendFile(message.Channel(), "chart.png", b)
					return
				}
			}

			url, err := bot.UploadToImgur(b, "chart.png")
			if err != nil {
				service.SendMessage(message.Channel(), fmt.Sprintf("Sorry %s, there was a problem uploading the chart to imgur.", message.UserName()))
				log.Println("Error uploading chart: ", err)
				return
			}

			if service.Name() == comicjerk.DiscordServiceName {
				service.SendMessage(message.Channel(), fmt.Sprintf("Here's your chart <@%s>: %s", message.UserID(), url))
			} else {
				service.SendMessage(message.Channel(), fmt.Sprintf("Here's your chart %s: %s", message.UserName(), url))
			}
		}()
	}
}

// New will create a new comic plugin.
func New() comicjerk.Plugin {
	p := &chartPlugin{
		SimplePlugin: *comicjerk.NewSimplePlugin("Chart"),
	}
	p.MessageFunc = p.messageFunc
	p.HelpFunc = p.helpFunc
	return p
}
