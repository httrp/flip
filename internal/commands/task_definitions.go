package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// Definitions cache to avoid repeated file reads
var (
	definitionsCache     *TaskDefinitions
	definitionsCacheMu   sync.RWMutex
	definitionsCacheTime time.Time
	definitionsCachePath string
)

// TaskDefinitions stores definitions for organizations, projects, contexts, and people
// Supports both simple array format and brain hierarchical format
type TaskDefinitions struct {
	Organizations []OrganizationDef `yaml:"organizations"`
	Projects      []ProjectDef      `yaml:"projects"`
	Contexts      []ContextDef      `yaml:"contexts"`
	People        []PersonDef       `yaml:"people,omitempty"`
	Source        string            `yaml:"-"` // Where definitions were loaded from
}

// OrganizationDef defines an organization with its abbreviation
// Extended to support brain format fields
type OrganizationDef struct {
	Name         string `yaml:"name"`
	Abbreviation string `yaml:"abbreviation"`
	Description  string `yaml:"description,omitempty"`
	Type         string `yaml:"type,omitempty"`   // brain format: "work", "personal"
	Color        string `yaml:"color,omitempty"`  // brain format: hex color
	Status       string `yaml:"status,omitempty"` // brain format: "active", "archived"
}

// ProjectDef defines a project with its abbreviation and organization
// Extended to support brain format fields
type ProjectDef struct {
	Name         string `yaml:"name"`
	Abbreviation string `yaml:"abbreviation"`
	Organization string `yaml:"organization,omitempty"`
	Description  string `yaml:"description,omitempty"`
	Type         string `yaml:"type,omitempty"`   // brain format: project type
	Status       string `yaml:"status,omitempty"` // brain format: "active", "completed", "archived"
	Color        string `yaml:"color,omitempty"`  // brain format: hex color
}

// ContextDef defines a context with its abbreviation
// Extended to support brain format fields
type ContextDef struct {
	Name         string `yaml:"name"`
	Abbreviation string `yaml:"abbreviation"`
	Description  string `yaml:"description,omitempty"`
	Organization string `yaml:"organization,omitempty"` // brain format: nested under org
	Status       string `yaml:"status,omitempty"`       // brain format: "active", "ongoing", "completed"
}

// PersonDef defines a person with their details
// Extended to support brain format fields
type PersonDef struct {
	Name         string `yaml:"name"`
	Abbreviation string `yaml:"abbreviation"`           // Our internal abbreviation (e.g., "SELF", "JD")
	Organization string `yaml:"organization,omitempty"` // Which organization they belong to
	OrgCode      string `yaml:"org_code,omitempty"`     // Official org-assigned code (e.g., "DH2024", "EMP-123")
	Role         string `yaml:"role,omitempty"`
	Email        string `yaml:"email,omitempty"`
	Description  string `yaml:"description,omitempty"`
}

// BrainOrganizations represents the brain format for organizations.yaml
type BrainOrganizations struct {
	Organizations map[string]BrainOrganizationDef `yaml:"organizations"`
}

type BrainOrganizationDef struct {
	Name        string `yaml:"name"`
	Type        string `yaml:"type,omitempty"`
	Description string `yaml:"description,omitempty"`
	Color       string `yaml:"color,omitempty"`
	Status      string `yaml:"status,omitempty"`
}

// BrainContexts represents the brain format for contexts.yaml
// Contexts are nested under organization keys
type BrainContexts struct {
	Contexts map[string]map[string]BrainContextDef `yaml:"contexts"`
}

type BrainContextDef struct {
	Name        string `yaml:"name"`
	Status      string `yaml:"status,omitempty"`
	Description string `yaml:"description,omitempty"`
}

// BrainProjects represents the brain format for projects.yaml (if it exists)
type BrainProjects struct {
	Projects map[string]BrainProjectDef `yaml:"projects"`
}

