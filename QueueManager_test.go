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
				Body:   "do the job ",
			})
		}
	}()

	qm.QueueHandler("DEMO", "DEMO", new(DemoDemoQueue))
	//fmt.Println(qm)
	select {}
}
