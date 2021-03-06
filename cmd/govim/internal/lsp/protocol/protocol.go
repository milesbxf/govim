// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package protocol

import (
	"context"

	"github.com/myitcv/govim/cmd/govim/internal/jsonrpc2"
	"github.com/myitcv/govim/cmd/govim/internal/lsp/telemetry/trace"
	"github.com/myitcv/govim/cmd/govim/internal/lsp/xlog"
)

const defaultMessageBufferSize = 20
const defaultRejectIfOverloaded = false

func canceller(ctx context.Context, conn *jsonrpc2.Conn, id jsonrpc2.ID) {
	ctx = detatchContext(ctx)
	ctx, span := trace.StartSpan(ctx, "protocol.canceller")
	defer span.End()
	conn.Notify(ctx, "$/cancelRequest", &CancelParams{ID: id})
}

func NewClient(stream jsonrpc2.Stream, client Client) (*jsonrpc2.Conn, Server, xlog.Logger) {
	log := xlog.New(NewLogger(client))
	conn := jsonrpc2.NewConn(stream)
	conn.Capacity = defaultMessageBufferSize
	conn.RejectIfOverloaded = defaultRejectIfOverloaded
	conn.Handler = clientHandler(log, client)
	conn.Canceler = jsonrpc2.Canceler(canceller)
	return conn, &serverDispatcher{Conn: conn}, log
}

func NewServer(stream jsonrpc2.Stream, server Server) (*jsonrpc2.Conn, Client, xlog.Logger) {
	conn := jsonrpc2.NewConn(stream)
	client := &clientDispatcher{Conn: conn}
	log := xlog.New(NewLogger(client))
	conn.Capacity = defaultMessageBufferSize
	conn.RejectIfOverloaded = defaultRejectIfOverloaded
	conn.Handler = serverHandler(log, server)
	conn.Canceler = jsonrpc2.Canceler(canceller)
	return conn, client, log
}

func sendParseError(ctx context.Context, log xlog.Logger, req *jsonrpc2.Request, err error) {
	if _, ok := err.(*jsonrpc2.Error); !ok {
		err = jsonrpc2.NewErrorf(jsonrpc2.CodeParseError, "%v", err)
	}
	if err := req.Reply(ctx, nil, err); err != nil {
		log.Errorf(ctx, "%v", err)
	}
}
