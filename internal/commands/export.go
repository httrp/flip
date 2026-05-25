package commands

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// ExportTemplate describes a discovered template or reference document.
type ExportTemplate struct {
	Name   string `json:"name"`
	Path   string `json:"path"`
	Source string `json:"source"`
	Kind   string `json:"kind"` // latex|html|docx
}

// ExportConvertResult describes a finished export conversion.
type ExportConvertResult struct {
	Input      string `json:"input"`
	Output     string `json:"output"`
	Format     string `json:"format"`
	Template   string `json:"template,omitempty"`
	BrainName  string `json:"brain_name,omitempty"`
	OutputRoot string `json:"output_root"`
}

// ExportDoctorResult provides dependency and environment diagnostics.
type ExportDoctorResult struct {
	PandocAvailable  bool             `json:"pandoc_available"`
	PandocPath       string           `json:"pandoc_path,omitempty"`
	XeLaTeXAvailable bool             `json:"xelatex_available"`
	XeLaTeXPath      string           `json:"xelatex_path,omitempty"`
	OutputDirDefault string           `json:"output_dir_default"`
	TemplateCount    int              `json:"template_count"`
	Templates        []ExportTemplate `json:"templates,omitempty"`
	InstallHints     []string         `json:"install_hints,omitempty"`
}

// NewExportCommand creates the export command group.
func NewExportCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export markdown files to PDF, HTML, and DOCX",
		Long:  "Convert markdown files with pandoc while keeping output outside your brain by default.",
	}

	cmd.AddCommand(newExportConvertCommand())
	cmd.AddCommand(newExportTemplateCommand())
	cmd.AddCommand(newExportDoctorCommand())
	return cmd
}

func newExportConvertCommand() *cobra.Command {
	var (
		format      string
		outputFile  string
		outputDir   string
		templateArg string
		reference   string
		landscape   bool
		openAfter   bool
		brainName   string
		jsonOutput  bool
	)

	cmd := &cobra.Command{
		Use:   "convert <input.md>",
		Short: "Convert a markdown file to another format",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			JSONOutput = jsonOutput
			result, err := runExportConvert(args[0], format, outputFile, outputDir, templateArg, reference, landscape, openAfter, brainName)
			if err != nil {
				if JSONOutput {
					OutputJSONError("export-convert", err)
					return nil
				}
				return err
			}

			if JSONOutput {
				OutputJSONSuccess("export-convert", result)
				return nil
			}

			PrintSuccessf("Export complete: %s", result.Output)
			PrintInfof("Format: %s", strings.ToUpper(result.Format))
			if result.Template != "" {
				PrintInfof("Template: %s", result.Template)
			}
			PrintInfof("Output root: %s", result.OutputRoot)
			return nil
		},
	}

	cmd.Flags().StringVarP(&format, "format", "f", "pdf", "Output format: pdf|html|docx")
	cmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output file path (overrides output-dir)")
	cmd.Flags().StringVar(&outputDir, "output-dir", "", "Output directory root (default: ~/Desktop/flip-output)")
	cmd.Flags().StringVar(&templateArg, "template", "", "Pandoc template name or path")
	cmd.Flags().StringVar(&reference, "reference-doc", "", "Reference DOCX path (recommended for docx styling)")
	cmd.Flags().BoolVar(&landscape, "landscape", false, "Use landscape page layout for PDF")
	cmd.Flags().BoolVar(&openAfter, "open", false, "Open exported file after successful conversion")
	cmd.Flags().StringVar(&brainName, "brain", "", "Brain name for output path context (default: auto-detect)")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output JSON")
	return cmd
}

func newExportTemplateCommand() *cobra.Command {
	var (
		format     string
		jsonOutput bool
	)

	cmd := &cobra.Command{
		Use:   "template",
		Short: "Template discovery commands",
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List discovered templates and reference docs",
		RunE: func(cmd *cobra.Command, args []string) error {
			JSONOutput = jsonOutput
			templates, err := discoverTemplates(format)
			if err != nil {
				if JSONOutput {
					OutputJSONError("export-template-list", err)
					return nil
				}
				return err
			}

			if JSONOutput {
				OutputJSONSuccess("export-template-list", map[string]any{
					"format":    normalizeFormat(format),
					"count":     len(templates),
					"templates": templates,
				})
				return nil
			}

			if len(templates) == 0 {
				PrintWarning("No templates found")
				return nil
			}
			fmt.Printf("\nDiscovered templates (%d):\n", len(templates))
			for _, t := range templates {
				fmt.Printf("- %s [%s] (%s)\n  %s\n", t.Name, t.Kind, t.Source, t.Path)
			}
			return nil
		},
	}
	listCmd.Flags().StringVarP(&format, "format", "f", "pdf", "Filter templates: pdf|html|docx")
	listCmd.Flags().BoolVar(&jsonOutput, "json", false, "Output JSON")

	cmd.AddCommand(listCmd)
	return cmd
}

