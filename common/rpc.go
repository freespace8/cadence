// Copyright (c) 2017 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package common

import "go.uber.org/yarpc"

const (
	// LibraryVersionHeaderName refers to the name of the
	// tchannel / http header that contains the client
	// library version
	LibraryVersionHeaderName = "cadence-client-library-version"

	// FeatureVersionHeaderName refers to the name of the
	// tchannel / http header that contains the client
	// feature version
	// the feature version sent from client represents the
	// feature set of the cadence client library support.
	// This can be used for client capibility check, on
	// Cadence server, for backward compatibility
	FeatureVersionHeaderName = "cadence-client-feature-version"

	// Header name to pass the feature flags from customer to client or server
	ClientFeatureFlagsHeaderName = "cadence-client-feature-flags"

	// ClientImplHeaderName refers to the name of the
	// header that contains the client implementation
	ClientImplHeaderName = "cadence-client-name"
	// AuthorizationTokenHeaderName refers to the jwt token in the request
	AuthorizationTokenHeaderName = "cadence-authorization"

	// AutoforwardingClusterHeaderName refers to the name of the header that contains the source cluster of the auto-forwarding request
	AutoforwardingClusterHeaderName = "cadence-forwarding-cluster"

	// PartitionConfigHeaderName refers to the name of the header that contains the json encoded partition config of the request
	PartitionConfigHeaderName = "cadence-workflow-partition-config"
	// IsolationGroupHeaderName refers to the name of the header that contains the isolation group of the client
	IsolationGroupHeaderName = "cadence-worker-isolation-group"

	// ClientIsolationGroupHeaderName refers to the name of the header that contains the isolation group which the client request is from
	ClientIsolationGroupHeaderName = "cadence-client-isolation-group"
)

type (
	// RPCFactory Creates a dispatcher that knows how to transport requests.
	RPCFactory interface {
		GetDispatcher() *yarpc.Dispatcher
		GetMaxMessageSize() int
	}
)
