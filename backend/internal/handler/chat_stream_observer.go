package handler

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

type deliveredChatResponseWriter struct {
	gin.ResponseWriter
	observer *deliveredChatStreamObserver
}

func (w *deliveredChatResponseWriter) Write(payload []byte) (int, error) {
	n, err := w.ResponseWriter.Write(payload)
	if n > 0 && w.observer != nil {
		if observeErr := w.observer.Observe(payload[:n]); err == nil {
			err = observeErr
		}
	}
	return n, err
}

func (w *deliveredChatResponseWriter) WriteString(payload string) (int, error) {
	n, err := w.ResponseWriter.WriteString(payload)
	if n > 0 && w.observer != nil {
		if observeErr := w.observer.Observe([]byte(payload[:n])); err == nil {
			err = observeErr
		}
	}
	return n, err
}

type deliveredChatStreamSnapshot struct {
	Content       string
	FinishReason  string
	ErrorCode     string
	ErrorMessage  string
	Done          bool
	CheckpointSeq int64
}

type deliveredChatStreamObserver struct {
	mu            sync.Mutex
	pending       string
	eventData     []string
	content       strings.Builder
	finishReason  string
	errorCode     string
	errorMessage  string
	done          bool
	checkpointSeq int64
	checkpoint    func(sequence int64, content string) error
	checkpointErr error
	frozen        bool
	cancelDone    <-chan struct{}
}

func newDeliveredChatStreamObserver(
	checkpoint ...func(sequence int64, content string) error,
) *deliveredChatStreamObserver {
	observer := &deliveredChatStreamObserver{}
	if len(checkpoint) > 0 {
		observer.checkpoint = checkpoint[0]
	}
	return observer
}

func (o *deliveredChatStreamObserver) Observe(payload []byte) error {
	if o == nil || len(payload) == 0 {
		return nil
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.frozen || cancellationSignaled(o.cancelDone) {
		o.frozen = true
		return o.checkpointErr
	}

	o.pending += string(payload)
	for {
		newline := strings.IndexByte(o.pending, '\n')
		if newline < 0 {
			return o.checkpointErr
		}
		line := strings.TrimSuffix(o.pending[:newline], "\r")
		o.pending = o.pending[newline+1:]
		o.processLine(line)
		if o.checkpointErr != nil {
			o.frozen = true
			return o.checkpointErr
		}
	}
}

func (o *deliveredChatStreamObserver) bindCancellation(done <-chan struct{}) {
	if o == nil {
		return
	}
	o.mu.Lock()
	o.cancelDone = done
	if cancellationSignaled(done) {
		o.frozen = true
	}
	o.mu.Unlock()
}

func (o *deliveredChatStreamObserver) Snapshot() deliveredChatStreamSnapshot {
	if o == nil {
		return deliveredChatStreamSnapshot{}
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	return deliveredChatStreamSnapshot{
		Content:       o.content.String(),
		FinishReason:  o.finishReason,
		ErrorCode:     o.errorCode,
		ErrorMessage:  o.errorMessage,
		Done:          o.done && o.errorCode == "",
		CheckpointSeq: o.checkpointSeq,
	}
}

func cancellationSignaled(done <-chan struct{}) bool {
	if done == nil {
		return false
	}
	select {
	case <-done:
		return true
	default:
		return false
	}
}

func (o *deliveredChatStreamObserver) Freeze() {
	if o == nil {
		return
	}
	o.mu.Lock()
	o.frozen = true
	o.mu.Unlock()
}

func (o *deliveredChatStreamObserver) ForceCheckpoint() (int64, error) {
	if o == nil {
		return 0, nil
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	o.checkpointSeq++
	if o.checkpoint == nil {
		return o.checkpointSeq, nil
	}
	err := o.checkpoint(o.checkpointSeq, o.content.String())
	if err != nil {
		o.checkpointErr = err
	}
	return o.checkpointSeq, err
}

func (o *deliveredChatStreamObserver) processLine(line string) {
	if line == "" {
		o.processEvent()
		return
	}
	if strings.HasPrefix(line, ":") {
		return
	}
	field, value, found := strings.Cut(line, ":")
	if !found || field != "data" {
		return
	}
	o.eventData = append(o.eventData, strings.TrimPrefix(value, " "))
}

func (o *deliveredChatStreamObserver) processEvent() {
	if len(o.eventData) == 0 {
		return
	}
	payload := strings.Join(o.eventData, "\n")
	o.eventData = o.eventData[:0]
	if strings.TrimSpace(payload) == "[DONE]" {
		o.done = true
		return
	}

	var event struct {
		Choices []struct {
			Delta struct {
				Content *string `json:"content"`
			} `json:"delta"`
			FinishReason *string `json:"finish_reason"`
		} `json:"choices"`
		Error json.RawMessage `json:"error"`
	}
	if err := json.Unmarshal([]byte(payload), &event); err != nil {
		return
	}

	contentChanged := false
	for i := range event.Choices {
		if event.Choices[i].Delta.Content != nil {
			o.content.WriteString(*event.Choices[i].Delta.Content)
			contentChanged = true
		}
		if event.Choices[i].FinishReason != nil &&
			strings.TrimSpace(*event.Choices[i].FinishReason) != "" {
			o.finishReason = strings.TrimSpace(*event.Choices[i].FinishReason)
		}
	}
	if contentChanged {
		o.writeCheckpoint()
	}
	if len(event.Error) == 0 || string(event.Error) == "null" {
		return
	}

	var detail struct {
		Code    any    `json:"code"`
		Type    string `json:"type"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(event.Error, &detail); err != nil {
		o.errorCode = "STREAM_ERROR"
		o.errorMessage = "The chat stream reported an error"
		return
	}
	o.errorCode = strings.TrimSpace(detail.Type)
	if detail.Code != nil {
		o.errorCode = strings.TrimSpace(fmt.Sprint(detail.Code))
	}
	if o.errorCode == "" {
		o.errorCode = "STREAM_ERROR"
	}
	o.errorMessage = strings.TrimSpace(detail.Message)
	if o.errorMessage == "" {
		o.errorMessage = "The chat stream reported an error"
	}
}

func (o *deliveredChatStreamObserver) writeCheckpoint() {
	o.checkpointSeq++
	if o.checkpoint == nil {
		return
	}
	if err := o.checkpoint(o.checkpointSeq, o.content.String()); err != nil {
		o.checkpointErr = err
	}
}
