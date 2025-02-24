// The MIT License (MIT)
//
// Copyright (c) 2020 Uber Technologies, Inc.
//
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

package common

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/yarpcerrors"

	"github.com/uber/cadence/common/backoff"
	"github.com/uber/cadence/common/log"
	"github.com/uber/cadence/common/log/tag"
	"github.com/uber/cadence/common/metrics"
	"github.com/uber/cadence/common/types"
)

func TestIsServiceTransientError_ContextTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	time.Sleep(100 * time.Millisecond)

	require.False(t, IsServiceTransientError(ctx.Err()))
}

func TestIsServiceTransientError_YARPCDeadlineExceeded(t *testing.T) {
	yarpcErr := yarpcerrors.DeadlineExceededErrorf("yarpc deadline exceeded")
	require.False(t, IsServiceTransientError(yarpcErr))
}

func TestIsServiceTransientError_ContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	require.False(t, IsServiceTransientError(ctx.Err()))
}

func TestIsContextTimeoutError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	time.Sleep(50 * time.Millisecond)
	require.True(t, IsContextTimeoutError(ctx.Err()))
	require.True(t, IsContextTimeoutError(&types.InternalServiceError{Message: ctx.Err().Error()}))

	yarpcErr := yarpcerrors.DeadlineExceededErrorf("yarpc deadline exceeded")
	require.True(t, IsContextTimeoutError(yarpcErr))

	require.False(t, IsContextTimeoutError(errors.New("some random error")))

	ctx, cancel = context.WithCancel(context.Background())
	cancel()
	require.False(t, IsContextTimeoutError(ctx.Err()))
}

func TestConvertDynamicConfigMapPropertyToIntMap(t *testing.T) {
	dcValue := make(map[string]interface{})
	for idx, value := range []interface{}{int(0), int32(1), int64(2), float64(3.0)} {
		dcValue[strconv.Itoa(idx)] = value
	}

	intMap, err := ConvertDynamicConfigMapPropertyToIntMap(dcValue)
	require.NoError(t, err)
	require.Len(t, intMap, 4)
	for i := 0; i != 4; i++ {
		require.Equal(t, i, intMap[i])
	}
}

func TestCreateHistoryStartWorkflowRequest_ExpirationTimeWithCron(t *testing.T) {
	domainID := uuid.New()
	request := &types.StartWorkflowExecutionRequest{
		RetryPolicy: &types.RetryPolicy{
			InitialIntervalInSeconds:    60,
			ExpirationIntervalInSeconds: 60,
		},
		CronSchedule: "@every 300s",
	}
	now := time.Now()
	startRequest, err := CreateHistoryStartWorkflowRequest(domainID, request, now, nil)
	require.NoError(t, err)
	require.NotNil(t, startRequest)

	expirationTime := startRequest.GetExpirationTimestamp()
	require.NotNil(t, expirationTime)
	require.True(t, time.Unix(0, expirationTime).Sub(now) > 60*time.Second)
}

