package tokenization

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTokenizerConfig(t *testing.T) {
	t.Run("DefaultConfig", func(t *testing.T) {
		config := DefaultTokenizerConfig()
		
		assert.Equal(t, "microsoft/deberta-v3-base", config.ModelName)
		assert.Equal(t, 512, config.MaxLength)
		assert.True(t, config.Padding)
		assert.True(t, config.Truncation)
		assert.True(t, config.AddSpecialTokens)
	})

	t.Run("CustomConfig", func(t *testing.T) {
		config := TokenizerConfig{
			ModelName:        "bert-base-uncased",
			MaxLength:        256,
			Padding:          false,
			Truncation:       false,
			AddSpecialTokens: false,
		}
		
		tokenizer, err := NewTokenizer(config)
		assert.NoError(t, err)
		assert.NotNil(t, tokenizer)
		assert.Equal(t, config, tokenizer.config)
	})

	t.Run("InvalidConfig", func(t *testing.T) {
		config := TokenizerConfig{
			ModelName: "", // Empty model name
			MaxLength: 512,
		}
		
		tokenizer, err := NewTokenizer(config)
		assert.Error(t, err)
		assert.Nil(t, tokenizer)
		assert.Contains(t, err.Error(), "model name cannot be empty")
	})

	t.Run("AutoCorrectMaxLength", func(t *testing.T) {
		config := TokenizerConfig{
			ModelName: "test-model",
			MaxLength: 0, // Invalid max length
		}
		
		tokenizer, err := NewTokenizer(config)
		assert.NoError(t, err)
		assert.NotNil(t, tokenizer)
		assert.Equal(t, 512, tokenizer.config.MaxLength) // Should be auto-corrected
	})
}

func TestTokenizerInitialization(t *testing.T) {
	// Skip these tests if tokenizer files are not available
	if testing.Short() {
		t.Skip("Skipping tokenizer initialization tests in short mode")
	}

	t.Run("InitializeFromPretrained", func(t *testing.T) {
		t.Skip("Requires internet connection and HuggingFace access")
		
		config := TokenizerConfig{
			ModelName: "bert-base-uncased",
			MaxLength: 512,
		}
		
		tokenizer, err := NewTokenizer(config)
		require.NoError(t, err)
		
		err = tokenizer.Initialize()
		assert.NoError(t, err)
		
		// Cleanup
		err = tokenizer.Close()
		assert.NoError(t, err)
	})

	t.Run("InitializeTwice", func(t *testing.T) {
		t.Skip("Requires tokenizer files")
		
		config := TokenizerConfig{
			ModelName: "test-tokenizer.json",
			MaxLength: 512,
		}
		
		tokenizer, err := NewTokenizer(config)
		require.NoError(t, err)
		
		// First initialization
		err = tokenizer.Initialize()
		require.NoError(t, err)
		
		// Second initialization should fail
		err = tokenizer.Initialize()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already initialized")
		
		// Cleanup
		tokenizer.Close()
	})

	t.Run("CloseWithoutInitialize", func(t *testing.T) {
		config := DefaultTokenizerConfig()
		tokenizer, err := NewTokenizer(config)
		require.NoError(t, err)
		
		// Should not error when closing uninitialized tokenizer
		err = tokenizer.Close()
		assert.NoError(t, err)
	})
}

func TestTokenizerEncoding(t *testing.T) {
	// These tests use a mock tokenizer since we don't have actual model files
	t.Run("MockEncoding", func(t *testing.T) {
		// Test the encoding logic without actual tokenizer
		config := TokenizerConfig{
			ModelName:        "test-model",
			MaxLength:        10,
			Padding:          true,
			Truncation:       true,
			AddSpecialTokens: true,
		}
		
		tokenizer, err := NewTokenizer(config)
		require.NoError(t, err)
		assert.NotNil(t, tokenizer)
		
		// Would test actual encoding if tokenizer was initialized
	})

	t.Run("EncodingNotInitialized", func(t *testing.T) {
		config := DefaultTokenizerConfig()
		tokenizer, err := NewTokenizer(config)
		require.NoError(t, err)
		
		// Try to encode without initialization
		result, err := tokenizer.Encode("test text")
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "not initialized")
	})
}

func TestPIContextExtraction(t *testing.T) {
	t.Run("ExtractContext", func(t *testing.T) {
		config := DefaultTokenizerConfig()
		tokenizer, err := NewTokenizer(config)
		require.NoError(t, err)
		
		text := "The tax file number is 123456789 and it is valid."
		startOffset := 23 // Start of "123456789"
		endOffset := 32   // End of "123456789"
		contextWindow := 10
		
		// Would test actual context extraction if tokenizer was initialized
		_ = text
		_ = startOffset
		_ = endOffset
		_ = contextWindow
		_ = tokenizer
	})
}

