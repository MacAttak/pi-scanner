package detection

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

// detector implements the Detector interface
type detector struct {
	config    *Config
	matchers  []PatternMatcher
	mu        sync.RWMutex
	compiled  map[string]*regexp.Regexp
}

// NewDetector creates a new detector with default configuration
func NewDetector() Detector {
	return NewDetectorWithConfig(DefaultConfig())
}

// NewDetectorWithConfig creates a new detector with custom configuration
func NewDetectorWithConfig(config *Config) Detector {
	d := &detector{
		config:   config,
		matchers: []PatternMatcher{},
		compiled: make(map[string]*regexp.Regexp),
	}
	
	// Initialize pattern matchers
	d.initializeMatchers()
	
	return d
}

// Name returns the detector name
func (d *detector) Name() string {
	return "pattern-detector"
}

// Detect analyzes content and returns findings
func (d *detector) Detect(ctx context.Context, content []byte, filename string) ([]Finding, error) {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	
	// Check file size limit
	if d.config.MaxFileSize > 0 && int64(len(content)) > d.config.MaxFileSize {
		return nil, fmt.Errorf("file too large: %d bytes (max: %d)", len(content), d.config.MaxFileSize)
	}
	
	// Skip excluded paths
	if d.shouldExclude(filename) {
		return nil, nil
	}
	
	findings := []Finding{}
	contentStr := string(content)
	
	// Track which positions have been matched to avoid overlaps
	type matchRange struct {
		start, end int
		piType     PIType
	}
	var matchedRanges []matchRange
	
	// Apply each matcher
	for _, matcher := range d.matchers {
		matches := matcher.Match(content)
		
		for _, match := range matches {
			// Check if this match overlaps with an existing match
			overlaps := false
			for _, existing := range matchedRanges {
				if (match.StartIndex >= existing.start && match.StartIndex < existing.end) ||
				   (match.EndIndex > existing.start && match.EndIndex <= existing.end) {
					overlaps = true
					break
				}
			}
			
			if overlaps {
				continue
			}
			
			// Add to matched ranges
			matchedRanges = append(matchedRanges, matchRange{
				start:  match.StartIndex,
				end:    match.EndIndex,
				piType: matcher.Type(),
			})
			
			// Calculate line and column
			line, column := d.getPosition(contentStr, match.StartIndex)
			
			// Extract context
			contextBefore, contextAfter := d.extractContext(contentStr, match.StartIndex, match.EndIndex)
			
			finding := Finding{
				Type:           matcher.Type(),
				Match:          match.Value,
				File:           filename,
				Line:           line,
				Column:         column,
				Context:        match.Value,
				ContextBefore:  contextBefore,
				ContextAfter:   contextAfter,
				DetectedAt:     time.Now(),
				DetectorName:   d.Name(),
				Confidence:     0.8, // Base confidence for pattern match
				ContextModifier: d.getContextModifier(filename),
			}
			
			// Set initial risk level based on type
			finding.RiskLevel = d.calculateRiskLevel(finding.Type)
			
			findings = append(findings, finding)
		}
	}
	
	// Sort findings by position for consistent ordering
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Line != findings[j].Line {
			return findings[i].Line < findings[j].Line
		}
		return findings[i].Column < findings[j].Column
	})
	
	return findings, nil
}

// initializeMatchers sets up all pattern matchers
func (d *detector) initializeMatchers() {
	// ABN matcher - 11 digits (check first to avoid TFN confusion)
	d.matchers = append(d.matchers, &regexMatcher{
		pattern: `\b\d{2}[\s]?\d{3}[\s]?\d{3}[\s]?\d{3}\b`,
		piType:  PITypeABN,
		d:       d,
	})
	
	// Medicare matcher
	d.matchers = append(d.matchers, &regexMatcher{
		pattern: `\b[2-6]\d{3}[\s\-]?\d{5}[\s\-]?\d{1}(?:/\d)?\b`,
		piType:  PITypeMedicare,
		d:       d,
	})
	
	// TFN matcher - exactly 9 digits (after ABN to avoid confusion)
	d.matchers = append(d.matchers, &regexMatcher{
		pattern: `\b\d{3}[\s\-]?\d{3}[\s\-]?\d{3}\b`,
		piType:  PITypeTFN,
		d:       d,
		validator: func(match string) bool {
			// Remove spaces and dashes
			clean := regexp.MustCompile(`[\s\-]`).ReplaceAllString(match, "")
			// Must be exactly 9 digits and not start with 0
			// Also check it's not part of a longer number
			return len(clean) == 9 && clean[0] != '0'
		},
	})
	
	// BSB matcher - exactly 6 digits
	d.matchers = append(d.matchers, &regexMatcher{
		pattern: `\b\d{3}[\-]?\d{3}\b`,
		piType:  PITypeBSB,
		d:       d,
		validator: func(match string) bool {
			// Remove dashes
			clean := strings.ReplaceAll(match, "-", "")
			// Must be exactly 6 digits
			return len(clean) == 6
		},
	})
	
	// Email matcher
	d.matchers = append(d.matchers, &regexMatcher{
		pattern: `\b[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}\b`,
		piType:  PITypeEmail,
		d:       d,
	})
	
	// Phone matcher (Australian formats) - updated to include landline format
	d.matchers = append(d.matchers, &regexMatcher{
		pattern: `\b(?:\+?61|0)[2-9]\d{8}\b|\(\d{2}\)\s*\d{4}\s*\d{4}`,
		piType:  PITypePhone,
		d:       d,
	})
	
	// Name matcher (simple version - will be enhanced with ML later)
	d.matchers = append(d.matchers, &regexMatcher{
		pattern: `\b[A-Z][a-z]+\s+[A-Z][a-z]+(?:\s+[A-Z][a-z]+)?\b`,
		piType:  PITypeName,
		d:       d,
	})
}

