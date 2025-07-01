package detection

import (
	"context"
	"testing"

	"github.com/MacAttak/pi-scanner/pkg/detection"
	"github.com/MacAttak/pi-scanner/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMultiLanguageTestFileDetection verifies test file detection across languages
func TestMultiLanguageTestFileDetection(t *testing.T) {
	// Create detectors
	baseDetector := detection.NewDetector()
	contextDetector := testutil.NewMockContextDetector(baseDetector)

	testCases := []struct {
		name           string
		filename       string
		code           string
		shouldSuppress bool
	}{
		// Go test files
		{
			name:           "Go test file with _test.go",
			filename:       "user_test.go",
			code:           `func TestUser(t *testing.T) { tfn := "123456782" }`,
			shouldSuppress: true,
		},
		{
			name:           "Go test file in test directory",
			filename:       "test/integration/api_test.go",
			code:           `const testTFN = "123456782"`,
			shouldSuppress: true,
		},

		// Java test files
		{
			name:           "Java test class",
			filename:       "UserServiceTest.java",
			code:           `public class UserServiceTest { String tfn = "123456782"; }`,
			shouldSuppress: true,
		},
		{
			name:           "Java test in src/test",
			filename:       "src/test/java/com/example/ValidationTest.java",
			code:           `private static final String TEST_TFN = "123456782";`,
			shouldSuppress: true,
		},

		// Python test files
		{
			name:           "Python test_ prefix",
			filename:       "test_validation.py",
			code:           `def test_tfn(): tfn = "123456782"`,
			shouldSuppress: true,
		},
		{
			name:           "Python _test suffix",
			filename:       "tfn_validation_test.py",
			code:           `TEST_TFN = "123456782"`,
			shouldSuppress: true,
		},
		{
			name:           "Python conftest",
			filename:       "conftest.py",
			code:           `@pytest.fixture\ndef valid_tfn(): return "123456782"`,
			shouldSuppress: true,
		},

		// JavaScript/TypeScript test files
		{
			name:           "JavaScript test file",
			filename:       "user.test.js",
			code:           `const testTFN = "123456782";`,
			shouldSuppress: true,
		},
		{
			name:           "TypeScript spec file",
			filename:       "validation.spec.ts",
			code:           `describe('validation', () => { const tfn = "123456782"; });`,
			shouldSuppress: true,
		},
		{
			name:           "Test in __tests__ directory",
			filename:       "__tests__/tfn-validator.js",
			code:           `test('validates TFN', () => { expect(validateTFN("123456782")).toBe(true); });`,
			shouldSuppress: true,
		},

		// Scala test files
		{
			name:           "Scala test class",
			filename:       "UserServiceTest.scala",
			code:           `class UserServiceTest { val testTFN = "123456782" }`,
			shouldSuppress: true,
		},
		{
			name:           "Scala spec file",
			filename:       "ValidationSpec.scala",
			code:           `class ValidationSpec extends FlatSpec { val tfn = "123456782" }`,
			shouldSuppress: true,
		},

		// Non-test files (should NOT suppress)
		{
			name:           "Production Go file",
			filename:       "user.go",
			code:           `type User struct { TFN string }; u.TFN = "123456782"`,
			shouldSuppress: false,
		},
		{
			name:           "Production Java file",
			filename:       "UserService.java",
			code:           `public class UserService { private String tfn = "123456782"; }`,
			shouldSuppress: false,
		},
		{
			name:           "Production Python file",
			filename:       "user_manager.py",
			code:           `class User: tfn = "123456782"`,
			shouldSuppress: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			// Get detections from base detector
			baseFindings, err := baseDetector.Detect(ctx, []byte(tc.code), tc.filename)
			require.NoError(t, err)

			// Get detections from context-aware detector
			contextFindings, err := contextDetector.Detect(ctx, []byte(tc.code), tc.filename)
			require.NoError(t, err)

			// Check if TFN was detected in base
			baseTFNFound := false
			for _, f := range baseFindings {
				if f.Type == detection.PITypeTFN {
					baseTFNFound = true
					break
				}
			}

			// Check if TFN was detected in context-aware
			contextTFNFound := false
			for _, f := range contextFindings {
				if f.Type == detection.PITypeTFN {
					contextTFNFound = true
					break
				}
			}

			if tc.shouldSuppress {
				assert.True(t, baseTFNFound, "Base detector should find TFN")
				assert.False(t, contextTFNFound, "Context detector should suppress TFN in test file: %s", tc.filename)
			} else {
				assert.True(t, baseTFNFound, "Base detector should find TFN")
				assert.True(t, contextTFNFound, "Context detector should not suppress TFN in production file: %s", tc.filename)
			}
		})
	}
}

// TestContextSuppressionRate validates the overall suppression rate
func TestContextSuppressionRate(t *testing.T) {
	// Generate test data with known counts
	testData := []struct {
		filename string
		code     string
		isTest   bool
	}{
		// Test files with PI (should be suppressed)
		{"user_test.go", `func TestTFN(t *testing.T) { tfn := "123456782" }`, true},
		{"ValidationTest.java", `public class ValidationTest { String tfn = "123456782"; }`, true},
		{"test_tfn.py", `def test_tfn(): return "123456782"`, true},
		{"tfn.spec.ts", `describe('tfn', () => { const tfn = "123456782"; });`, true},
		{"TFNSpec.scala", `class TFNSpec { val tfn = "123456782" }`, true},

		// Production files with PI (should NOT be suppressed)
		{"user.go", `type User struct { TFN string }; u.TFN = "123456782"`, false},
		{"User.java", `public class User { private String tfn = "123456782"; }`, false},
		{"user.py", `class User: tfn = "123456782"`, false},
	}

	baseDetector := detection.NewDetector()
	contextDetector := testutil.NewMockContextDetector(baseDetector)
	ctx := context.Background()

	testFilesSuppressed := 0
	testFilesTotal := 0

	for _, td := range testData {
		baseFindings, _ := baseDetector.Detect(ctx, []byte(td.code), td.filename)
		contextFindings, _ := contextDetector.Detect(ctx, []byte(td.code), td.filename)

		baseTFNFound := false
		for _, f := range baseFindings {
			if f.Type == detection.PITypeTFN {
				baseTFNFound = true
				break
			}
		}

		contextTFNFound := false
		for _, f := range contextFindings {
			if f.Type == detection.PITypeTFN {
				contextTFNFound = true
				break
			}
		}

		if td.isTest && baseTFNFound {
			testFilesTotal++
			if !contextTFNFound {
				testFilesSuppressed++
			}
		}
	}

	suppressionRate := float64(testFilesSuppressed) / float64(testFilesTotal)
	t.Logf("Test file suppression: %d/%d = %.1f%%", testFilesSuppressed, testFilesTotal, suppressionRate*100)

	assert.GreaterOrEqual(t, suppressionRate, 0.7, "Should suppress at least 70% of test file detections")
}
