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

type DemoDemoQueue struct {
}

func (b DemoDemoQueue) Execute(payload *QueuePayload) *QueueResult {

	fmt.Println(payload)

	return NewQueueResult(true, "ok", nil)
}

func TestNewQueueManager(t *testing.T) {
	// 模拟系统ENV
	os.Setenv("DB_REDIS_BUS_HOST","127.0.0.1")
	os.Setenv("DB_REDIS_BUS_PORT","6379")
	os.Setenv("DB_REDIS_BUS_PASSWORD","")

	host := os.Getenv("DB_REDIS_BUS_HOST")
	port := os.Getenv("DB_REDIS_BUS_PORT")
	password := os.Getenv("DB_REDIS_BUS_PASSWORD")

	// 需要先初始化redis实例
	rds := redis.NewClient(&redis.Options{
		Addr:     host + ":" + port,     // remote host
		Password: password, // password
	})

	// 初始化队列控制器
	qm := NewQueueManager()
	// 注册redis实例到队列控制器
	qm.UseRedis(rds)
	// 队列的worker数
	qm.WorkerNum = 4
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
				Body:   "do the job channel[DEMO::DEMO]",
			})
            
            qm.QueuePush(&QueuePayload{
				IsFast: true,
				Topic:  "DEMO",
				Group:  "",
				Body:   "do the job on channel[DEMO]",
			})
            


		}
	}()
	
    // TOPIC + GROUP 为一个独立的队列通道
	go qm.QueueHandler("DEMO", "DEMO", new(DemoDemoQueue))
    
    // 当不传GROUP参数，TOPIC为独立的一个通道
    go qm.QueueHandler("DEMO", "", new(DemoDemoQueue))
    

	//fmt.Println(qm)
	select {}
}
```

执行效果：

```
=== RUN   TestNewQueueManager
&{7a9a8094-95b3-11ea-9183-34363bd057d6 true DEMO DEMO do the job  0 0}
&{6f4c252a-95e6-11ea-9636-34363bd057d6 true DEMO  do the job on channel[DEMO] 0 0}
Worker 0 FastQueues {6f4c252a-95e6-11ea-9636-34363bd057d6 true DEMO  do the job on channel[DEMO] 0 0} true ok
Worker 3 FastQueues {7a9a8094-95b3-11ea-9183-34363bd057d6 true DEMO DEMO do the job  0 0} true ok
&{7a9a827e-95b3-11ea-9183-34363bd057d6 true DEMO DEMO do the job  0 0}
Worker 2 FastQueues {7a9a827e-95b3-11ea-9183-34363bd057d6 true DEMO DEMO do the job  0 0} true ok
&{6f4c2926-95e6-11ea-9636-34363bd057d6 true DEMO  do the job on channel[DEMO] 0 0}
Worker 1 FastQueues {6f4c2926-95e6-11ea-9636-34363bd057d6 true DEMO  do the job on channel[DEMO] 0 0} true ok
&{7a9a8490-95b3-11ea-9183-34363bd057d6 true DEMO DEMO do the job  0 0}
Worker 3 FastQueues {7a9a8490-95b3-11ea-9183-34363bd057d6 true DEMO DEMO do the job  0 0} true ok
&{6f4c2d04-95e6-11ea-9636-34363bd057d6 true DEMO  do the job on channel[DEMO] 0 0}
Worker 1 FastQueues {6f4c2d04-95e6-11ea-9636-34363bd057d6 true DEMO  do the job on channel[DEMO] 0 0} true ok
&{6f4c30f6-95e6-11ea-9636-34363bd057d6 true DEMO  do the job on channel[DEMO] 0 0}
Worker 2 FastQueues {6f4c30f6-95e6-11ea-9636-34363bd057d6 true DEMO  do the job on channel[DEMO] 0 0} true ok
```