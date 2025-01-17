package handlers

import (
	"bufio"
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gookit/slog"
	"github.com/pkg/errors"
	"github.com/valyala/fasthttp"
	help "gitlab.dev.ict/golang/go-ai/helpers"
	"gitlab.dev.ict/golang/go-ai/models/sse"
	"gitlab.dev.ict/golang/libs/utils"
)

const (
	FmtEvtFull        = "event: %s\nid: %s\ndata: %s\nretry: %d\n\n"
	FmtEvtFullNoRetry = "event: %s\nid: %s\ndata: %s\n\n"
	FmtEvt            = "event: %s\ndata: %s\n\n"
	FmtData           = "%sdata: %s\n\n"
	CookUID           = "userId"
	CookUName         = "username"
	ANON              = "anonymous"
)

type EventType = sse.EventType

func (a *AppHandler) SSE(c *fiber.Ctx) error {
	connectionId := c.Query("connectionId", "")

	c.Set(fiber.HeaderContentType, "text/event-stream")
	c.Set(fiber.HeaderCacheControl, "no-cache")
	c.Set(fiber.HeaderConnection, "keep-alive")
	c.Set(fiber.HeaderTransferEncoding, "chunked")

	log := help.Log(c)
	log.WithData(slog.M{"connectionId": connectionId}).Info("SSE: Start init SSE handler...")

	user, err := addUser(c)
	if err != nil {
		log.Errorf("Error while adding user: %v", err)
		return err
	}

	startTime := time.Now()
	ctx := c.Context()
	c.Context().SetBodyStreamWriter(fasthttp.StreamWriter(func(w *bufio.Writer) {
		log := a.log.RecWithCtx(utils.GenerateCtxWithRid(), "sse").AddData(slog.M{"user": user.Login, "uuid": user.UUID, "connId": connectionId})
		log.Infof("Create SSE stream for user=[%s:%s]", user.Login, user.UUID)

		user.StoreConnection(log, connectionId, w)
		defer func() {
			user.CleanupConnection(log, connectionId)
			log.Infof("SSE connection closed for userUUID: %s, connectionId: %s, duration: %v", user.UUID, connectionId, time.Since(startTime))
		}()

		if err := streamSSE(log, user, connectionId, a.hbInterval, ctx); err != nil {
			log.Errorf("Error in SSE stream: %v", err)
		} else {
			log.Infof("SSE stream closed gracefully for user %s, connectionId %s", user.UUID, connectionId)
		}

	}))

	log.Debugf("SSE: inited!")
	return nil
}

func streamSSE(logger *slog.Record, user *User, connectionId string, hbInterval int, ctx context.Context) error {
	ticker := time.NewTicker(time.Duration(hbInterval) * time.Second)
	defer ticker.Stop()

	// Retrieve the writer from ActiveConns
	user.mu.Lock()
	writer, exists := user.ActiveConns[connectionId]
	user.mu.Unlock()

	if !exists {
		return fmt.Errorf("no active connection found for connectionId=%s", connectionId)
	}

	const maxRetries = 3                       // Maximum retry attempts for flushing
	const backoffBase = 100 * time.Millisecond // Base backoff duration for exponential backoff

	for cycle := 1; ; cycle++ {
		select {
		case msg := <-user.ChanEventMsg:
			if err := handleEvent(logger, user, msg, writer); err != nil {
				logger.Errorf("Error in handleEvent: %v", err)
				return err // Propagate error to close the connection
			}
		case msg := <-user.ChanMsgBB:
			if err := createAndLogEventWithRetry(logger, writer, sse.EvtChatGptResp, FmtEvt, string(msg), user); err != nil {
				logger.Errorf("Error in createAndLogEvent: %v", err)
				return err
			}
		case msg := <-user.ChanMessages:
			if err := createAndLogEventWithRetry(logger, writer, sse.EvtChatGptResp, FmtEvt, msg, user); err != nil {
				logger.Errorf("Error in createAndLogEvent: %v", err)
				return err
			}

		case msg := <-user.ChanWithSSEMsg:
			logger.Infof("Sending pre-formatted SSE message to user=[%s]. Msg=[%s]", user.Login, strconv.Quote(msg))
			if _, err := writer.WriteString(msg); err != nil {
				logger.Errorf("Error writing pre-formatted message: %v", err)
				return err
			}
		case <-ticker.C:
			err := createAndLogEventWithRetry(logger, writer, sse.EvtNul, FmtData, fmt.Sprintf("Cycle: %d, Time: %s", cycle, time.Now().Format(time.RFC3339)), user)
			if err != nil {
				logger.Errorf("Error during heartbeat event: %v", err)
				return err
			}
		case <-ctx.Done():
			logger.Warnf("User %s disconnected, connectionId=%s", user.UUID, connectionId)
			return nil
		}

		// Retry mechanism for flushing
		for attempt := 1; attempt <= maxRetries; attempt++ {
			if err := writer.Flush(); err != nil {
				logger.Warnf("Error flushing buffer (attempt %d/%d): %v", attempt, maxRetries, err)
				if attempt == maxRetries {
					return errors.Wrap(err, "Max retries reached while flushing buffer, closing connection")
				}
				time.Sleep(backoffBase * time.Duration(1<<attempt)) // Exponential backoff
				continue
			}
			// Successful flush, exit retry loop
			break
		}
	}
}

