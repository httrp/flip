# AI Features & Architecture ( flip ai )

This document outlines the architecture and workflows for AI integration within flip, both for the CLI and the VS Code Extension. As of March 2026, flip contains deep integrations for features like intelligent notes generation, summarizing, research, and prompt templates.

## 1. Automatic Setup Validation
To prevent unhandled errors when executing AI commands without prior configuration:
- A global `EnsureAISetup()` validation gate triggers at the start of any AI workflow (e.g., `flip ai research`).
- **CLI behavior:** Prompts the user interactively (using `promptui`) asking if they want to run `flip ai setup`.
- **VS Code behavior:** Catches the special `AI_SETUP_REQUIRED` error string returned via JSON stdout. The extension gracefully hooks this error and automatically triggers the native `flip.aiSetup` command palette form.

## 2. Prompt Template Selection & Variants
Flip leverages the `templates/ai/<command>` directory internally and in local brains to manage prompt schemas.
- **No Silent Defaults:** In interactive environments (Terminal UI & VS Code QuickPick), standard templates (marked as `default`) do not implicitly bypass user selection anymore. The UI will prominently list the Default template (e.g. marked with a ⭐ in VS Code) at the top but allows the user to diverge and pick a variant like "Research (Deep Dive)" or "Research (Quick & Dirty)".
- **Headless mode:** Only non-interactive automated calls automatically bypass UI and load the default template.

## 3. On-The-Fly Generation
- When no template exists, or the user selects "Create new template", the workflow will auto-scaffold a markdown template within the active `.flip-brain/templates/ai/`.
- This scaffolding is populated using `TemplateTypePrompt` metadata boilerplate, enforcing structured prompt sections (`## Rolle` and `## Aufgabe`).

## 4. Native Journal Integration
Content retrieved from LLMs is not isolated.
- The output from AI commands seamlessly pipes through flip's regular `NoteOptions` lifecycle. 
- AI notes are physically written to the `notes/` directory.
- Notes are correctly hyperlinked to the **daily journal** using the native `AddLinkToJournal` capability.

## 5. Asynchronous Long-Running Notifications
AI completions are often long-running context streams:
- **CLI:** Handled directly via continuous stdout streaming or visual terminal spinners depending on the subcommand.
- **VS Code:** AI jobs run under `vscode.window.withProgress`, offering a non-blocking notification bar indicator. Upon completion, a toast (`showInformationMessage("✨ AI note generated successfully!")`) actively informs the user before the generated markdown file is previewed in the active editor.