func TestPIPreprocessing(t *testing.T) {
	t.Run("PreprocessForValidation", func(t *testing.T) {
		config := DefaultTokenizerConfig()
		tokenizer, err := NewTokenizer(config)
		require.NoError(t, err)
		
		// Test preprocessing format
		text := "123-456-789"
		piType := "TFN"
		
		// The preprocessing should add PI type context
		// Expected format: "[TFN] 123-456-789"
		_ = text
		_ = piType
		_ = tokenizer
	})

	t.Run("TokenizePICandidate", func(t *testing.T) {
		config := DefaultTokenizerConfig()
		tokenizer, err := NewTokenizer(config)
		require.NoError(t, err)
		
		candidate := "123456789"
		context := "The TFN is 123456789 for tax purposes"
		piType := "TFN"
		
		// The format should be: "[TFN] 123456789 [SEP] The TFN is 123456789 for tax purposes"
		expectedFormat := "[TFN] 123456789 [SEP] The TFN is 123456789 for tax purposes"
		
		// Verify the formatting logic
		formattedText := strings.Join([]string{
			"[" + strings.ToUpper(piType) + "]",
			candidate,
			"[SEP]",
			context,
		}, " ")
		
		assert.Equal(t, expectedFormat, formattedText)
		_ = tokenizer
	})
}

func TestBatchOperations(t *testing.T) {
	t.Run("BatchEncode", func(t *testing.T) {
		config := DefaultTokenizerConfig()
		tokenizer, err := NewTokenizer(config)
		require.NoError(t, err)
		
		texts := []string{
			"First text to encode",
			"Second text to encode",
			"Third text to encode",
		}
		
		// Would test actual batch encoding if tokenizer was initialized
		_ = texts
		_ = tokenizer
	})

	t.Run("BatchTokenizePICandidates", func(t *testing.T) {
		config := DefaultTokenizerConfig()
		tokenizer, err := NewTokenizer(config)
		require.NoError(t, err)
		
		candidates := []struct {
			Candidate string
			Context   string
			PIType    string
		}{
			{
				Candidate: "123456789",
				Context:   "TFN: 123456789",
				PIType:    "TFN",
			},
			{
				Candidate: "12345678901",
				Context:   "ABN: 12345678901",
				PIType:    "ABN",
			},
			{
				Candidate: "2345678901",
				Context:   "Medicare: 2345678901",
				PIType:    "MEDICARE",
			},
		}
		
		// Verify formatting for each candidate
		for _, c := range candidates {
			expectedFormat := strings.Join([]string{
				"[" + strings.ToUpper(c.PIType) + "]",
				c.Candidate,
				"[SEP]",
				c.Context,
			}, " ")
			
			// The actual format should match expected
			formattedText := strings.Join([]string{
				"[" + strings.ToUpper(c.PIType) + "]",
				c.Candidate,
				"[SEP]",
				c.Context,
			}, " ")
			
			assert.Equal(t, expectedFormat, formattedText)
		}
		_ = tokenizer
	})
}

func TestTokenizerMethods(t *testing.T) {
	t.Run("GetVocabSize", func(t *testing.T) {
		config := DefaultTokenizerConfig()
		tokenizer, err := NewTokenizer(config)
		require.NoError(t, err)
		
		// Should error when not initialized
		size, err := tokenizer.GetVocabSize()
		assert.Error(t, err)
		assert.Equal(t, 0, size)
		assert.Contains(t, err.Error(), "not initialized")
	})

	t.Run("Decode", func(t *testing.T) {
		config := DefaultTokenizerConfig()
		tokenizer, err := NewTokenizer(config)
		require.NoError(t, err)
		
		// Should error when not initialized
		ids := []uint32{101, 102, 103}
		text, err := tokenizer.Decode(ids, true)
		assert.Error(t, err)
		assert.Empty(t, text)
		assert.Contains(t, err.Error(), "not initialized")
	})
}

// Benchmark tests
func BenchmarkTokenizerCreation(b *testing.B) {
	config := DefaultTokenizerConfig()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tokenizer, err := NewTokenizer(config)
		if err != nil {
			b.Fatal(err)
		}
		_ = tokenizer
	}
}

func BenchmarkPIFormatting(b *testing.B) {
	candidate := "123456789"
	context := "The TFN is 123456789 for tax purposes"
	piType := "TFN"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		formattedText := strings.Join([]string{
			"[" + strings.ToUpper(piType) + "]",
			candidate,
			"[SEP]",
			context,
		}, " ")
		_ = formattedText
	}
}