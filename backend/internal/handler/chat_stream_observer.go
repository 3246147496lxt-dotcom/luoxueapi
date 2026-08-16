package handler

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

const (
	deliveredChatActivityCheckpointInterval = 500 * time.Millisecond
	deliveredChatActivityCheckpointBatch    = 64
	deliveredChatActivityReorderWindow      = 32
	deliveredChatActivityReorderMaxWait     = 50 * time.Millisecond
	deliveredChatUnsequencedDedupWindow     = 128
	deliveredChatPendingSummaryEventLimit   = 256
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
	Content                 string
	FinishReason            string
	ErrorCode               string
	ErrorMessage            string
	Done                    bool
	CheckpointSeq           int64
	ResponseID              string
	ReasoningMode           string
	ReasoningEffort         string
	Activities              []service.ChatMessageActivity
	ResponsesTerminalSeen   bool
	ResponsesTerminalStatus string
}

// deliveredChatStreamCheckpoint is the batched persistence payload. The
// legacy content-only callback remains supported; new callers should bind this
// snapshot callback instead so content and Activity share one monotonic write.
type deliveredChatStreamCheckpoint struct {
	Sequence        int64
	Content         string
	ResponseID      string
	ReasoningMode   string
	ReasoningEffort string
	Activities      []service.ChatMessageActivity
}

type deliveredChatActivityKey struct {
	ItemID       string
	OutputIndex  int
	SummaryIndex int
}

type deliveredChatObservedActivity struct {
	activity service.ChatMessageActivity
	text     strings.Builder
	textDone bool
}

func (a *deliveredChatObservedActivity) appendText(delta string) {
	if a == nil || delta == "" {
		return
	}
	_, _ = a.text.WriteString(delta)
}

func (a *deliveredChatObservedActivity) replaceText(text string) {
	if a == nil {
		return
	}
	a.text.Reset()
	_, _ = a.text.WriteString(text)
}

func (a *deliveredChatObservedActivity) currentText() string {
	if a == nil {
		return ""
	}
	return a.text.String()
}

type deliveredChatResponsesEvent struct {
	eventType string
	payload   []byte
	sequence  *int64
}

type deliveredChatOutputItemIdentity struct {
	itemID    string
	reasoning bool
}

type deliveredChatPendingSummaryEvent struct {
	eventType string
	payload   []byte
	sequence  int64
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

	responseID              string
	reasoningMode           string
	reasoningEffort         string
	modeAuthoritative       bool
	effortAuthoritative     bool
	activities              map[deliveredChatActivityKey]*deliveredChatObservedActivity
	nextSortOrder           int64
	outputItemIdentities    map[int]deliveredChatOutputItemIdentity
	pendingSummaryEvents    map[int][]deliveredChatPendingSummaryEvent
	pendingSummaryCount     int
	responsesTerminalSeen   bool
	responsesTerminalStatus string

	responsesSequenceInitialized bool
	nextResponsesSequence        int64
	lastResponsesSequence        int64
	pendingResponsesEvents       map[int64]deliveredChatResponsesEvent
	pendingResponsesSince        time.Time
	pendingResponsesTimer        *time.Timer
	pendingResponsesTimerID      uint64
	unsequencedSeen              map[[sha256.Size]byte]struct{}
	unsequencedOrder             [][sha256.Size]byte

	snapshotCheckpoint       func(deliveredChatStreamCheckpoint) error
	snapshotCheckpointDirty  bool
	snapshotCheckpointEvents int
	lastSnapshotCheckpointAt time.Time
	lastSnapshotActivitySize map[deliveredChatActivityKey]int
	lastSnapshotContentSize  int

	snapshotDispatchMu               sync.Mutex
	snapshotDispatchCond             *sync.Cond
	snapshotDispatchPending          *deliveredChatStreamCheckpoint
	snapshotDispatchRunning          bool
	snapshotDispatchLatestSequence   int64
	snapshotDispatchFinishedSequence int64
	snapshotDispatchErr              error
}

func newDeliveredChatStreamObserver(
	checkpoint ...func(sequence int64, content string) error,
) *deliveredChatStreamObserver {
	observer := &deliveredChatStreamObserver{
		activities:               make(map[deliveredChatActivityKey]*deliveredChatObservedActivity),
		outputItemIdentities:     make(map[int]deliveredChatOutputItemIdentity),
		pendingSummaryEvents:     make(map[int][]deliveredChatPendingSummaryEvent),
		lastResponsesSequence:    -1,
		pendingResponsesEvents:   make(map[int64]deliveredChatResponsesEvent),
		unsequencedSeen:          make(map[[sha256.Size]byte]struct{}),
		lastSnapshotActivitySize: make(map[deliveredChatActivityKey]int),
	}
	observer.snapshotDispatchCond = sync.NewCond(&observer.snapshotDispatchMu)
	if len(checkpoint) > 0 {
		observer.checkpoint = checkpoint[0]
	}
	return observer
}

// bindSnapshotCheckpoint opts the observer into bounded Activity persistence.
// It is intentionally separate from the legacy constructor callback so the
// existing Chat handler remains source-compatible until its integration point
// switches to the richer payload.
func (o *deliveredChatStreamObserver) bindSnapshotCheckpoint(
	checkpoint func(deliveredChatStreamCheckpoint) error,
) {
	if o == nil {
		return
	}
	o.mu.Lock()
	o.snapshotCheckpoint = checkpoint
	o.mu.Unlock()
}

