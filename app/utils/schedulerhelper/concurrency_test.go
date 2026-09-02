package schedulerhelper

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"
)

// recordLogger 记录 Warn/Error 调用的测试用 JobLogger（避免测试写日志文件）
type recordLogger struct {
	mu     sync.Mutex
	warns  []string
	errors []string
}

func (r *recordLogger) Debug(jobID, format string, args ...interface{}) {}
func (r *recordLogger) Info(jobID, format string, args ...interface{})  {}
func (r *recordLogger) Warn(jobID, format string, args ...interface{}) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.warns = append(r.warns, fmt.Sprintf(format, args...))
}
func (r *recordLogger) Error(jobID, format string, args ...interface{}) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.errors = append(r.errors, fmt.Sprintf(format, args...))
}
func (r *recordLogger) Fatal(jobID, format string, args ...interface{}) {}
func (r *recordLogger) LogJobExecution(result *JobResult)              {}
func (r *recordLogger) LogJobLifecycle(job *Job, action string)        {}
func (r *recordLogger) Close() error                                   { return nil }

func (r *recordLogger) errorCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.errors)
}

// newBlockingExecutor 返回阻塞到 release 关闭（或 ctx 超时）才返回的执行器
func newBlockingExecutor(name string, release <-chan struct{}) *MockExecutor {
	e := NewMockExecutor(name)
	e.SetExecuteFn(func(ctx context.Context, job *Job) error {
		select {
		case <-release:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})
	return e
}

// TestBlockDiscardConcurrent 修复#14：BlockDiscard 下并发触发只允许 1 个执行
// （修复前 canExecute 与计数递增分离，多个触发可同时通过检查导致超限执行）
func TestBlockDiscardConcurrent(t *testing.T) {
	s := NewJobScheduler()
	s.logger = &recordLogger{}
	defer s.Stop()

	release := make(chan struct{})
	executor := newBlockingExecutor("blocking", release)
	s.RegisterExecutor(executor)

	job := &Job{
		ID: "job-discard", Group: "test-group", Name: "并发丢弃测试", ExecutorName: "blocking",
		Status: StatusEnabled, CronExpression: "0 0 0 1 1 *",
		BlockingPolicy: BlockDiscard, Timeout: 10 * time.Second,
	}
	if _, err := s.AddOrUpdateJob(job); err != nil {
		t.Fatal(err)
	}

	fn := s.createJobFunc(job)
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); fn() }()
	}
	time.Sleep(100 * time.Millisecond) // 等 10 次检查全部完成（1 个在跑、9 个被拒）
	close(release)
	wg.Wait()

	if got := executor.GetExecCount(); got != 1 {
		t.Fatalf("BlockDiscard 并发 10 次触发应只执行 1 次，实际 %d 次", got)
	}
	if job.RunningCount != 0 {
		t.Fatalf("执行完毕后 RunningCount 应归零，实际 %d", job.RunningCount)
	}
}

// TestBlockParallelConcurrent 修复#14：BlockParallel 并发上限生效
func TestBlockParallelConcurrent(t *testing.T) {
	s := NewJobScheduler()
	s.logger = &recordLogger{}
	defer s.Stop()

	release := make(chan struct{})
	executor := newBlockingExecutor("blocking", release)
	s.RegisterExecutor(executor)

	job := &Job{
		ID: "job-parallel", Group: "test-group", Name: "并发上限测试", ExecutorName: "blocking",
		Status: StatusEnabled, CronExpression: "0 0 0 1 1 *",
		BlockingPolicy: BlockParallel, ParallelNum: 2, Timeout: 10 * time.Second,
	}
	if _, err := s.AddOrUpdateJob(job); err != nil {
		t.Fatal(err)
	}

	fn := s.createJobFunc(job)
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); fn() }()
	}
	time.Sleep(100 * time.Millisecond) // 等检查完成（2 个在跑、8 个被拒）
	close(release)
	wg.Wait()

	if got := executor.GetExecCount(); got != 2 {
		t.Fatalf("BlockParallel 并行数 2 时并发 10 次触发应恰好执行 2 次，实际 %d 次", got)
	}
}

// TestSendResultChannelFull 修复#15：通道满时结果丢弃有留痕、不 panic
func TestSendResultChannelFull(t *testing.T) {
	origTimeout := resultSendTimeout
	resultSendTimeout = 10 * time.Millisecond
	defer func() { resultSendTimeout = origTimeout }()

	s := NewJobScheduler()
	l := &recordLogger{}
	s.logger = l

	// 填满缓冲
	for i := 0; i < cap(s.jobResults); i++ {
		s.jobResults <- &JobResult{JobID: "filler"}
	}

	// 再发送应走超时丢弃分支：不 panic、记录错误日志
	s.sendResult(&JobResult{JobID: "job-x", Status: "FAILED", RetryCount: 1})

	if l.errorCount() == 0 {
		t.Fatal("通道满丢弃结果时应记录错误日志")
	}

	// 排空缓冲，便于 Stop 正常 close
	for i := 0; i < cap(s.jobResults); i++ {
		<-s.jobResults
	}
	s.Stop()
}

// TestStopWhileJobsRunning 修复#16：任务在跑时 Stop 不 panic，
// 复现"goroutine 已越过 createJobFunc 但尚未 wg.Add"即 Stop 的竞态窗口
func TestStopWhileJobsRunning(t *testing.T) {
	for round := 0; round < 5; round++ {
		s := NewJobScheduler()
		s.logger = &recordLogger{}

		release := make(chan struct{})
		executor := newBlockingExecutor("blocking", release)
		s.RegisterExecutor(executor)

		job := &Job{
			ID: "job-stop", Group: "test-group", Name: "停止竞态测试", ExecutorName: "blocking",
			Status: StatusEnabled, CronExpression: "* * * * * *",
			BlockingPolicy: BlockParallel, ParallelNum: 100, Timeout: 300 * time.Millisecond,
		}
		if _, err := s.AddOrUpdateJob(job); err != nil {
			t.Fatal(err)
		}
		s.Start()

		go s.createJobFunc(job)() // 手动触发执行
		runtime.Gosched()         // 让 goroutine 有机会起跑但未必到达 wg.Add
		s.Stop()                  // 立即停止（置位 stopped → wg.Wait → close）
		close(release)            // 兜底放行（执行器 ctx 超时也会自行放行）

		// Stop 后新执行请求应被拒绝
		if err := s.ExecuteNow(job.ID); err == nil {
			t.Fatalf("第 %d 轮：Stop 之后 ExecuteNow 应返回错误", round)
		}
	}
}

// TestStopIdempotent 修复#16：二次 Stop 不 panic（stopOnce 防 double-close）
func TestStopIdempotent(t *testing.T) {
	s := NewJobScheduler()
	s.logger = &recordLogger{}
	s.Stop()
	s.Stop()
}
