package wasmedge

import "testing"

func TestLogLevels(t *testing.T) {
	// The log APIs have no observable state to read back; they must simply
	// not crash for every level.
	for _, level := range []LogLevel{
		LogLevel_Trace,
		LogLevel_Debug,
		LogLevel_Info,
		LogLevel_Warn,
		LogLevel_Error,
		LogLevel_Critical,
	} {
		SetLogLevel(level)
	}
	SetLogDebugLevel()
	SetLogErrorLevel()
	SetLogOff()
}
