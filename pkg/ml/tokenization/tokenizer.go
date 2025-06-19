package tokenization

import (
	"fmt"
	"strings"
	"sync"

	"github.com/daulet/tokenizers"
)

// Tokenizer wraps the HuggingFace tokenizer for text preprocessing
type Tokenizer struct {
	tokenizer *tokenizers.Tokenizer
	config    TokenizerConfig
	mu        sync.RWMutex
}

// TokenizerConfig holds configuration for the tokenizer
type TokenizerConfig struct {
	ModelName      string `json:"model_name"`      // HuggingFace model name or path
	MaxLength      int    `json:"max_length"`      // Maximum sequence length
	Padding        bool   `json:"padding"`         // Whether to pad sequences
	Truncation     bool   `json:"truncation"`      // Whether to truncate sequences
	AddSpecialTokens bool `json:"add_special_tokens"` // Whether to add special tokens
}

// EncodingResult represents the tokenization output
type EncodingResult struct {
	IDs               []uint32   `json:"ids"`                 // Token IDs
	TypeIDs           []uint32   `json:"type_ids"`            // Token type IDs
	Tokens            []string   `json:"tokens"`              // Decoded tokens
	AttentionMask     []uint32   `json:"attention_mask"`      // Attention mask
	SpecialTokensMask []uint32   `json:"special_tokens_mask"` // Special tokens mask
	Offsets           []tokenizers.Offset `json:"offsets"`     // Token offsets in original text
	Length            int        `json:"length"`              // Actual sequence length
}

// DefaultTokenizerConfig returns default configuration for DeBERTa models
func DefaultTokenizerConfig() TokenizerConfig {
	return TokenizerConfig{
		ModelName:        "microsoft/deberta-v3-base",
		MaxLength:        512,
		Padding:          true,
		Truncation:       true,
		AddSpecialTokens: true,
	}
}

// NewTokenizer creates a new tokenizer instance
func NewTokenizer(config TokenizerConfig) (*Tokenizer, error) {
	if config.ModelName == "" {
		return nil, fmt.Errorf("model name cannot be empty")
	}

	if config.MaxLength <= 0 {
		config.MaxLength = 512
	}

	return &Tokenizer{
		config: config,
	}, nil
}

// Initialize loads the tokenizer model
func (t *Tokenizer) Initialize() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.tokenizer != nil {
		return fmt.Errorf("tokenizer already initialized")
	}

	// Try to load from HuggingFace model hub
	tk, err := tokenizers.FromPretrained(t.config.ModelName)
	if err != nil {
		// If that fails, try loading from a local file path
		tk, err = tokenizers.FromFile(t.config.ModelName)
		if err != nil {
			return fmt.Errorf("failed to load tokenizer: %w", err)
		}
	}

	t.tokenizer = tk
	return nil
}

// Close releases the tokenizer resources
func (t *Tokenizer) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.tokenizer != nil {
		err := t.tokenizer.Close()
		t.tokenizer = nil
		return err
	}
	return nil
}

// Encode tokenizes a single text string
func (t *Tokenizer) Encode(text string) (*EncodingResult, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.tokenizer == nil {
		return nil, fmt.Errorf("tokenizer not initialized")
	}

	// Prepare encoding options
	options := []tokenizers.EncodeOption{
		tokenizers.WithReturnAllAttributes(), // Get all attributes
	}

	// Encode the text
	encoding := t.tokenizer.EncodeWithOptions(text, t.config.AddSpecialTokens, options...)

	// Apply truncation if needed
	ids := encoding.IDs
	typeIDs := encoding.TypeIDs
	tokens := encoding.Tokens
	attentionMask := encoding.AttentionMask
	specialTokensMask := encoding.SpecialTokensMask
	offsets := encoding.Offsets

	if t.config.Truncation && len(ids) > t.config.MaxLength {
		ids = ids[:t.config.MaxLength]
		typeIDs = typeIDs[:t.config.MaxLength]
		tokens = tokens[:t.config.MaxLength]
		attentionMask = attentionMask[:t.config.MaxLength]
		specialTokensMask = specialTokensMask[:t.config.MaxLength]
		offsets = offsets[:t.config.MaxLength]
	}

	// Apply padding if needed
	if t.config.Padding && len(ids) < t.config.MaxLength {
		padLength := t.config.MaxLength - len(ids)
		
		// Pad with zeros (assuming 0 is the padding token ID)
		for i := 0; i < padLength; i++ {
			ids = append(ids, 0)
			typeIDs = append(typeIDs, 0)
			tokens = append(tokens, "[PAD]")
			attentionMask = append(attentionMask, 0)
			specialTokensMask = append(specialTokensMask, 1)
			offsets = append(offsets, tokenizers.Offset{0, 0})
		}
	}

	return &EncodingResult{
		IDs:               ids,
		TypeIDs:           typeIDs,
		Tokens:            tokens,
		AttentionMask:     attentionMask,
		SpecialTokensMask: specialTokensMask,
		Offsets:           offsets,
		Length:            len(encoding.IDs), // Original length before padding
	}, nil
}

