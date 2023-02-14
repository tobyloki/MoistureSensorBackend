package main

const MINUTE = "MINUTES"
const HOUR = "HOURS"
const DAY = "DAYS"

// create enum for granularityUnit of minute, hour, day from json
type GranularityUnit string

const (
	Minute GranularityUnit = MINUTE
	Hour   GranularityUnit = HOUR
	Day    GranularityUnit = DAY
)

type SchedulerMessage struct {
	// name of the key must be capitalized to be exported
	SensorThingName  string          `json:"sensorThingName"`
	ActuatorId       string          `json:"actuatorId"`
	GranularityValue int             `json:"granularityValue"`
	GranularityUnit  GranularityUnit `json:"granularityUnit"`
}

type ResetActuatorMessage struct {
	// name of the key must be capitalized to be exported
	SensorThingName string `json:"sensorThingName"`
	ActuatorId      string `json:"actuatorId"`
}

type UpdateActuatorMessage struct {
	// name of the key must be capitalized to be exported
	ActuatorId string `json:"actuatorId"`
	Key        string `json:"key"`
	Value      string `json:"value`
}
