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
	stop    chan struct{}
	next    *runner
	stopped int32
}

type StressRunner struct {
	cps        int32
	ecs        int32
	ec         int32
	stop       chan struct{}
	wait       sync.WaitGroup
	task       Runnable
	sentinel   *runner
	waitRunner sync.WaitGroup
	duration   time.Duration
}

func NewStressRunner(cps int32, duration time.Duration, task Runnable) *StressRunner {
	return &StressRunner{
		cps:      cps,
		task:     task,
		stop:     make(chan struct{}),
		sentinel: &runner{},
		duration: duration,
	}
}

func (s *StressRunner) Start(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, s.duration)

	s.startRunner()

	s.wait.Add(1)

	go func() {
		defer cancel()

		for {
			select {
			case <-time.After(time.Second):
				s.step()
			case <-ctx.Done():
				goto STOP
			case <-s.stop:
				goto STOP
			}
		}

	STOP:
		s.stopRunners()
	}()
}

func (s *StressRunner) startRunner() {
	runner := &runner{
		stop: make(chan struct{}),
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
			s.waitRunner.Done()
		}()

		s.waitRunner.Add(1)

		for {
			select {
			case <-runner.stop:
				logrus.Infof("runner [%d] stopped", runner.id)
				atomic.StoreInt32(&runner.stopped, 1)
				return
			default:
				if atomic.LoadInt32(&s.ecs) < s.cps {
					atomic.AddInt32(&s.ecs, 1)
					s.task()
				}
				time.Sleep(25 * time.Millisecond)
			}
		}
	}()
}

func (s *StressRunner) stopRunners() {
	for s.sentinel.next != nil {
		if atomic.LoadInt32(&s.sentinel.next.stopped) == 0 {
			s.sentinel.next.stop <- struct{}{}
		}
		s.sentinel.next = s.sentinel.next.next
	}

	s.waitRunner.Wait()

	s.ec += s.ecs

	s.wait.Done()

	logrus.Infof("Stress runner stopped, exec count %d", s.ec)
}

func (s *StressRunner) Stop() {
	s.stop <- struct{}{}
}

func (s *StressRunner) Wait() {
	s.wait.Wait()
}

func (s *StressRunner) step() {
	// get runner count
	prev, next := s.sentinel, s.sentinel.next

	var rc int32

	for next != nil {
		rc += 1

		if atomic.LoadInt32(&next.stopped) == 1 {
			prev.next = next.next
			next = next.next
			continue
		}

		prev = next
		next = next.next
	}

	// record exec count
	ecs := atomic.SwapInt32(&s.ecs, 0)
	s.ec += ecs

	if ecs == s.cps {
		return
	}

	if rc == 0 {
		s.startRunner()
		return
	}

	var gps = max(ecs/rc, 1)

	if ecs < s.cps {
		diff := int(max((s.cps-ecs)/gps, 1))
		for i := 0; i < diff; i++ {
			s.startRunner()
		}
	}
}

func max(a, b int32) int32 {
	if a > b {
		return a
	}
	return b
}
