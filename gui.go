package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"

	"github.com/jroimartin/gocui"
	"github.com/tomponline/fsclient/fsclient"
)

type App struct {
	gui      *gocui.Gui
	fs       *fsclient.Client
	callData map[string]*CallData
	callIdx  []string
	fsEvents chan map[string]string
}

type CallData struct {
	Idx                      int
	UUID                     string
	CallDirection            string
	CallerOrigCallerIDNumber string
	SIPToUser                string
	Events                   []*EventData
}

type EventData struct {
	EventName     string
	EventSubclass string
	Vars          []string
}

func (app *App) setDefaultView() {
	app.gui.Update(func(g *gocui.Gui) error {
		v, err := app.gui.View("eventlist")
		if err != nil {
			return err
		}

		fmt.Fprintln(v, "Waiting for calls...")

		return nil
	})
}

func (app *App) addNewCall(event map[string]string, uuid string) {
	firstCall := false
	if len(app.callIdx) == 0 {
		firstCall = true
	}

	app.callIdx = append(app.callIdx, uuid)

	cd := &CallData{
		Idx:                      len(app.callIdx) - 1,
		UUID:                     uuid,
		CallDirection:            event["Call-Direction"],
		CallerOrigCallerIDNumber: event["Caller-Orig-Caller-ID-Number"],
		SIPToUser:                event["variable_sip_to_user"],
		Events: []*EventData{
			&EventData{
				EventName:     event["Event-Name"],
				EventSubclass: event["Event-Subclass"],
				Vars:          []string{},
			},
		},
	}

	app.callData[uuid] = cd

	app.gui.Update(func(g *gocui.Gui) error {
		v, err := g.View("eventlist")
		if err != nil {
			return err
		}

		if firstCall {
			v.Clear()
			fmt.Fprintln(v, fmt.Sprintf("| %-14v | %-36v | %-20v | %-20v |", "Call Direction", "Channel ID", "CLI", "To"))
		}

		fmt.Fprintln(v, fmt.Sprintf(
			"| %-14v | %-36v | %-20v | %-20v |",
			cd.CallDirection, uuid, cd.CallerOrigCallerIDNumber, cd.SIPToUser))

		if firstCall {
			if err := app.setCursor(v, 1); err != nil {
				return err
			}

			//app.gui.Cursor = true
		}

		return nil
	})
}

func (app *App) updateCall(event map[string]string, callData *CallData) {
	eventData := &EventData{
		EventName:     event["Event-Name"],
		EventSubclass: event["Event-Subclass"],
		Vars:          []string{},
	}

	callData.Events = append(callData.Events, eventData)

	if event["Call-Direction"] != callData.CallDirection {
		callData.CallDirection = event["Call-Direction"]
		app.updateLine(callData)
	}
}

func (app *App) updateLine(callData *CallData) {
	app.gui.Update(func(g *gocui.Gui) error {
		v, err := g.View("eventlist")
		if err != nil {
			return err
		}

		// Take a copy of the buffer
		old := bytes.Buffer{}
		writer := bufio.NewWriter(&old)
		_, err = io.Copy(writer, v)
		if err != nil {
			log.Print("Error copying bytes for buffer update: %v", err)
			return err
		}

		writer.Flush()
		v.Clear()

		scanner := bufio.NewScanner(&old)
		lc := 0                // line count
		tl := callData.Idx + 1 // target line + header line
		for scanner.Scan() {
			if lc == tl {
				//log.Printf("Line %d requires update", lc)
				fmt.Fprintln(v, fmt.Sprintf(
					"| %-14v | %-36v | %-20v | %-20v |",
					callData.CallDirection, callData.UUID, callData.CallerOrigCallerIDNumber, callData.SIPToUser))
			} else if scanner.Text() != "" {
				//log.Printf("Copying line %d", lc)
				fmt.Fprintln(v, scanner.Text())
			}

			lc++
		}

		if err = scanner.Err(); err != nil {
			log.Print("Error scanning bytes for buffer update: %v", err)
			return err
		}

		return nil
	})
}

func (app *App) handleKeyEvent(key gocui.Key) {
	eventList, err := app.gui.View("eventlist")
	if err != nil {
		return
	}

	switch key {
	case gocui.KeyArrowUp:
		app.setCursor(eventList, -1)
	case gocui.KeyArrowDown:
		app.setCursor(eventList, 1)
	}
}

func (app *App) setCursor(v *gocui.View, my int) error {
	// Only move the cursor if its visible
	/*if !app.gui.Cursor {
		return nil
	}*/

	cx, cy := v.Cursor()
	ny := cy + my

	//if ny < 0 || ny >= len(app.callIdx) {
	if ny < 1 || ny > len(app.callIdx) {
		return nil
	}

	log.Printf("Setting cursor %d", ny)

	app.gui.Update(func(g *gocui.Gui) error {
		if err := v.SetCursor(cx, ny); err != nil {
			return err
		}

		return nil
	})

	return nil
}
