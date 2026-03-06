package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/suifei/gopherpaw/internal/config"
	"github.com/suifei/gopherpaw/internal/skills"
)

var skillsCmd = &cobra.Command{
	Use:   "skills",
	Short: "Skill management",
	Long:  "list, config, import - list, configure, or import skills from URL",
}

var skillsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List loaded skills",
	RunE:  runSkillsList,
}

var skillsConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Show skill config",
	RunE:  runSkillsConfig,
}

var skillsImportCmd = &cobra.Command{
	Use:   "import [URL]",
	Short: "Import SKILL.md from URL (e.g. raw.githubusercontent.com)",
	Args:  cobra.ExactArgs(1),
	RunE:  runSkillsImport,
}

var skillsEnableCmd = &cobra.Command{
	Use:   "enable [name]",
	Short: "Enable a skill by name",
	Args:  cobra.ExactArgs(1),
	RunE:  runSkillsEnable,
}

var skillsDisableCmd = &cobra.Command{
	Use:   "disable [name]",
	Short: "Disable a skill by name",
	Args:  cobra.ExactArgs(1),
	RunE:  runSkillsDisable,
}

var skillsCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a new skill in customized_skills directory",
	Args:  cobra.ExactArgs(1),
	RunE:  runSkillsCreate,
}

var skillsDeleteCmd = &cobra.Command{
	Use:   "delete [name]",
	Short: "Delete a skill from customized_skills directory",
	Args:  cobra.ExactArgs(1),
	RunE:  runSkillsDelete,
}

func init() {
	skillsCmd.AddCommand(skillsListCmd)
	skillsCmd.AddCommand(skillsConfigCmd)
	skillsCmd.AddCommand(skillsImportCmd)
	skillsCmd.AddCommand(skillsEnableCmd)
	skillsCmd.AddCommand(skillsDisableCmd)
	skillsCmd.AddCommand(skillsCreateCmd)
	skillsCmd.AddCommand(skillsDeleteCmd)
}

func runSkillsList(cmd *cobra.Command, args []string) error {
	cfgPath, _ := cmd.Root().PersistentFlags().GetString("config")
	if cfgPath == "" {
		cfgPath = "configs/config.yaml"
	}
	if p := os.Getenv("GOPHERPAW_CONFIG"); p != "" {
		cfgPath = p
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return err
	}
	workingDir := config.ResolveWorkingDir(cfg.WorkingDir)
	configDir := filepath.Dir(cfgPath)
	mgr := skills.NewManager()
	if err := mgr.LoadSkills(workingDir, configDir, cfg.Skills); err != nil {
		return err
	}
	enabled := mgr.GetEnabledSkills()
	if len(enabled) == 0 {
		fmt.Println("无已加载的 Skills")
		fmt.Printf("请在 %s 或 %s 下创建 active_skills/<name>/SKILL.md\n", workingDir, configDir)
		return nil
	}
	for _, s := range enabled {
		fmt.Printf("- %s: %s\n", s.Name, s.Description)
	}
	return nil
}

func runSkillsConfig(cmd *cobra.Command, args []string) error {
	cfgPath, _ := cmd.Root().PersistentFlags().GetString("config")
	if cfgPath == "" {
		cfgPath = "configs/config.yaml"
	}
	if p := os.Getenv("GOPHERPAW_CONFIG"); p != "" {
		cfgPath = p
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return err
	}
	fmt.Printf("active_dir: %s\n", cfg.Skills.ActiveDir)
	fmt.Printf("customized_dir: %s\n", cfg.Skills.CustomizedDir)
	fmt.Printf("working_dir: %s\n", config.ResolveWorkingDir(cfg.WorkingDir))
	return nil
}

func runSkillsImport(cmd *cobra.Command, args []string) error {
	cfgPath, _ := cmd.Root().PersistentFlags().GetString("config")
	if cfgPath == "" {
		cfgPath = "configs/config.yaml"
	}
	if p := os.Getenv("GOPHERPAW_CONFIG"); p != "" {
		cfgPath = p
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return err
	}
	workingDir := config.ResolveWorkingDir(cfg.WorkingDir)
	configDir := filepath.Dir(cfgPath)
	mgr := skills.NewManager()
	if err := mgr.LoadSkills(workingDir, configDir, cfg.Skills); err != nil {
		return err
	}
	name, err := mgr.ImportFromURL(context.Background(), args[0], workingDir, cfg.Skills)
	if err != nil {
		return fmt.Errorf("import: %w", err)
	}
	fmt.Printf("已导入 skill: %s\n", name)
	return nil
}

