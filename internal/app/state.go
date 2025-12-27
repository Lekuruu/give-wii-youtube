package app

import (
	"fmt"

	"github.com/Lekuruu/give-wii-youtube/internal/providers"
)

type State struct {
	Config     *Config
	Logger     *Logger
	Storage    Storage
	Provider   providers.Provider
	Categories *CategoryListing
}

func NewState() *State {
	config := NewConfig()
	if config == nil {
		fmt.Println("Failed to initialize config")
		return nil
	}

	logger := NewLogger("wii-youtube")
	if logger == nil {
		fmt.Println("Failed to initialize logger")
		return nil
	}

	storage := NewStorage(config, logger)
	if storage == nil {
		logger.Error("Failed to initialize storage")
		return nil
	}

	categories, err := LoadCategories(config.Categories.ConfigPath)
	if err != nil {
		// Use default empty categories
		logger.Errorf("Failed to load categories: %v", err)
		categories = &CategoryListing{Entries: []Category{}}
	}

	return &State{
		Config:     config,
		Logger:     logger,
		Storage:    storage,
		Categories: categories,
	}
}
