// The MIT License (MIT)

// Copyright (c) 2017-2020 Uber Technologies Inc.

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package errorinjectors

// Code generated by gowrap. DO NOT EDIT.
// template: template/errorinjector.tmpl
// gowrap: http://github.com/hexdigest/gowrap

import (
	"context"

	"github.com/uber/cadence/common/log"
	"github.com/uber/cadence/common/persistence"
)

// injectorQueueManager implements persistence.QueueManager interface instrumented with error injection.
type injectorQueueManager struct {
	wrapped   persistence.QueueManager
	errorRate float64
	logger    log.Logger
}

// NewQueueManager creates a new instance of QueueManager with error injection.
func NewQueueManager(
	wrapped persistence.QueueManager,
	errorRate float64,
	logger log.Logger,
) persistence.QueueManager {
	return &injectorQueueManager{
		wrapped:   wrapped,
		errorRate: errorRate,
		logger:    logger,
	}
}

func (c *injectorQueueManager) Close() {
	c.wrapped.Close()
	return
}

func (c *injectorQueueManager) DeleteMessageFromDLQ(ctx context.Context, messageID int64) (err error) {
	fakeErr := generateFakeError(c.errorRate)
	var forwardCall bool
	if forwardCall = shouldForwardCallToPersistence(fakeErr); forwardCall {
		err = c.wrapped.DeleteMessageFromDLQ(ctx, messageID)
	}

	if fakeErr != nil {
		logErr(c.logger, "QueueManager.DeleteMessageFromDLQ", fakeErr, forwardCall, err)
		err = fakeErr
		return
	}
	return
}

func (c *injectorQueueManager) DeleteMessagesBefore(ctx context.Context, messageID int64) (err error) {
	fakeErr := generateFakeError(c.errorRate)
	var forwardCall bool
	if forwardCall = shouldForwardCallToPersistence(fakeErr); forwardCall {
		err = c.wrapped.DeleteMessagesBefore(ctx, messageID)
	}

	if fakeErr != nil {
		logErr(c.logger, "QueueManager.DeleteMessagesBefore", fakeErr, forwardCall, err)
		err = fakeErr
		return
	}
	return
}

func (c *injectorQueueManager) EnqueueMessage(ctx context.Context, messagePayload []byte) (err error) {
	fakeErr := generateFakeError(c.errorRate)
	var forwardCall bool
	if forwardCall = shouldForwardCallToPersistence(fakeErr); forwardCall {
		err = c.wrapped.EnqueueMessage(ctx, messagePayload)
	}

	if fakeErr != nil {
		logErr(c.logger, "QueueManager.EnqueueMessage", fakeErr, forwardCall, err)
		err = fakeErr
		return
	}
	return
}

func (c *injectorQueueManager) EnqueueMessageToDLQ(ctx context.Context, messagePayload []byte) (err error) {
	fakeErr := generateFakeError(c.errorRate)
	var forwardCall bool
	if forwardCall = shouldForwardCallToPersistence(fakeErr); forwardCall {
		err = c.wrapped.EnqueueMessageToDLQ(ctx, messagePayload)
	}

	if fakeErr != nil {
		logErr(c.logger, "QueueManager.EnqueueMessageToDLQ", fakeErr, forwardCall, err)
		err = fakeErr
		return
	}
	return
}

func (c *injectorQueueManager) GetAckLevels(ctx context.Context) (m1 map[string]int64, err error) {
	fakeErr := generateFakeError(c.errorRate)
	var forwardCall bool
	if forwardCall = shouldForwardCallToPersistence(fakeErr); forwardCall {
		m1, err = c.wrapped.GetAckLevels(ctx)
	}

	if fakeErr != nil {
		logErr(c.logger, "QueueManager.GetAckLevels", fakeErr, forwardCall, err)
		err = fakeErr
		return
	}
	return
}

