import * as vscode from 'vscode';
import { getFlipClient } from '../flip-client';

// Template types
interface TemplateInfo {
    brain_type: string;
    template_type: string;
    content: string;
    is_default: boolean;
    path?: string;
}

interface TemplateListResponse {
    success: boolean;
    templates: TemplateInfo[];
    error?: string;
}

interface TemplateGetResponse {
    success: boolean;
    template: TemplateInfo;
    error?: string;
}

interface TemplateResetResponse {
    success: boolean;
    message?: string;
    template?: TemplateInfo;
    error?: string;
}

// Brain type labels
const BRAIN_TYPES = [
    { label: '$(brain) Flip', value: 'flip' },
    { label: '$(note) Obsidian', value: 'obsidian' },
    { label: '$(list-tree) Logseq', value: 'logseq' },
    { label: '$(symbol-namespace) Dendron', value: 'dendron' },
    { label: '$(beaker) Foam', value: 'foam' }
];

// Template type labels
const TEMPLATE_TYPES = [
    { label: '$(note) Note', value: 'note', description: 'Regular notes' },
    { label: '$(calendar) Meeting', value: 'meeting', description: 'Meeting notes' },
    { label: '$(book) Journal', value: 'journal', description: 'Daily journal entries' },
    { label: '$(checklist) Task', value: 'task', description: 'Task notes' }
];

/**
 * Main templates command - shows template management menu
 */
export async function templateCommand(): Promise<void> {
    const actions = [
        { label: '$(list-unordered) List Templates', description: 'View all templates for a brain type', action: 'list' },
        { label: '$(eye) View Template', description: 'View template content', action: 'view' },
        { label: '$(edit) Edit Template', description: 'Edit template in editor', action: 'edit' },
        { label: '$(refresh) Reset Template', description: 'Reset template to default', action: 'reset' },
        { label: '$(folder-opened) Open Template Directory', description: 'Open templates folder', action: 'open' }
    ];

    const selected = await vscode.window.showQuickPick(actions, {
        placeHolder: 'Select template action',
        title: 'Flip: Template Management'
    });

    if (!selected) {
        return;
    }

    switch (selected.action) {
        case 'list':
            await listTemplatesCommand();
            break;
        case 'view':
            await viewTemplateCommand();
            break;
        case 'edit':
            await editTemplateCommand();
            break;
        case 'reset':
            await resetTemplateCommand();
            break;
        case 'open':
            await openTemplateDirectoryCommand();
            break;
    }
}

/**
 * List all templates for a brain type
 */
export async function listTemplatesCommand(): Promise<void> {
    // Select brain type
    const brainType = await vscode.window.showQuickPick(BRAIN_TYPES, {
        placeHolder: 'Select brain type',
        title: 'List Templates'
    });

    if (!brainType) {
        return;
    }

    const client = getFlipClient();
    const result = await client.listTemplates(brainType.value);

    if (!result.success || !result.data?.templates) {
        vscode.window.showErrorMessage(`Failed to list templates: ${result.error}`);
        return;
    }

    // Show templates in QuickPick
    const items = result.data.templates.map((t: TemplateInfo) => ({
        label: `${t.is_default ? '$(check)' : '$(edit)'} ${t.template_type}`,
        description: t.is_default ? 'Using default' : 'Customized',
        detail: t.path,
        template: t
    }));

    const selected = await vscode.window.showQuickPick(items, {
        placeHolder: 'Select template to view',
        title: `Templates for ${brainType.label}`
    });

    if (selected) {
        await showTemplateContent(selected.template);
    }
}

/**
 * View a specific template
 */
export async function viewTemplateCommand(): Promise<void> {
    const { brainType, templateType } = await selectBrainAndTemplateType();
    if (!brainType || !templateType) {
        return;
    }

    const client = getFlipClient();
    const result = await client.getTemplate(brainType.value, templateType.value);

    if (!result.success || !result.data?.template) {
        vscode.window.showErrorMessage(`Failed to get template: ${result.error}`);
        return;
    }

    await showTemplateContent(result.data.template);
}

/**
 * Edit a template in the editor
 */
export async function editTemplateCommand(): Promise<void> {
    const { brainType, templateType } = await selectBrainAndTemplateType();
    if (!brainType || !templateType) {
        return;
    }

    const client = getFlipClient();
    const result = await client.getTemplate(brainType.value, templateType.value);

    if (!result.success || !result.data?.template) {
        vscode.window.showErrorMessage(`Failed to get template: ${result.error}`);
        return;
    }

    const templatePath = result.data.template.path;
    if (templatePath) {
        const doc = await vscode.workspace.openTextDocument(templatePath);
        await vscode.window.showTextDocument(doc);
        
        vscode.window.showInformationMessage(
            'Template placeholders: {{title}}, {{date}}, {{time}}, {{tags}}, {{participants}}, {{id}}'
        );
    } else {
        vscode.window.showErrorMessage('Template path not found');
    }
}