// Test to ensure we get the right value for FirstDecisionTaskBackoff during StartWorkflow request,
// with & without cron, delayStart and jitterStart.
// - Also see tests in cron_test.go for more exhaustive testing.
func TestFirstDecisionTaskBackoffDuringStartWorkflow(t *testing.T) {
	var tests = []struct {
		cron               bool
		jitterStartSeconds int32
		delayStartSeconds  int32
	}{
		{true, 0, 0},
		{true, 15, 0},
		{true, 0, 600},
		{true, 15, 600},
		{false, 0, 0},
		{false, 15, 0},
		{false, 0, 600},
		{false, 15, 600},
	}

	rand.Seed(int64(time.Now().Nanosecond()))

	for idx, tt := range tests {
		t.Run(strconv.Itoa(idx), func(t *testing.T) {
			domainID := uuid.New()
			request := &types.StartWorkflowExecutionRequest{
				DelayStartSeconds:  Int32Ptr(tt.delayStartSeconds),
				JitterStartSeconds: Int32Ptr(tt.jitterStartSeconds),
			}
			if tt.cron {
				request.CronSchedule = "* * * * *"
			}

			// Run X loops so that the test isn't flaky, since jitter adds randomness.
			caseCount := 10
			exactCount := 0
			for i := 0; i < caseCount; i++ {
				// Start at the minute boundary so we know what the backoff should be
				startTime, _ := time.Parse(time.RFC3339, "2018-12-17T08:00:00+00:00")
				startRequest, err := CreateHistoryStartWorkflowRequest(domainID, request, startTime, nil)
				require.NoError(t, err)
				require.NotNil(t, startRequest)

				backoff := startRequest.GetFirstDecisionTaskBackoffSeconds()

				expectedWithoutJitter := tt.delayStartSeconds
				if tt.cron {
					expectedWithoutJitter += 60
				}
				expectedMax := expectedWithoutJitter + tt.jitterStartSeconds

				if backoff == expectedWithoutJitter {
					exactCount++
				}

				if tt.jitterStartSeconds == 0 {
					require.Equal(t, expectedWithoutJitter, backoff, "test specs = %v", tt)
				} else {
					require.True(t, backoff >= expectedWithoutJitter && backoff <= expectedMax,
						"test specs = %v, backoff (%v) should be >= %v and <= %v",
						tt, backoff, expectedWithoutJitter, expectedMax)
				}

			}

			// If jitter is > 0, we want to detect whether jitter is being applied - BUT we don't want the test
			// to be flaky if the code randomly chooses a jitter of 0, so we try to have enough data points by
			// checking the next X cron times AND by choosing a jitter thats not super low.

			if tt.jitterStartSeconds > 0 && exactCount == caseCount {
				// Test to make sure a jitter test case sometimes doesn't get exact values
				t.Fatalf("FAILED to jitter properly? Test specs = %v\n", tt)
			} else if tt.jitterStartSeconds == 0 && exactCount != caseCount {
				// Test to make sure a non-jitter test case always gets exact values
				t.Fatalf("Jittered when we weren't supposed to? Test specs = %v\n", tt)
			}

		})
	}
}

func TestCreateHistoryStartWorkflowRequest(t *testing.T) {
	var tests = []struct {
		delayStartSeconds  int
		cronSeconds        int
		jitterStartSeconds int
	}{
		{0, 0, 0},
		{100, 0, 0},
		{100, 300, 0},
		{0, 0, 2000},
		{100, 0, 2000},
		{0, 300, 2000},
		{100, 300, 2000},
		{0, 300, 2000},
		{0, 300, 2000},
	}

	for idx, tt := range tests {
		t.Run(strconv.Itoa(idx), func(t *testing.T) {
			testExpirationTime(t, tt.delayStartSeconds, tt.cronSeconds, tt.jitterStartSeconds)
		})
	}
}

