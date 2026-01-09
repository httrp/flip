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

	return cmd
}

// NewDefinitionsListCommand lists all definitions for VS Code
func NewDefinitionsListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all definitions (organizations, projects, contexts, people)",
		RunE: func(cmd *cobra.Command, args []string) error {
			defs, err := loadTaskDefinitions()
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
	var name, abbr, desc string

	cmd := &cobra.Command{
		Use:   "add-org",
		Short: "Add a new organization definition",
		RunE: func(cmd *cobra.Command, args []string) error {
			if abbr == "" {
				OutputJSONError("definitions-add-org", fmt.Errorf("--abbreviation is required"))
				return nil
			}

			defs, err := loadTaskDefinitions()
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
	var name, abbr, org, role string

	cmd := &cobra.Command{
		Use:   "add-person",
		Short: "Add a new person definition",
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				OutputJSONError("definitions-add-person", fmt.Errorf("--name is required"))
				return nil
			}

			defs, err := loadTaskDefinitions()
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

	return cmd
}
