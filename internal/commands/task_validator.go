package commands

import (
	"fmt"
	"strings"
)

// ValidationResult contains validation information
type ValidationResult struct {
	Valid       bool
	Warnings    []string
	Suggestions []string
}

// ValidateOrganization checks if an organization exists in definitions
func ValidateOrganization(org string, defs *TaskDefinitions) *ValidationResult {
	result := &ValidationResult{
		Valid:       false,
		Warnings:    []string{},
		Suggestions: []string{},
	}

	if org == "" {
		result.Valid = true
		return result
	}

	// Exact match
	for _, def := range defs.Organizations {
		if def.Abbreviation == org || def.Name == org {
			result.Valid = true
			return result
		}
	}

	// Not found - add warning
	result.Warnings = append(result.Warnings, fmt.Sprintf("Organization '%s' not found in definitions", org))

	// Fuzzy matching for suggestions
	suggestions := findSimilarOrganizations(org, defs)
	if len(suggestions) > 0 {
		result.Suggestions = suggestions
	}

	return result
}

// ValidateProject checks if a project exists in definitions
func ValidateProject(proj string, defs *TaskDefinitions) *ValidationResult {
	result := &ValidationResult{
		Valid:       false,
		Warnings:    []string{},
		Suggestions: []string{},
	}

	if proj == "" {
		result.Valid = true
		return result
	}

	// Exact match
	for _, def := range defs.Projects {
		if def.Abbreviation == proj || def.Name == proj {
			result.Valid = true
			return result
		}
	}

	// Not found - add warning
	result.Warnings = append(result.Warnings, fmt.Sprintf("Project '%s' not found in definitions", proj))

	// Fuzzy matching for suggestions
	suggestions := findSimilarProjects(proj, defs)
	if len(suggestions) > 0 {
		result.Suggestions = suggestions
	}

	return result
}

// ValidateContext checks if a context exists in definitions
func ValidateContext(ctx string, defs *TaskDefinitions) *ValidationResult {
	result := &ValidationResult{
		Valid:       false,
		Warnings:    []string{},
		Suggestions: []string{},
	}

	if ctx == "" {
		result.Valid = true
		return result
	}

	// Exact match
	for _, def := range defs.Contexts {
		if def.Abbreviation == ctx || def.Name == ctx {
			result.Valid = true
			return result
		}
	}

	// Not found - add warning
	result.Warnings = append(result.Warnings, fmt.Sprintf("Context '%s' not found in definitions", ctx))

	// Fuzzy matching for suggestions
	suggestions := findSimilarContexts(ctx, defs)
	if len(suggestions) > 0 {
		result.Suggestions = suggestions
	}

	return result
}

// ValidatePerson checks if a person exists in definitions
func ValidatePerson(person string, defs *TaskDefinitions) *ValidationResult {
	result := &ValidationResult{
		Valid:       false,
		Warnings:    []string{},
		Suggestions: []string{},
	}

	if person == "" {
		result.Valid = true
		return result
	}

	// Exact match
	for _, def := range defs.People {
		if def.Abbreviation == person || def.Name == person {
			result.Valid = true
			return result
		}
	}

	// Not found - add warning
	result.Warnings = append(result.Warnings, fmt.Sprintf("Person '%s' not found in definitions", person))

	// Fuzzy matching for suggestions
	suggestions := findSimilarPeople(person, defs)
	if len(suggestions) > 0 {
		result.Suggestions = suggestions
	}

	return result
}

// findSimilarOrganizations uses fuzzy matching to find similar organizations
func findSimilarOrganizations(input string, defs *TaskDefinitions) []string {
	suggestions := []string{}
	input = strings.ToLower(input)

	for _, org := range defs.Organizations {
		abbr := strings.ToLower(org.Abbreviation)
		name := strings.ToLower(org.Name)

		// Case-insensitive match
		if abbr == input || name == input {
			suggestions = append(suggestions, fmt.Sprintf("[%s] %s", org.Abbreviation, org.Name))
			continue
		}

		// Contains match
		if strings.Contains(abbr, input) || strings.Contains(name, input) {
			suggestions = append(suggestions, fmt.Sprintf("[%s] %s", org.Abbreviation, org.Name))
			continue
		}

		// Levenshtein distance would be ideal here, but simple contains works for now
	}

	return suggestions
}

