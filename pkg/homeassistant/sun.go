package homeassistant

import (
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

	sun.Rising, err = castAttribute[bool](sunState, "rising")
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

	sun.Elevation, err = castAttribute[float64](sunState, "elevation")
	if err != nil {
		return sun, err
	}

	sun.Azimuth, err = castAttribute[float64](sunState, "azimuth")
	if err != nil {
		return sun, err
	}

	return sun, nil
}