func newExportDoctorCommand() *cobra.Command {
	var (
		format     string
		jsonOutput bool
	)

	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Check export dependencies and setup",
		RunE: func(cmd *cobra.Command, args []string) error {
			JSONOutput = jsonOutput
			result, err := runExportDoctor(format)
			if err != nil {
				if JSONOutput {
					OutputJSONError("export-doctor", err)
					return nil
				}
				return err
			}

			if JSONOutput {
				OutputJSONSuccess("export-doctor", result)
				return nil
			}

			fmt.Println("\nExport doctor")
			fmt.Println("-------------")
			if result.PandocAvailable {
				PrintSuccessf("pandoc: %s", result.PandocPath)
			} else {
				PrintErrorf("pandoc: not found")
			}
			if normalizeFormat(format) == "pdf" {
				if result.XeLaTeXAvailable {
					PrintSuccessf("xelatex: %s", result.XeLaTeXPath)
				} else {
					PrintWarning("xelatex not found (PDF export may fail)")
				}
			}
			PrintInfof("Default output dir: %s", result.OutputDirDefault)
			PrintInfof("Templates discovered: %d", result.TemplateCount)
			for _, hint := range result.InstallHints {
				PrintInfo(hint)
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&format, "format", "f", "pdf", "Target format: pdf|html|docx")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output JSON")
	return cmd
}

func runExportConvert(inputPath, format, outputFile, outputDir, templateArg, referenceDoc string, landscape, openAfter bool, brainName string) (*ExportConvertResult, error) {
	resolvedFormat, err := normalizeAndValidateFormat(format)
	if err != nil {
		return nil, err
	}

	inputAbs, err := filepath.Abs(inputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve input path: %w", err)
	}

	inputInfo, err := os.Stat(inputAbs)
	if err != nil {
		return nil, fmt.Errorf("failed to read input file: %w", err)
	}
	if inputInfo.IsDir() {
		return nil, fmt.Errorf("input must be a markdown file, got directory: %s", inputAbs)
	}
	if strings.ToLower(filepath.Ext(inputAbs)) != ExtMarkdown {
		return nil, fmt.Errorf("input must end with .md: %s", inputAbs)
	}

	pandocPath, err := exec.LookPath("pandoc")
	if err != nil {
		return nil, fmt.Errorf("pandoc not found. Run 'flip export doctor' for install hints")
	}

	if resolvedFormat == "pdf" {
		if _, err := exec.LookPath("xelatex"); err != nil {
			PrintWarning("xelatex not found. PDF export may fail depending on your pandoc setup")
		}
	}

	brain, _ := detectBrainForFile(inputAbs, brainName)
	finalOutput, outputRoot, err := resolveOutputPath(inputAbs, resolvedFormat, outputFile, outputDir, brain)
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(filepath.Dir(finalOutput), 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	templateResolved, err := resolveTemplateArg(templateArg, resolvedFormat)
	if err != nil {
		return nil, err
	}

	args := []string{inputAbs, "-o", finalOutput, "--from", "markdown"}
	inputDir := filepath.Dir(inputAbs)
	switch resolvedFormat {
	case "docx":
		if referenceDoc != "" {
			args = append(args, "--reference-doc", referenceDoc)
		}
		if templateResolved != "" {
			args = append(args, "--template", templateResolved)
		}
	case "html":
		args = append(args, "--standalone", "--embed-resources", "--resource-path", inputDir)
		if templateResolved != "" {
			args = append(args, "--template", templateResolved)
		}
	case "pdf":
		args = append(args, "--pdf-engine=xelatex", "--resource-path", inputDir)
		if templateResolved != "" {
			args = append(args, "--template", templateResolved)
		}
		if landscape {
			args = append(args,
				"-V", "classoption=landscape",
				"-V", "geometry=landscape,margin=2cm",
			)
		}
	}

	cmd := exec.Command(pandocPath, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("pandoc conversion failed: %s", msg)
	}

	if openAfter {
		if err := openWithSystem(finalOutput); err != nil {
			PrintWarningf("Could not open output file: %v", err)
		}
	}

	res := &ExportConvertResult{
		Input:      inputAbs,
		Output:     finalOutput,
		Format:     resolvedFormat,
		Template:   templateResolved,
		OutputRoot: outputRoot,
	}
	if brain != nil {
		res.BrainName = brain.Name
	}
	return res, nil
}

func runExportDoctor(format string) (*ExportDoctorResult, error) {
	resolved := normalizeFormat(format)
	result := &ExportDoctorResult{}

	if p, err := exec.LookPath("pandoc"); err == nil {
		result.PandocAvailable = true
		result.PandocPath = p
	}
	if p, err := exec.LookPath("xelatex"); err == nil {
		result.XeLaTeXAvailable = true
		result.XeLaTeXPath = p
	}

	defaultDir, err := defaultExportRootDir()
	if err != nil {
		return nil, err
	}
	result.OutputDirDefault = defaultDir

	templates, err := discoverTemplates(resolved)
	if err != nil {
		return nil, err
	}
	result.TemplateCount = len(templates)
	result.Templates = templates

	if !result.PandocAvailable {
		result.InstallHints = append(result.InstallHints, installHintForPandoc())
	}
	if resolved == "pdf" && !result.XeLaTeXAvailable {
		result.InstallHints = append(result.InstallHints, installHintForXeLaTeX())
	}

	return result, nil
}

func normalizeAndValidateFormat(format string) (string, error) {
	f := normalizeFormat(format)
	switch f {
	case "pdf", "html", "docx":
		return f, nil
	default:
		return "", fmt.Errorf("unsupported format: %s", format)
	}
}

func normalizeFormat(format string) string {
	f := strings.TrimSpace(strings.ToLower(format))
	f = strings.TrimPrefix(f, ".")
	return f
}

func detectBrainForFile(inputAbs, brainName string) (*Brain, error) {
	if brainName != "" {
		return getBrainByName(brainName)
	}

	cfg, err := loadWorkspaceConfig()
	if err != nil {
		return nil, err
	}

	for _, ws := range cfg.Workspaces {
		if ws.Name != cfg.ActiveWorkspace {
			continue
		}
		for i := range ws.Brains {
			brainPath, err := filepath.Abs(ws.Brains[i].Path)
			if err != nil {
				continue
			}
			if isWithinPath(inputAbs, brainPath) {
				return &ws.Brains[i], nil
			}
		}
	}
	return nil, errors.New("brain not detected")
}

func resolveOutputPath(inputAbs, format, outputFile, outputDir string, brain *Brain) (string, string, error) {
	if outputFile != "" {
		abs, err := filepath.Abs(outputFile)
		if err != nil {
			return "", "", fmt.Errorf("failed to resolve output file: %w", err)
		}
		return abs, filepath.Dir(abs), nil
	}

	root := strings.TrimSpace(outputDir)
	if root == "" {
		envRoot := strings.TrimSpace(os.Getenv("FLIP_EXPORT_OUTPUT_DIR"))
		if envRoot != "" {
			root = envRoot
		} else {
			defaultRoot, err := defaultExportRootDir()
			if err != nil {
				return "", "", err
			}
			root = defaultRoot
		}
	}

	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", "", fmt.Errorf("failed to resolve output dir: %w", err)
	}

	fileBase := strings.TrimSuffix(filepath.Base(inputAbs), filepath.Ext(inputAbs)) + "." + format
	datePart := time.Now().Format("2006-01-02")
	segments := []string{rootAbs}

	if brain != nil {
		segments = append(segments, brain.Name, datePart)
		brainPathAbs, err := filepath.Abs(brain.Path)
		if err == nil {
			rel, relErr := filepath.Rel(brainPathAbs, filepath.Dir(inputAbs))
			if relErr == nil && rel != "." && !strings.HasPrefix(rel, "..") {
				segments = append(segments, rel)
			}
		}
	}

	dir := filepath.Join(segments...)
	return filepath.Join(dir, fileBase), rootAbs, nil
}

func defaultExportRootDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to detect home directory: %w", err)
	}
	return filepath.Join(home, "Desktop", "flip-output"), nil
}

