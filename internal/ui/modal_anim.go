package ui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/harmonica"
)

type modalAnimTickMsg struct{}

func modalAnimTick() tea.Cmd {
	return tea.Tick(time.Second/60, func(time.Time) tea.Msg {
		return modalAnimTickMsg{}
	})
}

const (
	modalSpringFPS     = 60.0
	modalSpringOmega   = 5.0
	modalSpringDamping = 0.7
)

func newModalSpring() harmonica.Spring {
	return harmonica.NewSpring(harmonica.FPS(modalSpringFPS), modalSpringOmega, modalSpringDamping)
}
