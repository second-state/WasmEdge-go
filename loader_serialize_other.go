//go:build !darwin || !arm64

package wasmedge

func loaderSerializePlatformError() error {
	return nil
}