/**
 * Reset a template to its default
 */
export async function resetTemplateCommand(): Promise<void> {
    // Ask if reset single or all
    const scope = await vscode.window.showQuickPick([
        { label: '$(file) Single Template', value: 'single' },
        { label: '$(files) All Templates for Brain Type', value: 'all' }
    ], {
        placeHolder: 'What would you like to reset?',
        title: 'Reset Templates'
    });

    if (!scope) {
        return;
    }

    // Select brain type
    const brainType = await vscode.window.showQuickPick(BRAIN_TYPES, {
        placeHolder: 'Select brain type',
        title: 'Reset Templates'
    });

    if (!brainType) {
        return;
    }

    const client = getFlipClient();

    if (scope.value === 'all') {
        // Confirm reset all
        const confirm = await vscode.window.showWarningMessage(
            `Reset ALL templates for ${brainType.label} to defaults?`,
            { modal: true },
            'Reset All'
        );

        if (confirm !== 'Reset All') {
            return;
        }

        const result = await client.resetTemplates(brainType.value, undefined, true);

        if (!result.success) {
            vscode.window.showErrorMessage(`Failed to reset templates: ${result.error}`);
            return;
        }

        vscode.window.showInformationMessage(result.data?.message || 'Templates reset successfully');
    } else {
        // Select template type
        const templateType = await vscode.window.showQuickPick(TEMPLATE_TYPES, {
            placeHolder: 'Select template type to reset',
            title: 'Reset Template'
        });

        if (!templateType) {
            return;
        }

        // Confirm reset single
        const confirm = await vscode.window.showWarningMessage(
            `Reset ${templateType.label} template for ${brainType.label} to default?`,
            { modal: true },
            'Reset'
        );

        if (confirm !== 'Reset') {
            return;
        }

        const result = await client.resetTemplates(brainType.value, templateType.value, false);

        if (!result.success) {
            vscode.window.showErrorMessage(`Failed to reset template: ${result.error}`);
            return;
        }

        vscode.window.showInformationMessage(result.data?.message || 'Template reset successfully');
    }
}

/**
 * Open the template directory
 */
export async function openTemplateDirectoryCommand(): Promise<void> {
    const client = getFlipClient();
    const result = await client.listTemplates('flip');

    if (!result.success || !result.data?.templates?.length) {
        vscode.window.showErrorMessage('Failed to get template directory');
        return;
    }

    // Extract directory from path
    const firstTemplatePath = result.data.templates[0].path;
    if (firstTemplatePath) {
        const templateDir = firstTemplatePath.substring(0, firstTemplatePath.lastIndexOf('/'));
        const parentDir = templateDir.substring(0, templateDir.lastIndexOf('/'));
        
        await vscode.commands.executeCommand('revealFileInOS', vscode.Uri.file(parentDir));
    }
}

// Helper functions

async function selectBrainAndTemplateType(): Promise<{ brainType: typeof BRAIN_TYPES[0] | undefined, templateType: typeof TEMPLATE_TYPES[0] | undefined }> {
    const brainType = await vscode.window.showQuickPick(BRAIN_TYPES, {
        placeHolder: 'Select brain type',
        title: 'Select Template'
    });

    if (!brainType) {
        return { brainType: undefined, templateType: undefined };
    }

    const templateType = await vscode.window.showQuickPick(TEMPLATE_TYPES, {
        placeHolder: 'Select template type',
        title: 'Select Template'
    });

    return { brainType, templateType };
}

async function showTemplateContent(template: TemplateInfo): Promise<void> {
    // Create a virtual document to show template content
    const content = [
        `# Template: ${template.template_type} (${template.brain_type})`,
        ``,
        `**Status:** ${template.is_default ? 'Default' : 'Customized'}`,
        template.path ? `**Path:** ${template.path}` : '',
        ``,
        `## Content`,
        '```markdown',
        template.content,
        '```',
        ``,
        `## Available Placeholders`,
        `- \`{{title}}\` - Note title`,
        `- \`{{date}}\` - Current date (YYYY-MM-DD)`,
        `- \`{{time}}\` - Current time (HH:MM)`,
        `- \`{{weekday}}\` - Day of week`,
        `- \`{{tags}}\` - Tags placeholder`,
        `- \`{{participants}}\` - Meeting participants`,
        `- \`{{id}}\` - Unique ID`,
        `- \`{{created}}\` - Creation timestamp`,
        `- \`{{updated}}\` - Last update timestamp`
    ].filter(Boolean).join('\n');

    const doc = await vscode.workspace.openTextDocument({
        content,
        language: 'markdown'
    });
    await vscode.window.showTextDocument(doc, { preview: true });
}
