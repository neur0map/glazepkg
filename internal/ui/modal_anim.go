package ui

import (
	"math"
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
	modalSpringFPS      = 60.0
	modalSpringOmega    = 5.0
	modalSpringDamping  = 0.7
	modalAnimDoneEpsPos = 0.01
	modalAnimDoneEpsVel = 0.05
)

func newModalSpring() harmonica.Spring {
	return harmonica.NewSpring(harmonica.FPS(modalSpringFPS), modalSpringOmega, modalSpringDamping)
}

func modalAnimSettled(pos, vel, target float64) bool {
	return math.Abs(pos-target) < modalAnimDoneEpsPos && math.Abs(vel) < modalAnimDoneEpsVel
}
