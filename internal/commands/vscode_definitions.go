package commands

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// DefinitionItem represents a single definition for JSON output
type DefinitionItem struct {
	Name         string `json:"name"`
	Abbreviation string `json:"abbreviation"`
	Description  string `json:"description,omitempty"`
	Organization string `json:"organization,omitempty"`
}

// DefinitionsListResult is the response from vscode definitions list
type DefinitionsListResult struct {
	Organizations []DefinitionItem `json:"organizations"`
	Projects      []DefinitionItem `json:"projects"`
	Contexts      []DefinitionItem `json:"contexts"`
	People        []DefinitionItem `json:"people"`
	Source        string           `json:"source"`
}

// NewVSCodeDefinitionsCommand creates the vscode definitions subcommand
func NewVSCodeDefinitionsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "definitions",
		Short: "Definition-related VS Code integration commands",
	}

	cmd.AddCommand(NewDefinitionsListCommand())
	cmd.AddCommand(NewDefinitionsAddOrgCommand())
	cmd.AddCommand(NewDefinitionsAddPersonCommand())
	cmd.AddCommand(NewDefinitionsRemoveCommand())

	return cmd
}

// NewDefinitionsListCommand lists all definitions for VS Code
func NewDefinitionsListCommand() *cobra.Command {
	var jsonOutput bool
	var brainName string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all definitions (organizations, projects, contexts, people)",
		RunE: func(cmd *cobra.Command, args []string) error {
			JSONOutput = jsonOutput

			defs, err := loadTaskDefinitionsForBrain(brainName)
			if err != nil {
				OutputJSONError("definitions-list", err)
				return nil
			}

			// Convert to result format
			orgs := make([]DefinitionItem, len(defs.Organizations))
			for i, o := range defs.Organizations {
				orgs[i] = DefinitionItem{
					Name:         o.Name,
					Abbreviation: o.Abbreviation,
					Description:  o.Description,
				}
			}

			projs := make([]DefinitionItem, len(defs.Projects))
			for i, p := range defs.Projects {
				projs[i] = DefinitionItem{
					Name:         p.Name,
					Abbreviation: p.Abbreviation,
					Description:  p.Description,
					Organization: p.Organization,
				}
			}

			ctxs := make([]DefinitionItem, len(defs.Contexts))
			for i, c := range defs.Contexts {
				ctxs[i] = DefinitionItem{
					Name:         c.Name,
					Abbreviation: c.Abbreviation,
					Description:  c.Description,
					Organization: c.Organization,
				}
			}

			people := make([]DefinitionItem, len(defs.People))
			for i, p := range defs.People {
				people[i] = DefinitionItem{
					Name:         p.Name,
					Abbreviation: p.Abbreviation,
					Description:  p.Description,
					Organization: p.Organization,
				}
			}

			result := DefinitionsListResult{
				Organizations: orgs,
				Projects:      projs,
				Contexts:      ctxs,
				People:        people,
				Source:        defs.Source,
			}

			OutputJSONSuccess("definitions-list", result)
			return nil
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output JSON (for VS Code integration)")
	cmd.Flags().StringVar(&brainName, "brain", "", "Brain name to use (optional, uses active brain if not set)")

	return cmd
}

// AddOrgResult is the response from adding an organization
type AddOrgResult struct {
	Name         string `json:"name"`
	Abbreviation string `json:"abbreviation"`
	Description  string `json:"description,omitempty"`
}

// NewDefinitionsAddOrgCommand adds a new organization (non-interactive)
func NewDefinitionsAddOrgCommand() *cobra.Command {
	var name, abbr, desc, brainName string
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "add-org",
		Short: "Add a new organization definition",
		RunE: func(cmd *cobra.Command, args []string) error {
			JSONOutput = jsonOutput
			if abbr == "" {
				OutputJSONError("definitions-add-org", fmt.Errorf("--abbreviation is required"))
				return nil
			}

			defs, err := loadTaskDefinitionsForBrain(brainName)
			if err != nil {
				OutputJSONError("definitions-add-org", err)
				return nil
			}

			// Use abbreviation as name if name not provided
			if name == "" {
				name = abbr
			}

			if err := defs.addOrganization(name, abbr, desc); err != nil {
				OutputJSONError("definitions-add-org", err)
				return nil
			}

			if err := saveTaskDefinitions(defs); err != nil {
				OutputJSONError("definitions-add-org", err)
				return nil
			}

			result := AddOrgResult{
				Name:         name,
				Abbreviation: abbr,
				Description:  desc,
			}

			OutputJSONSuccess("definitions-add-org", result)
			return nil
		},
	}

	cmd.Flags().StringVar(&abbr, "abbreviation", "", "Organization abbreviation (required)")
	cmd.Flags().StringVar(&name, "name", "", "Organization full name (optional, defaults to abbreviation)")
	cmd.Flags().StringVar(&desc, "description", "", "Organization description (optional)")
	cmd.Flags().StringVar(&brainName, "brain", "", "Brain name to use (optional, uses active brain if not set)")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output JSON (for VS Code integration)")

	return cmd
}

