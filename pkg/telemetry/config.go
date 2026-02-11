package telemetry

// В идеале делать под каждый компонент реализовать Config, но для тривиального кейса можно положиться на дефолтные значения

// import "time"

// type Config struct {
// 	ServiceName    string
// 	ServiceVersion string
// 	Environment    string

// 	OTLP OTLPConfig

// 	Logs    LogsConfig
// 	Metrics MetricsConfig
// 	Traces  TracesConfig
// }

// type OTLPConfig struct {
// 	Endpoint string
// 	Insecure bool
// 	Timeout  time.Duration
// }

// type LogsConfig struct {
// 	MaxQueueSize       int
// 	MaxExportBatchSize int
// 	BatchTimeout       time.Duration
// 	ExportTimeout      time.Duration
// }

// type MetricsConfig struct {
// 	ExportInterval time.Duration
// 	ExportTimeout  time.Duration
// }

// type TracesConfig struct {
// 	MaxQueueSize       int
// 	MaxExportBatchSize int
// 	BatchTimeout       time.Duration
// 	ExportTimeout      time.Duration
// 	SampleRatio        float64
// }
