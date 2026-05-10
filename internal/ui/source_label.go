package ui

import "github.com/neur0map/glazepkg/internal/model"

func sourceLabel(source model.Source) string {
	if source == model.SourceBrewCask {
		return "cask"
	}
	return string(source)
}
