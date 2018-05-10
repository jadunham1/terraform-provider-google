package google

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/hashicorp/terraform/helper/resource"

	"google.golang.org/api/appengine/v1"
)

type AppEngineOperationWaiter struct {
	Service *appengine.APIService
	Op      *appengine.Operation
	AppId   string
}

func (w *AppEngineOperationWaiter) RefreshFunc() resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		op, err := w.Service.Apps.Operations.Get(w.AppId, w.Op.Name).Do()
		if err != nil {
			return nil, "", err
		}

		log.Printf("[DEBUG] Got %v when asking for operation %q", op.Done, w.Op.Name)
		return op, strconv.FormatBool(op.Done), nil
	}
}

func (w *AppEngineOperationWaiter) Conf() *resource.StateChangeConf {
	return &resource.StateChangeConf{
		Pending: []string{"false"},
		Target:  []string{"true"},
		Refresh: w.RefreshFunc(),
	}
}

// AppEngineOperationError wraps appengine.Status and implements the
// error interface so it can be returned.
type AppEngineOperationError appengine.Status

func (e AppEngineOperationError) Error() string {
	return e.Message
}

func appEngineOperationWait(client *appengine.APIService, op *appengine.Operation, appId, activity string) error {
	return appEngineOperationWaitTime(client, op, appId, activity, 4)
}

func appEngineOperationWaitTime(client *appengine.APIService, op *appengine.Operation, appId, activity string, timeoutMin int) error {
	w := &AppEngineOperationWaiter{
		Service: client,
		Op:      op,
		AppId:   appId,
	}

	state := w.Conf()
	state.Delay = 10 * time.Second
	state.Timeout = time.Duration(timeoutMin) * time.Minute
	state.MinTimeout = 2 * time.Second
	opRaw, err := state.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for %s: %s", activity, err)
	}

	resultOp := opRaw.(*appengine.Operation)
	if resultOp.Error != nil {
		return AppEngineOperationError(*resultOp.Error)
	}

	return nil
}