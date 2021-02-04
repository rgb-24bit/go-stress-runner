package runner

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
)

type Runnable func()

type runner struct {
	id      int32
	done    chan struct{}
	next    *runner
	stopped int32
}

type StressRunner struct {
	cps       int32
	cnt       int32
	cancel    context.CancelFunc
	total     int32
	task      Runnable
	sentinel  *runner
	waitGroup sync.WaitGroup
	duration  time.Duration
}

func NewStressRunner(cps int32, duration time.Duration, task Runnable) *StressRunner {
	return &StressRunner{
		cps:      cps,
		task:     task,
		sentinel: &runner{},
		duration: duration,
	}
}

func (s *StressRunner) Start(ctx context.Context) {
	ctx, s.cancel = context.WithTimeout(ctx, s.duration)

	for {
		select {
		case <-time.After(time.Second):
			s.step()
		case <-ctx.Done():
			goto STOPPED
		}
	}

STOPPED:
	s.stopRunners()
}

func (s *StressRunner) startRunner() {
	runner := &runner{
		done: make(chan struct{}),
		id:   atomic.AddInt32(&s.sentinel.id, 1),
		next: s.sentinel.next,
	}
	s.sentinel.next = runner

	go func() {
		logrus.Infof("runner [%d] start", runner.id)

		defer func() {
			if err := recover(); err != nil {
				logrus.Errorf("runner [%d] stopped abnormal, error %v", runner.id, err)
				atomic.StoreInt32(&runner.stopped, 1)
			}
			s.waitGroup.Done()
		}()

		s.waitGroup.Add(1)

		for {
			select {
			case <-runner.done:
				logrus.Infof("runner [%d] stopped", runner.id)
				atomic.StoreInt32(&runner.stopped, 1)
				return
			default:
				if atomic.LoadInt32(&s.cnt) < s.cps {
					atomic.AddInt32(&s.cnt, 1)
					s.task()
				}
				time.Sleep(25 * time.Millisecond)
			}
		}
	}()
}

func (s *StressRunner) stopRunner() {
	for s.sentinel.next != nil {
		if atomic.LoadInt32(&s.sentinel.next.stopped) == 0 {
			s.sentinel.next.done <- struct{}{}
			break
		}
		s.sentinel.next = s.sentinel.next.next
	}
}

func (s *StressRunner) stopRunners() {
	for s.sentinel.next != nil {
		s.sentinel.next.done <- struct{}{}
		s.sentinel.next = s.sentinel.next.next
	}

	s.waitGroup.Wait()

	s.total += s.cnt

	logrus.Infof("Stress runner stopped, total exec count %d", s.total)
}

func (s *StressRunner) Stop() {
	s.cancel()
	s.waitGroup.Wait()
}

func (s *StressRunner) step() {
	// get runner count
	prev, next := s.sentinel, s.sentinel.next

	var rc int32

	for next != nil {
		rc += 1

		if atomic.LoadInt32(&next.stopped) == 1 {
			prev.next = next.next
			continue
		}

		prev = next
		next = next.next
	}

	// record total exec count
	ec := atomic.SwapInt32(&s.cnt, 0)

	s.total += ec

	if ec == s.cps {
		return
	}

	if rc == 0 {
		s.startRunner()
		return
	}

	var gps = max(ec/rc, 1)

	logrus.Infof("gps %d, ec %d, rc %d", gps, ec, rc)
	if ec < s.cps {
		diff := int(max((s.cps-ec)/gps, 1))
		for i := 0; i < diff; i++ {
			s.startRunner()
		}
	} else {
		diff := int((ec - s.cps) / gps)
		for i := 0; i < diff; i++ {
			s.stopRunner()
		}
	}
}

func max(a, b int32) int32 {
	if a > b {
		return a
	}
	return b
}
