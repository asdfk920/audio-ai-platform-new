package kafkax

// Config Kafka/Redpanda 生产者配置。Brokers 为空时表示未启用，Producer 方法为 no-op。
type Config struct {
	Brokers []string
}
