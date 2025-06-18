package cocorahs

import (
	"errors"
	"fmt"
	"time"

	"codeberg.org/jhot/hats/internal/util"
	"github.com/samber/lo"
)

type GaugeCatch struct {
	Value   float64 `json:"value"`
	IsTrace bool    `json:"isTrace"`
}

type DailyObservation struct {
	ObservationDate string     `json:"obsDate"`
	ObservationTime string     `json:"obsTime"`
	GaugeCatch      GaugeCatch `json:"gaugeCatch"`
}

func (d DailyObservation) GetObservationDateTime() time.Time {
	parsed, _ := time.Parse("2006-01-02T15:04", fmt.Sprintf("%sT%s", d.ObservationDate, d.ObservationTime))
	return parsed
}

type StationMetadata struct {
	Id                  int64  `json:"id"`
	StationNumber       string `json:"stationNumber"`
	LastObservationDate string `json:"lastObsDate"`
}

type StationArchive struct {
	StationMetadata   StationMetadata    `json:"stationMetadata"`
	DailyObservations []DailyObservation `json:"dailyObs"`
}

func (s StationArchive) GetLastObservationDate() (time.Time, bool, error) {
	parsed, err := time.ParseInLocation("2006-01-02", s.StationMetadata.LastObservationDate, time.Local)
	if err != nil {
		return parsed, false, err
	}
	today := time.Now()

	return parsed, util.SameDay(parsed, today), nil
}

func (s StationArchive) GetLastObservation(requireToday bool) (DailyObservation, error) {
	lastObsDate, isToday, err := s.GetLastObservationDate()
	if err != nil {
		return DailyObservation{}, fmt.Errorf("error parsing last observation date %w", err)
	}

	if requireToday && !isToday {
		return DailyObservation{}, fmt.Errorf("%s is not today", lastObsDate.Format("2006-01-02"))
	}

	if lastOb, found := lo.Find(s.DailyObservations, func(d DailyObservation) bool {
		return util.SameDay(lastObsDate, d.GetObservationDateTime())
	}); found {
		return lastOb, nil
	}

	return DailyObservation{}, errors.New("couldn't find last observation")
}