// AddPersonResult is the response from adding a person
type AddPersonResult struct {
	Name         string `json:"name"`
	Abbreviation string `json:"abbreviation"`
	Organization string `json:"organization,omitempty"`
	Role         string `json:"role,omitempty"`
}

// NewDefinitionsAddPersonCommand adds a new person definition
func NewDefinitionsAddPersonCommand() *cobra.Command {
	var name, abbr, org, role, brainName string
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "add-person",
		Short: "Add a new person definition",
		RunE: func(cmd *cobra.Command, args []string) error {
			JSONOutput = jsonOutput
			if name == "" {
				OutputJSONError("definitions-add-person", fmt.Errorf("--name is required"))
				return nil
			}

			defs, err := loadTaskDefinitionsForBrain(brainName)
			if err != nil {
				OutputJSONError("definitions-add-person", err)
				return nil
			}

			// Use first name as abbreviation if not provided
			if abbr == "" {
				// Simple abbreviation: first letter of first name + first letter of last name
				parts := strings.Fields(name)
				if len(parts) > 0 {
					abbr = strings.ToUpper(parts[0][:1])
					if len(parts) > 1 {
						abbr += strings.ToUpper(parts[len(parts)-1][:1])
					}
				}
			}

			if err := defs.addPerson(name, abbr, org, "", role, ""); err != nil {
				OutputJSONError("definitions-add-person", err)
				return nil
			}

			if err := saveTaskDefinitions(defs); err != nil {
				OutputJSONError("definitions-add-person", err)
				return nil
			}

			result := AddPersonResult{
				Name:         name,
				Abbreviation: abbr,
				Organization: org,
				Role:         role,
			}

			OutputJSONSuccess("definitions-add-person", result)
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Person's name (required)")
	cmd.Flags().StringVar(&abbr, "abbreviation", "", "Person's abbreviation (optional, auto-generated if not provided)")
	cmd.Flags().StringVar(&org, "organization", "", "Person's organization (optional)")
	cmd.Flags().StringVar(&role, "role", "", "Person's role (optional)")
	cmd.Flags().StringVar(&brainName, "brain", "", "Brain name to use (optional, uses active brain if not set)")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output JSON (for VS Code integration)")

	return cmd
}

// NewDefinitionsRemoveCommand removes a definition
func NewDefinitionsRemoveCommand() *cobra.Command {
	var defType, abbr, brainName string
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "remove",
		Short: "Remove a definition (organization, person, context)",
		RunE: func(cmd *cobra.Command, args []string) error {
			JSONOutput = jsonOutput
			if abbr == "" {
				OutputJSONError("definitions-remove", fmt.Errorf("--abbreviation is required"))
				return nil
			}
			if defType == "" {
				OutputJSONError("definitions-remove", fmt.Errorf("--type is required (organization, person, context)"))
				return nil
			}

			defs, err := loadTaskDefinitionsForBrain(brainName)
			if err != nil {
				OutputJSONError("definitions-remove", err)
				return nil
			}

			var removed bool
			switch defType {
			case "organization", "org":
				for i, o := range defs.Organizations {
					if strings.EqualFold(o.Abbreviation, abbr) {
						defs.Organizations = append(defs.Organizations[:i], defs.Organizations[i+1:]...)
						removed = true
						break
					}
				}
			case "person", "people":
				for i, p := range defs.People {
					if strings.EqualFold(p.Abbreviation, abbr) {
						defs.People = append(defs.People[:i], defs.People[i+1:]...)
						removed = true
						break
					}
				}
			case "context":
				for i, c := range defs.Contexts {
					if strings.EqualFold(c.Abbreviation, abbr) {
						defs.Contexts = append(defs.Contexts[:i], defs.Contexts[i+1:]...)
						removed = true
						break
					}
				}
			default:
				OutputJSONError("definitions-remove", fmt.Errorf("unknown type: %s", defType))
				return nil
			}

			if !removed {
				OutputJSONError("definitions-remove", fmt.Errorf("%s with abbreviation '%s' not found", defType, abbr))
				return nil
			}

			if err := saveTaskDefinitions(defs); err != nil {
				OutputJSONError("definitions-remove", err)
				return nil
			}

			OutputJSONSuccess("definitions-remove", map[string]string{
				"type":         defType,
				"abbreviation": abbr,
				"status":       "removed",
			})
			return nil
		},
	}

	cmd.Flags().StringVar(&defType, "type", "", "Definition type (organization, person, context)")
	cmd.Flags().StringVar(&abbr, "abbreviation", "", "Abbreviation to remove")
	cmd.Flags().StringVar(&brainName, "brain", "", "Brain name to use (optional)")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output JSON")

	return cmd
}
