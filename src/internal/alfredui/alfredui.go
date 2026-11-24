// Package alfredui builds the JSON actions that an Alfred UI (Alfred 6)
// script action writes to stdout.
// See https://www.alfredapp.com/help/alfredui/actions/json/
package alfredui

import (
	"encoding/json"
	"os"
)

const (
	ColourError   = "systemRedColor"
	ColourInfo    = "secondaryLabelColor"
	ColourSuccess = "systemGreenColor"
)

type Props map[string]any

type Update struct {
	IDs map[string]Props `json:"ids"`
}

// Finish closes the UI and passes args and variables to the next workflow object.
type Finish struct {
	Args      any               `json:"args,omitempty"`
	Variables map[string]string `json:"variables,omitempty"`
}

// Sheet shows (Root set) or closes (Close set) a modal sheet.
type Sheet struct {
	ID    string `json:"id,omitempty"`
	Close string `json:"close,omitempty"`
	Root  any    `json:"root,omitempty"`
}

// Pulse reconfigures the UI's pulse. Interval 0 disables it.
type Pulse struct {
	Interval float64 `json:"interval"`
}

type Actions struct {
	Variables map[string]string `json:"variables,omitempty"`
	Update    *Update           `json:"update,omitempty"`
	Sheet     *Sheet            `json:"sheet,omitempty"`
	Pulse     *Pulse            `json:"pulse,omitempty"`
	Finish    *Finish           `json:"finish,omitempty"`
}

func Emit(a Actions) error { return json.NewEncoder(os.Stdout).Encode(a) }

func NewUpdate() Actions { return Actions{Update: &Update{IDs: map[string]Props{}}} }

// Set merges p into the properties already set for id.
func (a Actions) Set(id string, p Props) Actions {
	if a.Update == nil {
		a.Update = &Update{IDs: map[string]Props{}}
	}
	if cur, ok := a.Update.IDs[id]; ok {
		for k, v := range p {
			cur[k] = v
		}
		return a
	}
	a.Update.IDs[id] = p
	return a
}

func (a Actions) Status(labelID, msg, colour string) Actions {
	return a.Set(labelID, Props{"text": msg, "colour": colour})
}

func (a Actions) Enable(enabled bool, ids ...string) Actions {
	for _, id := range ids {
		a = a.Set(id, Props{"enabled": enabled})
	}
	return a
}

func (a Actions) SetVar(k, v string) Actions {
	if a.Variables == nil {
		a.Variables = map[string]string{}
	}
	a.Variables[k] = v
	return a
}

func (a Actions) WithSheet(sh *Sheet) Actions {
	a.Sheet = sh
	return a
}
