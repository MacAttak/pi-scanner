package ast

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/MacAttak/pi-scanner/pkg/discovery"
)

// BenchmarkAnalyzeFile benchmarks single file analysis
func BenchmarkAnalyzeFile(b *testing.B) {
	analyzer := NewAnalyzer(DefaultBankingConfig())
	ctx := context.Background()

	// Create test files of different sizes
	testCases := []struct {
		name     string
		language Language
		size     int // lines of code
	}{
		{"Small_Java", LanguageJava, 50},
		{"Medium_Java", LanguageJava, 500},
		{"Large_Java", LanguageJava, 2000},
		{"Small_Python", LanguagePython, 50},
		{"Medium_Python", LanguagePython, 500},
		{"Large_Python", LanguagePython, 2000},
	}

	for _, tc := range testCases {
		content := generateTestCode(tc.language, tc.size)
		fileName := fmt.Sprintf("test.%s", getExtension(tc.language))

		b.Run(tc.name, func(b *testing.B) {
			b.SetBytes(int64(len(content)))
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_, err := analyzer.AnalyzeFile(ctx, fileName, content)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkAnalyzeRepository benchmarks repository-wide analysis
func BenchmarkAnalyzeRepository(b *testing.B) {
	analyzer := NewAnalyzer(DefaultBankingConfig())
	ctx := context.Background()

	// Create test repository structures
	testCases := []struct {
		name      string
		fileCount int
	}{
		{"Small_Repo_10_files", 10},
		{"Medium_Repo_100_files", 100},
		{"Large_Repo_500_files", 500},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			// Create temporary directory
			tmpDir := b.TempDir()
			files := createTestRepository(b, tmpDir, tc.fileCount)

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_, err := analyzer.AnalyzeRepository(ctx, tmpDir, files)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkRiskLevelDetermination benchmarks risk level calculation
func BenchmarkRiskLevelDetermination(b *testing.B) {
	analyzer := NewAnalyzer(DefaultBankingConfig())

	testPaths := []string{
		"src/main/java/com/bank/payment/PaymentService.java",
		"src/main/java/com/bank/model/Customer.java",
		"src/test/java/com/bank/PaymentTest.java",
		"src/main/java/com/bank/util/StringHelper.java",
		"build/generated/sources/annotationProcessor/java/main/com/bank/model/Customer_.java",
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		for _, path := range testPaths {
			_ = analyzer.DetermineRiskLevel(path)
			_ = analyzer.DetermineRiskZone(path)
		}
	}
}

// BenchmarkConcurrentAnalysis benchmarks concurrent file analysis
func BenchmarkConcurrentAnalysis(b *testing.B) {
	configs := []struct {
		name       string
		concurrent int
	}{
		{"Sequential", 1},
		{"Concurrent_2", 2},
		{"Concurrent_4", 4},
		{"Concurrent_8", 8},
	}

	for _, cfg := range configs {
		b.Run(cfg.name, func(b *testing.B) {
			config := DefaultBankingConfig()
			config.ConcurrentAnalysis = cfg.concurrent
			analyzer := NewAnalyzer(config)
			ctx := context.Background()

			// Create test repository
			tmpDir := b.TempDir()
			files := createTestRepository(b, tmpDir, 100)

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_, err := analyzer.AnalyzeRepository(ctx, tmpDir, files)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// Helper functions

func generateTestCode(language Language, lines int) []byte {
	var code string

	switch language {
	case LanguageJava:
		code = generateJavaCode(lines)
	case LanguagePython:
		code = generatePythonCode(lines)
	case LanguageScala:
		code = generateScalaCode(lines)
	default:
		code = generateJavaCode(lines)
	}

	return []byte(code)
}

func generateJavaCode(lines int) string {
	template := `package com.example.test;

import java.util.*;

public class TestClass {
    private String field1;
    private int field2;

    public TestClass() {
        this.field1 = "test";
        this.field2 = 42;
    }

    %s
}
`
	methods := ""
	methodLines := 10
	numMethods := (lines - 15) / methodLines

	for i := 0; i < numMethods; i++ {
		methods += fmt.Sprintf(`
    public void method%d() {
        int x = %d;
        String s = "value" + x;
        System.out.println(s);
        for (int i = 0; i < 10; i++) {
            x += i;
        }
    }
`, i, i)
	}

	return fmt.Sprintf(template, methods)
}

func generatePythonCode(lines int) string {
	template := `#!/usr/bin/env python3
import os
import sys

class TestClass:
    def __init__(self):
        self.field1 = "test"
        self.field2 = 42

    %s
`
	methods := ""
	methodLines := 8
	numMethods := (lines - 10) / methodLines

	for i := 0; i < numMethods; i++ {
		methods += fmt.Sprintf(`
    def method%d(self):
        x = %d
        s = f"value{x}"
        print(s)
        for i in range(10):
            x += i
`, i, i)
	}

	return fmt.Sprintf(template, methods)
}

func generateScalaCode(lines int) string {
	template := `package com.example.test

import scala.collection.mutable

class TestClass {
    private var field1: String = "test"
    private var field2: Int = 42

    %s
}
`
	methods := ""
	methodLines := 8
	numMethods := (lines - 10) / methodLines

	for i := 0; i < numMethods; i++ {
		methods += fmt.Sprintf(`
    def method%d(): Unit = {
        var x = %d
        val s = s"value${x}"
        println(s)
        for (i <- 0 until 10) {
            x += i
        }
    }
`, i, i)
	}

	return fmt.Sprintf(template, methods)
}

func getExtension(language Language) string {
	switch language {
	case LanguageJava:
		return "java"
	case LanguagePython:
		return "py"
	case LanguageScala:
		return "scala"
	default:
		return "txt"
	}
}

func createTestRepository(b *testing.B, tmpDir string, fileCount int) []discovery.FileResult {
	var files []discovery.FileResult

	// Create directory structure
	dirs := []string{
		"src/main/java/com/bank/model",
		"src/main/java/com/bank/service",
		"src/main/java/com/bank/payment",
		"src/test/java/com/bank",
		"src/main/resources",
	}

	for _, dir := range dirs {
		fullPath := filepath.Join(tmpDir, dir)
		err := os.MkdirAll(fullPath, 0755)
		if err != nil {
			b.Fatal(err)
		}
	}

	// Distribute files across directories
	filesPerDir := fileCount / len(dirs)
	fileNum := 0

	for _, dir := range dirs {
		for i := 0; i < filesPerDir && fileNum < fileCount; i++ {
			fileName := fmt.Sprintf("TestFile%d.java", fileNum)
			filePath := filepath.Join(tmpDir, dir, fileName)

			content := generateJavaCode(100) // 100 lines per file
			err := os.WriteFile(filePath, []byte(content), 0644)
			if err != nil {
				b.Fatal(err)
			}

			files = append(files, discovery.FileResult{
				Path:     filePath,
				Size:     int64(len(content)),
				IsBinary: false,
				IsHidden: false,
			})

			fileNum++
		}
	}

	return files
}
