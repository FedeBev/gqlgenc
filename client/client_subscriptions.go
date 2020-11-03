package client

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"time"

	"golang.org/x/xerrors"

	"github.com/gorilla/websocket"
)

// WSClient is the websocket client instance
type WSClient struct {
	conn            *websocket.Conn
	options         *WSClientOptions
	dataChannels    map[string]chan json.RawMessage
	connectionAck   chan bool
	nextOperationID int
}

// WSClientOptions are all the options available for the ws creation
type WSClientOptions struct {
	url     *url.URL
	headers http.Header
	payload map[string]interface{}
	errChan chan error
}

// WSClientOptionsFunc are all the options available for the creation of the client
type WSClientOptionsFunc func(*WSClientOptions)

// WithCustomHeaders adds custom header ad the connection request
func WithCustomHeaders(customHeaders http.Header) WSClientOptionsFunc {
	return func(s *WSClientOptions) {
		for k, h := range customHeaders {
			s.headers[k] = h
		}
	}
}

// WithPayload adds a payload at the init message
func WithPayload(payload map[string]interface{}) WSClientOptionsFunc {
	return func(s *WSClientOptions) {
		s.payload = payload
	}
}

// WithErrorChannel sets an error channel where all the ws errors will be reported
func WithErrorChannel(errChan chan error) WSClientOptionsFunc {
	return func(s *WSClientOptions) {
		s.errChan = errChan
	}
}

type gqlMsgType string

// https://github.com/apollographql/subscriptions-transport-ws/blob/master/src/message-types.ts
const (
	gqlConnectionInit      gqlMsgType = "connection_init"      // Client -> Server
	gqlConnectionTerminate gqlMsgType = "connection_terminate" // Client -> Server
	gqlStart               gqlMsgType = "start"                // Client -> Server
	gqlStop                gqlMsgType = "stop"                 // Client -> Server
	gqlConnectionAck       gqlMsgType = "connection_ack"       // Server -> Client
	gqlConnectionError     gqlMsgType = "connection_error"     // Server -> Client
	gqlConnectionKeepAlive gqlMsgType = "ka"                   // Server -> Client
	gqlData                gqlMsgType = "data"                 // Server -> Client
	gqlError               gqlMsgType = "error"                // Server -> Client
	gqlComplete            gqlMsgType = "complete"             // Server -> Client
)

