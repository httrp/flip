package commands

import (
	"encoding/json"
	"fmt"

	"github.com/httrp/flip/internal/brain"
	"github.com/httrp/flip/internal/templates"
	"github.com/spf13/cobra"
)

// NewVSCodeTemplatesCommand creates the vscode templates parent command
func NewVSCodeTemplatesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "templates",
		Short: "Template-related VS Code integration commands",
	}

	cmd.AddCommand(
		newVSCodeTemplateListCommand(),
		newVSCodeTemplateGetCommand(),
		newVSCodeTemplateResetCommand(),
		newVSCodeTemplateGetDefaultCommand(),
	)

	return cmd
}

// TemplateInfo represents template information for JSON output
type TemplateInfo struct {
	BrainType    string `json:"brain_type"`
	TemplateType string `json:"template_type"`
	Content      string `json:"content"`
	IsDefault    bool   `json:"is_default"`
	Path         string `json:"path,omitempty"`
}

// TemplateListResponse represents a list of templates
type TemplateListResponse struct {
	Success   bool           `json:"success"`
	Templates []TemplateInfo `json:"templates"`
}

func parseBrainType(s string) brain.BrainType {
	switch s {
	case "flip":
		return brain.BrainTypeFlip
	case "obsidian":
		return brain.BrainTypeObsidian
	case "logseq":
		return brain.BrainTypeLogseq
	case "dendron":
		return brain.BrainTypeDendron
	case "foam":
		return brain.BrainTypeFoam
	default:
		return brain.BrainTypeFlip
	}
}

func parseTemplateType(s string) templates.TemplateType {
	switch s {
	case "note":
		return templates.TemplateTypeNote
	case "meeting":
		return templates.TemplateTypeMeeting
	case "journal":
		return templates.TemplateTypeJournal
	case "task":
		return templates.TemplateTypeTask
	default:
		return templates.TemplateTypeNote
	}
}

func templateJSONError(cmd *cobra.Command, msg string) error {
	return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]interface{}{
		"success": false,
		"error":   msg,
	})
}

func newVSCodeTemplateListCommand() *cobra.Command {
	var brainTypeStr string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List available templates (JSON)",
		RunE: func(cmd *cobra.Command, args []string) error {
			brainType := parseBrainType(brainTypeStr)
			templateTypes := []templates.TemplateType{
				templates.TemplateTypeNote,
				templates.TemplateTypeMeeting,
				templates.TemplateTypeJournal,
				templates.TemplateTypeTask,
			}

			var infos []TemplateInfo
			for _, tt := range templateTypes {
				content, _ := templates.Load(brainType, tt)
				path, _ := templates.GetTemplatePath(brainType, tt)
				defaultContent := templates.GetDefault(brainType, tt)

				infos = append(infos, TemplateInfo{
					BrainType:    brainTypeStr,
					TemplateType: string(tt),
					Content:      content,
					IsDefault:    content == defaultContent,
					Path:         path,
				})
			}

			resp := TemplateListResponse{
				Success:   true,
				Templates: infos,
			}
			return json.NewEncoder(cmd.OutOrStdout()).Encode(resp)
		},
	}

	cmd.Flags().StringVar(&brainTypeStr, "brain-type", "flip", "Brain type (flip, obsidian, logseq, dendron, foam)")

	return cmd
}

func newVSCodeTemplateGetCommand() *cobra.Command {
	var brainTypeStr string
	var templateTypeStr string

	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get template content (JSON)",
		RunE: func(cmd *cobra.Command, args []string) error {
			brainType := parseBrainType(brainTypeStr)
			templateType := parseTemplateType(templateTypeStr)

			content, err := templates.Load(brainType, templateType)
			if err != nil {
				return templateJSONError(cmd, "Failed to load template: "+err.Error())
			}

			path, _ := templates.GetTemplatePath(brainType, templateType)
			defaultContent := templates.GetDefault(brainType, templateType)

			info := TemplateInfo{
				BrainType:    brainTypeStr,
				TemplateType: templateTypeStr,
				Content:      content,
				IsDefault:    content == defaultContent,
				Path:         path,
			}

			return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]interface{}{
				"success":  true,
				"template": info,
			})
		},
	}

	cmd.Flags().StringVar(&brainTypeStr, "brain-type", "flip", "Brain type (flip, obsidian, logseq, dendron, foam)")
	cmd.Flags().StringVar(&templateTypeStr, "template-type", "note", "Template type (note, meeting, journal, task)")

	return cmd
}

func newVSCodeTemplateGetDefaultCommand() *cobra.Command {
	var brainTypeStr string
	var templateTypeStr string

	cmd := &cobra.Command{
		Use:   "get-default",
		Short: "Get default template content (JSON)",
		RunE: func(cmd *cobra.Command, args []string) error {
			brainType := parseBrainType(brainTypeStr)
			templateType := parseTemplateType(templateTypeStr)

			content := templates.GetDefault(brainType, templateType)

			info := TemplateInfo{
				BrainType:    brainTypeStr,
				TemplateType: templateTypeStr,
				Content:      content,
				IsDefault:    true,
			}

			return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]interface{}{
				"success":  true,
				"template": info,
			})
		},
	}

	cmd.Flags().StringVar(&brainTypeStr, "brain-type", "flip", "Brain type (flip, obsidian, logseq, dendron, foam)")
	cmd.Flags().StringVar(&templateTypeStr, "template-type", "note", "Template type (note, meeting, journal, task)")

	return cmd
}

func newVSCodeTemplateResetCommand() *cobra.Command {
	var brainTypeStr string
	var templateTypeStr string
	var all bool

	cmd := &cobra.Command{
		Use:   "reset",
		Short: "Reset template to default (JSON)",
		RunE: func(cmd *cobra.Command, args []string) error {
			brainType := parseBrainType(brainTypeStr)

			if all {
				// Reset all templates for this brain type
				if err := templates.ResetBrainTemplates(brainType); err != nil {
					return templateJSONError(cmd, "Failed to reset templates: "+err.Error())
				}

				return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]interface{}{
					"success": true,
					"message": fmt.Sprintf("All templates for %s reset to defaults", brainTypeStr),
				})
			}

			// Reset single template
			templateType := parseTemplateType(templateTypeStr)

			if err := templates.ResetTemplate(brainType, templateType); err != nil {
				return templateJSONError(cmd, "Failed to reset template: "+err.Error())
			}

			content := templates.GetDefault(brainType, templateType)
			path, _ := templates.GetTemplatePath(brainType, templateType)

			return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]interface{}{
				"success": true,
				"message": fmt.Sprintf("%s template for %s reset to default", templateTypeStr, brainTypeStr),
				"template": TemplateInfo{
					BrainType:    brainTypeStr,
					TemplateType: templateTypeStr,
					Content:      content,
					IsDefault:    true,
					Path:         path,
				},
			})
		},
	}

	cmd.Flags().StringVar(&brainTypeStr, "brain-type", "flip", "Brain type (flip, obsidian, logseq, dendron, foam)")
	cmd.Flags().StringVar(&templateTypeStr, "template-type", "note", "Template type (note, meeting, journal, task)")
	cmd.Flags().BoolVar(&all, "all", false, "Reset all templates for the brain type")

	return cmd
}
