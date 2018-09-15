// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package jobs

import (
	"github.com/c3systems/c3-sdk-go-example-mattermost/model"
)

type MessageExportJobInterface interface {
	MakeWorker() model.Worker
	MakeScheduler() model.Scheduler
}
