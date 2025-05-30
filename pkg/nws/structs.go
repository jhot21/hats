package nws

import "time"

type UnitValue[T any] struct {
	UnitCode string `json:"unitCode"`
	Value    T      `json:"value"`
}

type PointProps struct {
	GridId              string `json:"gridId"`
	GridX               int    `json:"gridX"`
	GridY               int    `json:"gridY"`
	Forecast            string `json:"forecast"`
	ForecastHourly      string `json:"forecastHourly"`
	ForecastGridData    string `json:"forecastGridData"`
	ForecastZone        string `json:"forecastZone"`
	TimeZone            string `json:"timeZone"`
	RadarStation        string `json:"radarStation"`
	ObservationStations string `json:"observationStations"`
}

type Point struct {
	ID         string     `json:"id"`
	Properties PointProps `json:"properties"`
	Lat        float64
	Lon        float64
}

type HourlyPeriod struct {
	Number                     int                `json:"number"`
	StartTime                  time.Time          `json:"startTime"`
	Endime                     time.Time          `json:"endTime"`
	IsDayTime                  bool               `json:"isDayTime"`
	Temperature                int                `json:"temperature"`
	ProbabilityOfPrecipitation UnitValue[int]     `json:"probabilityOfPrecipitation"`
	DewPoint                   UnitValue[float64] `json:"dewpoint"`
	ShortForecast              string             `json:"shortForecast"`
}

type HourlyProps struct {
	Updated time.Time      `json:"updated"`
	Units   string         `json:"units"`
	Periods []HourlyPeriod `json:"periods"`
}

type DailyPeriod struct {
	Number                     int                `json:"number,omitempty"`
	Name                       string             `json:"name,omitempty"`
	StartTime                  string             `json:"startTime,omitempty"`
	EndTime                    string             `json:"endTime,omitempty"`
	IsDaytime                  bool               `json:"isDaytime,omitempty"`
	Temperature                int                `json:"temperature,omitempty"`
	TemperatureUnit            string             `json:"temperatureUnit,omitempty"`
	TemperatureTrend           string             `json:"temperatureTrend,omitempty"`
	ProbabilityOfPrecipitation UnitValue[int]     `json:"probabilityOfPrecipitation,omitempty"`
	Dewpoint                   UnitValue[float64] `json:"dewpoint,omitempty"`
	RelativeHumidity           UnitValue[int]     `json:"relativeHumidity,omitempty"`
	WindSpeed                  string             `json:"windSpeed,omitempty"`
	WindDirection              string             `json:"windDirection,omitempty"`
	Icon                       string             `json:"icon,omitempty"`
	ShortForecast              string             `json:"shortForecast,omitempty"`
	DetailedForecast           string             `json:"detailedForecast,omitempty"`
}

type DailyProps struct {
	Updated time.Time     `json:"updated"`
	Units   string        `json:"units"`
	Periods []DailyPeriod `json:"periods"`
}

type Observations struct {
	Timestamp             time.Time          `json:"timestamp"`
	Temperature           UnitValue[float64] `json:"temperature"`
	DewPoint              UnitValue[float64] `json:"dewpoint"`
	WindDirection         UnitValue[int]     `json:"windDirection"`
	WindSpeed             UnitValue[float64] `json:"windSpeed"`
	PrecipitationLastHour UnitValue[float64] `json:"precipitationLastHour"`
}
