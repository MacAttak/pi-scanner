package detection

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/MacAttak/pi-scanner/pkg/validation"
)

// BenchmarkDetection tests detection performance with various content sizes
func BenchmarkDetection(b *testing.B) {
	benchmarks := []struct {
		name    string
		content string
		size    string
	}{
		{
			name:    "Small_100B",
			content: generateContent(100, 1),
			size:    "100B",
		},
		{
			name:    "Medium_10KB",
			content: generateContent(10*1024, 10),
			size:    "10KB",
		},
		{
			name:    "Large_100KB",
			content: generateContent(100*1024, 100),
			size:    "100KB",
		},
		{
			name:    "XLarge_1MB",
			content: generateContent(1024*1024, 1000),
			size:    "1MB",
		},
	}

	detector := NewDetector()
	ctx := context.Background()

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			content := []byte(bm.content)
			b.SetBytes(int64(len(content)))
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_, err := detector.Detect(ctx, content, "test.go")
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkSpecificPITypes benchmarks detection of individual PI types
func BenchmarkSpecificPITypes(b *testing.B) {
	detector := NewDetector()
	ctx := context.Background()

	piTypes := []struct {
		name    string
		content string
	}{
		{"TFN", "The tax file number is 123456782 for processing"},
		{"Medicare", "Patient medicare: 2123456701 confirmed"},
		{"ABN", "Business ABN: 51 824 753 556"},
		{"BSB", "Bank BSB: 012-345 for transfers"},
		{"Driver_License", "DL: 12345678 (NSW license)"},
	}

	for _, pi := range piTypes {
		b.Run(pi.name, func(b *testing.B) {
			content := []byte(pi.content)
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_, err := detector.Detect(ctx, content, "test.txt")
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkValidation benchmarks checksum validation performance
func BenchmarkValidation(b *testing.B) {
	// Create validators
	tfnValidator := &validation.TFNValidator{}
	abnValidator := &validation.ABNValidator{}
	medicareValidator := &validation.MedicareValidator{}

	b.Run("TFN_Checksum", func(b *testing.B) {
		tfn := "123456782"
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, _ = tfnValidator.Validate(tfn)
		}
	})

	b.Run("ABN_Checksum", func(b *testing.B) {
		abn := "51824753556"
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, _ = abnValidator.Validate(abn)
		}
	})

	b.Run("Medicare_Checksum", func(b *testing.B) {
		medicare := "2123456701"
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, _ = medicareValidator.Validate(medicare)
		}
	})
}

// BenchmarkFalsePositives benchmarks false positive filtering performance
func BenchmarkFalsePositives(b *testing.B) {
	detector := NewDetector()
	ctx := context.Background()

	// Content with many potential false positives
	content := []byte(`
		Order #123456789 processed
		Timestamp: 2024010112345678
		Version: 51.824.753.556
		Test TFN: 123456789
		Mock ABN: 11223344556
		Example medicare: 1234567890
	`)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := detector.Detect(ctx, content, "test.log")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParallelDetection benchmarks concurrent detection
func BenchmarkParallelDetection(b *testing.B) {
	detector := NewDetector()
	ctx := context.Background()
	content := []byte(generateContent(10*1024, 10))

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := detector.Detect(ctx, content, "test.go")
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// Helper function to generate test content
func generateContent(size int, piCount int) string {
	var sb strings.Builder

	// Add some PI data
	pis := []string{
		"TFN: 123456782",
		"Medicare: 2123456701",
		"ABN: 51824753556",
		"BSB: 012-345",
	}

	for i := 0; i < piCount && i < len(pis); i++ {
		sb.WriteString(fmt.Sprintf("Record %d: %s\n", i, pis[i%len(pis)]))
	}

	// Fill the rest with Lorem ipsum
	lorem := "Lorem ipsum dolor sit amet, consectetur adipiscing elit. "
	for sb.Len() < size {
		sb.WriteString(lorem)
	}

	return sb.String()[:size]
}

// BenchmarkRegexVsCustom compares regex pattern matching vs custom detection
func BenchmarkRegexVsCustom(b *testing.B) {
	content := []byte("The TFN is 123456782 and ABN is 51824753556")
	ctx := context.Background()

	b.Run("Regex_Based", func(b *testing.B) {
		detector := NewDetector()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, _ = detector.Detect(ctx, content, "test.txt")
		}
	})

	// Custom detection would go here if implemented
}

// BenchmarkMemoryAllocation tracks memory allocations during detection
func BenchmarkMemoryAllocation(b *testing.B) {
	detector := NewDetector()
	ctx := context.Background()
	content := []byte(generateContent(10*1024, 50))

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = detector.Detect(ctx, content, "test.go")
	}
}
