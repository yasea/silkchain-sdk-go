/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package comm

import (
	"context"
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	"google.golang.org/grpc/keepalive"

	fabcontext "github.com/hyperledger/fabric-sdk-go/pkg/context"
	"github.com/hyperledger/fabric-sdk-go/pkg/context/api/core"
	eventmocks "github.com/hyperledger/fabric-sdk-go/pkg/fab/events/mocks"
	fabmocks "github.com/hyperledger/fabric-sdk-go/pkg/fab/mocks"

	pb "github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/protos/peer"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

const (
	peerAddress = "localhost:9999"
	peerURL     = "grpc://" + peerAddress
)

var testStream = func(grpcconn *grpc.ClientConn) (grpc.ClientStream, error) {
	return pb.NewDeliverClient(grpcconn).Deliver(context.Background())
}

var invalidStream = func(grpcconn *grpc.ClientConn) (grpc.ClientStream, error) {
	return nil, errors.New("simulated error creating stream")
}

func TestConnection(t *testing.T) {
	channelID := "testchannel"

	context := newMockContext()

	conn, err := NewConnection(context, channelID, testStream, "")
	if err == nil {
		t.Fatalf("expected error creating new connection with empty URL")
	}
	conn, err = NewConnection(context, channelID, testStream, "invalidhost:0000",
		WithFailFast(true),
		WithCertificate(nil),
		WithHostOverride(""),
		WithKeepAliveParams(keepalive.ClientParameters{}),
		WithConnectTimeout(3*time.Second),
	)
	if err == nil {
		t.Fatalf("expected error creating new connection with invalid URL")
	}
	conn, err = NewConnection(context, channelID, invalidStream, peerURL)
	if err == nil {
		t.Fatalf("expected error creating new connection with invalid stream but got none")
	}

	conn, err = NewConnection(context, channelID, testStream, peerURL)
	if err != nil {
		t.Fatalf("error creating new connection: %s", err)
	}
	if conn.Closed() {
		t.Fatalf("expected connection to be open")
	}
	if conn.ChannelID() != channelID {
		t.Fatalf("expected channel ID [%s] but got [%s]", channelID, conn.ChannelID())
	}
	if conn.Stream() == nil {
		t.Fatalf("got invalid stream")
	}
	if _, err := context.Identity(); err != nil {
		t.Fatalf("error getting identity")
	}

	time.Sleep(1 * time.Second)

	conn.Close()
	if !conn.Closed() {
		t.Fatalf("expected connection to be closed")
	}

	// Calling close again should be ignored
	conn.Close()
}

// Use the Deliver server for testing
var testServer *eventmocks.MockEventhubServer

func TestMain(m *testing.M) {
	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)

	lis, err := net.Listen("tcp", peerAddress)
	if err != nil {
		panic(fmt.Sprintf("Error starting events listener %s", err))
	}

	testServer = eventmocks.NewMockEventhubServer()

	pb.RegisterEventsServer(grpcServer, testServer)

	go grpcServer.Serve(lis)

	time.Sleep(2 * time.Second)
	os.Exit(m.Run())
}

func newPeerConfig(peerURL string) *core.PeerConfig {
	return &core.PeerConfig{
		URL:         peerURL,
		GRPCOptions: make(map[string]interface{}),
	}
}

func newMockContext() fabcontext.Context {
	return fabmocks.NewMockContext(fabmocks.NewMockUser("test"))
}