type messageFormat struct {
	MsgType gqlMsgType      `json:"type"`
	ID      string          `json:"id,omitempty"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

// NewWsClient creates a new websocket client instance
func NewWsClient(wsURL string, opts ...WSClientOptionsFunc) (*WSClient, error) {
	u, err := url.Parse(wsURL)
	if err != nil {
		return nil, err
	}

	if u.Scheme != "ws" && u.Scheme != "wss" {
		return nil, xerrors.Errorf("Unsupported '%s' URL schema, supported types are ws and wss")
	}

	options := &WSClientOptions{
		url: u,
		headers: http.Header{
			"Sec-WebSocket-Protocol": []string{"graphql-ws"},
		},
	}
	for _, opt := range opts {
		opt(options)
	}

	c := &WSClient{
		options:         options,
		dataChannels:    make(map[string]chan json.RawMessage),
		connectionAck:   make(chan bool),
		nextOperationID: 1,
	}

	err = c.connect()
	if err != nil {
		return nil, err
	}

	return c, nil
}

// Close closes a websocket connection
func (w *WSClient) Close() {
	w.conn.Close()
}

// connect opens the connection with the target server
func (w *WSClient) connect() error {
	c, _, err := websocket.DefaultDialer.Dial(w.options.url.String(), w.options.headers)
	if err != nil {
		return xerrors.Errorf("%w", err)
	}
	w.conn = c

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	done := make(chan struct{})

	// handle messages
	go func() {
		defer close(done)
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				if closeErr, ok := err.(*websocket.CloseError); ok {
					if closeErr.Code != websocket.CloseNormalClosure {
						w.pushError(err)
					} else {
						break
					}
				} else {
					w.pushError(err)
				}
			} else {
				w.processIncomingMsg(message)
			}
		}
	}()

	payloadStr, err := json.Marshal(w.options.payload)
	if err != nil {
		w.pushError(err)
		return err
	}

	initMsg := messageFormat{
		MsgType: gqlConnectionInit,
		Payload: payloadStr,
	}

	initMsgStr, err := json.Marshal(initMsg)
	if err != nil {
		w.pushError(err)
		return err
	}

	err = c.WriteMessage(websocket.TextMessage, initMsgStr)
	if err != nil {
		w.pushError(err)
		return err
	}

	// wait for connection ack
	select {
	case <-w.connectionAck:
	case <-time.After(5 * time.Second):
		err := xerrors.New("Connection timeout exceeded")
		w.pushError(err)
		return err
	}

	// handle interrupt
	go func() {
		<-interrupt
		// Cleanly close the connection by sending a close message and then
		// waiting (with timeout) for the server to close the connection.
		err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		if err != nil {
			w.pushError(xerrors.Errorf("Websocket close err: %w", err))
			return
		}
		select {
		case <-done:
		case <-time.After(time.Second):
		}
		log.Println("Closing things")
		close(w.connectionAck)
		if w.options.errChan != nil {
			close(w.options.errChan)
		}

		for _, dataCh := range w.dataChannels {
			close(dataCh)
		}
		return
	}()

	return nil
}

func (w *WSClient) pushError(err error) {
	if w.options.errChan != nil {
		w.options.errChan <- err
	}
}

// processIncomingMsg will unmarshal the response except the payload field, that field will be unmarshalled by gqlparser
// if the received message is of type data, the paylod will be sent to the right data channel, according with the id
func (w *WSClient) processIncomingMsg(msg []byte) {
	go func() {
		log.Printf("Received msg %s\n", string(msg))
		d := messageFormat{}
		err := json.Unmarshal(msg, &d)
		if err != nil {
			w.pushError(xerrors.Errorf("Unexpected message format: %w", err))
			return
		}

		switch d.MsgType {
		case gqlConnectionAck:
			// avoid goroutine stuck if ack is sent multiple unexpected times
			select {
			case w.connectionAck <- true:
			case <-time.After(time.Second):
			}
			break
		case gqlConnectionError:
			w.pushError(xerrors.Errorf("Connection error %s", d.Payload))
			break
		case gqlConnectionKeepAlive:
			// TODO
			break
		case gqlData:
			if d.ID == "" {
				w.pushError(xerrors.New("Unknown data message ID "))
			}
			if c, ok := w.dataChannels[d.ID]; ok {
				// sends the message to the right channel
				// if none is listening, send timeout error
				select {
				case c <- d.Payload:
				case <-time.After(time.Second):
					w.pushError(xerrors.Errorf("Timeout pushing data to channel for subscription %s", d.ID))
				}
			} else {
				w.pushError(xerrors.Errorf("Channel for subscription %s does not exists", d.ID))
			}
			break
		case gqlError:
			w.pushError(xerrors.Errorf("error %s", d.Payload))
			break
		case gqlComplete:
			// TODO: close data channel
			break
		default:
			w.pushError(xerrors.Errorf("Unexpected message type %s", d.MsgType))
		}
	}()
}

type operationPayload struct {
	Variables     map[string]interface{} `json:"variables"`
	Extensions    map[string]interface{} `json:"extensions,omitempty"`
	OperationName string                 `json:"operationName"`
	Query         string                 `json:"query"`
	Context       map[string]interface{} `json:"context,omitempty"`
}

// Subscribe creates a new graphql subscription
func (w *WSClient) Subscribe(ctx context.Context, operation string, variables map[string]interface{}, context map[string]interface{}) (chan json.RawMessage, error) {
	operationID := fmt.Sprintf("%d", w.nextOperationID)
	w.nextOperationID++

	operationPayload := operationPayload{
		Variables: variables,
		Query:     operation,
		Context:   context,
	}

	payloadStr, err := json.Marshal(operationPayload)
	if err != nil {
		w.pushError(xerrors.Errorf("%w", err))
		return nil, err
	}

	newMsg := messageFormat{
		ID:      operationID,
		MsgType: gqlStart,
		Payload: payloadStr,
	}

	newMsgStr, err := json.Marshal(newMsg)
	if err != nil {
		w.pushError(xerrors.Errorf("%w", err))
		return nil, err
	}

	dataListenerChan := make(chan json.RawMessage)
	w.dataChannels[operationID] = dataListenerChan

	err = w.conn.WriteMessage(websocket.TextMessage, newMsgStr)
	if err != nil {
		w.pushError(xerrors.Errorf("%w", err))
		close(dataListenerChan)
		return nil, err
	}

	return dataListenerChan, nil
}

// ParseSubscriptionData unmarshal the data into the target interface
func (*WSClient) ParseSubscriptionData(d json.RawMessage, r interface{}) error {
	return parseSubscriptionResponse(d, r)
}

func parseSubscriptionResponse(body json.RawMessage, result interface{}) error {
	errResponse := &ErrorResponse{}

	if err := unmarshal(body, result); err != nil {
		if gqlErr, ok := err.(*GqlErrorList); ok {
			errResponse.GqlErrors = &gqlErr.Errors
		} else {
			return err
		}
	}

	if errResponse.HasErrors() {
		return errResponse
	}

	return nil
}
