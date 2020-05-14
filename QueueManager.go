package go_redis_queue_manager

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-redis/redis"
	"github.com/google/uuid"
	"runtime"
	"sync"
	"time"
)

type Queueable interface {
	Execute(*QueuePayload) *QueueResult
}
type QueuePayload struct {
	ID       string      `json:"id"`
	IsFast   bool        `json:"is_fast"`
	Topic    string      `json:"topic"`
	Group    string      `json:"group"`
	Body     interface{} `json:"body"`
	MaxRetry int         `json:"max_retry"`
	Retry    int         `json:"retry"`
}
type QueueResult struct {
	State   bool        `json:"state"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}
type QueueRecoveryListener func(stack string)

func NewQueueResult(state bool, msg string, data interface{}) *QueueResult {
	return &QueueResult{State: state, Message: msg, Data: data}
}

var instanceQueueManager *QueueManager
var onceQueueManager sync.Once

type QueueManager struct {
	db             *redis.Client
	MaxRetry       int
	Retry          int
	FastQueues     chan QueuePayload
	FallbackQueues chan QueuePayload
	WorkerNum      int
	OnRecovery     QueueRecoveryListener
}

func NewQueueManager() *QueueManager {
	onceQueueManager.Do(func() {
		instanceQueueManager = &QueueManager{}
		instanceQueueManager.MaxRetry = 3
		instanceQueueManager.Retry = 1
		instanceQueueManager.FastQueues = make(chan QueuePayload, 0)
		instanceQueueManager.FallbackQueues = make(chan QueuePayload, 0)
		instanceQueueManager.WorkerNum = 2
	})
	return instanceQueueManager
}

func (r *QueueManager) RegisterOnInterrupt(listener QueueRecoveryListener) {
	r.OnRecovery = listener
}

func (r *QueueManager) UseRedis(client *redis.Client) {
	r.db = client
}

func (r *QueueManager) GetDb() *redis.Client {
	return r.db
}

func (r *QueueManager) GetQueueName(topic string, group string) string {
	var name string
	if len(group) > 0 {
		name = fmt.Sprintf("Queue_%s::%s", topic, group)
	} else {
		name = fmt.Sprintf("Queue_%s", topic)
	}
	return name
}

func (r *QueueManager) QueueLen(topic string, group string) int64 {
	cmd := r.db.LLen(r.GetQueueName(topic, group))
	length, err := cmd.Result()
	if err != nil {
		return 0
	}
	return length
}

func (r *QueueManager) QueuePop(topic string, group string) (*QueuePayload, error) {
	var payload QueuePayload
	cmd := r.db.LPop(r.GetQueueName(topic, group))
	ret, err := cmd.Bytes()
	if err != nil {
		return nil, cmd.Err()
	}
	err = json.Unmarshal(ret, &payload)
	if err != nil {
		return nil, err
	}
	return &payload, nil
}

func (r *QueueManager) QueuePush(payload *QueuePayload) error {
	if len(payload.Topic) <= 0 {
		return errors.New("TopicId can not be empty")
	}
	id, err := uuid.NewUUID()
	if err != nil {
		return err
	}
	payload.ID = id.String()

	payloadStr, _ := json.Marshal(payload)

	r.db.RPush(r.GetQueueName(payload.Topic, payload.Group), payloadStr)
	return nil
}

func (r *QueueManager) QueueHandler(topic string, group string, i Queueable) {
	// something wrong enqueue again.
	defer func() {
		if err := recover(); err != nil {
			var stacktrace string
			for i := 1; ; i++ {
				_, f, l, got := runtime.Caller(i)
				if !got {
					break

				}
				stacktrace += fmt.Sprintf("%s:%d\n", f, l)
			}
			// when stack finishes
			logMessage := fmt.Sprintf("Trace: %s\n", err)
			logMessage += fmt.Sprintf("\n%s", stacktrace)
			fmt.Println(logMessage)
			if r.OnRecovery != nil {
				r.OnRecovery(stacktrace)
			}
		}
	}()

	go r.RoutinePopToChannel(topic, group)
	for n := 0; n < r.WorkerNum; n++ {
		go r.RoutineWorker(n, i)
	}
}

func (r *QueueManager) RoutineWorker(workerId int, i Queueable) {
	for {
		select {
		case fast := <-r.FastQueues:
			rs := i.Execute(&fast)
			fmt.Println("Worker", workerId, "FastQueues", fast, rs.State, rs.Message)
			if !rs.State {
				r.FallbackQueues <- fast
			}
		case fail := <-r.FallbackQueues:
			if fail.Retry < r.MaxRetry {
				fail.Retry++
				rs := i.Execute(&fail)
				fmt.Println("Worker", workerId, "FallbackQueues", fail, rs.State, rs.Message)
				if !rs.State {
					r.FallbackQueues <- fail
				} else {
					panic(rs.Message)
				}
			}
		}
	}
}

func (r *QueueManager) RoutinePopToChannel(topic string, group string) {
	for {
		if r.QueueLen(topic, group) > 0 {
			payload, err := r.QueuePop(topic, group)
			if err == nil {
				r.FastQueues <- *payload
			}
		}
		time.Sleep(time.Millisecond * 200)
	}

}
