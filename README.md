# Go Redis Queue Manger

Redis队列控制器，支持多worker并发队列，

支持使用Topic，Group，进行队列数据隔离


例子

```go
package go_redis_queue_manager

import (
	"fmt"
	"github.com/go-redis/redis"
	"os"
	"testing"
)

type StandaloneQueue struct {
}

func (b StandaloneQueue) Execute(payload *QueuePayload) *QueueResult {

	//fmt.Println(payload)

	return NewQueueResult(true, "StandaloneQueue.ok", nil)
}

type DemoDemoQueue struct {
}

func (b DemoDemoQueue) Execute(payload *QueuePayload) *QueueResult {

	//fmt.Println(payload)

	return NewQueueResult(true, "DemoDemoQueue.ok", nil)
}

type DemoDemo2Queue struct {
}

func (b DemoDemo2Queue) Execute(payload *QueuePayload) *QueueResult {

	//fmt.Println(payload)

	return NewQueueResult(true, "DemoDemoQueue2.ok ", nil)
}

func TestNewQueueManager(t *testing.T) {
	// 模拟系统ENV
	os.Setenv("DB_REDIS_BUS_HOST", "127.0.0.1")
	os.Setenv("DB_REDIS_BUS_PORT", "6379")
	os.Setenv("DB_REDIS_BUS_PASSWORD", "")

	host := os.Getenv("DB_REDIS_BUS_HOST")
	port := os.Getenv("DB_REDIS_BUS_PORT")
	password := os.Getenv("DB_REDIS_BUS_PASSWORD")

	// 需要先初始化redis实例
	rds := redis.NewClient(&redis.Options{
		Addr:     host + ":" + port, // remote host
		Password: password,          // password
	})

	// 初始化队列控制器
	qm := NewQueueManager()
	// 注册redis实例到队列控制器
	qm.UseRedis(rds)
	// 队列的worker数
	qm.WorkerNum = 4

	// 注册队列任务处理器，TOPIC::GROUP 方式命名，和入栈队列payload一致
	qm.Handlers = map[string]Queueable{
		"DEMO::DEMO":  DemoDemoQueue{},
		"DEMO::DEMO2": DemoDemo2Queue{},
		"STANDALONE":  StandaloneQueue{},
	}
	// 注册队列执行发生错误时，recovery的回调
	qm.RegisterOnInterrupt(func(stack string) {
		fmt.Println(stack)
	})

	go func() {
		// 模拟队列的插入
		for {
			qm.QueuePush(&QueuePayload{
				IsFast: true,
				Topic:  "DEMO",
				Group:  "DEMO",
				Body:   "do the job on channel[DEMO::DEMO]",
			})

			qm.QueuePush(&QueuePayload{
				IsFast: true,
				Topic:  "DEMO",
				Group:  "DEMO2",
				Body:   "do the job on channel[DEMO]",
			})
			qm.QueuePush(&QueuePayload{
				IsFast: true,
				Topic:  "STANDALONE",
				Group:  "",
				Body:   "do the job on STANDALONE",
			})

			//time.Sleep(time.Millisecond * 300)
		}
	}()

	// TOPIC + GROUP 为一个独立的队列通道
	go qm.QueueHandler("DEMO", "DEMO")
	go qm.QueueHandler("DEMO", "DEMO2")
	go qm.QueueHandler("STANDALONE", "")

	// 执行队列监听
	go qm.QueueRunner()
	select {}
}

```

执行效果：

```
Worker 0 FastQueues {a418c276-a95b-11ea-bd41-34363bd057d6 true DEMO DEMO2 do the job on channel[DEMO] 0 0} true DemoDemoQueue2.ok 
Worker 3 FastQueues {a418c9ce-a95b-11ea-bd41-34363bd057d6 true STANDALONE  do the job on STANDALONE 0 0} true StandaloneQueue.ok
Worker 2 FastQueues {a418cff0-a95b-11ea-bd41-34363bd057d6 true DEMO DEMO do the job on channel[DEMO::DEMO] 0 0} true DemoDemoQueue.ok
Worker 1 FastQueues {a418d8b0-a95b-11ea-bd41-34363bd057d6 true DEMO DEMO2 do the job on channel[DEMO] 0 0} true DemoDemoQueue2.ok 
Worker 0 FastQueues {a418e062-a95b-11ea-bd41-34363bd057d6 true STANDALONE  do the job on STANDALONE 0 0} true StandaloneQueue.ok
Worker 3 FastQueues {a418e918-a95b-11ea-bd41-34363bd057d6 true DEMO DEMO do the job on channel[DEMO::DEMO] 0 0} true DemoDemoQueue.ok
Worker 2 FastQueues {a418f8cc-a95b-11ea-bd41-34363bd057d6 true DEMO DEMO2 do the job on channel[DEMO] 0 0} true DemoDemoQueue2.ok 
Worker 1 FastQueues {a418fd68-a95b-11ea-bd41-34363bd057d6 true STANDALONE  do the job on STANDALONE 0 0} true StandaloneQueue.ok
Worker 0 FastQueues {a419060a-a95b-11ea-bd41-34363bd057d6 true DEMO DEMO do the job on channel[DEMO::DEMO] 0 0} true DemoDemoQueue.ok
Worker 3 FastQueues {a4190e66-a95b-11ea-bd41-34363bd057d6 true DEMO DEMO2 do the job on channel[DEMO] 0 0} true DemoDemoQueue2.ok 
Worker 2 FastQueues {a41914ec-a95b-11ea-bd41-34363bd057d6 true STANDALONE  do the job on STANDALONE 0 0} true StandaloneQueue.ok
```