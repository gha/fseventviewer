package main

import (
	"log"
	"log/syslog"

	"github.com/jroimartin/gocui"
	"github.com/tomponline/fsclient/fsclient"
)

var (
	keyEvents chan gocui.Key
)

func main() {
	log.SetFlags(0)
	syslogWriter, err := syslog.New(syslog.LOG_INFO, "fseventviewer")
	if err == nil {
		log.SetOutput(syslogWriter)
	}

	g, err := gocui.NewGui(gocui.Output256)
	if err != nil {
		log.Fatal(err)
	}

	g.BgColor = gocui.ColorBlack
	g.FgColor = gocui.ColorWhite

	defer g.Close()

	g.SetManagerFunc(layout)

	if err := g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
		log.Fatal(err)
	}
	if err := g.SetKeybinding("", gocui.KeyArrowUp, gocui.ModNone, cursorUp); err != nil {
		log.Fatal(err)
	}
	if err := g.SetKeybinding("", gocui.KeyArrowDown, gocui.ModNone, cursorDown); err != nil {
		log.Fatal(err)
	}

	filters := []string{}
	subs := []string{
		"all",
	}

	fs := fsclient.NewClient("127.0.0.1:8021", "ClueCon", filters, subs, initFunction)

	app := &App{
		gui:      g,
		fs:       fs,
		callData: make(map[string]*CallData),
		callIdx:  []string{},
		fsEvents: make(chan map[string]string),
	}

	// Create a channel to manage key presses
	keyEvents = make(chan gocui.Key)

	app.setDefaultView()
	go app.readEvents()

	if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
		log.Fatal(err)
	}
}

func layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()
	eventView, err := g.SetView("eventlist", 0, 0, maxX-1, maxY-1)
	if err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
	}

	eventView.Highlight = true
	eventView.Wrap = true
	//eventView.Editable = false
	//eventView.Autoscroll = false
	//eventView.SelBgColor = gocui.ColorBlack | gocui.ColorWhite
	eventView.BgColor = gocui.ColorBlack
	eventView.FgColor = gocui.ColorWhite

	// Set the active view to show the cursor
	g.SetCurrentView("eventlist")

	return nil
}

func newCallView(g *gocui.Gui, uuid string) error {
	maxX, maxY := g.Size()
	if _, err := g.SetView(uuid, 0, 0, maxX-1, maxY-1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
	}

	return nil
}

func quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}

func cursorUp(g *gocui.Gui, v *gocui.View) error {
	keyEvents <- gocui.KeyArrowUp
	return nil
}

func cursorDown(g *gocui.Gui, v *gocui.View) error {
	keyEvents <- gocui.KeyArrowDown
	return nil
}

func initFunction(fs *fsclient.Client) {
	log.Print("Connected to FreeSWITCH")
}

func (app *App) readEvents() {
	go app.readFSEvents()

	for {
		select {
		case event := <-app.fsEvents:
			app.processFSEvent(event)
		case key := <-keyEvents:
			app.handleKeyEvent(key)
		}
	}
}

func (app *App) readFSEvents() {
	for {
		app.fsEvents <- app.fs.NextEvent()
	}
}

func (app *App) processFSEvent(event map[string]string) {
	uuid := event["Channel-Call-UUID"]
	if uuid == "" {
		return
	}

	callData, callExists := app.callData[uuid]

	switch event["Event-Name"] {
	case "CHANNEL_CREATE":
		// Channel create
		if !callExists {
			app.addNewCall(event, uuid)
		}
	case "CHANNEL_DESTROY":
		// Channel destroy
	case "CHANNEL_BRIDGE":
		// Channel bridge
	default:
		// Other events
		if callExists {
			app.updateCall(event, callData)
		}
	}
}
