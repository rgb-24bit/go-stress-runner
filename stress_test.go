package runner

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	"github.com/stretchr/testify/assert"
)

var _ = Describe("StressRunner", func() {
	t := GinkgoT()

	Context("funcs", func() {
		It("max", func() {
			assert.Equal(t, int32(1), max(1, 0))
			assert.Equal(t, int32(1), max(0, 1))
		})
	})

	Context("runner", func() {
		ctx := context.Background()

		It("cps 0", func() {
			sr := NewStressRunner(0, 3 * time.Second, func() {
				time.Sleep(time.Millisecond * 300)
			})
			sr.Start(ctx)
			sr.Wait()

			assert.Equal(t, int32(0), sr.ec)
		})

		It("cps 10", func() {
			sr := NewStressRunner(10, 3 * time.Second, func() {
				time.Sleep(time.Millisecond * 300)
			})
			sr.Start(ctx)
			sr.Wait()

			assert.True(t, sr.ec > 20)
		})

		It("stop", func() {
			sr := NewStressRunner(10, 3 * time.Second, func() {
				time.Sleep(time.Millisecond * 300)
			})

			sr.Start(ctx)
			sr.Stop()
			sr.Wait()

			assert.True(t, sr.ec < 10)
		})

		It("panic", func() {
			sr := NewStressRunner(10, 3 * time.Second, func() {
				time.Sleep(time.Millisecond * 300)
				panic("test")
			})
			sr.Start(ctx)
			sr.Wait()

			// sec 1 - ecs 1, rc 1, cps 10, new rc 9(10 - 1)
			// sec 2 - ecs 9, rc 9, cps 10, new rc 1(10 - 9)
			// sec 3 - ecs 1, rc 1
			assert.Equal(t, int32(11), sr.ec)
		})
	})
})
