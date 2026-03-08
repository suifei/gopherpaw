// Package agent provides hooks for skill-based agent enhancements.
package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/suifei/gopherpaw/internal/skills"
)

// SkillReminderHook creates a hook that reminds the agent about relevant skills
// before processing user messages. This hook analyzes the last user message
// and adds a skill reminder to the system message if relevant skills are found.
func SkillReminderHook(skillMgr *skills.Manager) Hook {
	return func(ctx context.Context, agent *ReactAgent, chatID string, messages []Message) ([]Message, error) {
		// Find the last user message
		var lastUserMsg string
		for i := len(messages) - 1; i >= 0; i-- {
			if messages[i].Role == "user" {
				lastUserMsg = messages[i].Content
				break
			}
		}

		if lastUserMsg == "" {
			return messages, nil
		}

		// Check if any skills match the user query
		matched := skillMgr.SelectSkills(lastUserMsg)
		if len(matched) == 0 {
			return messages, nil
		}

		// Build skill reminder
		var skillNames []string
		var skillPaths []string
		for _, s := range matched {
			skillNames = append(skillNames, s.Name)
			if s.Path != "" {
				skillPaths = append(skillPaths, s.Path)
			}
		}

		// Create reminder message
		reminder := fmt.Sprintf("\n\n⚠️ SKILL REMINDER: Your request may require a specialized skill.\n\n"+
			"Relevant skills detected: %s\n\n"+
			"To use these skills, call read_file with the skill path:\n",
			strings.Join(skillNames, ", "))

		for _, path := range skillPaths {
			reminder += fmt.Sprintf("  - read_file file_path=\"%s\"\n", path)
		}
		reminder += "\nPlease read the skill file first before proceeding with your task.\n"

		// Add reminder to the first system message
		for i := range messages {
			if messages[i].Role == "system" {
				messages[i].Content += reminder
				break
			}
		}

		return messages, nil
	}
}

// FileExtensionSkillHook creates a hook that detects document creation requests
// and reminds the agent to use the appropriate skill.
func FileExtensionSkillHook(skillMgr *skills.Manager) Hook {
	// Map file extensions to skill names
	extToSkill := map[string]string{
		".docx": "docx",
		".xlsx": "xlsx",
		".pdf":  "pdf",
		".pptx": "pptx",
	}

	return func(ctx context.Context, agent *ReactAgent, chatID string, messages []Message) ([]Message, error) {
		// Find the last user message
		var lastUserMsg string
		for i := len(messages) - 1; i >= 0; i-- {
			if messages[i].Role == "user" {
				lastUserMsg = messages[i].Content
				break
			}
		}

		if lastUserMsg == "" {
			return messages, nil
		}

		// Check if user message contains document creation keywords
		lowerMsg := strings.ToLower(lastUserMsg)
		docKeywords := []string{"word", "excel", "powerpoint", "pdf", "document", "spreadsheet", "presentation", "docx", "xlsx", "pptx"}

		hasDocKeyword := false
		for _, kw := range docKeywords {
			if strings.Contains(lowerMsg, kw) {
				hasDocKeyword = true
				break
			}
		}

		if !hasDocKeyword {
			return messages, nil
		}

		// Detect which document type is requested
		var detectedSkills []string
		for ext, skillName := range extToSkill {
			if strings.Contains(lowerMsg, ext) || strings.Contains(lowerMsg, skillName) {
				detectedSkills = append(detectedSkills, skillName)
			}
		}

		// If no specific format mentioned but document keywords exist, remind generally
		if len(detectedSkills) == 0 && hasDocKeyword {
			detectedSkills = []string{"docx", "xlsx", "pdf", "pptx"}
		}

		// Build reminder
		var reminder string
		if len(detectedSkills) > 0 {
			reminder = fmt.Sprintf("\n\n⚠️ DOCUMENT SKILL REMINDER: You're creating a document.\n\n"+
				"Before using write_file, you MUST read the relevant skill file(s):\n\n")
			for _, skill := range detectedSkills {
				reminder += fmt.Sprintf("  read_file file_path=\"configs/active_skills/%s/SKILL.md\"\n", skill)
			}
			reminder += "\nThese skills provide proper document generation capabilities.\n"
		}

		if reminder != "" {
			// Add reminder to the first system message
			for i := range messages {
				if messages[i].Role == "system" {
					messages[i].Content += reminder
					break
				}
			}
		}

		return messages, nil
	}
}