// setReasoningMetadata seeds authoritative request-side reasoning controls.
// Upstream response metadata only fills values that the request omitted.
func (o *deliveredChatStreamObserver) setReasoningMetadata(mode, effort string) {
	if o == nil {
		return
	}
	mode = strings.TrimSpace(mode)
	effort = strings.TrimSpace(effort)
	if len(mode) > 32 {
		mode = ""
	}
	if len(effort) > 32 {
		effort = ""
	}
	if mode == service.WebChatReasoningModePro {
		effort = ""
	}
	o.mu.Lock()
	o.reasoningMode = mode
	o.reasoningEffort = effort
	o.modeAuthoritative = mode != ""
	o.effortAuthoritative = mode == service.WebChatReasoningModePro || effort != ""
	for _, observed := range o.activities {
		observed.activity.ReasoningMode = mode
		observed.activity.ReasoningEffort = effort
	}
	if (mode != "" || effort != "") && o.hasPersistableActivityLocked() {
		o.markSnapshotCheckpointDirtyLocked()
	}
	o.mu.Unlock()
}

func (o *deliveredChatStreamObserver) Observe(payload []byte) error {
	if o == nil || len(payload) == 0 {
		return nil
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.frozen || cancellationSignaled(o.cancelDone) {
		o.flushPendingResponsesEventsLocked()
		o.interruptActivitiesLocked()
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
		o.flushPendingResponsesEventsLocked()
		o.interruptActivitiesLocked()
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
		Content:      o.content.String(),
		FinishReason: o.finishReason,
		ErrorCode:    o.errorCode,
		ErrorMessage: o.errorMessage,
		Done: (o.responsesTerminalSeen &&
			o.responsesTerminalStatus == service.ChatMessageActivityStatusCompleted ||
			(!o.responsesTerminalSeen && o.done)) && o.errorCode == "",
		CheckpointSeq:           o.checkpointSeq,
		ResponseID:              o.responseID,
		ReasoningMode:           o.reasoningMode,
		ReasoningEffort:         o.reasoningEffort,
		Activities:              o.snapshotActivitiesLocked(),
		ResponsesTerminalSeen:   o.responsesTerminalSeen,
		ResponsesTerminalStatus: o.responsesTerminalStatus,
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
	o.flushPendingResponsesEventsLocked()
	o.interruptActivitiesLocked()
	o.frozen = true
	o.mu.Unlock()
}

func (o *deliveredChatStreamObserver) ForceCheckpoint() (int64, error) {
	if o == nil {
		return 0, nil
	}
	o.mu.Lock()
	o.flushPendingResponsesEventsLocked()
	o.checkpointSeq++
	sequence := o.checkpointSeq
	if o.checkpoint != nil {
		if err := o.checkpoint(sequence, o.content.String()); err != nil {
			o.checkpointErr = err
			o.mu.Unlock()
			return sequence, err
		}
	}
	if o.snapshotCheckpoint != nil && o.snapshotCheckpointDirty {
		o.enqueueSnapshotCheckpointLocked(o.checkpointPayloadLocked(sequence))
		o.recordSnapshotSizesLocked()
		o.snapshotCheckpointDirty = false
		o.snapshotCheckpointEvents = 0
		o.lastSnapshotCheckpointAt = time.Now()
	}
	checkpointErr := o.checkpointErr
	o.mu.Unlock()

	if checkpointErr != nil {
		return sequence, checkpointErr
	}
	return sequence, o.waitForSnapshotCheckpoints(o.snapshotCheckpointTarget())
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

	var envelope struct {
		Source    string          `json:"source"`
		EventType string          `json:"eventType"`
		Payload   json.RawMessage `json:"payload"`
	}
	if err := json.Unmarshal([]byte(payload), &envelope); err == nil &&
		strings.TrimSpace(envelope.Source) == service.ChatMessageActivitySourceOpenAIResponses {
		o.processResponsesEnvelope(strings.TrimSpace(envelope.EventType), envelope.Payload)
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
			_, _ = o.content.WriteString(*event.Choices[i].Delta.Content)
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

func (o *deliveredChatStreamObserver) processResponsesEnvelope(eventType string, payload json.RawMessage) {
	if !supportedDeliveredChatResponsesActivityEvent(eventType) ||
		len(payload) == 0 ||
		!json.Valid(payload) ||
		!strings.HasPrefix(strings.TrimSpace(string(payload)), "{") {
		return
	}
	event := deliveredChatResponsesEvent{
		eventType: eventType,
		payload:   append([]byte(nil), payload...),
	}
	if sequence := gjson.GetBytes(payload, "sequence_number"); sequence.Exists() && sequence.Type == gjson.Number {
		value := sequence.Int()
		if value < 0 {
			return
		}
		event.sequence = &value
	}
	forceCheckpoint := o.enqueueResponsesEventLocked(event)
	if o.snapshotCheckpointDirty {
		o.maybeWriteSnapshotCheckpointLocked(forceCheckpoint)
	}
}

func supportedDeliveredChatResponsesActivityEvent(eventType string) bool {
	switch eventType {
	case "response.created",
		"response.reasoning_summary_part.added",
		"response.reasoning_summary_text.delta",
		"response.reasoning_summary_text.done",
		"response.reasoning_summary_part.done",
		"response.output_item.added",
		"response.output_item.done",
		"response.completed",
		"response.incomplete",
		"response.failed":
		return true
	default:
		return false
	}
}

func (o *deliveredChatStreamObserver) enqueueResponsesEventLocked(event deliveredChatResponsesEvent) bool {
	if event.sequence == nil {
		fingerprint := sha256.Sum256(append(append([]byte(event.eventType), 0), event.payload...))
		if _, duplicate := o.unsequencedSeen[fingerprint]; duplicate {
			return false
		}
		o.unsequencedSeen[fingerprint] = struct{}{}
		o.unsequencedOrder = append(o.unsequencedOrder, fingerprint)
		if len(o.unsequencedOrder) > deliveredChatUnsequencedDedupWindow {
			oldest := o.unsequencedOrder[0]
			delete(o.unsequencedSeen, oldest)
			o.unsequencedOrder = o.unsequencedOrder[1:]
		}
		// Arrival order is the only ordering signal for an unsequenced lifecycle
		// boundary. Apply any already-delivered sequenced events first so a gap
		// timer cannot append an older delta after an authoritative done/terminal.
		forceCheckpoint := false
		if isDeliveredChatResponsesBoundaryEvent(event.eventType) {
			forceCheckpoint = o.flushPendingResponsesEventsLocked()
		}
		sequence := o.lastResponsesSequence + 1
		if sequence < 0 {
			sequence = 0
		}
		return o.applyResponsesEventLocked(event, sequence) || forceCheckpoint
	}

	sequence := *event.sequence
	if o.responsesSequenceInitialized && sequence < o.nextResponsesSequence {
		return false
	}
	if _, duplicate := o.pendingResponsesEvents[sequence]; duplicate {
		return false
	}
	if len(o.pendingResponsesEvents) == 0 {
		o.pendingResponsesSince = time.Now()
	}
	o.pendingResponsesEvents[sequence] = event
	if !o.responsesSequenceInitialized {
		minimum, contiguous := o.pendingResponsesSequenceStartLocked()
		if event.eventType == "response.created" || sequence <= 1 || contiguous ||
			len(o.pendingResponsesEvents) >= deliveredChatActivityReorderWindow ||
			isDeliveredChatResponsesTerminalEvent(event.eventType) {
			o.responsesSequenceInitialized = true
			o.nextResponsesSequence = minimum
		}
	}
	forceCheckpoint := false
	if o.responsesSequenceInitialized {
		forceCheckpoint = o.drainContiguousResponsesEventsLocked()
	}
	if len(o.pendingResponsesEvents) >= deliveredChatActivityReorderWindow ||
		isDeliveredChatResponsesTerminalEvent(event.eventType) ||
		(!o.pendingResponsesSince.IsZero() &&
			time.Since(o.pendingResponsesSince) >= deliveredChatActivityReorderMaxWait) {
		forceCheckpoint = o.flushPendingResponsesEventsLocked() || forceCheckpoint
	} else if len(o.pendingResponsesEvents) > 0 {
		o.schedulePendingResponsesFlushLocked()
	}
	return forceCheckpoint
}

// schedulePendingResponsesFlushLocked enforces the reorder window even when
// no further private Activity envelope arrives. Responses sequence numbers are
// global, while the Web Chat stream intentionally filters unrelated events
// such as output_text deltas. A missing private sequence therefore cannot rely
// on a later Activity event to wake the observer.
func (o *deliveredChatStreamObserver) schedulePendingResponsesFlushLocked() {
	if o.frozen || len(o.pendingResponsesEvents) == 0 || o.pendingResponsesSince.IsZero() {
		o.cancelPendingResponsesTimerLocked()
		return
	}
	if o.pendingResponsesTimer != nil {
		return
	}
	delay := time.Until(o.pendingResponsesSince.Add(deliveredChatActivityReorderMaxWait))
	if delay < 0 {
		delay = 0
	}
	o.pendingResponsesTimerID++
	timerID := o.pendingResponsesTimerID
	o.pendingResponsesTimer = time.AfterFunc(delay, func() {
		o.mu.Lock()
		defer o.mu.Unlock()
		if timerID != o.pendingResponsesTimerID {
			return
		}
		o.pendingResponsesTimer = nil
		if o.frozen || len(o.pendingResponsesEvents) == 0 {
			o.pendingResponsesSince = time.Time{}
			return
		}
		if remaining := time.Until(
			o.pendingResponsesSince.Add(deliveredChatActivityReorderMaxWait),
		); remaining > 0 {
			o.pendingResponsesTimerID++
			nextTimerID := o.pendingResponsesTimerID
			o.pendingResponsesTimer = time.AfterFunc(remaining, func() {
				o.flushPendingResponsesTimer(nextTimerID)
			})
			return
		}
		forceCheckpoint := o.flushPendingResponsesEventsLocked()
		if o.snapshotCheckpointDirty {
			o.maybeWriteSnapshotCheckpointLocked(forceCheckpoint)
		}
	})
}

func (o *deliveredChatStreamObserver) flushPendingResponsesTimer(timerID uint64) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if timerID != o.pendingResponsesTimerID {
		return
	}
	o.pendingResponsesTimer = nil
	if o.frozen || len(o.pendingResponsesEvents) == 0 {
		o.pendingResponsesSince = time.Time{}
		return
	}
	forceCheckpoint := o.flushPendingResponsesEventsLocked()
	if o.snapshotCheckpointDirty {
		o.maybeWriteSnapshotCheckpointLocked(forceCheckpoint)
	}
}

func (o *deliveredChatStreamObserver) cancelPendingResponsesTimerLocked() {
	o.pendingResponsesTimerID++
	if o.pendingResponsesTimer == nil {
		return
	}
	o.pendingResponsesTimer.Stop()
	o.pendingResponsesTimer = nil
}

func (o *deliveredChatStreamObserver) pendingResponsesSequenceStartLocked() (int64, bool) {
	minimum := int64(0)
	first := true
	for sequence := range o.pendingResponsesEvents {
		if first || sequence < minimum {
			minimum = sequence
			first = false
		}
	}
	_, hasNext := o.pendingResponsesEvents[minimum+1]
	return minimum, len(o.pendingResponsesEvents) > 1 && hasNext
}

func (o *deliveredChatStreamObserver) drainContiguousResponsesEventsLocked() bool {
	forceCheckpoint := false
	for {
		event, exists := o.pendingResponsesEvents[o.nextResponsesSequence]
		if !exists {
			if len(o.pendingResponsesEvents) == 0 {
				o.pendingResponsesSince = time.Time{}
				o.cancelPendingResponsesTimerLocked()
			}
			return forceCheckpoint
		}
		sequence := o.nextResponsesSequence
		delete(o.pendingResponsesEvents, sequence)
		forceCheckpoint = o.applyResponsesEventLocked(event, sequence) || forceCheckpoint
		o.nextResponsesSequence++
	}
}

func (o *deliveredChatStreamObserver) flushPendingResponsesEventsLocked() bool {
	o.cancelPendingResponsesTimerLocked()
	if len(o.pendingResponsesEvents) == 0 {
		o.pendingResponsesSince = time.Time{}
		return false
	}
	forceCheckpoint := false
	sequences := make([]int64, 0, len(o.pendingResponsesEvents))
	for sequence := range o.pendingResponsesEvents {
		sequences = append(sequences, sequence)
	}
	sort.Slice(sequences, func(i, j int) bool { return sequences[i] < sequences[j] })
	for _, sequence := range sequences {
		if sequence <= o.lastResponsesSequence {
			delete(o.pendingResponsesEvents, sequence)
			continue
		}
		event := o.pendingResponsesEvents[sequence]
		delete(o.pendingResponsesEvents, sequence)
		forceCheckpoint = o.applyResponsesEventLocked(event, sequence) || forceCheckpoint
	}
	o.responsesSequenceInitialized = true
	o.nextResponsesSequence = o.lastResponsesSequence + 1
	o.pendingResponsesSince = time.Time{}
	return forceCheckpoint
}

func (o *deliveredChatStreamObserver) applyResponsesEventLocked(
	event deliveredChatResponsesEvent,
	sequence int64,
) bool {
	if sequence <= o.lastResponsesSequence {
		return false
	}
	o.lastResponsesSequence = sequence
	o.captureResponsesMetadataLocked(event.payload)
	forceCheckpoint := false
	switch event.eventType {
	case "response.created":
		return false
	case "response.reasoning_summary_part.added",
		"response.reasoning_summary_text.delta",
		"response.reasoning_summary_text.done",
		"response.reasoning_summary_part.done":
		forceCheckpoint = o.applyReasoningSummaryEventLocked(
			event.eventType,
			event.payload,
			sequence,
		)
	case "response.output_item.added", "response.output_item.done":
		forceCheckpoint = o.applyReasoningOutputItemLocked(event.eventType, event.payload, sequence)
	case "response.completed", "response.incomplete", "response.failed":
		o.applyResponsesTerminalLocked(event.eventType, event.payload, sequence)
		forceCheckpoint = true
	}
	return forceCheckpoint
}

func (o *deliveredChatStreamObserver) captureResponsesMetadataLocked(payload []byte) {
	responseID := firstDeliveredChatString(payload, "response_id", "response.id")
	if len(responseID) > 128 {
		responseID = ""
	}
	if responseID != "" && responseID != o.responseID {
		o.responseID = responseID
		for _, observed := range o.activities {
			observed.activity.ResponseID = responseID
		}
		if o.hasPersistableActivityLocked() {
			o.markSnapshotCheckpointDirtyLocked()
		}
	}
	mode := firstDeliveredChatString(payload, "reasoning.mode", "response.reasoning.mode", "reasoning_mode")
	effort := firstDeliveredChatString(payload, "reasoning.effort", "response.reasoning.effort", "reasoning_effort")
	if len(mode) > 32 {
		mode = ""
	}
	if len(effort) > 32 {
		effort = ""
	}
	effectiveMode := o.reasoningMode
	if mode != "" && !o.modeAuthoritative {
		o.reasoningMode = mode
		effectiveMode = mode
	}
	effortCleared := effectiveMode == service.WebChatReasoningModePro && o.reasoningEffort != ""
	if effectiveMode == service.WebChatReasoningModePro {
		o.reasoningEffort = ""
		effort = ""
	} else if effort != "" && !o.effortAuthoritative {
		o.reasoningEffort = effort
	}
	modeApplied := mode != "" && !o.modeAuthoritative
	effortApplied := effort != "" && !o.effortAuthoritative
	if modeApplied || effortApplied || effortCleared {
		for _, observed := range o.activities {
			if modeApplied {
				observed.activity.ReasoningMode = mode
			}
			if effortApplied {
				observed.activity.ReasoningEffort = effort
			} else if effortCleared {
				observed.activity.ReasoningEffort = ""
			}
		}
		if o.hasPersistableActivityLocked() {
			o.markSnapshotCheckpointDirtyLocked()
		}
	}
}

func firstDeliveredChatString(payload []byte, paths ...string) string {
	for _, path := range paths {
		if value := strings.TrimSpace(gjson.GetBytes(payload, path).String()); value != "" {
			return value
		}
	}
	return ""
}

func deliveredChatOutputIndex(payload []byte) (int, bool) {
	value := gjson.GetBytes(payload, "output_index")
	if !value.Exists() || value.Type != gjson.Number || value.Int() < 0 {
		return 0, false
	}
	return int(value.Int()), true
}

func (o *deliveredChatStreamObserver) deliveredChatActivityCoordinatesLocked(
	payload []byte,
) (string, int, int) {
	outputIndex, _ := deliveredChatOutputIndex(payload)
	summaryIndex := int(gjson.GetBytes(payload, "summary_index").Int())
	itemID := firstDeliveredChatString(payload, "item_id", "item.id")
	if itemID == "" {
		if identity, exists := o.outputItemIdentities[outputIndex]; exists {
			itemID = identity.itemID
		}
		if itemID == "" {
			itemID = fmt.Sprintf("reasoning-output-%d", outputIndex)
		}
	}
	return itemID, outputIndex, summaryIndex
}

func (o *deliveredChatStreamObserver) applyReasoningSummaryEventLocked(
	eventType string,
	payload []byte,
	sequence int64,
) bool {
	outputIndex, valid := deliveredChatOutputIndex(payload)
	if !valid {
		return false
	}
	identity, confirmed := o.outputItemIdentities[outputIndex]
	if !confirmed {
		o.bufferReasoningSummaryEventLocked(eventType, payload, sequence, outputIndex)
		return false
	}
	if !identity.reasoning || !deliveredChatSummaryMatchesIdentity(payload, identity) {
		return false
	}
	return o.applyConfirmedReasoningSummaryEventLocked(eventType, payload, sequence)
}

func (o *deliveredChatStreamObserver) applyConfirmedReasoningSummaryEventLocked(
	eventType string,
	payload []byte,
	sequence int64,
) bool {
	switch eventType {
	case "response.reasoning_summary_part.added":
		o.applyReasoningPartAddedLocked(payload, sequence)
	case "response.reasoning_summary_text.delta":
		o.applyReasoningDeltaLocked(payload, sequence)
	case "response.reasoning_summary_text.done":
		o.applyReasoningTextDoneLocked(payload, sequence)
		return true
	case "response.reasoning_summary_part.done":
		o.applyReasoningPartDoneLocked(payload, sequence)
		return true
	}
	return false
}

func (o *deliveredChatStreamObserver) bufferReasoningSummaryEventLocked(
	eventType string,
	payload []byte,
	sequence int64,
	outputIndex int,
) {
	if o.pendingSummaryCount >= deliveredChatPendingSummaryEventLimit {
		return
	}
	o.pendingSummaryEvents[outputIndex] = append(
		o.pendingSummaryEvents[outputIndex],
		deliveredChatPendingSummaryEvent{
			eventType: eventType,
			payload:   append([]byte(nil), payload...),
			sequence:  sequence,
		},
	)
	o.pendingSummaryCount++
}

func deliveredChatSummaryMatchesIdentity(
	payload []byte,
	identity deliveredChatOutputItemIdentity,
) bool {
	itemID := firstDeliveredChatString(payload, "item_id", "item.id")
	return itemID == "" || itemID == identity.itemID
}

func (o *deliveredChatStreamObserver) registerOutputItemIdentityLocked(
	outputIndex int,
	itemID string,
	itemType string,
) []deliveredChatPendingSummaryEvent {
	if outputIndex < 0 {
		return nil
	}
	if itemID == "" {
		itemID = fmt.Sprintf("reasoning-output-%d", outputIndex)
	}
	identity, exists := o.outputItemIdentities[outputIndex]
	if !exists {
		identity = deliveredChatOutputItemIdentity{
			itemID:    itemID,
			reasoning: itemType == "reasoning",
		}
		o.outputItemIdentities[outputIndex] = identity
	}
	pending := o.pendingSummaryEvents[outputIndex]
	delete(o.pendingSummaryEvents, outputIndex)
	o.pendingSummaryCount -= len(pending)
	if !identity.reasoning {
		return nil
	}
	filtered := pending[:0]
	for _, event := range pending {
		if deliveredChatSummaryMatchesIdentity(event.payload, identity) {
			filtered = append(filtered, event)
		}
	}
	sort.SliceStable(filtered, func(i, j int) bool {
		return filtered[i].sequence < filtered[j].sequence
	})
	return filtered
}

func (o *deliveredChatStreamObserver) replayReasoningSummaryEventsLocked(
	pending []deliveredChatPendingSummaryEvent,
) bool {
	forceCheckpoint := false
	for _, event := range pending {
		forceCheckpoint = o.applyConfirmedReasoningSummaryEventLocked(
			event.eventType,
			event.payload,
			event.sequence,
		) || forceCheckpoint
	}
	return forceCheckpoint
}

func validDeliveredChatActivityCoordinates(itemID string, outputIndex, summaryIndex int) bool {
	return itemID != "" && len(itemID) <= 128 && outputIndex >= 0 && summaryIndex >= 0
}

func (o *deliveredChatStreamObserver) ensureActivityLocked(
	itemID string,
	outputIndex int,
	summaryIndex int,
	sequence int64,
) *deliveredChatObservedActivity {
	if itemID == "" {
		itemID = fmt.Sprintf("reasoning-output-%d", outputIndex)
	}
	key := deliveredChatActivityKey{ItemID: itemID, OutputIndex: outputIndex, SummaryIndex: summaryIndex}
	if observed := o.activities[key]; observed != nil {
		if sequence < observed.activity.SequenceStart {
			observed.activity.SequenceStart = sequence
		}
		if sequence > observed.activity.SequenceEnd {
			observed.activity.SequenceEnd = sequence
		}
		return observed
	}
	o.nextSortOrder++
	observed := &deliveredChatObservedActivity{activity: service.ChatMessageActivity{
		ResponseID:      o.responseID,
		Source:          service.ChatMessageActivitySourceOpenAIResponses,
		ActivityType:    service.ChatMessageActivityTypeReasoningSummary,
		ItemID:          itemID,
		OutputIndex:     outputIndex,
		SummaryIndex:    summaryIndex,
		SortOrder:       o.nextSortOrder,
		Status:          service.ChatMessageActivityStatusInProgress,
		SequenceStart:   sequence,
		SequenceEnd:     sequence,
		ReasoningMode:   o.reasoningMode,
		ReasoningEffort: o.reasoningEffort,
		StartedAt:       time.Now().UTC(),
		Metadata:        json.RawMessage(`{}`),
	}}
	o.activities[key] = observed
	return observed
}

func (o *deliveredChatStreamObserver) applyReasoningPartAddedLocked(payload []byte, sequence int64) {
	partType := firstDeliveredChatString(payload, "part.type")
	if partType != "" && partType != "summary_text" {
		return
	}
	itemID, outputIndex, summaryIndex := o.deliveredChatActivityCoordinatesLocked(payload)
	if !validDeliveredChatActivityCoordinates(itemID, outputIndex, summaryIndex) {
		return
	}
	observed := o.ensureActivityLocked(itemID, outputIndex, summaryIndex, sequence)
	if text := gjson.GetBytes(payload, "part.text"); text.Exists() && !observed.textDone {
		observed.replaceText(text.String())
	}
	o.touchActivityLocked(observed, sequence, service.ChatMessageActivityStatusInProgress, false, "response.reasoning_summary_part.added")
}

func (o *deliveredChatStreamObserver) applyReasoningDeltaLocked(payload []byte, sequence int64) {
	itemID, outputIndex, summaryIndex := o.deliveredChatActivityCoordinatesLocked(payload)
	if !validDeliveredChatActivityCoordinates(itemID, outputIndex, summaryIndex) {
		return
	}
	observed := o.ensureActivityLocked(itemID, outputIndex, summaryIndex, sequence)
	if !observed.textDone {
		observed.appendText(gjson.GetBytes(payload, "delta").String())
	}
	o.touchActivityLocked(observed, sequence, service.ChatMessageActivityStatusInProgress, false, "response.reasoning_summary_text.delta")
}

func (o *deliveredChatStreamObserver) applyReasoningTextDoneLocked(payload []byte, sequence int64) {
	itemID, outputIndex, summaryIndex := o.deliveredChatActivityCoordinatesLocked(payload)
	if !validDeliveredChatActivityCoordinates(itemID, outputIndex, summaryIndex) {
		return
	}
	observed := o.ensureActivityLocked(itemID, outputIndex, summaryIndex, sequence)
	observed.replaceText(gjson.GetBytes(payload, "text").String())
	observed.textDone = true
	o.touchActivityLocked(observed, sequence, service.ChatMessageActivityStatusCompleted, true, "response.reasoning_summary_text.done")
}

func (o *deliveredChatStreamObserver) applyReasoningPartDoneLocked(payload []byte, sequence int64) {
	partType := firstDeliveredChatString(payload, "part.type")
	if partType != "" && partType != "summary_text" {
		return
	}
	itemID, outputIndex, summaryIndex := o.deliveredChatActivityCoordinatesLocked(payload)
	if !validDeliveredChatActivityCoordinates(itemID, outputIndex, summaryIndex) {
		return
	}
	observed := o.ensureActivityLocked(itemID, outputIndex, summaryIndex, sequence)
	if text := gjson.GetBytes(payload, "part.text"); text.Exists() && !observed.textDone {
		observed.replaceText(text.String())
	}
	status := service.ChatMessageActivityStatusCompleted
	switch firstDeliveredChatString(payload, "status", "part.status") {
	case service.ChatMessageActivityStatusIncomplete:
		status = service.ChatMessageActivityStatusIncomplete
	case service.ChatMessageActivityStatusFailed:
		status = service.ChatMessageActivityStatusFailed
	case service.ChatMessageActivityStatusInterrupted,
		service.ChatMessageActivityStatusDisconnected:
		status = service.ChatMessageActivityStatusDisconnected
	case service.ChatMessageActivityStatusStopped:
		status = service.ChatMessageActivityStatusStopped
	}
	o.touchActivityLocked(observed, sequence, status, true, "response.reasoning_summary_part.done")
}

func (o *deliveredChatStreamObserver) applyReasoningOutputItemLocked(
	eventType string,
	payload []byte,
	sequence int64,
) bool {
	item := gjson.GetBytes(payload, "item")
	itemType := strings.TrimSpace(item.Get("type").String())
	outputIndex, valid := deliveredChatOutputIndex(payload)
	if !valid {
		return false
	}
	itemID := strings.TrimSpace(item.Get("id").String())
	pending := o.registerOutputItemIdentityLocked(outputIndex, itemID, itemType)
	identity := o.outputItemIdentities[outputIndex]
	itemID = identity.itemID
	if !identity.reasoning {
		return false
	}
	if !validDeliveredChatActivityCoordinates(itemID, outputIndex, 0) {
		return false
	}
	// Summary lifecycle events can arrive before the output item that proves
	// their identity. They are older than this item event, so replay them before
	// applying an output_item.done snapshot; otherwise a stale delta could be
	// appended after the final summary and downgrade its completed status.
	forceCheckpoint := o.replayReasoningSummaryEventsLocked(pending)
	status := service.ChatMessageActivityStatusInProgress
	terminal := eventType == "response.output_item.done"
	if terminal {
		status = service.ChatMessageActivityStatusCompleted
	}
	summaries := item.Get("summary").Array()
	if len(summaries) == 0 {
		observed := o.ensureActivityLocked(itemID, outputIndex, 0, sequence)
		o.touchActivityLocked(observed, sequence, status, terminal, eventType)
		return forceCheckpoint || terminal && strings.TrimSpace(observed.currentText()) != ""
	}
	for summaryIndex, summary := range summaries {
		if summaryType := strings.TrimSpace(summary.Get("type").String()); summaryType != "" && summaryType != "summary_text" {
			continue
		}
		observed := o.ensureActivityLocked(itemID, outputIndex, summaryIndex, sequence)
		if terminal && !observed.textDone {
			observed.replaceText(summary.Get("text").String())
		}
		o.touchActivityLocked(observed, sequence, status, terminal, eventType)
	}
	return forceCheckpoint || terminal && len(o.snapshotActivitiesLocked()) > 0
}

func (o *deliveredChatStreamObserver) applyResponsesTerminalLocked(eventType string, payload []byte, sequence int64) {
	status := service.ChatMessageActivityStatusCompleted
	switch eventType {
	case "response.incomplete":
		status = service.ChatMessageActivityStatusIncomplete
	case "response.failed":
		status = service.ChatMessageActivityStatusFailed
	}
	o.responsesTerminalSeen = true
	o.responsesTerminalStatus = status
	outputs := gjson.GetBytes(payload, "response.output").Array()
	for outputIndex, item := range outputs {
		itemType := strings.TrimSpace(item.Get("type").String())
		itemID := strings.TrimSpace(item.Get("id").String())
		pending := o.registerOutputItemIdentityLocked(outputIndex, itemID, itemType)
		identity := o.outputItemIdentities[outputIndex]
		if !identity.reasoning {
			continue
		}
		itemID = identity.itemID
		if !validDeliveredChatActivityCoordinates(itemID, outputIndex, 0) {
			continue
		}
		o.replayReasoningSummaryEventsLocked(pending)
		for summaryIndex, summary := range item.Get("summary").Array() {
			if summaryType := strings.TrimSpace(summary.Get("type").String()); summaryType != "" && summaryType != "summary_text" {
				continue
			}
			observed := o.ensureActivityLocked(itemID, outputIndex, summaryIndex, sequence)
			if !observed.textDone {
				observed.replaceText(summary.Get("text").String())
			}
			o.touchActivityLocked(observed, sequence, status, true, eventType)
		}
	}
	// A terminal response is the last opportunity to identify an output item.
	// Any summary still orphaned after inspecting response.output is untrusted.
	o.pendingSummaryEvents = make(map[int][]deliveredChatPendingSummaryEvent)
	o.pendingSummaryCount = 0
	for _, observed := range o.activities {
		o.touchActivityLocked(observed, sequence, status, true, eventType)
	}
	// A Responses terminal event is a durable boundary even when it contains no
	// reasoning item. Persist the latest delivered content before finalization.
	o.markSnapshotCheckpointDirtyLocked()
}

func isDeliveredChatResponsesTerminalEvent(eventType string) bool {
	switch eventType {
	case "response.completed", "response.incomplete", "response.failed":
		return true
	default:
		return false
	}
}

func isDeliveredChatResponsesBoundaryEvent(eventType string) bool {
	switch eventType {
	case "response.output_item.done",
		"response.reasoning_summary_text.done",
		"response.reasoning_summary_part.done",
		"response.completed",
		"response.incomplete",
		"response.failed":
		return true
	default:
		return false
	}
}

func (o *deliveredChatStreamObserver) touchActivityLocked(
	observed *deliveredChatObservedActivity,
	sequence int64,
	status string,
	terminal bool,
	eventType string,
) {
	if observed == nil {
		return
	}
	if sequence < observed.activity.SequenceStart {
		observed.activity.SequenceStart = sequence
	}
	if sequence > observed.activity.SequenceEnd {
		observed.activity.SequenceEnd = sequence
	}
	if status == service.ChatMessageActivityStatusCompleted {
		switch observed.activity.Status {
		case service.ChatMessageActivityStatusIncomplete,
			service.ChatMessageActivityStatusFailed,
			service.ChatMessageActivityStatusInterrupted,
			service.ChatMessageActivityStatusStopped,
			service.ChatMessageActivityStatusDisconnected:
			status = observed.activity.Status
		}
	}
	observed.activity.Status = status
	observed.activity.ResponseID = o.responseID
	observed.activity.ReasoningMode = o.reasoningMode
	observed.activity.ReasoningEffort = o.reasoningEffort
	if terminal && observed.activity.CompletedAt == nil {
		completedAt := time.Now().UTC()
		observed.activity.CompletedAt = &completedAt
	}
	metadata, _ := json.Marshal(map[string]string{"last_event": eventType})
	observed.activity.Metadata = metadata
	if strings.TrimSpace(observed.currentText()) != "" {
		o.markSnapshotCheckpointDirtyLocked()
	}
}

func (o *deliveredChatStreamObserver) interruptActivitiesLocked() {
	if o.responsesTerminalSeen {
		return
	}
	changed := false
	for _, observed := range o.activities {
		if observed.activity.Status == service.ChatMessageActivityStatusFailed ||
			observed.activity.Status == service.ChatMessageActivityStatusIncomplete {
			continue
		}
		observed.activity.Status = service.ChatMessageActivityStatusDisconnected
		completedAt := time.Now().UTC()
		observed.activity.CompletedAt = &completedAt
		metadata, _ := json.Marshal(map[string]string{"last_event": "client_disconnected"})
		observed.activity.Metadata = metadata
		changed = changed || strings.TrimSpace(observed.currentText()) != ""
	}
	if changed {
		o.markSnapshotCheckpointDirtyLocked()
	}
}

func (o *deliveredChatStreamObserver) hasPersistableActivityLocked() bool {
	for _, observed := range o.activities {
		if strings.TrimSpace(observed.currentText()) != "" {
			return true
		}
	}
	return false
}

func (o *deliveredChatStreamObserver) snapshotActivitiesLocked() []service.ChatMessageActivity {
	activities := make([]service.ChatMessageActivity, 0, len(o.activities))
	for _, observed := range o.activities {
		activity := observed.activity
		activity.Text = observed.currentText()
		if strings.TrimSpace(activity.Text) == "" {
			continue
		}
		activity.Metadata = append(json.RawMessage(nil), activity.Metadata...)
		if activity.CompletedAt != nil {
			completedAt := *activity.CompletedAt
			activity.CompletedAt = &completedAt
		}
		activities = append(activities, activity)
	}
	sort.SliceStable(activities, func(i, j int) bool {
		if activities[i].SortOrder != activities[j].SortOrder {
			return activities[i].SortOrder < activities[j].SortOrder
		}
		if activities[i].OutputIndex != activities[j].OutputIndex {
			return activities[i].OutputIndex < activities[j].OutputIndex
		}
		return activities[i].SummaryIndex < activities[j].SummaryIndex
	})
	return activities
}

func (o *deliveredChatStreamObserver) checkpointPayloadLocked(sequence int64) deliveredChatStreamCheckpoint {
	return deliveredChatStreamCheckpoint{
		Sequence:        sequence,
		Content:         o.content.String(),
		ResponseID:      o.responseID,
		ReasoningMode:   o.reasoningMode,
		ReasoningEffort: o.reasoningEffort,
		Activities:      o.snapshotActivitiesLocked(),
	}
}

func (o *deliveredChatStreamObserver) markSnapshotCheckpointDirtyLocked() {
	o.snapshotCheckpointDirty = true
	o.snapshotCheckpointEvents++
}

func (o *deliveredChatStreamObserver) maybeWriteSnapshotCheckpointLocked(force bool) {
	if o.snapshotCheckpoint == nil || !o.snapshotCheckpointDirty {
		return
	}
	if !force && o.content.Len() == 0 && !o.hasPersistableActivityLocked() {
		return
	}
	if !force && !o.lastSnapshotCheckpointAt.IsZero() &&
		time.Since(o.lastSnapshotCheckpointAt) < deliveredChatActivityCheckpointInterval &&
		o.snapshotCheckpointEvents < deliveredChatActivityCheckpointBatch {
		return
	}
	// Checkpoints store canonical full text so refresh and other devices can
	// recover without replaying deltas. Gate non-terminal snapshots by geometric
	// per-part growth; this keeps total serialized bytes linear for long
	// summaries instead of rewriting every accumulated prefix at a fixed rate.
	if !force && !o.snapshotGrowthReachedLocked() {
		return
	}
	o.checkpointSeq++
	o.enqueueSnapshotCheckpointLocked(o.checkpointPayloadLocked(o.checkpointSeq))
	o.recordSnapshotSizesLocked()
	o.snapshotCheckpointDirty = false
	o.snapshotCheckpointEvents = 0
	o.lastSnapshotCheckpointAt = time.Now()
}

func (o *deliveredChatStreamObserver) snapshotActivitySizesLocked() map[deliveredChatActivityKey]int {
	sizes := make(map[deliveredChatActivityKey]int, len(o.activities))
	for key, observed := range o.activities {
		if observed == nil {
			continue
		}
		size := observed.text.Len()
		if size > 0 {
			sizes[key] = size
		}
	}
	return sizes
}

func (o *deliveredChatStreamObserver) snapshotGrowthReachedLocked() bool {
	contentSize := o.content.Len()
	if contentSize > 0 && (o.lastSnapshotContentSize <= 0 ||
		contentSize < o.lastSnapshotContentSize ||
		contentSize-o.lastSnapshotContentSize >= o.lastSnapshotContentSize) {
		return true
	}
	current := o.snapshotActivitySizesLocked()
	for key, size := range current {
		previous, exists := o.lastSnapshotActivitySize[key]
		if !exists || previous <= 0 || size < previous || size-previous >= previous {
			return true
		}
	}
	return false
}

func (o *deliveredChatStreamObserver) recordSnapshotSizesLocked() {
	o.lastSnapshotActivitySize = o.snapshotActivitySizesLocked()
	o.lastSnapshotContentSize = o.content.Len()
}

// enqueueSnapshotCheckpointLocked coalesces snapshots behind at most one
// in-flight database write. It is called while the observer lock is held, but
// the repository callback always runs on the dispatcher goroutine without that
// lock, so a slow checkpoint cannot stall SSE delivery.
func (o *deliveredChatStreamObserver) enqueueSnapshotCheckpointLocked(
	checkpoint deliveredChatStreamCheckpoint,
) {
	if o.snapshotCheckpoint == nil {
		return
	}
	o.snapshotDispatchMu.Lock()
	checkpointCopy := checkpoint
	o.snapshotDispatchPending = &checkpointCopy
	if checkpoint.Sequence > o.snapshotDispatchLatestSequence {
		o.snapshotDispatchLatestSequence = checkpoint.Sequence
	}
	if !o.snapshotDispatchRunning {
		o.snapshotDispatchRunning = true
		callback := o.snapshotCheckpoint
		go o.runSnapshotCheckpointDispatcher(callback)
	}
	o.snapshotDispatchMu.Unlock()
}

func (o *deliveredChatStreamObserver) runSnapshotCheckpointDispatcher(
	callback func(deliveredChatStreamCheckpoint) error,
) {
	for {
		o.snapshotDispatchMu.Lock()
		pending := o.snapshotDispatchPending
		o.snapshotDispatchPending = nil
		if pending == nil {
			o.snapshotDispatchRunning = false
			o.snapshotDispatchCond.Broadcast()
			o.snapshotDispatchMu.Unlock()
			return
		}
		o.snapshotDispatchMu.Unlock()

		err := callback(*pending)

		o.snapshotDispatchMu.Lock()
		if err != nil && o.snapshotDispatchErr == nil {
			o.snapshotDispatchErr = err
		}
		if pending.Sequence > o.snapshotDispatchFinishedSequence {
			o.snapshotDispatchFinishedSequence = pending.Sequence
		}
		o.snapshotDispatchCond.Broadcast()
		o.snapshotDispatchMu.Unlock()
	}
}

func (o *deliveredChatStreamObserver) snapshotCheckpointTarget() int64 {
	o.snapshotDispatchMu.Lock()
	defer o.snapshotDispatchMu.Unlock()
	return o.snapshotDispatchLatestSequence
}

func (o *deliveredChatStreamObserver) waitForSnapshotCheckpoints(target int64) error {
	if target <= 0 {
		return nil
	}
	o.snapshotDispatchMu.Lock()
	defer o.snapshotDispatchMu.Unlock()
	for o.snapshotDispatchFinishedSequence < target &&
		(o.snapshotDispatchRunning || o.snapshotDispatchPending != nil) {
		o.snapshotDispatchCond.Wait()
	}
	return o.snapshotDispatchErr
}

func (o *deliveredChatStreamObserver) writeCheckpoint() {
	if o.checkpoint != nil {
		o.checkpointSeq++
		if err := o.checkpoint(o.checkpointSeq, o.content.String()); err != nil {
			o.checkpointErr = err
			return
		}
	} else if o.snapshotCheckpoint == nil {
		// Preserve the historical monotonic snapshot sequence for observers that
		// have no persistence callback at all.
		o.checkpointSeq++
	}
	if o.snapshotCheckpoint != nil {
		o.markSnapshotCheckpointDirtyLocked()
		o.maybeWriteSnapshotCheckpointLocked(false)
	}
}
