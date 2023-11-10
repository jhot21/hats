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

type Observations struct {
	Timestamp     time.Time          `json:"timestamp"`
	Temperature   UnitValue[float64] `json:"temperature"`
	DewPoint      UnitValue[float64] `json:"dewpoint"`
	WindDirection UnitValue[int]     `json:"windDirection"`
	WindSpeed     UnitValue[float64] `json:"windSpeed"`
}
