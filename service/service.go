package service

import (
	"github.com/exmonitor/exclient/database"
	"github.com/exmonitor/exlogger"
	"github.com/pkg/errors"
	"time"
)

type Config struct {
	DBClient        database.ClientInterface
	LoopIntervalSec int

	Logger        *exlogger.Logger
	TimeProfiling bool
}

func New(conf Config) (*Service, error) {
	if conf.DBClient == nil {
		return nil, errors.Wrap(invalidConfigError, "conf.DBClient must not be nil")
	}
	if conf.LoopIntervalSec == 0 {
		return nil, errors.Wrap(invalidConfigError, "conf.LoopIntervalSec must not be zero")
	}
	if conf.Logger == nil {
		return nil, errors.Wrap(invalidConfigError, "conf.Logger must not be nil")
	}

	newService := &Service{
		dbClient:        conf.DBClient,
		loopIntervalSec: conf.LoopIntervalSec,

		logger:        conf.Logger,
		timeProfiling: conf.TimeProfiling,
	}
	return newService, nil
}

type Service struct {
	dbClient        database.ClientInterface
	loopIntervalSec int

	logger        *exlogger.Logger
	timeProfiling bool
}

// make sure that the Loop is executed only once every x seconds defined in loopIntervalSec
func (s *Service) Boot() {

	// run tick goroutine
	tickChan := make(chan bool)
	go intervalTick(int(s.loopIntervalSec), tickChan)

	// run infinite loop
	for {
		// wait until we reached another interval tick
		select {
		case <-tickChan:
		}
		err := s.mainLoop()

		if err != nil {
			s.logger.LogError(err, "mainLoop failed")
		}
	}

}

func (s *Service) mainLoop() error {

	return nil
}

// send true to tickChan every intervalSec
func intervalTick(intervalSec int, tickChan chan bool) {
	for {
		// extract amount of second and minutes from the now time
		_, min, sec := time.Now().Clock()
		// get sum of total secs in hour as intervals can be bigger than 59 sec
		totalSeconds := min*60 + sec

		// check if we hit the interval
		if totalSeconds%intervalSec == 0 {
			// send msg to the channel that we got tick
			tickChan <- true
			time.Sleep(time.Second)
		}
		// this is rough value, so we are testing 10 times per sec to not have big offset
		time.Sleep(time.Millisecond * 100)
	}
}
