package homeassistant

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// StateToBool converts a state string into a boolean
//
//	States that return true: "on", "home", "open", "opening", "unlocked", "playing", "active", "good",
//		"walking", "charging", "alive", "heat", "heat_cool", "cool", "above_horizon", numbers > 0
//	All others return false
func StateToBool(state string) bool {
	switch state {
	// truthy
	case "on", "home", "open", "opening", "unlocked", "playing", "active", "good",
		"walking", "charging", "alive", "heat", "heat_cool", "cool", "above_horizon", "detected", "wet":
		return true
	// falsey
	case "off", "away", "closed", "closing", "locked", "paused", "inactive", "bad",
		"discharging", "dead", "below_horizon", "0", "0.0", "clear", "dry":
		return false
	}

	// number greater than zero
	val, err := strconv.ParseFloat(state, 64)
	if err == nil && val > 0.0 {
		return true
	}

	// unaccounted for
	return false
}

// BoolToService converts a boolean into the appropriate service string
//
//	For locks: true becomes "unlock" and false becomes "lock"
//	For covers: true becomes "open_cover" and false becomes "close_cover"
//	For all others: true becomes "turn_on" and false becomes "turn_off"
func BoolToService(entityId string, desiredState bool) string {
	domain := strings.Split(entityId, ".")[0]

	switch domain {
	case Domains.Lock:
		if desiredState {
			return Services.Unlock
		} else {
			return Services.Lock
		}
	case Domains.Cover:
		if desiredState {
			return Services.OpenCover
		} else {
			return Services.CloseCover
		}
	default:
		if desiredState {
			return Services.TurnOn
		} else {
			return Services.TurnOff
		}
	}
}

// Parse a Home Assistant duration in HH:MM:SS format
func ParseDuration(d string) (time.Duration, error) {
	var hour, min, sec int
	_, err := fmt.Sscanf(d, "%d:%d:%d", &hour, &min, &sec)
	if err != nil {
		return 5 * time.Minute, err
	}

	return time.Duration(hour)*time.Hour + time.Duration(min)*time.Minute + time.Duration(sec)*time.Second, nil
}

// func FormatDuration(d time.Duration) string {
// 	hours := math.Floor(d.Hours())
// 	minutes := math.Floor(d.Minutes() - (hours * 60))
// 	seconds := math.Floor(d.Seconds() - (hours * 60 * 60) - (minutes * 60))
// 	return fmt.Sprintf("%02.f:%02.f:%02.f", hours, minutes, seconds)
// }

func castAttribute[T any](state StateData, attributeName string) (T, error) {
	var val T

	attrInterface, attrExists := state.Attributes[attributeName]
	if !attrExists {
		return val, errors.New("attribute not found")
	}

	if val, ok := attrInterface.(T); ok {
		return val, nil
	} else {
		return val, errors.New("attribute couldn't be cast")
	}
}

func timeAttribute(state StateData, attributeName string) (time.Time, bool, error) {
	var t time.Time
	attrValue, err := castAttribute[string](state, attributeName)
	if err != nil {
		return t, false, err
	}

	t, err = time.Parse(time.RFC3339, attrValue)
	if err != nil {
		return t, false, err
	}

	local := t.Local()
	now := time.Now()
	today := local.Day() == now.Day() && local.Month() == now.Month() && local.Year() == now.Year()

	return local, today, nil
}
