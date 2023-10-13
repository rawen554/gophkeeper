package utils

const (
	err int = iota + 1
	warn
	info
	debug
)

func ConvertLogLevelToInt(logLevel string) int {
	switch logLevel {
	case "debug":
		return debug
	case "info":
		return info
	case "warn":
		return warn
	case "error":
		return err
	default:
		return 0
	}
}