func runSkillsEnable(cmd *cobra.Command, args []string) error {
	cfgPath, _ := cmd.Root().PersistentFlags().GetString("config")
	if cfgPath == "" {
		cfgPath = "configs/config.yaml"
	}
	if p := os.Getenv("GOPHERPAW_CONFIG"); p != "" {
		cfgPath = p
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return err
	}
	workingDir := config.ResolveWorkingDir(cfg.WorkingDir)
	configDir := filepath.Dir(cfgPath)
	mgr := skills.NewManager()
	if err := mgr.LoadSkills(workingDir, configDir, cfg.Skills); err != nil {
		return err
	}
	if err := mgr.EnableSkill(args[0]); err != nil {
		return err
	}
	fmt.Printf("已启用 skill: %s\n", args[0])
	return nil
}

func runSkillsDisable(cmd *cobra.Command, args []string) error {
	cfgPath, _ := cmd.Root().PersistentFlags().GetString("config")
	if cfgPath == "" {
		cfgPath = "configs/config.yaml"
	}
	if p := os.Getenv("GOPHERPAW_CONFIG"); p != "" {
		cfgPath = p
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return err
	}
	workingDir := config.ResolveWorkingDir(cfg.WorkingDir)
	configDir := filepath.Dir(cfgPath)
	mgr := skills.NewManager()
	if err := mgr.LoadSkills(workingDir, configDir, cfg.Skills); err != nil {
		return err
	}
	if err := mgr.DisableSkill(args[0]); err != nil {
		return err
	}
	fmt.Printf("已禁用 skill: %s\n", args[0])
	return nil
}

func runSkillsCreate(cmd *cobra.Command, args []string) error {
	cfgPath, _ := cmd.Root().PersistentFlags().GetString("config")
	if cfgPath == "" {
		cfgPath = "configs/config.yaml"
	}
	if p := os.Getenv("GOPHERPAW_CONFIG"); p != "" {
		cfgPath = p
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return err
	}
	workingDir := config.ResolveWorkingDir(cfg.WorkingDir)

	name := args[0]
	skillDir := filepath.Join(workingDir, cfg.Skills.CustomizedDir, name)
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		return fmt.Errorf("create skill dir: %w", err)
	}

	skillMd := fmt.Sprintf(`---
name: %s
description: Custom skill
---

# %s

TODO: Add skill description and instructions here.
`, name, name)

	skillPath := filepath.Join(skillDir, "SKILL.md")
	if err := os.WriteFile(skillPath, []byte(skillMd), 0644); err != nil {
		return fmt.Errorf("write SKILL.md: %w", err)
	}

	fmt.Printf("已创建 skill: %s\n", skillPath)
	fmt.Printf("请编辑 %s 添加技能内容\n", skillPath)
	return nil
}

func runSkillsDelete(cmd *cobra.Command, args []string) error {
	cfgPath, _ := cmd.Root().PersistentFlags().GetString("config")
	if cfgPath == "" {
		cfgPath = "configs/config.yaml"
	}
	if p := os.Getenv("GOPHERPAW_CONFIG"); p != "" {
		cfgPath = p
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return err
	}
	workingDir := config.ResolveWorkingDir(cfg.WorkingDir)

	name := args[0]
	skillDir := filepath.Join(workingDir, cfg.Skills.CustomizedDir, name)

	// Check if skill exists
	if _, err := os.Stat(skillDir); os.IsNotExist(err) {
		return fmt.Errorf("skill %q not found in customized_skills", name)
	}

	// Remove skill directory
	if err := os.RemoveAll(skillDir); err != nil {
		return fmt.Errorf("delete skill: %w", err)
	}

	fmt.Printf("已删除 skill: %s\n", name)
	return nil
}