// shouldExclude checks if a file should be excluded from scanning
func (d *detector) shouldExclude(filename string) bool {
	for _, pattern := range d.config.ExcludePaths {
		if matched, _ := filepath.Match(pattern, filename); matched {
			return true
		}
		if strings.Contains(filename, pattern) {
			return true
		}
	}
	return false
}

// getContextModifier returns a risk modifier based on file context
func (d *detector) getContextModifier(filename string) float32 {
	// Check if it's a test file
	for _, pattern := range d.config.TestPathPatterns {
		if matched, _ := filepath.Match(pattern, filename); matched {
			return 0.1
		}
		if strings.Contains(filename, strings.Trim(pattern, "*/")) {
			return 0.1
		}
	}
	
	// Check if it's a mock file
	for _, pattern := range d.config.MockPathPatterns {
		if matched, _ := filepath.Match(pattern, filename); matched {
			return 0.1
		}
		if strings.Contains(filename, strings.Trim(pattern, "*/")) {
			return 0.1
		}
	}
	
	return 1.0
}

// calculateRiskLevel determines risk level based on PI type
func (d *detector) calculateRiskLevel(piType PIType) RiskLevel {
	weight := d.config.RiskWeights[piType]
	
	switch {
	case weight >= 90:
		return RiskLevelHigh
	case weight >= 60:
		return RiskLevelMedium
	default:
		return RiskLevelLow
	}
}

// getPosition calculates line and column from byte index
func (d *detector) getPosition(content string, index int) (line, column int) {
	line = 1
	column = 1
	
	for i := 0; i < index && i < len(content); i++ {
		if content[i] == '\n' {
			line++
			column = 1
		} else {
			column++
		}
	}
	
	return line, column
}

// extractContext extracts surrounding context
func (d *detector) extractContext(content string, start, end int) (before, after string) {
	contextSize := 50
	
	// Extract before context
	beforeStart := start - contextSize
	if beforeStart < 0 {
		beforeStart = 0
	}
	before = content[beforeStart:start]
	
	// Extract after context
	afterEnd := end + contextSize
	if afterEnd > len(content) {
		afterEnd = len(content)
	}
	after = content[end:afterEnd]
	
	return before, after
}

// getRegexp returns a compiled regex, using cache
func (d *detector) getRegexp(pattern string) (*regexp.Regexp, error) {
	d.mu.RLock()
	if re, ok := d.compiled[pattern]; ok {
		d.mu.RUnlock()
		return re, nil
	}
	d.mu.RUnlock()
	
	d.mu.Lock()
	defer d.mu.Unlock()
	
	// Double-check after acquiring write lock
	if re, ok := d.compiled[pattern]; ok {
		return re, nil
	}
	
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	
	d.compiled[pattern] = re
	return re, nil
}

// regexMatcher implements PatternMatcher using regex
type regexMatcher struct {
	pattern   string
	piType    PIType
	d         *detector
	validator func(string) bool
}

// Match finds all pattern matches in content
func (m *regexMatcher) Match(content []byte) []PatternMatch {
	re, err := m.d.getRegexp(m.pattern)
	if err != nil {
		return nil
	}
	
	var matches []PatternMatch
	allMatches := re.FindAllIndex(content, -1)
	
	for _, match := range allMatches {
		if len(match) >= 2 {
			value := string(content[match[0]:match[1]])
			
			// Apply validator if present
			if m.validator != nil && !m.validator(value) {
				continue
			}
			
			matches = append(matches, PatternMatch{
				Value:      value,
				StartIndex: match[0],
				EndIndex:   match[1],
			})
		}
	}
	
	return matches
}

// Type returns the PI type this matcher detects
func (m *regexMatcher) Type() PIType {
	return m.piType
}