## go-stress-runner

> A simple stress runner to execute tasks with the target concurrent number

![CI](https://github.com/rgb-24bit/go-stress-runner/workflows/CI/badge.svg)

### Install

```go
go get -u github.com/rgb-24bit/go-stress-runner
```

### Usage

```go
stressRunner := runner.NewStressRunner(100, time.Minute, func() {
	// do something
})
stressRunner.Start(context.TODO())
stressRunner.Wait()
```