func resolveTemplateArg(templateArg, format string) (string, error) {
	tmpl := strings.TrimSpace(templateArg)
	if tmpl == "" {
		return autoTemplateForFormat(format)
	}

	if pathExists(tmpl) {
		abs, err := filepath.Abs(tmpl)
		if err != nil {
			return "", err
		}
		return abs, nil
	}

	templates, err := discoverTemplates(format)
	if err != nil {
		return "", err
	}
	for _, t := range templates {
		if strings.EqualFold(t.Name, tmpl) {
			return t.Path, nil
		}
	}

	// Let pandoc attempt name-based template resolution.
	return tmpl, nil
}

func autoTemplateForFormat(format string) (string, error) {
	if format != "pdf" {
		return "", nil
	}
	templates, err := discoverTemplates("pdf")
	if err != nil {
		return "", err
	}
	for _, t := range templates {
		if strings.Contains(strings.ToLower(t.Name), "eisvogel") {
			return t.Path, nil
		}
	}
	return "", nil
}

func discoverTemplates(format string) ([]ExportTemplate, error) {
	resolved, err := normalizeAndValidateFormat(format)
	if err != nil {
		return nil, err
	}

	dirs := templateSearchDirs()
	seen := map[string]bool{}
	out := make([]ExportTemplate, 0)

	for _, entry := range dirs {
		if entry.Path == "" || seen[entry.Path] {
			continue
		}
		seen[entry.Path] = true

		info, err := os.Stat(entry.Path)
		if err != nil || !info.IsDir() {
			continue
		}

		items, err := os.ReadDir(entry.Path)
		if err != nil {
			continue
		}
		for _, item := range items {
			if item.IsDir() {
				continue
			}

			name := item.Name()
			kind := templateKindForFile(name)
			if !matchesTemplateKind(kind, resolved) {
				continue
			}

			full := filepath.Join(entry.Path, name)
			base := strings.TrimSuffix(name, filepath.Ext(name))
			out = append(out, ExportTemplate{
				Name:   base,
				Path:   full,
				Source: entry.Source,
				Kind:   kind,
			})
		}
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Name == out[j].Name {
			return out[i].Path < out[j].Path
		}
		return out[i].Name < out[j].Name
	})

	return out, nil
}

