package app

import (
	"github.com/go-gl/glfw/v3.3/glfw"
)

type EventHandler struct {
	options map[glfw.Key]keyOption
}

func NewEventHandler() *EventHandler {
	return &EventHandler{options: make(map[glfw.Key]keyOption)}
}

type KeyCallbackKind int

const (
	Switch KeyCallbackKind = iota
	Hold
)

type keyOption struct {
	kind  KeyCallbackKind
	value *bool
}

func (eh *EventHandler) AddOption(key glfw.Key, value *bool, kind KeyCallbackKind) {
	eh.options[key] = keyOption{
		kind:  kind,
		value: value,
	}
}

func (eh *EventHandler) KeyCallback() glfw.KeyCallback {
	return func(w *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
		option, found := eh.options[key]
		if !found {
			return
		}

		switch option.kind {
		case Switch:
			if action == glfw.Press {
				*option.value = !*option.value
			}
		case Hold:
			*option.value = (action != glfw.Release)
		}
	}
}
