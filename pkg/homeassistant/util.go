package homeassistant

import (
	"regexp"
	"strings"
)

// StateToBool converts a state string into a boolean
//
//	States that return true: "on", "home", "open", "playing", non-zero numbers, etc.
//	All others return false
func StateToBool(state string) bool {
	trueRegex := regexp.MustCompile(`^(on|home|open(ing)?|unlocked|playing|good|walking|charging|alive|heat|cool|heat_cool|[1-9][\d\.]*|0\.0*[1-9]\d*)$`)
	return trueRegex.MatchString(state)
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
