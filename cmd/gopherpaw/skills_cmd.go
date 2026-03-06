package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/suifei/gopherpaw/internal/config"
	"github.com/suifei/gopherpaw/internal/skills"
	"github.com/spf13/cobra"
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

func init() {
	skillsCmd.AddCommand(skillsListCmd)
	skillsCmd.AddCommand(skillsConfigCmd)
	skillsCmd.AddCommand(skillsImportCmd)
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
