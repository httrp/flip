package commands

// Icon variables (themeable). Defaults to ASCII; use SetIconTheme to change.
var (
	IconCheck      = "[OK]"
	IconError      = "[X]"
	IconActive     = "*"
	IconDefault    = "#"
	IconBrain      = "+"
	IconWorkspace  = "~"
	IconInfo       = "i"
	IconWarning    = "!"
	IconQuestion   = "?"
	IconArrow      = ">"
	IconSuccess    = "+"
	IconTree       = "|"
	IconTreeBranch = "+-"
	IconTreeLast   = "`-"
	IconTreeSpace  = "  "
	IconTreeLine   = "| "
)

// SetIconTheme switches icon set. Supported: "ascii", "emoji", "mixed" (few emojis for key items).
func SetIconTheme(theme string) {
	switch theme {
	case "emoji":
		IconCheck = "✅"
		IconError = "❌"
		IconActive = "🎯"
		IconDefault = "⭐"
		IconBrain = "🧠"
		IconWorkspace = "🌌"
		IconInfo = "ℹ️"
		IconWarning = "⚠️"
		IconQuestion = "?"
		IconArrow = "➜"
		IconSuccess = "✅"
		IconTree = "|"
		IconTreeBranch = "+-"
		IconTreeLast = "`-"
		IconTreeSpace = "  "
		IconTreeLine = "| "
	case "mixed":
		// Keep most ASCII; use emoji for key concepts and states
		IconCheck = "✅"
		IconError = "❌"
		IconActive = "*" // subtle for layout
		IconDefault = "⭐"
		IconBrain = "🧠"
		IconWorkspace = "🌌"
		IconInfo = "i"
		IconWarning = "!"
		IconQuestion = "?"
		IconArrow = ">"
		IconSuccess = "[OK]"
		IconTree = "|"
		IconTreeBranch = "+-"
		IconTreeLast = "`-"
		IconTreeSpace = "  "
		IconTreeLine = "| "
	default: // ascii
		IconCheck = "[OK]"
		IconError = "[X]"
		IconActive = "*"
		IconDefault = "#"
		IconBrain = "+"
		IconWorkspace = "~"
		IconInfo = "i"
		IconWarning = "!"
		IconQuestion = "?"
		IconArrow = ">"
		IconSuccess = "+"
		IconTree = "|"
		IconTreeBranch = "+-"
		IconTreeLast = "`-"
		IconTreeSpace = "  "
		IconTreeLine = "| "
	}
}
