package commands

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewFileCache(t *testing.T) {
	cache := NewFileCache(5*time.Minute, 100)
	
	if cache == nil {
		t.Fatal("NewFileCache returned nil")
	}
	if cache.maxAge != 5*time.Minute {
		t.Errorf("maxAge should be 5 minutes")
	}
	if cache.maxSize != 100 {
		t.Errorf("maxSize should be 100")
	}
}

func TestFileCacheGetSet(t *testing.T) {
	cache := NewFileCache(5*time.Minute, 100)
	
	files := []CachedFileInfo{
		{Name: "test.md", Path: "/test/test.md", IsDir: false},
	}
	
	// Should not exist initially
	_, ok := cache.Get("/test")
	if ok {
		t.Error("Cache should be empty initially")
	}
	
	// Set and get
	cache.Set("/test", files)
	entry, ok := cache.Get("/test")
	
	if !ok {
		t.Error("Cache entry should exist after Set")
	}
	if len(entry.Files) != 1 {
		t.Error("Cache entry should have 1 file")
	}
}

func TestFileCacheExpiration(t *testing.T) {
	// Create cache with very short expiration
	cache := NewFileCache(1*time.Millisecond, 100)
	
	files := []CachedFileInfo{{Name: "test.md"}}
	cache.Set("/test", files)
	
	// Wait for expiration
	time.Sleep(5 * time.Millisecond)
	
	// Should be expired
	_, ok := cache.Get("/test")
	if ok {
		t.Error("Cache entry should be expired")
	}
}

func TestFileCacheInvalidate(t *testing.T) {
	cache := NewFileCache(5*time.Minute, 100)
	
	files := []CachedFileInfo{{Name: "test.md"}}
	cache.Set("/test", files)
	
	// Invalidate
	cache.Invalidate("/test")
	
	_, ok := cache.Get("/test")
	if ok {
		t.Error("Cache entry should be invalidated")
	}
}

func TestFileCacheInvalidateAll(t *testing.T) {
	cache := NewFileCache(5*time.Minute, 100)
	
	cache.Set("/test1", []CachedFileInfo{{Name: "a.md"}})
	cache.Set("/test2", []CachedFileInfo{{Name: "b.md"}})
	
	cache.InvalidateAll()
	
	_, ok1 := cache.Get("/test1")
	_, ok2 := cache.Get("/test2")
	
	if ok1 || ok2 {
		t.Error("All cache entries should be invalidated")
	}
}

func TestFileCacheEviction(t *testing.T) {
	// Cache with max size 2
	cache := NewFileCache(5*time.Minute, 2)
	
	cache.Set("/test1", []CachedFileInfo{{Name: "a.md"}})
	time.Sleep(1 * time.Millisecond)
	cache.Set("/test2", []CachedFileInfo{{Name: "b.md"}})
	time.Sleep(1 * time.Millisecond)
	cache.Set("/test3", []CachedFileInfo{{Name: "c.md"}})
	
	// Oldest entry should be evicted
	_, ok1 := cache.Get("/test1")
	_, ok3 := cache.Get("/test3")
	
	if ok1 {
		t.Error("Oldest entry should be evicted")
	}
	if !ok3 {
		t.Error("Newest entry should exist")
	}
}

func TestLazyDirReaderWithExtensions(t *testing.T) {
	reader := NewLazyDirReader().WithExtensions(".md", ".txt")
	
	if len(reader.extensions) != 2 {
		t.Errorf("Expected 2 extensions, got %d", len(reader.extensions))
	}
}

func TestLazyDirReaderWithExcludeDirs(t *testing.T) {
	reader := NewLazyDirReader().WithExcludeDirs("vendor")
	
	// Should have default excludes plus new one
	found := false
	for _, d := range reader.excludeDirs {
		if d == "vendor" {
			found = true
			break
		}
	}
	if !found {
		t.Error("vendor should be in excludeDirs")
	}
}

func TestLazyDirReaderReadDir(t *testing.T) {
	// Create temp directory with files
	tempDir := t.TempDir()
	
	// Create test files
	os.WriteFile(filepath.Join(tempDir, "test.md"), []byte("# Test"), 0644)
	os.WriteFile(filepath.Join(tempDir, "test.txt"), []byte("text"), 0644)
	os.Mkdir(filepath.Join(tempDir, "subdir"), 0755)
	
	reader := NewLazyDirReader()
	files, err := reader.ReadDir(tempDir)
	
	if err != nil {
		t.Fatalf("ReadDir failed: %v", err)
	}
	
	if len(files) != 3 {
		t.Errorf("Expected 3 entries, got %d", len(files))
	}
}