func testExpirationTime(t *testing.T, delayStartSeconds int, cronSeconds int, jitterSeconds int) {
	domainID := uuid.New()
	request := &types.StartWorkflowExecutionRequest{
		RetryPolicy: &types.RetryPolicy{
			InitialIntervalInSeconds:    60,
			ExpirationIntervalInSeconds: 60,
		},
		DelayStartSeconds:  Int32Ptr(int32(delayStartSeconds)),
		JitterStartSeconds: Int32Ptr(int32(jitterSeconds)),
	}
	if cronSeconds > 0 {
		request.CronSchedule = fmt.Sprintf("@every %ds", cronSeconds)
	}
	minDelay := delayStartSeconds + cronSeconds
	maxDelay := delayStartSeconds + 2*cronSeconds + jitterSeconds
	now := time.Now()
	startRequest, err := CreateHistoryStartWorkflowRequest(domainID, request, now, nil)
	require.NoError(t, err)
	require.NotNil(t, startRequest)

	expirationTime := startRequest.GetExpirationTimestamp()
	require.NotNil(t, expirationTime)

	// Since we assign the expiration time after we create the workflow request,
	// There's a chance that the test thread might sleep or get deprioritized and
	// expirationTime - now may not be equal to DelayStartSeconds. Adding 2 seconds
	// buffer to avoid this test being flaky
	require.True(
		t,
		time.Unix(0, expirationTime).Sub(now) >= (time.Duration(minDelay)+58)*time.Second,
		"Integration test took too short: %f seconds vs %f seconds",
		time.Duration(time.Unix(0, expirationTime).Sub(now)).Round(time.Millisecond).Seconds(),
		time.Duration((time.Duration(minDelay)+58)*time.Second).Round(time.Millisecond).Seconds(),
	)
	require.True(
		t,
		time.Unix(0, expirationTime).Sub(now) < (time.Duration(maxDelay)+68)*time.Second,
		"Integration test took too long: %f seconds vs %f seconds",
		time.Duration(time.Unix(0, expirationTime).Sub(now)).Round(time.Millisecond).Seconds(),
		time.Duration((time.Duration(minDelay)+68)*time.Second).Round(time.Millisecond).Seconds(),
	)
}

func TestCreateHistoryStartWorkflowRequest_ExpirationTimeWithoutCron(t *testing.T) {
	domainID := uuid.New()
	request := &types.StartWorkflowExecutionRequest{
		RetryPolicy: &types.RetryPolicy{
			InitialIntervalInSeconds:    60,
			ExpirationIntervalInSeconds: 60,
		},
	}
	now := time.Now()
	startRequest, err := CreateHistoryStartWorkflowRequest(domainID, request, now, nil)
	require.NoError(t, err)
	require.NotNil(t, startRequest)

	expirationTime := startRequest.GetExpirationTimestamp()
	require.NotNil(t, expirationTime)
	delta := time.Unix(0, expirationTime).Sub(now)
	require.True(t, delta > 58*time.Second)
	require.True(t, delta < 62*time.Second)
}

func TestConvertIndexedValueTypeToInternalType(t *testing.T) {
	values := []types.IndexedValueType{types.IndexedValueTypeString, types.IndexedValueTypeKeyword, types.IndexedValueTypeInt, types.IndexedValueTypeDouble, types.IndexedValueTypeBool, types.IndexedValueTypeDatetime}
	for _, expected := range values {
		require.Equal(t, expected, ConvertIndexedValueTypeToInternalType(int(expected), nil))
		require.Equal(t, expected, ConvertIndexedValueTypeToInternalType(float64(expected), nil))

		buffer, err := expected.MarshalText()
		require.NoError(t, err)
		require.Equal(t, expected, ConvertIndexedValueTypeToInternalType(buffer, nil))
		require.Equal(t, expected, ConvertIndexedValueTypeToInternalType(string(buffer), nil))
	}
}