type BrainProjectDef struct {
	Name         string `yaml:"name"`
	Organization string `yaml:"organization,omitempty"`
	Type         string `yaml:"type,omitempty"`
	Status       string `yaml:"status,omitempty"`
	Description  string `yaml:"description,omitempty"`
	Color        string `yaml:"color,omitempty"`
}

// BrainPeople represents the brain format for people.yaml
type BrainPeople struct {
	People map[string]BrainPersonDef `yaml:"people"`
}

type BrainPersonDef struct {
	Name         string `yaml:"name"`
	Organization string `yaml:"organization,omitempty"`
	OrgCode      string `yaml:"org_code,omitempty"` // Official organization code
	Role         string `yaml:"role,omitempty"`
	Email        string `yaml:"email,omitempty"`
}

// getTaskDefinitionsPath returns the path to the task definitions file
// First checks brain's definitions, then falls back to user config
func getTaskDefinitionsPath() (string, error) {
	// Try to get active brain's definitions first
	brain, err := getActiveBrain()
	if err == nil && brain != nil {
		brainDefPath := filepath.Join(brain.Path, "definitions", "task-definitions.yaml")
		if _, err := os.Stat(brainDefPath); err == nil {
			return brainDefPath, nil
		}
	}

	// Fall back to user config directory
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}

	flipDir := filepath.Join(configDir, "flip")
	if err := os.MkdirAll(flipDir, 0755); err != nil {
		return "", err
	}

	return filepath.Join(flipDir, "task-definitions.yaml"), nil
}

// loadTaskDefinitions loads the task definitions from disk with caching
// Priority: 1) Brain format, 2) Simple format, 3) Default
// Uses file modification time to determine if cache is still valid
func loadTaskDefinitions() (*TaskDefinitions, error) {
	// Try to determine the path we should be checking
	var checkPath string
	var isBrain bool

	// Try brain format first
	brain, err := getActiveBrain()
	if err == nil && brain != nil {
		checkPath = filepath.Join(brain.Path, "definitions")
		isBrain = true
	} else {
		// Fall back to simple format
		checkPath, _ = getTaskDefinitionsPath()
	}

	// Check cache validity using file modification time
	definitionsCacheMu.RLock()
	cacheValid := false
	if definitionsCache != nil && definitionsCachePath == checkPath {
		// Check if file has been modified since cache was created
		if stat, err := os.Stat(checkPath); err == nil {
			// For brain format, check the directory; for simple, check the file
			cacheValid = !stat.ModTime().After(definitionsCacheTime)
		}
	}

	if cacheValid {
		defs := definitionsCache
		definitionsCacheMu.RUnlock()
		return defs, nil
	}
	definitionsCacheMu.RUnlock()

	// Cache is invalid or doesn't exist, load from disk
	definitionsCacheMu.Lock()
	defer definitionsCacheMu.Unlock()

	var defs *TaskDefinitions

	if isBrain {
		brainDefs, err := loadBrainDefinitions(brain.Path)
		if err == nil && brainDefs != nil {
			brainDefs.Source = "brain"
			defs = brainDefs
		}
	}

	// If brain loading failed or not applicable, fall back to simple format
	if defs == nil {
		path, err := getTaskDefinitionsPath()
		if err != nil {
			defs = createDefaultDefinitions()
			defs.Source = "default"
		} else {
			// If file doesn't exist, create default
			if _, err := os.Stat(path); os.IsNotExist(err) {
				defs = createDefaultDefinitions()
				defs.Source = "default"
				if saveErr := saveTaskDefinitions(defs); saveErr != nil {
					return defs, saveErr
				}
			} else {
				data, err := os.ReadFile(path)
				if err != nil {
					defs = createDefaultDefinitions()
					defs.Source = "default"
				} else {
					var loadedDefs TaskDefinitions
					if err := yaml.Unmarshal(data, &loadedDefs); err != nil {
						defs = createDefaultDefinitions()
						defs.Source = "default"
					} else {
						loadedDefs.Source = "simple"
						defs = &loadedDefs
					}
				}
			}
		}
	}

	// Update cache
	definitionsCache = defs
	definitionsCachePath = checkPath
	definitionsCacheTime = time.Now()

	return defs, nil
}