// EncodeBatch tokenizes multiple texts
func (t *Tokenizer) EncodeBatch(texts []string) ([]*EncodingResult, error) {
	results := make([]*EncodingResult, len(texts))
	
	for i, text := range texts {
		result, err := t.Encode(text)
		if err != nil {
			return nil, fmt.Errorf("failed to encode text at index %d: %w", i, err)
		}
		results[i] = result
	}
	
	return results, nil
}

// Decode converts token IDs back to text
func (t *Tokenizer) Decode(ids []uint32, skipSpecialTokens bool) (string, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.tokenizer == nil {
		return "", fmt.Errorf("tokenizer not initialized")
	}

	return t.tokenizer.Decode(ids, skipSpecialTokens), nil
}

// ExtractPIContext extracts context around potential PI for better validation
func (t *Tokenizer) ExtractPIContext(text string, startOffset, endOffset int, contextWindow int) (*EncodingResult, error) {
	// Calculate context boundaries
	contextStart := startOffset - contextWindow
	if contextStart < 0 {
		contextStart = 0
	}
	
	contextEnd := endOffset + contextWindow
	if contextEnd > len(text) {
		contextEnd = len(text)
	}
	
	// Extract context text
	contextText := text[contextStart:contextEnd]
	
	// Encode the context
	result, err := t.Encode(contextText)
	if err != nil {
		return nil, fmt.Errorf("failed to encode context: %w", err)
	}
	
	// Find the PI tokens within the context
	// This helps the model focus on the relevant part
	piStart := startOffset - contextStart
	piEnd := endOffset - contextStart
	
	// Mark which tokens correspond to the PI
	for _, offset := range result.Offsets {
		tokenStart := int(offset[0])
		tokenEnd := int(offset[1])
		
		// Check if this token overlaps with the PI
		if tokenStart < piEnd && tokenEnd > piStart {
			// This token is part of or contains the PI
			// We could add a special marker or adjust attention
		}
	}
	
	return result, nil
}

// PreprocessForPIValidation prepares text for PI validation
func (t *Tokenizer) PreprocessForPIValidation(text string, piType string) (*EncodingResult, error) {
	// Add PI type context to help the model
	processedText := fmt.Sprintf("[%s] %s", piType, text)
	
	return t.Encode(processedText)
}

// GetVocabSize returns the vocabulary size
func (t *Tokenizer) GetVocabSize() (int, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.tokenizer == nil {
		return 0, fmt.Errorf("tokenizer not initialized")
	}

	return int(t.tokenizer.VocabSize()), nil
}

// TokenizePICandidate prepares a PI candidate with context for validation
func (t *Tokenizer) TokenizePICandidate(candidate string, context string, piType string) (*EncodingResult, error) {
	// Format: [CLS] [PI_TYPE] candidate [SEP] context [SEP]
	// This format helps the model understand what to validate
	
	formattedText := fmt.Sprintf("[%s] %s [SEP] %s", 
		strings.ToUpper(piType), 
		candidate, 
		context)
	
	return t.Encode(formattedText)
}

// BatchTokenizePICandidates tokenizes multiple PI candidates efficiently
func (t *Tokenizer) BatchTokenizePICandidates(candidates []struct {
	Candidate string
	Context   string
	PIType    string
}) ([]*EncodingResult, error) {
	texts := make([]string, len(candidates))
	
	for i, c := range candidates {
		texts[i] = fmt.Sprintf("[%s] %s [SEP] %s", 
			strings.ToUpper(c.PIType), 
			c.Candidate, 
			c.Context)
	}
	
	return t.EncodeBatch(texts)
}