func TestLazyDirReaderFilterByExtension(t *testing.T) {
	tempDir := t.TempDir()
	
	os.WriteFile(filepath.Join(tempDir, "test.md"), []byte("# Test"), 0644)
	os.WriteFile(filepath.Join(tempDir, "test.txt"), []byte("text"), 0644)
	os.WriteFile(filepath.Join(tempDir, "test.go"), []byte("package main"), 0644)
	
	reader := NewLazyDirReader().WithExtensions(".md")
	files, err := reader.ReadDir(tempDir)
	
	if err != nil {
		t.Fatalf("ReadDir failed: %v", err)
	}
	
	if len(files) != 1 {
		t.Errorf("Expected 1 .md file, got %d", len(files))
	}
	if files[0].Name != "test.md" {
		t.Errorf("Expected test.md, got %s", files[0].Name)
	}
}

func TestLazyDirReaderExcludeDir(t *testing.T) {
	tempDir := t.TempDir()
	
	os.Mkdir(filepath.Join(tempDir, ".git"), 0755)
	os.Mkdir(filepath.Join(tempDir, "notes"), 0755)
	
	reader := NewLazyDirReader()
	files, err := reader.ReadDir(tempDir)
	
	if err != nil {
		t.Fatalf("ReadDir failed: %v", err)
	}
	
	// .git should be excluded by default
	for _, f := range files {
		if f.Name == ".git" {
			t.Error(".git should be excluded")
		}
	}
}

func TestLazyDirReaderCaching(t *testing.T) {
	tempDir := t.TempDir()
	os.WriteFile(filepath.Join(tempDir, "test.md"), []byte("# Test"), 0644)
	
	reader := NewLazyDirReader()
	
	// First read
	files1, _ := reader.ReadDir(tempDir)
	
	// Second read should be cached
	files2, _ := reader.ReadDir(tempDir)
	
	if len(files1) != len(files2) {
		t.Error("Cached result should match")
	}
}

func TestPaginatedResult(t *testing.T) {
	tempDir := t.TempDir()
	
	// Create 5 files
	for i := 0; i < 5; i++ {
		os.WriteFile(filepath.Join(tempDir, string(rune('a'+i))+".md"), []byte("test"), 0644)
	}
	
	reader := NewLazyDirReader()
	
	// Page 0, size 2
	result, err := reader.ReadDirPaginated(tempDir, 0, 2)
	if err != nil {
		t.Fatalf("ReadDirPaginated failed: %v", err)
	}
	
	if len(result.Files) != 2 {
		t.Errorf("Expected 2 files, got %d", len(result.Files))
	}
	if result.TotalCount != 5 {
		t.Errorf("Expected total 5, got %d", result.TotalCount)
	}
	if !result.HasMore {
		t.Error("Should have more pages")
	}
	
	// Last page
	result2, _ := reader.ReadDirPaginated(tempDir, 2, 2)
	if result2.HasMore {
		t.Error("Last page should not have more")
	}
}

func TestListMarkdownFiles(t *testing.T) {
	tempDir := t.TempDir()
	
	os.WriteFile(filepath.Join(tempDir, "test.md"), []byte("# Test"), 0644)
	os.WriteFile(filepath.Join(tempDir, "test.txt"), []byte("text"), 0644)
	
	files, err := ListMarkdownFiles(tempDir)
	if err != nil {
		t.Fatalf("ListMarkdownFiles failed: %v", err)
	}
	
	if len(files) != 1 {
		t.Errorf("Expected 1 markdown file, got %d", len(files))
	}
}

func TestGetFileCache(t *testing.T) {
	cache1 := GetFileCache()
	cache2 := GetFileCache()
	
	if cache1 != cache2 {
		t.Error("GetFileCache should return singleton")
	}
}

func TestFileInfo(t *testing.T) {
	info := CachedFileInfo{
		Name:    "test.md",
		Path:    "/path/to/test.md",
		IsDir:   false,
		Size:    1024,
		ModTime: time.Now(),
	}
	
	if info.Name != "test.md" {
		t.Error("Name mismatch")
	}
	if info.IsDir {
		t.Error("Should not be directory")
	}
}
