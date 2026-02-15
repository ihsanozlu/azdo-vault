package internal

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type VariableGroupList struct {
	Value []VariableGroup `json:"value"`
}

type VariableGroup struct {
	Id          int                 `json:"id,omitempty"`
	Name        string              `json:"name"`
	Description string              `json:"description,omitempty"`
	Type        string              `json:"type,omitempty"`
	Variables   map[string]Variable `json:"variables"`
}

type Variable struct {
	Value    string `json:"value,omitempty"`
	IsSecret bool   `json:"isSecret,omitempty"`
}

func ListVariableGroups(orgURL, project string) ([]VariableGroup, error) {

	cmd := exec.Command(
		"az", "pipelines", "variable-group", "list",
		"--organization", orgURL,
		"--project", project,
		"--output", "json",
	)

	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var groups []VariableGroup
	err = json.Unmarshal(out, &groups)
	return groups, err
}

func GetVariableGroup(orgURL, project string, id int) (*VariableGroup, error) {

	cmd := exec.Command(
		"az", "pipelines", "variable-group", "show",
		"--group-id", fmt.Sprintf("%d", id),
		"--organization", orgURL,
		"--project", project,
		"--output", "json",
	)

	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var group VariableGroup
	err = json.Unmarshal(out, &group)
	return &group, err
}

func CreateVariableGroup(
	orgURL,
	project,
	name string,
	variables map[string]Variable,
) (int, error) {

	args := []string{
		"pipelines", "variable-group", "create",
		"--name", name,
		"--organization", orgURL,
		"--project", project,
		"--authorize", "true",
	}

	// Add non-secret variables
	varPairs := []string{}
	for k, v := range variables {
		if !v.IsSecret {
			varPairs = append(varPairs, fmt.Sprintf("%s=%s", k, v.Value))
		}
	}

	if len(varPairs) == 0 {
		// Azure requires at least one variable
		varPairs = append(varPairs, "temp=placeholder")
	}

	args = append(args, "--variables")
	args = append(args, varPairs...)

	args = append(args, "--output", "json")

	cmd := exec.Command("az", args...)

	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println("AZ ERROR:", string(out))
		return 0, err
	}

	var result VariableGroup
	err = json.Unmarshal(out, &result)

	return result.Id, err
}

func AddVariableToGroup(orgURL, project string, groupID int, name, value string, isSecret bool) error {

	args := []string{
		"pipelines", "variable-group", "variable", "create",
		"--group-id", fmt.Sprintf("%d", groupID),
		"--name", name,
		"--organization", orgURL,
		"--project", project,
	}

	if isSecret {
		args = append(args, "--secret", "true")
	} else {
		args = append(args, "--value", value)
	}

	cmd := exec.Command("az", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func BackupVariableGroups(orgURL, project, backupPath string, selectedGroups []string) error {

	groups, err := ListVariableGroups(orgURL, project)
	if err != nil {
		return err
	}
	backupAll := len(selectedGroups) == 1 && selectedGroups[0] == "all"

	if len(groups) == 0 {
		fmt.Println("No variable groups found")
		return nil
	}

	err = os.MkdirAll(backupPath, 0755)
	if err != nil {
		return err
	}

	for _, g := range groups {

		if !backupAll && !contains(selectedGroups, g.Name) {
			continue
		}

		fullGroup, err := GetVariableGroup(orgURL, project, g.Id)
		if err != nil {
			return err
		}

		filePath := filepath.Join(backupPath, g.Name+".json")

		data, err := json.MarshalIndent(fullGroup, "", "  ")
		if err != nil {
			return err
		}

		err = os.WriteFile(filePath, data, 0644)
		if err != nil {
			return err
		}

		fmt.Println("✔ Backed up:", g.Name)
	}

	return nil
}

func contains(list []string, item string) bool {
	for _, v := range list {
		if v == item {
			return true
		}
	}
	return false
}

func RestoreVariableGroupsFromBackup(
	targetOrgURL,
	targetProject,
	backupPath string,
	selectedGroups []string,
) error {

	files, err := os.ReadDir(backupPath)
	if err != nil {
		return err
	}

	restoreAll := len(selectedGroups) == 1 && selectedGroups[0] == "all"

	for _, f := range files {

		if !strings.HasSuffix(f.Name(), ".json") {
			continue
		}

		groupName := strings.TrimSuffix(f.Name(), ".json")

		if !restoreAll && !contains(selectedGroups, groupName) {
			continue
		}

		filePath := filepath.Join(backupPath, f.Name())

		data, err := os.ReadFile(filePath)
		if err != nil {
			return err
		}

		var group VariableGroup
		err = json.Unmarshal(data, &group)
		if err != nil {
			return err
		}

		existing, err := FindVariableGroupByName(targetOrgURL, targetProject, group.Name)
		if err != nil {
			return err
		}

		var groupID int

		if existing != nil {
			fmt.Println("Updating existing group:", group.Name)
			groupID = existing.Id
		} else {
			fmt.Println("Creating group:", group.Name)

			groupID, err = CreateVariableGroup(
				targetOrgURL,
				targetProject,
				group.Name,
				group.Variables,
			)
			if err != nil {
				return err
			}

			continue
		}

		for name, variable := range group.Variables {

			if variable.IsSecret {
				fmt.Println("⚠ Skipping secret:", name)
				continue
			}

			err := UpdateVariableInGroup(
				targetOrgURL,
				targetProject,
				groupID,
				name,
				variable.Value,
			)

			if err != nil {
				err = AddVariableToGroup(
					targetOrgURL,
					targetProject,
					groupID,
					name,
					variable.Value,
					false,
				)
				if err != nil {
					return err
				}
			}
		}
	}

	fmt.Println("✔ Variable groups restored")
	return nil
}

func FindVariableGroupByName(orgURL, project, name string) (*VariableGroup, error) {
	groups, err := ListVariableGroups(orgURL, project)
	if err != nil {
		return nil, err
	}

	for _, g := range groups {
		if g.Name == name {
			return &g, nil
		}
	}

	return nil, nil
}

func UpdateVariableInGroup(orgURL, project string, groupID int, name, value string) error {

	args := []string{
		"pipelines", "variable-group", "variable", "update",
		"--group-id", fmt.Sprintf("%d", groupID),
		"--name", name,
		"--value", value,
		"--organization", orgURL,
		"--project", project,
	}

	cmd := exec.Command("az", args...)
	return cmd.Run()
}
