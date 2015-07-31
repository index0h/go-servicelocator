package servicelocator

type LoggerInterface interface {
	Debug(args ...interface{})
	Error(args ...interface{})
	Fatal(args ...interface{})
	Warn(args ...interface{})
}