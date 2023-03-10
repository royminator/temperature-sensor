package temp

import (
	"bufio"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

const (
	MAX_RAW_READING = 4095
	MAX_TEMP        = 50
	MIN_TEMP        = -50
)

type (
	Sensor struct {
		TempSource *bufio.Scanner
		Ticker     *time.Ticker
		Quit       chan bool
	}

	ReadingsProcessor struct {
		Readings           chan TemperatureReading
		Quit               chan bool
		readingCount       uint
		measurement        TemperatureMeasurement
		PublishingInterval time.Duration
	}

	TemperatureReading struct {
		Temperature float64
		TimeStamp   time.Time
	}

	TemperatureMeasurement struct {
		Time    MeasurementTime `json:"time"`
		Min     float64         `json:"min"`
		Max     float64         `json:"max"`
		Average float64         `json:"avg"`
	}

	MeasurementTime struct {
		Start time.Time `json:"start"`
		End   time.Time `json:"end"`
	}
)

func NewReadingsProcessor(publishingInterval time.Duration) ReadingsProcessor {

	return ReadingsProcessor{
		Readings:           make(chan TemperatureReading, 1),
		PublishingInterval: publishingInterval,
		Quit:               make(chan bool, 1),
	}
}

func (processor *ReadingsProcessor) Run(measurements chan<- TemperatureMeasurement) {

	processor.reset()

	for {
		select {
		case <-processor.Quit:
			return
		default:
			reading := <-processor.Readings
			fmt.Println("reading: ", reading)
			processor.accumulate(reading)

			if processor.shouldPublish() {
				measurements <- processor.measurement
				processor.reset()
			}
		}
	}
}

func (processor *ReadingsProcessor) reset() {

	processor.readingCount = 0
	processor.measurement = TemperatureMeasurement{
		Time: MeasurementTime{
			time.Now().UTC(),
			time.Now().UTC(),
		},
		Min:     math.MaxFloat64,
		Max:     -math.MaxFloat64,
		Average: 0,
	}
}

func (processor *ReadingsProcessor) accumulate(reading TemperatureReading) {

	currMeasurement := processor.measurement
	startTime := currMeasurement.Time.Start

	if processor.readingCount == 0 {
		startTime = reading.TimeStamp
	}

	average := accumulateAverage(currMeasurement.Average, reading.Temperature, processor.readingCount)

	min := math.Min(currMeasurement.Min, reading.Temperature)
	max := math.Max(currMeasurement.Max, reading.Temperature)

	processor.readingCount++
	processor.measurement = TemperatureMeasurement{
		MeasurementTime{startTime, reading.TimeStamp},
		round2(min),
		round2(max),
		round2(average),
	}
}

func (processor ReadingsProcessor) shouldPublish() bool {

	startTime := processor.measurement.Time.Start
	endTime := processor.measurement.Time.End
	return endTime.Sub(startTime) >= processor.PublishingInterval
}

func NewSensor(tempScanner *bufio.Scanner, ticker *time.Ticker) Sensor {

	return Sensor{
		TempSource: tempScanner,
		Ticker:     time.NewTicker(time.Millisecond * 100),
		Quit:       make(chan bool, 1),
	}
}

func (sensor Sensor) Start(readings chan<- TemperatureReading) {

	for sensor.TempSource.Scan() {
		temp := sensor.getTemperature()
		timeStamp := time.Now().UTC()
		readings <- TemperatureReading{temp, timeStamp}
	}

	sensor.Quit <- true
}

func (sensor Sensor) getTemperature() float64 {

	<-sensor.Ticker.C
	temp := sensor.readNext()
	return rawTempToFloat(temp)
}

func (sensor Sensor) readNext() uint {

	tempStr := strings.TrimSpace(sensor.TempSource.Text())
	temp, _ := strconv.ParseUint(tempStr, 10, 16)
	return uint(temp)
}

func rawTempToFloat(raw uint) float64 {
	return lerp(float64(raw)/MAX_RAW_READING, MIN_TEMP, MAX_TEMP)
}

func lerp(val float64, min float64, max float64) float64 {
	return val*(max-min) + min
}

func accumulateAverage(avg float64, val float64, n uint) float64 {
	return avg + (val-avg)/float64(n+1)
}

func round2(val float64) float64 {
	return math.Round(val*100) / 100
}
