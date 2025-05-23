package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/ZaparooProject/zaparoo-core/pkg/api/models"
	"github.com/ZaparooProject/zaparoo-core/pkg/config"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

var (
	ErrRequestTimeout = errors.New("request timed out")
	ErrInvalidParams  = errors.New("invalid params")
)

const ApiPath = "/api/v0.1"

// Disable runZapScript and returns a function that enables it back
func ZapScriptWrapper(cfg *config.Instance) func() {
	_, err := LocalClient(
		cfg,
		models.MethodSettingsUpdate,
		"{\"runZapScript\":false}",
	)
	if err != nil {
		log.Error().Err(err).Msg("error disabling run")
		_, _ = fmt.Fprintf(os.Stderr, "Error disabling run: %v\n", err)
		os.Exit(1)
	}

	return func() {
		// TODO: this should be in a defer or signal handler to or else it won't
		// run if there was a crash or unhandled error
		_, err = LocalClient(
			cfg,
			models.MethodSettingsUpdate,
			"{\"runZapScript\":true}",
		)
		if err != nil {
			log.Error().Err(err).Msg("error enabling run")
			_, _ = fmt.Fprintf(os.Stderr, "Error enabling run: %v\n", err)
			os.Exit(1)
		}
	}

}

// LocalClient sends a single unauthenticated method with params to the local
// running API service, waits for a response until timeout then disconnects.
func LocalClient(
	cfg *config.Instance,
	method string,
	params string,
) (string, error) {
	localWebsocketUrl := url.URL{
		Scheme: "ws",
		Host:   "localhost:" + strconv.Itoa(cfg.ApiPort()),
		Path:   ApiPath,
	}

	id, err := uuid.NewUUID()
	if err != nil {
		return "", err
	}

	req := models.RequestObject{
		JSONRPC: "2.0",
		ID:      &id,
		Method:  method,
	}

	if len(params) == 0 {
		req.Params = nil
	} else if json.Valid([]byte(params)) {
		req.Params = []byte(params)
	} else {
		return "", ErrInvalidParams
	}

	c, _, err := websocket.DefaultDialer.Dial(localWebsocketUrl.String(), nil)
	if err != nil {
		return "", err
	}
	defer func(c *websocket.Conn) {
		err := c.Close()
		if err != nil {
			log.Warn().Err(err).Msg("error closing websocket")
		}
	}(c)

	done := make(chan struct{})
	var resp *models.ResponseObject

	go func() {
		defer close(done)
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Error().Err(err).Msg("error reading message")
				return
			}

			var m models.ResponseObject
			err = json.Unmarshal(message, &m)
			if err != nil {
				continue
			}

			if m.JSONRPC != "2.0" {
				log.Error().Msg("invalid jsonrpc version")
				continue
			}

			if m.ID != id {
				continue
			}

			resp = &m
			return
		}
	}()

	err = c.WriteJSON(req)
	if err != nil {
		return "", err
	}

	timer := time.NewTimer(config.ApiRequestTimeout)
	select {
	case <-done:
		break
	case <-timer.C:
		return "", ErrRequestTimeout
	}

	if resp == nil {
		return "", ErrRequestTimeout
	}

	if resp.Error != nil {
		return "", errors.New(resp.Error.Message)
	}

	var b []byte
	b, err = json.Marshal(resp.Result)
	if err != nil {
		return "", err
	}

	return string(b), nil
}

func WaitNotification(
	cfg *config.Instance,
	id string,
) (string, error) {
	u := url.URL{
		Scheme: "ws",
		Host:   "localhost:" + strconv.Itoa(cfg.ApiPort()),
		Path:   ApiPath,
	}

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return "", err
	}
	defer func(c *websocket.Conn) {
		err := c.Close()
		if err != nil {
			log.Warn().Err(err).Msg("error closing websocket")
		}
	}(c)

	done := make(chan struct{})
	var resp *models.RequestObject

	go func() {
		defer close(done)
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Error().Err(err).Msg("error reading message")
				return
			}

			var m models.RequestObject
			err = json.Unmarshal(message, &m)
			if err != nil {
				continue
			}

			if m.JSONRPC != "2.0" {
				log.Error().Msg("invalid jsonrpc version")
				continue
			}

			if m.ID != nil {
				continue
			}

			if m.Method != id {
				continue
			}

			resp = &m

			return
		}
	}()

	timer := time.NewTimer(config.ApiRequestTimeout)
	select {
	case <-done:
		break
	case <-timer.C:
		return "", ErrRequestTimeout
	}

	if resp == nil {
		return "", ErrRequestTimeout
	}

	var b []byte
	b, err = json.Marshal(resp.Params)
	if err != nil {
		return "", err
	}

	return string(b), nil
}
