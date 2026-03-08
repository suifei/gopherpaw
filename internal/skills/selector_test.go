// Package skills provides tests for skill selection.
package skills

import (
	"testing"
)

func TestManager_SelectSkills(t *testing.T) {
	mgr := NewManager()
	mgr.skills = map[string]*Skill{
		"docx": {
			Name:        "docx",
			Description: "Word document processing",
			Keywords:    []string{"word", "docx", "document", "报告", "文档"},
			Enabled:     true,
		},
		"pptx": {
			Name:        "pptx",
			Description: "PowerPoint presentation processing",
			Keywords:    []string{"powerpoint", "pptx", "presentation", "演示", "幻灯片"},
			Enabled:     true,
		},
		"xlsx": {
			Name:        "xlsx",
			Description: "Excel spreadsheet processing",
			Keywords:    []string{"excel", "xlsx", "spreadsheet", "表格", "电子表"},
			Enabled:     true,
		},
		"pdf": {
			Name:        "pdf",
			Description: "PDF document processing",
			Keywords:    []string{"pdf", "adobe", "acrobat", "提取", "转换"},
			Enabled:     false, // Disabled
		},
	}

	tests := []struct {
		name           string
		query          string
		expectedSkills []string
	}{
		{
			name:           "empty query returns all enabled skills",
			query:          "",
			expectedSkills: []string{"docx", "pptx", "xlsx"},
		},
		{
			name:           "direct name match",
			query:          "create a docx file",
			expectedSkills: []string{"docx"},
		},
		{
			name:           "keyword match",
			query:          "生成一份报告",
			expectedSkills: []string{"docx"},
		},
		{
			name:           "description word match",
			query:          "process a word document",
			expectedSkills: []string{"docx"},
		},
		{
			name:           "multiple matches",
			query:          "create office document and presentation",
			expectedSkills: []string{"docx", "pptx"},
		},
		{
			name:           "chinese keyword match",
			query:          "制作幻灯片",
			expectedSkills: []string{"pptx"},
		},
		{
			name:           "spreadsheet match",
			query:          "create excel spreadsheet",
			expectedSkills: []string{"xlsx"},
		},
		{
			name:           "disabled skill not returned - use unique keyword",
			query:          "使用pdf2image转换PDF",
			expectedSkills: nil, // pdf is disabled
		},
		{
			name:           "no match returns empty",
			query:          "write python code",
			expectedSkills: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := mgr.SelectSkills(tt.query)

			if len(results) != len(tt.expectedSkills) {
				t.Errorf("expected %d skills, got %d", len(tt.expectedSkills), len(results))
				return
			}

			resultNames := make(map[string]bool)
			for _, r := range results {
				resultNames[r.Name] = true
			}

			for _, expected := range tt.expectedSkills {
				if !resultNames[expected] {
					t.Errorf("expected skill %q not found in results", expected)
				}
			}
		})
	}
}

func TestManager_GetRelevantSkillsContent(t *testing.T) {
	mgr := NewManager()
	mgr.skills = map[string]*Skill{
		"docx": {
			Name:        "docx",
			Description: "Word document processing",
			Keywords:    []string{"word", "docx", "document"},
			Enabled:     true,
			Content:     "DOCX content here",
		},
		"pptx": {
			Name:        "pptx",
			Description: "PowerPoint processing",
			Keywords:    []string{"pptx", "presentation"},
			Enabled:     true,
			Content:     "PPTX content here",
		},
	}

	t.Run("matching query returns relevant content", func(t *testing.T) {
		content := mgr.GetRelevantSkillsContent("create a word docx document")
		if content == "" {
			t.Error("expected non-empty content")
		}
		// Should contain docx content but not pptx
		if !contains(content, "docx") && !contains(content, "DOCX") {
			t.Error("expected docx content to be included")
		}
	})

	t.Run("no match returns all skills", func(t *testing.T) {
		content := mgr.GetRelevantSkillsContent("write python code")
		// Should return all skills as fallback
		if content == "" {
			t.Error("expected non-empty content (fallback to all skills)")
		}
	})
}

func TestManager_DynamicQuery(t *testing.T) {
	mgr := NewManager()
	mgr.skills = map[string]*Skill{
		"docx": {
			Name:        "docx",
			Description: "Word document processing",
			Keywords:    []string{"word", "docx", "document"},
			Enabled:     true,
			Content:     "DOCX content",
		},
	}

	t.Run("SetQuery and GetDynamicSystemPromptAddition", func(t *testing.T) {
		mgr.SetQuery("create a word document")
		content := mgr.GetDynamicSystemPromptAddition()

		if content == "" {
			t.Error("expected non-empty content")
		}
		if !contains(content, "DOCX") {
			t.Error("expected DOCX content to be included")
		}
	})

	t.Run("ClearQuery returns all skills", func(t *testing.T) {
		mgr.SetQuery("create a word document")
		mgr.ClearQuery()
		content := mgr.GetDynamicSystemPromptAddition()

		// Should return all skills when query is cleared
		if content == "" {
			t.Error("expected non-empty content")
		}
	})
}

func TestSelector_SelectByScore(t *testing.T) {
	skills := map[string]*Skill{
		"docx": {
			Name:        "docx",
			Description: "Word document processing",
			Keywords:    []string{"word", "docx"},
			Enabled:     true,
			Content:     "DOCX content",
		},
		"pptx": {
			Name:        "pptx",
			Description: "PowerPoint processing",
			Keywords:    []string{"pptx", "presentation"},
			Enabled:     true,
			Content:     "PPTX content",
		},
	}

	sel := NewSelector(skills)

	t.Run("exact name match gets highest score", func(t *testing.T) {
		results := sel.SelectByScore("docx", 0.1)

		if len(results) == 0 {
			t.Fatal("expected at least one result")
		}

		// First result should be docx with highest score
		if results[0].Skill.Name != "docx" {
			t.Errorf("expected first result to be docx, got %s", results[0].Skill.Name)
		}

		if results[0].Score < 0.9 {
			t.Errorf("expected high score for exact match, got %f", results[0].Score)
		}
	})

	t.Run("no results below threshold", func(t *testing.T) {
		results := sel.SelectByScore("random unrelated query", 0.5)

		if len(results) != 0 {
			t.Errorf("expected no results for unrelated query, got %d", len(results))
		}
	})
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || indexOf(s, substr) >= 0))
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
