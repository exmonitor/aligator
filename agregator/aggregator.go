package agregator

import (
	"github.com/exmonitor/exclient/database/spec/status"
)

func AggregateStatuses(array []*status.AgregatedServiceStatus) []*status.AgregatedServiceStatus {
	var changed bool
	for {
		// set changes to false at start of each aggregate loop
		changed = false
		for i := 0; i < len(array); i++ {
			// we always compare items 'i' and 'i+1' so we need to be sure we are not out of array range
			if i+1 < len(array) {
				if array[i] == nil {
					// remove nil item from array
					array = append(array[:i], array[i+1:]...)
					// as array is now changed we should break and start over
					changed = true
					break
				}
				if array[i].Result == array[i+1].Result {
					// we have two records next to each other with same status result, we can aggregate them into one
					a := array[i]
					b := array[i+1]
					// same status result next to each other, we can merge them into one
					merged := &status.AgregatedServiceStatus{
						Id:            a.Id,
						ServiceID:     a.ServiceID,
						Aggregated:    a.Aggregated + b.Aggregated,
						Result:        a.Result,
						Interval:      a.Interval,
						TimestampFrom: a.TimestampFrom,
						TimestampTo:   b.TimestampTo,
					}
					// replace
					// should remove items 'i' and 'i+1' and put 'merged' item instead of them
					array = append(append(array[:i], merged), array[i+2:]...)
					// as array is now changed we should break and start over
					changed = true
					break
				}
			}
		}
		if !changed {
			//  if we iterate over whole array without change we can exit as there is no more possible change
			break
		}
	}
	return array
}