// Handle structured SSE events
func handleEvent(logger *slog.Record, user *User, event sse.Event, writer *bufio.Writer) error {
	logger.Infof("Sending structured SSE event to user: %s", event)
	msg, _ := event.MakeMsgSSE(logger)

	return withRetry(logger, 3, 100*time.Millisecond, func() error {
		_, err := writer.Write([]byte(msg))
		return err
	})
}

// Handle formatted string-based SSE events
func createAndLogEvent(logger *slog.Record, writer *bufio.Writer, eventType EventType, format, message string, user *User, additionalInfo ...interface{}) error {
	eventMessage := fmt.Sprintf(format, sse.DictEvents[eventType], strings.Trim(strconv.Quote(message), `"`))
	if eventType != sse.EvtNul {
		logger.Infof("Send event: [%s] to user=[%s:%s] %v", strconv.Quote(eventMessage), user.Login, user.UUID, additionalInfo)
	}

	return withRetry(logger, 3, 100*time.Millisecond, func() error {
		_, err := writer.WriteString(eventMessage)
		return err
	})
}

// Retry logic with exponential backoff
func withRetry(logger *slog.Record, maxRetries int, backoffBase time.Duration, operation func() error) error {
	for attempt := 1; attempt <= maxRetries; attempt++ {
		if err := operation(); err != nil {
			logger.Warnf("Retry attempt %d/%d failed: %v", attempt, maxRetries, err)
			if attempt == maxRetries {
				return errors.Wrap(err, "Max retries reached, operation failed")
			}
			time.Sleep(backoffBase * time.Duration(1<<attempt)) // Exponential backoff
			continue
		}

		// Successful operation, exit retry loop
		return nil
	}
	return nil
}

// createAndLogEventWithRetry creates an event message and retries if writing fails
func createAndLogEventWithRetry(logger *slog.Record, writer *bufio.Writer, eventType EventType, format, message string, user *User, additionalInfo ...interface{}) error {
	const maxRetries = 3
	const backoffBase = 100 * time.Millisecond

	eventMessage := fmt.Sprintf(format, sse.DictEvents[eventType], strings.Trim(strconv.Quote(message), `"`))
	if eventType != sse.EvtNul {
		logger.Infof("Send event: [%s] to user=[%s:%s] %v", strconv.Quote(eventMessage), user.Login, user.UUID, additionalInfo)
	}

	for attempt := 1; attempt <= maxRetries; attempt++ {
		if _, err := writer.WriteString(eventMessage); err != nil {
			logger.Warnf("Error writing event (attempt %d/%d): %v", attempt, maxRetries, err)
			if attempt == maxRetries {
				return errors.Wrap(err, "Max retries reached while writing event")
			}
			time.Sleep(backoffBase * time.Duration(1<<attempt))
			continue
		}

		// Successful write, exit retry loop
		break
	}

	return nil
}

func (a *AppHandler) LogActiveSSEConnections() {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("MONITOR SSE CONNECTIONS! Current active users: %d\n", usersCount()))
	appStoreForUsers.Range(func(key, value interface{}) bool {
		user := value.(*User)
		user.mu.Lock()
		defer user.mu.Unlock()

		sb.WriteString(fmt.Sprintf("- %s: UUID=%s connTime=%s\n", user.Login, user.UUID, user.ConnTime.Format("2006-01-02T15:04:05.999")))

		for connID := range user.ActiveConns {
			sb.WriteString(fmt.Sprintf("\tConnection: connectionId=%s\n", connID))
		}
		return true
	})
	a.log.Warn(sb.String())
}
