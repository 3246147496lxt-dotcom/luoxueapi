package service

// 国产供应商 Anthropic 协议转换路径的上游 SSE 行泵。
//
// CC×anthropic 与 Responses×anthropic 的上游请求上下文在客户端断开后
// 仍需继续读取一段时间以完成用量统计。如果上游保持连接但长时间不发
// 数据，直接调用 scanner.Scan 会永久阻塞，导致响应体和连接池槽位无法
// 归还。行泵以 gateway.stream_data_interval_timeout 作为逐行读取上限，
// 在超时后让调用方关闭响应体并结束排水。

import (
	"bufio"
	"errors"
	"io"
	"time"
)

// errAnthropicNativeStreamIdle 表示上游流读间隔超时。
var errAnthropicNativeStreamIdle = errors.New("stream data interval timeout")

type anthropicNativeLineEvent struct {
	line string
	err  error
}

// anthropicNativeLinePump 以独立 goroutine 泵送 scanner 的行，并对相邻行
// 的到达间隔施加 interval 上限。interval <= 0 时保持旧的无界读取行为。
type anthropicNativeLinePump struct {
	events   chan anthropicNativeLineEvent
	done     chan struct{}
	timer    *time.Timer
	interval time.Duration
}

func newAnthropicNativeLinePump(scanner *bufio.Scanner, interval time.Duration) *anthropicNativeLinePump {
	p := &anthropicNativeLinePump{
		events:   make(chan anthropicNativeLineEvent, 16),
		done:     make(chan struct{}),
		interval: interval,
	}
	if interval > 0 {
		p.timer = time.NewTimer(interval)
	}
	go func() {
		defer close(p.events)
		for scanner.Scan() {
			select {
			case p.events <- anthropicNativeLineEvent{line: scanner.Text()}:
			case <-p.done:
				return
			}
		}
		if err := scanner.Err(); err != nil {
			select {
			case p.events <- anthropicNativeLineEvent{err: err}:
			case <-p.done:
			}
		}
	}()
	return p
}

// next returns the next line, io.EOF on normal stream completion, or the
// idle sentinel when no line arrives before interval.
func (p *anthropicNativeLinePump) next() (string, error) {
	var timeoutCh <-chan time.Time
	if p.timer != nil {
		timeoutCh = p.timer.C
	}
	// Prefer already-buffered lines (or a closed events channel) before
	// consulting the timer. Without this fast path, a timer firing at the same
	// instant the scanner reaches EOF could be selected nondeterministically and
	// report a false idle timeout after the complete stream was already queued.
	select {
	case ev, ok := <-p.events:
		if !ok {
			return "", io.EOF
		}
		p.resetTimer()
		return ev.line, ev.err
	default:
	}
	select {
	case ev, ok := <-p.events:
		if !ok {
			return "", io.EOF
		}
		p.resetTimer()
		return ev.line, ev.err
	case <-timeoutCh:
		return "", errAnthropicNativeStreamIdle
	}
}

func (p *anthropicNativeLinePump) resetTimer() {
	if p.timer == nil {
		return
	}
	if !p.timer.Stop() {
		select {
		case <-p.timer.C:
		default:
		}
	}
	p.timer.Reset(p.interval)
}

// stop terminates the pump goroutine. Callers should close the response body
// first when returning because of idle timeout, so a blocked scanner read is
// released promptly.
func (p *anthropicNativeLinePump) stop() {
	close(p.done)
	if p.timer != nil {
		if !p.timer.Stop() {
			select {
			case <-p.timer.C:
			default:
			}
		}
	}
}

func (s *OpenAIGatewayService) anthropicNativeStreamInterval() time.Duration {
	if s != nil && s.cfg != nil && s.cfg.Gateway.StreamDataIntervalTimeout > 0 {
		return time.Duration(s.cfg.Gateway.StreamDataIntervalTimeout) * time.Second
	}
	return 0
}