func TestValidateDomainUUID(t *testing.T) {
	testCases := []struct {
		msg        string
		domainUUID string
		valid      bool
	}{
		{
			msg:        "empty",
			domainUUID: "",
			valid:      false,
		},
		{
			msg:        "invalid",
			domainUUID: "some random uuid",
			valid:      false,
		},
		{
			msg:        "valid",
			domainUUID: uuid.New(),
			valid:      true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.msg, func(t *testing.T) {
			err := ValidateDomainUUID(tc.domainUUID)
			if tc.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestConvertErrToGetTaskFailedCause(t *testing.T) {
	testCases := []struct {
		err                 error
		expectedFailedCause types.GetTaskFailedCause
	}{
		{
			err:                 errors.New("some random error"),
			expectedFailedCause: types.GetTaskFailedCauseUncategorized,
		},
		{
			err:                 context.DeadlineExceeded,
			expectedFailedCause: types.GetTaskFailedCauseTimeout,
		},
		{
			err:                 &types.ServiceBusyError{},
			expectedFailedCause: types.GetTaskFailedCauseServiceBusy,
		},
		{
			err:                 &types.ShardOwnershipLostError{},
			expectedFailedCause: types.GetTaskFailedCauseShardOwnershipLost,
		},
	}

	for _, tc := range testCases {
		require.Equal(t, tc.expectedFailedCause, ConvertErrToGetTaskFailedCause(tc.err))
	}
}

func TestToServiceTransientError(t *testing.T) {
	t.Run("it converts nil", func(t *testing.T) {
		assert.NoError(t, ToServiceTransientError(nil))
	})

	t.Run("it keeps transient errors", func(t *testing.T) {
		err := &types.InternalServiceError{}
		assert.Equal(t, err, ToServiceTransientError(err))
		assert.True(t, IsServiceTransientError(ToServiceTransientError(err)))
	})

	t.Run("it converts errors to transient errors", func(t *testing.T) {
		err := fmt.Errorf("error")
		assert.True(t, IsServiceTransientError(ToServiceTransientError(err)))
	})
}

func TestIntersectionStringSlice(t *testing.T) {
	t.Run("it returns all items", func(t *testing.T) {
		a := []string{"a", "b", "c"}
		b := []string{"a", "b", "c"}
		c := IntersectionStringSlice(a, b)
		assert.ElementsMatch(t, []string{"a", "b", "c"}, c)
	})

	t.Run("it returns no item", func(t *testing.T) {
		a := []string{"a", "b", "c"}
		b := []string{"d", "e", "f"}
		c := IntersectionStringSlice(a, b)
		assert.ElementsMatch(t, []string{}, c)
	})

	t.Run("it returns intersection", func(t *testing.T) {
		a := []string{"a", "b", "c"}
		b := []string{"c", "b", "f"}
		c := IntersectionStringSlice(a, b)
		assert.ElementsMatch(t, []string{"c", "b"}, c)
	})
}

func TestAwaitWaitGroup(t *testing.T) {
	t.Run("wait group done before timeout", func(t *testing.T) {
		var wg sync.WaitGroup

		wg.Add(1)
		wg.Done()

		got := AwaitWaitGroup(&wg, time.Second)
		require.True(t, got)
	})

	t.Run("wait group done after timeout", func(t *testing.T) {
		var (
			wg    sync.WaitGroup
			doneC = make(chan struct{})
		)

		wg.Add(1)
		go func() {
			<-doneC
			wg.Done()
		}()

		got := AwaitWaitGroup(&wg, time.Microsecond)
		require.False(t, got)

		doneC <- struct{}{}
		close(doneC)
	})
}

func TestIsValidIDLength(t *testing.T) {
	var (
		// test setup
		scope = metrics.NoopScope(0)

		// arguments
		metricCounter      = 0
		idTypeViolationTag = tag.ClusterName("idTypeViolationTag")
		domainName         = "domain_name"
		id                 = "12345"
	)

	mockWarnCall := func(logger *log.MockLogger) {
		logger.On(
			"Warn",
			"ID length exceeds limit.",
			[]tag.Tag{
				tag.WorkflowDomainName(domainName),
				tag.Name(id),
				idTypeViolationTag,
			},
		).Once()
	}

	t.Run("valid id length, no warnings", func(t *testing.T) {
		logger := new(log.MockLogger)
		got := IsValidIDLength(id, scope, 7, 10, metricCounter, domainName, logger, idTypeViolationTag)
		require.True(t, got, "expected true, because id length is 5 and it's less than error limit 10")
	})

	t.Run("valid id length, with warnings", func(t *testing.T) {
		logger := new(log.MockLogger)
		mockWarnCall(logger)

		got := IsValidIDLength(id, scope, 4, 10, metricCounter, domainName, logger, idTypeViolationTag)
		require.True(t, got, "expected true, because id length is 5 and it's less than error limit 10")

		// logger should be called once
		logger.AssertExpectations(t)
	})

	t.Run("non valid id length", func(t *testing.T) {
		logger := new(log.MockLogger)
		mockWarnCall(logger)

		got := IsValidIDLength(id, scope, 1, 4, metricCounter, domainName, logger, idTypeViolationTag)
		require.False(t, got, "expected false, because id length is 5 and it's more than error limit 4")

		// logger should be called once
		logger.AssertExpectations(t)
	})
}

func TestIsEntityNotExistsError(t *testing.T) {
	t.Run("is entity not exists error", func(t *testing.T) {
		err := &types.EntityNotExistsError{}
		require.True(t, IsEntityNotExistsError(err), "expected true, because err is entity not exists error")
	})

	t.Run("is not entity not exists error", func(t *testing.T) {
		err := fmt.Errorf("generic error")
		require.False(t, IsEntityNotExistsError(err), "expected false, because err is a generic error")
	})
}

func TestCreateXXXRetryPolicyWithSetExpirationInterval(t *testing.T) {
	for name, c := range map[string]struct {
		createFn func() backoff.RetryPolicy

		wantInitialInterval       time.Duration
		wantMaximumInterval       time.Duration
		wantSetExpirationInterval time.Duration
	}{
		"CreatePersistenceRetryPolicy": {
			createFn:                  CreatePersistenceRetryPolicy,
			wantInitialInterval:       retryPersistenceOperationInitialInterval,
			wantMaximumInterval:       retryPersistenceOperationMaxInterval,
			wantSetExpirationInterval: retryPersistenceOperationExpirationInterval,
		},
		"CreateHistoryServiceRetryPolicy": {
			createFn:                  CreateHistoryServiceRetryPolicy,
			wantInitialInterval:       historyServiceOperationInitialInterval,
			wantMaximumInterval:       historyServiceOperationMaxInterval,
			wantSetExpirationInterval: historyServiceOperationExpirationInterval,
		},
		"CreateMatchingServiceRetryPolicy": {
			createFn:                  CreateMatchingServiceRetryPolicy,
			wantInitialInterval:       matchingServiceOperationInitialInterval,
			wantMaximumInterval:       matchingServiceOperationMaxInterval,
			wantSetExpirationInterval: matchingServiceOperationExpirationInterval,
		},
		"CreateFrontendServiceRetryPolicy": {
			createFn:                  CreateFrontendServiceRetryPolicy,
			wantInitialInterval:       frontendServiceOperationInitialInterval,
			wantMaximumInterval:       frontendServiceOperationMaxInterval,
			wantSetExpirationInterval: frontendServiceOperationExpirationInterval,
		},
		"CreateAdminServiceRetryPolicy": {
			createFn:                  CreateAdminServiceRetryPolicy,
			wantInitialInterval:       adminServiceOperationInitialInterval,
			wantMaximumInterval:       adminServiceOperationMaxInterval,
			wantSetExpirationInterval: adminServiceOperationExpirationInterval,
		},
		"CreateReplicationServiceBusyRetryPolicy": {
			createFn:                  CreateReplicationServiceBusyRetryPolicy,
			wantInitialInterval:       replicationServiceBusyInitialInterval,
			wantMaximumInterval:       replicationServiceBusyMaxInterval,
			wantSetExpirationInterval: replicationServiceBusyExpirationInterval,
		},
	} {
		t.Run(name, func(t *testing.T) {
			want := backoff.NewExponentialRetryPolicy(c.wantInitialInterval)
			want.SetMaximumInterval(c.wantMaximumInterval)
			want.SetExpirationInterval(c.wantSetExpirationInterval)
			got := c.createFn()
			require.Equal(t, want, got)
		})
	}
}

func TestCreateXXXRetryPolicyWithMaximumAttempts(t *testing.T) {
	for name, c := range map[string]struct {
		createFn func() backoff.RetryPolicy

		wantInitialInterval time.Duration
		wantMaximumInterval time.Duration
		wantMaximumAttempts int
	}{
		"CreateDlqPublishRetryPolicy": {
			createFn:            CreateDlqPublishRetryPolicy,
			wantInitialInterval: retryKafkaOperationInitialInterval,
			wantMaximumInterval: retryKafkaOperationMaxInterval,
			wantMaximumAttempts: retryKafkaOperationMaxAttempts,
		},
		"CreateTaskProcessingRetryPolicy": {
			createFn:            CreateTaskProcessingRetryPolicy,
			wantInitialInterval: retryTaskProcessingInitialInterval,
			wantMaximumInterval: retryTaskProcessingMaxInterval,
			wantMaximumAttempts: retryTaskProcessingMaxAttempts,
		},
	} {
		t.Run(name, func(t *testing.T) {
			want := backoff.NewExponentialRetryPolicy(c.wantInitialInterval)
			want.SetMaximumInterval(c.wantMaximumInterval)
			want.SetMaximumAttempts(c.wantMaximumAttempts)
			got := c.createFn()
			require.Equal(t, want, got)
		})
	}
}

func TestValidateRetryPolicy_Success(t *testing.T) {
	for name, policy := range map[string]*types.RetryPolicy{
		"nil policy": nil,
		"MaximumAttempts is no zero": &types.RetryPolicy{
			InitialIntervalInSeconds:    2,
			BackoffCoefficient:          1,
			MaximumIntervalInSeconds:    0,
			MaximumAttempts:             1,
			ExpirationIntervalInSeconds: 0,
		},
		"ExpirationIntervalInSeconds is no zero": &types.RetryPolicy{
			InitialIntervalInSeconds:    2,
			BackoffCoefficient:          1,
			MaximumIntervalInSeconds:    0,
			MaximumAttempts:             0,
			ExpirationIntervalInSeconds: 1,
		},
		"MaximumIntervalInSeconds is greater than InitialIntervalInSeconds": &types.RetryPolicy{
			InitialIntervalInSeconds:    2,
			BackoffCoefficient:          1,
			MaximumIntervalInSeconds:    0,
			MaximumAttempts:             0,
			ExpirationIntervalInSeconds: 1,
		},
		"MaximumIntervalInSeconds equals InitialIntervalInSeconds": &types.RetryPolicy{
			InitialIntervalInSeconds:    2,
			BackoffCoefficient:          1,
			MaximumIntervalInSeconds:    2,
			MaximumAttempts:             0,
			ExpirationIntervalInSeconds: 1,
		},
	} {
		t.Run(name, func(t *testing.T) {
			require.NoError(t, ValidateRetryPolicy(policy))
		})
	}
}

func TestValidateRetryPolicy_Error(t *testing.T) {
	for name, c := range map[string]struct {
		policy  *types.RetryPolicy
		wantErr *types.BadRequestError
	}{
		"InitialIntervalInSeconds equals 0": {
			policy: &types.RetryPolicy{
				InitialIntervalInSeconds: 0,
			},
			wantErr: &types.BadRequestError{Message: "InitialIntervalInSeconds must be greater than 0 on retry policy."},
		},
		"InitialIntervalInSeconds less than 0": {
			policy: &types.RetryPolicy{
				InitialIntervalInSeconds: -1,
			},
			wantErr: &types.BadRequestError{Message: "InitialIntervalInSeconds must be greater than 0 on retry policy."},
		},
		"BackoffCoefficient equals 0": {
			policy: &types.RetryPolicy{
				InitialIntervalInSeconds: 1,
				BackoffCoefficient:       0,
			},
			wantErr: &types.BadRequestError{Message: "BackoffCoefficient cannot be less than 1 on retry policy."},
		},
		"MaximumIntervalInSeconds equals -1": {
			policy: &types.RetryPolicy{
				InitialIntervalInSeconds: 1,
				BackoffCoefficient:       1,
				MaximumIntervalInSeconds: -1,
			},
			wantErr: &types.BadRequestError{Message: "MaximumIntervalInSeconds cannot be less than 0 on retry policy."},
		},
		"MaximumIntervalInSeconds equals 1 and less than InitialIntervalInSeconds": {
			policy: &types.RetryPolicy{
				InitialIntervalInSeconds: 2,
				BackoffCoefficient:       1,
				MaximumIntervalInSeconds: 1,
			},
			wantErr: &types.BadRequestError{Message: "MaximumIntervalInSeconds cannot be less than InitialIntervalInSeconds on retry policy."},
		},
		"MaximumAttempts equals -1": {
			policy: &types.RetryPolicy{
				InitialIntervalInSeconds: 2,
				BackoffCoefficient:       1,
				MaximumIntervalInSeconds: 0,
				MaximumAttempts:          -1,
			},
			wantErr: &types.BadRequestError{Message: "MaximumAttempts cannot be less than 0 on retry policy."},
		},
		"ExpirationIntervalInSeconds equals -1": {
			policy: &types.RetryPolicy{
				InitialIntervalInSeconds:    2,
				BackoffCoefficient:          1,
				MaximumIntervalInSeconds:    0,
				MaximumAttempts:             0,
				ExpirationIntervalInSeconds: -1,
			},
			wantErr: &types.BadRequestError{Message: "ExpirationIntervalInSeconds cannot be less than 0 on retry policy."},
		},
		"MaximumAttempts and ExpirationIntervalInSeconds equal 0": {
			policy: &types.RetryPolicy{
				InitialIntervalInSeconds:    2,
				BackoffCoefficient:          1,
				MaximumIntervalInSeconds:    0,
				MaximumAttempts:             0,
				ExpirationIntervalInSeconds: 0,
			},
			wantErr: &types.BadRequestError{Message: "MaximumAttempts and ExpirationIntervalInSeconds are both 0. At least one of them must be specified."},
		},
	} {
		t.Run(name, func(t *testing.T) {
			got := ValidateRetryPolicy(c.policy)
			require.Error(t, got)
			require.ErrorContains(t, got, c.wantErr.Message)
		})
	}
}

func TestConvertGetTaskFailedCauseToErr(t *testing.T) {
	for cause, wantErr := range map[types.GetTaskFailedCause]error{
		types.GetTaskFailedCauseServiceBusy:        &types.ServiceBusyError{},
		types.GetTaskFailedCauseTimeout:            context.DeadlineExceeded,
		types.GetTaskFailedCauseShardOwnershipLost: &types.ShardOwnershipLostError{},
		types.GetTaskFailedCauseUncategorized:      &types.InternalServiceError{Message: "uncategorized error"},
	} {
		t.Run(cause.String(), func(t *testing.T) {
			gotErr := ConvertGetTaskFailedCauseToErr(cause)
			require.Equal(t, wantErr, gotErr)
		})
	}
}

func TestWorkflowIDToHistoryShard(t *testing.T) {
	for _, c := range []struct {
		workflowID     string
		numberOfShards int

		want int
	}{
		{
			workflowID:     "",
			numberOfShards: 1000,
			want:           242,
		},
		{
			workflowID:     "workflowId",
			numberOfShards: 1000,
			want:           580,
		},
	} {
		t.Run(fmt.Sprintf("%s-%v", c.workflowID, c.numberOfShards), func(t *testing.T) {
			got := WorkflowIDToHistoryShard(c.workflowID, c.numberOfShards)
			require.Equal(t, c.want, got)
		})
	}
}

func TestDomainIDToHistoryShard(t *testing.T) {
	for _, c := range []struct {
		domainID       string
		numberOfShards int

		want int
	}{
		{
			domainID:       "",
			numberOfShards: 1000,
			want:           242,
		},
		{
			domainID:       "domainId",
			numberOfShards: 1000,
			want:           600,
		},
	} {
		t.Run(fmt.Sprintf("%s-%v", c.domainID, c.numberOfShards), func(t *testing.T) {
			got := DomainIDToHistoryShard(c.domainID, c.numberOfShards)
			require.Equal(t, c.want, got)
		})
	}
}
