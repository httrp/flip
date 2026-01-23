package commands

// lazy.go - Lazy loading utilities for better performance
//
// Provides:
//   - Lazy file listing with caching
//   - Paginated directory scanning
//   - On-demand content loading

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// FileCache caches file listings for a directory
type FileCache struct {
	mu       sync.RWMutex
	entries  map[string]*CacheEntry
	maxAge   time.Duration
	maxSize  int // Maximum entries to cache
}

// CacheEntry holds cached file information
type CacheEntry struct {
	Files     []CachedFileInfo
	LoadedAt  time.Time
	Directory string
}

// CachedFileInfo holds basic file information for caching
type CachedFileInfo struct {
	Name    string
	Path    string
	IsDir   bool
	Size    int64
	ModTime time.Time
}

// Global file cache instance
var (
	fileCache     *FileCache
	fileCacheOnce sync.Once
)

// GetFileCache returns the global file cache
func GetFileCache() *FileCache {
	fileCacheOnce.Do(func() {
		fileCache = NewFileCache(5*time.Minute, 100)
	})
	return fileCache
}

// NewFileCache creates a new file cache
func NewFileCache(maxAge time.Duration, maxSize int) *FileCache {
	return &FileCache{
		entries: make(map[string]*CacheEntry),
		maxAge:  maxAge,
		maxSize: maxSize,
	}
}

// Get retrieves a cached entry if valid
func (c *FileCache) Get(dir string) (*CacheEntry, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.entries[dir]
	if !exists {
		return nil, false
	}

	// Check if expired
	if time.Since(entry.LoadedAt) > c.maxAge {
		return nil, false
	}

	return entry, true
}

// Set stores a cache entry
func (c *FileCache) Set(dir string, files []CachedFileInfo) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Evict old entries if at capacity
	if len(c.entries) >= c.maxSize {
		c.evictOldest()
	}

	c.entries[dir] = &CacheEntry{
		Files:     files,
		LoadedAt:  time.Now(),
		Directory: dir,
	}
}

// Invalidate removes a cache entry
func (c *FileCache) Invalidate(dir string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, dir)
}

// InvalidateAll clears all cache entries
func (c *FileCache) InvalidateAll() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = make(map[string]*CacheEntry)
}

// evictOldest removes the oldest cache entry
func (c *FileCache) evictOldest() {
	var oldestKey string
	var oldestTime time.Time

	for key, entry := range c.entries {
		if oldestKey == "" || entry.LoadedAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.LoadedAt
		}
	}

	if oldestKey != "" {
		delete(c.entries, oldestKey)
	}
}

// LazyDirReader reads directories lazily with caching
type LazyDirReader struct {
	cache       *FileCache
	extensions  []string // Filter by extensions (e.g., ".md")
	excludeDirs []string // Directories to skip
}

// NewLazyDirReader creates a new lazy directory reader
func NewLazyDirReader() *LazyDirReader {
	return &LazyDirReader{
		cache:       GetFileCache(),
		extensions:  []string{},
		excludeDirs: []string{".git", "node_modules", ".obsidian", ".trash"},
	}
}

// WithExtensions filters by file extensions
func (r *LazyDirReader) WithExtensions(exts ...string) *LazyDirReader {
	r.extensions = exts
	return r
}

// WithExcludeDirs adds directories to exclude
func (r *LazyDirReader) WithExcludeDirs(dirs ...string) *LazyDirReader {
	r.excludeDirs = append(r.excludeDirs, dirs...)
	return r
}

// ReadDir reads a directory with caching
func (r *LazyDirReader) ReadDir(dir string) ([]CachedFileInfo, error) {
	// Check cache first
	if entry, ok := r.cache.Get(dir); ok {
		return r.filterFiles(entry.Files), nil
	}

	// Read directory
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	files := make([]CachedFileInfo, 0, len(entries))
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			continue
		}

		files = append(files, CachedFileInfo{
			Name:    e.Name(),
			Path:    filepath.Join(dir, e.Name()),
			IsDir:   e.IsDir(),
			Size:    info.Size(),
			ModTime: info.ModTime(),
		})
	}

	// Cache the result
	r.cache.Set(dir, files)

	return r.filterFiles(files), nil
}