// findSimilarProjects uses fuzzy matching to find similar projects
func findSimilarProjects(input string, defs *TaskDefinitions) []string {
	suggestions := []string{}
	input = strings.ToLower(input)

	for _, proj := range defs.Projects {
		abbr := strings.ToLower(proj.Abbreviation)
		name := strings.ToLower(proj.Name)

		if abbr == input || name == input {
			suggestions = append(suggestions, fmt.Sprintf("[%s] %s", proj.Abbreviation, proj.Name))
			continue
		}

		if strings.Contains(abbr, input) || strings.Contains(name, input) {
			suggestions = append(suggestions, fmt.Sprintf("[%s] %s", proj.Abbreviation, proj.Name))
		}
	}

	return suggestions
}

// findSimilarContexts uses fuzzy matching to find similar contexts
func findSimilarContexts(input string, defs *TaskDefinitions) []string {
	suggestions := []string{}
	input = strings.ToLower(input)

	for _, ctx := range defs.Contexts {
		abbr := strings.ToLower(ctx.Abbreviation)
		name := strings.ToLower(ctx.Name)

		if abbr == input || name == input {
			suggestions = append(suggestions, fmt.Sprintf("[%s] %s", ctx.Abbreviation, ctx.Name))
			continue
		}

		if strings.Contains(abbr, input) || strings.Contains(name, input) {
			suggestions = append(suggestions, fmt.Sprintf("[%s] %s", ctx.Abbreviation, ctx.Name))
		}
	}

	return suggestions
}

// findSimilarPeople uses fuzzy matching to find similar people
func findSimilarPeople(input string, defs *TaskDefinitions) []string {
	suggestions := []string{}
	input = strings.ToLower(input)

	for _, person := range defs.People {
		abbr := strings.ToLower(person.Abbreviation)
		name := strings.ToLower(person.Name)

		if abbr == input || name == input {
			suggestions = append(suggestions, fmt.Sprintf("[%s] %s", person.Abbreviation, person.Name))
			continue
		}

		if strings.Contains(abbr, input) || strings.Contains(name, input) {
			suggestions = append(suggestions, fmt.Sprintf("[%s] %s", person.Abbreviation, person.Name))
		}
	}

	return suggestions
}

// CheckDuplicates finds duplicate abbreviations or names across definitions
type DuplicateInfo struct {
	Type         string // "organization", "project", "context", "person"
	Abbreviation string
	Names        []string // Multiple names for same abbreviation
	Locations    []string // Where found
}

// FindDuplicates scans definitions for duplicates and inconsistencies
func FindDuplicates(defs *TaskDefinitions) []DuplicateInfo {
	duplicates := []DuplicateInfo{}

	// Check organizations
	orgMap := make(map[string][]string)
	for _, org := range defs.Organizations {
		orgMap[org.Abbreviation] = append(orgMap[org.Abbreviation], org.Name)
	}
	for abbr, names := range orgMap {
		if len(names) > 1 {
			duplicates = append(duplicates, DuplicateInfo{
				Type:         "organization",
				Abbreviation: abbr,
				Names:        names,
			})
		}
	}

	// Check projects
	projMap := make(map[string][]string)
	for _, proj := range defs.Projects {
		projMap[proj.Abbreviation] = append(projMap[proj.Abbreviation], proj.Name)
	}
	for abbr, names := range projMap {
		if len(names) > 1 {
			duplicates = append(duplicates, DuplicateInfo{
				Type:         "project",
				Abbreviation: abbr,
				Names:        names,
			})
		}
	}

	// Check contexts
	ctxMap := make(map[string][]string)
	for _, ctx := range defs.Contexts {
		ctxMap[ctx.Abbreviation] = append(ctxMap[ctx.Abbreviation], ctx.Name)
	}
	for abbr, names := range ctxMap {
		if len(names) > 1 {
			duplicates = append(duplicates, DuplicateInfo{
				Type:         "context",
				Abbreviation: abbr,
				Names:        names,
			})
		}
	}

	// Check people
	peopleMap := make(map[string][]string)
	for _, person := range defs.People {
		peopleMap[person.Abbreviation] = append(peopleMap[person.Abbreviation], person.Name)
	}
	for abbr, names := range peopleMap {
		if len(names) > 1 {
			duplicates = append(duplicates, DuplicateInfo{
				Type:         "person",
				Abbreviation: abbr,
				Names:        names,
			})
		}
	}

	return duplicates
}
