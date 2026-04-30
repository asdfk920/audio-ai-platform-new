// Package statuspersist MQTT/Redis 过期等路径的异步落库（通道 + worker，不阻塞消费）。
package statuspersist

import (
	"context"
	"database/sql"
	"time"

	"github.com/jacklau/audio-ai-platform/services/device/internal/repo"
	"github.com/zeromicro/go-zero/core/logx"
)

// Job 单条异步落库任务。
type Job struct {
	DeviceID        int64
	SN              string
	RunState        string
	Battery         int32
	FirmwareVersion string
	OnlineStatus    int16
	IP              string
	Source          string
}

// Pool 异步写 device_status + 更新 device 在线字段。
type Pool struct {
	db      *sql.DB
	ch      chan Job
	workers int
	log     logx.Logger
}

// NewPool queueSize/workers 非法时使用默认值。
func NewPool(db *sql.DB, queueSize, workers int) *Pool {
	if queueSize <= 0 {
		queueSize = 4096
	}
	if workers <= 0 {
		workers = 4
	}
	return &Pool{
		db: db, ch: make(chan Job, queueSize), workers: workers,
		log: logx.WithContext(context.Background()),
	}
}

// Start 启动 worker；ctx 取消时停止消费。
func (p *Pool) Start(ctx context.Context) {
	if p == nil || p.db == nil {
		return
	}
	w := p.workers
	if w <= 0 {
		w = 4
	}
	for i := 0; i < w; i++ {
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case job, ok := <-p.ch:
					if !ok {
						return
					}
					p.run(job)
				}
			}
		}()
	}
}

func (p *Pool) run(job Job) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	now := time.Now()
	if err := repo.InsertDeviceStatusRow(ctx, p.db, job.DeviceID, job.SN, job.RunState, job.Battery, job.FirmwareVersion, job.OnlineStatus, now, job.Source); err != nil {
		p.log.Errorf("InsertDeviceStatusRow sn=%s: %v", job.SN, err)
	}
	if err := repo.UpdateDeviceOnlineMeta(ctx, p.db, job.DeviceID, job.OnlineStatus, now, job.FirmwareVersion, job.IP); err != nil {
		p.log.Errorf("UpdateDeviceOnlineMeta id=%d: %v", job.DeviceID, err)
	}
}

// Offer 非阻塞投递；队列满则丢弃并打日志。
func (p *Pool) Offer(job Job) {
	if p == nil || p.db == nil {
		return
	}
	select {
	case p.ch <- job:
	default:
		p.log.Errorf("statuspersist queue full, drop sn=%s", job.SN)
	}
}