// filterFiles applies extension and exclusion filters
func (r *LazyDirReader) filterFiles(files []CachedFileInfo) []CachedFileInfo {
	if len(r.extensions) == 0 && len(r.excludeDirs) == 0 {
		return files
	}

	result := make([]CachedFileInfo, 0, len(files))
	for _, f := range files {
		// Skip excluded directories
		if f.IsDir {
			skip := false
			for _, ex := range r.excludeDirs {
				if f.Name == ex {
					skip = true
					break
				}
			}
			if skip {
				continue
			}
			result = append(result, f)
			continue
		}

		// Filter by extension
		if len(r.extensions) > 0 {
			ext := strings.ToLower(filepath.Ext(f.Name))
			for _, e := range r.extensions {
				if ext == strings.ToLower(e) {
					result = append(result, f)
					break
				}
			}
		} else {
			result = append(result, f)
		}
	}

	return result
}

// ReadDirRecursive reads a directory tree with pagination
func (r *LazyDirReader) ReadDirRecursive(dir string, maxDepth int) ([]CachedFileInfo, error) {
	var result []CachedFileInfo

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		// Calculate depth
		relPath, _ := filepath.Rel(dir, path)
		depth := strings.Count(relPath, string(os.PathSeparator))

		if depth > maxDepth {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip excluded directories
		if d.IsDir() {
			for _, ex := range r.excludeDirs {
				if d.Name() == ex {
					return filepath.SkipDir
				}
			}
		}

		// Filter by extension
		if !d.IsDir() && len(r.extensions) > 0 {
			ext := strings.ToLower(filepath.Ext(d.Name()))
			matched := false
			for _, e := range r.extensions {
				if ext == strings.ToLower(e) {
					matched = true
					break
				}
			}
			if !matched {
				return nil
			}
		}

		info, err := d.Info()
		if err != nil {
			return nil
		}

		result = append(result, CachedFileInfo{
			Name:    d.Name(),
			Path:    path,
			IsDir:   d.IsDir(),
			Size:    info.Size(),
			ModTime: info.ModTime(),
		})

		return nil
	})

	return result, err
}

// PaginatedResult holds paginated file results
type PaginatedResult struct {
	Files      []CachedFileInfo
	TotalCount int
	Page       int
	PageSize   int
	HasMore    bool
}

// ReadDirPaginated reads a directory with pagination
func (r *LazyDirReader) ReadDirPaginated(dir string, page, pageSize int) (*PaginatedResult, error) {
	files, err := r.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	total := len(files)
	start := page * pageSize
	end := start + pageSize

	if start >= total {
		return &PaginatedResult{
			Files:      []CachedFileInfo{},
			TotalCount: total,
			Page:       page,
			PageSize:   pageSize,
			HasMore:    false,
		}, nil
	}

	if end > total {
		end = total
	}

	return &PaginatedResult{
		Files:      files[start:end],
		TotalCount: total,
		Page:       page,
		PageSize:   pageSize,
		HasMore:    end < total,
	}, nil
}

// Convenience functions

// ListMarkdownFiles lists markdown files in a directory (cached)
func ListMarkdownFiles(dir string) ([]CachedFileInfo, error) {
	return NewLazyDirReader().
		WithExtensions(".md").
		ReadDir(dir)
}

// ListMarkdownFilesRecursive lists markdown files recursively
func ListMarkdownFilesRecursive(dir string, maxDepth int) ([]CachedFileInfo, error) {
	return NewLazyDirReader().
		WithExtensions(".md").
		ReadDirRecursive(dir, maxDepth)
}

// InvalidateDirectoryCache removes a directory from the cache
func InvalidateDirectoryCache(dir string) {
	GetFileCache().Invalidate(dir)
}