// invalidateDefinitionsCache clears the definitions cache, forcing a reload on next access
func invalidateDefinitionsCache() {
	definitionsCacheMu.Lock()
	defer definitionsCacheMu.Unlock()
	definitionsCache = nil
	definitionsCachePath = ""
	definitionsCacheTime = time.Time{}
}

// loadBrainDefinitions loads definitions from brain's definitions directory
// Returns nil if brain format doesn't exist
func loadBrainDefinitions(brainPath string) (*TaskDefinitions, error) {
	defsDir := filepath.Join(brainPath, "definitions")

	// Check if definitions directory exists
	if _, err := os.Stat(defsDir); os.IsNotExist(err) {
		return nil, err
	}

	defs := &TaskDefinitions{
		Organizations: []OrganizationDef{},
		Projects:      []ProjectDef{},
		Contexts:      []ContextDef{},
		People:        []PersonDef{},
		Source:        "brain",
	}

	// Load organizations.yaml
	orgPath := filepath.Join(defsDir, "organizations.yaml")
	if _, err := os.Stat(orgPath); err == nil {
		if orgs, err := loadBrainOrganizations(orgPath); err == nil {
			defs.Organizations = orgs
		}
	}

	// Load contexts.yaml
	ctxPath := filepath.Join(defsDir, "contexts.yaml")
	if _, err := os.Stat(ctxPath); err == nil {
		if contexts, err := loadBrainContexts(ctxPath); err == nil {
			defs.Contexts = contexts
		}
	}

	// Load projects.yaml (if it exists)
	projPath := filepath.Join(defsDir, "projects.yaml")
	if _, err := os.Stat(projPath); err == nil {
		if projects, err := loadBrainProjects(projPath); err == nil {
			defs.Projects = projects
		}
	}

	// Load people.yaml
	peoplePath := filepath.Join(defsDir, "people.yaml")
	if _, err := os.Stat(peoplePath); err == nil {
		if people, err := loadBrainPeople(peoplePath); err == nil {
			defs.People = people
		}
	}

	// Only return if we found at least one file
	if len(defs.Organizations) > 0 || len(defs.Projects) > 0 || len(defs.Contexts) > 0 || len(defs.People) > 0 {
		return defs, nil
	}

	return nil, fmt.Errorf("no brain definitions found")
}

// loadBrainOrganizations loads organizations from brain format
func loadBrainOrganizations(path string) ([]OrganizationDef, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var brainOrgs BrainOrganizations
	if err := yaml.Unmarshal(data, &brainOrgs); err != nil {
		return nil, err
	}

	// Convert to unified format
	orgs := []OrganizationDef{}
	for abbr, org := range brainOrgs.Organizations {
		orgs = append(orgs, OrganizationDef{
			Name:         org.Name,
			Abbreviation: abbr,
			Description:  org.Description,
			Type:         org.Type,
			Color:        org.Color,
			Status:       org.Status,
		})
	}

	// Sort by abbreviation for consistency
	sort.Slice(orgs, func(i, j int) bool {
		return orgs[i].Abbreviation < orgs[j].Abbreviation
	})

	return orgs, nil
}

// loadBrainContexts loads contexts from brain format
func loadBrainContexts(path string) ([]ContextDef, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var brainCtxs BrainContexts
	if err := yaml.Unmarshal(data, &brainCtxs); err != nil {
		return nil, err
	}

	// Convert to unified format
	contexts := []ContextDef{}
	for orgKey, orgContexts := range brainCtxs.Contexts {
		for ctxAbbr, ctx := range orgContexts {
			contexts = append(contexts, ContextDef{
				Name:         ctx.Name,
				Abbreviation: ctxAbbr,
				Description:  ctx.Description,
				Organization: orgKey,
				Status:       ctx.Status,
			})
		}
	}

	// Sort by abbreviation
	sort.Slice(contexts, func(i, j int) bool {
		return contexts[i].Abbreviation < contexts[j].Abbreviation
	})

	return contexts, nil
}