type templateDir struct {
	Path   string
	Source string
}

func templateSearchDirs() []templateDir {
	dirs := make([]templateDir, 0)

	if envDir := strings.TrimSpace(os.Getenv("PANDOC_DATA_DIR")); envDir != "" {
		dirs = append(dirs, templateDir{Path: filepath.Join(envDir, "templates"), Source: "env:PANDOC_DATA_DIR"})
	}

	if pdDir := strings.TrimSpace(pandocDataDir()); pdDir != "" {
		dirs = append(dirs, templateDir{Path: filepath.Join(pdDir, "templates"), Source: "pandoc-data-dir"})
	}

	if wd, err := os.Getwd(); err == nil {
		dirs = append(dirs, templateDir{Path: filepath.Join(wd, "templates"), Source: "project"})
	}

	if home, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs,
			templateDir{Path: filepath.Join(home, ".pandoc", "templates"), Source: "legacy-home"},
			templateDir{Path: filepath.Join(home, ".local", "share", "pandoc", "templates"), Source: "home-local-share"},
		)
	}

	return dirs
}

func pandocDataDir() string {
	cmd := exec.Command("pandoc", "--data-dir")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func templateKindForFile(name string) string {
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".latex", ".tex":
		return "latex"
	case ".html", ".htm":
		return "html"
	case ".docx":
		return "docx"
	default:
		return ""
	}
}

func matchesTemplateKind(kind, format string) bool {
	switch format {
	case "pdf":
		return kind == "latex"
	case "html":
		return kind == "html"
	case "docx":
		return kind == "docx"
	default:
		return false
	}
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func isWithinPath(path, base string) bool {
	rel, err := filepath.Rel(base, path)
	if err != nil {
		return false
	}
	return rel == "." || (!strings.HasPrefix(rel, "..") && rel != "")
}

func openWithSystem(path string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", path)
	case "linux":
		cmd = exec.Command("xdg-open", path)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", "", path)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
	return cmd.Start()
}

func installHintForPandoc() string {
	switch runtime.GOOS {
	case "darwin":
		return "Install pandoc: brew install pandoc"
	case "linux":
		return "Install pandoc: sudo apt-get install pandoc (or your distro package manager)"
	case "windows":
		return "Install pandoc: winget install --id JohnMacFarlane.Pandoc"
	default:
		return "Install pandoc from https://pandoc.org/installing.html"
	}
}

func installHintForXeLaTeX() string {
	switch runtime.GOOS {
	case "darwin":
		return "Install LaTeX engine: brew install --cask mactex-no-gui"
	case "linux":
		return "Install LaTeX engine: sudo apt-get install texlive-xetex"
	case "windows":
		return "Install LaTeX engine: winget install --id MiKTeX.MiKTeX (or TeX Live)"
	default:
		return "Install xelatex to support PDF rendering"
	}
}
