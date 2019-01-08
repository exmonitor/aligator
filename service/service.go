package service

import (
	"sort"
	"time"

	"github.com/exmonitor/exclient/database"
	"github.com/exmonitor/exclient/database/spec/status"
	"github.com/exmonitor/exlogger"
	"github.com/pkg/errors"

	"github.com/exmonitor/aligator/agregator"
	"github.com/exmonitor/chronos"
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

	s.logger.Log("booting main loop")

	// run tick goroutine
	tickChan := make(chan bool)
	go intervalTick(int(s.loopIntervalSec), tickChan)

	// run infinite loop
	for {
		// wait until we reached another interval tick
		select {
		case <-tickChan:
			s.logger.Log("main loop received tick")
		}
		// time profiling
		t := chronos.New()
		// execute main loop
		err := s.mainLoop()
		// log time profiling
		t.Finish()
		if s.timeProfiling {
			s.logger.LogDebug("finished main loop in %sms", t.StringMilisec())
		}

		if err != nil {
			s.logger.LogError(err, "mainLoop failed")
		}
	}
}

var offsetFrom = time.Hour * 2400
var offsetTo = time.Minute * 6

func (s *Service) mainLoop() error {
	// work on interval (-2400h, -6m),
	// reason for -2400h - if services were not running fro some time there might some more records waiting for aggregation
	// reason for -6m - we still need to keep records  with age less than 5 min for notification service
	from := time.Now().Add(-offsetFrom)
	to := time.Now().Add(-offsetTo)

	// get all serviceStatus records that are yet not aggregated
	serviceStatusArray, err := s.dbClient.ES_GetServicesStatus(from, to)

	// sort array by serviceID
	sort.Slice(serviceStatusArray, func(i, j int) bool {
		return serviceStatusArray[i].Id < serviceStatusArray[j].Id
	})
	// array
	var usedIDs []int
	for _, serviceStatus := range serviceStatusArray {
		// check if this service ID was already used
		if isIntInArray(serviceStatus.Id, usedIDs) {
			// this service ID was already aggregated in previous run, just skip record
			continue
		} else {
			usedIDs = append(usedIDs, serviceStatus.Id)
		}
		lastAggregatedRecord, err := s.dbClient.ES_GetAggregatedServiceStatusByID(from, time.Now(), serviceStatus.Id)
		if err != nil {
			s.logger.LogError(err, "failed to get lastAggregatedRecord")
		}
		// get statuses with same serviceID and convert them to AggregatedStatus
		toAgregate := statusArrayToAggregatedStatusArray(getAllStatusesByID(serviceStatus.Id, serviceStatusArray))
		sort.Slice(toAgregate, func(i, j int) bool {
			return toAgregate[i].TimestampFrom.Before(toAgregate[j].TimestampFrom)
		})
		// insert last record at the start of the array
		toAgregate = append([]*status.AgregatedServiceStatus{lastAggregatedRecord}, toAgregate...)

		for i, item := range toAgregate {
			s.logger.Log("toAggregate item %d, content %s", i, item.String())
		}

		// finally do the data aggregation
		aggregatedStatusArray := agregator.AggregateStatuses(toAgregate)
		s.logger.Log("aggregated %d records into %d", len(toAgregate), len(aggregatedStatusArray))

		// save aggregated data back to db
		for i, item := range aggregatedStatusArray {
			s.logger.Log("DONE: aggregated item %d, content %s", i, item.String())
			err = s.dbClient.ES_SaveAggregatedServiceStatus(item)
			if err != nil {
				s.logger.LogError(err, "failed to save aggregatedStatus")
			}
		}
	}

	// clear aggregated records from db
	err = s.dbClient.ES_DeleteServicesStatus(from, to)
	if err != nil {

	}

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

// check if int is in array
func isIntInArray(i int, array []int) bool {
	for _, x := range array {
		if x == i {
			return true
		}
	}
	return false
}

// get a array of serviceStatus struct with same serviceID
func getAllStatusesByID(serviceID int, serviceStatusArray []*status.ServiceStatus) []*status.ServiceStatus {
	var result []*status.ServiceStatus
	for _, item := range serviceStatusArray {
		if item.Id == serviceID {
			result = append(result, item)
		}
	}
	return result
}

// convert array of []*ServiceStatus to []*AggregatedServiceStatus
func statusArrayToAggregatedStatusArray(statusArray []*status.ServiceStatus) []*status.AgregatedServiceStatus {
	// convert statuses to aggregated status
	var aggregatedStatusArray []*status.AgregatedServiceStatus
	for _, s := range statusArray {
		aggregatedStatusArray = append(aggregatedStatusArray, statusToAggregatedStatus(s))
	}
	return aggregatedStatusArray
}

// convert simple ServiceStatus to AggregatedServiceStatus
func statusToAggregatedStatus(s *status.ServiceStatus) *status.AgregatedServiceStatus {
	return &status.AgregatedServiceStatus{
		Id:            "",
		ServiceID:     s.Id,
		Interval:      s.Interval,
		Result:        s.Result,
		Aggregated:    1,
		AvgDuration:   s.Duration,
		TimestampFrom: s.InsertTimestamp,
		TimestampTo:   s.InsertTimestamp,
	}
}