func (c *injectorQueueManager) GetDLQAckLevels(ctx context.Context) (m1 map[string]int64, err error) {
	fakeErr := generateFakeError(c.errorRate)
	var forwardCall bool
	if forwardCall = shouldForwardCallToPersistence(fakeErr); forwardCall {
		m1, err = c.wrapped.GetDLQAckLevels(ctx)
	}

	if fakeErr != nil {
		logErr(c.logger, "QueueManager.GetDLQAckLevels", fakeErr, forwardCall, err)
		err = fakeErr
		return
	}
	return
}

func (c *injectorQueueManager) GetDLQSize(ctx context.Context) (i1 int64, err error) {
	fakeErr := generateFakeError(c.errorRate)
	var forwardCall bool
	if forwardCall = shouldForwardCallToPersistence(fakeErr); forwardCall {
		i1, err = c.wrapped.GetDLQSize(ctx)
	}

	if fakeErr != nil {
		logErr(c.logger, "QueueManager.GetDLQSize", fakeErr, forwardCall, err)
		err = fakeErr
		return
	}
	return
}

func (c *injectorQueueManager) RangeDeleteMessagesFromDLQ(ctx context.Context, firstMessageID int64, lastMessageID int64) (err error) {
	fakeErr := generateFakeError(c.errorRate)
	var forwardCall bool
	if forwardCall = shouldForwardCallToPersistence(fakeErr); forwardCall {
		err = c.wrapped.RangeDeleteMessagesFromDLQ(ctx, firstMessageID, lastMessageID)
	}

	if fakeErr != nil {
		logErr(c.logger, "QueueManager.RangeDeleteMessagesFromDLQ", fakeErr, forwardCall, err)
		err = fakeErr
		return
	}
	return
}

func (c *injectorQueueManager) ReadMessages(ctx context.Context, lastMessageID int64, maxCount int) (qpa1 []*persistence.QueueMessage, err error) {
	fakeErr := generateFakeError(c.errorRate)
	var forwardCall bool
	if forwardCall = shouldForwardCallToPersistence(fakeErr); forwardCall {
		qpa1, err = c.wrapped.ReadMessages(ctx, lastMessageID, maxCount)
	}

	if fakeErr != nil {
		logErr(c.logger, "QueueManager.ReadMessages", fakeErr, forwardCall, err)
		err = fakeErr
		return
	}
	return
}

func (c *injectorQueueManager) ReadMessagesFromDLQ(ctx context.Context, firstMessageID int64, lastMessageID int64, pageSize int, pageToken []byte) (qpa1 []*persistence.QueueMessage, ba1 []byte, err error) {
	fakeErr := generateFakeError(c.errorRate)
	var forwardCall bool
	if forwardCall = shouldForwardCallToPersistence(fakeErr); forwardCall {
		qpa1, ba1, err = c.wrapped.ReadMessagesFromDLQ(ctx, firstMessageID, lastMessageID, pageSize, pageToken)
	}

	if fakeErr != nil {
		logErr(c.logger, "QueueManager.ReadMessagesFromDLQ", fakeErr, forwardCall, err)
		err = fakeErr
		return
	}
	return
}

func (c *injectorQueueManager) UpdateAckLevel(ctx context.Context, messageID int64, clusterName string) (err error) {
	fakeErr := generateFakeError(c.errorRate)
	var forwardCall bool
	if forwardCall = shouldForwardCallToPersistence(fakeErr); forwardCall {
		err = c.wrapped.UpdateAckLevel(ctx, messageID, clusterName)
	}

	if fakeErr != nil {
		logErr(c.logger, "QueueManager.UpdateAckLevel", fakeErr, forwardCall, err)
		err = fakeErr
		return
	}
	return
}

func (c *injectorQueueManager) UpdateDLQAckLevel(ctx context.Context, messageID int64, clusterName string) (err error) {
	fakeErr := generateFakeError(c.errorRate)
	var forwardCall bool
	if forwardCall = shouldForwardCallToPersistence(fakeErr); forwardCall {
		err = c.wrapped.UpdateDLQAckLevel(ctx, messageID, clusterName)
	}

	if fakeErr != nil {
		logErr(c.logger, "QueueManager.UpdateDLQAckLevel", fakeErr, forwardCall, err)
		err = fakeErr
		return
	}
	return
}
