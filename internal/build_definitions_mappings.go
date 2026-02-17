package internal

import (
	"fmt"
	"strings"
)

// BuildEndpointMaps uses your existing service_connections.go implementation.
func BuildEndpointMaps(
	sourceOrgURL, sourceProject,
	targetOrgURL, targetProject,
	resourceGUID string,
) (map[string]string, map[string]string, error) {

	src, err := ListServiceConnections(sourceOrgURL, sourceProject, resourceGUID)
	if err != nil {
		return nil, nil, fmt.Errorf("list source service connections failed: %w", err)
	}
	tgt, err := ListServiceConnections(targetOrgURL, targetProject, resourceGUID)
	if err != nil {
		return nil, nil, fmt.Errorf("list target service connections failed: %w", err)
	}

	srcIDToName := map[string]string{}
	for _, e := range src {
		if e.Id != "" && e.Name != "" {
			srcIDToName[strings.ToLower(e.Id)] = e.Name
		}
	}

	tgtNameToID := map[string]string{}
	for _, e := range tgt {
		if e.Id != "" && e.Name != "" {
			tgtNameToID[e.Name] = e.Id
		}
	}

	return srcIDToName, tgtNameToID, nil
}

func BuildVarGroupMaps(
	sourceOrgURL, sourceProject,
	targetOrgURL, targetProject string,
) (map[int]string, map[string]int, error) {

	src, err := ListVariableGroups(sourceOrgURL, sourceProject)
	if err != nil {
		return nil, nil, err
	}
	tgt, err := ListVariableGroups(targetOrgURL, targetProject)
	if err != nil {
		return nil, nil, err
	}

	srcIDToName := map[int]string{}
	for _, g := range src {
		if g.Id != 0 && g.Name != "" {
			srcIDToName[g.Id] = g.Name
		}
	}

	tgtNameToID := map[string]int{}
	for _, g := range tgt {
		if g.Id != 0 && g.Name != "" {
			tgtNameToID[strings.ToLower(g.Name)] = g.Id
		}
	}

	return srcIDToName, tgtNameToID, nil
}

func BuildTaskGroupMaps(
	sourceOrgURL, sourceProject,
	targetOrgURL, targetProject,
	resourceGUID string,
) (map[string]string, map[string]string, error) {

	src, err := ListTaskGroups(sourceOrgURL, sourceProject, resourceGUID)
	if err != nil {
		return nil, nil, err
	}
	tgt, err := ListTaskGroups(targetOrgURL, targetProject, resourceGUID)
	if err != nil {
		return nil, nil, err
	}

	srcIDToName := map[string]string{}
	for _, tg := range src {
		if tg.Id != "" && tg.Name != "" {
			srcIDToName[strings.ToLower(tg.Id)] = tg.Name
		}
	}

	tgtNameToID := map[string]string{}
	for _, tg := range tgt {
		if tg.Id != "" && tg.Name != "" {
			tgtNameToID[tg.Name] = tg.Id
		}
	}

	return srcIDToName, tgtNameToID, nil
}

// RemapBuildDefinitionRefsByName rewrites variableGroups + service endpoints + taskgroups inside classic build definition JSON.
func RemapBuildDefinitionRefsByName(
	def map[string]any,
	srcEndpointIDToName map[string]string,
	tgtEndpointNameToID map[string]string,
	srcVarGroupIDToName map[int]string,
	tgtVarGroupNameToID map[string]int,
	srcTaskGroupIDToName map[string]string,
	tgtTaskGroupNameToID map[string]string,
) {
	if vgs, ok := def["variableGroups"].([]any); ok {
		outIDs := make([]int, 0, len(vgs))

		for _, x := range vgs {
			srcID := readVarGroupID(x)
			if srcID == 0 {
				continue
			}
			if name, ok := srcVarGroupIDToName[srcID]; ok {
				//if newID, ok := tgtVarGroupNameToID[name]; ok {
				if newID, ok := tgtVarGroupNameToID[strings.ToLower(name)]; ok {
					outIDs = append(outIDs, newID)
					continue
				}
			}
			outIDs = append(outIDs, srcID)
		}

		setBuildVariableGroups(def, outIDs)
	}

	process, _ := def["process"].(map[string]any)
	phasesAny, _ := process["phases"].([]any)

	for _, ph := range phasesAny {
		phase, _ := ph.(map[string]any)
		stepsAny, _ := phase["steps"].([]any)

		for _, st := range stepsAny {
			step, _ := st.(map[string]any)

			if task, ok := step["task"].(map[string]any); ok {
				if id, ok := task["id"].(string); ok && id != "" {
					if name, ok := srcTaskGroupIDToName[strings.ToLower(id)]; ok {
						if newID, ok := tgtTaskGroupNameToID[name]; ok {
							task["id"] = newID
						}
					}
				}
			}

			inputs, _ := step["inputs"].(map[string]any)
			for k, v := range inputs {
				s, ok := v.(string)
				if !ok || s == "" {
					continue
				}
				if !IsEndpointKey(k) {
					continue
				}

				if epName, ok := srcEndpointIDToName[strings.ToLower(s)]; ok {
					if newID, ok := tgtEndpointNameToID[epName]; ok {
						inputs[k] = newID
					}
				}
			}
		}
	}
}

func readVarGroupID(v any) int {
	switch t := v.(type) {
	case float64:
		return int(t)
	case int:
		return t
	case map[string]any:
		// expanded form: {"id": 93, "name": "...", ...}
		if idf, ok := t["id"].(float64); ok {
			return int(idf)
		}
		if idi, ok := t["id"].(int); ok {
			return idi
		}
	}
	return 0
}

// Azure DevOps wants: "variableGroups": [ {"id": 15}, {"id": 16} ]
func setBuildVariableGroups(def map[string]any, ids []int) {
	out := make([]any, 0, len(ids))
	for _, id := range ids {
		out = append(out, map[string]any{"id": id})
	}
	def["variableGroups"] = out
}