// loadBrainProjects loads projects from brain format (if file exists)
func loadBrainProjects(path string) ([]ProjectDef, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var brainProjs BrainProjects
	if err := yaml.Unmarshal(data, &brainProjs); err != nil {
		return nil, err
	}

	// Convert to unified format
	projects := []ProjectDef{}
	for abbr, proj := range brainProjs.Projects {
		projects = append(projects, ProjectDef{
			Name:         proj.Name,
			Abbreviation: abbr,
			Organization: proj.Organization,
			Description:  proj.Description,
			Type:         proj.Type,
			Status:       proj.Status,
			Color:        proj.Color,
		})
	}

	// Sort by abbreviation
	sort.Slice(projects, func(i, j int) bool {
		return projects[i].Abbreviation < projects[j].Abbreviation
	})

	return projects, nil
}

// loadBrainPeople loads people from brain format
func loadBrainPeople(path string) ([]PersonDef, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var brainPeople BrainPeople
	if err := yaml.Unmarshal(data, &brainPeople); err != nil {
		return nil, err
	}

	// Convert to unified format
	people := []PersonDef{}
	for abbr, person := range brainPeople.People {
		people = append(people, PersonDef{
			Name:         person.Name,
			Abbreviation: abbr,
			Organization: person.Organization,
			OrgCode:      person.OrgCode,
			Role:         person.Role,
			Email:        person.Email,
		})
	}

	// Sort by abbreviation
	sort.Slice(people, func(i, j int) bool {
		return people[i].Abbreviation < people[j].Abbreviation
	})

	return people, nil
}

// saveBrainDefinitions saves definitions to brain's individual yaml files
func saveBrainDefinitions(defs *TaskDefinitions) error {
	brain, err := getActiveBrain()
	if err != nil {
		return fmt.Errorf("no active brain: %w", err)
	}

	defsDir := filepath.Join(brain.Path, "definitions")
	if err := os.MkdirAll(defsDir, 0755); err != nil {
		return fmt.Errorf("failed to create definitions directory: %w", err)
	}

	// Save people.yaml in brain format
	if len(defs.People) > 0 {
		if err := saveBrainPeople(filepath.Join(defsDir, "people.yaml"), defs.People); err != nil {
			return fmt.Errorf("failed to save people: %w", err)
		}
	}

	// Save organizations.yaml in brain format
	if len(defs.Organizations) > 0 {
		if err := saveBrainOrganizations(filepath.Join(defsDir, "organizations.yaml"), defs.Organizations); err != nil {
			return fmt.Errorf("failed to save organizations: %w", err)
		}
	}

	// Invalidate cache after successful save
	invalidateDefinitionsCache()

	return nil
}

