package nws

import (
	"fmt"
	"strings"

	"codeberg.org/jhot/hats/internal/util"
	rsty "codeberg.org/jhot/hats/pkg/resty"
	"github.com/go-resty/resty/v2"
	"github.com/samber/lo"
)

const (
	NWS_BASE_URL = "https://api.weather.gov"
)

type NwsClient struct {
	restClient *resty.Client
}

func New() *NwsClient {
	return &NwsClient{
		restClient: resty.New().
			SetBaseURL(NWS_BASE_URL).
			SetHeader("Accept", "application/geo+json").
			SetHeader("User-Agent", util.GetEnvWithDefault("NWS_USER_AGENT", "(HATS-client, hats@jhot.me)")),
	}
}

func Limit(value int) rsty.RestyOption {
	if value > 0 {
		return func(r *resty.Request) {
			r.SetQueryParam("limit", fmt.Sprintf("%d", value))
		}
	}
	return func(r *resty.Request) {}
}

func (c *NwsClient) CoordinatesToPoint(lat float64, lon float64) (Point, error) {
	var p Point
	_, err := c.restClient.R().SetResult(&p).Get(fmt.Sprintf("points/%f,%f", lat, lon))
	p.Lat = lat
	p.Lon = lon
	return p, err
}

func (p Point) GetWebURL() string {
	return fmt.Sprintf("https://forecast.weather.gov/MapClick.php?lat=%f&lon=%f", p.Lat, p.Lon)
}

func (c *NwsClient) GetHourlyForecast(p Point) (HourlyProps, error) {
	var data struct {
		Properties HourlyProps `json:"properties"`
	}
	_, err := util.CheckSuccess(c.restClient.R().SetResult(&data).
		Get(removeBase(p.Properties.ForecastHourly)))
	if err != nil {
		return HourlyProps{}, nil
	}

	return data.Properties, nil
}

func (c *NwsClient) GetDailyForecast(p Point) (DailyProps, error) {
	var data struct {
		Properties DailyProps `json:"properties"`
	}
	_, err := util.CheckSuccess(c.restClient.R().SetResult(&data).
		Get(removeBase(p.Properties.Forecast)))
	if err != nil {
		return DailyProps{}, nil
	}

	return data.Properties, nil
}

func (p HourlyProps) GetHighLow(hours int) (high int, low int) {
	if hours > len(p.Periods) {
		hours = len(p.Periods)
	}

	temps := lo.Map(p.Periods[0:hours-1], func(p HourlyPeriod, _ int) int {
		return p.Temperature
	})

	return lo.Max(temps), lo.Min(temps)
}

func (c *NwsClient) GetStations(p Point, opts ...rsty.RestyOption) ([]string, error) {
	var data struct {
		ObservationStations []string `json:"observationStations"`
	}

	req := c.restClient.R().SetResult(&data)

	for _, opt := range opts {
		opt(req)
	}

	_, err := util.CheckSuccess(req.Get(removeBase(p.Properties.ObservationStations)))
	if err != nil {
		return []string{}, err
	}

	return data.ObservationStations, nil
}

func (c *NwsClient) GetObservations(p Point, opts ...rsty.RestyOption) ([]Observations, error) {
	stations, err := c.GetStations(p, Limit(1))
	if err != nil {
		return []Observations{}, fmt.Errorf("%w: error getting stations", err)
	}

	var data struct {
		Features []struct {
			Properties Observations `json:"properties"`
		} `json:"features"`
	}

	req := c.restClient.R().SetResult(&data)

	for _, opt := range opts {
		opt(req)
	}

	_, err = util.CheckSuccess(req.Get(removeBase(stations[0] + "/observations")))
	if err != nil {
		return []Observations{}, err
	}

	var output []Observations
	for _, feat := range data.Features {
		output = append(output, feat.Properties)
	}

	return output, nil
}

func (c *NwsClient) GetLatestObservations(p Point) (Observations, error) {
	stations, err := c.GetStations(p, Limit(1))
	if err != nil {
		return Observations{}, fmt.Errorf("%w: error getting stations", err)
	}

	var data struct {
		Properties Observations `json:"properties"`
	}
	_, err = util.CheckSuccess(c.restClient.R().SetResult(&data).
		Get(removeBase(stations[0] + "/observations/latest")))
	if err != nil {
		return Observations{}, err
	}

	return data.Properties, nil
}

func removeBase(longUrl string) string {
	return strings.TrimPrefix(longUrl, NWS_BASE_URL+"/")
}
