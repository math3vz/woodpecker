// Copyright 2023 Woodpecker Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package rpc

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// AuthInterceptor is a client interceptor for authentication
type AuthInterceptor struct {
	authClient  *AuthClient
	accessToken string
}

// NewAuthInterceptor returns a new auth interceptor
func NewAuthInterceptor(
	authClient *AuthClient,
	refreshDuration time.Duration,
) (*AuthInterceptor, error) {
	interceptor := &AuthInterceptor{
		authClient: authClient,
	}

	err := interceptor.scheduleRefreshToken(refreshDuration)
	if err != nil {
		return nil, err
	}

	return interceptor, nil
}

// Unary returns a client interceptor to authenticate unary RPC
func (interceptor *AuthInterceptor) Unary() grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		return invoker(interceptor.attachToken(ctx), method, req, reply, cc, opts...)
	}
}

// Stream returns a client interceptor to authenticate stream RPC
func (interceptor *AuthInterceptor) Stream() grpc.StreamClientInterceptor {
	return func(
		ctx context.Context,
		desc *grpc.StreamDesc,
		cc *grpc.ClientConn,
		method string,
		streamer grpc.Streamer,
		opts ...grpc.CallOption,
	) (grpc.ClientStream, error) {
		return streamer(interceptor.attachToken(ctx), desc, cc, method, opts...)
	}
}

func (interceptor *AuthInterceptor) attachToken(ctx context.Context) context.Context {
	return metadata.AppendToOutgoingContext(ctx, "token", interceptor.accessToken)
}

func (interceptor *AuthInterceptor) scheduleRefreshToken(refreshDuration time.Duration) error {
	err := interceptor.refreshToken()
	if err != nil {
		return err
	}

	go func() {
		wait := refreshDuration
		for {
			time.Sleep(wait)
			err := interceptor.refreshToken()
			if err != nil {
				wait = time.Second
			} else {
				wait = refreshDuration
			}
		}
	}()

	return nil
}

func (interceptor *AuthInterceptor) refreshToken() error {
	accessToken, _, err := interceptor.authClient.Auth()
	if err != nil {
		return err
	}

	interceptor.accessToken = accessToken
	log.Trace().Str("token", accessToken).Msg("Token refreshed")

	return nil
}
