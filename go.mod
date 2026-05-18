module github.com/jacklau/audio-ai-platform

go 1.25.0

require (
	github.com/aws/aws-sdk-go-v2 v1.30.3
	github.com/aws/aws-sdk-go-v2/config v1.27.27
	github.com/aws/aws-sdk-go-v2/service/s3 v1.58.0
	github.com/eclipse/paho.mqtt.golang v1.4.3
	github.com/golang-jwt/jwt/v4 v4.5.2
	github.com/jacklau/audio-ai-platform/common v0.0.0-00010101000000-000000000000
	github.com/redis/go-redis/v9 v9.16.0
	github.com/segmentio/kafka-go v0.4.47
	go.uber.org/zap v1.27.1
	golang.org/x/crypto v0.49.0
)

require (
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.6.3 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.17.27 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.16.11 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.3.15 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.6.15 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.0 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.3.13 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.11.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.3.15 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.11.17 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.17.13 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.22.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.26.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.30.3 // indirect
	github.com/aws/smithy-go v1.20.3 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/gorilla/websocket v1.5.0 // indirect
	github.com/klauspost/compress v1.17.11 // indirect
	github.com/pierrec/lz4/v4 v4.1.15 // indirect
	github.com/stretchr/testify v1.11.1 // indirect
	github.com/xdg-go/scram v1.2.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/net v0.51.0 // indirect
	golang.org/x/sync v0.16.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
)

// 锁定间接依赖，避免 tidy/get 拉到要求 Go 1.23+ 的版本
replace (
	github.com/jacklau/audio-ai-platform/common => ./common
	github.com/redis/go-redis/v9 => github.com/redis/go-redis/v9 v9.3.0
	golang.org/x/crypto => golang.org/x/crypto v0.17.0
	golang.org/x/net => golang.org/x/net v0.23.0
	golang.org/x/oauth2 => golang.org/x/oauth2 v0.24.0
	golang.org/x/sync => golang.org/x/sync v0.11.0
	golang.org/x/sys => golang.org/x/sys v0.30.0
	golang.org/x/text => golang.org/x/text v0.14.0
	google.golang.org/grpc => google.golang.org/grpc v1.58.3
	google.golang.org/protobuf => google.golang.org/protobuf v1.31.0
)
