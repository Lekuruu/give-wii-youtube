package app

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

type Config struct {
	Server struct {
		Host string `env:"HOST" envDefault:"0.0.0.0"`
		Port int    `env:"PORT" envDefault:"5005"`
		Url  string `env:"URL" envDefault:"http://localhost:5005"`
	}
	Storage struct {
		Type string `env:"STORAGE_TYPE" envDefault:"local"`
		Path string `env:"STORAGE_PATH" envDefault:"./data"`
	}
	Video struct {
		Quality        string `env:"VIDEO_QUALITY" envDefault:"360"`
		DownloadFolder string `env:"DOWNLOAD_FOLDER" envDefault:"./data/downloads"`
	}
	Cache struct {
		Duration int `env:"CACHE_DURATION" envDefault:"300"`
	}
	Categories struct {
		ConfigPath string `env:"CATEGORIES_CONFIG" envDefault:"./categories.json"`
	}
}

func NewConfig() *Config {
	// Try to load .env file if it exists
	godotenv.Load()

	var config Config
	if err := env.Parse(&config); err != nil {
		return nil
	}
	return &config
}

type Category struct {
	Name           string `json:"name"`
	Route          string `json:"route"`
	TrendingParam  string `json:"trendingParam"`
	SearchFallback string `json:"searchFallback"`
}

type CategoryListing struct {
	Entries []Category `json:"categories"`
}

func LoadCategories(path string) (*CategoryListing, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read categories file: %w", err)
	}

	var config CategoryListing
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse categories file: %w", err)
	}

	return &config, nil
}

func (c *CategoryListing) GetCategoryNames() []string {
	names := make([]string, len(c.Entries))
	for i, cat := range c.Entries {
		names[i] = cat.Name
	}
	return names
}

func (c *CategoryListing) GetCategory(name string) *Category {
	for _, cat := range c.Entries {
		if cat.Name == name {
			return &cat
		}
	}
	return nil
}

func (c *CategoryListing) GetTrendingParameters() map[string]string {
	params := make(map[string]string)
	for _, cat := range c.Entries {
		if cat.TrendingParam != "" {
			params[cat.Name] = cat.TrendingParam
		}
	}
	return params
}
