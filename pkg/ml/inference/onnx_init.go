package inference

import (
	"fmt"
	"os"
	"runtime"

	ort "github.com/yalue/onnxruntime_go"
)

// InitializeONNXRuntime sets up the ONNX Runtime library path based on platform
func InitializeONNXRuntime() error {
	var libraryPath string

	switch runtime.GOOS {
	case "darwin":
		// macOS - check common installation paths
		paths := []string{
			"/opt/homebrew/lib/libonnxruntime.dylib", // Homebrew on Apple Silicon
			"/usr/local/lib/libonnxruntime.dylib",    // Homebrew on Intel Mac
			"/usr/lib/libonnxruntime.dylib",          // System location
		}
		
		for _, path := range paths {
			if _, err := os.Stat(path); err == nil {
				libraryPath = path
				break
			}
		}
		
		if libraryPath == "" {
			return fmt.Errorf("ONNX Runtime library not found. Please install it using: brew install onnxruntime")
		}

	case "linux":
		// Linux - check common installation paths
		paths := []string{
			"/usr/local/lib/libonnxruntime.so",
			"/usr/lib/libonnxruntime.so",
			"/usr/lib/x86_64-linux-gnu/libonnxruntime.so",
			"/usr/lib/aarch64-linux-gnu/libonnxruntime.so",
		}
		
		for _, path := range paths {
			if _, err := os.Stat(path); err == nil {
				libraryPath = path
				break
			}
		}
		
		if libraryPath == "" {
			return fmt.Errorf("ONNX Runtime library not found. Please install it from https://github.com/microsoft/onnxruntime/releases")
		}

	case "windows":
		// Windows - check common installation paths
		paths := []string{
			"C:\\Program Files\\onnxruntime\\lib\\onnxruntime.dll",
			"onnxruntime.dll", // Current directory
		}
		
		for _, path := range paths {
			if _, err := os.Stat(path); err == nil {
				libraryPath = path
				break
			}
		}
		
		if libraryPath == "" {
			return fmt.Errorf("ONNX Runtime library not found. Please download it from https://github.com/microsoft/onnxruntime/releases")
		}

	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	// Set the library path
	ort.SetSharedLibraryPath(libraryPath)
	return nil
}