package external

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"github.com/dmarques/weather-go-agent/internal/application/chat"
)

type OpenMeteoClient struct {
	httpClient *http.Client
}

// NewOpenMeteoClient creates a new OpenMeteoClient with the provided HTTP client.
func NewOpenMeteoClient(httpClient *http.Client) *OpenMeteoClient {
	return &OpenMeteoClient{httpClient: httpClient}
}

type geocodingResponse struct {
	Results []struct {
		Name      string  `json:"name"`
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	} `json:"results"`
}

type forecastResponse struct {
	Current struct {
		Temperature float64 `json:"temperature_2m"`
		Code        int     `json:"weather_code"`
		Time        string  `json:"time"`
	} `json:"current"`
	Daily struct {
		RainProbability []int `json:"precipitation_probability_max"`
	} `json:"daily"`
}

// GetWeather retrieves the current weather and forecast for the specified location using the Open-Meteo API.
func (c *OpenMeteoClient) GetWeather(ctx context.Context, location string) (chat.WeatherResult, error) {
	geoURL := "https://geocoding-api.open-meteo.com/v1/search?" + queryValues(map[string]string{
		"name":     location,
		"count":    "1",
		"language": "pt",
		"format":   "json",
	})

	var geo geocodingResponse
	if err := c.getJSON(ctx, geoURL, &geo); err != nil {
		return chat.WeatherResult{}, err
	}
	if len(geo.Results) == 0 {
		return chat.WeatherResult{}, fmt.Errorf("city not found")
	}

	place := geo.Results[0]
	forecastURL := "https://api.open-meteo.com/v1/forecast?" + queryValues(map[string]string{
		"latitude":      strconv.FormatFloat(place.Latitude, 'f', 6, 64),
		"longitude":     strconv.FormatFloat(place.Longitude, 'f', 6, 64),
		"current":       "temperature_2m,weather_code",
		"daily":         "precipitation_probability_max",
		"forecast_days": "1",
		"timezone":      "auto",
	})

	var forecast forecastResponse
	if err := c.getJSON(ctx, forecastURL, &forecast); err != nil {
		return chat.WeatherResult{}, err
	}

	rainProbability := 0
	if len(forecast.Daily.RainProbability) > 0 {
		rainProbability = forecast.Daily.RainProbability[0]
	}

	return chat.WeatherResult{
		City:            place.Name,
		TemperatureC:    forecast.Current.Temperature,
		Condition:       weatherDescription(forecast.Current.Code),
		RainProbability: rainProbability,
		ObservationTime: forecast.Current.Time,
	}, nil
}

// getJSON performs an HTTP GET request to the specified endpoint and decodes the JSON response into the target structure.
func (c *OpenMeteoClient) getJSON(ctx context.Context, endpoint string, target any) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	response, err := c.httpClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("weather service returned status %d", response.StatusCode)
	}
	return json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(target)
}

// queryValues constructs a URL-encoded query string from the provided map of key-value pairs.
func queryValues(values map[string]string) string {
	query := url.Values{}
	for key, value := range values {
		query.Set(key, value)
	}
	return query.Encode()
}

// weatherDescription maps Open-Meteo weather codes to human-readable descriptions in Portuguese.
func weatherDescription(code int) string {
	switch code {
	case 0:
		return "céu limpo"
	case 1, 2:
		return "parcialmente nublado"
	case 3:
		return "nublado"
	case 45, 48:
		return "neblina"
	case 51, 53, 55:
		return "garoa"
	case 61, 63, 65, 80, 81, 82:
		return "chuva"
	case 95, 96, 99:
		return "trovoada"
	default:
		return "condição desconhecida"
	}
}
