package validations

import (
	"time"

	constants "github.com/containersol/prescale-operator/internal"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// Time PreFilter prevents controllers with this prefilter within the first 10 seconds of startup of being triggered
func StartupFilter() predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			startTime := constants.StartTime
			if time.Since(startTime).Seconds() < 10 {
				return false
			}
			return true
		},
	}
}
