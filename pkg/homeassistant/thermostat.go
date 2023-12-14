package homeassistant

import (
	"fmt"
)

type ThermostatMode string

const ThermostatModeOff ThermostatMode = "off"
const ThermostatModeHeat ThermostatMode = "heat"
const ThermostatModeCool ThermostatMode = "cool"
const ThermostatModeHeatCool ThermostatMode = "heat_cool"

type ThermostatFanMode string

const ThermostatFanModeAuto ThermostatFanMode = "Auto low"
const ThermostatFanModeLow ThermostatFanMode = "Low"

type ThermostatState struct {
	Mode           ThermostatMode
	FanMode        ThermostatFanMode
	TargetTempHigh float64
	TargetTempLow  float64
}

func (c *RestClient) GetThermostatState(entityId string) (output ThermostatState, err error) {
	output = ThermostatState{
		Mode: ThermostatModeOff,
	}
	state, err := c.GetState(entityId)

	if err != nil {
		return output, err
	}

	defer func() {
		if panicErr := recover(); panicErr != nil {
			err = fmt.Errorf("error parsing attribute value: %v", panicErr)
		}
	}()

	switch state.State {
	case string(ThermostatModeHeat):
		output.Mode = ThermostatModeHeat
		output.TargetTempLow = state.Attributes["temperature"].(float64)
	case string(ThermostatModeCool):
		output.Mode = ThermostatModeCool
		output.TargetTempHigh = state.Attributes["temperature"].(float64)
	case string(ThermostatModeHeatCool):
		output.Mode = ThermostatModeHeatCool
		output.TargetTempHigh = state.Attributes["target_temp_high"].(float64)
		output.TargetTempLow = state.Attributes["target_temp_low"].(float64)
	default:
		output.Mode = ThermostatModeOff
	}

	switch state.Attributes["fan_mode"].(string) {
	case string(ThermostatFanModeLow):
		output.FanMode = ThermostatFanModeLow
	default:
		output.FanMode = ThermostatFanModeAuto
	}

	return output, err
}

func (c *RestClient) SetThermostatState(entityId string, desiredState ThermostatState) error {
	if desiredState.Mode == ThermostatModeOff {
		return c.CallService(entityId, Services.TurnOff)
	}

	if state, _ := c.GetState(entityId); !StateToBool(state.State) {
		err := c.CallService(entityId, Services.TurnOn)
		if err != nil {
			return fmt.Errorf("error turning on thermostat: %w", err)
		}
	}

	extras := map[string]any{
		"hvac_mode": string(desiredState.Mode),
	}

	switch desiredState.Mode {
	case ThermostatModeCool:
		extras["temperature"] = desiredState.TargetTempHigh
	case ThermostatModeHeat:
		extras["temperature"] = desiredState.TargetTempLow
	case ThermostatModeHeatCool:
		extras["target_temp_high"] = desiredState.TargetTempHigh
		extras["target_temp_low"] = desiredState.TargetTempLow
	}

	serviceCalls := []*CallServiceInput{{
		EntityID: entityId, Service: Services.SetTemperature, Extras: extras,
	}}

	if desiredState.FanMode != "" {
		serviceCalls = append(serviceCalls, &CallServiceInput{
			EntityID: entityId, Service: Services.SetFanMode, Extras: map[string]any{
				ExtraProps.FanMode: string(desiredState.FanMode),
			},
		})
	}

	return c.CallServices(serviceCalls...)
}

func (c *RestClient) SetThermostatFanMode(entityId string, mode ThermostatFanMode) error {
	return c.CallService(entityId, Services.SetFanMode, map[string]any{
		ExtraProps.FanMode: string(mode),
	})
}
