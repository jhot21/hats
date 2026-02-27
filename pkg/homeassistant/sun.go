package homeassistant

import (
	"math"
	"time"
)

type Sun struct {
	AboveHorizon bool
	Rising       bool // True if the Sun is currently rising, after solar midnight and before solar noon

	NextRising      time.Time // Date and time of the next sun rising
	NextRisingToday bool

	NextSetting      time.Time // Date and time of the next sun setting
	NextSettingToday bool

	NextDawn      time.Time // Date and time of the next dawn
	NextDawnToday bool

	NextDusk      time.Time // Date and time of the next dusk
	NextDuskToday bool

	NextNoon      time.Time // Date and time of the next solar noon
	NextNoonToday bool

	NextMidnight      time.Time // Date and time of the next solar midnight
	NextMidnightToday bool

	Elevation float64 // The angle between the sun and the horizon
	Azimuth   float64 // The angle is shown clockwise from north
}

func NewSun(sunState StateData) (Sun, error) {
	var err error
	sun := Sun{
		AboveHorizon: StateToBool(sunState.State),
	}

	sun.Rising, err = CastAttribute[bool](sunState, "rising")
	if err != nil {
		return sun, err
	}

	sun.NextRising, sun.NextRisingToday, err = timeAttribute(sunState, "next_rising")
	if err != nil {
		return sun, err
	}

	sun.NextSetting, sun.NextSettingToday, err = timeAttribute(sunState, "next_setting")
	if err != nil {
		return sun, err
	}

	sun.NextDawn, sun.NextDawnToday, err = timeAttribute(sunState, "next_dawn")
	if err != nil {
		return sun, err
	}

	sun.NextDusk, sun.NextDuskToday, err = timeAttribute(sunState, "next_dusk")
	if err != nil {
		return sun, err
	}

	sun.NextNoon, sun.NextNoonToday, err = timeAttribute(sunState, "next_noon")
	if err != nil {
		return sun, err
	}

	sun.NextMidnight, sun.NextMidnightToday, err = timeAttribute(sunState, "next_midnight")
	if err != nil {
		return sun, err
	}

	sun.Elevation, err = CastAttribute[float64](sunState, "elevation")
	if err != nil {
		return sun, err
	}

	sun.Azimuth, err = CastAttribute[float64](sunState, "azimuth")
	if err != nil {
		return sun, err
	}

	return sun, nil
}

// How direct is the sun compared to a window with a certain azimuth.
// I don't know how to find the true angle because I took trig 20+ years ago, so we'll just use the greater between elevation and azimuth.
//   - If the sun is straight on (elevation 0, sun azimuth equal to window azimuth), return 1.0
//   - If the sun is at a 90 degree angle, return 0.0
//   - In-between will return a value between 0.0 and 1.0
//   - If the sun is below the horizon or the azimuth difference is greater than 90 degrees, return -1.0
func (s Sun) DirectnessToWindow(windowAzimuth float64) float64 {
	if s.Elevation < 0 {
		return -1.0
	}

	absAzimuth := math.Abs(s.Azimuth - windowAzimuth)

	if absAzimuth > 90.0 {
		return -1.0
	}

	max := s.Elevation
	if absAzimuth > s.Elevation {
		max = absAzimuth
	}

	return (90.0 - max) / 90.0
}