// saveBrainPeople saves people to brain format (map structure)
func saveBrainPeople(path string, people []PersonDef) error {
	// Convert array to map format
	peopleMap := make(map[string]BrainPersonDef)
	for _, p := range people {
		peopleMap[p.Abbreviation] = BrainPersonDef{
			Name:         p.Name,
			Organization: p.Organization,
			OrgCode:      p.OrgCode,
			Role:         p.Role,
			Email:        p.Email,
		}
	}

	brainPeople := BrainPeople{People: peopleMap}
	data, err := yaml.Marshal(brainPeople)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// saveBrainOrganizations saves organizations to brain format (map structure)
func saveBrainOrganizations(path string, orgs []OrganizationDef) error {
	// Convert array to map format
	orgsMap := make(map[string]BrainOrganizationDef)
	for _, o := range orgs {
		orgsMap[o.Abbreviation] = BrainOrganizationDef{
			Name:        o.Name,
			Type:        o.Type,
			Description: o.Description,
			Color:       o.Color,
			Status:      o.Status,
		}
	}

	brainOrgs := BrainOrganizations{Organizations: orgsMap}
	data, err := yaml.Marshal(brainOrgs)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// saveTaskDefinitions saves the task definitions to disk
func saveTaskDefinitions(defs *TaskDefinitions) error {
	// If source is "brain", save to brain's individual yaml files
	if defs.Source == "brain" {
		return saveBrainDefinitions(defs)
	}

	// Otherwise save to unified task-definitions.yaml
	path, err := getTaskDefinitionsPath()
	if err != nil {
		return err
	}

	data, err := yaml.Marshal(defs)
	if err != nil {
		return err
	}

	// Add header comment
	header := `# Task Definitions
# Define your organizations, projects, and contexts here with abbreviations
# This file can be in your brain's definitions folder or in ~/.config/flip/

`
	content := header + string(data)

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return err
	}

	// Invalidate cache after successful save
	invalidateDefinitionsCache()

	return nil
}

// createDefaultDefinitions creates default definitions
func createDefaultDefinitions() *TaskDefinitions {
	return &TaskDefinitions{
		Organizations: []OrganizationDef{
			{
				Name:         "Personal",
				Abbreviation: "PERS",
				Description:  "Personal tasks and projects",
			},
			{
				Name:         "Work",
				Abbreviation: "WORK",
				Description:  "Work-related tasks",
			},
		},
		Projects: []ProjectDef{
			{
				Name:         "Example Project",
				Abbreviation: "EX",
				Organization: "WORK",
				Description:  "Example project",
			},
		},
		Contexts: []ContextDef{
			{
				Name:         "Meeting",
				Abbreviation: "MTG",
				Description:  "Tasks from meetings",
			},
			{
				Name:         "Email",
				Abbreviation: "EMAIL",
				Description:  "Tasks from emails",
			},
		},
	}
}

// findOrganizationByAbbr finds an organization by abbreviation
func (d *TaskDefinitions) findOrganizationByAbbr(abbr string) *OrganizationDef {
	for i := range d.Organizations {
		if d.Organizations[i].Abbreviation == abbr {
			return &d.Organizations[i]
		}
	}
	return nil
}

// findProjectByAbbr finds a project by abbreviation
func (d *TaskDefinitions) findProjectByAbbr(abbr string) *ProjectDef {
	for i := range d.Projects {
		if d.Projects[i].Abbreviation == abbr {
			return &d.Projects[i]
		}
	}
	return nil
}

// findContextByAbbr finds a context by abbreviation
func (d *TaskDefinitions) findContextByAbbr(abbr string) *ContextDef {
	for i := range d.Contexts {
		if d.Contexts[i].Abbreviation == abbr {
			return &d.Contexts[i]
		}
	}
	return nil
}

// getOrganizationChoices returns formatted choices for organization selection
func (d *TaskDefinitions) getOrganizationChoices() []string {
	choices := make([]string, len(d.Organizations))
	for i, org := range d.Organizations {
		choices[i] = fmt.Sprintf("[%s] %s", org.Abbreviation, org.Name)
	}
	sort.Strings(choices)
	return choices
}

// getProjectChoices returns formatted choices for project selection
func (d *TaskDefinitions) getProjectChoices() []string {
	choices := make([]string, len(d.Projects))
	for i, proj := range d.Projects {
		if proj.Organization != "" {
			choices[i] = fmt.Sprintf("[%s] %s (%s)", proj.Abbreviation, proj.Name, proj.Organization)
		} else {
			choices[i] = fmt.Sprintf("[%s] %s", proj.Abbreviation, proj.Name)
		}
	}
	sort.Strings(choices)
	return choices
}

// getContextChoices returns formatted choices for context selection
func (d *TaskDefinitions) getContextChoices() []string {
	choices := make([]string, len(d.Contexts))
	for i, ctx := range d.Contexts {
		choices[i] = fmt.Sprintf("[%s] %s", ctx.Abbreviation, ctx.Name)
	}
	sort.Strings(choices)
	return choices
}

// parseAbbreviationFromChoice extracts abbreviation from a choice string like "[ABBR] Name"
func parseAbbreviationFromChoice(choice string) string {
	if len(choice) < 2 {
		return ""
	}
	// Extract text between [ and ]
	start := 0
	end := len(choice)
	for i, c := range choice {
		if c == '[' {
			start = i + 1
		} else if c == ']' {
			end = i
			break
		}
	}
	if start < end {
		return choice[start:end]
	}
	return ""
}

// hasOrganization checks if an organization name or abbreviation already exists
func (d *TaskDefinitions) hasOrganization(name, abbr string) bool {
	for _, org := range d.Organizations {
		if strings.EqualFold(org.Name, name) || strings.EqualFold(org.Abbreviation, abbr) {
			return true
		}
	}
	return false
}

// hasProject checks if a project name or abbreviation already exists
func (d *TaskDefinitions) hasProject(name, abbr string) bool {
	for _, proj := range d.Projects {
		if strings.EqualFold(proj.Name, name) || strings.EqualFold(proj.Abbreviation, abbr) {
			return true
		}
	}
	return false
}

// hasContext checks if a context name or abbreviation already exists
func (d *TaskDefinitions) hasContext(name, abbr string) bool {
	for _, ctx := range d.Contexts {
		if strings.EqualFold(ctx.Name, name) || strings.EqualFold(ctx.Abbreviation, abbr) {
			return true
		}
	}
	return false
}

// hasPerson checks if a person name or abbreviation already exists
func (d *TaskDefinitions) hasPerson(name, abbr string) bool {
	for _, person := range d.People {
		if strings.EqualFold(person.Name, name) || strings.EqualFold(person.Abbreviation, abbr) {
			return true
		}
	}
	return false
}

// addOrganization adds a new organization to the definitions
func (d *TaskDefinitions) addOrganization(name, abbr, description string) error {
	if d.hasOrganization(name, abbr) {
		return fmt.Errorf("organization with name '%s' or abbreviation '%s' already exists", name, abbr)
	}
	d.Organizations = append(d.Organizations, OrganizationDef{
		Name:         name,
		Abbreviation: abbr,
		Description:  description,
	})
	return nil
}

// addProject adds a new project to the definitions
func (d *TaskDefinitions) addProject(name, abbr, org, description string) error {
	if d.hasProject(name, abbr) {
		return fmt.Errorf("project with name '%s' or abbreviation '%s' already exists", name, abbr)
	}
	d.Projects = append(d.Projects, ProjectDef{
		Name:         name,
		Abbreviation: abbr,
		Organization: org,
		Description:  description,
	})
	return nil
}

// addContext adds a new context to the definitions
func (d *TaskDefinitions) addContext(name, abbr, description string) error {
	if d.hasContext(name, abbr) {
		return fmt.Errorf("context with name '%s' or abbreviation '%s' already exists", name, abbr)
	}
	d.Contexts = append(d.Contexts, ContextDef{
		Name:         name,
		Abbreviation: abbr,
		Description:  description,
	})
	return nil
}

// addPerson adds a new person to the definitions
func (d *TaskDefinitions) addPerson(name, abbr, org, orgCode, role, email string) error {
	if d.hasPerson(name, abbr) {
		return fmt.Errorf("person with name '%s' or abbreviation '%s' already exists", name, abbr)
	}
	d.People = append(d.People, PersonDef{
		Name:         name,
		Abbreviation: abbr,
		Organization: org,
		OrgCode:      orgCode,
		Role:         role,
		Email:        email,
	})
	return nil
}
